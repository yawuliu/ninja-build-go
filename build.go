package main

import (
	"fmt"
	"github.com/ahrtr/gocontainer/set"
	"github.com/edwingeng/deque"
	"log"
)

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

// / Add a target to our plan (including all its dependencies).
// / Returns false if we don't need to build this target; may
// / fill in |err| with an error message if there's a problem.
func (p *Plan) AddTarget(target *Node, err *string) bool {
	// 实现添加目标到计划的逻辑
	return true
}

// Pop a ready edge off the queue of edges to build.
// Returns NULL if there's no work to do.
func (this *Plan) FindWork() *Edge {
	if this.ready_.IsEmpty() {
		return nil
	}

	work := this.ready_.top()
	this.ready_.pop()
	return work
}

// / Returns true if there's more work to be done.
func (p *Plan) more_to_do() bool {
	return p.wanted_edges_ > 0 && p.command_edges_ > 0
}

// / Dumps the current state of the plan.
func (this *Plan) Dump() {
	fmt.Printf("pending: %d\n", len(this.want_))
	for first, second := range this.want_ {
		if second != kWantNothing {
			fmt.Printf("want ")
		}
		first.Dump("")
	}
	fmt.Printf("ready: %d\n", this.ready_.Size())
}

type EdgeResult int8

const (
	kEdgeFailed    EdgeResult = 0
	kEdgeSucceeded EdgeResult = 1
)

// / Mark an edge as done building (whether it succeeded or failed).
// / If any of the edge's outputs are dyndep bindings of their dependents,
// / this loads dynamic dependencies from the nodes' paths.
// / Returns 'false' if loading dyndep info fails and 'true' otherwise.
func (this *Plan) EdgeFinished(edge *Edge, result EdgeResult, err *string) bool {
	second, ok := this.want_[edge]
	if !ok {
		panic("e != want_.end()")
	}
	directly_wanted := second != kWantNothing

	// See if this job frees up any delayed jobs.
	if directly_wanted {
		edge.pool().EdgeFinished(edge)
	}
	edge.pool().RetrieveReadyEdges(this.ready_)

	// The rest of this function only applies to successful commands.
	if result != kEdgeSucceeded {
		return true
	}

	if directly_wanted {
		this.wanted_edges_--
	}
	delete(this.want_, second)
	edge.outputs_ready_ = true

	// Check off any nodes we were waiting for with this edge.
	for _, o := range edge.outputs_ {
		if !this.NodeFinished(o, err) {
			return false
		}
	}
	return true
}

func MEM_FN() {

}

// / Clean the given node during the build.
// / Return false on error.
func (this *Plan) CleanNode(scan *DependencyScan, node *Node, err *string) bool {
	node.set_dirty(false)

	for _, oe := range node.out_edges() {
		// Don't process edges that we don't actually want.
		want_e, ok := this.want_[oe]
		if !ok || want_e == kWantNothing {
			continue
		}

		// Don't attempt to clean an edge if it failed to load deps.
		if oe.deps_missing_ {
			continue
		}
		// If all non-order-only inputs for this edge are now clean,
		// we might have changed the dirty state of the outputs.
		begin := 0
		end := len(oe.inputs_) - oe.order_only_deps_
		found := false
		for i := begin; i < end; i++ {
			if oe.inputs_[i].dirty() {
				found = true
				break
			}
		}
		if found {
			// Recompute most_recent_input.
			var most_recent_input *Node = nil
			for i := begin; i < end; i++ {
				if most_recent_input == nil || oe.inputs_[i].mtime() > most_recent_input.mtime() {
					most_recent_input = oe.inputs_[i]
				}

				// Now, this edge is dirty if any of the outputs are dirty.
				// If the edge isn't dirty, clean the outputs and mark the edge as not
				// wanted.
				outputs_dirty := false
				if !scan.RecomputeOutputsDirty(oe, most_recent_input, &outputs_dirty, err) {
					return false
				}
				if !outputs_dirty {
					for _, o := range oe.outputs_ {
						if !this.CleanNode(scan, o, err) {
							return false
						}
					}

					this.want_[oe] = kWantNothing
					this.wanted_edges_--
					if !oe.is_phony() {
						this.command_edges_--
						if this.builder_ != nil {
							this.builder_.status_.EdgeRemovedFromPlan(oe)
						}
					}
				}
			}
		}
	}
	return true
}

// / Number of edges with commands to run.
func (this *Plan) command_edge_count() int { return this.command_edges_ }

// / Reset state.  Clears want and ready sets.
func (this *Plan) Reset() {
	this.command_edges_ = 0
	this.wanted_edges_ = 0
	this.ready_.Clear()
	this.want_ = map[*Edge]Want{}
}

// After all targets have been added, prepares the ready queue for find work.
func (this *Plan) PrepareQueue() {
	this.ComputeCriticalPath()
	this.ScheduleInitialEdges()
}

// / Update the build plan to account for modifications made to the graph
// / by information loaded from a dyndep file.
func (this *Plan) DyndepsLoaded(scan *DependencyScan, node *Node, ddf DyndepFile, err *string) bool {
	// Recompute the dirty state of all our direct and indirect dependents now
	// that our dyndep information has been loaded.
	if !this.RefreshDyndepDependents(scan, node, err) {
		return false
	}

	// We loaded dyndep information for those out_edges of the dyndep node that
	// specify the node in a dyndep binding, but they may not be in the plan.
	// Starting with those already in the plan, walk newly-reachable portion
	// of the graph through the dyndep-discovered dependencies.

	// Find edges in the the build plan for which we have new dyndep info.
	dyndep_roots := []*Dyndeps{}
	for first, oe := range ddf {
		edge := first

		// If the edge outputs are ready we do not need to consider it here.
		if edge.outputs_ready() {
			continue
		}

		_, ok := this.want_[edge]

		// If the edge has not been encountered before then nothing already in the
		// plan depends on it so we do not need to consider the edge yet either.
		if !ok {
			continue
		}

		// This edge is already in the plan so queue it for the walk.
		dyndep_roots = append(dyndep_roots, oe)
	}

	// Walk dyndep-discovered portion of the graph to add it to the build plan.
	dyndep_walk := set.New() // std::set<Edge*>
	for _, oei := range dyndep_roots {
		for _, i := range oei.implicit_inputs_ {
			if !this.AddSubTarget(i, oei.outputs_[0], err, &dyndep_walk) && *err != "" {
				return false
			}
		}
	}

	// Add out edges from this node that are in the plan (just as
	// Plan::NodeFinished would have without taking the dyndep code path).
	for _, oe := range node.out_edges() {
		want_e := this.want_.find(*oe)
		if want_e == this.want_.end() {
			continue
		}
		dyndep_walk.Add(want_e.first)
	}

	// See if any encountered edges are now ready.
	dyndep_walk.Iterate(func(wi interface{}) bool {
		want_e, ok := this.want_[wi]
		if !ok {
			return
		}
		if !this.EdgeMaybeReady(want_e, err) {
			return false
		}
		return
	})

	return true
}

// / Enumerate possible steps we want for an edge.
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

func (this *Plan) ComputeCriticalPath() {
	METRIC_RECORD("ComputeCriticalPath")

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

	topo_sort := TopoSort{}
	for _, target := range this.targets_ {
		topo_sort.VisitTarget(target)
	}

	sorted_edges := topo_sort.result()

	// First, reset all weights to 1.
	for _, edge := range sorted_edges {
		edge.set_critical_path_weight(this.EdgeWeightHeuristic(edge))
	}

	// Second propagate / increment weights from
	// children to parents. Scan the list
	// in reverse order to do so.
	for reverse_it := len(sorted_edges) - 1; reverse_it >= 0; reverse_it-- {
		edge := sorted_edges[reverse_it]
		edge_weight := edge.critical_path_weight()

		for _, input := range edge.inputs_ {
			producer := input.in_edge()
			if producer == nil {
				continue
			}
			producer_weight := producer.critical_path_weight()
			candidate_weight := edge_weight + this.EdgeWeightHeuristic(producer)
			if candidate_weight > producer_weight {
				producer.set_critical_path_weight(candidate_weight)
			}
		}
	}
}

type TopoSort struct {
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

	visited_set_  []*Edge //std::unordered_set<
	sorted_edges_ []*Edge
}

func (this *TopoSort) VisitTarget(target *Node) {
	producer := target.in_edge()
	if producer != nil {
		this.Visit(producer)
	}
}

func (this *TopoSort) result() []*Edge { return this.sorted_edges_ }

func (this *TopoSort) Visit(edge *Edge) {
	insertion := this.visited_set_.emplace(edge)
	if !insertion.second {
		return
	}
	for _, input := range edge.inputs_ {
		producer := input.in_edge()
		if producer != nil {
			this.Visit(producer)
		}
	}
	this.sorted_edges_ = append(this.sorted_edges_, edge)
}

func (this *Plan) RefreshDyndepDependents(scan *DependencyScan, node *Node, err *string) bool {
	// Collect the transitive closure of dependents and mark their edges
	// as not yet visited by RecomputeDirty.
	dependents := set.New() //set<Node*>
	this.UnmarkDependents(node, &dependents)

	// Update the dirty state of all dependents and check if their edges
	// have become wanted.
	for _, i := range dependents {
		n := i

		// Check if this dependent node is now dirty.  Also checks for new cycles.
		validation_nodes := []*Node{}
		if !scan.RecomputeDirty(n, validation_nodes, err) {
			return false
		}

		// Add any validation nodes found during RecomputeDirty as new top level
		// targets.
		for _, v := range validation_nodes {
			in_edge := v.in_edge()
			if in_edge != nil {
				if !in_edge.outputs_ready() && !this.AddTarget(v, err) {
					return false
				}
			}
		}
		if !n.dirty() {
			continue
		}
		// This edge was encountered before.  However, we may not have wanted to
		// build it if the outputs were not known to be dirty.  With dyndep
		// information an output is now known to be dirty, so we want the edge.
		edge := n.in_edge()
		if edge && !edge.outputs_ready() {
			panic("edge && !edge.outputs_ready()")
		}
		want_e := this.want_.find(edge)
		if want_e != this.want_.end() {
			panic("want_e != this.want_.end()")
		}
		if want_e.second == kWantNothing {
			want_e.second = kWantToStart
			this.EdgeWanted(edge)
		}
	}
	return true
}
func (this *Plan) UnmarkDependents(node *Node, dependents set.Interface) { //map[*Node]bool
	for _, oe := range node.out_edges() {
		edge := oe

		_, ok := this.want_[edge]
		if !ok {
			continue
		}

		if edge.mark_ != VisitNone {
			edge.mark_ = VisitNone
			for _, o := range edge.outputs_ {
				if dependents.Add(o) {
					this.UnmarkDependents(o, dependents)
				}
			}
		}
	}
}
func (this *Plan) AddSubTarget(node *Node, dependent *Node, err *string, dyndep_walk map[*Edge]bool) bool {
	edge := node.in_edge()
	if edge == nil {
		// Leaf node, this can be either a regular input from the manifest
		// (e.g. a source file), or an implicit input from a depfile or dyndep
		// file. In the first case, a dirty flag means the file is missing,
		// and the build should stop. In the second, do not do anything here
		// since there is no producing edge to add to the plan.
		if node.dirty() && !node.generated_by_dep_loader() {
			referenced := ""
			if dependent != nil {
				referenced = ", needed by '" + dependent.path() + "',"
			}
			*err = "'" + node.path() + "'" + referenced +
				" missing and no known rule to make it"
		}
		return false
	}

	if edge.outputs_ready() {
		return false // Don't need to do anything.
	}
	// If an entry in want_ does not already exist for edge, create an entry which
	// maps to kWantNothing, indicating that we do not want to build this entry itself.
	this.want_[edge] = kWantNothing
	want_ins := kWantNothing
	want := kWantNothing

	if dyndep_walk != nil && want == kWantToFinish {
		return false // Don't need to do anything with already-scheduled edge.
	}

	// If we do need to build edge and we haven't already marked it as wanted,
	// mark it now.
	if node.dirty() && want == kWantNothing {
		want = kWantToStart
		this.EdgeWanted(edge)
	}

	if dyndep_walk != nil {
		dyndep_walk[edge] = true
	}

	if !want_ins.second {
		return true // We've already processed the inputs.
	}

	for _, i := range edge.inputs_ {
		if !this.AddSubTarget(*i, node, err, dyndep_walk) && !err.empty() {
			return false
		}
	}

	return true
}

// Add edges that kWantToStart into the ready queue
// Must be called after ComputeCriticalPath and before FindWork
func (this *Plan) ScheduleInitialEdges() {
	// Add ready edges to queue.
	if this.ready_.IsEmpty() {
		panic("ready_.empty()")
	}
	pools := set.New() // std::set<Pool*>

	for first, second := range this.want_ {
		edge := first
		want := second
		if want == kWantToStart && edge.AllInputsReady() {
			pool := edge.pool()
			if pool.ShouldDelayEdge() {
				pool.DelayEdge(edge)
				pools.Add(pool)
			} else {
				this.ScheduleWork(it)
			}
		}
	}

	// Call RetrieveReadyEdges only once at the end so higher priority
	// edges are retrieved first, not the ones that happen to be first
	// in the want_ map.
	for _, it := range pools {
		it.RetrieveReadyEdges(&this.ready_)
	}
}

// / Update plan with knowledge that the given node is up to date.
// / If the node is a dyndep binding on any of its dependents, this
// / loads dynamic dependencies from the node's path.
// / Returns 'false' if loading dyndep info fails and 'true' otherwise.
func (this *Plan) NodeFinished(node *Node, err *string) bool {
	// If this node provides dyndep info, load it now.
	if node.dyndep_pending() {
		if this.builder_ != nil {
			panic("dyndep requires Plan to have a Builder")
		}
		// Load the now-clean dyndep file.  This will also update the
		// build plan and schedule any new work that is ready.
		return this.builder_.LoadDyndeps(node, err)
	}

	// See if we we want any edges from this node.
	for _, oe := range node.out_edges() {
		want_e := this.want_.find(*oe)
		if want_e == this.want_.end() {
			continue
		}

		// See if the edge is now ready.
		if !this.EdgeMaybeReady(want_e, err) {
			return false
		}
	}
	return true
}

func (this *Plan) EdgeWanted(edge *Edge) {
	this.wanted_edges_++
	if !edge.is_phony() {
		this.command_edges_++
		if this.builder_ != nil {
			this.builder_.status_.EdgeAddedToPlan(edge)
		}
	}
}
func (this *Plan) EdgeMaybeReady(want_e Want, err *string) bool {
	edge := want_e.first
	if edge.AllInputsReady() {
		if want_e.second != kWantNothing {
			this.ScheduleWork(want_e)
		} else {
			// We do not need to build this edge, but we might need to build one of
			// its dependents.
			if !this.EdgeFinished(edge, kEdgeSucceeded, err) {
				return false
			}
		}
	}
	return true
}

// / Submits a ready edge as a candidate for execution.
// / The edge may be delayed from running, for example if it's a member of a
// / currently-full pool.
func (p *Plan) ScheduleWork(want_e Want) {}

// / The result of waiting for a command.
type Result struct {
	edge   *Edge
	status ExitStatus
	output string
}

func NewResult() *Result {
	ret := Result{}
	ret.edge = nil
	return &ret
}
func (this *Result) success() bool { return this.status == ExitSuccess }

type CommandRunner interface {
	ReleaseCommandRunner()
	StartCommand(edge *Edge) bool
	WaitForCommand() (Result, bool)
	GetActiveEdges() []*Edge
	CanRunMore() int64
	Abort()
}

func CommandRunnerfactory(config *BuildConfig) CommandRunner {

}

type DryRunCommandRunner struct {
	CommandRunner
	finished_ deque.Deque // <Edge*>
}

func (this *DryRunCommandRunner) StartCommand(edge *Edge) bool {
	this.finished_.PushBack(edge)
	return true
}

func (this *DryRunCommandRunner) WaitForCommand(result *Result) bool {
	if this.finished_.Empty() {
		return false
	}

	result.status = ExitSuccess
	result.edge = this.finished_.Front()
	this.finished_.PopFront()
	return true
}

func (d *DryRunCommandRunner) GetActiveEdges() []*Edge {
	return []*Edge{}
}

func (d *DryRunCommandRunner) Abort() {}

func NewBuilder(state *State, config *BuildConfig, build_log *BuildLog,
	deps_log *DepsLog, disk_interface DiskInterface, status Status,
	start_time_millis int64) *Builder {
	ret := Builder{}
	ret.state_ = state
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
	ret.scan_ = NewDependencyScan(state, build_log, deps_log, disk_interface, ret.config_.depfile_parser_options, ret.explanations_)
	ret.lock_file_path_ = ".ninja_lock"
	build_dir := ret.state_.bindings_.LookupVariable("builddir")
	if build_dir != "" {
		ret.lock_file_path_ = build_dir + "/" + ret.lock_file_path_
	}
	ret.status_.SetExplanations(ret.explanations_)
	return &ret
}

func (this *Builder) RealeaseBuilder() {
	this.Cleanup()
	this.status_.SetExplanations(nil)
}

// / Clean up after interrupted commands by deleting output files.
func (this *Builder) Cleanup() {
	if this.command_runner_ != nil {
		active_edges := this.command_runner_.GetActiveEdges()
		this.command_runner_.Abort()

		for _, e := range active_edges {
			depfile := e.GetUnescapedDepfile()
			for _, o := range e.outputs_ {
				// Only delete this output if it was actually modified.  This is
				// important for things like the generator where we don't want to
				// delete the manifest file if we can avoid it.  But if the rule
				// uses a depfile, always delete.  (Consider the case where we
				// need to rebuild an output because of a modified header file
				// mentioned in a depfile, and the command touches its depfile
				// but is interrupted before it touches its output file.)
				err := ""
				new_mtime := this.disk_interface_.Stat(o.path(), &err)
				if new_mtime == -1 { // Log and ignore Stat() errors.
					this.status_.Error("%s", err)
				}
				if !depfile.empty() || o.mtime() != new_mtime {
					this.disk_interface_.RemoveFile(o.path())
				}
			}
			if depfile != "" {
				this.disk_interface_.RemoveFile(depfile)
			}
		}
	}

	err := ""
	if this.disk_interface_.Stat(this.lock_file_path_, &err) > 0 {
		this.disk_interface_.RemoveFile(this.lock_file_path_)
	}
}

func (this *Builder) AddTarget(name string, err *string) *Node {
	node := this.state_.LookupNode(name)
	if node == nil {
		*err = "unknown target: '" + name + "'"
		return nil
	}
	if !this.AddTarget2(node, err) {
		return nil
	}
	return node
}

// / Add a target to the build, scanning dependencies.
// / @return false on error.
func (this *Builder) AddTarget2(target *Node, err *string) bool {
	validation_nodes := []*Node{}
	if !this.scan_.RecomputeDirty(target, validation_nodes, err) {
		return false
	}

	in_edge := target.in_edge()
	if in_edge == nil || !in_edge.outputs_ready() {
		if !this.plan_.AddTarget(target, err) {
			return false
		}
	}

	// Also add any validation nodes found during RecomputeDirty as top level
	// targets.
	for _, n := range validation_nodes {
		validation_in_edge := n.in_edge()
		if validation_in_edge != nil {
			if !validation_in_edge.outputs_ready() && !this.plan_.AddTarget(*n, err) {
				return false
			}
		}
	}

	return true
}

// / Returns true if the build targets are already up to date.
func (this *Builder) AlreadyUpToDate() bool {
	return !this.plan_.more_to_do()
}

// / Run the build.  Returns false on error.
// / It is an error to call this function when AlreadyUpToDate() is true.
func (this *Builder) Build(err *string) bool {
	if !this.AlreadyUpToDate() {
		panic("!AlreadyUpToDate() ")
	}
	this.plan_.PrepareQueue()

	pending_commands := 0
	failures_allowed := this.config_.failures_allowed

	// Set up the command runner if we haven't done so already.
	if this.command_runner_ == nil {
		if this.config_.dry_run {
			this.command_runner_.reset(NewDryRunCommandRunner)
		} else {
			this.command_runner_.reset(CommandRunnerfactory(this.config_))
		}
	}

	// We are about to start the build process.
	this.status_.BuildStarted()

	// This main loop runs the entire build process.
	// It is structured like this:
	// First, we attempt to start as many commands as allowed by the
	// command runner.
	// Second, we attempt to wait for / reap the next finished command.
	for this.plan_.more_to_do() {
		// See if we can start any more commands.
		if failures_allowed != 0 {
			capacity := this.command_runner_.CanRunMore()
			for capacity > 0 {
				edge := this.plan_.FindWork()
				if !edge {
					break
				}

				if edge.GetBindingBool("generator") {
					this.scan_.build_log().Close()
				}

				if !this.StartEdge(edge, err) {
					this.Cleanup()
					this.status_.BuildFinished()
					return false
				}

				if edge.is_phony() {
					if !this.plan_.EdgeFinished(edge, kEdgeSucceeded, err) {
						this.Cleanup()
						this.status_.BuildFinished()
						return false
					}
				} else {
					pending_commands++

					capacity--

					// Re-evaluate capacity.
					current_capacity := this.command_runner_.CanRunMore()
					if current_capacity < capacity {
						capacity = current_capacity
					}
				}
			}

			// We are finished with all work items and have no pending
			// commands. Therefore, break out of the main loop.
			if pending_commands == 0 && !this.plan_.more_to_do() {
				break
			}
		}

		// See if we can reap any finished commands.
		if pending_commands != 0 {
			result := Result{}
			if !this.command_runner_.WaitForCommand(&result) ||
				result.status == ExitInterrupted {
				this.Cleanup()
				this.status_.BuildFinished()
				*err = "interrupted by user"
				return false
			}

			pending_commands--
			if !this.FinishCommand(&result, err) {
				this.Cleanup()
				this.status_.BuildFinished()
				return false
			}

			if !result.success() {
				if failures_allowed != 0 {
					failures_allowed--
				}
			}

			// We made some progress; start the main loop over.
			continue
		}

		// If we get here, we cannot make any more progress.
		this.status_.BuildFinished()
		if failures_allowed == 0 {
			if this.config_.failures_allowed > 1 {
				*err = "subcommands failed"
			} else {
				*err = "subcommand failed"
			}
		} else if failures_allowed < this.config_.failures_allowed {
			*err = "cannot make progress due to previous errors"
		} else {
			*err = "stuck [this is a bug]"
		}

		return false
	}

	this.status_.BuildFinished()
	return true
}

func (this *Builder) StartEdge(edge *Edge, err *string) bool {
	METRIC_RECORD("StartEdge")
	if edge.is_phony() {
		return true
	}

	start_time_millis := GetTimeMillis() - this.start_time_millis_
	this.running_edges_[edge] = start_time_millis

	this.status_.BuildEdgeStarted(edge, start_time_millis)

	var build_start TimeStamp = -1
	if this.config_.dry_run {
		build_start = 0
	}
	// Create directories necessary for outputs and remember the current
	// filesystem mtime to record later
	// XXX: this will block; do we care?
	for _, o := range edge.outputs_ {
		ok, _ := this.disk_interface_.MakeDirs(o.path())
		if !ok {
			return false
		}
		if build_start == -1 {
			this.disk_interface_.WriteFile(this.lock_file_path_, "")
			build_start = this.disk_interface_.Stat(this.lock_file_path_, err)
			if build_start == -1 {
				build_start = 0
			}
		}
	}

	edge.command_start_time_ = build_start

	// Create depfile directory if needed.
	// XXX: this may also block; do we care?
	depfile := edge.GetUnescapedDepfile()
	ok, _ := this.disk_interface_.MakeDirs(depfile)
	if depfile != "" && !ok {
		return false
	}

	// Create response file, if needed
	// XXX: this may also block; do we care?
	rspfile := edge.GetUnescapedRspfile()
	if rspfile != "" {
		content := edge.GetBinding("rspfile_content")
		if !this.disk_interface_.WriteFile(rspfile, content) {
			return false
		}
	}

	// start command computing and run it
	if !this.command_runner_.StartCommand(edge) {
		err.assign("command '" + edge.EvaluateCommand(false) + "' failed.")
		return false
	}

	return true
}

// / Update status ninja logs following a command termination.
// / @return false if the build can not proceed further due to a fatal error.
func (this *Builder) FinishCommand(result *Result, err *string) bool {
	METRIC_RECORD("FinishCommand")

	edge := result.edge

	// First try to extract dependencies from the result, if any.
	// This must happen first as it filters the command output (we want
	// to filter /showIncludes output, even on compile failure) and
	// extraction itself can fail, which makes the command fail from a
	// build perspective.
	deps_nodes := []*Node{}
	deps_type := edge.GetBinding("deps")
	deps_prefix := edge.GetBinding("msvc_deps_prefix")
	if deps_type != "" {
		extract_err := ""
		if !this.ExtractDeps(result, deps_type, deps_prefix, &deps_nodes,
			&extract_err) &&
			result.success() {
			if !result.output.empty() {
				result.output.append("\n")
			}
			result.output.append(extract_err)
			result.status = ExitFailure
		}
	}

	start_time_millis := int64(0)
	end_time_millis := int64(0)
	it := this.running_edges_.find(edge)
	start_time_millis = it.second
	end_time_millis = GetTimeMillis() - this.start_time_millis_
	this.running_edges_.erase(it)

	this.status_.BuildEdgeFinished(edge, start_time_millis, end_time_millis,
		result.success(), result.output)

	// The rest of this function only applies to successful commands.
	if !result.success() {
		return this.plan_.EdgeFinished(edge, kEdgeFailed, err)
	}

	// Restat the edge outputs
	var record_mtime TimeStamp = 0
	if !this.config_.dry_run {
		restat := edge.GetBindingBool("restat")
		generator := edge.GetBindingBool("generator")
		node_cleaned := false
		record_mtime = edge.command_start_time_

		// restat and generator rules must restat the outputs after the build
		// has finished. if record_mtime == 0, then there was an error while
		// attempting to touch/stat the temp file when the edge started and
		// we should fall back to recording the outputs' current mtime in the
		// log.
		if record_mtime == 0 || restat || generator {
			for _, o := range edge.outputs_ {
				var new_mtime TimeStamp = this.disk_interface_.Stat(o.path(), err)
				if new_mtime == -1 {
					return false
				}
				if new_mtime > record_mtime {
					record_mtime = new_mtime
				}
				if (*o).mtime() == new_mtime && restat {
					// The rule command did not change the output.  Propagate the clean
					// state through the build graph.
					// Note that this also applies to nonexistent outputs (mtime == 0).
					if !this.plan_.CleanNode(&this.scan_, *o, err) {
						return false
					}
					node_cleaned = true
				}
			}
		}
		if node_cleaned {
			record_mtime = edge.command_start_time_
		}
	}

	if !this.plan_.EdgeFinished(edge, kEdgeSucceeded, err) {
		return false
	}

	// Delete any left over response file.
	rspfile := edge.GetUnescapedRspfile()
	if rspfile != "" && !g_keep_rsp {
		this.disk_interface_.RemoveFile(rspfile)
	}

	if this.scan_.build_log() != nil {
		if !this.scan_.build_log().RecordCommand(edge, int(start_time_millis),
			int(end_time_millis), record_mtime) {
			*err = string("Error writing to build log: ") + strerror(errno)
			return false
		}
	}

	if deps_type != "" && !this.config_.dry_run {
		if len(edge.outputs_) != 0 {
			panic("should have been rejected by parser")
		}
		for _, o := range edge.outputs_ {
			var deps_mtime TimeStamp = this.disk_interface_.Stat(o.path(), err)
			if deps_mtime == -1 {
				return false
			}
			if !this.scan_.deps_log().RecordDeps(o, deps_mtime, deps_nodes) {
				*err = "Error writing to deps log: " + strerror(errno)
				return false
			}
		}
	}
	return true
}

// / Used for tests.
func (this *Builder) SetBuildLog(log *BuildLog) {
	this.scan_.set_build_log(log)
}

func (this *Builder) ExtractDeps(result *Result, deps_type string, deps_prefix string, deps_nodes []*Node, err *string) bool {
	if deps_type == "msvc" {
		parser := CLParser{}
		output := ""
		if !parser.Parse(&result.output, deps_prefix, &output, err) {
			return false
		}
		result.output = output
		for _, i := range parser.includes_ {
			// ~0 is assuming that with MSVC-parsed headers, it's ok to always make
			// all backslashes (as some of the slashes will certainly be backslashes
			// anyway). This could be fixed if necessary with some additional
			// complexity in IncludesNormalize::Relativize.
			deps_nodes = append(deps_nodes, this.state_.GetNode(*i, ~0))
		}
	} else if deps_type == "gcc" {
		depfile := result.edge.GetUnescapedDepfile()
		if depfile == "" {
			*err = string("edge with deps=gcc but no depfile makes no sense")
			return false
		}

		// Read depfile content.  Treat a missing depfile as empty.
		content := ""
		switch this.disk_interface_.ReadFile(depfile, &content, err) {
		case NotFound:
			*err = ""
		case OtherError:
			return false
		case Okay:
		}
		if content == "" {
			return true
		}

		deps := NewDepfileParser(this.config_.depfile_parser_options)
		if !deps.Parse(&content, err) {
			return false
		}

		// XXX check depfile matches expected output.
		//deps_nodes.reserve(deps.ins_.size());
		for _, i := range deps.ins_ {
			var slash_bits uint64 = 0
			CanonicalizePath(i, &slash_bits)
			deps_nodes = append(deps_nodes, this.state_.GetNode(*i, slash_bits))
		}

		if !g_keep_depfile {
			if this.disk_interface_.RemoveFile(depfile) < 0 {
				*err = string("deleting depfile: ") + strerror(errno) + string("\n")
				return false
			}
		}
	} else {
		log.Fatalf("unknown deps type '%s'", deps_type)
	}

	return true
}

// / Load the dyndep information provided by the given node.
func (this *Builder) LoadDyndeps(node *Node, err *string) bool {
	// Load the dyndep information provided by this node.
	ddf := DyndepFile{}
	if !this.scan_.LoadDyndeps1(node, ddf, err) {
		return false
	}

	// Update the build plan to account for dyndep modifications to the graph.
	if !this.plan_.DyndepsLoaded(&this.scan_, node, ddf, err) {
		return false
	}

	return true
}
