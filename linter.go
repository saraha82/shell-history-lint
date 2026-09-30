package main

import (
	"bufio"
	"encoding/json"
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

// Format picks which timestamp convention to look for while parsing.
// Left to guess (FormatAuto), the two conventions are almost always
// distinguishable line by line, but a plain bash command that happens to
// start with ": <digits>:<digits>;" (": " is a real bash no-op builtin) would
// be misread as a zsh timestamp, and a literal "#<digits>" command line in a
// zsh history would be misread as a bash one. --format lets a caller who
// knows which shell wrote the file rule that out.
type Format int

const (
	FormatAuto Format = iota
	FormatBash
	FormatZsh
)

// ParseFormat validates the --format flag value.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "", "auto":
		return FormatAuto, nil
	case "bash":
		return FormatBash, nil
	case "zsh":
		return FormatZsh, nil
	default:
		return FormatAuto, fmt.Errorf("unknown format %q (want auto, bash, or zsh)", s)
	}
}

// Entry is one command pulled out of a history stream.
type Entry struct {
	Line      int
	Timestamp int64 // unix seconds, 0 if the format didn't carry one
	Command   string
}

// Finding is a single rule violation tied to the line it came from.
// The command text is left out on purpose: it is the thing that matched a
// secret rule, and copying it into CI logs would just move the leak.
type Finding struct {
	Line      int    `json:"line"`
	Rule      string `json:"rule"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// Output selects how findings are written.
type Output int

const (
	OutputText Output = iota
	// OutputJSON writes one JSON object per line (JSON Lines) rather than a
	// single array, so findings can still be emitted as they're found
	// without buffering the whole result.
	OutputJSON
)

// ParseOutput validates the --output flag value.
func ParseOutput(s string) (Output, error) {
	switch s {
	case "", "text":
		return OutputText, nil
	case "json":
		return OutputJSON, nil
	default:
		return OutputText, fmt.Errorf("unknown output %q (want text or json)", s)
	}
}

func writeFinding(w io.Writer, out Output, f *Finding) error {
	if out == OutputJSON {
		b, err := json.Marshal(f)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "%s\n", b)
		return err
	}
	_, err := fmt.Fprintf(w, "%d: [%s] %s: %s\n", f.Line, f.Severity, f.Rule, f.Message)
	return err
}

// Lint reads a history stream one line at a time and writes findings to w
// as they're discovered. It only ever holds the current line in memory
// (plus bufio's read-ahead chunk), so it's safe to point at a history file
// of any size without pre-loading it. format controls which timestamp
// convention is recognized; FormatAuto tries both. rules is the active rule
// set - pass the package-level rules for the defaults, or a Config-filtered
// subset. out picks plain text or JSON Lines for the findings.
func Lint(r io.Reader, w io.Writer, format Format, out Output, rules []rule) (int, error) {
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

		if format != FormatZsh {
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
		}

		entry := Entry{Line: lineNum}
		matchedZsh := false
		if format != FormatBash {
			if m := zshExtendedPattern.FindStringSubmatch(line); m != nil {
				ts, _ := strconv.ParseInt(m[1], 10, 64)
				entry.Timestamp = ts
				entry.Command = m[3]
				matchedZsh = true
			}
		}
		if !matchedZsh {
			entry.Timestamp = pendingTimestamp
			entry.Command = line
			pendingTimestamp = 0
		}

		if strings.TrimSpace(entry.Command) != "" {
			for _, rl := range rules {
				if f := rl.check(entry); f != nil {
					if err := writeFinding(w, out, f); err != nil {
						return total, err
					}
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
