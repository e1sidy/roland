// Package embed provides embedded assets for Roland.
package embed

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed CLAUDE.md
var DefaultClaudeMD string

//go:embed claude-settings.json
var DefaultClaudeSettings string

// WriteClaudeMD writes the default CLAUDE.md to the given directory.
// Template variables are substituted: {{.Home}} → homePath.
// If force is false, an existing file is not overwritten.
func WriteClaudeMD(dir, homePath string, force bool) error {
	dest := filepath.Join(dir, "CLAUDE.md")

	if !force {
		if _, err := os.Stat(dest); err == nil {
			return nil // Already exists, don't overwrite.
		}
	}

	content := strings.ReplaceAll(DefaultClaudeMD, "{{.Home}}", homePath)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	return os.WriteFile(dest, []byte(content), 0o644)
}

// WriteClaudeSettings writes the default claude-settings.json to
// the .claude directory within the given target directory.
func WriteClaudeSettings(dir string) error {
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return fmt.Errorf("create .claude dir: %w", err)
	}
	dest := filepath.Join(claudeDir, "settings.json")
	if _, err := os.Stat(dest); err == nil {
		return nil // Already exists.
	}
	return os.WriteFile(dest, []byte(DefaultClaudeSettings), 0o644)
}
