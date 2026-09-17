package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run does the actual work and returns a process exit code, so tests (later)
// can call it without touching the real stdio or os.Exit.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("shell-history-lint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	formatFlag := fs.String("format", "auto", "history format to parse: auto, bash, or zsh")
	configFlag := fs.String("config", "", "path to a JSON config file for enabling/disabling and tuning rules")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	format, err := ParseFormat(*formatFlag)
	if err != nil {
		fmt.Fprintf(stderr, "shell-history-lint: %v\n", err)
		return 2
	}

	activeRules := rules
	if *configFlag != "" {
		cfg, err := LoadConfig(*configFlag)
		if err != nil {
			fmt.Fprintf(stderr, "shell-history-lint: %v\n", err)
			return 2
		}
		activeRules, err = cfg.Apply(rules)
		if err != nil {
			fmt.Fprintf(stderr, "shell-history-lint: %v\n", err)
			return 2
		}
	}

	var in io.Reader

	if rest := fs.Args(); len(rest) > 0 {
		f, err := os.Open(rest[0])
		if err != nil {
			fmt.Fprintf(stderr, "shell-history-lint: %v\n", err)
			return 2
		}
		defer f.Close()
		in = f
	} else {
		in = os.Stdin
	}

	count, err := Lint(in, stdout, format, activeRules)
	if err != nil {
		fmt.Fprintf(stderr, "shell-history-lint: %v\n", err)
		return 2
	}
	if count > 0 {
		return 1
	}
	return 0
}
