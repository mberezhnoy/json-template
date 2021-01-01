package json_template

import (
	"testing"
)

func TestTokenize1(t *testing.T) {
	data := `
		x = args["str"]
	`
	tokens, err := tokenize([]byte(data))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(tokens) != 6 {
		t.Fatalf("len(tokens) = %d ", len(tokens))
	}
	if tokens[0].token != tokenWord {
		t.Fatal("token 0 type")
	}
	if string(tokens[0].data) != "x" {
		t.Fatal("token 0 content")
	}
	if tokens[1].token != tokenEqual {
		t.Fatal("token 1 type")
	}
	if tokens[2].token != tokenWord {
		t.Fatal("token 2 type")
	}
	if string(tokens[2].data) != "args" {
		t.Fatal("token 2 content")
	}
	if tokens[3].token != tokenBracketSO {
		t.Fatal("token 3 type")
	}
	if tokens[4].token != tokenString {
		t.Fatal("token 4 type")
	}
	if string(tokens[4].data) != `"str"` {
		t.Fatal("token 4 content")
	}
	if tokens[5].token != tokenBracketSC {
		t.Fatal("token 5 type")
	}
}

func TestTokenize2(t *testing.T) {
	data := "``{dddddd}``-1.34"
	tokens, err := tokenize([]byte(data))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("len(tokens) = %d ", len(tokens))
	}
	if tokens[0].token != tokenObject {
		t.Fatal("token 0 type")
	}
	if string(tokens[0].data) != "{dddddd}" {
		t.Fatal("token 0 content")
	}
	if tokens[1].token != tokenNum {
		t.Fatal("token 1 type")
	}
	if string(tokens[1].data) != "-1.34" {
		t.Fatal("token 1 content")
	}

}
func TestTokenize3(t *testing.T) {
	data := "32 xx()`x`{\"s\":\"``\"}`x`"
	tokens, err := tokenize([]byte(data))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(tokens) != 5 {
		t.Fatalf("len(tokens) = %d ", len(tokens))
	}
	if tokens[0].token != tokenNum {
		t.Fatal("token 0 type")
	}
	if string(tokens[0].data) != "32" {
		t.Fatal("token 0 content")
	}
	if tokens[1].token != tokenWord {
		t.Fatal("token 1 type")
	}
	if string(tokens[1].data) != "xx" {
		t.Fatal("token 1 content")
	}
	if tokens[2].token != tokenBracketRO {
		t.Fatal("token 2 type")
	}
	if tokens[3].token != tokenBracketRC {
		t.Fatal("token 3 type")
	}
	if tokens[4].token != tokenObject {
		t.Fatal("token 4 type")
	}
	if string(tokens[4].data) != "{\"s\":\"``\"}" {
		t.Fatal("token 4 content")
	}
}

func TestTokenize4(t *testing.T) {
	data := `ss.dd("x\"x")`
	tokens, err := tokenize([]byte(data))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(tokens) != 6 {
		t.Fatalf("len(tokens) = %d ", len(tokens))
	}
	if tokens[1].token != tokenDot {
		t.Fatal("token 3 type")
	}
	if tokens[4].token != tokenString {
		t.Fatal("token 4 type")
	}
	if string(tokens[4].data) != `"x\"x"` {
		t.Fatal("token 4 type")
	}
}

func TestTokenize5(t *testing.T) {
	data := `res = xx
	if x()
		@
	end`
	_, err := tokenize([]byte(data))
	if err == nil {
		t.Fatal("err is nil")
	}
	pErr, ok := err.(ParseError)
	if !ok {
		t.Fatalf("err type %T != ParseError", err)
	}
	if pErr.Msg != ErrUnexpectedSymbol {
		t.Fatal("err msg:", pErr.Msg)
	}
	if pErr.Pos.line != 3 || pErr.Pos.column != 2 {
		t.Fatal("err position:", pErr.Pos)
	}
}

func TestTokenize6(t *testing.T) {
	data := `12..34`
	_, err := tokenize([]byte(data))
	if err == nil {
		t.Fatal("err is nil")
	}
	pErr, ok := err.(ParseError)
	if !ok {
		t.Fatalf("err type %T != ParseError", err)
	}
	if pErr.Msg != ErrParseNumber {
		t.Fatal("err msg:", pErr.Msg)
	}
}

func TestTokenize7(t *testing.T) {
	data := "xx = `y"
	_, err := tokenize([]byte(data))
	if err == nil {
		t.Fatal("err is nil")
	}
	pErr, ok := err.(ParseError)
	if !ok {
		t.Fatalf("err type %T != ParseError", err)
	}
	if pErr.Msg != ErrUnexpectedObjEnd {
		t.Fatal("err msg:", pErr.Msg)
	}
}
func TestTokenize8(t *testing.T) {
	data := "xx = ``{}"
	_, err := tokenize([]byte(data))
	if err == nil {
		t.Fatal("err is nil")
	}
	pErr, ok := err.(ParseError)
	if !ok {
		t.Fatalf("err type %T != ParseError", err)
	}
	if pErr.Msg != ErrUnexpectedObjEnd {
		t.Fatal("err msg:", pErr.Msg)
	}
}
func TestTokenize9(t *testing.T) {
	data := `d="xx`
	_, err := tokenize([]byte(data))
	if err == nil {
		t.Fatal("err is nil")
	}
	pErr, ok := err.(ParseError)
	if !ok {
		t.Fatalf("err type %T != ParseError", err)
	}
	if pErr.Msg != ErrUnexpectedStrEnd {
		t.Fatal("err msg:", pErr.Msg)
	}
}
func TestTokenize10(t *testing.T) {
	data := "`@`{}`@`"
	_, err := tokenize([]byte(data))
	if err == nil {
		t.Fatal("err is nil")
	}
	pErr, ok := err.(ParseError)
	if !ok {
		t.Fatalf("err type %T != ParseError", err)
	}
	if pErr.Msg != ErrIllegalObjQuote {
		t.Fatal("err msg:", pErr.Msg)
	}
}

func TestTokenize11(t *testing.T) {
	data := `if x y=x else y=z end`
	tokens, err := tokenize([]byte(data))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(tokens) != 10 {
		t.Fatalf("len(tokens) = %d ", len(tokens))
	}
	if tokens[0].token != tokenKwIf {
		t.Fatal("token 0 type")
	}
	if tokens[5].token != tokenKwElse {
		t.Fatal("token 5 type")
	}
	if tokens[9].token != tokenKwEnd {
		t.Fatal("token 9 type")
	}
}
func TestTokenize12(t *testing.T) {
	data := `for _ x in args`
	tokens, err := tokenize([]byte(data))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(tokens) != 5 {
		t.Fatalf("len(tokens) = %d ", len(tokens))
	}
	if tokens[0].token != tokenKwFor {
		t.Fatal("token 0 type")
	}
	if tokens[1].token != tokenWord {
		t.Fatal("token 1 type")
	}
	if string(tokens[1].data) != "_" {
		t.Fatal("token 1 content")
	}
	if tokens[3].token != tokenKwIn {
		t.Fatal("token 0 type")
	}
}

func TestTokenize13(t *testing.T) {
	data := `fn(x,y)`
	tokens, err := tokenize([]byte(data))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(tokens) != 6 {
		t.Fatalf("len(tokens) = %d ", len(tokens))
	}
	if tokens[0].token != tokenWord {
		t.Fatal("token 0 type")
	}
	if string(tokens[0].data) != "fn" {
		t.Fatal("token 0 content")
	}
	if tokens[1].token != tokenBracketRO {
		t.Fatal("token 1 type")
	}
	if tokens[2].token != tokenWord {
		t.Fatal("token 2 type")
	}
	if string(tokens[2].data) != "x" {
		t.Fatal("token 2 content")
	}
	if tokens[3].token != tokenComa {
		t.Fatal("token 3 type")
	}
	if tokens[4].token != tokenWord {
		t.Fatal("token 4 type")
	}
	if string(tokens[4].data) != "y" {
		t.Fatal("token 4 content")
	}
	if tokens[5].token != tokenBracketRC {
		t.Fatal("token 5 type")
	}

}

func TestTokenize14(t *testing.T) {
	data := `
    result = ` + "``" + `{
			"obj":{}, 
			"arr":[], 
			"info": "test template"
		}` + "``" + `
    x
	`
	tokens, err := tokenize([]byte(data))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(tokens) != 4 {
		t.Fatalf("len(tokens) = %d ", len(tokens))
	}

	pos := tokens[2].start
	if pos.line != 2 || pos.column != 13 {
		t.Fatal("Incorrect token position", pos)
	}
	pos = tokens[3].start
	if pos.line != 7 || pos.column != 4 {
		t.Fatal("Incorrect token position", pos)
	}

}
