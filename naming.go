package mcptool

import (
	"fmt"
	"regexp"
)

// validToolName matches the convention <platform>_<verb_or_noun>(_<more>)*
// in lowercase snake_case. Allows digits after the first letter.
var validToolName = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)+$`)

// ValidateToolName returns an error if name does not match the canonical
// snake_case convention. Per-package tests should call this against every
// tool to fail fast on naming drift.
func ValidateToolName(name string) error {
	if !validToolName.MatchString(name) {
		return fmt.Errorf("mcptool: invalid tool name %q (want snake_case like linkedin_search_people)", name)
	}
	return nil
}

// ValidateDescription returns an error if desc is empty or longer than
// maxLen. Conventionally maxLen is 120 to keep the agent's tool inventory
// compact.
func ValidateDescription(desc string, maxLen int) error {
	if desc == "" {
		return fmt.Errorf("mcptool: description must not be empty")
	}
	if maxLen > 0 && len(desc) > maxLen {
		return fmt.Errorf("mcptool: description %q exceeds %d chars (got %d)", desc, maxLen, len(desc))
	}
	return nil
}

// ValidateTools applies ValidateToolName and ValidateDescription to every
// tool, plus checks for duplicate Name. Returns the first error encountered;
// per-package tests typically iterate over the returned errs slice via
// ValidateToolsAll.
func ValidateTools(tools []Tool) error {
	seen := map[string]bool{}
	for _, t := range tools {
		if err := ValidateToolName(t.Name); err != nil {
			return err
		}
		if err := ValidateDescription(t.Description, 120); err != nil {
			return fmt.Errorf("tool %s: %w", t.Name, err)
		}
		if seen[t.Name] {
			return fmt.Errorf("mcptool: duplicate tool name %q", t.Name)
		}
		seen[t.Name] = true
		if t.Invoke == nil {
			return fmt.Errorf("tool %s: Invoke must not be nil", t.Name)
		}
		if t.InputSchema == nil {
			return fmt.Errorf("tool %s: InputSchema must not be nil", t.Name)
		}
	}
	return nil
}
