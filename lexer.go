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
  p := this.ofs_;
  const char* q;
  for {
    this.ofs_ = p;
    
{
	yych := uint8(0)
	yybm := []uint8{
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		128,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
	};
	yych = *p;
	if (yybm[0+yych] & 128) {
		goto yy81;
	}
	if (yych <= 0x00) {
		goto yy77
	}
	if (yych == '$') {
		goto yy84
	}
	goto yy79;
yy77:
	++p;
	{ break; }
yy79:
	++p;
yy80:
	{ break; }
yy81:
	yych = *++p;
	if (yybm[0+yych] & 128) {
		goto yy81;
	}
	{ continue; }
yy84:
	yych = *(q = ++p);
	if (yych == '\n') {
		goto yy85
	}
	if (yych == '\r') {
		goto yy87
	}
	goto yy80;
yy85:
	++p;
	{ continue; }
yy87:
	yych = *++p;
	if (yych == '\n') {
		goto yy89
	}
	p = q;
	goto yy80;
yy89:
	++p;
	{ continue; }
}

  }
}

// / Read a $-escaped string.
func (this*Lexer) ReadEvalString(eval *EvalString, path bool, err *string) bool {
	p := this.ofs_;
	const char* q;
	const char* start;
	for {
		start = p;

		{
			yych := uint8(0)
			yybm := []uint8{
			0,  16,  16,  16,  16,  16,  16,  16,
				16,  16,   0,  16,  16,   0,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				32,  16,  16,  16,   0,  16,  16,  16,
				16,  16,  16,  16,  16, 208, 144,  16,
				208, 208, 208, 208, 208, 208, 208, 208,
				208, 208,   0,  16,  16,  16,  16,  16,
				16, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 208,  16,  16,  16,  16, 208,
				16, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 208, 208, 208, 208, 208, 208,
				208, 208, 208,  16,   0,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
				16,  16,  16,  16,  16,  16,  16,  16,
		};
			yych = *p;
			if (yybm[0+yych] & 16) {
				goto yy102;
			}
			if (yych <= '\r') {
				if (yych <= 0x00) {
					goto yy100
				}
				if (yych <= '\n') {
					goto yy105
				}
				goto yy107;
			} else {
				if (yych <= ' ') {
					goto yy105
				}
				if (yych <= '$') {
					goto yy109
				}
				goto yy105;
			}
		yy100:
			p++
			{
				this.last_token_ = start;
				return Error("unexpected EOF", err);
			}
		yy102:
			yych = *++p;
			if (yybm[0+yych] & 16) {
				goto yy102;
			}
			{
				eval.AddText(StringPiece(start, p - start));
				continue;
			}
		yy105:
			++p;
			{
				if (path) {
					p = start;
					break;
				} else {
					if (*start == '\n')
					break;
					eval.AddText(StringPiece(start, 1));
					continue;
				}
			}
		yy107:
			yych = *++p;
			if (yych == '\n') {
				goto yy110
			}
			{
				this.last_token_ = start;
				return Error(this.DescribeLastError(), err);
			}
		yy109:
			yych = *++p;
			if (yybm[0+yych] & 64) {
				goto yy122;
			}
			if (yych <= ' ') {
				if (yych <= '\f') {
					if (yych == '\n') {
						goto yy114
					}
					goto yy112;
				} else {
					if (yych <= '\r') {
						goto yy117
					}
					if (yych <= 0x1F) {
						goto yy112
					}
					goto yy118;
				}
			} else {
				if (yych <= '/') {
					if (yych == '$') {
						goto yy120
					}
					goto yy112;
				} else {
					if (yych <= ':') {
						goto yy125
					}
					if (yych <= '`') {
						goto yy112
					}
					if (yych <= '{') {
						goto yy127
					}
					goto yy112;
				}
			}
		yy110:
			++p;
			{
				if (path) {
					p = start
				}
				break;
			}
		yy112:
			++p;
		yy113:
			{
				this.last_token_ = start;
				return Error("bad $-escape (literal $ must be written as $$)", err);
			}
		yy114:
			yych = *++p;
			if (yybm[0+yych] & 32) {
				goto yy114;
			}
			{
				continue;
			}
		yy117:
			yych = *++p;
			if (yych == '\n') {
				goto yy128
			}
			goto yy113;
		yy118:
			++p;
			{
				eval.AddText(StringPiece(" ", 1));
				continue;
			}
		yy120:
			++p;
			{
				eval.AddText(StringPiece("$", 1));
				continue;
			}
		yy122:
			yych = *++p;
			if (yybm[0+yych] & 64) {
				goto yy122;
			}
			{
				eval.AddSpecial(StringPiece(start + 1, p - start - 1));
				continue;
			}
		yy125:
			++p;
			{
				eval.AddText(StringPiece(":", 1));
				continue;
			}
		yy127:
			yych = *(q = ++p);
			if (yybm[0+yych] & 128) {
				goto yy131;
			}
			goto yy113;
		yy128:
			yych = *++p;
			if (yych == ' ') {
				goto yy128
			}
			{
				continue;
			}
		yy131:
			yych = *++p;
			if (yybm[0+yych] & 128) {
				goto yy131;
			}
			if (yych == '}') {
				goto yy134
			}
			p = q;
			goto yy113;
		yy134:
			++p;
			{
				eval.AddSpecial(StringPiece(start + 2, p - start - 3));
				continue;
			}
		}

	}
	this.last_token_ = start;
	this.ofs_ = p;
	if (path) {
		this.EatWhitespace()
	}
	// Non-path strings end in newlines, so there's no whitespace to eat.
	return true;
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
	if this.last_token_ {
		switch this.last_token_[0] {
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
	this.ofs_ = this.input_
	this.last_token_ = nil
}

// / Read a Token from the Token enum.
func (this *Lexer) ReadToken() Token {}

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

// / Read a simple identifier (a rule or variable name).
// / Returns false if a name can't be read.
func (this *Lexer) ReadIdent(out *string) bool {
  p := this.ofs_;
  const char* start;
  for {
    start = p;
    
{
	yych := uint8(0)
	yybm := []uint8{
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0, 128, 128,   0, 
		128, 128, 128, 128, 128, 128, 128, 128, 
		128, 128,   0,   0,   0,   0,   0,   0, 
		  0, 128, 128, 128, 128, 128, 128, 128, 
		128, 128, 128, 128, 128, 128, 128, 128, 
		128, 128, 128, 128, 128, 128, 128, 128, 
		128, 128, 128,   0,   0,   0,   0, 128, 
		  0, 128, 128, 128, 128, 128, 128, 128, 
		128, 128, 128, 128, 128, 128, 128, 128, 
		128, 128, 128, 128, 128, 128, 128, 128, 
		128, 128, 128,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
		  0,   0,   0,   0,   0,   0,   0,   0, 
	};
	yych = *p;
	if (yybm[0+yych] & 128) {
		goto yy95;
	}
	++p;
	{
      this.last_token_ = start;
      return false;
    }
yy95:
	yych = *++p;
	if (yybm[0+yych] & 128) {
		goto yy95;
	}
	{
      out.assign(start, p - start);
      break;
    }
}

  }
	this.last_token_ = start;
	this.ofs_ = p;
	this.EatWhitespace();
  return true;
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
  // Compute line/column.
  line := 1;
  line_start := this.input_
  for (const char* p = this.input_; p < this.last_token_; p++) {
    if (*p == '\n') {
      line++
      line_start = p + 1;
    }
  }
  col := this.last_token_ ? int(this.last_token_ - line_start) : 0;

  buf := ""
  snprintf(buf, sizeof(buf), "%s:%d: ", this.filename_, line);
  *err = buf;
  *err += message + "\n";

  // Add some context to the message.
  kTruncateColumn := 72;
  if (col > 0 && col < kTruncateColumn) {
    len:=0
    truncated := true;
    for len = 0; len < kTruncateColumn; len++ {
      if (line_start[len] == 0 || line_start[len] == '\n') {
        truncated = false;
        break;
      }
    }
    *err += string(line_start, len);
    if (truncated) {
		*err += "..."
	}
    *err += "\n";
    *err += string(col, ' ');
    *err += "^ near here";
  }

  return false;
}
