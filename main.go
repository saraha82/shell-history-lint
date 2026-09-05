package main

import (
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
	var in io.Reader

	if len(args) > 0 {
		f, err := os.Open(args[0])
		if err != nil {
			fmt.Fprintf(stderr, "shell-history-lint: %v\n", err)
			return 2
		}
		defer f.Close()
		in = f
	} else {
		in = os.Stdin
	}

	count, err := Lint(in, stdout)
	if err != nil {
		fmt.Fprintf(stderr, "shell-history-lint: %v\n", err)
		return 2
	}
	if count > 0 {
		return 1
	}
	return 0
}
