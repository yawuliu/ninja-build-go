package ninja_go

import (
	"fmt"
	"os/exec"
)

// RunBrowsePython 函数，模拟C++中的RunBrowsePython函数
func RunBrowsePython(state *State, ninjaCommand string, inputFile string, args []string) {
	// 构建Python命令
	command := []string{"python", "-", "--ninja-command", ninjaCommand, "-f", inputFile}
	command = append(command, args...)
	err := exec.Command(command[0], command[1:]...).Run()
	if err != nil {
		if _, ok := err.(*exec.Error); ok && err.(*exec.Error).Err == exec.ErrNotFound {
			fmt.Printf("ninja: %s is required for the browse tool\n", "python")
		} else {
			panic(err)
		}
	}
}
