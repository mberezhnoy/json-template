package json_template

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

const (
	charZero      = 0
	charDot       = 46
	charComa      = 44
	charNum       = 48
	charLater     = 97
	charMinus     = 45
	charBracketRO = 40
	charBracketRC = 41
	charBracketSO = 91
	charBracketSC = 93
	charSlash     = 92
	charQuote     = 34
	charBackQuote = 96
	charNewLine   = 10
	charSpace     = 32
	charEqual     = 61
	charPercent   = 37
)

var charMap [256]byte

func init() {
	for i := 48; i <= 57; i++ {
		charMap[i] = charNum
	}
	for i := 97; i <= 122; i++ {
		charMap[i] = charLater
	}
	for i := 65; i <= 90; i++ {
		charMap[i] = charLater
	}
	charMap[95] = charLater

	charMap[charDot] = charDot
	charMap[charComa] = charComa
	charMap[charMinus] = charMinus
	charMap[charBracketRO] = charBracketRO
	charMap[charBracketRC] = charBracketRC
	charMap[charBracketSO] = charBracketSO
	charMap[charBracketSC] = charBracketSC
	charMap[charSlash] = charSlash
	charMap[charQuote] = charQuote
	charMap[charBackQuote] = charBackQuote
	charMap[charNewLine] = charNewLine
	charMap[charEqual] = charEqual
	charMap[charPercent] = charPercent
	charMap[charSpace] = charSpace
	charMap[9] = charSpace
	charMap[11] = charSpace
	charMap[12] = charSpace
	charMap[13] = charSpace
	charMap[133] = charSpace
	charMap[160] = charSpace
}

type tokenType int

const (
	tokenWord tokenType = iota + 1
	tokenDot
	tokenComa
	tokenBracketRO
	tokenBracketRC
	tokenBracketSO
	tokenBracketSC
	tokenEqual
	tokenNum
	tokenString
	tokenObject
	tokenKwIf
	tokenKwFor
	tokenKwIn
	tokenKwElse
	tokenKwEnd
)

var tokenTypeNames = []string{"none", "Word", ".", ",", "(", ")", "[", "]", "=", "Num", "String", "Object", "if", "for", "in", "else", "end"}

func (t tokenType) String() string {
	if t >= 0 && int(t) < len(tokenTypeNames) {
		return tokenTypeNames[t]
	}
	return fmt.Sprintf("token#%d", t)
}

type token struct {
	token      tokenType
	data       []byte
	start, end Position
}

type tokenizerState int

const (
	tokenizerStateNone tokenizerState = iota
	tokenizerStateWord
	tokenizerStateNumber
	tokenizerStateString
	tokenizerStateObject
)

func tokenize(data []byte) ([]token, error) {
	t := tokenizer{
		state: tokenizerStateNone,
		data:  data,
	}
	t.cur.line = 1
	var err error
	for err == nil {
		err = t.walk()
	}
	if err == io.EOF {
		err = nil
	}
	return t.tokens, err
}

type tokenizer struct {
	state      tokenizerState
	cur        Position
	data       []byte
	tokens     []token
	tokenStart Position
	dataStart  Position
	objQuote   []byte
}

func (t *tokenizer) walk() error {
	if t.cur.offset >= len(t.data) {
		return t.onDataEnd()
	}
	switch t.state {
	case tokenizerStateNone:
		err := t.walkIfStateNone()
		if err != nil {
			return err
		}
	case tokenizerStateWord:
		t.walkIfStateWord()
	case tokenizerStateNumber:
		err := t.walkIfStateNumber()
		if err != nil {
			return err
		}
	case tokenizerStateString:
		t.walkIfStateString()
	case tokenizerStateObject:
		t.walkIfStateObject()
	}

	return nil
}

func (t *tokenizer) onDataEnd() error {
	switch t.state {
	case tokenizerStateWord:
		t.closeWord()
		return io.EOF
	case tokenizerStateNumber:
		err := t.closeNumber()
		if err != nil {
			return err
		}
		return io.EOF
	case tokenizerStateString:
		return ParseError{
			Msg: ErrUnexpectedStrEnd,
			Pos: t.tokenStart,
		}
	case tokenizerStateObject:
		return ParseError{
			Msg: ErrUnexpectedObjEnd,
			Pos: t.tokenStart,
		}
	}
	return io.EOF
}

func (t *tokenizer) closeWord() {
	ct := token{
		token: tokenWord,
		data:  t.data[t.tokenStart.offset:t.cur.offset],
		start: t.tokenStart,
		end:   t.cur,
	}

	if len(ct.data) < 5 {
		switch string(ct.data) {
		case "if":
			ct.token = tokenKwIf
		case "for":
			ct.token = tokenKwFor
		case "in":
			ct.token = tokenKwIn
		case "else":
			ct.token = tokenKwElse
		case "end":
			ct.token = tokenKwEnd
		}
	}
	t.tokens = append(t.tokens, ct)
	t.state = tokenizerStateNone
}

func (t *tokenizer) closeNumber() error {
	data := t.data[t.tokenStart.offset:t.cur.offset]

	_, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return ParseError{
			Msg: ErrParseNumber,
			Pos: t.tokenStart,
		}
	}

	t.tokens = append(t.tokens, token{
		token: tokenNum,
		data:  data,
		start: t.tokenStart,
		end:   t.cur,
	})
	t.state = tokenizerStateNone

	return nil
}

func (t *tokenizer) walkIfStateNone() error {
	char := t.data[t.cur.offset]
	ct := charMap[char]
	switch ct {
	case charZero, charSlash:
		return ParseError{
			Msg: ErrUnexpectedSymbol,
			Pos: t.cur,
		}
	case charLater:
		t.tokenStart = t.cur
		t.state = tokenizerStateWord
	case charMinus, charNum:
		t.tokenStart = t.cur
		t.state = tokenizerStateNumber
	case charQuote:
		t.tokenStart = t.cur
		t.state = tokenizerStateString
	case charBackQuote:
		return t.openObject()
	case charDot:
		t.tokens = append(t.tokens, token{
			token: tokenDot,
			start: t.cur,
		})
	case charComa:
		t.tokens = append(t.tokens, token{
			token: tokenComa,
			start: t.cur,
		})
	case charEqual:
		t.tokens = append(t.tokens, token{
			token: tokenEqual,
			start: t.cur,
		})
	case charBracketRO:
		t.tokens = append(t.tokens, token{
			token: tokenBracketRO,
			start: t.cur,
		})
	case charBracketRC:
		t.tokens = append(t.tokens, token{
			token: tokenBracketRC,
			start: t.cur,
		})
	case charBracketSO:
		t.tokens = append(t.tokens, token{
			token: tokenBracketSO,
			start: t.cur,
		})
	case charBracketSC:
		t.tokens = append(t.tokens, token{
			token: tokenBracketSC,
			start: t.cur,
		})
	case charNewLine:
		t.cur.line++
		t.cur.column = -1
	}

	t.cur.inc(1)
	return nil
}

func (t *tokenizer) openObject() error {
	t.tokenStart = t.cur
	t.state = tokenizerStateObject
	data := t.data[t.cur.offset:]
	i := 1
	for {
		if i >= len(data) {
			return ParseError{
				Msg: ErrUnexpectedObjEnd,
				Pos: t.cur,
			}
		}
		char := data[i]
		if char == charBackQuote {
			break
		}
		ct := charMap[char]
		if ct != charLater && ct != charNum {
			pos := t.cur
			pos.inc(i)
			return ParseError{
				Msg: ErrIllegalObjQuote,
				Pos: pos,
			}
		}
		i++
	}

	i++
	t.cur.inc(i)
	t.objQuote = data[:i]
	t.dataStart = t.cur

	return nil
}

func (t *tokenizer) walkIfStateWord() {
	char := t.data[t.cur.offset]
	ct := charMap[char]
	if ct != charLater && ct != charNum {
		t.closeWord()
		return
	}
	t.cur.inc(1)
}

func (t *tokenizer) walkIfStateNumber() error {
	char := t.data[t.cur.offset]
	ct := charMap[char]
	if ct != charDot && ct != charNum {
		return t.closeNumber()
	}
	t.cur.inc(1)
	return nil
}

func (t *tokenizer) walkIfStateString() {
	char := t.data[t.cur.offset]
	ct := charMap[char]
	if char == charQuote && t.data[t.cur.offset-1] != charSlash {
		t.tokens = append(t.tokens, token{
			token: tokenString,
			data:  t.data[t.tokenStart.offset : t.cur.offset+1],
			start: t.tokenStart,
			end:   t.cur,
		})
		t.state = tokenizerStateNone
	}
	if ct == charNewLine {
		t.cur.line++
		t.cur.column = -1
	}
	t.cur.inc(1)
}

func (t *tokenizer) walkIfStateObject() {
	char := t.data[t.cur.offset]
	ct := charMap[char]
	if char == charBackQuote && bytes.HasPrefix(t.data[t.cur.offset:], t.objQuote) {
		data := t.data[t.dataStart.offset:t.cur.offset]
		t.cur.inc(len(t.objQuote))
		t.tokens = append(t.tokens, token{
			token: tokenObject,
			data:  data,
			start: t.tokenStart,
			end:   t.cur,
		})
		t.state = tokenizerStateNone
		return
	}

	if ct == charNewLine {
		t.cur.line++
		t.cur.column = -1
	}

	t.cur.inc(1)
}
