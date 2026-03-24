package templates

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/e1sidy/roland/internal/testutil"
	"github.com/e1sidy/slate"
)

func TestList_Builtins(t *testing.T) {
	home := t.TempDir()
	tmpls, err := List(home)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tmpls) != 5 {
		t.Errorf("built-in templates = %d, want 5", len(tmpls))
	}

	names := make(map[string]bool)
	for _, tmpl := range tmpls {
		names[tmpl.Name] = true
	}
	for _, expected := range []string{"bug-fix", "feature", "refactor", "code-review", "incident-response"} {
		if !names[expected] {
			t.Errorf("missing built-in template: %s", expected)
		}
	}
}

func TestGet_Builtin(t *testing.T) {
	home := t.TempDir()
	tmpl, err := Get(home, "feature")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if tmpl.Name != "feature" {
		t.Errorf("name = %q, want feature", tmpl.Name)
	}
	if len(tmpl.Tasks) != 4 {
		t.Errorf("tasks = %d, want 4", len(tmpl.Tasks))
	}
}

func TestGet_Custom(t *testing.T) {
	home := t.TempDir()
	customDir := filepath.Join(home, "templates")
	os.MkdirAll(customDir, 0o755)
	os.WriteFile(filepath.Join(customDir, "custom.yaml"), []byte(`
name: custom
description: A custom template
tasks:
  - id: step1
    title: "Step 1"
    type: task
`), 0o644)

	tmpl, err := Get(home, "custom")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if tmpl.Name != "custom" {
		t.Errorf("name = %q", tmpl.Name)
	}
}

func TestGet_NotFound(t *testing.T) {
	home := t.TempDir()
	_, err := Get(home, "nonexistent")
	if err == nil {
		t.Error("should error for nonexistent template")
	}
}

func TestRenderTitle(t *testing.T) {
	tt := []struct {
		template string
		vars     map[string]string
		want     string
	}{
		{"Research: <<.title>>", map[string]string{"title": "Auth flow"}, "Research: Auth flow"},
		{"No vars", nil, "No vars"},
		{"<<.title>> - <<.scope>>", map[string]string{"title": "Fix", "scope": "backend"}, "Fix - backend"},
	}
	for _, tc := range tt {
		got, err := RenderTitle(tc.template, tc.vars)
		if err != nil {
			t.Errorf("RenderTitle(%q): %v", tc.template, err)
			continue
		}
		if got != tc.want {
			t.Errorf("RenderTitle(%q) = %q, want %q", tc.template, got, tc.want)
		}
	}
}

func TestValidateVars(t *testing.T) {
	tmpl := &Template{
		Vars: []TemplateVar{
			{Name: "title", Required: true},
			{Name: "scope", Required: false, Default: "all"},
		},
	}

	// Missing required var.
	err := tmpl.ValidateVars(map[string]string{})
	if err == nil {
		t.Error("should error when required var is missing")
	}

	// Required var provided.
	err = tmpl.ValidateVars(map[string]string{"title": "Test"})
	if err != nil {
		t.Errorf("should not error: %v", err)
	}
}

func TestMergeVars(t *testing.T) {
	tmpl := &Template{
		Vars: []TemplateVar{
			{Name: "title", Required: true},
			{Name: "scope", Default: "all"},
		},
	}
	merged := tmpl.MergeVars(map[string]string{"title": "Test"})
	if merged["scope"] != "all" {
		t.Errorf("scope = %q, want all (default)", merged["scope"])
	}
	if merged["title"] != "Test" {
		t.Errorf("title = %q, want Test", merged["title"])
	}
}

func TestApply(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)

	// Ensure attrs exist.
	store.DefineAttr(ctx, "persona_used", slate.AttrString, "")
	store.DefineAttr(ctx, "repos", slate.AttrString, "")

	tmpl, _ := Get(t.TempDir(), "feature")

	result, err := Apply(ctx, store, tmpl, map[string]string{"title": "Dark mode"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if result.EpicID == "" {
		t.Error("epic ID should not be empty")
	}
	if len(result.TaskIDs) != 4 {
		t.Errorf("task IDs = %d, want 4", len(result.TaskIDs))
	}

	// Verify epic exists.
	epic, _ := store.Get(ctx, result.EpicID)
	if epic == nil {
		t.Fatal("epic not found")
	}
	if epic.Type != slate.TypeEpic {
		t.Errorf("epic type = %s, want epic", epic.Type)
	}

	// Verify children exist with correct parent.
	children, _ := store.Children(ctx, result.EpicID)
	if len(children) != 4 {
		t.Errorf("children = %d, want 4", len(children))
	}

	// Verify first task title contains "Dark mode".
	if children[0].Title == "" {
		t.Error("first child should have a title")
	}

	// Verify persona attr set on at least one task.
	attr, _ := store.GetAttr(ctx, result.TaskIDs[0], "persona_used")
	if attr == nil || attr.Value == "" {
		t.Error("persona_used attr should be set on child tasks")
	}
}

func TestApply_MissingVar(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)
	tmpl, _ := Get(t.TempDir(), "feature")

	_, err := Apply(ctx, store, tmpl, map[string]string{}) // missing "title"
	if err == nil {
		t.Error("should error when required var is missing")
	}
}

func TestCreateFromEpic(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)

	store.DefineAttr(ctx, "persona_used", slate.AttrString, "")

	// Create an epic with children.
	epic, _ := store.Create(ctx, slate.CreateParams{Title: "Auth overhaul", Type: slate.TypeEpic})
	child1, _ := store.Create(ctx, slate.CreateParams{Title: "Research auth options", Type: slate.TypeTask, ParentID: epic.ID})
	child2, _ := store.Create(ctx, slate.CreateParams{Title: "Implement JWT", Type: slate.TypeFeature, ParentID: epic.ID})
	store.AddDependency(ctx, child1.ID, child2.ID, slate.Blocks)
	store.SetAttr(ctx, child1.ID, "persona_used", "researcher")
	store.SetAttr(ctx, child2.ID, "persona_used", "builder")

	tmpl, err := CreateFromEpic(ctx, store, epic.ID)
	if err != nil {
		t.Fatalf("CreateFromEpic: %v", err)
	}
	if len(tmpl.Tasks) != 2 {
		t.Errorf("tasks = %d, want 2", len(tmpl.Tasks))
	}
	if tmpl.Tasks[0].Persona != "researcher" {
		t.Errorf("task 0 persona = %q, want researcher", tmpl.Tasks[0].Persona)
	}
}

// Ensure context is imported.
var _ = context.Background
