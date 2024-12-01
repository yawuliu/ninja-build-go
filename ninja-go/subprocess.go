package ninja_go

import (
	"fmt"
	"os/exec"
	"sync"
)

type Subprocess struct {
	cmd          *exec.Cmd
	output       string
	wg           sync.WaitGroup
	use_console_ bool
}

func (this *Subprocess) GetOutput() string {
	return this.output
}

func (this *Subprocess) Clear() {
	//if pipe_ {
	//	if !CloseHandle(pipe_) {
	//		Win32Fatal("CloseHandle")
	//	}
	//}
	//// Reap child if forgotten.
	//if child_ {
	//	this.Finish()
	//}
}

// captureOutput captures the output of the subprocess.
func (s *Subprocess) captureOutput() {
	defer s.wg.Done()
	out, err := s.cmd.CombinedOutput()
	if err != nil {
		s.output = fmt.Sprintf("Error: %v\nOutput: %s", err, out)
	} else {
		s.output = string(out)
	}
}

func (this *Subprocess) Start(set *SubprocessSet, command string) bool {
	this.wg.Add(1)
	var err error
	if this.use_console_ {
		err = this.cmd.Start()
	} else {
		err = this.cmd.Start()
		if err == nil {
			go this.captureOutput()
		}
	}
	if err != nil {
		return false
	}
	return true
}
func (this *Subprocess) OnPipeReady() {}

type SubprocessSet struct {
	running_  []*Subprocess
	finished_ []*Subprocess // std::queue<Subprocess*>
}

// Add adds a new subprocess to the set.
func (this *SubprocessSet) Add(command string, useConsole bool) *Subprocess {
	sub := NewSubprocess(command, useConsole)
	if succ := sub.Start(this, command); !succ {
		return nil
	}
	this.running_ = append(this.running_, sub)
	return sub
}

// NextFinished returns the next finished subprocess.
func (s *SubprocessSet) NextFinished() *Subprocess {
	if len(s.finished_) == 0 {
		return nil
	}
	sub := s.finished_[0]
	s.finished_ = s.finished_[1:]
	return sub
}

// Clear clears the subprocess set.
func (s *SubprocessSet) Clear() {
	for _, sub := range s.running_ {
		sub.cmd.Process.Kill()
		sub.cmd.Wait()
	}
	s.running_ = nil
}

func removeSubprocess(slice []*Subprocess, sub *Subprocess) []*Subprocess {
	for i, s := range slice {
		if s == sub {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
