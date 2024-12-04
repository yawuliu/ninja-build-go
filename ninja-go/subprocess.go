package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
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

// https://github.com/go-cmd/cmd
func (this *Subprocess) Start(set *SubprocessSet, command string) bool {
	fmt.Println(command)
	// command = strings.ReplaceAll(command, "\\", "/")
	// fmt.Println(command)
	//if true || this.use_console {
	if runtime.GOOS == "windows" {
		this.cmd = exec.Command("cmd") // , "/C", command
	} else {
		this.cmd = exec.Command("bash") //
	}
	//} else {
	//	arr := strings.Split(command, " ")
	//	if len(arr) > 1 {
	//		this.cmd = exec.Command(arr[0], arr[1:]...)
	//	} else {
	//		this.cmd = exec.Command(arr[0])
	//	}
	//}
	buffer := bytes.Buffer{}
	buffer.Write([]byte(command))
	buffer.WriteString("\n")
	this.cmd.Stdin = &buffer
	//stdin, err := this.cmd.StdinPipe()
	//if err != nil {
	//	log.Fatalln(err)
	//	return false
	//}
	// this.cmd.Stdout = os.Stdout
	this.cmd.Stderr = os.Stderr
	//this.cmd.Env = os.Environ()
	err := this.cmd.Start()
	if err != nil {
		return false
	}
	//Write to stdin
	//_, err = stdin.Write([]byte(command + "\n"))
	//if err != nil {
	//	log.Fatalln(err)
	//	return false
	//}
	//
	//// Close stdin to signal the end of input
	//stdin.Close()
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
	buf, _ := this.cmd.CombinedOutput()
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
	fmt.Printf("Add %s\n", command)
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
	if err != nil {
		log.Fatalln(err)
	}
	this.running_ = this.running_[1:]
	this.finished_ = append(this.finished_, subproc)
	return false
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
