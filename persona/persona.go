// Package persona manages agent behavior templates.
//
// Roland provides 4 built-in personas (builder, researcher, reviewer, planner)
// embedded in the binary, plus support for custom personas stored in the
// filesystem at ROLAND_HOME/personas/.
//
// Custom personas take precedence over built-in ones with the same name.
package persona

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed templates/*.md
var templateFS embed.FS

// PersonaInfo describes a persona and its source.
type PersonaInfo struct {
	Name   string // e.g., "builder"
	Source string // "builtin" or "custom"
	Path   string // Filesystem path (empty for built-in)
}

// builtinNames returns the names of all embedded personas.
func builtinNames() []string {
	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	sort.Strings(names)
	return names
}

// Names returns the names of all built-in personas.
func Names() []string {
	return builtinNames()
}

// Get returns the content of a persona by name.
// Custom personas (in ROLAND_HOME/personas/) take precedence over built-in.
func Get(home, name string) (string, error) {
	// Check custom first.
	customPath := filepath.Join(home, "personas", name+".md")
	if data, err := os.ReadFile(customPath); err == nil {
		return string(data), nil
	}

	// Fall back to embedded.
	data, err := templateFS.ReadFile("templates/" + name + ".md")
	if err != nil {
		return "", fmt.Errorf("persona %q not found", name)
	}
	return string(data), nil
}

// IsValid returns true if the persona exists (built-in or custom).
func IsValid(home, name string) bool {
	_, err := Get(home, name)
	return err == nil
}

// Create creates a new custom persona, optionally copying from a base.
// If fromBase is empty, an empty template is created.
func Create(home, name, fromBase string) error {
	personasDir := filepath.Join(home, "personas")
	if err := os.MkdirAll(personasDir, 0o755); err != nil {
		return fmt.Errorf("create personas dir: %w", err)
	}

	dest := filepath.Join(personasDir, name+".md")
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("persona %q already exists at %s", name, dest)
	}

	var content string
	if fromBase != "" {
		base, err := Get(home, fromBase)
		if err != nil {
			return fmt.Errorf("base persona: %w", err)
		}
		// Replace the title line.
		lines := strings.SplitN(base, "\n", 2)
		content = fmt.Sprintf("# %s\n", strings.Title(name))
		if len(lines) > 1 {
			content += lines[1]
		}
	} else {
		content = fmt.Sprintf("# %s\n\nDescribe this persona's role and constraints here.\n", strings.Title(name))
	}

	return os.WriteFile(dest, []byte(content), 0o644)
}

// Edit returns the filesystem path to a persona file for editing.
// For custom personas, returns the custom file path.
// For built-in personas, creates a custom copy first, then returns that.
func Edit(home, name string) (string, error) {
	personasDir := filepath.Join(home, "personas")
	if err := os.MkdirAll(personasDir, 0o755); err != nil {
		return "", fmt.Errorf("create personas dir: %w", err)
	}

	dest := filepath.Join(personasDir, name+".md")
	if _, err := os.Stat(dest); err == nil {
		return dest, nil // Custom file already exists.
	}

	// Must be a built-in — create a custom copy.
	content, err := Get(home, name)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write persona: %w", err)
	}
	return dest, nil
}

// Delete removes a custom persona file.
// Built-in personas cannot be deleted.
func Delete(home, name string) error {
	// Check if it's a built-in.
	for _, bn := range builtinNames() {
		if bn == name {
			customPath := filepath.Join(home, "personas", name+".md")
			if _, err := os.Stat(customPath); err != nil {
				return fmt.Errorf("cannot delete built-in persona %q", name)
			}
			// Custom override of a built-in — safe to delete (reverts to built-in).
			return os.Remove(customPath)
		}
	}

	// Pure custom persona.
	customPath := filepath.Join(home, "personas", name+".md")
	if _, err := os.Stat(customPath); err != nil {
		return fmt.Errorf("persona %q not found", name)
	}
	return os.Remove(customPath)
}

// ListAll returns all personas (built-in + custom), sorted by name.
// Custom personas that override built-in ones are listed as "custom".
func ListAll(home string) ([]PersonaInfo, error) {
	seen := make(map[string]PersonaInfo)

	// Built-in personas.
	for _, name := range builtinNames() {
		seen[name] = PersonaInfo{Name: name, Source: "builtin"}
	}

	// Custom personas (override built-in if same name).
	personasDir := filepath.Join(home, "personas")
	entries, err := os.ReadDir(personasDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".md")
			seen[name] = PersonaInfo{
				Name:   name,
				Source: "custom",
				Path:   filepath.Join(personasDir, e.Name()),
			}
		}
	}

	// Sort by name.
	result := make([]PersonaInfo, 0, len(seen))
	for _, p := range seen {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}
