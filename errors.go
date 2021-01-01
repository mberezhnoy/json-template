package json_template

import (
	"errors"
	"fmt"
)

type RuntimeError struct {
	Err error
	Pos Position
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("[%d:%d] %s", e.Pos.line, e.Pos.offset, e.Err.Error())
}

type Position struct {
	offset int
	line   int
	column int
}

func (p *Position) inc(i int) {
	p.offset += i
	p.column += i
}

type ParseError struct {
	Msg string
	Pos Position
}

func (e ParseError) Error() string {
	return fmt.Sprintf("[%d:%d] %s", e.Pos.line, e.Pos.offset, e.Msg)
}

var ErrIncorrectName = errors.New("Incorrect name")
var ErrNotFunction = errors.New("Value is not a function")
var ErrIncorrectFunction = errors.New("Incorrect function")

const (
	ErrParseNumber               = "error in numeric token"
	ErrUnexpectedSymbol          = "unexpected symbol"
	ErrUnexpectedToken           = "unexpected token"
	ErrUnexpectedObjEnd          = "unexpected end on parse object declaration"
	ErrUnexpectedStrEnd          = "unexpected end on parse string declaration"
	ErrIllegalObjQuote           = "illegal charter in object declaration"
	ErrUnexpectedIfEnd           = "unexpected end in `if` block "
	ErrUnexpectedConstructionEnd = "unexpected construction end"
	ErrUnexpectedForEnd          = "unexpected end in `for` block "
	ErrVarName                   = "inadmissible var name"
)
