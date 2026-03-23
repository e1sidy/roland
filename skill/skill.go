// Package skill manages Claude Code skills — reusable context directories
// that are auto-injected into task workspaces based on matching criteria.
package skill

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// SkillEntry holds metadata about a registered skill.
type SkillEntry struct {
	// Location is the absolute path to the skill directory.
	Location string `json:"location"`

	// Personas lists persona names that trigger auto-injection.
	Personas []string `json:"personas,omitempty"`

	// TaskTypes lists task types that trigger auto-injection.
	TaskTypes []string `json:"task_types,omitempty"`

	// Tags lists labels/tags that trigger auto-injection.
	Tags []string `json:"tags,omitempty"`

	// Version tracks the skill version for compatibility checks.
	Version string `json:"version,omitempty"`
}

// SkillConfig holds the skill registry.
type SkillConfig struct {
	Schema string                `json:"$schema,omitempty"`
	Skills map[string]*SkillEntry `json:"skills"`
}

// SkillInfo combines a name with its entry for listing.
type SkillInfo struct {
	Name  string
	Entry *SkillEntry
}

// SkillsPath returns the path to skills.json.
func SkillsPath(home string) string {
	return filepath.Join(home, ".skills", "skills.json")
}

// DefaultSkillsDir returns the default directory for skill storage.
func DefaultSkillsDir(home string) string {
	return filepath.Join(home, ".skills")
}

// LoadSkills reads skills.json from the given home directory.
func LoadSkills(home string) (*SkillConfig, error) {
	path := SkillsPath(home)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SkillConfig{Skills: make(map[string]*SkillEntry)}, nil
		}
		return nil, fmt.Errorf("read skills: %w", err)
	}

	var sc SkillConfig
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("parse skills: %w", err)
	}
	if sc.Skills == nil {
		sc.Skills = make(map[string]*SkillEntry)
	}
	return &sc, nil
}

// SaveSkills writes skills.json to the given home directory.
func SaveSkills(home string, sc *SkillConfig) error {
	dir := filepath.Dir(SkillsPath(home))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create skills dir: %w", err)
	}

	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal skills: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(SkillsPath(home), data, 0o644)
}

// Add registers a new skill. If external is false, the skill directory
// is copied into the skills directory.
func Add(home, skillPath, name string, external bool) (*SkillEntry, error) {
	sc, err := LoadSkills(home)
	if err != nil {
		return nil, err
	}

	if _, exists := sc.Skills[name]; exists {
		return nil, fmt.Errorf("skill %q already registered", name)
	}

	// Validate skill has a SKILL.md file.
	absPath, err := filepath.Abs(skillPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	if _, err := os.Stat(filepath.Join(absPath, "SKILL.md")); err != nil {
		return nil, fmt.Errorf("skill directory must contain SKILL.md: %w", err)
	}

	location := absPath
	if !external {
		// Copy into skills directory.
		destDir := filepath.Join(DefaultSkillsDir(home), name)
		if err := copyDir(absPath, destDir); err != nil {
			return nil, fmt.Errorf("copy skill: %w", err)
		}
		location = destDir
	}

	entry := &SkillEntry{Location: location}
	sc.Skills[name] = entry

	if err := SaveSkills(home, sc); err != nil {
		return nil, err
	}
	return entry, nil
}

// Remove unregisters a skill and optionally deletes its files.
func Remove(home, name string, deleteFiles bool) error {
	sc, err := LoadSkills(home)
	if err != nil {
		return err
	}

	entry, exists := sc.Skills[name]
	if !exists {
		return fmt.Errorf("skill %q not registered", name)
	}

	delete(sc.Skills, name)
	if err := SaveSkills(home, sc); err != nil {
		return err
	}

	if deleteFiles {
		os.RemoveAll(entry.Location)
	}
	return nil
}

// SetTags updates the matching criteria for a skill.
func SetTags(home, name string, personas, taskTypes, tags []string) error {
	sc, err := LoadSkills(home)
	if err != nil {
		return err
	}

	entry, exists := sc.Skills[name]
	if !exists {
		return fmt.Errorf("skill %q not registered", name)
	}

	if personas != nil {
		entry.Personas = personas
	}
	if taskTypes != nil {
		entry.TaskTypes = taskTypes
	}
	if tags != nil {
		entry.Tags = tags
	}

	return SaveSkills(home, sc)
}

// List returns all registered skills, sorted by name.
func List(home string) ([]*SkillInfo, error) {
	sc, err := LoadSkills(home)
	if err != nil {
		return nil, err
	}

	var skills []*SkillInfo
	for name, entry := range sc.Skills {
		skills = append(skills, &SkillInfo{Name: name, Entry: entry})
	}
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	return skills, nil
}

// Get returns a single skill entry.
func Get(home, name string) (*SkillEntry, error) {
	sc, err := LoadSkills(home)
	if err != nil {
		return nil, err
	}
	entry, exists := sc.Skills[name]
	if !exists {
		return nil, fmt.Errorf("skill %q not registered", name)
	}
	return entry, nil
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}
