package main

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

func real_main() error {
	config := BuildConfig{}
	options := Options{}
	options.input_file = "build.ninja"

	// setvbuf(stdout, NULL, _IOLBF, BUFSIZ)
	ninja_command := os.Args[0]

	exit_code := ReadFlags(os.Args, &options, &config)
	if exit_code >= 0 {
		os.Exit(exit_code)
	}

	status := Statusfactory(&config)

	if options.working_dir != "" {
		// The formatting of this string, complete with funny quotes, is
		// so Emacs can properly identify that the cwd has changed for
		// subsequent commands.
		// Don't print this if a tool is being used, so that tool output
		// can be piped into a file without this string showing up.
		if options.tool == nil && config.verbosity != NO_STATUS_UPDATE {
			status.Info("Entering directory `%s'", options.working_dir)
		}

		if err := os.Chdir(options.working_dir); err != nil {
			log.Fatalf("chdir to '%s' - %v", options.working_dir, err)
		}
	}

	if options.tool != nil && options.tool.when == RUN_AFTER_FLAGS {
		// None of the RUN_AFTER_FLAGS actually use a NinjaMain, but it's needed
		// by other tools.
		// ninja := NewNinjaMain(ninja_command, &config)
		os.Exit(options.tool.func1(&options, os.Args))
	}

	// Limit number of rebuilds, to prevent infinite loops.
	kCycleLimit := 100
	for cycle := 1; cycle <= kCycleLimit; cycle++ {
		ninja := NewNinjaMain(ninja_command, &config)

		parser_opts := NewManifestParserOptions()
		if options.phony_cycle_should_err {
			parser_opts.phony_cycle_action_ = kPhonyCycleActionError
		}
		parser := NewManifestParser(&ninja.state_, ninja.disk_interface_, parser_opts)
		var err string
		if !parser.Load(options.input_file, &err, nil) {
			status.Error("%s", err)
			os.Exit(1)
		}

		if options.tool != nil && options.tool.when == RUN_AFTER_LOAD {
			os.Exit(options.tool.func1(&options, os.Args))
		}

		if !ninja.EnsureBuildDirExists() {
			os.Exit(1)
		}

		if !ninja.OpenBuildLog(false) || !ninja.OpenDepsLog(false) {
			os.Exit(1)
		}

		if options.tool != nil && options.tool.when == RUN_AFTER_LOGS {
			os.Exit(options.tool.func1(&options, os.Args))
		}

		// Attempt to rebuild the manifest before building anything else
		if ninja.RebuildManifest(options.input_file, &err, status) {
			// In dry_run mode the regeneration will succeed without changing the
			// manifest forever. Better to return immediately.
			if config.dry_run {
				os.Exit(0)
			}
			// Start the build over with the new manifest.
			continue
		} else if err != "" {
			status.Error("rebuilding '%s': %s", options.input_file, err)
			os.Exit(1)
		}

		ninja.ParsePreviousElapsedTimes()

		result := ninja.RunBuild(os.Args, status)
		if g_metrics != nil {
			ninja.DumpMetrics()
		}
		os.Exit(result)
	}

	status.Error("manifest '%s' still dirty after %d tries, perhaps system time is not set", options.input_file, kCycleLimit)
	os.Exit(1)
	return nil
}

func main() {
	go TerminateHandler()
	err := real_main()
	if err != nil {
		log.Println(err)
		os.Exit(2)
	}
}
