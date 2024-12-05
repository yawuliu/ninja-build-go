package main

import "fmt"

type Cleaner struct {
	state_               *State
	config_              *BuildConfig
	dyndep_loader_       *DyndepLoader
	removed_             map[string]bool
	cleaned_             map[*Node]bool
	cleaned_files_count_ int
	disk_interface_      DiskInterface
	status_              int
}

func NewCleaner(state *State, config *BuildConfig, disk_interface DiskInterface) *Cleaner {
	ret := Cleaner{}
	ret.state_ = state
	ret.config_ = config
	ret.removed_ = make(map[string]bool)
	ret.cleaned_ = make(map[*Node]bool)
	ret.dyndep_loader_ = NewDyndepLoader(state, disk_interface, nil)
	ret.cleaned_files_count_ = 0
	ret.disk_interface_ = disk_interface
	ret.status_ = 0
	return &ret
}

func (this *Cleaner) RemoveFile(path string) int {
	return this.disk_interface_.RemoveFile(path)
}

func (this *Cleaner) FileExists(path string) bool {
	_, notExist, err := this.disk_interface_.Stat(path)
	if err != nil {
		Error("%s", err.Error())
		return false
	}
	return !notExist // Treat Stat() errors as "file does not exist".
}

func (this *Cleaner) Report(path string) {
	this.cleaned_files_count_++
	if this.IsVerbose() {
		fmt.Printf("Remove %s\n", path)
	}
}

// / @return whether the cleaner is in verbose mode.
func (this *Cleaner) IsVerbose() bool {
	return this.config_.Verbosity != QUIET && (this.config_.Verbosity == VERBOSE || this.config_.DryRun)
}

func (this *Cleaner) Remove(path string) {
	if !this.IsAlreadyRemoved(path) {
		this.removed_[path] = true
		if this.config_.DryRun {
			if this.FileExists(path) {
				this.Report(path)
			}
		} else {
			ret := this.RemoveFile(path)
			if ret == 0 {
				this.Report(path)
			} else if ret == -1 {
				this.status_ = 1
			}
		}
	}
}

func (this *Cleaner) IsAlreadyRemoved(path string) bool {
	_, ok := this.removed_[path]
	return !ok
}

func (this *Cleaner) RemoveEdgeFiles(edge *Edge) {
	depfile := edge.GetUnescapedDepfile()
	if depfile != "" {
		this.Remove(depfile)
	}

	rspfile := edge.GetUnescapedRspfile()
	if rspfile != "" {
		this.Remove(rspfile)
	}
}

func (this *Cleaner) PrintHeader() {
	if this.config_.Verbosity == QUIET {
		return
	}
	fmt.Printf("Cleaning...")
	if this.IsVerbose() {
		fmt.Printf("\n")
	} else {
		fmt.Printf(" ")
	}
	//fflush(stdout);
}

func (this *Cleaner) PrintFooter() {
	if this.config_.Verbosity == QUIET {
		return
	}
	fmt.Printf("%d files.\n", this.cleaned_files_count_)
}

func (this *Cleaner) CleanAll(generator bool) int {
	this.Reset()
	this.PrintHeader()
	this.LoadDyndeps()
	for _, e := range this.state_.edges_ {
		// Do not try to remove phony targets
		if e.is_phony() {
			continue
		}
		// Do not remove generator's files unless generator specified.
		if !generator && e.GetBindingBool("generator") {
			continue
		}
		for _, out_node := range e.outputs_ {
			this.Remove(out_node.path())
		}

		this.RemoveEdgeFiles(e)
	}
	this.PrintFooter()
	return this.status_
}

func (this *Cleaner) CleanDead(entries Entries) int {
	this.Reset()
	this.PrintHeader()
	this.LoadDyndeps()
	for first, _ := range entries {
		n := this.state_.LookupNode(first)
		// Detecting stale outputs works as follows:
		//
		// - If it has no Node, it is not in the build graph, or the deps log
		//   anymore, hence is stale.
		//
		// - If it isn't an output or input for any edge, it comes from a stale
		//   entry in the deps log, but no longer referenced from the build
		//   graph.
		//
		if n == nil || (n.in_edge() == nil && len(n.out_edges()) == 0) {
			this.Remove(first)
		}
	}
	this.PrintFooter()
	return this.status_
}

func (this *Cleaner) DoCleanTarget(target *Node) {
	e := target.in_edge()
	if e != nil {
		// Do not try to remove phony targets
		if !e.is_phony() {
			this.Remove(target.path())
			this.RemoveEdgeFiles(e)
		}
		for _, n := range e.inputs_ {
			next := n
			// call DoCleanTarget recursively if this node has not been visited
			if _, ok := this.cleaned_[next]; !ok {
				this.DoCleanTarget(next)
			}
		}
	}

	// mark this target to be cleaned already
	this.cleaned_[target] = true
}

func (this *Cleaner) CleanTarget(target *Node) int {
	if target == nil {
		panic("target==nil")
	}

	this.Reset()
	this.PrintHeader()
	this.LoadDyndeps()
	this.DoCleanTarget(target)
	this.PrintFooter()
	return this.status_
}

func (this *Cleaner) CleanTargetByKey(target string) int {
	if target == "" {
		panic("target ==\"\"")
	}

	this.Reset()
	node := this.state_.LookupNode(target)
	if node != nil {
		this.CleanTarget(node)
	} else {
		Error("unknown target '%s'", target)
		this.status_ = 1
	}
	return this.status_
}

func (this *Cleaner) CleanTargets(targets []string) int {
	this.Reset()
	this.PrintHeader()
	this.LoadDyndeps()
	for i := 0; i < len(targets); i++ {
		target_name := targets[i]
		if target_name == "" {
			Error("failed to canonicalize '': empty path")
			this.status_ = 1
			continue
		}
		slash_bits := uint64(0)
		CanonicalizePath(&target_name, &slash_bits)
		target := this.state_.LookupNode(target_name)
		if target != nil {
			if this.IsVerbose() {
				fmt.Printf("Target %s\n", target_name)
			}
			this.DoCleanTarget(target)
		} else {
			Error("unknown target '%s'", target_name)
			this.status_ = 1
		}
	}
	this.PrintFooter()
	return this.status_
}

func (this *Cleaner) DoCleanRule(rule *Rule) {
	if rule == nil {
		panic("rule==nil")
	}

	for _, e := range this.state_.edges_ {
		if e.rule().name() == rule.name() {
			for _, out_node := range e.outputs_ {
				this.Remove(out_node.path())
				this.RemoveEdgeFiles(e)
			}
		}
	}
}

func (this *Cleaner) CleanRule(rule *Rule) int {
	if rule == nil {
		panic("rule==nil")
	}

	this.Reset()
	this.PrintHeader()
	this.LoadDyndeps()
	this.DoCleanRule(rule)
	this.PrintFooter()
	return this.status_
}

func (this *Cleaner) CleanRuleByKey(rule string) int {
	if rule == "" {
		panic("rule==\"\"")
	}

	this.Reset()
	r := this.state_.bindings_.LookupRule(rule)
	if r != nil {
		this.CleanRule(r)
	} else {
		Error("unknown rule '%s'", rule)
		this.status_ = 1
	}
	return this.status_
}

func (this *Cleaner) CleanRules(rules []string) int {
	if len(rules) == 0 {
		panic("len(rules) = 0")
	}

	this.Reset()
	this.PrintHeader()
	this.LoadDyndeps()
	for i := 0; i < len(rules); i++ {
		rule_name := rules[i]
		rule := this.state_.bindings_.LookupRule(rule_name)
		if rule != nil {
			if this.IsVerbose() {
				fmt.Printf("Rule %s\n", rule_name)
			}
			this.DoCleanRule(rule)
		} else {
			Error("unknown rule '%s'", rule_name)
			this.status_ = 1
		}
	}
	this.PrintFooter()
	return this.status_
}

func (this *Cleaner) Reset() {
	this.status_ = 0
	this.cleaned_files_count_ = 0
	this.removed_ = map[string]bool{}
	this.cleaned_ = map[*Node]bool{}
}

func (this *Cleaner) LoadDyndeps() {
	// Load dyndep files that exist, before they are cleaned.
	for _, e := range this.state_.edges_ {
		var dyndep *Node = nil
		dyndep = e.dyndep_
		if dyndep != nil && dyndep.dyndep_pending() {
			// Capture and ignore errors loading the dyndep file.
			// We clean as much of the graph as we know.
			err := ""
			this.dyndep_loader_.LoadDyndeps(dyndep, &err)
		}
	}
}
