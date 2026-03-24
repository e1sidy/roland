package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/e1sidy/slate"

	"github.com/e1sidy/roland/learning"
	"github.com/spf13/cobra"
)

func learnCmd() *cobra.Command {
	var (
		since   string
		persona string
		show    string
		reset   string
		calibration bool
		calType string
	)

	cmd := &cobra.Command{
		Use:   "learn",
		Short: "Extract patterns from completed tasks to improve personas",
		Long:  "Analyzes checkpoints, blockers, close reasons, and file co-changes to find recurring patterns.\nPresents suggestions for user approval before enriching persona files.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if show != "" {
				return runLearnShow(show)
			}
			if reset != "" {
				return runLearnReset(reset)
			}
			if calibration {
				return runCalibration(cmd, since, calType, persona)
			}
			return runLearnAnalyze(cmd, since, persona)
		},
	}

	cmd.Flags().StringVar(&since, "since", "", "Only analyze tasks closed after this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&persona, "persona", "", "Filter by persona")
	cmd.Flags().StringVar(&show, "show", "", "Show current learnings for a persona")
	cmd.Flags().StringVar(&reset, "reset", "", "Reset learnings for a persona")
	cmd.Flags().BoolVar(&calibration, "calibration", false, "Show estimation calibration report")
	cmd.Flags().StringVar(&calType, "type", "", "Filter calibration by task type (with --calibration)")

	return cmd
}

func runLearnAnalyze(cmd *cobra.Command, since, persona string) error {
	store, err := openSlateStore(cfg)
	if err != nil {
		return err
	}
	defer store.Close()

	params := learning.AnalyzeParams{
		Persona:        persona,
		MinOccurrences: 3,
	}
	if since != "" {
		t, err := time.Parse("2006-01-02", since)
		if err != nil {
			return fmt.Errorf("invalid date %q: %w", since, err)
		}
		params.Since = &t
	}

	patterns, err := learning.Analyze(cmd.Context(), store, params)
	if err != nil {
		return err
	}

	if len(patterns) == 0 {
		fmt.Fprintln(os.Stderr, "No recurring patterns found. Need more completed tasks with checkpoints.")
		return nil
	}

	// Group by persona for interactive review.
	groups := learning.GroupByPersona(patterns)
	reader := bufio.NewReader(os.Stdin)

	for personaName, pats := range groups {
		fmt.Fprintf(os.Stderr, "\n=== Patterns for %s (%d found) ===\n", personaName, len(pats))

		var approved []learning.Pattern
		for _, p := range pats {
			fmt.Fprintf(os.Stderr, "\n  [%s] %q (found in %d tasks: %s)\n",
				p.Category, p.Text, p.Occurrences, strings.Join(p.TaskIDs, ", "))
			fmt.Fprint(os.Stderr, "  Apply to persona? [Y/n] ")

			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer == "" || answer == "y" || answer == "yes" {
				approved = append(approved, p)
				fmt.Fprintln(os.Stderr, "  ✓ Approved")
			} else {
				fmt.Fprintln(os.Stderr, "  ✗ Skipped")
			}
		}

		if len(approved) > 0 {
			if err := learning.EnrichPersona(cfg.Home, personaName, approved); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: failed to enrich %s: %v\n", personaName, err)
			} else {
				fmt.Fprintf(os.Stderr, "  Applied %d patterns to %s persona\n", len(approved), personaName)
			}
		}
	}

	return nil
}

func runLearnShow(persona string) error {
	content, err := learning.ShowLearnings(cfg.Home, persona)
	if err != nil {
		return err
	}
	if content == "" {
		fmt.Fprintf(os.Stderr, "No learnings for %s\n", persona)
		return nil
	}
	fmt.Println(content)
	return nil
}

func runLearnReset(persona string) error {
	if err := learning.ResetLearnings(cfg.Home, persona); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Reset learnings for %s\n", persona)
	return nil
}

func runCalibration(cmd *cobra.Command, since, calType, persona string) error {
	store, err := openSlateStore(cfg)
	if err != nil {
		return err
	}
	defer store.Close()

	params := learning.CalibrateParams{
		Persona: persona,
	}
	if since != "" {
		t, err := time.Parse("2006-01-02", since)
		if err != nil {
			return fmt.Errorf("invalid date %q: %w", since, err)
		}
		params.Since = &t
	}
	if calType != "" {
		t := slate.TaskType(calType)
		params.Type = &t
	}

	report, err := learning.Calibrate(cmd.Context(), store, params)
	if err != nil {
		return err
	}

	if report.Total == 0 {
		fmt.Fprintln(os.Stderr, "No tasks with estimates found. Set estimate on tasks with: slate update <id> --estimate <hours>")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Estimation Calibration (%d tasks):\n\n", report.Total)
	for _, e := range report.Entries {
		fmt.Fprintf(os.Stderr, "  %-20s  estimate: %s  actual: %s  ratio: %.1fx  (N=%d)\n",
			e.GroupBy, e.MedianEstimate.Round(time.Minute), e.MedianActual.Round(time.Minute), e.Ratio, e.SampleSize)
	}
	return nil
}

func decisionsCmd() *cobra.Command {
	var (
		search  string
		persona string
		taskID  string
		rebuild bool
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "decisions",
		Short: "Browse and search the decision library",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openSlateStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			if rebuild {
				entries, err := learning.IndexDecisions(cmd.Context(), store, learning.AnalyzeParams{})
				if err != nil {
					return err
				}
				if err := learning.SaveDecisionIndex(cfg.Home, entries); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "Rebuilt decision index: %d entries\n", len(entries))
				return nil
			}

			// Load from index (or build on-the-fly).
			entries, err := learning.LoadDecisionIndex(cfg.Home)
			if err != nil || entries == nil {
				entries, err = learning.IndexDecisions(cmd.Context(), store, learning.AnalyzeParams{})
				if err != nil {
					return err
				}
			}

			// Apply filters.
			if persona != "" {
				entries = learning.FilterByPersona(entries, persona)
			}
			if taskID != "" {
				entries = learning.FilterByTask(entries, taskID)
			}
			if search != "" {
				entries = learning.SearchDecisions(entries, search)
			}

			if jsonOut {
				data, err := json.MarshalIndent(entries, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			if len(entries) == 0 {
				fmt.Fprintln(os.Stderr, "No decisions found.")
				return nil
			}

			for _, e := range entries {
				fmt.Printf("%s | %s | %s | %s\n",
					e.Date.Format("2006-01-02"), e.TaskID, e.Persona, e.Text)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "Search decisions by keyword")
	cmd.Flags().StringVar(&persona, "persona", "", "Filter by persona")
	cmd.Flags().StringVar(&taskID, "task", "", "Filter by task ID")
	cmd.Flags().BoolVar(&rebuild, "rebuild", false, "Rebuild decision index from checkpoints")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}
