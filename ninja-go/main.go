package ninja_go

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func TerminateHandler() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	s := <-quit
	fmt.Println("terminate handler called:", s)
}

func real_main(args []string) error {
	config := NewBuildConfig()
	options := Options{}
	options.InputFile = "build.ninja"

	// setvbuf(stdout, NULL, _IOLBF, BUFSIZ)
	ninja_command := args[0]

	exit_code := ReadFlags(&args, &options, config)
	if exit_code >= 0 {
		os.Exit(exit_code)
	}

	status := Statusfactory(config)

	if options.WorkingDir != "" {
		// The formatting of this string, complete with funny quotes, is
		// so Emacs can properly identify that the cwd has changed for
		// subsequent commands.
		// Don't print this if a tool is being used, so that tool output
		// can be piped into a file without this string showing up.
		if options.Tool == nil && config.Verbosity != NO_STATUS_UPDATE {
			status.Info("Entering directory `%s'", options.WorkingDir)
		}

		if err := os.Chdir(options.WorkingDir); err != nil {
			log.Fatalf("chdir to '%s' - %v", options.WorkingDir, err)
		}
	}

	if options.Tool != nil && options.Tool.When == RUN_AFTER_FLAGS {
		// None of the RUN_AFTER_FLAGS actually use a NinjaMain, but it's needed
		// by other tools.
		// ninja := NewNinjaMain(ninja_command, &config)
		os.Exit(options.Tool.Func1(&options, &args))
	}

	// Limit number of rebuilds, to prevent infinite loops.
	kCycleLimit := 100
	for cycle := 1; cycle <= kCycleLimit; cycle++ {
		ninja := NewNinjaMain(ninja_command, config)

		parser_opts := NewManifestParserOptions()
		if options.PhonyCycleShouldErr {
			parser_opts.PhonyCycleAction = KPhonyCycleActionError
		}
		parser := NewManifestParser(ninja.State_, ninja.DiskInterface, parser_opts)
		var err string
		if !parser.Load(options.InputFile, &err, nil) {
			status.Error("%s", err)
			os.Exit(1)
		}

		if options.Tool != nil && options.Tool.When == RUN_AFTER_LOAD {
			os.Exit(options.Tool.Func1(&options, &args))
		}

		if !ninja.EnsureBuildDirExists() {
			os.Exit(1)
		}

		if !ninja.OpenBuildLog(false) || !ninja.OpenDepsLog(false) {
			os.Exit(1)
		}

		if options.Tool != nil && options.Tool.When == RUN_AFTER_LOGS {
			os.Exit(options.Tool.Func1(&options, &args))
		}

		// Attempt to rebuild the manifest before building anything else
		if ninja.RebuildManifest(options.InputFile, &err, status) {
			// In dry_run mode the regeneration will succeed without changing the
			// manifest forever. Better to return immediately.
			if config.DryRun {
				os.Exit(0)
			}
			// Start the build over with the new manifest.
			continue
		} else if err != "" {
			status.Error("rebuilding '%s': %s", options.InputFile, err)
			os.Exit(1)
		}

		ninja.ParsePreviousElapsedTimes()

		result := ninja.RunBuild(&args, status)
		if GMetrics != nil {
			ninja.DumpMetrics()
		}
		os.Exit(result)
	}

	status.Error("manifest '%s' still dirty after %d tries, perhaps system time is not set", options.InputFile, kCycleLimit)
	os.Exit(1)
	return nil
}

func main() {
	go TerminateHandler()
	err := real_main(os.Args)
	if err != nil {
		log.Println(err)
		os.Exit(2)
	}
}
