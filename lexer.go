package main

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
	last_token_ Token
}

func NewLexer0() *Lexer {}

// / Helper ctor useful for tests.
func NewLexer(input string) *Lexer {}

// / Skip past whitespace (called after each read token/ident/etc.).
func (this *Lexer) EatWhitespace() {}

// / Read a $-escaped string.
func ReadEvalString(eval *EvalString, path bool, err *string) bool {}

// / Return a human-readable form of a token, used in error messages.
func TokenName(t Token) string {}

// / Return a human-readable token hint, used in error messages.
func TokenErrorHint(expected Token) string {}

// / If the last token read was an ERROR token, provide more info
// / or the empty string.
func (this *Lexer) DescribeLastError() string {}

// / Start parsing some input.
func (this *Lexer) Start(filename, input string) {}

// / Read a Token from the Token enum.
func (this *Lexer) ReadToken() Token {}

// / Rewind to the last read Token.
func (this *Lexer) UnreadToken() {}

// / If the next token is \a token, read it and return true.
func (this *Lexer) PeekToken(token Token) bool {}

// / Read a simple identifier (a rule or variable name).
// / Returns false if a name can't be read.
func (this *Lexer) ReadIdent(out *string) bool {}

// / Read a path (complete with $escapes).
// / Returns false only on error, returned path may be empty if a delimiter
// / (space, newline) is hit.
func (this *Lexer) ReadPath(path *EvalString, err *string) bool {
	return ReadEvalString(path, true, err)
}

// / Read the value side of a var = value line (complete with $escapes).
// / Returns false only on error.
func (this *Lexer) ReadVarValue(value *EvalString, err *string) bool {
	return ReadEvalString(value, false, err)
}

// / Construct an error message with context.
func (this *Lexer) Error(message string, err *string) bool {}
