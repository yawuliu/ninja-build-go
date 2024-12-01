package main

type Parser interface {
	Load(filename string, err *string, parent *Lexer) bool
	ExpectToken(expected Token, err *string) bool
	Parse(filename, input string, err *string) bool
}
