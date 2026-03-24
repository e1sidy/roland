package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/e1sidy/roland/internal/testutil"
	"github.com/e1sidy/slate"
)

func TestParse(t *testing.T) {
	yaml := []byte(`
name: test-pipeline
pipeline:
  - step: research
    trigger: on_create
    persona: researcher
  - step: implement
    trigger: on_dep_done
    persona: builder
  - step: review
    trigger: on_dep_done
    persona: reviewer
`)
	p, err := Parse(yaml)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.Name != "test-pipeline" {
		t.Errorf("name = %q", p.Name)
	}
	if len(p.Steps) != 3 {
		t.Errorf("steps = %d, want 3", len(p.Steps))
	}
}

func TestParse_MissingName(t *testing.T) {
	_, err := Parse([]byte("pipeline:\n  - step: a\n    trigger: on_create\n"))
	if err == nil {
		t.Error("should error without name")
	}
}

func TestParse_NoSteps(t *testing.T) {
	_, err := Parse([]byte("name: empty\npipeline: []\n"))
	if err == nil {
		t.Error("should error without steps")
	}
}

func TestInitialSteps(t *testing.T) {
	p := &Pipeline{
		Steps: []PipelineStep{
			{Step: "a", Trigger: TriggerOnCreate},
			{Step: "b", Trigger: TriggerOnDepDone},
			{Step: "c", Trigger: TriggerOnCreate},
		},
	}
	initial := p.InitialSteps()
	if len(initial) != 2 {
		t.Errorf("initial = %d, want 2", len(initial))
	}
}

func TestStepByID(t *testing.T) {
	p := &Pipeline{
		Steps: []PipelineStep{
			{Step: "a", Persona: "researcher"},
			{Step: "b", Persona: "builder"},
		},
	}
	s := p.StepByID("b")
	if s == nil || s.Persona != "builder" {
		t.Error("should find step b with builder persona")
	}
	if p.StepByID("nonexistent") != nil {
		t.Error("nonexistent should return nil")
	}
}

func TestRun_BasicSequence(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)

	// Create parent + children.
	parent, _ := store.Create(ctx, slate.CreateParams{Title: "Epic"})
	child1, _ := store.Create(ctx, slate.CreateParams{Title: "Research", ParentID: parent.ID})
	child2, _ := store.Create(ctx, slate.CreateParams{Title: "Implement", ParentID: parent.ID})
	store.AddDependency(ctx, child1.ID, child2.ID, slate.Blocks)

	p := &Pipeline{
		Name: "test",
		Steps: []PipelineStep{
			{Step: "research", Trigger: TriggerOnCreate, Persona: "researcher"},
			{Step: "implement", Trigger: TriggerOnDepDone, Persona: "builder"},
		},
	}
	taskMap := map[string]string{
		"research":  child1.ID,
		"implement": child2.ID,
	}

	delegated := []string{}
	delegateFn := func(ctx context.Context, taskID, persona string, repos []string) error {
		delegated = append(delegated, taskID+":"+persona)
		return nil
	}

	// Run with very short interval — but child1 won't close by itself.
	// Close child1 manually to simulate agent completion.
	go func() {
		time.Sleep(200 * time.Millisecond)
		store.CloseTask(ctx, child1.ID, "done", "agent")
		time.Sleep(200 * time.Millisecond)
		store.CloseTask(ctx, child2.ID, "done", "agent")
	}()

	runCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	result, err := Run(runCtx, store, p, parent.ID, taskMap, 100*time.Millisecond, delegateFn)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Should have delegated research immediately (on_create).
	if result.StepsDelegated < 1 {
		t.Errorf("delegated = %d, want >= 1", result.StepsDelegated)
	}
	if len(delegated) < 1 {
		t.Error("should have delegated at least research")
	}
}
