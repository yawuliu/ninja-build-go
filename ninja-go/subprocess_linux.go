//go:build linux

package ninja_go

import "os/exec"

func NewSubprocess(command string, use_console bool) *Subprocess {
	return &Subprocess{
		cmd:          exec.Command("bash", "-c", command),
		use_console_: use_console,
	}
}
func NewSubprocessSet() *SubprocessSet {
	return &SubprocessSet{}
}

func (s *SubprocessSet) DoWork() bool {
	return false
}

func (s *SubprocessSet) Close() {

}

func (s *Subprocess) Wait() ExitStatus {
	s.wg.Wait()
	return ExitSuccess
}

func (this *Subprocess) Finish() ExitStatus {
	return this.Wait()
}

func (this *Subprocess) Done() bool {
	return false
}
