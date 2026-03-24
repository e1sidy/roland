// Package main is the CLI entry point for Roland.
package main

import (
	"fmt"
	"os"

	"github.com/e1sidy/roland"
	"github.com/spf13/cobra"
)

// cfg holds the loaded config, set in PersistentPreRunE.
var cfg *roland.Config

func main() {
	root := rootCmd()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "roland",
		Short: "Workspace orchestration for AI coding agents",
		Long: `Roland manages the full lifecycle of agent-driven development:
  pickup → work → checkpoint → ship → done

It integrates with Slate for task management and provides personas,
hooks, skills, and workspace isolation via git worktrees.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip config loading for commands that don't need it.
			name := cmd.Name()
			if name == "init" || name == "completion" || name == "help" || name == "version" {
				return nil
			}
			// Also skip for root command (just prints help).
			if cmd.Parent() == nil {
				return nil
			}

			home, err := roland.ResolveHome()
			if err != nil {
				return fmt.Errorf("resolve home: %w", err)
			}

			c, err := roland.LoadConfig(home)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			cfg = c
			return nil
		},
	}

	// Register subcommands.
	cmd.AddCommand(
		versionCmd(),
		completionCmd(),
		initCmd(),
		configCmd(),
		repoCmd(),
		worktreeCmd(),
		personaCmd(),
		skillCmd(),
		hookCmd(),
		pickupCmd(),
		workCmd(),
		checkpointCmd(),
		shipCmd(),
		doneCmd(),
		statusCmd(),
		cleanCmd(),
		openCmd(),
		learnCmd(),
		decisionsCmd(),
		delegateCmd(),
		watchCmd(),
		handoffCmd(),
		templateCmd(),
	)

	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Roland version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("roland " + roland.Version)
		},
	}
}
