package main

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
func NewManifestParser(state *State, file_reader *FileReader, options *ManifestParserOptions) *ManifestParser {
	ret := ManifestParser{}
	ret.Parser = NewParser(state, file_reader)
	ret.options_ = options
	ret.quiet_ = false
	ret.env_ = &state.bindings_
	return &ret
}

// / Parse a text string of input.  Used by tests.
func (this *ManifestParser) ParseTest(input string, err string) bool {
	this.quiet_ = true
	return this.Parse("input", input, err)
}

// / Parse a file, given its contents as a string.
func (this *ManifestParser) Parse(filename, input string, err string) bool {
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
func (this *ManifestParser) ParsePool(err string) bool                             {
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

  while (this.lexer_.PeekToken(INDENT)) {
     key := ""
    value := EvalString{}
		if (!this.ParseLet(&key, &value, err)) {
		return false
	}

    if (key == "depth") {
       depth_string := value.Evaluate(env_);
      depth = atol(depth_string);
      if (depth < 0)
        return this.lexer_.Error("invalid pool depth", err);
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
func (this *ManifestParser) ParseRule(err string) bool                             {
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

  while (this.lexer_.PeekToken(INDENT)) {
     key := ""
     value := EvalString{}
    if (!this.ParseLet(&key, &value, err)) {
		return false
	}

    if (Rule::IsReservedBinding(key)) {
      rule.AddBinding(key, value);
    } else {
      // Die on other keyvals for now; revisit if we want to add a
      // scope here.
      return this.lexer_.Error("unexpected variable '" + key + "'", err);
    }
  }

  if (rule.bindings_["rspfile"].empty() !=  rule.bindings_["rspfile_content"].empty()) {
    return this.lexer_.Error("rspfile and rspfile_content need to be "
                        "both specified", err);
  }

  if (rule.bindings_["command"].empty()) {
	  return this.lexer_.Error("expected 'command =' line", err)
  }

	this.env_.AddRule(rule);
  return true;
}
func (this *ManifestParser) ParseLet(key string, value *EvalString, err string) bool {
  if (!this.lexer_.ReadIdent(key)) {
	  return this.lexer_.Error("expected variable name", err)
  }
  if (!this.ExpectToken(Lexer::EQUALS, err)){
	return false;
  }
  if (!this.lexer_.ReadVarValue(value, err)) {
	  return false
  }
  return true;
}
func (this *ManifestParser) ParseEdge(err string) bool                             {
	this.ins_.clear();
	this.outs_.clear();
	this.validations_.clear();

  {
     out := EvalString{}
    if (!this.lexer_.ReadPath(&out, err)) {
		return false
	}
    while (!out.empty()) {
	  this.outs_.push_back(std::move(out));

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
		this.outs_.push_back(std::move(out));
      ++implicit_outs;
    }
  }

  if (this.outs_.empty()) {
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
  if (!rule) {
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
	  this.ins_.push_back(std::move(in));
  }

  // Add all implicit deps, counting how many as we go.
  implicit := 0;
  if (lexer_.PeekToken(Lexer::PIPE)) {
    for (;;) {
      EvalString in;
      if (!lexer_.ReadPath(&in, err)){
			return false;
	  }
      if (in.empty()){
			break;
	  }
      ins_.push_back(std::move(in));
      ++implicit;
    }
  }

  // Add all order-only deps, counting how many as we go.
  order_only := 0;
  if (lexer_.PeekToken(Lexer::PIPE2)) {
    for (;;) {
      EvalString in;
      if (!lexer_.ReadPath(&in, err)){
		  return false;
	  }
      if (in.empty()){
			break;
	  }
      ins_.push_back(std::move(in));
      ++order_only;
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
      this.validations_.push_back(std::move(validation));
    }
  }

  if !this.ExpectToken(NEWLINE, err) {
	return false;
  }

  // Bindings on edges are rare, so allocate per-edge envs only when needed.
   has_indent_token := this.lexer_.PeekToken(INDENT);
   env := has_indent_token ? new BindingEnv(env_) : env_;
  while (has_indent_token) {
    string key;
    EvalString val;
    if (!ParseLet(&key, &val, err)) {
		return false
	}

    env.AddBinding(key, val.Evaluate(env_));
    has_indent_token = this.lexer_.PeekToken(Lexer::INDENT);
  }

  edge := this.state_.AddEdge(rule);
  edge.env_ = env;

  pool_name := edge.GetBinding("pool");
  if (!pool_name.empty()) {
    pool := this.state_.LookupPool(pool_name);
    if pool == nil{
		return this.lexer_.Error("unknown pool name '"+pool_name+"'", err)
	}
    edge.pool_ = pool;
  }

  edge.outputs_.reserve(this.outs_.size());
  for i := 0, e = this.outs_.size(); i != e; ++i {
    path := this.outs_[i].Evaluate(env);
    if (path.empty()) {
		return lexer_.Error("empty path", err)
	}
    slash_bits := uint64(0)
	CanonicalizePath(&path, &slash_bits);
    if (!state_.AddOut(edge, path, slash_bits, err)) {
      lexer_.Error(std::string(*err), err);
      return false;
    }
  }

  if (edge.outputs_.empty()) {
    // All outputs of the edge are already created by other edges. Don't add
    // this edge.  Do this check before input nodes are connected to the edge.
    this.state_.edges_.pop_back();
    delete edge;
    return true;
  }
  edge.implicit_outs_ = implicit_outs;

  edge.inputs_.reserve(ins_.size());
  for (vector<EvalString>::iterator i = ins_.begin(); i != ins_.end(); ++i) {
    string path = i.Evaluate(env);
    if (path.empty()) {
		return lexer_.Error("empty path", err)
	}
    uint64_t slash_bits;
    CanonicalizePath(&path, &slash_bits);
    state_.AddIn(edge, path, slash_bits);
  }
  edge.implicit_deps_ = implicit;
  edge.order_only_deps_ = order_only;

  edge.validations_.reserve(validations_.size());
  for (std::vector<EvalString>::iterator v = validations_.begin();
      v != validations_.end(); ++v) {
    string path = v.Evaluate(env);
    if (path.empty()) {
		return lexer_.Error("empty path", err)
	}
    uint64_t slash_bits;
    CanonicalizePath(&path, &slash_bits);
    state_.AddValidation(edge, path, slash_bits);
  }

  if (options_.phony_cycle_action_ == kPhonyCycleActionWarn &&
      edge.maybe_phonycycle_diagnostic()) {
    // CMake 2.8.12.x and 3.0.x incorrectly write phony build statements
    // that reference themselves.  Ninja used to tolerate these in the
    // build graph but that has since been fixed.  Filter them out to
    // support users of those old CMake versions.
    Node* out = edge.outputs_[0];
    vector<Node*>::iterator new_end =
        remove(edge.inputs_.begin(), edge.inputs_.end(), out);
    if (new_end != edge.inputs_.end()) {
      edge.inputs_.erase(new_end, edge.inputs_.end());
      if (!quiet_) {
        Warning("phony target '%s' names itself as an input; "
                "ignoring [-w phonycycle=warn]",
                out.path());
      }
    }
  }

  // Lookup, validate, and save any dyndep binding.  It will be used later
  // to load generated dependency information dynamically, but it must
  // be one of our manifest-specified inputs.
  dyndep := edge.GetUnescapedDyndep();
  if (!dyndep.empty()) {
    uint64_t slash_bits;
    CanonicalizePath(&dyndep, &slash_bits);
    edge.dyndep_ = state_.GetNode(dyndep, slash_bits);
    edge.dyndep_.set_dyndep_pending(true);
    vector<Node*>::iterator dgi =
      std::find(edge.inputs_.begin(), edge.inputs_.end(), edge.dyndep_);
    if (dgi == edge.inputs_.end()) {
      return lexer_.Error("dyndep '" + dyndep + "' is not an input", err);
    }
    assert(!edge.dyndep_.generated_by_dep_loader());
  }

  return true;
}
func (this *ManifestParser) ParseDefault(err string) bool                          {
  EvalString eval;
  if (!lexer_.ReadPath(&eval, err))
    return false;
  if (eval.empty()) {
	  return lexer_.Error("expected target name", err)
  }

  do {
    string path = eval.Evaluate(env_);
    if (path.empty())
      return lexer_.Error("empty path", err);
    uint64_t slash_bits;  // Unused because this only does lookup.
    CanonicalizePath(&path, &slash_bits);
    std::string default_err;
    if (!state_.AddDefault(path, &default_err))
      return lexer_.Error(default_err, err);

    eval.Clear();
    if (!lexer_.ReadPath(&eval, err))
      return false;
  } while (!eval.empty());

  return ExpectToken(Lexer::NEWLINE, err);
}

// / Parse either a 'subninja' or 'include' line.
func (this *ManifestParser) ParseFileInclude(new_scope bool, err string) bool {
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
