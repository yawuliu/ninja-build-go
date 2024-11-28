package main

type Parser struct {
	state_       *State
	file_reader_ *FileReader
	lexer_       Lexer
}

func NewParser(state *State, file_reader *FileReader) *Parser {
	ret := Parser{}
	ret.state_ = state
	ret.file_reader_ = file_reader
	return &ret
}

// / Load and parse a file.
func (this *Parser) Load(filename string, err *string, parent *Lexer) bool {

}

// / If the next token is not \a expected, produce an error string
// / saying "expected foo, got bar".
func (this *Parser) ExpectToken(expected Token, err *string) bool {

}

// / Parse a file, given its contents as a string.
func (this *Parser) Parse(filename, input string, err *string) bool {

}
