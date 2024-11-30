package main

type DyndepParser struct {
	Parser
	dyndep_file_ DyndepFile
	env_         BindingEnv
}

func NewDyndepParser(state *State, file_reader FileReader, dyndep_file DyndepFile) *DyndepParser {
	ret := DyndepParser{}
	ret.Parser = *NewParser(state, file_reader)
	ret.dyndep_file_ = dyndep_file
	return &ret
}

// / Parse a text string of input.  Used by tests.
func (this *DyndepParser) ParseTest(input string, err *string) bool {
	return this.Parse("input", input, err)
}

// / Parse a file, given its contents as a string.
func (this *DyndepParser) Parse(filename string, input string, err *string) bool {
  this.lexer_.Start(filename, input);

  // Require a supported ninja_dyndep_version value immediately so
  // we can exit before encountering any syntactic surprises.
  haveDyndepVersion := false;

  for  {
    token := this.lexer_.ReadToken();
    switch (token) {
    case BUILD: {
      if (!haveDyndepVersion) {
        return this.lexer_.Error("expected 'ninja_dyndep_version = ...'", err)
      }
      if !this.ParseEdge(err) {
        return false
      }
    }
    case IDENT: {
      this.lexer_.UnreadToken();
      if (haveDyndepVersion) {
        return this.lexer_.Error(string("unexpected ")+TokenName(token), err)
      }
      if !this.ParseDyndepVersion(err) {
        return false
      }
      haveDyndepVersion = true;
    }
    case ERROR:
      return this.lexer_.Error(this.lexer_.DescribeLastError(), err);
    case TEOF:
      if (!haveDyndepVersion) {
        return this.lexer_.Error("expected 'ninja_dyndep_version = ...'", err)
      }
      return true;
    case NEWLINE:
      return false;
    default:
      return this.lexer_.Error(string("unexpected ") + TokenName(token), err);
    }
  }
  return false;  // not reached
}

func (this *DyndepParser) ParseDyndepVersion(err *string) bool                     {
  name := ""
   let_value := EvalString{}
  if !this.ParseLet(&name, &let_value, err) {
    return false
  }
  if name != "ninja_dyndep_version" {
    return this.lexer_.Error("expected 'ninja_dyndep_version = ...'", err);
  }
  version := let_value.Evaluate(&this.env_);
  major := 0
  minor := 0
  ParseVersion(version, &major, &minor);
  if (major != 1 || minor != 0) {
    return this.lexer_.Error(string("unsupported 'ninja_dyndep_version = ") + version + "'", err);
  }
  return true;
}
func (this *DyndepParser) ParseLet(key *string, value *EvalString, err *string) bool {
  if (!this.lexer_.ReadIdent(key)) {
    return this.lexer_.Error("expected variable name", err)
  }
  return (this.ExpectToken(EQUALS, err) && this.lexer_.ReadVarValue(value, err));
}
func (this *DyndepParser) ParseEdge(err *string) bool                              {
  // Parse one explicit output.  We expect it to already have an edge.
  // We will record its dynamically-discovered dependency information.
  var dyndeps *Dyndeps= nil
  {
    var  out0 EvalString
    if (!this.lexer_.ReadPath(&out0, err)) {
      return false
    }
    if (out0.empty()) {
      return this.lexer_.Error("expected path", err)
    }

    path := out0.Evaluate(&this.env_);
    if path=="" {
      return this.lexer_.Error("empty path", err)
    }
     slash_bits := uint64(0)
    CanonicalizePath(&path, &slash_bits);
    node := this.state_.LookupNode(path);
    if node==nil || node.in_edge()==nil {
      return this.lexer_.Error("no build statement exists for '"+path+"'", err)
    }
    edge := node.in_edge();
    res := DyndepFile::value_type(edge, Dyndeps())
    this.dyndep_file_[res] = true
    if res ==nil {
      return this.lexer_.Error("multiple statements for '"+path+"'", err)
    }
    dyndeps = &res.first.second;
  }

  // Disallow explicit outputs.
  {
     out := EvalString{}
    if !this.lexer_.ReadPath(&out, err) {
      return false
    }
    if (!out.empty()) {
      return this.lexer_.Error("explicit outputs not supported", err)
    }
  }

  // Parse implicit outputs, if any.
  outs := []EvalString{}
  if this.lexer_.PeekToken(PIPE) {
    for  {
       out := EvalString{}
      if (!this.lexer_.ReadPath(&out, err)) {
        return err
      }
      if (out.empty()) {
        break
      }
      outs = append(outs, out);
    }
  }

  if !this.ExpectToken(COLON, err) {
    return false;
  }

   rule_name := ""
  if !this.lexer_.ReadIdent(&rule_name) || rule_name != "dyndep" {
    return this.lexer_.Error("expected build command name 'dyndep'", err)
  }

  // Disallow explicit inputs.
  {
    in := EvalString{}
    if (!this.lexer_.ReadPath(&in, err)) {
      return false
    }
    if (!in.empty()) {
      return this.lexer_.Error("explicit inputs not supported", err)
    }
  }

  // Parse implicit inputs, if any.
  ins := []EvalString{}
  if this.lexer_.PeekToken(PIPE) {
    for  {
       in := EvalString{}
      if !this.lexer_.ReadPath(&in, err) {
        return err
      }
      if (in.empty()) {
        break
      }
      ins = append(ins, in)
    }
  }

  // Disallow order-only inputs.
  if this.lexer_.PeekToken(PIPE2) {
    return this.lexer_.Error("order-only inputs not supported", err);
  }

  if !this.ExpectToken(NEWLINE, err) {
    return false;
  }

  if this.lexer_.PeekToken(INDENT) {
    key := ""
     val := EvalString{}
    if (!this.ParseLet(&key, &val, err)){
      return false;
    }
    if (key != "restat"){
      return this.lexer_.Error("binding is not 'restat'", err);
    }
    value := val.Evaluate(&this.env_)
    dyndeps.restat_ = value!=""
  }

  for _, in :=range ins {
    path := in.Evaluate(&this.env_);
    if path=="" {
      return this.lexer_.Error("empty path", err)
    }
    slash_bits := uint64(0)
    CanonicalizePath(&path, &slash_bits);
    n := this.state_.GetNode(path, slash_bits);
    dyndeps.implicit_inputs_ =append(dyndeps.implicit_inputs_, n);
  }

  for _,out :=range outs {
    path := out.Evaluate(&this.env_);
    if path=="" {
      return this.lexer_.Error("empty path", err)
    }
    slash_bits := uint64(0)
    CanonicalizePath(&path, &slash_bits);
    n := this.state_.GetNode(path, slash_bits);
    dyndeps.implicit_outputs_ = append(dyndeps.implicit_outputs_, n)
  }

  return true;
}
