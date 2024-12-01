package main

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

type Subprocess struct {
	cmd          *exec.Cmd
	output       string
	wg           sync.WaitGroup
	use_console_ bool
}

func NewSubprocess(command string, use_console bool) *Subprocess {
	return &Subprocess{
		cmd:          exec.Command("cmd", "/c", command),
		use_console_: use_console,
	}
}

func (this *Subprocess) Finish() ExitStatus {
	return this.Wait()
}

func (this *Subprocess) Done() bool {
	return this.cmd.ProcessState.Exited()
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
func (s *Subprocess) Wait() ExitStatus {
	s.wg.Wait()
	return s.determineExitStatus()
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

// determineExitStatus determines the exit status of the subprocess.
func (s *Subprocess) determineExitStatus() ExitStatus {
	if s.cmd.ProcessState.Exited() {
		if s.cmd.ProcessState.ExitCode() == 0 {
			return ExitSuccess
		} else if s.cmd.ProcessState.ExitCode() == 3 {
			return ExitInterrupted
		}
	}
	return ExitFailure
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

// ioport_ is the I/O completion port for the subprocess set.
var ioport_ syscall.Handle

// NewSubprocessSet creates a new SubprocessSet.
func NewSubprocessSet() *SubprocessSet {
	var err error
	ioport_, err = syscall.CreateIoCompletionPort(syscall.InvalidHandle, 0, 0, 1)
	if err != nil {
		panic(err)
	}
	return &SubprocessSet{}
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

// DoWork waits for any state change in subprocesses.
func (s *SubprocessSet) DoWork() bool {
	var bytesRead uint32
	var key uint32
	var overlapped *syscall.Overlapped

	err := syscall.GetQueuedCompletionStatus(ioport_, &bytesRead, &key, &overlapped, syscall.INFINITE)
	if err != nil {
		panic(err)
	}

	sub := *(**Subprocess)(unsafe.Pointer(&key))
	sub.OnPipeReady()

	if sub.Done() {
		s.running_ = removeSubprocess(s.running_, sub)
		s.finished_ = append(s.finished_, sub)
	}

	return false
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

// Close closes the I/O completion port.
func (s *SubprocessSet) Close() {
	syscall.CloseHandle(ioport_)
}

func removeSubprocess(slice []*Subprocess, sub *Subprocess) []*Subprocess {
	for i, s := range slice {
		if s == sub {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
