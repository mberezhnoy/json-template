package json_template

import (
	"fmt"
	"testing"
)

func text2Ast(code string) (*astNode, error) {
	tokens, err := tokenize([]byte(code))
	if err != nil {
		return nil, err
	}
	ap := astParser{tokens: tokens}
	node, err := ap.parse()
	if err != nil {
		return nil, err
	}
	return node, nil
}

func checkAst(t *testing.T, node, expect *astNode, path string) {
	if expect == nil && node == nil {
		return
	}
	if expect == nil && node != nil {
		t.Fatalf("expect nil in %s, got %s ", path, node.cmd)
	}
	if expect != nil && node == nil {
		t.Fatalf("expect %s in %s, got nil ", expect.cmd, path)
	}
	if expect.cmd < 0 {
		return
	}
	if node.cmd != expect.cmd {
		t.Fatalf("expect %s.%s got %s", path, expect.cmd, node.cmd)
	}
	if expect.data != "" && node.data != expect.data {
		t.Fatalf("incorrect node %s.%s content: expect %s, got %s ", path, node.cmd, expect.data, node.data)
	}
	if len(expect.child) > 0 && len(expect.child) != len(node.child) {
		t.Fatalf("incorrect number of child nodes in %s.%s: expect %d, got %d ", path, node.cmd, len(expect.child), len(node.child))
	}
	for i, child := range expect.child {
		newPath := fmt.Sprintf("%s.%s[%d]", path, node.cmd, i)
		checkAst(t, node.child[i], child, newPath)
	}
}

func TestAst1(t *testing.T) {
	code := `result = args`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdSetVar,
				child: []*astNode{
					{cmd: astCmdVarName, data: "result"},
					{cmd: astCmdVarName, data: "args"},
				},
			},
		},
	}
	checkAst(t, node, expect, "")
}

func TestAst2(t *testing.T) {
	code := `result[] = "x"`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdAppend,
				child: []*astNode{
					{
						cmd: astCmdVarPath,
						child: []*astNode{
							{cmd: astCmdVarName, data: "result"},
						},
					},
					{cmd: astCmdConst, data: `"x"`},
				},
			},
		},
	}
	checkAst(t, node, expect, "")
}

func TestAst3(t *testing.T) {
	code := `result.xxx.yyy =  %%["zzz"]%%`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdJsonSet,
				child: []*astNode{
					{
						cmd: astCmdVarPath,
						child: []*astNode{
							{cmd: astCmdVarName, data: "result"},
							{cmd: astCmdConst, data: `"xxx"`},
							{cmd: astCmdConst, data: `"yyy"`},
						},
					},
					{cmd: astCmdConst, data: `["zzz"]`},
				},
			},
		},
	}
	checkAst(t, node, expect, "")
}

func TestAst4(t *testing.T) {
	code := `result[0] =  args[xxx]`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdJsonSet,
				child: []*astNode{
					{
						cmd: astCmdVarPath,
						child: []*astNode{
							{cmd: astCmdVarName, data: "result"},
							{cmd: astCmdConst, data: `0`},
						},
					},
					{
						cmd: astCmdVarPath,
						child: []*astNode{
							{cmd: astCmdVarName, data: "args"},
							{cmd: astCmdVarName, data: "xxx"},
						},
					},
				},
			},
		},
	}
	checkAst(t, node, expect, "")
}

func TestAst5(t *testing.T) {
	code := `result[fn(x,y)][] =  .st(args)`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdAppend,
				child: []*astNode{
					{
						cmd: astCmdVarPath,
						child: []*astNode{
							{cmd: astCmdVarName, data: "result"},
							{
								cmd: astCmdFunction,
								child: []*astNode{
									{cmd: astCmdVarName, data: "fn"},
									{cmd: astCmdVarName, data: "x"},
									{cmd: astCmdVarName, data: "y"},
								},
							},
						},
					},

					{
						cmd: astCmdStrTemplate,
						child: []*astNode{
							{cmd: astCmdVarName, data: "st"},
							{cmd: astCmdVarName, data: "args"},
						},
					},
				},
			},
		},
	}
	checkAst(t, node, expect, "")
}

func TestAst6(t *testing.T) {
	code := `if args
		result = args
	end
	`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdIf,
				child: []*astNode{
					{cmd: astCmdVarName, data: "args"},
					{
						cmd: astCmdCodeBlock,
						child: []*astNode{
							{cmd: -1},
						},
					},
					nil,
				},
			},
		},
	}
	checkAst(t, node, expect, "")
}

func TestAst7(t *testing.T) {
	code := `if args
	else
		result = args
		result.x = 1
	end
	`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdIf,
				child: []*astNode{
					{cmd: astCmdVarName, data: "args"},
					nil,
					{
						cmd: astCmdCodeBlock,
						child: []*astNode{
							{cmd: -1},
							{cmd: -1},
						},
					},
				},
			},
		},
	}
	checkAst(t, node, expect, "")

}

func TestAst8(t *testing.T) {
	code := `if args
		result = args
	else
		result = args
		result.x = 1
	end
	`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdIf,
				child: []*astNode{
					{cmd: astCmdVarName, data: "args"},
					{
						cmd: astCmdCodeBlock,
						child: []*astNode{
							{cmd: -1},
						},
					},
					{
						cmd: astCmdCodeBlock,
						child: []*astNode{
							{cmd: -1},
							{cmd: -1},
						},
					},
				},
			},
		},
	}
	checkAst(t, node, expect, "")
}

func TestAst9(t *testing.T) {
	code := `
	for fn1(x)
		x = fn2(x)
	end
	`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdFor,
				child: []*astNode{
					{
						cmd: astCmdFunction,
						child: []*astNode{
							{cmd: -1},
							{cmd: -1},
						},
					},
					{
						cmd: astCmdCodeBlock,
						child: []*astNode{
							{cmd: -1},
						},
					},
				},
			},
		},
	}
	checkAst(t, node, expect, "")
}

func TestAst10(t *testing.T) {
	code := `
	for x in args
		result[]=x
	end
	`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdForeach,
				child: []*astNode{
					{cmd: astCmdVarName, data: "x"},
					nil,
					{cmd: astCmdVarName, data: "args"},
					{
						cmd: astCmdCodeBlock,
						child: []*astNode{
							{cmd: -1},
						},
					},
				},
			},
		},
	}
	checkAst(t, node, expect, "")
}

func TestAst11(t *testing.T) {
	code := `
	for key val in fn(args)
		result[key]=val
	end
	`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdForeach,
				child: []*astNode{
					{cmd: astCmdVarName, data: "key"},
					{cmd: astCmdVarName, data: "val"},
					{
						cmd:   astCmdFunction,
						child: []*astNode{{cmd: -1}, {cmd: -1}},
					},
					{
						cmd: astCmdCodeBlock,
						child: []*astNode{
							{cmd: -1},
						},
					},
				},
			},
		},
	}
	checkAst(t, node, expect, "")
}

func TestAst12(t *testing.T) {
	code := `
	for _ val in fn(args)
		result[]=val
	end
	`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdForeach,
				child: []*astNode{
					nil,
					{cmd: astCmdVarName, data: "val"},
					{
						cmd:   astCmdFunction,
						child: []*astNode{{cmd: -1}, {cmd: -1}},
					},
					{
						cmd: astCmdCodeBlock,
						child: []*astNode{
							{cmd: -1},
						},
					},
				},
			},
		},
	}
	checkAst(t, node, expect, "")
}

func TestAst13(t *testing.T) {
	code := `
	for key _ in fn(args)
		result[key]=1
	end
	`
	node, err := text2Ast(code)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	expect := &astNode{
		cmd: astCmdCodeBlock,
		child: []*astNode{
			{
				cmd: astCmdForeach,
				child: []*astNode{
					{cmd: astCmdVarName, data: "key"},
					nil,
					{
						cmd:   astCmdFunction,
						child: []*astNode{{cmd: -1}, {cmd: -1}},
					},
					{
						cmd: astCmdCodeBlock,
						child: []*astNode{
							{cmd: -1},
						},
					},
				},
			},
		},
	}
	checkAst(t, node, expect, "")
}
