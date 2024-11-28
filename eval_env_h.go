package main

type Env interface {
	ReleaseEnv()
	LookupVariable(var1 string) string
}

type TokenType int8

const (
	RAW     TokenType = 0
	SPECIAL           = 1
)

type TokenPair struct {
	key string
	tp  TokenType
}
type TokenList []TokenPair

type EvalString struct {
	parsed_ TokenList

	// If we hold only a single RAW token, then we keep it here instead of
	// pushing it on TokenList. This saves a bunch of allocations for
	// what is a common case. If parsed_ is nonempty, then this value
	// must be ignored.
	single_token_ string
}

type Bindings map[string]*EvalString

type Rule struct {
	name_     string
	bindings_ Bindings
}

type BindingEnv struct {
	Env
	bindings_ map[string]string
	rules_    map[string]*Rule
	parent_   *BindingEnv
}
