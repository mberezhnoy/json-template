package json_template

import "fmt"

type astCmd int

const (
	astCmdCodeBlock astCmd = iota
	astCmdIf
	astCmdVarName
	astCmdFor
	astCmdForeach
	astCmdSetVar

	astCmdJsonSet
	astCmdAppend
	astCmdVarPath

	astCmdConst
	astCmdFunction
	astCmdStrTemplate
)

var astCmdNames = []string{"CodeBlock", "If", "VarName", "For", "Foreach", "SetVar", "JsonSet", "Append",
	"VarPath", "Const", "Function", "StrTemplate"}

func (a astCmd) String() string {
	if a >= 0 && int(a) < len(astCmdNames) {
		return astCmdNames[a]
	}
	return fmt.Sprintf("Cmd#%d", a)
}

type astNode struct {
	cmd        astCmd
	data       string
	start, end Position
	parent     *astNode
	child      []*astNode
}

func (a *astNode) String() string {
	return a.str("")
}

func (a *astNode) str(ident string) string {
	if a == nil {
		return ident + "nil\n"
	}
	res := ident + a.cmd.String() + " " + a.data + "\n"
	ident = ident + "  "
	for _, child := range a.child {
		res += child.str(ident)
	}
	return res
}

type astParser struct {
	cur    int
	tokens []token
}

func (a *astParser) parse() (*astNode, error) {
	node, err := a.parseCodeBlock()
	if err != nil {
		return nil, err
	}
	if a.cur < len(a.tokens) {
		return nil, ParseError{
			Msg: ErrUnexpectedToken,
			Pos: a.tokens[0].start,
		}
	}
	return node, nil
}

func (a *astParser) nextCommandInCodeBlock() (*astNode, error) {
	if a.cur >= len(a.tokens) {
		return nil, nil
	}
	t := a.tokens[a.cur]
	switch t.token {
	case tokenKwEnd, tokenKwElse:
		return nil, nil
	case tokenKwFor:
		return a.parseFor()
	case tokenKwIf:
		return a.parseIf()
	case tokenWord:
		return a.parseAssign()
	}

	return nil, ParseError{
		Msg: ErrUnexpectedSymbol,
		Pos: t.start,
	}
}

func (a *astParser) parseIf() (*astNode, error) {
	t := a.tokens[a.cur]
	node := &astNode{
		cmd:   astCmdIf,
		start: t.start,
		child: make([]*astNode, 3),
	}

	a.cur++
	var condition, thenDo, elseDo *astNode

	//get condition
	condition, err := a.parseDataPrimitive()
	if err != nil {
		return nil, err
	}
	condition.parent = node
	node.child[0] = condition

	//get then block
	thenDo, err = a.parseCodeBlock()
	if err != nil {
		return nil, err
	}
	thenDo.parent = node
	if len(thenDo.child) == 0 {
		thenDo = nil
	}
	node.child[1] = thenDo

	//get else block
	if a.cur >= len(a.tokens) {
		return nil, ParseError{
			Msg: ErrUnexpectedIfEnd,
			Pos: node.start,
		}
	}
	t = a.tokens[a.cur]
	if t.token == tokenKwElse {
		a.cur++
		elseDo, err = a.parseCodeBlock()
		if err != nil {
			return nil, err
		}
		elseDo.parent = node
		node.child[2] = elseDo
	}

	//check: correct block close
	if a.cur >= len(a.tokens) {
		return nil, ParseError{
			Msg: ErrUnexpectedIfEnd,
			Pos: node.start,
		}
	}
	t = a.tokens[a.cur]
	if t.token != tokenKwEnd {
		return nil, ParseError{
			Msg: ErrUnexpectedToken,
			Pos: t.start,
		}
	}
	node.end = t.end
	a.cur++

	return node, nil
}

func (a *astParser) parseFor() (*astNode, error) {
	t := a.tokens[a.cur]
	node := &astNode{
		cmd:   astCmdFor,
		start: t.start,
		child: make([]*astNode, 0, 4),
	}
	a.cur++

	//check&init: foreach
	if a.cur+2 < len(a.tokens) {
		t1 := a.tokens[a.cur]
		t2 := a.tokens[a.cur+1]
		t3 := a.tokens[a.cur+2]
		if t2.token == tokenKwIn || t3.token == tokenKwIn {
			node.cmd = astCmdForeach
			node.child = append(node.child, nil, nil)
			if t1.token == tokenWord && string(t1.data) != "_" {
				node.child[0] = a.newVarNameNode(t1, node)
			}
			if t2.token == tokenWord && string(t2.data) != "_" {
				node.child[1] = a.newVarNameNode(t2, node)
			}

			//set cur to next node after `in`
			a.cur += 2
			if t2.token != tokenKwIn {
				a.cur++
			}
		}
	}

	//get data source
	data, err := a.parseDataPrimitive()
	if err != nil {
		return nil, err
	}
	node.child = append(node.child, data)

	//get actions
	actions, err := a.parseCodeBlock()
	if err != nil {
		return nil, err
	}
	node.child = append(node.child, actions)

	//check: correct block close
	if a.cur >= len(a.tokens) {
		return nil, ParseError{
			Msg: ErrUnexpectedForEnd,
			Pos: node.start,
		}
	}
	t = a.tokens[a.cur]
	if t.token != tokenKwEnd {
		return nil, ParseError{
			Msg: ErrUnexpectedToken,
			Pos: t.start,
		}
	}
	node.end = t.end
	a.cur++

	return node, nil
}

func (a *astParser) parseAssign() (*astNode, error) {
	start := a.tokens[a.cur].start
	if a.cur+2 >= len(a.tokens) {
		return nil, ParseError{
			Msg: ErrUnexpectedConstructionEnd,
			Pos: start,
		}
	}
	if a.tokens[a.cur+1].token == tokenEqual {
		return a.parseSetVar()
	}

	pathNode, err := a.parseVarPath()
	if err != nil {
		return nil, err
	}

	if a.cur+1 >= len(a.tokens) {
		return nil, ParseError{
			Msg: ErrUnexpectedConstructionEnd,
			Pos: start,
		}
	}
	if a.tokens[a.cur].token == tokenEqual {
		return a.createJsonSetNode(pathNode)
	}

	return a.createAppendNode(pathNode)
}

func (a *astParser) parseSetVar() (*astNode, error) {
	t := a.tokens[a.cur]
	node := &astNode{
		cmd:   astCmdSetVar,
		start: t.start,
		child: make([]*astNode, 2),
	}
	node.child[0] = a.newVarNameNode(t, node)
	a.cur += 2

	data, err := a.parseDataPrimitive()
	if err != nil {
		return nil, err
	}
	data.parent = node
	node.child[1] = data

	return node, nil
}

func (a *astParser) parseVarPath() (*astNode, error) {
	t := a.tokens[a.cur]
	node := &astNode{
		cmd:   astCmdVarPath,
		start: t.start,
		end:   t.end,
		child: make([]*astNode, 1),
	}
	node.child[0] = a.newVarNameNode(t, node)
	a.cur++
loop:
	for {
		if a.cur >= len(a.tokens) {
			break loop
		}
		if a.cur+1 >= len(a.tokens) {
			return nil, ParseError{
				Msg: ErrUnexpectedConstructionEnd,
				Pos: t.start,
			}
		}
		t := a.tokens[a.cur]
		switch t.token {
		case tokenDot:
			t := a.tokens[a.cur+1]
			if t.token != tokenWord {
				return nil, ParseError{
					Msg: ErrUnexpectedToken,
					Pos: t.start,
				}
			}
			child := &astNode{
				cmd:    astCmdConst,
				parent: node,
				data:   `"` + string(t.data) + `"`,
				start:  t.start,
				end:    t.end,
			}
			node.child = append(node.child, child)
			node.end = t.end
			a.cur += 2
		case tokenBracketSO:
			t := a.tokens[a.cur]
			if a.tokens[a.cur+1].token == tokenBracketSC {
				break loop
			}
			a.cur++
			child, err := a.parseDataPrimitive()
			if err != nil {
				return nil, err
			}
			node.child = append(node.child, child)
			if a.cur >= len(a.tokens) {
				return nil, ParseError{
					Msg: ErrUnexpectedConstructionEnd,
					Pos: t.start,
				}
			}
			t = a.tokens[a.cur]
			if t.token != tokenBracketSC {
				return nil, ParseError{
					Msg: ErrUnexpectedToken,
					Pos: t.start,
				}
			}
			node.end = t.end
			a.cur++
		default:
			break loop
		}
	}

	return node, nil
}

func (a *astParser) createJsonSetNode(pathNode *astNode) (*astNode, error) {
	a.cur++
	data, err := a.parseDataPrimitive()
	if err != nil {
		return nil, err
	}
	node := &astNode{
		cmd:   astCmdJsonSet,
		start: pathNode.start,
		end:   data.end,
		child: []*astNode{pathNode, data},
	}
	pathNode.parent = node
	data.parent = node
	return node, nil
}

func (a *astParser) createAppendNode(pathNode *astNode) (*astNode, error) {
	if a.cur+3 >= len(a.tokens) {
		return nil, ParseError{
			Msg: ErrUnexpectedConstructionEnd,
			Pos: pathNode.start,
		}
	}
	if a.tokens[a.cur].token != tokenBracketSO {
		return nil, ParseError{
			Msg: ErrUnexpectedToken,
			Pos: a.tokens[a.cur].start,
		}
	}
	if a.tokens[a.cur+1].token != tokenBracketSC {
		return nil, ParseError{
			Msg: ErrUnexpectedToken,
			Pos: a.tokens[a.cur+1].start,
		}
	}
	if a.tokens[a.cur+2].token != tokenEqual {
		return nil, ParseError{
			Msg: ErrUnexpectedToken,
			Pos: a.tokens[a.cur+2].start,
		}
	}

	a.cur += 3
	data, err := a.parseDataPrimitive()
	if err != nil {
		return nil, err
	}
	node := &astNode{
		cmd:   astCmdAppend,
		start: pathNode.start,
		end:   data.end,
		child: []*astNode{pathNode, data},
	}
	pathNode.parent = node
	data.parent = node
	return node, nil
}

func (a *astParser) parseDataPrimitive() (*astNode, error) {
	if a.cur >= len(a.tokens) {
		return nil, ParseError{
			Msg: ErrUnexpectedConstructionEnd,
			Pos: a.tokens[len(a.tokens)-1].end,
		}
	}
	/*
		number|string|object -> astCmdConst
		dot,word,BracketRO   -> astCmdStrTemplate
		word,BracketRO       -> astCmdFunction
		word,dot|BracketSO   -> astCmdVarPath
		word                 -> astCmdVarName
	*/
	var t1, t2, t3 tokenType
	t := a.tokens[a.cur]
	t1 = t.token
	if len(a.tokens) > a.cur+1 {
		t2 = a.tokens[a.cur+1].token
	}
	if len(a.tokens) > a.cur+2 {
		t3 = a.tokens[a.cur+2].token
	}
	switch {
	case t1 == tokenNum || t1 == tokenString || t1 == tokenObject:
		node := &astNode{
			cmd:   astCmdConst,
			data:  string(t.data),
			start: t.start,
			end:   t.end,
		}
		a.cur++
		return node, nil
	case t1 == tokenDot && t2 == tokenWord && t3 == tokenBracketRO:
		return a.parseStrTemplate()
	case t1 == tokenWord && t2 == tokenBracketRO:
		return a.parseFunction()
	case t1 == tokenWord && (t2 == tokenDot || t2 == tokenBracketSO):
		return a.parseVarPath()
	case t1 == tokenWord:
		a.cur++
		return a.newVarNameNode(t, nil), nil
	}
	return nil, ParseError{
		Msg: ErrUnexpectedToken,
		Pos: t.start,
	}
}

func (a *astParser) parseStrTemplate() (*astNode, error) {
	node := &astNode{
		cmd:   astCmdStrTemplate,
		start: a.tokens[a.cur].start,
		child: make([]*astNode, 2),
	}
	node.child[0] = a.newVarNameNode(a.tokens[a.cur+1], node)
	a.cur += 3

	data, err := a.parseDataPrimitive()
	if err != nil {
		return nil, err
	}
	data.parent = node
	node.child[1] = data

	if a.cur >= len(a.tokens) {
		return nil, ParseError{
			Msg: ErrUnexpectedConstructionEnd,
			Pos: node.start,
		}
	}
	t := a.tokens[a.cur]
	if t.token != tokenBracketRC {
		return nil, ParseError{
			Msg: ErrUnexpectedToken,
			Pos: t.start,
		}
	}
	node.end = t.end
	a.cur++

	return node, nil
}

func (a *astParser) parseFunction() (*astNode, error) {
	t := a.tokens[a.cur]
	node := &astNode{
		cmd:   astCmdFunction,
		start: t.start,
		child: make([]*astNode, 1),
	}
	node.child[0] = a.newVarNameNode(t, node)
	a.cur += 2
	for {
		data, err := a.parseDataPrimitive()
		if err != nil {
			return nil, err
		}
		node.child = append(node.child, data)

		if a.cur >= len(a.tokens) {
			return nil, ParseError{
				Msg: ErrUnexpectedConstructionEnd,
				Pos: node.start,
			}
		}

		t := a.tokens[a.cur]
		a.cur++

		if t.token == tokenBracketRC {
			node.end = t.end
			return node, nil
		}

		if t.token != tokenComa {
			return nil, ParseError{
				Msg: ErrUnexpectedToken,
				Pos: t.start,
			}
		}
	}
}

func (a *astParser) parseCodeBlock() (*astNode, error) {
	node := &astNode{
		cmd:   astCmdCodeBlock,
		child: make([]*astNode, 0, 4),
	}
	var err error
	var child *astNode
	for err == nil {
		child, err = a.nextCommandInCodeBlock()
		if child == nil {
			break
		}
		child.parent = node
		node.child = append(node.child, child)
	}
	if err != nil {
		return nil, err
	}
	if len(node.child) > 0 {
		node.start = node.child[0].start
		node.end = node.child[len(node.child)-1].end
	}
	return node, nil
}

func (a *astParser) newVarNameNode(t token, parent *astNode) *astNode {
	return &astNode{
		cmd:    astCmdVarName,
		parent: parent,
		data:   string(t.data),
		start:  t.start,
		end:    t.end,
	}
}
