package ninja_go

type RealCommandRunner struct {
	CommandRunner
	config_          *BuildConfig
	subprocs_        SubprocessSet
	subproc_to_edge_ map[*Subprocess]*Edge
}

func NewRealCommandRunner(config *BuildConfig) *RealCommandRunner {
	ret := RealCommandRunner{}
	ret.config_ = config
	ret.subproc_to_edge_ = make(map[*Subprocess]*Edge)
	return &ret
}
func (this *RealCommandRunner) CanRunMore() int64 {
	subproc_number := len(this.subprocs_.running_) + len(this.subprocs_.finished_)

	capacity := float64(this.config_.Parallelism - subproc_number)

	if this.config_.MaxLoadAverage > 0.0 {
		load_capacity := this.config_.MaxLoadAverage - GetLoadAverage()
		if load_capacity < capacity {
			capacity = load_capacity
		}
	}

	if capacity < 0 {
		capacity = 0
	}

	if capacity == 0 && len(this.subprocs_.running_) == 0 {
		// Ensure that we make progress.
		capacity = 1
	}

	return int64(capacity)
}

func (this *RealCommandRunner) StartCommand(edge *Edge) bool {
	command := edge.EvaluateCommand(false)
	subproc := this.subprocs_.Add(command, edge.use_console())
	if subproc == nil {
		return false
	}
	this.subproc_to_edge_[subproc] = edge

	return true
}

func (this *RealCommandRunner) WaitForCommand(result *Result) bool {
	var subproc *Subprocess = nil
	for {
		subproc = this.subprocs_.NextFinished()
		if subproc != nil {
			break
		}
		interrupted := this.subprocs_.DoWork()
		if interrupted {
			return false
		}
	}

	result.status = subproc.Finish()
	result.output = subproc.GetOutput()

	second, _ := this.subproc_to_edge_[subproc]
	result.edge = second
	delete(this.subproc_to_edge_, subproc)

	return true
}

func (this *RealCommandRunner) GetActiveEdges() []*Edge {
	edges := []*Edge{}
	for _, second := range this.subproc_to_edge_ {
		edges = append(edges, second)
	}
	return edges
}
func (this *RealCommandRunner) Abort() {
	this.subprocs_.Clear()
}
