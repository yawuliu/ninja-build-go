package ninja_go

import (
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Subprocess struct {
	cmd         *exec.Cmd
	use_console bool
}

func NewSubprocess(use_console bool) *Subprocess {
	ret := Subprocess{}
	ret.use_console = use_console
	return &ret
}
func (this *Subprocess) Start(set *SubprocessSet, command string) bool {
	if this.use_console {
		if runtime.GOOS == "windows" {
			this.cmd = exec.Command("cmd", "/c", command)
		} else {
			this.cmd = exec.Command("bash", "-c", command)
		}
	} else {
		arr := strings.Split(command, " ")
		if len(arr) > 1 {
			this.cmd = exec.Command(arr[0], arr[1:]...)
		} else {
			this.cmd = exec.Command(arr[0])
		}
	}
	err := this.cmd.Start()
	if err != nil {
		return false
	}
	return true
}

const CONTROL_C_EXIT = 0x00F00F00

func (this *Subprocess) Finish() ExitStatus {
	if this.cmd == nil {
		return ExitFailure
	}
	if this.cmd.ProcessState == nil {
		err := this.cmd.Wait()
		if err != nil {
			log.Fatalf("cmd.Wait: %v", err)
		}
	}
	exit_code := this.cmd.ProcessState.ExitCode()
	if exit_code == 0 {
		return ExitSuccess
	} else if exit_code == CONTROL_C_EXIT {
		return ExitInterrupted
	} else {
		return ExitFailure
	}
}

func (this *Subprocess) Done() bool {
	return this.cmd.ProcessState.Exited()
}

func (this *Subprocess) GetOutput() string {
	buf, _ := this.cmd.Output()
	return string(buf)
}

type SubprocessSet struct {
	running_  []*Subprocess
	finished_ []*Subprocess // std::queue<Subprocess*>
}

// NewSubprocessSet creates a new SubprocessSet.
func NewSubprocessSet() *SubprocessSet {
	return &SubprocessSet{}
}

func (this *SubprocessSet) NotifyInterrupted(dwCtrlType int) bool {
	for _, task := range this.running_ {
		err := task.cmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Fatalln(err)
			continue
		}
	}
	return true
}

// Add adds a new subprocess to the set.
func (this *SubprocessSet) Add(command string, useConsole bool) *Subprocess {
	subprocess := NewSubprocess(useConsole)
	if succ := subprocess.Start(this, command); succ {
		this.running_ = append(this.running_, subprocess)
	} else {
		this.finished_ = append(this.finished_, subprocess)
	}
	return subprocess
}

func (this *SubprocessSet) DoWork() bool {
	if len(this.running_) == 0 {
		return true
	}
	subproc := this.running_[0]
	err := subproc.cmd.Wait()
	succ := true
	if err != nil {
		succ = false
	}
	this.running_ = this.running_[1:]
	this.finished_ = append(this.finished_, subproc)
	return succ
}

// NextFinished returns the next finished subprocess.
func (s *SubprocessSet) NextFinished() *Subprocess {
	if len(s.finished_) == 0 {
		return nil
	}
	subproc := s.finished_[0]
	s.finished_ = s.finished_[1:]
	return subproc
}

// Clear clears the subprocess set.
func (s *SubprocessSet) Clear() {
	for _, sub := range s.running_ {
		if sub.cmd.ProcessState == nil { //running
			sub.cmd.Process.Signal(os.Interrupt)
			//sub.cmd.Process.Kill()
			sub.cmd.Wait()
		}
	}
	s.running_ = nil
}
