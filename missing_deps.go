package main

type MissingDependencyScannerDelegate interface {
	ReleaseMissingDependencyScannerDelegate()
	OnMissingDep(node *Node, path string, generator *Rule)
}

type MissingDependencyPrinter struct {
	MissingDependencyScannerDelegate
}

func (this *MissingDependencyPrinter) OnMissingDep(node *Node, path string, generator *Rule) {}
func (this *MissingDependencyPrinter) OnStats(nodes_processed, nodes_missing_deps,
	missing_dep_path_count, generated_nodes,
	generator_rules int) {
}

type InnerAdjacencyMap map[*Edge]bool
type AdjacencyMap map[*Edge]InnerAdjacencyMap
type MissingDependencyScanner struct {
	delegate_               *MissingDependencyScannerDelegate
	deps_log_               *DepsLog
	state_                  *State
	disk_interface_         *DiskInterface
	seen_                   map[*Node]bool
	nodes_missing_deps_     map[*Node]bool
	generated_nodes_        map[*Node]bool
	generator_rules_        map[*Node]bool
	missing_dep_path_count_ int

	adjacency_map_ AdjacencyMap
}

func NewMissingDependencyScanner(delegate MissingDependencyScannerDelegate, deps_log *DepsLog, state *State, disk_interface DiskInterface) *MissingDependencyScanner {
	ret := MissingDependencyScanner{}

	return &ret
}
func (this *MissingDependencyScanner) ProcessNode(node *Node) {}
func (this *MissingDependencyScanner) PrintStats()            {}

func (this *MissingDependencyScanner) HadMissingDeps() bool {
	return len(this.nodes_missing_deps_) != 0
}

func (this *MissingDependencyScanner) ProcessNodeDeps(node *Node, dep_nodes **Node, dep_nodes_count int) {
}

func (this *MissingDependencyScanner) PathExistsBetween(from, to *Edge) bool {}
