package main

import "github.com/edwingeng/deque"

type Subprocess struct {
	buf_         string
	use_console_ bool
}

func (this *Subprocess) Finish() ExitStatus {}

func (this *Subprocess) Done() bool {}

func (this *Subprocess) GetOutput() string {}

func newSubprocess(use_console bool) *Subprocess {}
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
func (this *Subprocess) Start(set *SubprocessSet, command string) bool {}
func (this *Subprocess) OnPipeReady()                                  {}

type SubprocessSet struct {
	running_  []*Subprocess
	finished_ deque.Deque // std::queue<Subprocess*>
}

func NewSubprocessSet() *SubprocessSet     {}
func (this *SubprocessSet) SubprocessSet() {}

func (this *SubprocessSet) Add(command string, use_console bool) *Subprocess {

}
func (this *SubprocessSet) DoWork() bool              {}
func (this *SubprocessSet) NextFinished() *Subprocess {}
func (this *SubprocessSet) Clear()                    {}
