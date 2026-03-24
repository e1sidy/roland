package pipeline

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/e1sidy/slate"
)

// RunResult summarizes the pipeline execution.
type RunResult struct {
	StepsCompleted int      `json:"steps_completed"`
	StepsDelegated int      `json:"steps_delegated"`
	Errors         []string `json:"errors,omitempty"`
	AllDone        bool     `json:"all_done"`
}

// DelegateFunc is called to delegate a task to a persona.
// Implementations should create workspace and launch agent in background.
type DelegateFunc func(ctx context.Context, taskID, persona string, repos []string) error

// Run executes a pipeline: monitors child tasks and auto-delegates
// next steps when trigger conditions are met.
func Run(ctx context.Context, store *slate.Store, pipeline *Pipeline, parentID string, taskMap map[string]string, interval time.Duration, delegate DelegateFunc) (*RunResult, error) {
	result := &RunResult{}

	// Track which steps have been delegated to avoid re-delegation.
	delegated := make(map[string]bool)

	// Delegate initial steps (on_create triggers).
	for _, step := range pipeline.InitialSteps() {
		taskID, ok := taskMap[step.Step]
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("step %s: no task ID in map", step.Step))
			continue
		}
		persona := step.Persona
		if persona == "" {
			persona = "builder"
		}
		if err := delegate(ctx, taskID, persona, step.Repos); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("step %s: delegate: %v", step.Step, err))
			continue
		}
		delegated[step.Step] = true
		result.StepsDelegated++
		fmt.Fprintf(os.Stderr, "Pipeline: delegated %s (%s) to %s\n", step.Step, taskID, persona)
	}

	// Poll loop: watch for step completions and trigger next steps.
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	maxIterations := 1000 // safety limit
	for i := 0; i < maxIterations; i++ {
		select {
		case <-ctx.Done():
			return result, nil
		case <-ticker.C:
			allDone, err := pollPipeline(ctx, store, pipeline, parentID, taskMap, delegated, delegate, result)
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
			}
			if allDone {
				result.AllDone = true
				fmt.Fprintf(os.Stderr, "Pipeline: all steps complete\n")
				return result, nil
			}
		}
	}

	result.Errors = append(result.Errors, "max iterations reached")
	return result, nil
}

func pollPipeline(ctx context.Context, store *slate.Store, p *Pipeline, parentID string, taskMap map[string]string, delegated map[string]bool, delegate DelegateFunc, result *RunResult) (bool, error) {
	children, err := store.Children(ctx, parentID)
	if err != nil {
		return false, fmt.Errorf("get children: %w", err)
	}

	// Build status map.
	childStatus := make(map[string]slate.Status)
	for _, c := range children {
		childStatus[c.ID] = c.Status
	}

	allTerminal := true

	for _, step := range p.Steps {
		taskID, ok := taskMap[step.Step]
		if !ok {
			continue
		}

		status, exists := childStatus[taskID]
		if !exists {
			continue
		}

		if !status.IsTerminal() {
			allTerminal = false
		}

		// Check if this step completed and should trigger next steps.
		if status.IsTerminal() && !delegated[step.Step+"_completed"] {
			delegated[step.Step+"_completed"] = true
			result.StepsCompleted++
			fmt.Fprintf(os.Stderr, "Pipeline: step %s completed (%s)\n", step.Step, status)

			// Find and delegate next steps.
			for _, nextStep := range p.Steps {
				if nextStep.Trigger != TriggerOnDepDone {
					continue
				}
				nextTaskID, ok := taskMap[nextStep.Step]
				if !ok || delegated[nextStep.Step] {
					continue
				}

				// Check if ALL deps of this next step are done.
				// For simplicity: a step with on_dep_done triggers when
				// the immediately preceding step (by index) completes.
				canDelegate := true
				for _, c := range children {
					if c.ID == nextTaskID {
						continue
					}
					// Check deps of this task.
					deps, _ := store.ListDependents(ctx, nextTaskID)
					for _, d := range deps {
						depStatus := childStatus[d.FromID]
						if !depStatus.IsTerminal() {
							canDelegate = false
							break
						}
					}
				}

				if canDelegate {
					persona := nextStep.Persona
					if persona == "" {
						persona = "builder"
					}
					if err := delegate(ctx, nextTaskID, persona, nextStep.Repos); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("step %s: delegate: %v", nextStep.Step, err))
						continue
					}
					delegated[nextStep.Step] = true
					result.StepsDelegated++
					fmt.Fprintf(os.Stderr, "Pipeline: delegated %s (%s) to %s\n", nextStep.Step, nextTaskID, persona)
				}
			}
		}
	}

	return allTerminal && len(children) > 0, nil
}
