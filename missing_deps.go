package main

import (
	"fmt"
	"github.com/ahrtr/gocontainer/set"
	"slices"
)

type NodeStoringImplicitDepLoader struct {
	ImplicitDepLoader
	dep_nodes_output_ []*Node
}

func NewNodeStoringImplicitDepLoader(state *State, deps_log *DepsLog, disk_interface DiskInterface,
	depfile_parser_options *DepfileParserOptions, explanations Explanations, dep_nodes_output []*Node) *NodeStoringImplicitDepLoader {
	ret := NodeStoringImplicitDepLoader{}
	ret.ImplicitDepLoader = *NewImplicitDepLoader(state, deps_log, disk_interface, depfile_parser_options, explanations)
	ret.dep_nodes_output_ = dep_nodes_output
	return &ret
}

func (this *NodeStoringImplicitDepLoader) ProcessDepfileDeps(edge *Edge, depfile_ins []string, err *string) bool {
	for _, str := range depfile_ins {
		slash_bits := uint64(0)
		CanonicalizePath(&str, &slash_bits)
		node := this.state_.GetNode(str, slash_bits)
		this.dep_nodes_output_ = append(this.dep_nodes_output_, node)
	}
	return true
}

type MissingDependencyScannerDelegate interface {
	ReleaseMissingDependencyScannerDelegate()
	OnMissingDep(node *Node, path string, generator *Rule)
}

type MissingDependencyPrinter struct {
	MissingDependencyScannerDelegate
}

func (this *MissingDependencyPrinter) OnMissingDep(node *Node, path string, generator *Rule) {
	fmt.Sprintf("Missing dep: " + node.path() + " uses " + path + " (generated by " + generator.name() + ")\n")
}
func (this *MissingDependencyPrinter) OnStats(nodes_processed, nodes_missing_deps,
	missing_dep_path_count, generated_nodes,
	generator_rules int) {
}

type InnerAdjacencyMap map[*Edge]bool
type AdjacencyMap map[*Edge]InnerAdjacencyMap
type MissingDependencyScanner struct {
	delegate_               MissingDependencyScannerDelegate
	deps_log_               *DepsLog
	state_                  *State
	disk_interface_         DiskInterface
	seen_                   map[*Node]bool
	nodes_missing_deps_     map[*Node]bool
	generated_nodes_        map[*Node]bool
	generator_rules_        map[*Rule]bool
	missing_dep_path_count_ int

	adjacency_map_ AdjacencyMap
}

func NewMissingDependencyScanner(delegate MissingDependencyScannerDelegate, deps_log *DepsLog, state *State, disk_interface DiskInterface) *MissingDependencyScanner {
	ret := MissingDependencyScanner{}
	ret.delegate_ = delegate
	ret.deps_log_ = deps_log
	ret.state_ = state
	ret.disk_interface_ = disk_interface
	ret.missing_dep_path_count_ = 0
	return &ret
}
func (this *MissingDependencyScanner) ProcessNode(node *Node) {
	if node == nil {
		return
	}
	edge := node.in_edge()
	if edge == nil {
		return
	}
	_, ok := this.seen_[node]
	if ok {
		return
	}

	for _, in := range edge.inputs_ {
		this.ProcessNode(in)
	}

	deps_type := edge.GetBinding("deps")
	if deps_type != "" {
		deps := this.deps_log_.GetDeps(node)
		if deps != nil {
			this.ProcessNodeDeps(node, deps.nodes)
		}
	} else {
		parser_opts := DepfileParserOptions{}
		depfile_deps := []*Node{}
		dep_loader := NewNodeStoringImplicitDepLoader(this.state_, this.deps_log_, this.disk_interface_,
			&parser_opts, nil,
			depfile_deps)
		err := ""
		dep_loader.LoadDeps(edge, &err)
		if len(depfile_deps) != 0 {
			this.ProcessNodeDeps(node, depfile_deps)
		}
	}
}
func (this *MissingDependencyScanner) PrintStats() {
	fmt.Printf("Processed  %d nodes.\n", len(this.seen_))
	if this.HadMissingDeps() {
		fmt.Printf("Error: There are %d missing dependency paths.\n", this.missing_dep_path_count_)
		fmt.Printf("%d targets had depfile dependencies on "+
			"%d distinct generated inputs "+
			"(from %d  rules) "+
			" without a non-depfile dep path to the generator.\n",
			len(this.nodes_missing_deps_), len(this.generated_nodes_), len(this.generator_rules_))
		fmt.Printf("There might be build flakiness if any of the targets listed " +
			"above are built alone, or not late enough, in a clean output " +
			"directory.\n")
	} else {
		fmt.Printf("No missing dependencies on generated files found.\n")
	}
}

func (this *MissingDependencyScanner) HadMissingDeps() bool {
	return len(this.nodes_missing_deps_) != 0
}

func (this *MissingDependencyScanner) ProcessNodeDeps(node *Node, dep_nodes []*Node) {
	edge := node.in_edge()
	deplog_edges := []*Edge{}
	for i := 0; i < len(dep_nodes); i++ {
		deplog_node := dep_nodes[i]
		// Special exception: A dep on build.ninja can be used to mean "always
		// rebuild this target when the build is reconfigured", but build.ninja is
		// often generated by a configuration tool like cmake or gn. The rest of
		// the build "implicitly" depends on the entire build being reconfigured,
		// so a missing dep path to build.ninja is not an actual missing dependency
		// problem.
		if deplog_node.path() == "build.ninja" {
			return
		}
		deplog_edge := deplog_node.in_edge()
		if deplog_edge != nil && !slices.Contains(deplog_edges, deplog_edge) {
			deplog_edges = append(deplog_edges, deplog_edge)
		}
	}
	missing_deps := []*Edge{}
	for _, de := range deplog_edges {
		if !this.PathExistsBetween(de, edge) {
			missing_deps = append(missing_deps, de)
		}
	}

	if len(missing_deps) != 0 {
		missing_deps_rule_names := set.New() // std::set<std::string>
		for _, ne := range missing_deps {
			for i := 0; i < len(dep_nodes); i++ {
				if dep_nodes[i].in_edge() == ne {
					this.generated_nodes_[dep_nodes[i]] = true
					this.generator_rules_[ne.rule()] = true
					missing_deps_rule_names.Add(ne.rule().name())
					this.delegate_.OnMissingDep(node, dep_nodes[i].path(), (*ne).rule())
				}
			}
		}
		this.missing_dep_path_count_ += missing_deps_rule_names.Size()
		this.nodes_missing_deps_[node] = true
	}
}

func (this *MissingDependencyScanner) PathExistsBetween(from, to *Edge) bool {
	second, ok := this.adjacency_map_[from]
	var inner_it InnerAdjacencyMap = nil
	if ok {
		_, ok1 := second[to]
		if ok1 {
			return true
		}
	} else {
		inner_it = InnerAdjacencyMap{}
		this.adjacency_map_[from] = inner_it
	}
	found := false
	for i := 0; i < len(to.inputs_); i++ {
		e := to.inputs_[i].in_edge()
		if e != nil && (e == from || this.PathExistsBetween(from, e)) {
			found = true
			break
		}
	}
	second[to] = found
	return found
}
