package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// zsh's EXTENDED_HISTORY format: ": <start-ts>:<duration>;<command>"
var zshExtendedPattern = regexp.MustCompile(`^: (\d+):(\d+);(.*)$`)

// bash's HISTTIMEFORMAT writes the timestamp as its own comment line
// immediately above the command it belongs to.
var bashTimestampPattern = regexp.MustCompile(`^#(\d+)$`)

// Entry is one command pulled out of a history stream.
type Entry struct {
	Line      int
	Timestamp int64 // unix seconds, 0 if the format didn't carry one
	Command   string
}

// Finding is a single rule violation tied to the line it came from.
type Finding struct {
	Line     int
	Rule     string
	Severity string
	Message  string
}

// Lint reads a history stream one line at a time and writes findings to w
// as they're discovered. It only ever holds the current line in memory
// (plus bufio's read-ahead chunk), so it's safe to point at a history file
// of any size without pre-loading it.
func Lint(r io.Reader, w io.Writer) (int, error) {
	reader := bufio.NewReaderSize(r, 64*1024)
	lineNum := 0
	var pendingTimestamp int64
	total := 0

	for {
		raw, readErr := reader.ReadString('\n')
		if len(raw) == 0 {
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				return total, readErr
			}
		}
		lineNum++
		line := strings.TrimRight(raw, "\r\n")

		if m := bashTimestampPattern.FindStringSubmatch(line); m != nil {
			ts, err := strconv.ParseInt(m[1], 10, 64)
			if err == nil {
				pendingTimestamp = ts
			}
			if readErr == io.EOF {
				break
			}
			continue
		}

		entry := Entry{Line: lineNum}
		if m := zshExtendedPattern.FindStringSubmatch(line); m != nil {
			ts, _ := strconv.ParseInt(m[1], 10, 64)
			entry.Timestamp = ts
			entry.Command = m[3]
		} else {
			entry.Timestamp = pendingTimestamp
			entry.Command = line
			pendingTimestamp = 0
		}

		if strings.TrimSpace(entry.Command) != "" {
			for _, rl := range rules {
				if f := rl.check(entry); f != nil {
					fmt.Fprintf(w, "%d: [%s] %s: %s\n", f.Line, f.Severity, f.Rule, f.Message)
					total++
				}
			}
		}

		if readErr == io.EOF {
			break
		}
	}

	return total, nil
}
