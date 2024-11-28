package main

import (
	"fmt"
	"log"
)

func NewNode(path string, slash_bits uint64) *Node {
	ret := Node{}
	ret.path_ = path
	ret.slash_bits_ = slash_bits
	return &ret
}

// / Return false on error.
func (this *Node) Stat(disk_interface *DiskInterface, err *string) bool {

}

// / If the file doesn't exist, set the mtime_ from its dependencies
func (this *Node) UpdatePhonyMtime(mtime TimeStamp) {}

// / Return false on error.
func (this *Node) StatIfNecessary(disk_interface DiskInterface, err *string) bool {
	if this.status_known() {
		return true
	}
	return this.Stat(disk_interface, err)
}

// / Mark as not-yet-stat()ed and not dirty.
func (this *Node) ResetState() {
	this.mtime_ = -1
	this.exists_ = ExistenceStatusUnknown
	this.dirty_ = false
}

// / Mark the Node as already-stat()ed and missing.
func (this *Node) MarkMissing() {
	if this.mtime_ == -1 {
		this.mtime_ = 0
	}
	this.exists_ = ExistenceStatusMissing
}

func (this *Node) exists() bool {
	return this.exists_ == ExistenceStatusExists
}

func (this *Node) status_known() bool {
	return this.exists_ != ExistenceStatusUnknown
}

func (this *Node) path() string {
	return this.path_
}

// / Get |path()| but use slash_bits to convert back to original slash styles.
func (this *Node) PathDecanonicalized() string {
	return PathDecanonicalized(this.path_, this.slash_bits_)
}
func PathDecanonicalized(path string, slash_bits uint64) string {

}
func (this *Node) slash_bits() uint64 { return this.slash_bits_ }

func (this *Node) mtime() TimeStamp { return this.mtime_ }

func (this *Node) dirty() bool          { return this.dirty_ }
func (this *Node) set_dirty(dirty bool) { this.dirty_ = dirty }
func (this *Node) MarkDirty()           { this.dirty_ = true }

func (this *Node) dyndep_pending() bool            { return this.dyndep_pending_ }
func (this *Node) set_dyndep_pending(pending bool) { this.dyndep_pending_ = pending }

func (this *Node) in_edge() *Edge         { return this.in_edge_ }
func (this *Node) set_in_edge(edge *Edge) { this.in_edge_ = edge }

// / Indicates whether this node was generated from a depfile or dyndep file,
// / instead of being a regular input or output from the Ninja manifest.
func (this *Node) generated_by_dep_loader() bool { return this.generated_by_dep_loader_ }

func (this *Node) set_generated_by_dep_loader(value bool) {
	this.generated_by_dep_loader_ = value
}

func (this *Node) id() int       { return this.id_ }
func (this *Node) set_id(id int) { this.id_ = id }

func (this *Node) out_edges() []*Edge            { return this.out_edges_ }
func (this *Node) validation_out_edges() []*Edge { return this.validation_out_edges_ }
func (this *Node) AddOutEdge(edge *Edge)         { this.out_edges_ = append(this.out_edges_, edge) }
func (this *Node) AddValidationOutEdge(edge *Edge) {
	this.validation_out_edges_ = append(this.validation_out_edges_, edge)
}

func (this *Node) Dump(prefix string) {}

func (this *InputsCollector) VisitNode(node *Node) {}

// / Retrieve list of visited input nodes. A dependency always appears
// / before its dependents in the result, but final order depends on the
// / order of the VisitNode() calls performed before this.
func (this *InputsCollector) inputs() []string { return this.inputs_ }

// / Same as inputs(), but returns the list of visited nodes as a list of
// / strings, with optional shell escaping.
func (this *InputsCollector) GetInputsAsStrings(shell_escape bool) []string {

}

// / Reset collector state.
func (this *InputsCollector) Reset() {
	this.inputs_ = []*Node{}
	this.visited_nodes_= map[*Node]bool{}
}

func NewEdge() *Edge {
	ret := Edge{}
	return &ret
}

// / Return true if all inputs' in-edges are ready.
func (this*Edge )AllInputsReady() bool {
  for (vector<Node*>::const_iterator i = inputs_.begin();
       i != inputs_.end(); ++i) {
    if ((*i).in_edge() && !(*i).in_edge().outputs_ready()) {
		return false
	}
  }
  return true;
}


/// Expand all variables in a command and return it as a string.
/// If incl_rsp_file is enabled, the string will also contain the
/// full contents of a response file (if applicable)
func (this*Edge ) EvaluateCommand( incl_rsp_file bool) string {
   command := GetBinding("command");
  if (incl_rsp_file) {
    rspfile_content := GetBinding("rspfile_content");
    if (!rspfile_content.empty()) {
		command += ";rspfile=" + rspfile_content
	}
  }
  return command;
}

/// Returns the shell-escaped value of |key|.
func (this*Edge ) GetBinding(key string) string {
   env := NewEdgeEnv(this, kShellEscape);
  return env.LookupVariable(key);
}

 func (this*Edge )  GetBindingBool(key string) bool {
	   return !GetBinding(key).empty();
 }

/// Like GetBinding("depfile"), but without shell escaping.
func (this*Edge ) GetUnescapedDepfile() string{
   env := NewEdgeEnv(this, kDoNotEscape);
  return env.LookupVariable("depfile");
}

/// Like GetBinding("dyndep"), but without shell escaping.
func (this*Edge ) GetUnescapedDyndep()  string{
   env := NewEdgeEnv(this, kDoNotEscape);
  return env.LookupVariable("dyndep");
}
/// Like GetBinding("rspfile"), but without shell escaping.
func (this*Edge )GetUnescapedRspfile()  string{
  env := NewEdgeEnv(this, kDoNotEscape);
  return env.LookupVariable("rspfile");
}

func (this*Edge ) Dump(prefix string) {
	fmt.Printf("%s[ ", prefix);
  for (vector<Node*>::const_iterator i = inputs_.begin(); i != inputs_.end() && *i != NULL; ++i) {
		fmt.Printf("%s ", (*i).path().c_str());
  }
	fmt.Printf("--%s. ", rule_.name().c_str());
  for (vector<Node*>::const_iterator i = outputs_.begin(); i != outputs_.end() && *i != NULL; ++i) {
		fmt.Printf("%s ", (*i).path().c_str());
  }
  if (!validations_.empty()) {
	  fmt.Printf(" validations ");
    for (std::vector<Node*>::const_iterator i = validations_.begin();
         i != validations_.end() && *i != NULL; ++i) {
		  fmt.Printf("%s ", (*i).path().c_str());
    }
  }
  if (pool_) {
    if (!pool_.name().empty()) {
      fmt.Printf("(in pool '%s')", pool_.name().c_str());
    }
  } else {
	  fmt.Printf("(null pool?)");
  }
	fmt.Printf("] 0x%p\n", this);
}

// critical_path_weight is the priority during build scheduling. The
// "critical path" between this edge's inputs and any target node is
// the path which maximises the sum oof weights along that path.
// NOTE: Defaults to -1 as a marker smaller than any valid weight
func (this*Edge ) critical_path_weight() int64 { return this.critical_path_weight_; }
func (this*Edge ) set_critical_path_weight( critical_path_weight int64) {
	this.critical_path_weight_ = critical_path_weight;
}

func (this*Edge )  rule() *Rule { return this.rule_; }
func (this*Edge ) pool() *Pool  { return this.pool_; }
 func (this*Edge ) weight() int { return 1; }
 func (this*Edge ) outputs_ready() bool { return this.outputs_ready_; }

func (this*Edge )  is_implicit( index int64) bool {
	return index >= int64(len(this.inputs_) - this.order_only_deps_ - this.implicit_deps_) && !this.is_order_only(index);
}
func (this*Edge )  is_order_only( index int64) bool{
	return index >= int64(len(this.inputs_) - this.order_only_deps_)
}

func (this*Edge ) is_implicit_out(index int64) bool {
return index >= int64(len(this.outputs_) - this.implicit_outs_)
}

func (this*Edge ) is_phony() bool {
  return this.rule_ == &kPhonyRule;
}

func (this*Edge ) use_console() bool {
	  return pool() == &State::kConsolePool;
}

func (this*Edge )  maybe_phonycycle_diagnostic() bool {
  // CMake 2.8.12.x and 3.0.x produced self-referencing phony rules
  // of the form "build a: phony ... a ...".   Restrict our
  // "phonycycle" diagnostic option to the form it used.
  return is_phony() && outputs_.size() == 1 && implicit_outs_ == 0 &&
      implicit_deps_ == 0;
}




func NewDependencyScan(state *State, build_log * BuildLog, deps_log *DepsLog, disk_interface *DiskInterface,
	depfile_parser_options *DepfileParserOptions , explanations *Explanations) *DependencyScan {
	ret := DependencyScan{}
	ret.build_log_ = build_log
	ret.disk_interface_ = disk_interface
	ret.dep_loader_ = (state, deps_log, disk_interface, depfile_parser_options, explanations)
	ret.dyndep_loader_ = (state, disk_interface)
	ret.explanations_ = explanations
	return &ret
}

/// Update the |dirty_| state of the given nodes by transitively inspecting
/// their input edges.
/// Examine inputs, outputs, and command lines to judge whether an edge
/// needs to be re-run, and update outputs_ready_ and each outputs' |dirty_|
/// state accordingly.
/// Appends any validation nodes found to the nodes parameter.
/// Returns false on failure.
func (this*DependencyScan) RecomputeDirty(node *Node, validation_nodes []*Node, err *string) bool {
  std::vector<Node*> stack;
  std::vector<Node*> new_validation_nodes;

  std::deque<Node*> nodes(1, initial_node);

  // RecomputeNodeDirty might return new validation nodes that need to be
  // checked for dirty state, keep a queue of nodes to visit.
  while (!nodes.empty()) {
    node := nodes.front();
    nodes.pop_front();

    stack.clear();
    new_validation_nodes.clear();

    if (!RecomputeNodeDirty(node, &stack, &new_validation_nodes, err))
      return false;
    nodes.insert(nodes.end(), new_validation_nodes.begin(),
                              new_validation_nodes.end());
    if (!new_validation_nodes.empty()) {
      assert(validation_nodes &&
          "validations require RecomputeDirty to be called with validation_nodes");
      validation_nodes.insert(validation_nodes.end(),
                           new_validation_nodes.begin(),
                           new_validation_nodes.end());
    }
  }

  return true;
}

/// Recompute whether any output of the edge is dirty, if so sets |*dirty|.
/// Returns false on failure.
func (this*DependencyScan)  RecomputeOutputsDirty(edge *Edge,  most_recent_input *Node, dirty *bool,  err *string) bool {
   command := edge.EvaluateCommand(/*incl_rsp_file=*/true);
  for (vector<Node*>::iterator o = edge.outputs_.begin();o != edge.outputs_.end(); ++o) {
    if (RecomputeOutputDirty(edge, most_recent_input, command, *o)) {
      *outputs_dirty = true;
      return true;
    }
  }
  return true;
}

func (this*DependencyScan)  build_log() *BuildLog {
	return this.build_log_;
}
func (this*DependencyScan) set_build_log(log *BuildLog) {
	this.build_log_ = log;
}

func (this*DependencyScan)  deps_log() * DepsLog{
	return this.dep_loader_.deps_log();
}

/// Load a dyndep file from the given node's path and update the
/// build graph with the new information.  One overload accepts
/// a caller-owned 'DyndepFile' object in which to store the
/// information loaded from the dyndep file.
  func (this*DependencyScan)  LoadDyndeps(node *Node, err *string) bool {
	  return this.dyndep_loader_.LoadDyndeps(node, err)
  }

func (this*DependencyScan)  LoadDyndeps1(node *Node, ddf *DyndepFile, err *string) bool {
	return this.dyndep_loader_.LoadDyndeps1(node, ddf, err)
}

func (this*DependencyScan)  RecomputeNodeDirty(node *Node, stack []*Node, validation_nodes []*Node, err *string)bool {
  edge := node.in_edge();
  if (!edge) {
    // If we already visited this leaf node then we are done.
    if (node.status_known()) {
		return true
	}
    // This node has no in-edge; it is dirty if it is missing.
    if (!node.StatIfNecessary(disk_interface_, err)) {
		return false
	}
    if (!node.exists()) {
		explanations_.Record(node, "%s has no in-edge and is missing",
			node.path().c_str())
	}
    node.set_dirty(!node.exists());
    return true;
  }

  // If we already finished this edge then we are done.
  if (edge.mark_ == Edge::VisitDone){
		return true
	}

  // If we encountered this edge earlier in the call stack we have a cycle.
  if (!VerifyDAG(node, stack, err)) {
	  return false
  }

  // Mark the edge temporarily while in the call stack.
  edge.mark_ = Edge::VisitInStack;
  stack.push_back(node);

   dirty := false;
  edge.outputs_ready_ = true;
  edge.deps_missing_ = false;

  if (!edge.deps_loaded_) {
    // This is our first encounter with this edge.
    // If there is a pending dyndep file, visit it now:
    // * If the dyndep file is ready then load it now to get any
    //   additional inputs and outputs for this and other edges.
    //   Once the dyndep file is loaded it will no longer be pending
    //   if any other edges encounter it, but they will already have
    //   been updated.
    // * If the dyndep file is not ready then since is known to be an
    //   input to this edge, the edge will not be considered ready below.
    //   Later during the build the dyndep file will become ready and be
    //   loaded to update this edge before it can possibly be scheduled.
    if (edge.dyndep_ && edge.dyndep_.dyndep_pending()) {
      if (!RecomputeNodeDirty(edge.dyndep_, stack, validation_nodes, err)) {
		  return false
	  }

      if (!edge.dyndep_.in_edge() ||
          edge.dyndep_.in_edge().outputs_ready()) {
        // The dyndep file is ready, so load it now.
        if (!LoadDyndeps(edge.dyndep_, err)) {
			return false
		}
      }
    }
  }

  // Load output mtimes so we can compare them to the most recent input below.
  for (vector<Node*>::iterator o = edge.outputs_.begin();
       o != edge.outputs_.end(); ++o) {
    if (!(*o).StatIfNecessary(disk_interface_, err)) {
		return false
	}
  }

  if (!edge.deps_loaded_) {
    // This is our first encounter with this edge.  Load discovered deps.
    edge.deps_loaded_ = true;
    if (!dep_loader_.LoadDeps(edge, err)) {
      if (!err.empty()) {
		  return false
	  }
      // Failed to load dependency info: rebuild to regenerate it.
      // LoadDeps() did explanations_.Record() already, no need to do it here.
      dirty = edge.deps_missing_ = true;
    }
  }

  // Store any validation nodes from the edge for adding to the initial
  // nodes.  Don't recurse into them, that would trigger the dependency
  // cycle detector if the validation node depends on this node.
  // RecomputeDirty will add the validation nodes to the initial nodes
  // and recurse into them.
  validation_nodes.insert(validation_nodes.end(),
      edge.validations_.begin(), edge.validations_.end());

  // Visit all inputs; we're dirty if any of the inputs are dirty.
  Node* most_recent_input = NULL;
  for (vector<Node*>::iterator i = edge.inputs_.begin();
       i != edge.inputs_.end(); ++i) {
    // Visit this input.
    if (!RecomputeNodeDirty(*i, stack, validation_nodes, err))
      return false;

    // If an input is not ready, neither are our outputs.
    if (Edge* in_edge = (*i).in_edge()) {
      if (!in_edge.outputs_ready_) {
		  edge.outputs_ready_ = false
	  }
    }

    if (!edge.is_order_only(i - edge.inputs_.begin())) {
      // If a regular input is dirty (or missing), we're dirty.
      // Otherwise consider mtime.
      if ((*i).dirty()) {
        explanations_.Record(node, "%s is dirty", (*i).path().c_str());
        dirty = true;
      } else {
        if (!most_recent_input || (*i).mtime() > most_recent_input.mtime()) {
          most_recent_input = *i;
        }
      }
    }
  }

  // We may also be dirty due to output state: missing outputs, out of
  // date outputs, etc.  Visit all outputs and determine whether they're dirty.
  if (!dirty) {
	  if !RecomputeOutputsDirty(edge, most_recent_input, &dirty, err) {
		  return false
	  }
  }

  // Finally, visit each output and update their dirty state if necessary.
  for (vector<Node*>::iterator o = edge.outputs_.begin();o != edge.outputs_.end(); ++o) {
    if (dirty) {
		(*o).MarkDirty()
	}
  }

  // If an edge is dirty, its outputs are normally not ready.  (It's
  // possible to be clean but still not be ready in the presence of
  // order-only inputs.)
  // But phony edges with no inputs have nothing to do, so are always
  // ready.
  if (dirty && !(edge.is_phony() && edge.inputs_.empty())) {
	  edge.outputs_ready_ = false
  }

  // Mark the edge as finished during this walk now that it will no longer
  // be in the call stack.
  edge.mark_ = Edge::VisitDone;
  assert(stack.back() == node);
  stack.pop_back();

  return true;
}

func (this*DependencyScan)  VerifyDAG(node *Node, stack []*Node, err *string)bool {
  edge := node.in_edge();
  if edge ==nil {
	  panic("edge ==nil")
  }

  // If we have no temporary mark on the edge then we do not yet have a cycle.
  if edge.mark_ != VisitInStack {
		return true
 }


  // We have this edge earlier in the call stack.  Find it.
  vector<Node*>::iterator start = stack.begin();
  while (start != stack.end() && (*start).in_edge() != edge)
    ++start;
  assert(start != stack.end());

  // Make the cycle clear by reporting its start as the node at its end
  // instead of some other output of the starting edge.  For example,
  // running 'ninja b' on
  //   build a b: cat c
  //   build c: cat a
  // should report a . c . a instead of b . c . a.
  *start = node;

  // Construct the error message rejecting the cycle.
  *err = "dependency cycle: ";
  for (vector<Node*>::const_iterator i = start; i != stack.end(); ++i) {
    err.append((*i).path());
    err.append(" . ");
  }
  err.append((*start).path());

  if ((start + 1) == stack.end() && edge.maybe_phonycycle_diagnostic()) {
    // The manifest parser would have filtered out the self-referencing
    // input if it were not configured to allow the error.
    err.append(" [-w phonycycle=err]");
  }

  return false;
}

/// Recompute whether a given single output should be marked dirty.
/// Returns true if so.
func (this*DependencyScan) RecomputeOutputDirty(edge *Edge, most_recent_input *Edge, command string,  output *Node)bool {
  command := edge.EvaluateCommand(/*incl_rsp_file=*/true);
  for (vector<Node*>::iterator o = edge.outputs_.begin(); o != edge.outputs_.end(); ++o) {
    if (RecomputeOutputDirty(edge, most_recent_input, command, *o)) {
      *outputs_dirty = true;
      return true;
    }
  }
  return true;
}

func (this*DependencyScan) RecordExplanation(node *Node, fmt string, args...interface{}) {}


type EscapeKind int8
const(
	kShellEscape EscapeKind = 0
	kDoNotEscape  EscapeKind = 1
)


type EdgeEnv struct {
	Env
	lookups_ []string
	edge_ *Edge
	escape_in_out_ EscapeKind
	recursive_ bool
}

func NewEdgeEnv(edge *Edge, escape EscapeKind) * EdgeEnv {
	ret := EdgeEnv{}
	ret.edge_ = edge
	ret.escape_in_out_ = escape
	ret.recursive_ = false
	return &ret
}
func (this* EdgeEnv ) LookupVariable(var1 string) string {
  if (var1 == "in" || var1 == "in_newline") {
    explicit_deps_count := len(this.edge_.inputs_) - this.edge_.implicit_deps_ -
		this.edge_.order_only_deps_;
    return this.MakePathList(this.edge_.inputs_.data(), explicit_deps_count,
                        var1 == "in" ? ' ' : '\n');
  } else if (var1 == "out") {
    explicit_outs_count := len(this.edge_.outputs_) - this.edge_.implicit_outs_;
    return this.MakePathList(&this.edge_.outputs_[0], explicit_outs_count, ' ');
  }

  // Technical note about the lookups_ vector.
  //
  // This is used to detect cycles during recursive variable expansion
  // which can be seen as a graph traversal problem. Consider the following
  // example:
  //
  //    rule something
  //      command = $foo $foo $var1
  //      var1 = $var2
  //      var2 = $var3
  //      var3 = $var1
  //      foo = FOO
  //
  // Each variable definition can be seen as a node in a graph that looks
  // like the following:
  //
  //   command --> foo
  //      |
  //      v
  //    var1 <-----.
  //      |        |
  //      v        |
  //    var2 ---> var3
  //
  // The lookups_ vector is used as a stack of visited nodes/variables
  // during recursive expansion. Entering a node adds an item to the
  // stack, leaving the node removes it.
  //
  // The recursive_ flag is used as a small performance optimization
  // to never record the starting node in the stack when beginning a new
  // expansion, since in most cases, expansions are not recursive
  // at all.
  //
  if this.recursive_ {
    it := std::find(lookups_.begin(), lookups_.end(), var);
    if (it != this.lookups_.end()) {
      cycle := ""
      for (; it != this.lookups_.end(); ++it){
			cycle.append(*it + " . ")
	  }
      cycle.append(var1);
      log.Fatal("cycle in rule variables: " + cycle)
    }
  }

  // See notes on BindingEnv::LookupWithFallback.
  eval := this.edge_.rule_.GetBinding(var1);
  record_varname := this.recursive_ && eval
  if (record_varname) {
	  this.lookups_.push_back(var1)
  }

  // In practice, variables defined on rules never use another rule variable.
  // For performance, only start checking for cycles after the first lookup.
	this.recursive_ = true;
  result := this.edge_.env_.LookupWithFallback(var1, eval, this);
  if record_varname {
	  this.lookups_.pop_back()
  }
  return result;
}

/// Given a span of Nodes, construct a list of paths suitable for a command
/// line.
func (this* EdgeEnv )  MakePathList(span *Node,  size int64,  sep int32) string {
  string result;
  for (const Node* const* i = span; i != span + size; ++i) {
    if (!result.empty())
      result.push_back(sep);
    const string& path = (*i).PathDecanonicalized();
    if (escape_in_out_ == kShellEscape) {
      GetWin32EscapedString(path, &result);
    } else {
      result.append(path);
    }
  }
  return result;
}
