package main

type Parser struct {
	state_       *State
	file_reader_ FileReader
	lexer_       Lexer
}

func NewParser(state *State, file_reader FileReader) *Parser {
	ret := Parser{}
	ret.state_ = state
	ret.file_reader_ = file_reader
	return &ret
}

// / Load and parse a file.
func (this *Parser) Load(filename string, err *string, parent *Lexer) bool {
	// If |parent| is not NULL, metrics collection has been started by a parent
	// Parser::Load() in our call stack. Do not start a new one here to avoid
	// over-counting parsing times.
	METRIC_RECORD_IF(".ninja parse", parent == nil)
	contents := ""
	read_err := ""
	if this.file_reader_.ReadFile(filename, &contents, &read_err) != Okay {
		*err = "loading '" + filename + "': " + read_err
		if parent != nil {
			parent.Error(string(*err), err)
		}
		return false
	}

	return Parse(filename, contents, err)
}

// / If the next token is not \a expected, produce an error string
// / saying "expected foo, got bar".
func (this *Parser) ExpectToken(expected Token, err *string) bool {
	token := this.lexer_.ReadToken()
	if token != expected {
		message := string("expected ") + TokenName(expected)
		message += string(", got ") + TokenName(token)
		message += TokenErrorHint(expected)
		return this.lexer_.Error(message, err)
	}
	return true
}

// / Parse a file, given its contents as a string.
func (this *Parser) Parse(filename, input string, err *string) bool {

}
