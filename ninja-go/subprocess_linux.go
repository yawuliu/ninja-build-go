//go:build linux

package ninja_go

import "os/exec"

func NewSubprocess(command string, use_console bool) *Subprocess {
	return &Subprocess{
		cmd:          exec.Command("bash", "-c", command),
		use_console_: use_console,
	}
}
