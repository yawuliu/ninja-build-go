package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// RunBrowsePython 函数，模拟C++中的RunBrowsePython函数
func RunBrowsePython(state *State, ninjaCommand string, inputFile string, argc int, argv []string) {
	// 创建管道
	pipefd_r, pipefd_w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	defer pipefd_r.Close()
	defer pipefd_w.Close()

	// 创建子进程
	pid, err := fork()
	if err != nil {
		panic(err)
	}

	if pid > 0 {
		// 父进程
		pipefd_r.Close()
		pipefd_w.Close()

		// 构建Python命令
		command := []string{"python", "-", "--ninja-command", ninjaCommand, "-f", inputFile}
		command = append(command, argv...)
		err = exec.Command(command[0], command[1:]...).Run()
		if err != nil {
			if _, ok := err.(*exec.Error); ok && err.(*exec.Error).Err == exec.ErrNotFound {
				fmt.Printf("ninja: %s is required for the browse tool\n", "python")
			} else {
				panic(err)
			}
		}
		os.Exit(1)
	} else {
		// 子进程
		defer os.Exit(0)

		// 写入Python脚本到管道
		browsePy := []byte(`# Python script content here`)
		_, err = pipefd_w.Write(browsePy)
		if err != nil {
			panic(err)
		}
	}
}

// fork 函数，模拟C++中的fork
func fork() (pid int, err error) {
	pid, _, err = syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	if err != nil {
		return -1, err
	}
	return pid, nil
}
