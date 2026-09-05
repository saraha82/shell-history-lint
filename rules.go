package main

import "regexp"

// rule matches a single line of a history entry against one risky pattern.
// Rules are intentionally simple regex checks on the command text - no
// shell parsing, so they can false-positive on quoting edge cases, but
// they're cheap enough to run on every line of a multi-year history file.
type rule struct {
	id       string
	severity string
	pattern  *regexp.Regexp
	message  string
}

func (r rule) check(e Entry) *Finding {
	if r.pattern.MatchString(e.Command) {
		return &Finding{Line: e.Line, Rule: r.id, Severity: r.severity, Message: r.message}
	}
	return nil
}

var rules = []rule{
	{
		id:       "dangerous-rm",
		severity: "error",
		pattern:  regexp.MustCompile(`\brm\s+(-[a-zA-Z]*[rf][a-zA-Z]*\s+)+(-\S+\s+)*(/|~|\$HOME|\*)(\s|$)`),
		message:  "recursive/forced delete aimed at a broad path",
	},
	{
		id:       "pipe-to-shell",
		severity: "error",
		pattern:  regexp.MustCompile(`\b(curl|wget)\b[^|]*\|\s*(sudo\s+)?(sh|bash|zsh)\b`),
		message:  "downloading and piping straight into a shell",
	},
	{
		id:       "chmod-world-writable",
		severity: "warning",
		pattern:  regexp.MustCompile(`\bchmod\b.*\b(777|a\+rwx)\b`),
		message:  "chmod grants world write access",
	},
	{
		id:       "inline-secret",
		severity: "warning",
		pattern:  regexp.MustCompile(`(?i)\b[a-z_]*(password|secret|token|api_?key|access_key)[a-z_]*\s*=\s*['"]?[^\s'"$][^\s'"]{3,}`),
		message:  "credential-looking value written directly in a command",
	},
	{
		id:       "credential-in-url",
		severity: "error",
		pattern:  regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s:/@]+:[^\s:/@]+@`),
		message:  "URL contains an embedded username and password",
	},
}
