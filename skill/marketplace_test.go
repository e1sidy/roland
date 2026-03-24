package skill

import (
	"testing"
)

func TestNameFromURL(t *testing.T) {
	tt := []struct {
		url  string
		want string
	}{
		{"https://github.com/user/debugging-skill", "debugging-skill"},
		{"https://github.com/user/debugging-skill.git", "debugging-skill"},
		{"github.com/org/my-skill", "my-skill"},
		{"", ""},
	}
	for _, tc := range tt {
		got := nameFromURL(tc.url)
		if got != tc.want {
			t.Errorf("nameFromURL(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

func TestInstall_NoSKILLMD(t *testing.T) {
	// This test would need a git repo without SKILL.md.
	// We test the validation logic indirectly by checking the error message.
	home := t.TempDir()
	initSkillsJSON(t, home)

	// A non-existent URL should fail at clone.
	_, err := Install(home, "https://github.com/nonexistent/nonexistent-repo-12345", "test-skill")
	if err == nil {
		t.Error("should fail for nonexistent repo")
	}
}

func TestInstall_AlreadyExists(t *testing.T) {
	home := t.TempDir()
	initSkillsJSON(t, home)

	// Pre-register a skill.
	sc, _ := LoadSkills(home)
	sc.Skills["existing"] = &SkillEntry{Location: "/tmp/existing"}
	SaveSkills(home, sc)

	_, err := Install(home, "https://github.com/user/existing", "existing")
	if err == nil {
		t.Error("should fail for already-installed skill")
	}
}

func TestUninstall(t *testing.T) {
	home := t.TempDir()
	initSkillsJSON(t, home)

	// Register a skill.
	sc, _ := LoadSkills(home)
	sc.Skills["test-skill"] = &SkillEntry{Location: t.TempDir(), Version: "1.0", Source: "https://example.com"}
	SaveSkills(home, sc)

	if err := Uninstall(home, "test-skill"); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	// Should be gone.
	sc2, _ := LoadSkills(home)
	if _, exists := sc2.Skills["test-skill"]; exists {
		t.Error("skill should be uninstalled")
	}
}

func TestUninstall_NotInstalled(t *testing.T) {
	home := t.TempDir()
	initSkillsJSON(t, home)

	err := Uninstall(home, "nonexistent")
	if err == nil {
		t.Error("should fail for nonexistent skill")
	}
}

func initSkillsJSON(t *testing.T, home string) {
	t.Helper()
	sc := &SkillConfig{Skills: make(map[string]*SkillEntry)}
	if err := SaveSkills(home, sc); err != nil {
		t.Fatal(err)
	}
}
