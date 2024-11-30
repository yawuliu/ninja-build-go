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
func (p *Plan) EdgeFinished(edge *Edge,  result EdgeResult, err  *string) bool {
  map<Edge*, Want>::iterator e = want_.find(edge);
  assert(e != want_.end());
  bool directly_wanted = e.second != kWantNothing;

  // See if this job frees up any delayed jobs.
  if (directly_wanted)
    edge.pool().EdgeFinished(*edge);
  edge.pool().RetrieveReadyEdges(&ready_);

  // The rest of this function only applies to successful commands.
  if (result != kEdgeSucceeded)
    return true;

  if (directly_wanted)
    --wanted_edges_;
  want_.erase(e);
  edge.outputs_ready_ = true;

  // Check off any nodes we were waiting for with this edge.
  for (vector<Node*>::iterator o = edge.outputs_.begin();
       o != edge.outputs_.end(); ++o) {
    if (!NodeFinished(*o, err))
      return false;
  }
  return true;
}

/// Clean the given node during the build.
/// Return false on error.
func (p *Plan) CleanNode(scan *DependencyScan,  node *Node, err *string) bool{
  node.set_dirty(false);

  for (vector<Edge*>::const_iterator oe = node.out_edges().begin();
       oe != node.out_edges().end(); ++oe) {
    // Don't process edges that we don't actually want.
    map<Edge*, Want>::iterator want_e = want_.find(*oe);
    if (want_e == want_.end() || want_e.second == kWantNothing)
      continue;

    // Don't attempt to clean an edge if it failed to load deps.
    if ((*oe).deps_missing_)
      continue;

    // If all non-order-only inputs for this edge are now clean,
    // we might have changed the dirty state of the outputs.
    vector<Node*>::iterator
        begin = (*oe).inputs_.begin(),
        end = (*oe).inputs_.end() - (*oe).order_only_deps_;
#if __cplusplus < 201703L
#define MEM_FN mem_fun
#else
#define MEM_FN mem_fn  // mem_fun was removed in C++17.
#endif
    if (find_if(begin, end, MEM_FN(&Node::dirty)) == end) {
      // Recompute most_recent_input.
      Node* most_recent_input = NULL;
      for (vector<Node*>::iterator i = begin; i != end; ++i) {
        if (!most_recent_input || (*i).mtime() > most_recent_input.mtime())
          most_recent_input = *i;
      }

      // Now, this edge is dirty if any of the outputs are dirty.
      // If the edge isn't dirty, clean the outputs and mark the edge as not
      // wanted.
      bool outputs_dirty = false;
      if (!scan.RecomputeOutputsDirty(*oe, most_recent_input,
                                       &outputs_dirty, err)) {
        return false;
      }
      if (!outputs_dirty) {
        for (vector<Node*>::iterator o = (*oe).outputs_.begin();
             o != (*oe).outputs_.end(); ++o) {
          if (!CleanNode(scan, *o, err))
            return false;
        }

        want_e.second = kWantNothing;
        --wanted_edges_;
        if (!(*oe).is_phony()) {
          --command_edges_;
          if (builder_)
            builder_.status_.EdgeRemovedFromPlan(*oe);
        }
      }
    }
  }
  return true;
}

/// Number of edges with commands to run.
func (this *Plan)  command_edge_count() int { return this.command_edges_; }

/// Reset state.  Clears want and ready sets.
func (p *Plan) Reset() {
  command_edges_ = 0;
  wanted_edges_ = 0;
  ready_.clear();
  want_.clear();
}

// After all targets have been added, prepares the ready queue for find work.
func (p *Plan) PrepareQueue() {
  ComputeCriticalPath();
  ScheduleInitialEdges();
}

/// Update the build plan to account for modifications made to the graph
/// by information loaded from a dyndep file.
func (p *Plan)  DyndepsLoaded(scan *DependencyScan, node *Node, ddf *DyndepFile,  err *string) bool {
  // Recompute the dirty state of all our direct and indirect dependents now
  // that our dyndep information has been loaded.
  if (!RefreshDyndepDependents(scan, node, err))
    return false;

  // We loaded dyndep information for those out_edges of the dyndep node that
  // specify the node in a dyndep binding, but they may not be in the plan.
  // Starting with those already in the plan, walk newly-reachable portion
  // of the graph through the dyndep-discovered dependencies.

  // Find edges in the the build plan for which we have new dyndep info.
  std::vector<DyndepFile::const_iterator> dyndep_roots;
  for (DyndepFile::const_iterator oe = ddf.begin(); oe != ddf.end(); ++oe) {
    Edge* edge = oe.first;

    // If the edge outputs are ready we do not need to consider it here.
    if (edge.outputs_ready())
      continue;

    map<Edge*, Want>::iterator want_e = want_.find(edge);

    // If the edge has not been encountered before then nothing already in the
    // plan depends on it so we do not need to consider the edge yet either.
    if (want_e == want_.end())
      continue;

    // This edge is already in the plan so queue it for the walk.
    dyndep_roots.push_back(oe);
  }

  // Walk dyndep-discovered portion of the graph to add it to the build plan.
  std::set<Edge*> dyndep_walk;
  for (std::vector<DyndepFile::const_iterator>::iterator
       oei = dyndep_roots.begin(); oei != dyndep_roots.end(); ++oei) {
    DyndepFile::const_iterator oe = *oei;
    for (vector<Node*>::const_iterator i = oe.second.implicit_inputs_.begin();
         i != oe.second.implicit_inputs_.end(); ++i) {
      if (!AddSubTarget(*i, oe.first.outputs_[0], err, &dyndep_walk) &&
          !err.empty())
        return false;
    }
  }

  // Add out edges from this node that are in the plan (just as
  // Plan::NodeFinished would have without taking the dyndep code path).
  for (vector<Edge*>::const_iterator oe = node.out_edges().begin();
       oe != node.out_edges().end(); ++oe) {
    map<Edge*, Want>::iterator want_e = want_.find(*oe);
    if (want_e == want_.end())
      continue;
    dyndep_walk.insert(want_e.first);
  }

  // See if any encountered edges are now ready.
  for (set<Edge*>::iterator wi = dyndep_walk.begin();
       wi != dyndep_walk.end(); ++wi) {
    map<Edge*, Want>::iterator want_e = want_.find(*wi);
    if (want_e == want_.end())
      continue;
    if (!EdgeMaybeReady(want_e, err))
      return false;
  }

  return true;
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

func (p *Plan)   ComputeCriticalPath(){
  METRIC_RECORD("ComputeCriticalPath");

  // Convenience class to perform a topological sort of all edges
  // reachable from a set of unique targets. Usage is:
  //
  // 1) Create instance.
  //
  // 2) Call VisitTarget() as many times as necessary.
  //    Note that duplicate targets are properly ignored.
  //
  // 3) Call result() to get a sorted list of edges,
  //    where each edge appears _after_ its parents,
  //    i.e. the edges producing its inputs, in the list.
  //
  struct TopoSort {
    void VisitTarget(const Node* target) {
      Edge* producer = target.in_edge();
      if (producer)
        Visit(producer);
    }

    const std::vector<Edge*>& result() const { return sorted_edges_; }

   private:
    // Implementation note:
    //
    // This is the regular depth-first-search algorithm described
    // at https://en.wikipedia.org/wiki/Topological_sorting, except
    // that:
    //
    // - Edges are appended to the end of the list, for performance
    //   reasons. Hence the order used in result().
    //
    // - Since the graph cannot have any cycles, temporary marks
    //   are not necessary, and a simple set is used to record
    //   which edges have already been visited.
    //
    void Visit(Edge* edge) {
      auto insertion = visited_set_.emplace(edge);
      if (!insertion.second)
        return;

      for (const Node* input : edge.inputs_) {
        Edge* producer = input.in_edge();
        if (producer)
          Visit(producer);
      }
      sorted_edges_.push_back(edge);
    }

    std::unordered_set<Edge*> visited_set_;
    std::vector<Edge*> sorted_edges_;
  };

  TopoSort topo_sort;
  for (const Node* target : targets_) {
    topo_sort.VisitTarget(target);
  }

  const auto& sorted_edges = topo_sort.result();

  // First, reset all weights to 1.
  for (Edge* edge : sorted_edges)
    edge.set_critical_path_weight(EdgeWeightHeuristic(edge));

  // Second propagate / increment weights from
  // children to parents. Scan the list
  // in reverse order to do so.
  for (auto reverse_it = sorted_edges.rbegin();
       reverse_it != sorted_edges.rend(); ++reverse_it) {
    Edge* edge = *reverse_it;
    int64_t edge_weight = edge.critical_path_weight();

    for (const Node* input : edge.inputs_) {
      Edge* producer = input.in_edge();
      if (!producer)
        continue;

      int64_t producer_weight = producer.critical_path_weight();
      int64_t candidate_weight = edge_weight + EdgeWeightHeuristic(producer);
      if (candidate_weight > producer_weight)
        producer.set_critical_path_weight(candidate_weight);
    }
  }
}
func (p *Plan)  RefreshDyndepDependents(scan *DependencyScan, node *Node, err *string) bool{
  // Collect the transitive closure of dependents and mark their edges
  // as not yet visited by RecomputeDirty.
  set<Node*> dependents;
  UnmarkDependents(node, &dependents);

  // Update the dirty state of all dependents and check if their edges
  // have become wanted.
  for (set<Node*>::iterator i = dependents.begin();
       i != dependents.end(); ++i) {
    Node* n = *i;

    // Check if this dependent node is now dirty.  Also checks for new cycles.
    std::vector<Node*> validation_nodes;
    if (!scan.RecomputeDirty(n, &validation_nodes, err))
      return false;

    // Add any validation nodes found during RecomputeDirty as new top level
    // targets.
    for (std::vector<Node*>::iterator v = validation_nodes.begin();
         v != validation_nodes.end(); ++v) {
      if (Edge* in_edge = (*v).in_edge()) {
        if (!in_edge.outputs_ready() &&
            !AddTarget(*v, err)) {
          return false;
        }
      }
    }
    if (!n.dirty())
      continue;

    // This edge was encountered before.  However, we may not have wanted to
    // build it if the outputs were not known to be dirty.  With dyndep
    // information an output is now known to be dirty, so we want the edge.
    Edge* edge = n.in_edge();
    assert(edge && !edge.outputs_ready());
    map<Edge*, Want>::iterator want_e = want_.find(edge);
    assert(want_e != want_.end());
    if (want_e.second == kWantNothing) {
      want_e.second = kWantToStart;
      EdgeWanted(edge);
    }
  }
  return true;
}
func (p *Plan)   UnmarkDependents(node  *Node,  dependents map[*Node]bool){
  for (vector<Edge*>::const_iterator oe = node.out_edges().begin();
       oe != node.out_edges().end(); ++oe) {
    Edge* edge = *oe;

    map<Edge*, Want>::iterator want_e = want_.find(edge);
    if (want_e == want_.end())
      continue;

    if (edge.mark_ != Edge::VisitNone) {
      edge.mark_ = Edge::VisitNone;
      for (vector<Node*>::iterator o = edge.outputs_.begin();
           o != edge.outputs_.end(); ++o) {
        if (dependents.insert(*o).second)
          UnmarkDependents(*o, dependents);
      }
    }
  }
}
 func (p *Plan)  AddSubTarget(node  *Node, dependent *Node,  err *string, dyndep_walk  map[*Edge]bool) bool {
  Edge* edge = node.in_edge();
  if (!edge) {
     // Leaf node, this can be either a regular input from the manifest
     // (e.g. a source file), or an implicit input from a depfile or dyndep
     // file. In the first case, a dirty flag means the file is missing,
     // and the build should stop. In the second, do not do anything here
     // since there is no producing edge to add to the plan.
     if (node.dirty() && !node.generated_by_dep_loader()) {
       string referenced;
       if (dependent)
         referenced = ", needed by '" + dependent.path() + "',";
       *err = "'" + node.path() + "'" + referenced +
              " missing and no known rule to make it";
     }
     return false;
  }

  if (edge.outputs_ready())
    return false;  // Don't need to do anything.

  // If an entry in want_ does not already exist for edge, create an entry which
  // maps to kWantNothing, indicating that we do not want to build this entry itself.
  pair<map<Edge*, Want>::iterator, bool> want_ins =
    want_.insert(make_pair(edge, kWantNothing));
  Want& want = want_ins.first.second;

  if (dyndep_walk && want == kWantToFinish)
    return false;  // Don't need to do anything with already-scheduled edge.

  // If we do need to build edge and we haven't already marked it as wanted,
  // mark it now.
  if (node.dirty() && want == kWantNothing) {
    want = kWantToStart;
    EdgeWanted(edge);
  }

  if (dyndep_walk)
    dyndep_walk.insert(edge);

  if (!want_ins.second)
    return true;  // We've already processed the inputs.

  for (vector<Node*>::iterator i = edge.inputs_.begin();
       i != edge.inputs_.end(); ++i) {
    if (!AddSubTarget(*i, node, err, dyndep_walk) && !err.empty())
      return false;
  }

  return true;
 }

// Add edges that kWantToStart into the ready queue
// Must be called after ComputeCriticalPath and before FindWork
func (p *Plan)   ScheduleInitialEdges(){
  // Add ready edges to queue.
  assert(ready_.empty());
  std::set<Pool*> pools;

  for (std::map<Edge*, Plan::Want>::iterator it = want_.begin(),
           end = want_.end(); it != end; ++it) {
    Edge* edge = it.first;
    Plan::Want want = it.second;
    if (want == kWantToStart && edge.AllInputsReady()) {
      Pool* pool = edge.pool();
      if (pool.ShouldDelayEdge()) {
        pool.DelayEdge(edge);
        pools.insert(pool);
      } else {
        ScheduleWork(it);
      }
    }
  }

  // Call RetrieveReadyEdges only once at the end so higher priority
  // edges are retrieved first, not the ones that happen to be first
  // in the want_ map.
  for (std::set<Pool*>::iterator it=pools.begin(),
           end = pools.end(); it != end; ++it) {
    (*it).RetrieveReadyEdges(&ready_);
  }
}

/// Update plan with knowledge that the given node is up to date.
/// If the node is a dyndep binding on any of its dependents, this
/// loads dynamic dependencies from the node's path.
/// Returns 'false' if loading dyndep info fails and 'true' otherwise.
func (p *Plan) NodeFinished( node *Node, err *string) bool {
  // If this node provides dyndep info, load it now.
  if (node.dyndep_pending()) {
    assert(builder_ && "dyndep requires Plan to have a Builder");
    // Load the now-clean dyndep file.  This will also update the
    // build plan and schedule any new work that is ready.
    return builder_.LoadDyndeps(node, err);
  }

  // See if we we want any edges from this node.
  for (vector<Edge*>::const_iterator oe = node.out_edges().begin();
       oe != node.out_edges().end(); ++oe) {
    map<Edge*, Want>::iterator want_e = want_.find(*oe);
    if (want_e == want_.end())
      continue;

    // See if the edge is now ready.
    if (!EdgeMaybeReady(want_e, err))
      return false;
  }
  return true;
}

func (p *Plan)   EdgeWanted(edge *Edge) {
  ++wanted_edges_;
  if (!edge.is_phony()) {
    ++command_edges_;
    if (builder_)
      builder_.status_.EdgeAddedToPlan(edge);
  }
}
func (p *Plan) EdgeMaybeReady(want_e Want , err *string) bool {
  Edge* edge = want_e.first;
  if (edge.AllInputsReady()) {
    if (want_e.second != kWantNothing) {
      ScheduleWork(want_e);
    } else {
      // We do not need to build this edge, but we might need to build one of
      // its dependents.
      if (!EdgeFinished(edge, kEdgeSucceeded, err))
        return false;
    }
  }
  return true;
}

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
  finished_.push(edge);
  return true;
}

func (d *DryRunCommandRunner) WaitForCommand() (Result, bool) {
   if (finished_.empty())
     return false;

   result.status = ExitSuccess;
   result.edge = finished_.front();
   finished_.pop();
   return true;
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
