package json_template

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"text/template"
)

type compiler struct {
	deps           *Options
	vmCode         []vmCmd
	opCode         []opCode
	name2dataPtr   map[string]vmFnArg
	constData      []reflect.Value
	functions      []reflect.Value
	fnName2Id      map[string]int
	label2CodeLine map[string]int
	varDataSize    int
	tmpVarIsFree   []bool
	tmpVar2DataId  []int
	dataId2tmpVar  map[int]int
	inlineConst    map[string]int
}

func (c *compiler) compile(code string) error {
	var err error
	c.opCode, err = c.getOpCode(code)
	if err != nil {
		return err
	}

	err = c.init()
	if err != nil {
		return err
	}

	err = c.initOpCodeRefs()
	if err != nil {
		return err
	}

	err = c.buildVmCode()
	if err != nil {
		return err
	}

	return nil
}

func (c *compiler) getOpCode(code string) ([]opCode, error) {
	tokens, err := tokenize([]byte(code))
	if err != nil {
		return nil, err
	}
	ap := astParser{tokens: tokens}
	node, err := ap.parse()
	if err != nil {
		return nil, err
	}
	ob := opCodeBuilder{}
	return ob.build(node), nil
}

func (c *compiler) init() error {
	c.name2dataPtr = map[string]vmFnArg{}
	c.varDataSize = 2
	c.name2dataPtr["result"] = vmFnArg{1, 0}
	c.name2dataPtr["args"] = vmFnArg{1, 1}

	c.fnName2Id = map[string]int{}
	c.label2CodeLine = map[string]int{}
	c.dataId2tmpVar = map[int]int{}
	c.inlineConst = map[string]int{}

	if c.deps != nil {
		err := c.initDeps()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *compiler) initDeps() error {
	if c.deps.prototype != nil {
		c.initPrototype()
	}
	for name, val := range c.deps.constants {
		err := c.initNamedConst(name, val)
		if err != nil {
			return err
		}
	}
	for name, content := range c.deps.strTml {
		err := c.initNamedConst(name, content)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *compiler) initPrototype() {
	cid := len(c.constData)
	c.constData = append(c.constData, reflect.ValueOf(c.deps.prototype))
	fn, _ := c.getFunctionId("@clone")
	c.vmCode = append(c.vmCode, vmCmd{
		cmd:    vmCmdCall,
		target: 0,
		fn:     fn,
		fnArgs: []vmFnArg{{0, cid}},
	})
}

var rxName = regexp.MustCompile("^[a-zA-Z][_0-9a-zA-Z]*$")

func (c *compiler) initNamedConst(name string, val interface{}) error {
	cid := len(c.constData)
	c.constData = append(c.constData, reflect.ValueOf(val))
	c.name2dataPtr[name] = vmFnArg{0, cid}
	return nil
}

func (c *compiler) initStrTml(name, content string) error {
	if !rxName.MatchString(name) {
		return fmt.Errorf("incorrect name `%s` for string template", name)
	}
	t, err := template.New(name).Funcs(c.deps.strFunc).Parse(content)
	if err != nil {
		return err
	}
	cid := len(c.constData)
	c.constData = append(c.constData, reflect.ValueOf(t))
	c.name2dataPtr["%"+name] = vmFnArg{0, cid}
	return nil
}

func (c *compiler) getFunctionId(name string) (int, error) {
	id, ok := c.fnName2Id[name]
	if ok {
		return id, nil
	}
	id = len(c.functions)

	if c.deps != nil && c.deps.functions != nil {
		fn, ok := c.deps.functions[name]
		if ok {
			c.functions = append(c.functions, fn)
			return id, nil
		}
	}
	rfn, ok := buildInFunctions[name]
	if ok {
		c.functions = append(c.functions, rfn)
		return id, nil
	}

	return 0, fmt.Errorf("function %s not found", name)
}

func (c *compiler) initOpCodeRefs() error {
	vmCmdId := len(c.vmCode)
	for _, cmd := range c.opCode {
		switch cmd.cmd {
		case vmCmdCall, vmCmdJmp, vmCmdJmpIfEmpty, vmCmdJmpIfNotEmpty:
			vmCmdId++
		}

		switch cmd.cmd {
		case vmCmdCall:
			err := c.initVarName(cmd.target)
			if err != nil {
				return RuntimeError{err, cmd.pos}
			}
			_, err = c.getFunctionId(cmd.fn)
			if err != nil {
				return RuntimeError{err, cmd.pos}
			}
		case opCmdLabel:
			c.label2CodeLine[cmd.target] = vmCmdId
		case opCmdTmpVarFree:
			c.tempVarFree(cmd.target)
		case opCmdConst:
			err := c.initInlineConst(cmd.target, cmd.fnArgs[0])
			if err != nil {
				return RuntimeError{err, cmd.pos}
			}
		}
	}

	return nil
}

func (c *compiler) initVarName(name string) error {
	ptr, ok := c.name2dataPtr[name]
	if ok {
		if ptr.isVar == 0 {
			return fmt.Errorf("`%s` declarated as const", name)
		}
		return nil
	}

	if name[0] != '@' {
		//named var
		ptr = vmFnArg{1, c.varDataSize}
		c.name2dataPtr[name] = ptr
		c.varDataSize++
		return nil
	}

	//search free temp var
	for i, free := range c.tmpVarIsFree {
		if free {
			c.tmpVarIsFree[i] = false
			ptr = vmFnArg{1, c.tmpVar2DataId[i]}
			c.name2dataPtr[name] = ptr
			return nil
		}
	}

	//allocate new temp var
	ptr = vmFnArg{1, c.varDataSize}
	c.dataId2tmpVar[c.varDataSize] = len(c.tmpVarIsFree)
	c.tmpVarIsFree = append(c.tmpVarIsFree, false)
	c.tmpVar2DataId = append(c.tmpVar2DataId, c.varDataSize)
	c.name2dataPtr[name] = ptr
	c.varDataSize++
	return nil
}

func (c *compiler) tempVarFree(name string) {
	ptr, ok := c.name2dataPtr[name]
	if !ok {
		//should be unreachable
		panic(fmt.Errorf("free temp var `%s` before init", name))
	}
	if ptr.isVar == 0 {
		return
	}

	tmpId := c.dataId2tmpVar[ptr.dataId]

	c.tmpVarIsFree[tmpId] = true
}

func (c *compiler) initInlineConst(name string, data string) error {
	dataId, ok := c.inlineConst[data]
	if ok {
		c.name2dataPtr[name] = vmFnArg{0, dataId}
		return nil
	}

	val, err := c.inlineConstValue(data)
	if err != nil {
		return err
	}

	dataId = len(c.constData)
	c.constData = append(c.constData, val)
	c.name2dataPtr[name] = vmFnArg{0, dataId}
	c.inlineConst[data] = dataId
	return nil
}

func (c *compiler) inlineConstValue(data string) (reflect.Value, error) {
	var v interface{}
	err := json.Unmarshal([]byte(data), &v)
	if err != nil {
		return reflect.Value{}, err
	}

	switch v.(type) {
	case []interface{}, map[string]interface{}:
		v = json.RawMessage(data)
	}

	return reflect.ValueOf(v), nil
}

func (c *compiler) buildVmCode() error {

loop:
	for _, cmd := range c.opCode {
		var vmCmd vmCmd
		var err error

		switch cmd.cmd {
		case vmCmdCall:
			vmCmd, err = c.vmCmdCall(cmd)
		case vmCmdJmp:
			vmCmd, err = c.vmCmdJmp(cmd)
		case vmCmdJmpIfEmpty, vmCmdJmpIfNotEmpty:
			vmCmd, err = c.vmCmdJmpIf(cmd)
		default:
			continue loop
		}
		if err != nil {
			return RuntimeError{err, cmd.pos}
		}
		c.vmCode = append(c.vmCode, vmCmd)
	}

	return nil
}

func (c *compiler) vmCmdCall(code opCode) (vmCmd, error) {
	fnId, _ := c.getFunctionId(code.fn)
	args := make([]vmFnArg, len(code.fnArgs))
	for i, argName := range code.fnArgs {
		argPtr, ok := c.name2dataPtr[argName]
		if !ok {
			return vmCmd{}, fmt.Errorf("Unexpected reference `%s` in function %s call", argName, code.fn)
		}
		args[i] = argPtr
	}

	//validate args number if function call
	sign := c.functions[fnId].Type()
	if sign.IsVariadic() {
		if len(args) < sign.NumIn()-1 {
			return vmCmd{}, fmt.Errorf("wrong number of args for %s: want at least %d got %d", code.fn, sign.NumIn()-1, len(args))
		}
	} else {
		if len(args) != sign.NumIn() {
			return vmCmd{}, fmt.Errorf("wrong number of args for %s: want %d got %d", code.fn, sign.NumIn(), len(args))
		}
	}

	ptr := c.name2dataPtr[code.target]
	return vmCmd{
		cmd:     code.cmd,
		target:  ptr.dataId,
		fn:      fnId,
		fnArgs:  args,
		codePos: code.pos,
	}, nil
}

func (c *compiler) vmCmdJmp(code opCode) (vmCmd, error) {
	target, ok := c.label2CodeLine[code.target]
	if !ok {
		return vmCmd{}, fmt.Errorf("Unexpected lable `%s`", code.target)
	}
	return vmCmd{
		cmd:     code.cmd,
		target:  target,
		codePos: code.pos,
	}, nil

}

func (c *compiler) vmCmdJmpIf(code opCode) (vmCmd, error) {
	target, ok := c.label2CodeLine[code.target]
	if !ok {
		return vmCmd{}, fmt.Errorf("Unexpected lable `%s`", code.target)
	}

	argPtr, ok := c.name2dataPtr[code.fnArgs[0]]
	if !ok {
		return vmCmd{}, fmt.Errorf("Unexpected reference `%s` in jmpIf cmd", code.fn)
	}

	return vmCmd{
		cmd:     code.cmd,
		target:  target,
		codePos: code.pos,
		fnArgs:  []vmFnArg{argPtr},
	}, nil
}
