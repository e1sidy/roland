package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func createTestSkill(t *testing.T, dir, name string) string {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# "+name), 0o644)
	return skillDir
}

func TestAdd_Basic(t *testing.T) {
	home := t.TempDir()
	skillDir := createTestSkill(t, t.TempDir(), "debugging")

	entry, err := Add(home, skillDir, "debugging", true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if entry.Location != skillDir {
		t.Errorf("location = %q, want %q", entry.Location, skillDir)
	}
}

func TestAdd_Duplicate(t *testing.T) {
	home := t.TempDir()
	skillDir := createTestSkill(t, t.TempDir(), "debugging")

	Add(home, skillDir, "debugging", true)
	_, err := Add(home, skillDir, "debugging", true)
	if err == nil {
		t.Error("Add should fail on duplicate")
	}
}

func TestAdd_MissingSKILLMD(t *testing.T) {
	home := t.TempDir()
	emptyDir := t.TempDir()

	_, err := Add(home, emptyDir, "badskill", true)
	if err == nil {
		t.Error("Add should fail without SKILL.md")
	}
}

func TestAdd_CopyMode(t *testing.T) {
	home := t.TempDir()
	skillDir := createTestSkill(t, t.TempDir(), "local")

	entry, err := Add(home, skillDir, "local", false)
	if err != nil {
		t.Fatalf("Add (copy): %v", err)
	}
	// Location should be inside the home skills dir, not the original.
	if entry.Location == skillDir {
		t.Error("copy mode should change location to skills dir")
	}
	// SKILL.md should exist in the new location.
	if _, err := os.Stat(filepath.Join(entry.Location, "SKILL.md")); err != nil {
		t.Error("SKILL.md not found in copied location")
	}
}

func TestRemove(t *testing.T) {
	home := t.TempDir()
	skillDir := createTestSkill(t, t.TempDir(), "debugging")
	Add(home, skillDir, "debugging", true)

	if err := Remove(home, "debugging", false); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	skills, _ := List(home)
	if len(skills) != 0 {
		t.Error("skill should be removed")
	}
}

func TestRemove_NotFound(t *testing.T) {
	home := t.TempDir()
	err := Remove(home, "nonexistent", false)
	if err == nil {
		t.Error("Remove should fail for unknown skill")
	}
}

func TestSetTags(t *testing.T) {
	home := t.TempDir()
	skillDir := createTestSkill(t, t.TempDir(), "debugging")
	Add(home, skillDir, "debugging", true)

	if err := SetTags(home, "debugging", []string{"builder"}, []string{"bug"}, []string{"urgent"}); err != nil {
		t.Fatalf("SetTags: %v", err)
	}

	entry, _ := Get(home, "debugging")
	if len(entry.Personas) != 1 || entry.Personas[0] != "builder" {
		t.Errorf("Personas = %v", entry.Personas)
	}
	if len(entry.TaskTypes) != 1 || entry.TaskTypes[0] != "bug" {
		t.Errorf("TaskTypes = %v", entry.TaskTypes)
	}
	if len(entry.Tags) != 1 || entry.Tags[0] != "urgent" {
		t.Errorf("Tags = %v", entry.Tags)
	}
}

func TestList_Sorted(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	createTestSkill(t, dir, "zebra")
	createTestSkill(t, dir, "alpha")
	Add(home, filepath.Join(dir, "zebra"), "zebra", true)
	Add(home, filepath.Join(dir, "alpha"), "alpha", true)

	skills, err := List(home)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("List = %d, want 2", len(skills))
	}
	if skills[0].Name != "alpha" {
		t.Errorf("first = %q, want %q", skills[0].Name, "alpha")
	}
}

func TestMatch_Persona(t *testing.T) {
	entry := &SkillEntry{Personas: []string{"builder"}}
	ctx := &MatchContext{Persona: "builder"}
	if !Match(entry, ctx) {
		t.Error("should match on persona")
	}
}

func TestMatch_TaskType(t *testing.T) {
	entry := &SkillEntry{TaskTypes: []string{"bug"}}
	ctx := &MatchContext{TaskType: "bug"}
	if !Match(entry, ctx) {
		t.Error("should match on task type")
	}
}

func TestMatch_Tags(t *testing.T) {
	entry := &SkillEntry{Tags: []string{"urgent", "critical"}}
	ctx := &MatchContext{Labels: []string{"urgent"}}
	if !Match(entry, ctx) {
		t.Error("should match on tags")
	}
}

func TestMatch_OR(t *testing.T) {
	entry := &SkillEntry{Personas: []string{"reviewer"}, TaskTypes: []string{"bug"}}
	// Persona doesn't match, but task type does — OR logic.
	ctx := &MatchContext{Persona: "builder", TaskType: "bug"}
	if !Match(entry, ctx) {
		t.Error("should match when any dimension matches (OR)")
	}
}

func TestMatch_None(t *testing.T) {
	entry := &SkillEntry{Personas: []string{"reviewer"}}
	ctx := &MatchContext{Persona: "builder"}
	if Match(entry, ctx) {
		t.Error("should not match when no dimension matches")
	}
}

func TestMatch_EmptyEntry(t *testing.T) {
	entry := &SkillEntry{} // No criteria = manual-only.
	ctx := &MatchContext{Persona: "builder", TaskType: "bug", Labels: []string{"urgent"}}
	if Match(entry, ctx) {
		t.Error("empty entry should never auto-match")
	}
}

func TestInject_Basic(t *testing.T) {
	taskDir := t.TempDir()
	skillDir := t.TempDir()

	if err := Inject("debugging", skillDir, taskDir); err != nil {
		t.Fatalf("Inject: %v", err)
	}

	link := filepath.Join(taskDir, ".claude", "skills", "debugging")
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if target != skillDir {
		t.Errorf("symlink target = %q, want %q", target, skillDir)
	}
}

func TestInject_Idempotent(t *testing.T) {
	taskDir := t.TempDir()
	skillDir := t.TempDir()

	Inject("debugging", skillDir, taskDir)
	if err := Inject("debugging", skillDir, taskDir); err != nil {
		t.Fatalf("second Inject: %v", err)
	}
}

func TestEject(t *testing.T) {
	taskDir := t.TempDir()
	skillDir := t.TempDir()
	Inject("debugging", skillDir, taskDir)

	if err := Eject("debugging", taskDir); err != nil {
		t.Fatalf("Eject: %v", err)
	}

	link := filepath.Join(taskDir, ".claude", "skills", "debugging")
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Error("symlink should be removed after Eject")
	}
}

func TestInjected(t *testing.T) {
	taskDir := t.TempDir()
	Inject("alpha", t.TempDir(), taskDir)
	Inject("beta", t.TempDir(), taskDir)

	names, err := Injected(taskDir)
	if err != nil {
		t.Fatalf("Injected: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("Injected = %d, want 2", len(names))
	}
	if names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("Injected = %v, want [alpha beta]", names)
	}
}

func TestInjectMatching(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	skillDir := createTestSkill(t, dir, "debugging")
	Add(home, skillDir, "debugging", true)
	SetTags(home, "debugging", []string{"builder"}, nil, nil)

	taskDir := t.TempDir()
	ctx := &MatchContext{Persona: "builder"}

	injected, err := InjectMatching(home, taskDir, ctx)
	if err != nil {
		t.Fatalf("InjectMatching: %v", err)
	}
	if len(injected) != 1 || injected[0] != "debugging" {
		t.Errorf("injected = %v, want [debugging]", injected)
	}
}

func TestSkillVersion(t *testing.T) {
	home := t.TempDir()
	skillDir := createTestSkill(t, t.TempDir(), "test-skill")
	Add(home, skillDir, "test-skill", true)

	// Set version manually via the config.
	sc, _ := LoadSkills(home)
	sc.Skills["test-skill"].Version = "1.2.0"
	SaveSkills(home, sc)

	// Reload and verify.
	entry, err := Get(home, "test-skill")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if entry.Version != "1.2.0" {
		t.Errorf("Version = %q, want %q", entry.Version, "1.2.0")
	}
}
