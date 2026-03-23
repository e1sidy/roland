package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/e1sidy/roland/internal/gitutil"
)

// skillsDir returns the Claude skills directory within a task dir.
func skillsDir(taskDir string) string {
	return filepath.Join(taskDir, ".claude", "skills")
}

// Inject creates a symlink from the task directory's skills dir to the skill location.
// Idempotent: if the symlink already points to the correct location, it's a no-op.
func Inject(skillName, skillLocation, taskDir string) error {
	dir := skillsDir(taskDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create skills dir: %w", err)
	}

	link := filepath.Join(dir, skillName)

	// Check if symlink already exists and is correct.
	if gitutil.IsSymlink(link) {
		target, err := os.Readlink(link)
		if err == nil && target == skillLocation {
			return nil // Already correct.
		}
		os.Remove(link) // Wrong target, recreate.
	}

	return os.Symlink(skillLocation, link)
}

// Eject removes a skill symlink from the task directory.
func Eject(skillName, taskDir string) error {
	link := filepath.Join(skillsDir(taskDir), skillName)
	if !gitutil.IsSymlink(link) {
		return nil // Not injected.
	}
	return os.Remove(link)
}

// Injected returns the names of skills currently injected into the task directory.
func Injected(taskDir string) ([]string, error) {
	dir := skillsDir(taskDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read skills dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		link := filepath.Join(dir, e.Name())
		if gitutil.IsSymlink(link) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// InjectMatching loads the skill registry, finds matching skills, and injects them all.
// Returns the list of injected skill names.
func InjectMatching(home, taskDir string, ctx *MatchContext) ([]string, error) {
	sc, err := LoadSkills(home)
	if err != nil {
		return nil, fmt.Errorf("load skills: %w", err)
	}

	names := MatchAll(sc, ctx)
	var injected []string

	for _, name := range names {
		entry := sc.Skills[name]
		if err := Inject(name, entry.Location, taskDir); err != nil {
			return injected, fmt.Errorf("inject %q: %w", name, err)
		}
		injected = append(injected, name)
	}

	return injected, nil
}
