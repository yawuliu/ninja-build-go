//go:build windows

package ninja_go

import (
	"os/exec"
	"syscall"
	"unsafe"
)

func NewSubprocess(command string, use_console bool) *Subprocess {
	return &Subprocess{
		cmd:          exec.Command("cmd", "/c", command),
		use_console_: use_console,
	}
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

// Close closes the I/O completion port.
func (s *SubprocessSet) Close() {
	syscall.CloseHandle(ioport_)
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

func (s *Subprocess) Wait() ExitStatus {
	s.wg.Wait()
	return s.determineExitStatus()
}

func (this *Subprocess) Finish() ExitStatus {
	return this.Wait()
}

func (this *Subprocess) Done() bool {
	return this.cmd.ProcessState.Exited()
}
