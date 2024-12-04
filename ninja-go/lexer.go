package main

import "fmt"

// Token 是一个枚举，表示 lexer 可以识别的不同类型的 tokens。
type Token uint8

// 定义 Token 枚举值。
const (
	ERROR Token = iota
	BUILD
	COLON
	DEFAULT
	EQUALS
	IDENT
	INCLUDE
	INDENT
	NEWLINE
	PIPE
	PIPE2
	PIPEAT
	POOL
	RULE
	SUBNINJA
	TEOF
)

// Lexer 结构体包含 lexer 的状态。
type Lexer struct {
	filename_   string
	input_      string
	ofs_        int
	last_token_ int
}

func NewLexer0() *Lexer {
	ret := Lexer{}
	return &ret
}

// / Helper ctor useful for tests.
func NewLexer(input string) *Lexer {
	ret := Lexer{}
	ret.Start("input", input)
	return &ret
}

// / Skip past whitespace (called after each read token/ident/etc.).
func (this *Lexer) EatWhitespace() {
	p := this.ofs_
	q := 0
	for {
		this.ofs_ = p

		{
			yych := uint8(0)
			yybm := []uint8{
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				128, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
			}
			yych = this.input_[p]
			if (yybm[0+yych] & 128) != 0 {
				goto yy81
			}
			if yych <= 0x00 {
				goto yy77
			}
			if yych == '$' {
				goto yy84
			}
			goto yy79
		yy77:
			p++
			{
				break
			}
		yy79:
			p++
		yy80:
			{
				break
			}
		yy81:
			p++
			yych = this.input_[p]
			if (yybm[0+yych] & 128) != 0 {
				goto yy81
			}
			{
				continue
			}
		yy84:
			p++
			q = p
			yych = this.input_[q]
			if yych == '\n' {
				goto yy85
			}
			if yych == '\r' {
				goto yy87
			}
			goto yy80
		yy85:
			p++
			{
				continue
			}
		yy87:
			p++
			yych = this.input_[p]
			if yych == '\n' {
				goto yy89
			}
			p = q
			goto yy80
		yy89:
			p++
			{
				continue
			}
		}

	}
}

// isSpace 检查是否是空白字符
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

// / Read a $-escaped string.
func (this *Lexer) ReadEvalString(eval *EvalString, path bool, err *string) bool {
	p := this.ofs_
	q := 0
	start := 0
	for {
		start = p

		{
			yych := uint8(0)
			yybm := []uint8{
				0, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 0, 16, 16, 0, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				32, 16, 16, 16, 0, 16, 16, 16,
				16, 16, 16, 16, 16, 208, 144, 16,
				208, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 0, 16, 16, 16, 16, 16,
				16, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 208, 16, 16, 16, 16, 208,
				16, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 208, 16, 0, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
				16, 16, 16, 16, 16, 16, 16, 16,
			}
			yych = this.input_[p]
			if (yybm[0+yych] & 16) != 0 {
				goto yy102
			}
			if yych <= '\r' {
				if yych <= 0x00 {
					goto yy100
				}
				if yych <= '\n' {
					goto yy105
				}
				goto yy107
			} else {
				if yych <= ' ' {
					goto yy105
				}
				if yych <= '$' {
					goto yy109
				}
				goto yy105
			}
		yy100:
			p++
			{
				this.last_token_ = start
				return this.Error("unexpected EOF", err)
			}
		yy102:
			p++
			yych = this.input_[p]
			if (yybm[0+yych] & 16) != 0 {
				goto yy102
			}
			{
				eval.AddText(this.input_[start:p])
				continue
			}
		yy105:
			p++
			{
				if path {
					p = start
					break
				} else {
					if this.input_[start] == '\n' {
						break
					}
					eval.AddText(this.input_[start : start+1])
					continue
				}
			}
		yy107:
			p++
			yych = this.input_[p]
			if yych == '\n' {
				goto yy110
			}
			{
				this.last_token_ = start
				return this.Error(this.DescribeLastError(), err)
			}
		yy109:
			p++
			yych = this.input_[p]
			if (yybm[0+yych] & 64) != 0 {
				goto yy122
			}
			if yych <= ' ' {
				if yych <= '\f' {
					if yych == '\n' {
						goto yy114
					}
					goto yy112
				} else {
					if yych <= '\r' {
						goto yy117
					}
					if yych <= 0x1F {
						goto yy112
					}
					goto yy118
				}
			} else {
				if yych <= '/' {
					if yych == '$' {
						goto yy120
					}
					goto yy112
				} else {
					if yych <= ':' {
						goto yy125
					}
					if yych <= '`' {
						goto yy112
					}
					if yych <= '{' {
						goto yy127
					}
					goto yy112
				}
			}
		yy110:
			p++
			{
				if path {
					p = start
				}
				break
			}
		yy112:
			p++
		yy113:
			{
				this.last_token_ = start
				return this.Error("bad $-escape (literal $ must be written as $$)", err)
			}
		yy114:
			p++
			yych = this.input_[p]
			if (yybm[0+yych] & 32) != 0 {
				goto yy114
			}
			{
				continue
			}
		yy117:
			p++
			yych = this.input_[p]
			if yych == '\n' {
				goto yy128
			}
			goto yy113
		yy118:
			p++
			{
				eval.AddText(" ")
				continue
			}
		yy120:
			p++
			{
				eval.AddText("$")
				continue
			}
		yy122:
			p++
			yych = this.input_[p]
			if (yybm[0+yych] & 64) != 0 {
				goto yy122
			}
			{
				eval.AddSpecial(this.input_[start+1 : p])
				continue
			}
		yy125:
			p++
			{
				eval.AddText(":")
				continue
			}
		yy127:
			p++
			q = p
			yych = this.input_[q]
			if (yybm[0+yych] & 128) != 0 {
				goto yy131
			}
			goto yy113
		yy128:
			p++
			yych = this.input_[p]
			if yych == ' ' {
				goto yy128
			}
			{
				continue
			}
		yy131:
			p++
			yych = this.input_[p]
			if (yybm[0+yych] & 128) != 0 {
				goto yy131
			}
			if yych == '}' {
				goto yy134
			}
			p = q
			goto yy113
		yy134:
			p++
			{
				eval.AddSpecial(this.input_[start+2 : p-1])
				continue
			}
		}

	}
	this.last_token_ = start
	this.ofs_ = p
	if path {
		this.EatWhitespace()
	}
	// Non-path strings end in newlines, so there's no whitespace to eat.
	return true
}

// / Return a human-readable form of a token, used in error messages.
func TokenName(t Token) string {
	switch t {
	case ERROR:
		return "lexing error"
	case BUILD:
		return "'build'"
	case COLON:
		return "':'"
	case DEFAULT:
		return "'default'"
	case EQUALS:
		return "'='"
	case IDENT:
		return "identifier"
	case INCLUDE:
		return "'include'"
	case INDENT:
		return "indent"
	case NEWLINE:
		return "newline"
	case PIPE2:
		return "'||'"
	case PIPE:
		return "'|'"
	case PIPEAT:
		return "'|@'"
	case POOL:
		return "'pool'"
	case RULE:
		return "'rule'"
	case SUBNINJA:
		return "'subninja'"
	case TEOF:
		return "eof"
	}
	return "" // not reached
}

// / Return a human-readable token hint, used in error messages.
func TokenErrorHint(expected Token) string {
	switch expected {
	case COLON:
		return " ($ also escapes ':')"
	default:
		return ""
	}
}

// / If the last token read was an ERROR token, provide more info
// / or the empty string.
func (this *Lexer) DescribeLastError() string {
	last_token := this.input_[this.last_token_]
	if last_token != '\000' {
		switch last_token {
		case '\t':
			return "tabs are not allowed, use spaces"
		}
	}
	return "lexing error"
}

// / Start parsing some input.
func (this *Lexer) Start(filename, input string) {
	this.filename_ = filename
	this.input_ = input
	this.ofs_ = 0
	this.last_token_ = 0
}

// / Read a Token from the Token enum.
func (this *Lexer) ReadToken() Token {
	p := this.ofs_
	q := 0
	start := 0
	var token Token
	for {
		start = p
		{
			yych := uint8(0)
			yyaccept := uint32(0)
			yybm := []uint8{
				0, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 0, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				160, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 192, 192, 128,
				192, 192, 192, 192, 192, 192, 192, 192,
				192, 192, 128, 128, 128, 128, 128, 128,
				128, 192, 192, 192, 192, 192, 192, 192,
				192, 192, 192, 192, 192, 192, 192, 192,
				192, 192, 192, 192, 192, 192, 192, 192,
				192, 192, 192, 128, 128, 128, 128, 192,
				128, 192, 192, 192, 192, 192, 192, 192,
				192, 192, 192, 192, 192, 192, 192, 192,
				192, 192, 192, 192, 192, 192, 192, 192,
				192, 192, 192, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
			}
			yych = this.input_[p]
			if yybm[0+yych]&32 != 0 {
				goto yy9
			}
			if yych <= '^' {
				if yych <= ',' {
					if yych <= '\f' {
						if yych <= 0x00 {
							goto yy2
						}
						if yych == '\n' {
							goto yy6
						}
						goto yy4
					} else {
						if yych <= '\r' {
							goto yy8
						}
						if yych == '#' {
							goto yy12
						}
						goto yy4
					}
				} else {
					if yych <= ':' {
						if yych == '/' {
							goto yy4
						}
						if yych <= '9' {
							goto yy13
						}
						goto yy16
					} else {
						if yych <= '=' {
							if yych <= '<' {
								goto yy4
							}
							goto yy18
						} else {
							if yych <= '@' {
								goto yy4
							}
							if yych <= 'Z' {
								goto yy13
							}
							goto yy4
						}
					}
				}
			} else {
				if yych <= 'i' {
					if yych <= 'b' {
						if yych == '`' {
							goto yy4
						}
						if yych <= 'a' {
							goto yy13
						}
						goto yy20
					} else {
						if yych == 'd' {
							goto yy21
						}
						if yych <= 'h' {
							goto yy13
						}
						goto yy22
					}
				} else {
					if yych <= 'r' {
						if yych == 'p' {
							goto yy23
						}
						if yych <= 'q' {
							goto yy13
						}
						goto yy24
					} else {
						if yych <= 'z' {
							if yych <= 's' {
								goto yy25
							}
							goto yy13
						} else {
							if yych == '|' {
								goto yy26
							}
							goto yy4
						}
					}
				}
			}
		yy2:
			p++
			{
				token = TEOF
				break
			}
		yy4:
			p++
		yy5:
			{
				token = ERROR
				break
			}
		yy6:
			p++
			{
				token = NEWLINE
				break
			}
		yy8:
			p++
			yych = this.input_[p]
			if yych == '\n' {
				goto yy28
			}
			goto yy5
		yy9:
			yyaccept = 0
			p++
			q = p
			yych = this.input_[q]
			if yybm[0+yych]&32 != 0 {
				goto yy9
			}
			if yych <= '\f' {
				if yych == '\n' {
					goto yy6
				}
			} else {
				if yych <= '\r' {
					goto yy30
				}
				if yych == '#' {
					goto yy32
				}
			}
		yy11:
			{
				token = INDENT
				break
			}
		yy12:
			yyaccept = 1
			p++
			q = p
			yych = this.input_[q]
			if yych <= 0x00 {
				goto yy5
			}
			goto yy33
		yy13:
			p++
			yych = this.input_[p]
		yy14:
			if yybm[0+yych]&64 != 0 {
				goto yy13
			}
			{
				token = IDENT
				break
			}
		yy16:
			p++
			{
				token = COLON
				break
			}
		yy18:
			p++
			{
				token = EQUALS
				break
			}
		yy20:
			p++
			yych = this.input_[p]
			if yych == 'u' {
				goto yy36
			}
			goto yy14
		yy21:
			p++
			yych = this.input_[p]
			if yych == 'e' {
				goto yy37
			}
			goto yy14
		yy22:
			p++
			yych = this.input_[p]
			if yych == 'n' {
				goto yy38
			}
			goto yy14
		yy23:
			p++
			yych = this.input_[p]
			if yych == 'o' {
				goto yy39
			}
			goto yy14
		yy24:
			p++
			yych = this.input_[p]
			if yych == 'u' {
				goto yy40
			}
			goto yy14
		yy25:
			p++
			yych = this.input_[p]
			if yych == 'u' {
				goto yy41
			}
			goto yy14
		yy26:
			p++
			yych = this.input_[p]
			if yych == '@' {
				goto yy42
			}
			if yych == '|' {
				goto yy44
			}
			{
				token = PIPE
				break
			}
		yy28:
			p++
			{
				token = NEWLINE
				break
			}
		yy30:
			p++
			yych = this.input_[p]
			if yych == '\n' {
				goto yy28
			}
		yy31:
			p = q
			if yyaccept == 0 {
				goto yy11
			} else {
				goto yy5
			}
		yy32:
			p++
			yych = this.input_[p]
		yy33:
			if yybm[0+yych]&128 != 0 {
				goto yy32
			}
			if yych <= 0x00 {
				goto yy31
			}
			p++
			{
				continue
			}
		yy36:
			p++
			yych = this.input_[p]
			if yych == 'i' {
				goto yy46
			}
			goto yy14
		yy37:
			p++
			yych = this.input_[p]
			if yych == 'f' {
				goto yy47
			}
			goto yy14
		yy38:
			p++
			yych = this.input_[p]
			if yych == 'c' {
				goto yy48
			}
			goto yy14
		yy39:
			p++
			yych = this.input_[p]
			if yych == 'o' {
				goto yy49
			}
			goto yy14
		yy40:
			p++
			yych = this.input_[p]
			if yych == 'l' {
				goto yy50
			}
			goto yy14
		yy41:
			p++
			yych = this.input_[p]
			if yych == 'b' {
				goto yy51
			}
			goto yy14
		yy42:
			p++
			{
				token = PIPEAT
				break
			}
		yy44:
			p++
			{
				token = PIPE2
				break
			}
		yy46:
			p++
			yych = this.input_[p]
			if yych == 'l' {
				goto yy52
			}
			goto yy14
		yy47:
			p++
			yych = this.input_[p]
			if yych == 'a' {
				goto yy53
			}
			goto yy14
		yy48:
			p++
			yych = this.input_[p]
			if yych == 'l' {
				goto yy54
			}
			goto yy14
		yy49:
			p++
			yych = this.input_[p]
			if yych == 'l' {
				goto yy55
			}
			goto yy14
		yy50:
			p++
			yych = this.input_[p]
			if yych == 'e' {
				goto yy57
			}
			goto yy14
		yy51:
			p++
			yych = this.input_[p]
			if yych == 'n' {
				goto yy59
			}
			goto yy14
		yy52:
			p++
			yych = this.input_[p]
			if yych == 'd' {
				goto yy60
			}
			goto yy14
		yy53:
			p++
			yych = this.input_[p]
			if yych == 'u' {
				goto yy62
			}
			goto yy14
		yy54:
			p++
			yych = this.input_[p]
			if yych == 'u' {
				goto yy63
			}
			goto yy14
		yy55:
			p++
			yych = this.input_[p]
			if yybm[0+yych]&64 != 0 {
				goto yy13
			}
			{
				token = POOL
				break
			}
		yy57:
			p++
			yych = this.input_[p]
			if yybm[0+yych]&64 != 0 {
				goto yy13
			}
			{
				token = RULE
				break
			}
		yy59:
			p++
			yych = this.input_[p]
			if yych == 'i' {
				goto yy64
			}
			goto yy14
		yy60:
			p++
			yych = this.input_[p]
			if yybm[0+yych]&64 != 0 {
				goto yy13
			}
			{
				token = BUILD
				break
			}
		yy62:
			p++
			yych = this.input_[p]
			if yych == 'l' {
				goto yy65
			}
			goto yy14
		yy63:
			p++
			yych = this.input_[p]
			if yych == 'd' {
				goto yy66
			}
			goto yy14
		yy64:
			p++
			yych = this.input_[p]
			if yych == 'n' {
				goto yy67
			}
			goto yy14
		yy65:
			p++
			yych = this.input_[p]
			if yych == 't' {
				goto yy68
			}
			goto yy14
		yy66:
			p++
			yych = this.input_[p]
			if yych == 'e' {
				goto yy70
			}
			goto yy14
		yy67:
			p++
			yych = this.input_[p]
			if yych == 'j' {
				goto yy72
			}
			goto yy14
		yy68:
			p++
			yych = this.input_[p]
			if yybm[0+yych]&64 != 0 {
				goto yy13
			}
			{
				token = DEFAULT
				break
			}
		yy70:
			p++
			yych = this.input_[p]
			if (yybm[0+yych] & 64) != 0 {
				goto yy13
			}
			{
				token = INCLUDE
				break
			}
		yy72:
			p++
			yych = this.input_[p]
			if yych != 'a' {
				goto yy14
			}
			p++
			yych = this.input_[p]
			if (yybm[0+yych] & 64) != 0 {
				goto yy13
			}
			{
				token = SUBNINJA
				break
			}
		}

	}

	this.last_token_ = start
	this.ofs_ = p
	if token != NEWLINE && token != TEOF {
		this.EatWhitespace()
	}
	return token
}

// / Rewind to the last read Token.
func (this *Lexer) UnreadToken() {
	this.ofs_ = this.last_token_
}

// / If the next token is \a token, read it and return true.
func (this *Lexer) PeekToken(token Token) bool {
	t := this.ReadToken()
	if t == token {
		return true
	}
	this.UnreadToken()
	return false
}

// isIdent 检查是否是标识符字符
func isIdent(b byte) bool {
	return 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || '0' <= b && b <= '9' || b == '_' || b == '-'
}

// / Read a simple identifier (a rule or variable name).
// / Returns false if a name can't be read.
func (this *Lexer) ReadIdent(out *string) bool {
	p := this.ofs_
	start := 0
	for {
		start = p

		{
			yych := uint8(0)
			yybm := []uint8{
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 128, 128, 0,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 0, 0, 0, 0, 0, 0,
				0, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 0, 0, 0, 0, 128,
				0, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 128, 128, 128, 128, 128,
				128, 128, 128, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
			}
			yych = this.input_[p]
			if (yybm[0+yych] & 128) != 0 {
				goto yy95
			}
			p++
			{
				this.last_token_ = start
				return false
			}
		yy95:
			p++
			yych = this.input_[p]
			if (yybm[0+yych] & 128) != 0 {
				goto yy95
			}
			{
				*out = this.input_[start:p]
				break
			}
		}

	}
	this.last_token_ = start
	this.ofs_ = p
	this.EatWhitespace()
	return true
}

// / Read a path (complete with $escapes).
// / Returns false only on error, returned path may be empty if a delimiter
// / (space, newline) is hit.
func (this *Lexer) ReadPath(path *EvalString, err *string) bool {
	return this.ReadEvalString(path, true, err)
}

// / Read the value side of a var = value line (complete with $escapes).
// / Returns false only on error.
func (this *Lexer) ReadVarValue(value *EvalString, err *string) bool {
	return this.ReadEvalString(value, false, err)
}

// / Construct an error message with context.
func (this *Lexer) Error(message string, err *string) bool {
	*err = fmt.Sprintf("%s: %s", this.filename_, message)
	return false
}
