package json_template

import (
	"strconv"
	"strings"
)

type opCode struct {
	cmd    vmCmdType
	target string
	fn     string
	fnArgs []string
	pos    Position //todo:
}

func (o opCode) String() string {
	res := o.cmd.String() + " " + o.target + " " + o.fn
	if o.fn != "" || len(o.fnArgs) > 0 {
		res += "(" + strings.Join(o.fnArgs, ", ") + ")"
	}
	res += "\n"
	return res
}

type opCodeBuilder struct {
	lastId int
}

func (b *opCodeBuilder) newId() string {
	b.lastId++
	return "@" + strconv.Itoa(b.lastId)
}

func (b *opCodeBuilder) build(node *astNode) []opCode {
	if node == nil {
		return nil
	}
	switch node.cmd {
	case astCmdCodeBlock:
		return b.buildCodeBlock(node)
	case astCmdIf:
		return b.buildIf(node)
	case astCmdFor:
		return b.buildFor(node)
	case astCmdForeach:
		return b.buildForeach(node)
	case astCmdSetVar:
		return b.buildSetVar(node)
	case astCmdJsonSet:
		return b.buildJsonSet(node)
	case astCmdAppend:
		return b.buildAppend(node)
	}

	//should be unreachable
	panic(node)
}

func (b *opCodeBuilder) buildCodeBlock(node *astNode) []opCode {
	var code []opCode
	for _, child := range node.child {
		childCode := b.build(child)
		code = append(code, childCode...)
	}
	return code
}

func (b *opCodeBuilder) buildIf(node *astNode) []opCode {
	pos := node.child[0].start
	condVar, code := b.buildDataPrimitive(node.child[0])
	code = append(code, b.freeTmpVars(condVar)...)

	thenCode := b.build(node.child[1])
	elseCode := b.build(node.child[2])

	var code2 []opCode
	if len(thenCode) == 0 {
		if len(elseCode) == 0 {
			return nil
		} else {
			code2 = b.makeIfE(condVar, elseCode, pos)
		}
	} else {
		if len(elseCode) == 0 {
			code2 = b.makeIfT(condVar, thenCode, pos)
		} else {
			code2 = b.makeIfTE(condVar, thenCode, elseCode, pos)
		}
	}

	return append(code, code2...)
}

func (b *opCodeBuilder) makeIfT(varName string, thenCode []opCode, pos Position) []opCode {
	/*
		build code for construction: if var {%code%}
		commands:
		jmpIf var Empty to @end
		%code%
		@end
	*/
	code := make([]opCode, 0, len(thenCode)+2)
	lblEnd := b.newId()
	code = append(code, opCode{
		cmd:    vmCmdJmpIfEmpty,
		target: lblEnd,
		fnArgs: []string{varName},
		pos:    pos,
	})
	code = append(code, thenCode...)
	code = append(code, opCode{
		cmd:    opCmdLabel,
		target: lblEnd,
	})
	return code
}

func (b *opCodeBuilder) makeIfTE(varName string, thenCode, elseCode []opCode, pos Position) []opCode {
	/*
		build code for construction: if var {%code%} else {%code%}
		commands:
		jmpIf var Empty to @else
		%code%
		jmp @end
		@else
		%code%
		@end
	*/
	code := make([]opCode, 0, len(thenCode)+len(elseCode)+4)
	lblElse := b.newId()
	lblEnd := b.newId()
	code = append(code, opCode{
		cmd:    vmCmdJmpIfEmpty,
		target: lblElse,
		pos:    pos,
		fnArgs: []string{varName},
	})
	code = append(code, thenCode...)
	code = append(code, opCode{
		cmd:    vmCmdJmp,
		target: lblEnd,
		pos:    pos,
	})
	code = append(code, opCode{
		cmd:    opCmdLabel,
		target: lblElse,
	})
	code = append(code, elseCode...)
	code = append(code, opCode{
		cmd:    opCmdLabel,
		target: lblEnd,
	})
	return code
}

func (b *opCodeBuilder) makeIfE(varName string, elseCode []opCode, pos Position) []opCode {
	/*
		build code for construction: if var {} else {%code%}
		commands:
		jmpIf var Not Empty to @end
		%code%
		@end
	*/
	code := make([]opCode, 0, len(elseCode)+2)
	lblEnd := b.newId()
	code = append(code, opCode{
		cmd:    vmCmdJmpIfNotEmpty,
		target: lblEnd,
		fnArgs: []string{varName},
		pos:    pos,
	})
	code = append(code, elseCode...)
	code = append(code, opCode{
		cmd:    opCmdLabel,
		target: lblEnd,
	})
	return code
}

func (b *opCodeBuilder) buildFor(node *astNode) []opCode {
	/*
		@head
		%init condition var%
		jmpIf conditionVar  Empty to @end
		%action%
		jmp @head
		@end
	*/

	condVar, condCode := b.buildDataPrimitive(node.child[0])

	actCode := b.build(node.child[1])

	clearTmp := b.freeTmpVars(condVar)
	lblHead := b.newId()
	lblEnd := b.newId()

	code := make([]opCode, 0, len(condCode)+len(clearTmp)+len(actCode)+4)
	code = append(code, opCode{
		cmd:    opCmdLabel,
		target: lblHead,
	})
	code = append(code, condCode...)
	code = append(code, clearTmp...)
	code = append(code, opCode{
		cmd:    vmCmdJmpIfEmpty,
		target: lblEnd,
		fnArgs: []string{condVar},
		pos:    node.start,
	})
	code = append(code, actCode...)
	code = append(code, opCode{
		cmd:    vmCmdJmp,
		target: lblHead,
		pos:    node.start,
	})
	code = append(code, opCode{
		cmd:    opCmdLabel,
		target: lblEnd,
	})

	return code
}

func (b *opCodeBuilder) buildForeach(node *astNode) []opCode {
	/*
		code:
		%dataCode%
		call target=%tmpIterator% fn="@initIterator[k|v|kv]" args=[%dataVar%]
		@head
		call target=%tmpCond% fn="@iteratorStep" args[%tmpIterator%]
		TmpVarFree %tmpCond%
		jmpIf %tmpCond% Empty to @end
		call target=key fn="@iteratorKey" args[%tmpIterator%]  # if use key in foreach
		call target=val fn="@iteratorVal" args[%tmpIterator%]  # if use val in foreach
		%actionCode%
		jmp @head
		@end
		TmpVarFree %tmpIterator%
	*/
	var keyName, valName string
	if node.child[0] != nil {
		keyName = node.child[0].data
	}
	if node.child[1] != nil {
		valName = node.child[1].data
	}
	lblHead := b.newId()
	lblEnd := b.newId()
	varIterator := b.newId()
	varCondition := b.newId()

	//init iterator
	dataVar, code := b.buildDataPrimitive(node.child[2])
	code = append(code, b.freeTmpVars(dataVar)...)
	fnName := "@initIterator"
	if keyName != "" {
		fnName += "K"
	}
	if valName != "" {
		fnName += "V"
	}
	code = append(code, opCode{
		cmd:    vmCmdCall,
		target: varIterator,
		fn:     fnName,
		fnArgs: []string{dataVar},
		pos:    node.start,
	})

	//check foreach condition
	code = append(code, opCode{
		cmd:    opCmdLabel,
		target: lblHead,
	})
	code = append(code, opCode{
		cmd:    vmCmdCall,
		target: varCondition,
		fn:     "@iteratorStep",
		fnArgs: []string{varIterator},
		pos:    node.start,
	})
	code = append(code, b.freeTmpVars(varCondition)...)
	code = append(code, opCode{
		cmd:    vmCmdJmpIfEmpty,
		target: lblEnd,
		fnArgs: []string{varCondition},
		pos:    node.start,
	})

	//set key, val
	if keyName != "" {
		code = append(code, opCode{
			cmd:    vmCmdCall,
			target: keyName,
			fn:     "@iteratorKey",
			fnArgs: []string{varIterator},
			pos:    node.start,
		})
	}
	if valName != "" {
		code = append(code, opCode{
			cmd:    vmCmdCall,
			target: valName,
			fn:     "@iteratorVal",
			fnArgs: []string{varIterator},
			pos:    node.start,
		})
	}

	//foreach action code
	actCode := b.build(node.child[3])
	code = append(code, actCode...)

	//end
	code = append(code, opCode{
		cmd:    vmCmdJmp,
		target: lblHead,
		pos:    node.start,
	})
	code = append(code, opCode{
		cmd:    opCmdLabel,
		target: lblEnd,
	})
	code = append(code, b.freeTmpVars(varIterator)...)

	return code
}

func (b *opCodeBuilder) buildSetVar(node *astNode) []opCode {
	dataVar, code := b.buildDataPrimitive(node.child[1])
	code = append(code, b.freeTmpVars(dataVar)...)

	varName := node.child[0].data
	code = append(code, opCode{
		cmd:    vmCmdCall,
		target: varName,
		fn:     "@clone",
		fnArgs: []string{dataVar},
		pos:    node.start,
	})
	return code
}

func (b *opCodeBuilder) buildDataPrimitive(node *astNode) (string, []opCode) {
	switch node.cmd {
	case astCmdConst:
		return b.dataConst(node)
	case astCmdStrTemplate:
		return b.dataStrTemplate(node)
	case astCmdFunction:
		return b.dataFunction(node)
	case astCmdVarPath:
		return b.dataVarPath(node)
	case astCmdVarName:
		return node.data, nil
	}

	//should be unreachable
	panic(node)
}

func (b *opCodeBuilder) dataConst(node *astNode) (string, []opCode) {
	name := b.newId()
	cmd := opCode{
		cmd:    opCmdConst,
		target: name,
		fnArgs: []string{node.data},
	}
	return name, []opCode{cmd}
}

func (b *opCodeBuilder) dataStrTemplate(node *astNode) (string, []opCode) {
	templateName := "%" + node.child[0].data
	argName, code := b.buildDataPrimitive(node.child[1])
	code = append(code, b.freeTmpVars(argName)...)
	target := b.newId()
	code = append(code, opCode{
		cmd:    vmCmdCall,
		target: target,
		fn:     "@strTemplate",
		fnArgs: []string{templateName, argName},
		pos:    node.start,
	})
	return target, code
}

func (b *opCodeBuilder) dataFunction(node *astNode) (string, []opCode) {
	fnName := node.child[0].data
	var args []string
	var code []opCode
	for _, dataNode := range node.child[1:] {
		argName, argCode := b.buildDataPrimitive(dataNode)
		code = append(code, argCode...)
		args = append(args, argName)
	}
	code = append(code, b.freeTmpVars(args...)...)
	target := b.newId()
	code = append(code, opCode{
		cmd:    vmCmdCall,
		target: target,
		fn:     fnName,
		fnArgs: args,
		pos:    node.start,
	})
	return target, code

}

func (b *opCodeBuilder) dataVarPath(node *astNode) (string, []opCode) {
	varName := node.child[0].data
	path := []string{varName}
	var code []opCode
	for _, dataNode := range node.child[1:] {
		argName, argCode := b.buildDataPrimitive(dataNode)
		code = append(code, argCode...)
		path = append(path, argName)
	}
	code = append(code, b.freeTmpVars(path...)...)
	target := b.newId()
	code = append(code, opCode{
		cmd:    vmCmdCall,
		target: target,
		fn:     "@get",
		fnArgs: path,
		pos:    node.start,
	})
	return target, code
}

func (b *opCodeBuilder) freeTmpVars(names ...string) []opCode {
	var code []opCode
	for _, name := range names {
		if name[0] == '@' {
			code = append(code, opCode{
				cmd:    opCmdTmpVarFree,
				target: name,
			})
		}
	}
	return code
}

func (b *opCodeBuilder) buildJsonSet(node *astNode) []opCode {
	return b.buildJsonOperation("@jsonSet", node)
}

func (b *opCodeBuilder) buildAppend(node *astNode) []opCode {
	return b.buildJsonOperation("@append", node)
}

func (b *opCodeBuilder) buildJsonOperation(fn string, node *astNode) []opCode {
	varData, code := b.buildDataPrimitive(node.child[1])
	pathNode := node.child[0]
	varName := pathNode.child[0].data
	var pathVars []string
	for _, keyNode := range pathNode.child[1:] {
		keyName, keyCode := b.buildDataPrimitive(keyNode)
		code = append(code, keyCode...)
		pathVars = append(pathVars, keyName)
	}
	code = append(code, opCode{
		cmd:    vmCmdCall,
		target: varName,
		fn:     fn,
		fnArgs: append([]string{varName, varData}, pathVars...),
		pos:    node.start,
	})
	code = append(code, b.freeTmpVars(varData)...)
	code = append(code, b.freeTmpVars(pathVars...)...)
	return code
}
