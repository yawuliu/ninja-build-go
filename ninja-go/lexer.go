package ninja_go

import "fmt"

// Token 是一个枚举，表示 lexer 可以识别的不同类型的 tokens。
type Token int

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
	i := this.ofs_
	for {
		p := this.input_[i]
		if p != ' ' && p != '\t' && p != '\r' && p != '\n' {
			break
		}
		i++
	}
	this.ofs_ = i
}

// isSpace 检查是否是空白字符
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

// / Read a $-escaped string.
func (this *Lexer) ReadEvalString(eval *EvalString, path bool, err *string) bool {
	i := this.ofs_
	start := i
	for {
		p := this.input_[i]
		if p == '$' || p == 0 || (path && isSpace(p)) {
			break
		}
		i++
	}
	eval.AddText(this.input_[start:i])
	this.ofs_ = i
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

// skipComment 跳过注释
func (this *Lexer) skipComment(i int) int {
	for {
		p := this.input_[i]
		if p == '\n' || p == 0 {
			break
		}
		i++
	}
	return i
}

// readToken 读取一个令牌
func (this *Lexer) readToken(p int, start int) int {
	// 根据词法规则读取令牌
	// 这里需要根据实际的词法规则实现
	// 示例：读取标识符
	for isIdent(this.input_[p]) {
		p++
	}
	this.last_token_ = p //
	return p
}

// / Read a Token from the Token enum.
func (this *Lexer) ReadToken() Token {
	i := this.ofs_
	//p := this.input_[i]
	start := 0

	for {
		start = i
		p := this.input_[i]
		switch p {
		case 0:
			return TEOF
		case '\n':
			this.ofs_ = i + 1
			return NEWLINE
		case '#':
			// 处理注释
			i = this.skipComment(i)
		default:
			if isSpace(p) {
				this.ofs_ = i + 1
				this.EatWhitespace()
				i += 1
			} else {
				// 处理标识符和其他令牌
				i = this.readToken(i, start)
				break
			}
		}
	}
	return ERROR
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
	i := this.ofs_
	start := i
	for {
		p := this.input_[i]
		if !isIdent(p) {
			break
		}
		i++
	}
	*out = this.input_[start:i]
	this.ofs_ = i
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
