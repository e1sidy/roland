package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/e1sidy/roland"
	"github.com/e1sidy/roland/pipeline"
	"github.com/e1sidy/roland/templates"
	"github.com/e1sidy/roland/workspace"
	"github.com/spf13/cobra"
)

func pipelineCmd() *cobra.Command {
	var (
		varFlags []string
		interval string
	)

	cmd := &cobra.Command{
		Use:   "pipeline <template-name>",
		Short: "Apply a template and auto-run the agent sequence",
		Long: `Applies a task template, then automatically delegates tasks
to personas in sequence based on pipeline trigger rules.

The template must include a 'pipeline:' section defining trigger
conditions (on_create, on_dep_done) for each step.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateName := args[0]

			dur, err := time.ParseDuration(interval)
			if err != nil {
				return fmt.Errorf("invalid interval: %w", err)
			}

			store, err := openSlateStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			if err := roland.EnsureAttrs(cmd.Context(), store); err != nil {
				return err
			}

			// Load template.
			tmpl, err := templates.Get(cfg.Home, templateName)
			if err != nil {
				return fmt.Errorf("load template: %w", err)
			}

			// Parse vars.
			vars := make(map[string]string)
			for _, v := range varFlags {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) == 2 {
					vars[parts[0]] = parts[1]
				}
			}

			// Apply template to create tasks.
			result, err := templates.Apply(cmd.Context(), store, tmpl, vars)
			if err != nil {
				return fmt.Errorf("apply template: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Created epic %s with %d tasks\n", result.EpicID, len(result.TaskIDs))

			// Build task map: template task ID → Slate task ID.
			taskMap := make(map[string]string)
			for i, tt := range tmpl.Tasks {
				if i < len(result.TaskIDs) {
					taskMap[tt.ID] = result.TaskIDs[i]
				}
			}

			// Load pipeline config from the template.
			// Try reading the raw YAML for pipeline section.
			pipelineCfg, err := loadPipelineFromTemplate(templateName)
			if err != nil || pipelineCfg == nil {
				fmt.Fprintln(os.Stderr, "No pipeline section in template. Tasks created but not auto-delegated.")
				return nil
			}

			// Set up signal handling.
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			// Run the pipeline.
			fmt.Fprintf(os.Stderr, "\nStarting pipeline (interval: %s, Ctrl+C to stop)...\n\n", dur)

			delegateFn := func(ctx context.Context, taskID, persona string, repos []string) error {
				if len(repos) == 0 {
					for name := range cfg.Repos {
						repos = append(repos, name)
					}
				}
				_, err := setupWorkspace(ctx, setupOpts{
					Store:       store,
					Cfg:         cfg,
					TaskID:      taskID,
					PersonaName: persona,
					RepoNames:   repos,
				})
				if err != nil {
					return err
				}
				flags := cfg.AgentFlags[cfg.Agent]
				td, _ := workspace.Open(cfg.Home, taskID)
				if td != nil {
					pid, lerr := launchAgentBackground(td.Path, cfg.Agent, flags)
					if lerr != nil {
						return lerr
					}
					fmt.Fprintf(os.Stderr, "  Agent launched (PID %d)\n", pid)
				}
				return nil
			}

			pipelineResult, err := pipeline.Run(ctx, store, pipelineCfg, result.EpicID, taskMap, dur, delegateFn)
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "\nPipeline complete: %d delegated, %d completed\n",
				pipelineResult.StepsDelegated, pipelineResult.StepsCompleted)
			for _, e := range pipelineResult.Errors {
				fmt.Fprintf(os.Stderr, "  Error: %s\n", e)
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&varFlags, "var", nil, "Template variable (key=value)")
	cmd.Flags().StringVar(&interval, "interval", "30s", "Polling interval for pipeline progress")
	return cmd
}

// loadPipelineFromTemplate tries to load pipeline config from a template file.
func loadPipelineFromTemplate(name string) (*pipeline.Pipeline, error) {
	// Try custom templates directory first.
	customPath := cfg.Home + "/templates/" + name + ".yaml"
	if p, err := pipeline.LoadPipeline(customPath); err == nil {
		return p, nil
	}
	// No pipeline section found.
	return nil, nil
}
