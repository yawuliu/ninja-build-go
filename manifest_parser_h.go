package main

import (
	"slices"
	"strconv"
)

type PhonyCycleAction int8

const (
	kPhonyCycleActionWarn  PhonyCycleAction = 0
	kPhonyCycleActionError                  = 1
)

type ManifestParserOptions struct {
	phony_cycle_action_ PhonyCycleAction
}

func NewManifestParserOptions() *ManifestParserOptions {
	ret := ManifestParserOptions{}
	ret.phony_cycle_action_ = kPhonyCycleActionWarn
	return &ret
}

type ManifestParser struct {
	*Parser
	env_     *BindingEnv
	options_ *ManifestParserOptions
	quiet_   bool

	// ins_/out_/validations_ are reused across invocations to ParseEdge(),
	// to save on the otherwise constant memory reallocation.
	// subparser_ is reused solely to get better reuse out ins_/outs_/validation_.
	subparser_   *ManifestParser
	ins_         []EvalString
	outs_        []EvalString
	validations_ []EvalString
}

// = ManifestParserOptions()
func NewManifestParser(state *State, file_reader FileReader, options *ManifestParserOptions) *ManifestParser {
	ret := ManifestParser{}
	ret.Parser = NewParser(state, file_reader)
	ret.options_ = options
	ret.quiet_ = false
	ret.env_ = &state.bindings_
	return &ret
}

// / Parse a text string of input.  Used by tests.
func (this *ManifestParser) ParseTest(input string, err *string) bool {
	this.quiet_ = true
	return this.Parse("input", input, err)
}

// / Parse a file, given its contents as a string.
func (this *ManifestParser) Parse(filename, input string, err *string) bool {
	this.lexer_.Start(filename, input);

	for {
	    token := this.lexer_.ReadToken();
		switch (token) {
		case POOL:
			if (!this.ParsePool(err)) {
				return false
			}
		case BUILD:
			if (!this.ParseEdge(err)) {
				return false
			}
		case RULE:
			if (!this.ParseRule(err)) {
				return false
			}
		case DEFAULT:
			if (!this.ParseDefault(err)) {
				return false
			}
		case IDENT: {
			this.lexer_.UnreadToken();
			name := ""
			let_value := EvalString{}
			if (!this.ParseLet(name, &let_value, err)) {
				return false
			}
			value := let_value.Evaluate(this.env_);
			// Check ninja_required_version immediately so we can exit
			// before encountering any syntactic surprises.
			if name == "ninja_required_version" {
				CheckNinjaVersion(value)
			}
			this.env_.AddBinding(name, value);
		}
		case INCLUDE:
			if !this.ParseFileInclude(false, err) {
				return false
			}
		case SUBNINJA:
			if !this.ParseFileInclude(true, err) {
				return false
			}
		case ERROR: {
			return this.lexer_.Error(this.lexer_.DescribeLastError(), err);
		}
		case TEOF:
			return true;
		case NEWLINE:
			return false;
		default:
			return this.lexer_.Error(string("unexpected ") + TokenName(token), err);
		}
	}
	return false;  // not reached
}

// / Parse various statement types.
func (this *ManifestParser) ParsePool(err *string) bool                             {
   name := ""
  if (!this.lexer_.ReadIdent(&name)) {
	  return this.lexer_.Error("expected pool name", err)
  }

  if (!this.ExpectToken(NEWLINE, err)){
	return false;
	}

  if (this.state_.LookupPool(name) != nil) {
	  return this.lexer_.Error("duplicate pool '"+name+"'", err)
  }

  depth := -1;

  for this.lexer_.PeekToken(INDENT) {
     key := ""
    value := EvalString{}
		if (!this.ParseLet(key, &value, err)) {
		return false
	}

    if (key == "depth") {
       depth_string := value.Evaluate(this.env_);
      depth,_ = strconv.Atoi(depth_string);
      if (depth < 0) {
		  return this.lexer_.Error("invalid pool depth", err)
	  }
    } else {
      return this.lexer_.Error("unexpected variable '" + key + "'", err);
    }
  }

  if (depth < 0) {
	  return this.lexer_.Error("expected 'depth =' line", err)
  }

	this.state_.AddPool(NewPool(name, depth));
  return true;
}
func (this *ManifestParser) ParseRule(err *string) bool                             {
  name := ""
  if (!this.lexer_.ReadIdent(&name)) {
	  return this.lexer_.Error("expected rule name", err)
  }

  if (!this.ExpectToken(NEWLINE, err)){
	return false;
  }

  if (this.env_.LookupRuleCurrentScope(name) != nil) {
	  return this.lexer_.Error("duplicate rule '"+name+"'", err)
  }

  rule := NewRule(name);  // XXX scoped_ptr

  for this.lexer_.PeekToken(INDENT) {
     key := ""
     value := EvalString{}
    if (!this.ParseLet(key, &value, err)) {
		return false
	}

    if IsReservedBinding(key) {
      rule.AddBinding(key, &value);
    } else {
      // Die on other keyvals for now; revisit if we want to add a
      // scope here.
      return this.lexer_.Error("unexpected variable '" + key + "'", err);
    }
  }

  if (rule.bindings_["rspfile"].empty() !=  rule.bindings_["rspfile_content"].empty()) {
    return this.lexer_.Error("rspfile and rspfile_content need to be " +
                        "both specified", err);
  }

  if (rule.bindings_["command"].empty()) {
	  return this.lexer_.Error("expected 'command =' line", err)
  }

	this.env_.AddRule(rule);
  return true;
}
func (this *ManifestParser) ParseLet(key string, value *EvalString, err *string) bool {
  if (!this.lexer_.ReadIdent(key)) {
	  return this.lexer_.Error("expected variable name", err)
  }
  if (!this.ExpectToken(EQUALS, err)){
	return false;
  }
  if (!this.lexer_.ReadVarValue(value, err)) {
	  return false
  }
  return true;
}
func (this *ManifestParser) ParseEdge(err *string) bool                             {
	this.ins_ =[]EvalString{}
	this.outs_=[]EvalString{}
	this.validations_=[]EvalString{}

  {
     out := EvalString{}
    if (!this.lexer_.ReadPath(&out, err)) {
		return false
	}
    for !out.empty() {
	  this.outs_ = append(this.outs_, out)

      out.Clear();
      if (!this.lexer_.ReadPath(&out, err)) {
		  return false
	  }
    }
  }

  // Add all implicit outs, counting how many as we go.
  implicit_outs := 0;
  if (this.lexer_.PeekToken(PIPE)) {
    for {
       out := EvalString{}
      if (!this.lexer_.ReadPath(&out, err)) {
		  return false
	  }
      if (out.empty()) {
		  break
	  }
		this.outs_ = append(this.outs_, out)
      implicit_outs ++
    }
  }

  if len(this.outs_)==0 {
	  return this.lexer_.Error("expected path", err)
  }

  if (!this.ExpectToken(COLON, err)){
	return false;
  }

  rule_name := ""
  if (!this.lexer_.ReadIdent(&rule_name)) {
	  return this.lexer_.Error("expected build command name", err)
  }

  rule := this.env_.LookupRule(rule_name);
  if rule==nil {
	  return this.lexer_.Error("unknown build rule '"+rule_name+"'", err)
  }

  for  {
    // XXX should we require one path here?
     in := EvalString{}
    if (!this.lexer_.ReadPath(&in, err)) {
		return false
	}
    if (in.empty()) {
		break
	}
	  this.ins_ = append(this.ins_, in)
  }

  // Add all implicit deps, counting how many as we go.
  implicit := 0;
  if (this.lexer_.PeekToken(PIPE)) {
    for  {
       in :=EvalString{}
      if (!this.lexer_.ReadPath(&in, err)){
			return false;
	  }
      if (in.empty()){
			break;
	  }
      this.ins_ =append(this.ins_, in)
      implicit++
    }
  }

  // Add all order-only deps, counting how many as we go.
  order_only := 0;
  if this.lexer_.PeekToken(PIPE2) {
    for  {
       in := EvalString{}
      if !this.lexer_.ReadPath(&in, err) {
		  return false;
	  }
      if (in.empty()){
			break;
	  }
		this.ins_= append(this.ins_, in)
     order_only ++
    }
  }

  // Add all validations, counting how many as we go.
  if this.lexer_.PeekToken(PIPEAT) {
    for  {
      validation := EvalString{}
	  if (! this.lexer_.ReadPath(& validation, err)){
		return false;
	  }
      if validation.empty() {
		break;
	  }
      this.validations_ =append(this.validations_, validation)
    }
  }

  if !this.ExpectToken(NEWLINE, err) {
	return false;
  }

  // Bindings on edges are rare, so allocate per-edge envs only when needed.
   has_indent_token := this.lexer_.PeekToken(INDENT);
   env :=  this.env_
   if has_indent_token {
	   env= NewBindingEnv(this.env_)
   }
  for has_indent_token {
    key := ""
     val := EvalString{}
    if !this.ParseLet(key, &val, err) {
		return false
	}

    env.AddBinding(key, val.Evaluate(this.env_));
    has_indent_token = this.lexer_.PeekToken(INDENT);
  }

  edge := this.state_.AddEdge(rule);
  edge.env_ = env;

  pool_name := edge.GetBinding("pool");
  if pool_name!="" {
    pool := this.state_.LookupPool(pool_name);
    if pool == nil{
		return this.lexer_.Error("unknown pool name '"+pool_name+"'", err)
	}
    edge.pool_ = pool;
  }

  edge.outputs_.reserve(len(this.outs_))
  for i := 0; i < len(this.outs_);i++  {
    path := this.outs_[i].Evaluate(env);
    if path=="" {
		return this.lexer_.Error("empty path", err)
	}
    slash_bits := uint64(0)
	CanonicalizePath(&path, &slash_bits);
    if (!this.state_.AddOut(edge, path, slash_bits, err)) {
		this.lexer_.Error(*err, err);
      return false;
    }
  }

  if len(edge.outputs_)==0 {
    // All outputs of the edge are already created by other edges. Don't add
    // this edge.  Do this check before input nodes are connected to the edge.
    this.state_.edges_.pop_back();
    delete edge;
    return true;
  }
  edge.implicit_outs_ = implicit_outs;

  edge.inputs_.reserve(len(this.ins_));
  for _,i := range this.ins_ {
    path := i.Evaluate(env);
    if path=="" {
		return this.lexer_.Error("empty path", err)
	}
    slash_bits := uint64(0)
	  CanonicalizePath(&path, &slash_bits);
    this.state_.AddIn(edge, path, slash_bits);
  }
  edge.implicit_deps_ = implicit;
  edge.order_only_deps_ = order_only;

  edge.validations_.reserve(len(this.validations_))
  for _,v := range this.validations_ {
    path := v.Evaluate(env);
    if path=="" {
		return this.lexer_.Error("empty path", err)
	}
    slash_bits := uint64(0)
		CanonicalizePath(&path, &slash_bits);
    this.state_.AddValidation(edge, path, slash_bits);
  }

  if (this.options_.phony_cycle_action_ == kPhonyCycleActionWarn &&  edge.maybe_phonycycle_diagnostic()) {
    // CMake 2.8.12.x and 3.0.x incorrectly write phony build statements
    // that reference themselves.  Ninja used to tolerate these in the
    // build graph but that has since been fixed.  Filter them out to
    // support users of those old CMake versions.
    out := edge.outputs_[0];
    new_end :=  remove(edge.inputs_.begin(), edge.inputs_.end(), out);
    if (new_end != edge.inputs_.end()) {
      edge.inputs_.erase(new_end, edge.inputs_.end());
      if !this.quiet_ {
        Warning("phony target '%s' names itself as an input; "+
                "ignoring [-w phonycycle=warn]",  out.path());
      }
    }
  }

  // Lookup, validate, and save any dyndep binding.  It will be used later
  // to load generated dependency information dynamically, but it must
  // be one of our manifest-specified inputs.
  dyndep := edge.GetUnescapedDyndep();
  if dyndep!="" {
    slash_bits := uint64(0)
	  CanonicalizePath(&dyndep, &slash_bits);
    edge.dyndep_ = this.state_.GetNode(dyndep, slash_bits);
    edge.dyndep_.set_dyndep_pending(true);
    if !slices.Contains(edge.inputs_,edge.dyndep_) {
      return this.lexer_.Error("dyndep '" + dyndep + "' is not an input", err);
    }
    if (!edge.dyndep_.generated_by_dep_loader()) {
		panic("!edge.dyndep_.generated_by_dep_loader()")
	}
  }

  return true;
}
func (this *ManifestParser) ParseDefault(err *string) bool                          {
   eval :=EvalString{}
  if (!this.lexer_.ReadPath(&eval, err)) {
	  return false
  }
  if (eval.empty()) {
	  return this.lexer_.Error("expected target name", err)
  }

  for {
    path := eval.Evaluate(this.env_);
    if path=="" {
	  return this.lexer_.Error("empty path", err)
	}
    slash_bits := uint64(0)  // Unused because this only does lookup.
    CanonicalizePath(&path, &slash_bits);
    default_err := ""
    if !this.state_.AddDefault(path, &default_err) {
		return this.lexer_.Error(default_err, err)
	}

    eval.Clear();
    if (!this.lexer_.ReadPath(&eval, err)) {
		return false
	}
	  if  eval.empty(){
		  break
	  }
  }

  return this.ExpectToken(NEWLINE, err);
}

// / Parse either a 'subninja' or 'include' line.
func (this *ManifestParser) ParseFileInclude(new_scope bool, err *string) bool {
   eval := EvalString{}
  if (!this.lexer_.ReadPath(&eval, err)) {
	  return false
  }
   path := eval.Evaluate(this.env_);

  if (this.subparser_ == nil) {
	  this.subparser_.reset(NewManifestParser(this.state_, this.file_reader_, this.options_));
  }
  if (new_scope) {
	  this.subparser_.env_ = NewBindingEnv(this.env_);
  } else {
	  this.subparser_.env_ = this.env_;
  }

  if (!this.subparser_.Load(path, err, &this.lexer_)) {
	  return false
  }

  if (!this.ExpectToken(NEWLINE, err)){
	return false;
  }

  return true;
}
