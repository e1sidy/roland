package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreate_Basic(t *testing.T) {
	home := t.TempDir()
	td, err := Create(home, "st-ab12", "Fix Auth Bug")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if td.TaskID != "st-ab12" {
		t.Errorf("TaskID = %q, want %q", td.TaskID, "st-ab12")
	}
	if td.Slug != "st-ab12-fix-auth-bug" {
		t.Errorf("Slug = %q, want %q", td.Slug, "st-ab12-fix-auth-bug")
	}
	// Directory should exist.
	if _, err := os.Stat(td.Path); err != nil {
		t.Errorf("task dir not found: %v", err)
	}
}

func TestCreate_EmptyTitle(t *testing.T) {
	home := t.TempDir()
	td, err := Create(home, "st-cd56", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if td.Slug != "st-cd56" {
		t.Errorf("Slug = %q, want %q", td.Slug, "st-cd56")
	}
}

func TestOpen_PrefixMatch(t *testing.T) {
	home := t.TempDir()
	_, err := Create(home, "st-ab12", "Some Title Here")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	td, err := Open(home, "st-ab12")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if td.TaskID != "st-ab12" {
		t.Errorf("TaskID = %q, want %q", td.TaskID, "st-ab12")
	}
}

func TestOpen_NotFound(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, "tasks"), 0o755)

	_, err := Open(home, "st-none")
	if err == nil {
		t.Error("Open should fail for non-existent task")
	}
}

func TestRemove(t *testing.T) {
	home := t.TempDir()
	td, _ := Create(home, "st-rm01", "To Remove")

	if err := Remove(home, "st-rm01"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(td.Path); !os.IsNotExist(err) {
		t.Error("task dir still exists after Remove")
	}
}

func TestList_Empty(t *testing.T) {
	home := t.TempDir()
	dirs, err := List(home)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(dirs) != 0 {
		t.Errorf("List() = %d, want 0", len(dirs))
	}
}

func TestList_Multiple(t *testing.T) {
	home := t.TempDir()
	Create(home, "st-aa11", "Alpha")
	Create(home, "st-bb22", "Beta")
	Create(home, "st-cc33", "Gamma")

	dirs, err := List(home)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(dirs) != 3 {
		t.Fatalf("List() = %d, want 3", len(dirs))
	}
	// Should be sorted.
	if dirs[0].TaskID != "st-aa11" {
		t.Errorf("first = %q, want %q", dirs[0].TaskID, "st-aa11")
	}
}

func TestExtractTaskID(t *testing.T) {
	tt := []struct {
		input string
		want  string
	}{
		{"st-ab12-fix-auth", "st-ab12"},
		{"st-ab12.1-refactor", "st-ab12.1"},
		{"st-ab12.1.2-deep-nested", "st-ab12.1.2"},
		{"st-cd56", "st-cd56"},
		{"no-match-here", ""},
		{"/path/to/tasks/st-ef78-something", "st-ef78"},
	}
	for _, tc := range tt {
		if got := ExtractTaskID(tc.input); got != tc.want {
			t.Errorf("ExtractTaskID(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestMakeSlug(t *testing.T) {
	tt := []struct {
		id    string
		title string
		want  string
	}{
		{"st-ab12", "Fix Auth Bug", "st-ab12-fix-auth-bug"},
		{"st-ab12", "", "st-ab12"},
		{"st-ab12", "Hello!!! World???", "st-ab12-hello-world"},
		{"st-ab12", "   spaces   everywhere   ", "st-ab12-spaces-everywhere"},
		{"st-ab12", "UPPERCASE Title", "st-ab12-uppercase-title"},
	}
	for _, tc := range tt {
		if got := makeSlug(tc.id, tc.title); got != tc.want {
			t.Errorf("makeSlug(%q, %q) = %q, want %q", tc.id, tc.title, got, tc.want)
		}
	}
}

func TestMakeSlug_Truncate(t *testing.T) {
	slug := makeSlug("st-ab12", "This is a very long title that should be truncated to fifty characters total")
	if len(slug) > 50 {
		t.Errorf("slug len = %d, want ≤ 50: %q", len(slug), slug)
	}
}
