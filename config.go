package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config lets a user selectively disable rules or override their severity
// without touching the source. It's plain JSON so it can sit in a dotfiles
// repo next to the history file it's meant to check.
type Config struct {
	Disable  []string          `json:"disable"`
	Severity map[string]string `json:"severity"`
}

// LoadConfig reads and parses a config file from path.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return c, nil
}

var validSeverities = map[string]bool{"error": true, "warning": true}

// Apply returns the subset of defaultRules left after the config's disable
// list is removed, with severity overrides applied. It errors on a rule id
// or severity the config references but doesn't exist - almost always a
// typo, and failing loudly beats silently linting with fewer checks than
// the user asked for.
func (c Config) Apply(defaultRules []rule) ([]rule, error) {
	byID := make(map[string]bool, len(defaultRules))
	for _, r := range defaultRules {
		byID[r.id] = true
	}

	disabled := make(map[string]bool, len(c.Disable))
	for _, id := range c.Disable {
		if !byID[id] {
			return nil, fmt.Errorf("config: unknown rule %q in disable list", id)
		}
		disabled[id] = true
	}

	for id, sev := range c.Severity {
		if !byID[id] {
			return nil, fmt.Errorf("config: unknown rule %q in severity overrides", id)
		}
		if !validSeverities[sev] {
			return nil, fmt.Errorf("config: invalid severity %q for rule %q (want error or warning)", sev, id)
		}
	}

	active := make([]rule, 0, len(defaultRules))
	for _, r := range defaultRules {
		if disabled[r.id] {
			continue
		}
		if sev, ok := c.Severity[r.id]; ok {
			r.severity = sev
		}
		active = append(active, r)
	}
	return active, nil
}
