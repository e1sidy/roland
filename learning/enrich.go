package learning

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/e1sidy/roland/persona"
)

const learnedHeader = "## Learned Patterns"

// EnrichPersona appends learned patterns to a persona file.
// Creates a custom persona override if the persona is built-in.
func EnrichPersona(home, personaName string, patterns []Pattern) error {
	content, err := persona.Get(home, personaName)
	if err != nil {
		return fmt.Errorf("get persona %s: %w", personaName, err)
	}

	// Build the learned section.
	var lines []string
	lines = append(lines, "", learnedHeader, "")
	for _, p := range patterns {
		taskRefs := strings.Join(p.TaskIDs, ", ")
		lines = append(lines, fmt.Sprintf("- %s (from %s)", p.Text, taskRefs))
	}
	learnedSection := strings.Join(lines, "\n")

	// If existing content has a learned section, replace it.
	if idx := strings.Index(content, learnedHeader); idx >= 0 {
		content = strings.TrimRight(content[:idx], "\n")
	}

	content = content + "\n" + learnedSection + "\n"

	// Write to custom persona file (never overwrite built-in templates).
	personaDir := filepath.Join(home, "personas")
	if err := os.MkdirAll(personaDir, 0o755); err != nil {
		return fmt.Errorf("create personas dir: %w", err)
	}
	path := filepath.Join(personaDir, personaName+".md")
	return os.WriteFile(path, []byte(content), 0o644)
}

// ShowLearnings extracts the learned patterns section from a persona.
// Returns empty string if no learnings exist.
func ShowLearnings(home, personaName string) (string, error) {
	content, err := persona.Get(home, personaName)
	if err != nil {
		return "", fmt.Errorf("get persona %s: %w", personaName, err)
	}

	idx := strings.Index(content, learnedHeader)
	if idx < 0 {
		return "", nil
	}

	return strings.TrimSpace(content[idx:]), nil
}

// ResetLearnings removes the learned patterns section from a persona file.
func ResetLearnings(home, personaName string) error {
	content, err := persona.Get(home, personaName)
	if err != nil {
		return fmt.Errorf("get persona %s: %w", personaName, err)
	}

	idx := strings.Index(content, learnedHeader)
	if idx < 0 {
		return nil // nothing to reset
	}

	content = strings.TrimRight(content[:idx], "\n") + "\n"

	personaDir := filepath.Join(home, "personas")
	if err := os.MkdirAll(personaDir, 0o755); err != nil {
		return fmt.Errorf("create personas dir: %w", err)
	}
	path := filepath.Join(personaDir, personaName+".md")
	return os.WriteFile(path, []byte(content), 0o644)
}
