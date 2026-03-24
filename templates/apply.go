package templates

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/e1sidy/slate"
)

// ApplyResult holds the outcome of applying a template.
type ApplyResult struct {
	EpicID  string   `json:"epic_id"`
	TaskIDs []string `json:"task_ids"`
}

// Apply creates a task tree from a template in Slate.
// Creates a parent epic, then child tasks with dependencies.
func Apply(ctx context.Context, store *slate.Store, tmpl *Template, vars map[string]string) (*ApplyResult, error) {
	if err := tmpl.ValidateVars(vars); err != nil {
		return nil, err
	}
	merged := tmpl.MergeVars(vars)

	// Render epic title.
	epicTitle := tmpl.Name
	if title, ok := merged["title"]; ok {
		epicTitle = title
	}

	// Create parent epic.
	epic, err := store.Create(ctx, slate.CreateParams{
		Title: epicTitle,
		Type:  slate.TypeEpic,
	})
	if err != nil {
		return nil, fmt.Errorf("create epic: %w", err)
	}

	result := &ApplyResult{EpicID: epic.ID}

	// Map template task ID → Slate task ID (for dependency linking).
	idMap := make(map[string]string)

	// Create tasks in order (deps reference earlier tasks by template ID).
	for _, tt := range tmpl.Tasks {
		title, err := RenderTitle(tt.TitleTemplate, merged)
		if err != nil {
			return nil, fmt.Errorf("render title for %s: %w", tt.ID, err)
		}

		taskType := slate.TaskType(tt.Type)
		if taskType == "" {
			taskType = slate.TypeTask
		}

		task, err := store.Create(ctx, slate.CreateParams{
			Title:    title,
			Type:     taskType,
			Priority: slate.Priority(tt.Priority),
			ParentID: epic.ID,
			Labels:   tt.Labels,
		})
		if err != nil {
			return nil, fmt.Errorf("create task %s: %w", tt.ID, err)
		}

		idMap[tt.ID] = task.ID
		result.TaskIDs = append(result.TaskIDs, task.ID)

		// Set orchestration attrs (persona, repos).
		if tt.Persona != "" {
			store.SetAttr(ctx, task.ID, "persona_used", tt.Persona)
		}
		if len(tt.Repos) > 0 {
			reposJSON, _ := json.Marshal(tt.Repos)
			store.SetAttr(ctx, task.ID, "repos", string(reposJSON))
		}

		// Add dependencies.
		for _, depID := range tt.Deps {
			slateDepID, ok := idMap[depID]
			if !ok {
				continue // dep not yet created (ordering issue in template)
			}
			// depID blocks this task: slateDepID → task.ID
			store.AddDependency(ctx, slateDepID, task.ID, slate.Blocks)
		}
	}

	return result, nil
}

// CreateFromEpic reverse-engineers a template from a completed epic.
func CreateFromEpic(ctx context.Context, store *slate.Store, epicID string) (*Template, error) {
	epic, err := store.Get(ctx, epicID)
	if err != nil {
		return nil, fmt.Errorf("get epic: %w", err)
	}

	children, err := store.Children(ctx, epicID)
	if err != nil {
		return nil, fmt.Errorf("get children: %w", err)
	}

	if len(children) == 0 {
		return nil, fmt.Errorf("epic %s has no children", epicID)
	}

	tmpl := &Template{
		Name:        fmt.Sprintf("from-%s", epicID),
		Description: fmt.Sprintf("Template derived from epic: %s", epic.Title),
		Vars: []TemplateVar{
			{Name: "title", Required: true},
		},
	}

	// Build ID map for dependency references.
	slateToTemplate := make(map[string]string) // Slate ID → template task ID
	for i, child := range children {
		templateID := fmt.Sprintf("task-%d", i+1)
		slateToTemplate[child.ID] = templateID
	}

	for _, child := range children {
		templateID := slateToTemplate[child.ID]

		// Get persona from attr.
		personaAttr, _ := store.GetAttr(ctx, child.ID, "persona_used")
		personaName := ""
		if personaAttr != nil {
			personaName = personaAttr.Value
		}

		// Get dependencies.
		deps, _ := store.ListDependents(ctx, child.ID)
		var depIDs []string
		for _, d := range deps {
			if tid, ok := slateToTemplate[d.FromID]; ok {
				depIDs = append(depIDs, tid)
			}
		}

		tt := TemplateTask{
			ID:            templateID,
			TitleTemplate: "<<.title>>: " + child.Title,
			Type:          string(child.Type),
			Priority:      int(child.Priority),
			Persona:       personaName,
			Deps:          depIDs,
			Labels:        child.Labels,
		}
		tmpl.Tasks = append(tmpl.Tasks, tt)
	}

	return tmpl, nil
}
