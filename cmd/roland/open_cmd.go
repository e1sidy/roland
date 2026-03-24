package main

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

func openCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open [task-id]",
		Short: "Open a task workspace in the configured IDE",
		Long:  "Opens the task workspace directory in the configured IDE (VS Code, Cursor, etc.).",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, taskDir, err := resolveTaskID(cfg.Home, args)
			if err != nil {
				return fmt.Errorf("resolve task: %w", err)
			}

			ideCmd := cfg.IDE.Command()
			if ideCmd == "" {
				return fmt.Errorf("no IDE configured; use 'roland config ide <name>' to set one")
			}

			binary, err := exec.LookPath(ideCmd)
			if err != nil {
				return fmt.Errorf("IDE command %q not found in PATH: %w", ideCmd, err)
			}

			openProc := exec.Command(binary, taskDir)
			if err := openProc.Start(); err != nil {
				return fmt.Errorf("open IDE: %w", err)
			}

			fmt.Printf("Opened %s in %s\n", taskDir, cfg.IDE)
			return nil
		},
	}

	return cmd
}
