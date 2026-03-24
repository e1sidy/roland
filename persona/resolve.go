package persona

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvePersona returns persona content with repo-aware resolution.
// Resolution order:
//  1. ~/.roland/personas/<repo>-<persona>.md (project-specific override)
//  2. ~/.roland/personas/<persona>.md (custom)
//  3. Embedded persona/templates/<persona>.md (built-in)
//
// If repo is empty, falls back to standard Get() behavior (steps 2-3).
func ResolvePersona(home, name, repo string) (string, error) {
	if repo != "" {
		// Try project-specific override first.
		overrideName := repo + "-" + name
		overridePath := filepath.Join(home, "personas", overrideName+".md")
		if data, err := os.ReadFile(overridePath); err == nil {
			return string(data), nil
		}
	}

	// Fall through to standard resolution.
	return Get(home, name)
}

// ListOverrides returns all per-project persona override names.
// An override has the format "<repo>-<persona>.md".
func ListOverrides(home string) ([]string, error) {
	personaDir := filepath.Join(home, "personas")
	entries, err := os.ReadDir(personaDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	builtins := make(map[string]bool)
	for _, name := range Names() {
		builtins[name] = true
	}

	var overrides []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		// An override has a dash and the base name is a valid persona.
		if idx := strings.LastIndex(name, "-"); idx > 0 {
			base := name[idx+1:]
			if builtins[base] || isCustomPersona(home, base) {
				overrides = append(overrides, name)
			}
		}
	}
	return overrides, nil
}

// isCustomPersona checks if a persona exists as a custom file.
func isCustomPersona(home, name string) bool {
	path := filepath.Join(home, "personas", name+".md")
	_, err := os.Stat(path)
	return err == nil
}
