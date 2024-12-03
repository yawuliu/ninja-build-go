package ninja_go

import (
	"fmt"
	"github.com/ahrtr/gocontainer/queue/priorityqueue"
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
	ret := &Plan{builder_: builder, want_: make(map[*Edge]Want),
		ready_: priorityqueue.New().WithComparator(&EdgeCmp{})}
	return ret
}

// / Add a target to our plan (including all its dependencies).
// / Returns false if we don't need to build this target; may
// / fill in |err| with an error message if there's a problem.
func (this *Plan) AddTarget(target *Node, err *string) bool {
	this.targets_ = append(this.targets_, target)
	return this.AddSubTarget(target, nil, err, nil)
}

func (this *Plan) AddSubTarget(node *Node, dependent *Node, err *string, dyndep_walk map[*Edge]bool) bool {
	edge := node.in_edge()
	if edge == nil {
		if node.dirty() && !node.generated_by_dep_loader() {
			referenced := ""
			if dependent != nil {
				referenced = ", needed by '" + dependent.path_ + "',"
			}
			*err = fmt.Sprintf("'%s'%s missing and no known rule to make it", node.path_, referenced)
		}
		return false
	}

	if edge.outputs_ready_ {
		return false // 不需要做任何事情
	}

	// 如果 want_ 中没有 edge 的条目，则创建一个条目
	if _, exists := this.want_[edge]; !exists {
		this.want_[edge] = kWantNothing
	}
	want := this.want_[edge]

	if dyndep_walk != nil && want == kWantToFinish {
		return false // 已经计划过的边不需要再次处理
	}

	if node.dirty() && want == kWantNothing {
		want = kWantToStart
		this.want_[edge] = want
		this.EdgeWanted(edge)
	}

	if dyndep_walk != nil {
		dyndep_walk[edge] = true
	}

	for _, input := range edge.inputs_ {
		if !this.AddSubTarget(input, node, err, dyndep_walk) && *err != "" {
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

// Pop a ready edge off the queue of edges to build.
// Returns NULL if there's no work to do.
func (this *Plan) FindWork() *Edge {
	if this.ready_.IsEmpty() {
		return nil
	}

	work := this.ready_.Poll()
	return work.(*Edge)
}

// / Returns true if there's more work to be done.
func (this *Plan) more_to_do() bool {
	return this.wanted_edges_ > 0 && this.command_edges_ > 0
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
	delete(this.want_, edge)
	edge.outputs_ready_ = true

	// Check off any nodes we were waiting for with this edge.
	for _, o := range edge.outputs_ {
		if !this.NodeFinished(o, err) {
			return false
		}
	}
	return true
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
func EdgeWeightHeuristic(edge *Edge) int64 {
	if edge.is_phony() {
		return 0
	} else {
		return 1
	}

}
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

	topo_sort := TopoSort{visited_set_: make(map[*Edge]bool)}
	for _, target := range this.targets_ {
		topo_sort.VisitTarget(target)
	}

	sorted_edges := topo_sort.result()

	// First, reset all weights to 1.
	for _, edge := range sorted_edges {
		edge.set_critical_path_weight(EdgeWeightHeuristic(edge))
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
			candidate_weight := edge_weight + EdgeWeightHeuristic(producer)
			if candidate_weight > producer_weight {
				producer.set_critical_path_weight(candidate_weight)
			}
		}
	}
}

// Add edges that kWantToStart into the ready queue
// Must be called after ComputeCriticalPath and before FindWork
func (this *Plan) ScheduleInitialEdges() {
	// Add ready edges to queue.
	if !this.ready_.IsEmpty() {
		panic("ready_.empty()")
	}
	pools := map[*Pool]bool{} // std::set<*>

	for first, second := range this.want_ {
		edge := first
		want := second
		if want == kWantToStart && edge.AllInputsReady() {
			pool := edge.pool()
			if pool.ShouldDelayEdge() {
				pool.DelayEdge(edge)
				pools[pool] = true
			} else {
				this.ScheduleWork(this.want_, edge)
			}
		}
	}

	// Call RetrieveReadyEdges only once at the end so higher priority
	// edges are retrieved first, not the ones that happen to be first
	// in the want_ map.
	for it, _ := range pools {
		it.RetrieveReadyEdges(this.ready_)
	}
}

// / Update plan with knowledge that the given node is up to date.
// / If the node is a dyndep binding on any of its dependents, this
// / loads dynamic dependencies from the node's path.
// / Returns 'false' if loading dyndep info fails and 'true' otherwise.
func (this *Plan) NodeFinished(node *Node, err *string) bool {
	// If this node provides dyndep info, load it now.
	if node.dyndep_pending() {
		if this.builder_ == nil {
			panic("dyndep requires Plan to have a Builder")
		}
		// Load the now-clean dyndep file.  This will also update the
		// build plan and schedule any new work that is ready.
		return this.builder_.LoadDyndeps(node, err)
	}

	// See if we we want any edges from this node.
	for _, oe := range node.out_edges() {
		_, ok := this.want_[oe]
		if !ok {
			continue
		}

		// See if the edge is now ready.
		if !this.EdgeMaybeReady(this.want_, oe, err) {
			return false
		}
	}
	return true
}

func (this *Plan) EdgeMaybeReady(want_e map[*Edge]Want, want_e_first *Edge, err *string) bool {
	edge := want_e_first
	want_e_second := want_e[want_e_first]
	if edge.AllInputsReady() {
		if want_e_second != kWantNothing {
			this.ScheduleWork(want_e, edge)
		} else {
			// 我们不需要构建这个边，但可能需要构建它的依赖之一
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
func (this *Plan) ScheduleWork(want_e map[*Edge]Want, want_e_first *Edge) {
	want_e_second := want_e[want_e_first]
	if want_e_second == kWantToFinish {
		// 这条边已经被调度过了。如果一个边和它的依赖共享一个顺序输入，
		// 或者一个节点重复了一个输出边（见 https://github.com/ninja-build/ninja/pull/519），
		// 我们可能会再次来到这里。避免再次调度工作。
		return
	}
	if want_e_second != kWantToStart {
		panic("unexpected want state")
	}
	want_e[want_e_first] = kWantToFinish

	edge := want_e_first
	pool := edge.pool()
	if pool.ShouldDelayEdge() {
		pool.DelayEdge(edge)
		pool.RetrieveReadyEdges(this.ready_)
	} else {
		pool.EdgeScheduled(edge)
		this.ready_.Add(edge)
	}
}

// / Update the build plan to account for modifications made to the graph
// / by information loaded from a dyndep file.
func (this *Plan) DyndepsLoaded(scan *DependencyScan, node *Node, ddf DyndepFile, err *string) bool {
	// 重新计算所有直接和间接依赖项的脏状态
	if !this.RefreshDyndepDependents(scan, node, err) {
		return false
	}

	// 查找构建计划中具有新动态依赖信息的边
	dyndepRoots := make([]*Dyndeps, 0)
	for oe, info := range ddf {
		if oe.outputs_ready_ {
			continue
		}

		if _, exists := this.want_[oe]; !exists {
			continue
		}

		dyndepRoots = append(dyndepRoots, info)
	}

	// 遍历动态依赖发现的图的部分，将其添加到构建计划中
	dyndepWalk := map[*Edge]bool{}
	for _, oeInfo := range dyndepRoots {
		for _, input := range oeInfo.implicit_inputs_ {
			if !this.AddSubTarget(input, node, err, dyndepWalk) && *err != "" {
				return false
			}
		}
	}

	// 添加此节点的输出边到计划中
	for _, oe := range node.out_edges_ {
		if _, exists := this.want_[oe]; !exists {
			continue
		}
		dyndepWalk[oe] = true
	}

	// 检查遇到的边是否现在已就绪
	for we, _ := range dyndepWalk {
		if _, exists := this.want_[we]; !exists {
			continue
		}
		if !this.EdgeMaybeReady(this.want_, we, err) {
			return false
		}
	}

	return true
}

func (this *Plan) RefreshDyndepDependents(scan *DependencyScan, node *Node, err *string) bool {
	// Collect the transitive closure of dependents and mark their edges
	// as not yet visited by RecomputeDirty.
	dependents := map[*Node]bool{} //set<Node*>
	this.UnmarkDependents(node, dependents)

	// Update the dirty state of all dependents and check if their edges
	// have become wanted.
	for n, _ := range dependents {
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
		if edge == nil || !edge.outputs_ready() {
			panic("edge && !edge.outputs_ready()")
		}
		want_e_second, ok := this.want_[edge]
		if !ok {
			panic("want_e != this.want_.end()")
		}
		if want_e_second == kWantNothing {
			this.want_[edge] = kWantToStart
			this.EdgeWanted(edge)
		}
	}
	return true
}

func (this *Plan) UnmarkDependents(node *Node, dependents map[*Node]bool) { //map[*Node]bool
	for _, oe := range node.out_edges() {
		edge := oe

		_, ok := this.want_[edge]
		if !ok {
			continue
		}

		if edge.mark_ != VisitNone {
			edge.mark_ = VisitNone
			for _, o := range edge.outputs_ {
				_, ok := dependents[o]
				if !ok {
					this.UnmarkDependents(o, dependents)
				}
			}
		}
	}
}
