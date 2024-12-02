package ninja_go

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

func NewRule(name string) *Rule {
	ret := Rule{}
	ret.name_ = name
	ret.bindings_ = make(Bindings)
	return &ret
}

type BindingEnv struct {
	Env
	bindings_ map[string]string
	rules_    map[string]*Rule
	parent_   *BindingEnv
}

func NewBindingEnv() *BindingEnv {
	ret := BindingEnv{}
	ret.bindings_ = map[string]string{}
	ret.rules_ = map[string]*Rule{}
	return &ret
}
func NewBindingEnvWithParent(parent *BindingEnv) *BindingEnv {
	ret := BindingEnv{}
	ret.bindings_ = map[string]string{}
	ret.rules_ = map[string]*Rule{}
	ret.parent_ = parent
	return &ret
}

func (this *BindingEnv) ReleaseBindingEnv() {}

func (this *BindingEnv) LookupVariable(var1 string) string {
	i, ok := this.bindings_[var1]
	if ok {
		return i
	}

	if this.parent_ != nil {
		return this.parent_.LookupVariable(var1)
	}
	return ""
}

func (this *BindingEnv) AddRule(rule *Rule) {
	if this.LookupRuleCurrentScope(rule.name()) != nil {
		panic("this.LookupRuleCurrentScope(rule.name()) != nil")
	}
	this.rules_[rule.name()] = rule
}

func (this *BindingEnv) LookupRule(rule_name string) *Rule {
	i, ok := this.rules_[rule_name]
	if ok {
		return i
	}
	if this.parent_ != nil {
		return this.parent_.LookupRule(rule_name)
	}
	return nil
}

func (this *BindingEnv) LookupRuleCurrentScope(rule_name string) *Rule {
	i, ok := this.rules_[rule_name]
	if !ok {
		return nil
	}
	return i
}

func (this *BindingEnv) GetRules() map[string]*Rule {
	return this.rules_
}

func (this *BindingEnv) AddBinding(key, val string) {
	this.bindings_[key] = val
}

// / This is tricky.  Edges want lookup scope to go in this order:
// / 1) value set on edge itself (edge_.env_)
// / 2) value set on rule, with expansion in the edge's scope
// / 3) value set on enclosing scope of edge (edge_.env_.parent_)
// / This function takes as parameters the necessary info to do (2).
func (this *BindingEnv) LookupWithFallback(var1 string, eval *EvalString, env Env) string {
	item, ok := this.bindings_[var1]
	if ok {
		return item
	}

	if eval != nil {
		return eval.Evaluate(env)
	}

	if this.parent_ != nil {
		return this.parent_.LookupVariable(var1)
	}

	return ""
}

func (this *Rule) name() string {
	return this.name_
}

func (this *Rule) AddBinding(key string, val *EvalString) {
	this.bindings_[key] = val
}

func IsReservedBinding(var1 string) bool {
	return var1 == "command" ||
		var1 == "depfile" ||
		var1 == "dyndep" ||
		var1 == "description" ||
		var1 == "deps" ||
		var1 == "generator" ||
		var1 == "pool" ||
		var1 == "restat" ||
		var1 == "rspfile" ||
		var1 == "rspfile_content" ||
		var1 == "msvc_deps_prefix"
}

func (this *Rule) GetBinding(key string) *EvalString {
	if val, ok := this.bindings_[key]; ok {
		return val
	}
	return nil
}

// / @return The evaluated string with variable expanded using value found in
// /         environment @a env.
func (this *EvalString) Evaluate(env Env) string {
	if len(this.parsed_) == 0 {
		return this.single_token_
	}

	result := ""
	for _, item := range this.parsed_ {
		if item.tp == RAW {
			result += item.key
		} else {
			result += env.LookupVariable(item.key)
		}
	}
	return result
}

// / @return The string with variables not expanded.
func (this *EvalString) Unparse() string {
	result := ""
	if len(this.parsed_) == 0 && this.single_token_ != "" {
		result += this.single_token_
	} else {
		for _, item := range this.parsed_ {
			special := item.tp == SPECIAL
			if special {
				result += "${"
			}
			result += item.key
			if special {
				result += "}"
			}
		}
	}
	return result
}

func (this *EvalString) Clear() {
	this.parsed_ = []TokenPair{}
	this.single_token_ = ""
}
func (this *EvalString) empty() bool {
	return len(this.parsed_) == 0 && this.single_token_ == ""
}

func (this *EvalString) AddText(text string) {
	if len(this.parsed_) == 0 {
		this.single_token_ += text
	} else if len(this.parsed_) != 0 && this.parsed_[len(this.parsed_)-1].tp == RAW {
		this.parsed_[len(this.parsed_)-1].key += text
	} else {
		this.parsed_ = append(this.parsed_, TokenPair{key: text, tp: RAW})
	}
}
func (this *EvalString) AddSpecial(text string) {
	if len(this.parsed_) == 0 && this.single_token_ != "" {
		// Going from one to two tokens, so we can no longer apply
		// our single_token_ optimization and need to push everything
		// onto the vector.
		this.parsed_ = append(this.parsed_, TokenPair{key: this.single_token_, tp: RAW})
	}
	this.parsed_ = append(this.parsed_, TokenPair{key: text, tp: SPECIAL})
}

// / Construct a human-readable representation of the parsed state,
// / for use in tests.
func (this *EvalString) Serialize() string {
	result := ""
	if len(this.parsed_) == 0 && this.single_token_ != "" {
		result += "["
		result += this.single_token_
		result += "]"
	} else {
		for _, pair := range this.parsed_ {
			result += "["
			if pair.tp == SPECIAL {
				result += "$"
			}
			result += pair.key
			result += "]"
		}
	}
	return result
}
