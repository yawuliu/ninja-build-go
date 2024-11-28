package main

func RunBrowsePython(state *State, ninja_command,
	input_file string, args []string) {
	/*
		  // Fork off a Python process and have it run our code via its stdin.
		  // (Actually the Python process becomes the parent.)
		   pipefd := [2]int{}
		  if (pipe(pipefd) < 0) {
		    fmt.Fprintf(os.Stderr, "ninja: pipe");
		    return;
		  }

		  pid := fork();
		  if (pid < 0) {
		    fmt.Fprintf(os.Stderr, "ninja: fork");
		    return;
		  }

		  if (pid > 0) {  // Parent.
		    close(pipefd[1]);
		    for {
		      if (dup2(pipefd[0], 0) < 0) {
		        fmt.Fprintf(os.Stderr, "ninja: dup2");
		        break;
		      }

		      command := []string{}
		      command = append(command, NINJA_PYTHON);
		      command = append(command,"-");
		      command = append(command,"--ninja-command");
		      command = append(command,ninja_command);
		      command = append(command,"-f");
		      command = append(command,input_file);
		      for i := 0; i < len(args); i++ {
		          command= append(command, args[i])
		      }
		      command= append(command, "\n")
		      execvp(command[0], const_cast<char**>(&command[0]));
		      if (errno == ENOENT) {
		        fmt.Printf("ninja: %s is required for the browse tool\n", NINJA_PYTHON);
		      } else {
		        fmt.Fprintf(os.Stderr,"ninja: execvp");
		      }
		      break;
		    }
		    os.Exit(1);
		  } else {  // Child.
		    close(pipefd[0]);

		    // Write the script file into the stdin of the Python process.
		    // Only write n - 1 bytes, because Python 3.11 does not allow null
		    // bytes in source code anymore, so avoid writing the null string
		    // terminator.
		    // See https://github.com/python/cpython/issues/96670
		    kBrowsePyLength := len(kBrowsePy) - 1;
		    len := write(pipefd[1], kBrowsePy, kBrowsePyLength);
		    if len < kBrowsePyLength {
				fmt.Fprintf(os.Stderr, "ninja: write")
			}
		    close(pipefd[1]);
			os.Exit(0);
		  }
	*/
}
