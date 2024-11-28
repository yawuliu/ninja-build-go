package main

import "log"

type Plan struct {
  /// Keep track of which edges we want to build in this plan.  If this map does
  /// not contain an entry for an edge, we do not want to build the entry or its
  /// dependents.  If it does contain an entry, the enumeration indicates what
  /// we want for the edge.
  want_ map[*Edge]Want

   ready_ EdgePriorityQueue

  builder_ *Builder
  /// user provided targets in build order, earlier one have higher priority
  targets_ []*Node

  /// Total number of edges that have commands (not phony).
  command_edges_ int

  /// Total remaining number of wanted edges.
  wanted_edges_ int
}
func NewPlan(builder *Builder) *Plan {
  return &Plan{builder_: builder, want_: make(map[*Edge]Want)}
}
/// Add a target to our plan (including all its dependencies).
/// Returns false if we don't need to build this target; may
/// fill in |err| with an error message if there's a problem.
func (p *Plan) AddTarget(target *Node, err *string) bool {
  // 实现添加目标到计划的逻辑
  return true
}
// Pop a ready edge off the queue of edges to build.
// Returns NULL if there's no work to do.
func (p *Plan) FindWork() *Edge {
  // 实现查找工作的逻辑
  return nil
}
/// Returns true if there's more work to be done.
func (p *Plan) more_to_do() bool {
  return p.wanted_edges_ > 0 && p.command_edges_ > 0
}
/// Dumps the current state of the plan.
func (p *Plan) Dump() {
  // 实现计划转储的逻辑
}
type EdgeResult int8
const (
  kEdgeFailed EdgeResult =0
  kEdgeSucceeded EdgeResult =1
)

/// Mark an edge as done building (whether it succeeded or failed).
/// If any of the edge's outputs are dyndep bindings of their dependents,
/// this loads dynamic dependencies from the nodes' paths.
/// Returns 'false' if loading dyndep info fails and 'true' otherwise.
func (p *Plan) EdgeFinished(edge *Edge,  result EdgeResult, err  *string) bool {}

/// Clean the given node during the build.
/// Return false on error.
func (p *Plan) CleanNode(scan *DependencyScan,  node *Node, err *string) bool{

}

/// Number of edges with commands to run.
func (this *Plan)  command_edge_count() int { return this.command_edges_; }

/// Reset state.  Clears want and ready sets.
func (p *Plan) Reset() {}

// After all targets have been added, prepares the ready queue for find work.
func (p *Plan) PrepareQueue() {}

/// Update the build plan to account for modifications made to the graph
/// by information loaded from a dyndep file.
func (p *Plan)  DyndepsLoaded(scan *DependencyScan, node *Node, ddf *DyndepFile,  err *string) bool {

}
/// Enumerate possible steps we want for an edge.
type Want int8
const (
  /// We do not want to build the edge, but we might want to build one of
  /// its dependents.
  kWantNothing Want = 0
  /// We want to build the edge, but have not yet scheduled it.
  kWantToStart Want = 1
  /// We want to build the edge, have scheduled it, and are waiting
  /// for it to complete.
  kWantToFinish Want = 2
)

func (p *Plan)   ComputeCriticalPath(){}
func (p *Plan)  RefreshDyndepDependents(scan *DependencyScan, node *Node, err *string) bool{}
func (p *Plan)   UnmarkDependents(node  *Node,  dependents map[*Node]bool){}
 func (p *Plan)  AddSubTarget(node  *Node, dependent *Node,  err *string, dyndep_walk  map[*Edge]bool) bool {}

// Add edges that kWantToStart into the ready queue
// Must be called after ComputeCriticalPath and before FindWork
func (p *Plan)   ScheduleInitialEdges(){}

/// Update plan with knowledge that the given node is up to date.
/// If the node is a dyndep binding on any of its dependents, this
/// loads dynamic dependencies from the node's path.
/// Returns 'false' if loading dyndep info fails and 'true' otherwise.
func (p *Plan) NodeFinished( node *Node, err *string) bool {}

func (p *Plan)   EdgeWanted(edge *Edge) {}
func (p *Plan) EdgeMaybeReady(want_e Want , err *string) bool {}

/// Submits a ready edge as a candidate for execution.
/// The edge may be delayed from running, for example if it's a member of a
/// currently-full pool.
func (p *Plan)   ScheduleWork(want_e Want){}
/// The result of waiting for a command.
type Result struct{
  edge *Edge
  status ExitStatus
  output string
}
func NewResult() *Result{
  ret := Result{}
  ret.edge = nil
  return &ret
}
func(this*Result) success() bool { return this.status == ExitSuccess; }

type CommandRunner interface {
  ReleaseCommandRunner()
  StartCommand(edge *Edge) bool
  WaitForCommand() (Result, bool)
  GetActiveEdges() []*Edge
  CanRunMore()int64
  Abort()
}

func CommandRunnerfactory(config *BuildConfig) CommandRunner {

}

type DryRunCommandRunner struct{}

func (d *DryRunCommandRunner) StartCommand(edge *Edge) bool {
  // 实现启动命令的逻辑
  return true
}

func (d *DryRunCommandRunner) WaitForCommand() (Result, bool) {
  // 实现等待命令完成的逻辑
  return Result{}, true
}

func (d *DryRunCommandRunner) GetActiveEdges() []*Edge {
  return []*Edge{}
}

func (d *DryRunCommandRunner) Abort() {}

func NewBuilder(state *State, config *BuildConfig, build_log *BuildLog,
	deps_log *DepsLog, disk_interface DiskInterface, status Status,
	start_time_millis int64) *Builder {
	ret := Builder{}
	ret.state_= state
	ret.config_ = config
	ret.plan_ = &ret
	ret.status_ = status
	ret.start_time_millis_ = start_time_millis
	ret.disk_interface_ = disk_interface
	if g_explaining {
		ret.explanations_ = NewExplanations()
	} else {
		ret.explanations_ = nil
	}
	ret.scan_ = NewDependencyScan(state, build_log, deps_log, disk_interface, &ret.config_.depfile_parser_options, ret.explanations_.get())
	ret.lock_file_path_ = ".ninja_lock"
	build_dir := ret.state_.bindings_.LookupVariable("builddir")
	if build_dir!="" {
		ret.lock_file_path_ = build_dir + "/" + ret.lock_file_path_
	}
	ret.status_.SetExplanations(ret.explanations_.get())
	return &ret
}

func (this*Builder) RealeaseBuilder() {
	this.Cleanup()
	this.status_.SetExplanations(nil)
}


/// Clean up after interrupted commands by deleting output files.
func (this*Builder) Cleanup() {
  if this.command_runner_.get() {
    active_edges := this.command_runner_.GetActiveEdges();
    this.command_runner_.Abort();

    for _, e := range active_edges {
      depfile := e.GetUnescapedDepfile();
      for _, o := range e.outputs_ {
        // Only delete this output if it was actually modified.  This is
        // important for things like the generator where we don't want to
        // delete the manifest file if we can avoid it.  But if the rule
        // uses a depfile, always delete.  (Consider the case where we
        // need to rebuild an output because of a modified header file
        // mentioned in a depfile, and the command touches its depfile
        // but is interrupted before it touches its output file.)
        err := ""
        new_mtime := this.disk_interface_.Stat(o.path(), &err);
        if (new_mtime == -1) { // Log and ignore Stat() errors.
          this.status_.Error("%s", err)
        }
        if !depfile.empty() || o.mtime() != new_mtime {
          this.disk_interface_.RemoveFile(o.path())
        }
      }
      if !depfile.empty() {
        this.disk_interface_.RemoveFile(depfile)
      }
    }
  }

   err := ""
  if this.disk_interface_.Stat(this.lock_file_path_, &err) > 0 {
    this.disk_interface_.RemoveFile(this.lock_file_path_)
  }
}

func (this*Builder)  AddTarget( name string,  err *string) *Node {
node := this.state_.LookupNode(name);
  if node==nil {
    *err = "unknown target: '" + name + "'";
    return nil;
  }
  if !this.AddTarget2(node, err) {
    return nil
  }
  return node;
}

/// Add a target to the build, scanning dependencies.
/// @return false on error.
func (this*Builder)  AddTarget2( target *Node, err *string) bool {
  validation_nodes := []*Node{}
  if !this.scan_.RecomputeDirty(target, validation_nodes, err) {
    return false
  }

  in_edge := target.in_edge()
  if in_edge==nil || !in_edge.outputs_ready() {
    if !this.plan_.AddTarget(target, err) {
      return false;
    }
  }

  // Also add any validation nodes found during RecomputeDirty as top level
  // targets.
  for _, n:= range validation_nodes {
    validation_in_edge := n.in_edge()
    if validation_in_edge!=nil {
      if !validation_in_edge.outputs_ready() && !this.plan_.AddTarget(*n, err) {
        return false;
      }
    }
  }

  return true;
}

/// Returns true if the build targets are already up to date.
func (this*Builder)  AlreadyUpToDate() bool {
	return !this.plan_.more_to_do();
}

/// Run the build.  Returns false on error.
/// It is an error to call this function when AlreadyUpToDate() is true.
func (this*Builder)  Build(err *string) bool {
  if !this.AlreadyUpToDate() {
    panic("!AlreadyUpToDate() ")
  }
	this.plan_.PrepareQueue();

  pending_commands := 0;
  failures_allowed := this.config_.failures_allowed;

  // Set up the command runner if we haven't done so already.
  if !this.command_runner_.get() {
    if this.config_.dry_run {
		this.command_runner_.reset(NewDryRunCommandRunner)
	}else{
		 this.command_runner_.reset(CommandRunner::factory(this.config_))
	}
  }

  // We are about to start the build process.
  this.status_.BuildStarted();

  // This main loop runs the entire build process.
  // It is structured like this:
  // First, we attempt to start as many commands as allowed by the
  // command runner.
  // Second, we attempt to wait for / reap the next finished command.
  while (this.plan_.more_to_do()) {
    // See if we can start any more commands.
    if (failures_allowed) {
      capacity := this.command_runner_.CanRunMore();
      while (capacity > 0) {
        edge := this.plan_.FindWork();
        if (!edge) {
			break
		}

        if (edge.GetBindingBool("generator")) {
			this.scan_.build_log().Close();
        }

        if (!this.StartEdge(edge, err)) {
			this.Cleanup();
			this.status_.BuildFinished();
          return false;
        }

        if (edge.is_phony()) {
          if (!this.plan_.EdgeFinished(edge, Plan::kEdgeSucceeded, err)) {
            Cleanup();
            status_.BuildFinished();
            return false;
          }
        } else {
          ++pending_commands;

          --capacity;

          // Re-evaluate capacity.
          current_capacity := command_runner_.CanRunMore();
          if (current_capacity < capacity) {
			  capacity = current_capacity
		  }
        }
      }

       // We are finished with all work items and have no pending
       // commands. Therefore, break out of the main loop.
       if (pending_commands == 0 && !this.plan_.more_to_do()) {
		   break
	   }
    }

    // See if we can reap any finished commands.
    if (pending_commands) {
      CommandRunner::Result result;
      if (!this.command_runner_.WaitForCommand(&result) ||
          result.status == ExitInterrupted) {
		  this.Cleanup();
		  this.status_.BuildFinished();
        *err = "interrupted by user";
        return false;
      }

      --pending_commands;
      if (!this.FinishCommand(&result, err)) {
		  this.Cleanup();
		  this.status_.BuildFinished();
        return false;
      }

      if (!result.success()) {
        if (failures_allowed)
          failures_allowed--;
      }

      // We made some progress; start the main loop over.
      continue;
    }

    // If we get here, we cannot make any more progress.
		this.status_.BuildFinished();
    if (failures_allowed == 0) {
      if (this.config_.failures_allowed > 1)
        *err = "subcommands failed";
      else
        *err = "subcommand failed";
    } else if (failures_allowed < this.config_.failures_allowed)
      *err = "cannot make progress due to previous errors";
    else
      *err = "stuck [this is a bug]";

    return false;
  }

	this.status_.BuildFinished();
  return true;
}

func (this*Builder)  StartEdge(edge *Edge, err *string) bool {
  METRIC_RECORD("StartEdge");
  if (edge.is_phony()) {
    return true
  }

  start_time_millis := GetTimeMillis() - this.start_time_millis_;
	this.running_edges_.insert(make_pair(edge, start_time_millis));

  this.status_.BuildEdgeStarted(edge, start_time_millis);

  var build_start TimeStamp  = this.config_.dry_run ? 0 : -1;

  // Create directories necessary for outputs and remember the current
  // filesystem mtime to record later
  // XXX: this will block; do we care?
  for (vector<Node*>::iterator o = edge.outputs_.begin();
       o != edge.outputs_.end(); ++o) {
    if (! this.disk_interface_.MakeDirs((*o).path())) {
      return false
    }
    if (build_start == -1) {
      this.disk_interface_.WriteFile( this.lock_file_path_, "");
      build_start =  this.disk_interface_.Stat( this.lock_file_path_, err);
      if (build_start == -1) {
        build_start = 0
      }
    }
  }

  edge.command_start_time_ = build_start;

  // Create depfile directory if needed.
  // XXX: this may also block; do we care?
  depfile := edge.GetUnescapedDepfile();
  if (!depfile.empty() && ! this.disk_interface_.MakeDirs(depfile)) {
	  return false
  }

  // Create response file, if needed
  // XXX: this may also block; do we care?
  rspfile := edge.GetUnescapedRspfile();
  if (!rspfile.empty()) {
    content := edge.GetBinding("rspfile_content");
    if (! this.disk_interface_.WriteFile(rspfile, content)) {
		return false
	}
  }

  // start command computing and run it
  if (!this.command_runner_.StartCommand(edge)) {
    err.assign("command '" + edge.EvaluateCommand() + "' failed.");
    return false;
  }

  return true;
}

/// Update status ninja logs following a command termination.
/// @return false if the build can not proceed further due to a fatal error.
func (this*Builder)  FinishCommand(CommandRunner::Result* result,  err *string)bool {
  METRIC_RECORD("FinishCommand");

  edge := result.edge;

  // First try to extract dependencies from the result, if any.
  // This must happen first as it filters the command output (we want
  // to filter /showIncludes output, even on compile failure) and
  // extraction itself can fail, which makes the command fail from a
  // build perspective.
  deps_nodes := []*Node{}
   deps_type := edge.GetBinding("deps");
  deps_prefix := edge.GetBinding("msvc_deps_prefix");
  if (!deps_type.empty()) {
    extract_err :=""
    if (! this.ExtractDeps(result, deps_type, deps_prefix, &deps_nodes,
                     &extract_err) &&
        result.success()) {
      if (!result.output.empty()) {
        result.output.append("\n")
      }
      result.output.append(extract_err);
      result.status = ExitFailure;
    }
  }

   start_time_millis := int64(0)
  end_time_millis := int64(0)
  RunningEdgeMap::iterator it =  this.running_edges_.find(edge);
  start_time_millis = it.second;
  end_time_millis = GetTimeMillis() -  this.start_time_millis_;
  this.running_edges_.erase(it);

  this.status_.BuildEdgeFinished(edge, start_time_millis, end_time_millis,
                             result.success(), result.output);

  // The rest of this function only applies to successful commands.
  if (!result.success()) {
    return this.plan_.EdgeFinished(edge, Plan::kEdgeFailed, err);
  }

  // Restat the edge outputs
  var record_mtime TimeStamp = 0;
  if (!this.config_.dry_run) {
     restat := edge.GetBindingBool("restat");
    generator := edge.GetBindingBool("generator");
    node_cleaned := false;
    record_mtime = edge.command_start_time_;

    // restat and generator rules must restat the outputs after the build
    // has finished. if record_mtime == 0, then there was an error while
    // attempting to touch/stat the temp file when the edge started and
    // we should fall back to recording the outputs' current mtime in the
    // log.
    if (record_mtime == 0 || restat || generator) {
      for (vector<Node*>::iterator o = edge.outputs_.begin();o != edge.outputs_.end(); ++o) {
        var  new_mtime TimeStamp = this.disk_interface_.Stat((*o).path(), err);
        if (new_mtime == -1) {
			return false
		}
        if (new_mtime > record_mtime) {
			record_mtime = new_mtime
		}
        if ((*o).mtime() == new_mtime && restat) {
          // The rule command did not change the output.  Propagate the clean
          // state through the build graph.
          // Note that this also applies to nonexistent outputs (mtime == 0).
          if (! this.plan_.CleanNode(& this.scan_, *o, err)) {
			  return false
		  }
          node_cleaned = true;
        }
      }
    }
    if (node_cleaned) {
      record_mtime = edge.command_start_time_;
    }
  }

  if (!plan_.EdgeFinished(edge, Plan::kEdgeSucceeded, err)){
	return false;
	}

  // Delete any left over response file.
  rspfile := edge.GetUnescapedRspfile();
  if (!rspfile.empty() && !g_keep_rsp) {
    this.disk_interface_. RemoveFile(rspfile)
  }

  if ( this.scan_.build_log()) {
    if (! this.scan_.build_log().RecordCommand(edge, start_time_millis,
                                          end_time_millis, record_mtime)) {
      *err = string("Error writing to build log: ") + strerror(errno);
      return false;
    }
  }

  if (!deps_type.empty() && !config_.dry_run) {
    assert(!edge.outputs_.empty() && "should have been rejected by parser");
    for (std::vector<Node*>::const_iterator o = edge.outputs_.begin();o != edge.outputs_.end(); o++) {
       var deps_mtime TimeStamp = disk_interface_.Stat((*o).path(), err);
      if (deps_mtime == -1) {
		  return false
	  }
      if (!scan_.deps_log().RecordDeps(*o, deps_mtime, deps_nodes)) {
        *err = std::string("Error writing to deps log: ") + strerror(errno);
        return false;
      }
    }
  }
  return true;
}

/// Used for tests.
func (this*Builder)  SetBuildLog(log*BuildLog) {
	this.scan_.set_build_log(log);
}



func (this*Builder) ExtractDeps(CommandRunner::Result* result, deps_type string, deps_prefix string, deps_nodes []*Node, err *string) bool {
  if (deps_type == "msvc") {
     parser := CLParser{}
    output := ""
    if (!parser.Parse(result.output, deps_prefix, &output, err)) {
      return false
    }
    result.output = output;
    for (set<string>::iterator i = parser.includes_.begin();i != parser.includes_.end(); ++i) {
      // ~0 is assuming that with MSVC-parsed headers, it's ok to always make
      // all backslashes (as some of the slashes will certainly be backslashes
      // anyway). This could be fixed if necessary with some additional
      // complexity in IncludesNormalize::Relativize.
      deps_nodes.push_back(state_.GetNode(*i, ~0u));
    }
  } else if (deps_type == "gcc") {
    depfile := result.edge.GetUnescapedDepfile();
    if (depfile.empty()) {
      *err = string("edge with deps=gcc but no depfile makes no sense");
      return false;
    }

    // Read depfile content.  Treat a missing depfile as empty.
    content := ""
    switch (this.disk_interface_.ReadFile(depfile, &content, err)) {
    case NotFound:
      err.clear();
    case OtherError:
      return false;
    case Okay:
    }
    if (content.empty()) {
      return true
    }

     deps := NewDepfileParser( this.config_.depfile_parser_options);
    if (!deps.Parse(&content, err)) {
      return false
    }

    // XXX check depfile matches expected output.
    deps_nodes.reserve(deps.ins_.size());
    for (vector<string>::iterator i = deps.ins_.begin();
         i != deps.ins_.end(); ++i) {
      var slash_bits uint64 = 0
      CanonicalizePath(const_cast<char*>(i.str_), &i.len_, &slash_bits);
      deps_nodes.push_back(state_.GetNode(*i, slash_bits));
    }

    if (!g_keep_depfile) {
      if (this.disk_interface_.RemoveFile(depfile) < 0) {
        *err = string("deleting depfile: ") + strerror(errno) + string("\n");
        return false;
      }
    }
  } else {
    log.Fatalf("unknown deps type '%s'", deps_type);
  }

  return true;
}
/// Load the dyndep information provided by the given node.
func (this*Builder) LoadDyndeps( node *Node,  err *string) bool{
  // Load the dyndep information provided by this node.
   ddf := DyndepFile{}
  if (!this.scan_.LoadDyndeps(node, &ddf, err)) {
    return false
  }

  // Update the build plan to account for dyndep modifications to the graph.
  if (!this.plan_.DyndepsLoaded(&this.scan_, node, ddf, err)) {
    return false
  }

  return true;
}
