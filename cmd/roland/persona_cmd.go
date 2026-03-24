package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/e1sidy/roland/persona"
	"github.com/spf13/cobra"
)

func personaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "persona",
		Short: "Manage agent personas",
	}

	cmd.AddCommand(
		personaListCmd(),
		personaShowCmd(),
		personaCreateCmd(),
		personaEditCmd(),
		personaDeleteCmd(),
	)

	return cmd
}

func personaListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available personas",
		RunE: func(cmd *cobra.Command, args []string) error {
			personas, err := persona.ListAll(cfg.Home)
			if err != nil {
				return fmt.Errorf("list personas: %w", err)
			}
			if len(personas) == 0 {
				fmt.Println("No personas available.")
				return nil
			}
			for _, p := range personas {
				source := colorize(colorGray, p.Source)
				fmt.Printf("  %-20s %s\n", p.Name, source)
			}
			return nil
		},
	}
}

func personaShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Display a persona's content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := persona.Get(cfg.Home, args[0])
			if err != nil {
				return fmt.Errorf("get persona: %w", err)
			}
			fmt.Println(content)
			return nil
		},
	}
}

func personaCreateCmd() *cobra.Command {
	var fromBase string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new custom persona",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := persona.Create(cfg.Home, args[0], fromBase); err != nil {
				return fmt.Errorf("create persona: %w", err)
			}
			fmt.Printf("Created persona %q\n", args[0])
			if fromBase != "" {
				fmt.Printf("Based on %q\n", fromBase)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&fromBase, "from", "", "Base persona to copy from")

	return cmd
}

func personaEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <name>",
		Short: "Open a persona for editing in $EDITOR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := persona.Edit(cfg.Home, args[0])
			if err != nil {
				return fmt.Errorf("edit persona: %w", err)
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}

			editorCmd := exec.Command(editor, path)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr
			return editorCmd.Run()
		},
	}
}

func personaDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a custom persona",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := persona.Delete(cfg.Home, args[0]); err != nil {
				return fmt.Errorf("delete persona: %w", err)
			}
			fmt.Printf("Deleted persona %q\n", args[0])
			return nil
		},
	}
}
