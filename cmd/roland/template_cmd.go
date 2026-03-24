package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/e1sidy/roland/templates"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func templateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage task templates",
	}
	cmd.AddCommand(
		templateListCmd(),
		templateShowCmd(),
		templateApplyCmd(),
		templateCreateCmd(),
		templateDecomposeCmd(),
	)
	return cmd
}

func templateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			tmpls, err := templates.List(cfg.Home)
			if err != nil {
				return err
			}
			if len(tmpls) == 0 {
				fmt.Fprintln(os.Stderr, "No templates found.")
				return nil
			}
			for _, t := range tmpls {
				fmt.Printf("%-20s  %s (%d tasks)\n", t.Name, t.Description, len(t.Tasks))
			}
			return nil
		},
	}
}

func templateShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show template structure",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tmpl, err := templates.Get(cfg.Home, args[0])
			if err != nil {
				return err
			}
			data, err := yaml.Marshal(tmpl)
			if err != nil {
				return err
			}
			fmt.Print(string(data))
			return nil
		},
	}
}

func templateApplyCmd() *cobra.Command {
	var varFlags []string

	cmd := &cobra.Command{
		Use:   "apply <name>",
		Short: "Apply a template to create a task tree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tmpl, err := templates.Get(cfg.Home, args[0])
			if err != nil {
				return err
			}

			// Parse --var flags.
			vars := make(map[string]string)
			for _, v := range varFlags {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --var format %q (expected key=value)", v)
				}
				vars[parts[0]] = parts[1]
			}

			store, err := openSlateStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			result, err := templates.Apply(cmd.Context(), store, tmpl, vars)
			if err != nil {
				return err
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))

			fmt.Fprintf(os.Stderr, "Created epic %s with %d tasks\n", result.EpicID, len(result.TaskIDs))
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&varFlags, "var", nil, "Template variable (format: key=value, repeatable)")
	return cmd
}

func templateCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create --from <epic-id>",
		Short: "Create a template from a completed epic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openSlateStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			tmpl, err := templates.CreateFromEpic(cmd.Context(), store, args[0])
			if err != nil {
				return err
			}

			data, err := yaml.Marshal(tmpl)
			if err != nil {
				return err
			}

			fmt.Print(string(data))
			fmt.Fprintf(os.Stderr, "\nTemplate %q with %d tasks. Save to ~/.roland/templates/%s.yaml to reuse.\n",
				tmpl.Name, len(tmpl.Tasks), tmpl.Name)
			return nil
		},
	}
}

func templateDecomposeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "decompose <task-id>",
		Short: "Suggest subtask structure based on similar completed epics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openSlateStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			tmpl, err := templates.Decompose(cmd.Context(), store, args[0])
			if err != nil {
				return err
			}

			data, err := yaml.Marshal(tmpl)
			if err != nil {
				return err
			}

			fmt.Print(string(data))
			fmt.Fprintf(os.Stderr, "\nSuggested structure: %d tasks. Apply with: roland template apply <name>\n", len(tmpl.Tasks))
			return nil
		},
	}
}
