package skill

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Install downloads a skill from a Git URL and registers it.
// Clones to a temp dir, validates SKILL.md exists, copies to ~/.roland/skills/<name>/,
// registers in skills.json with source URL and version.
func Install(home, gitURL, name string) (*SkillEntry, error) {
	if name == "" {
		// Derive name from URL.
		name = nameFromURL(gitURL)
	}
	if name == "" {
		return nil, fmt.Errorf("cannot derive skill name from URL %q; use explicit name", gitURL)
	}

	// Check if already installed.
	sc, err := LoadSkills(home)
	if err != nil {
		return nil, err
	}
	if _, exists := sc.Skills[name]; exists {
		return nil, fmt.Errorf("skill %q already installed; use 'roland skill remove %s' first", name, name)
	}

	// Clone to temp dir.
	tmpDir, err := os.MkdirTemp("", "roland-skill-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneDest := filepath.Join(tmpDir, name)
	cmd := exec.Command("git", "clone", "--depth=1", gitURL, cloneDest)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git clone %s: %w", gitURL, err)
	}

	// Validate: must have SKILL.md.
	skillMD := filepath.Join(cloneDest, "SKILL.md")
	if _, err := os.Stat(skillMD); os.IsNotExist(err) {
		return nil, fmt.Errorf("skill at %s does not contain SKILL.md", gitURL)
	}

	// Get version (git tag or short commit hash).
	version := gitVersion(cloneDest)

	// Copy to skills directory.
	skillsDir := DefaultSkillsDir(home)
	destDir := filepath.Join(skillsDir, name)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("create skill dir: %w", err)
	}
	if err := copyDir(cloneDest, destDir); err != nil {
		return nil, fmt.Errorf("copy skill: %w", err)
	}

	// Register in skills.json.
	entry := &SkillEntry{
		Location: destDir,
		Version:  version,
		Source:   gitURL,
	}
	sc.Skills[name] = entry
	if err := SaveSkills(home, sc); err != nil {
		return nil, err
	}

	return entry, nil
}

// Uninstall removes an installed skill and its files.
func Uninstall(home, name string) error {
	sc, err := LoadSkills(home)
	if err != nil {
		return err
	}

	entry, exists := sc.Skills[name]
	if !exists {
		return fmt.Errorf("skill %q not installed", name)
	}

	// Remove skill directory.
	if entry.Location != "" {
		os.RemoveAll(entry.Location)
	}

	// Remove from registry.
	delete(sc.Skills, name)
	return SaveSkills(home, sc)
}

// nameFromURL extracts a skill name from a git URL.
func nameFromURL(url string) string {
	// Handle github.com/user/skill-name or github.com/user/skill-name.git
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// gitVersion returns the latest git tag or short commit hash.
func gitVersion(dir string) string {
	// Try tag first.
	cmd := exec.Command("git", "describe", "--tags", "--exact-match")
	cmd.Dir = dir
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	// Fall back to short hash.
	cmd = exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return "unknown"
}

// copyDir is defined in skill.go — reused here for marketplace installs.
