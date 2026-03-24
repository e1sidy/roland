package learning

import (
	"testing"

	"github.com/e1sidy/slate"
)

func TestExtractDecisions(t *testing.T) {
	cps := []*slate.Checkpoint{
		{TaskID: "t1", Decisions: "Use JWT over sessions\nAdd rate limiting"},
		{TaskID: "t2", Decisions: "Use JWT over sessions"},
		{TaskID: "t3", Decisions: "Use JWT over sessions\nPrefer PostgreSQL"},
	}
	personas := map[string]string{"t1": "builder", "t2": "builder", "t3": "builder"}

	patterns := ExtractDecisions(cps, personas)

	// "use jwt over sessions" appears in 3 tasks.
	found := false
	for _, p := range patterns {
		if p.Occurrences >= 3 && p.Category == "decision" {
			found = true
			if p.Persona != "builder" {
				t.Errorf("persona = %q, want builder", p.Persona)
			}
		}
	}
	if !found {
		t.Error("expected pattern with 3+ occurrences for JWT decision")
	}
}

func TestExtractBlockers(t *testing.T) {
	cps := []*slate.Checkpoint{
		{TaskID: "t1", Blockers: "Missing API docs"},
		{TaskID: "t2", Blockers: "Missing API docs"},
		{TaskID: "t3", Blockers: "Missing API docs\nCI flaky"},
	}
	personas := map[string]string{"t1": "builder", "t2": "builder", "t3": "builder"}

	patterns := ExtractBlockers(cps, personas)

	found := false
	for _, p := range patterns {
		if p.Occurrences >= 3 && p.Category == "blocker" {
			found = true
		}
	}
	if !found {
		t.Error("expected blocker pattern with 3+ occurrences")
	}
}

func TestExtractCoChanges(t *testing.T) {
	cps := []*slate.Checkpoint{
		{TaskID: "t1", Files: []string{"backend/auth.go", "frontend/login.tsx"}},
		{TaskID: "t2", Files: []string{"backend/auth.go", "frontend/login.tsx"}},
		{TaskID: "t3", Files: []string{"backend/auth.go", "frontend/login.tsx", "backend/auth_test.go"}},
	}
	personas := map[string]string{"t1": "builder", "t2": "builder", "t3": "builder"}

	patterns := ExtractCoChanges(cps, personas)

	// backend/auth.go + frontend/login.tsx should co-occur in 3 tasks.
	found := false
	for _, p := range patterns {
		if p.Occurrences >= 3 && p.Category == "co_change" {
			found = true
		}
	}
	if !found {
		t.Error("expected co-change pattern with 3+ occurrences")
	}
}

func TestExtractCoChanges_SameDir(t *testing.T) {
	cps := []*slate.Checkpoint{
		{TaskID: "t1", Files: []string{"backend/auth.go", "backend/auth_test.go"}},
		{TaskID: "t2", Files: []string{"backend/auth.go", "backend/auth_test.go"}},
		{TaskID: "t3", Files: []string{"backend/auth.go", "backend/auth_test.go"}},
	}

	patterns := ExtractCoChanges(cps, nil)

	// Same directory files should NOT generate co-change patterns.
	for _, p := range patterns {
		if p.Occurrences >= 3 {
			t.Errorf("same-dir files should not be co-change patterns: %q", p.Text)
		}
	}
}

func TestExtractCloseReasons(t *testing.T) {
	tasks := []*slate.Task{
		{ID: "t1", CloseReason: "completed implementation"},
		{ID: "t2", CloseReason: "completed implementation"},
		{ID: "t3", CloseReason: "completed implementation"},
		{ID: "t4", CloseReason: "descoped"},
	}
	personas := map[string]string{"t1": "builder", "t2": "builder", "t3": "builder", "t4": "builder"}

	patterns := ExtractCloseReasons(tasks, personas)

	found := false
	for _, p := range patterns {
		if p.Occurrences >= 3 && p.Category == "close_reason" {
			found = true
		}
	}
	if !found {
		t.Error("expected close reason pattern with 3+ occurrences")
	}
}

func TestFindRecurring(t *testing.T) {
	patterns := []Pattern{
		{Text: "a", Occurrences: 5},
		{Text: "b", Occurrences: 2},
		{Text: "c", Occurrences: 3},
		{Text: "d", Occurrences: 1},
	}

	result := FindRecurring(patterns, 3)
	if len(result) != 2 {
		t.Fatalf("got %d patterns, want 2", len(result))
	}
	// Should be sorted by occurrences desc.
	if result[0].Text != "a" {
		t.Errorf("first = %q, want a (5 occurrences)", result[0].Text)
	}
	if result[1].Text != "c" {
		t.Errorf("second = %q, want c (3 occurrences)", result[1].Text)
	}
}

func TestGroupByPersona(t *testing.T) {
	patterns := []Pattern{
		{Text: "a", Persona: "builder"},
		{Text: "b", Persona: "builder"},
		{Text: "c", Persona: "reviewer"},
		{Text: "d", Persona: ""},
	}

	groups := GroupByPersona(patterns)
	if len(groups["builder"]) != 2 {
		t.Errorf("builder = %d, want 2", len(groups["builder"]))
	}
	if len(groups["reviewer"]) != 1 {
		t.Errorf("reviewer = %d, want 1", len(groups["reviewer"]))
	}
	if len(groups["unknown"]) != 1 {
		t.Errorf("unknown = %d, want 1", len(groups["unknown"]))
	}
}

func TestNormalizeText(t *testing.T) {
	tt := []struct {
		input string
		want  string
	}{
		{"  Hello World!  ", "hello world"},
		{"- Use JWT over sessions.", "use jwt over sessions"},
		{"* Add rate limiting...", "add rate limiting"},
		{"", ""},
	}
	for _, tc := range tt {
		got := normalizeText(tc.input)
		if got != tc.want {
			t.Errorf("normalizeText(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
