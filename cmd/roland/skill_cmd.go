package main

import (
	"fmt"
	"strings"

	"github.com/e1sidy/roland/skill"
	"github.com/spf13/cobra"
)

func skillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage agent skills",
	}

	cmd.AddCommand(
		skillAddCmd(),
		skillListCmd(),
		skillTagCmd(),
		skillInjectCmd(),
		skillEjectCmd(),
	)

	return cmd
}

func skillAddCmd() *cobra.Command {
	var (
		name     string
		external bool
	)

	cmd := &cobra.Command{
		Use:   "add <path>",
		Short: "Register a skill from a directory",
		Long: `Registers a skill directory. The directory must contain a SKILL.md file.

By default, the skill is copied into ROLAND_HOME/.skills/. Use --external
to keep it at its current location (symlinked).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, err := skill.Add(cfg.Home, args[0], name, external)
			if err != nil {
				return fmt.Errorf("add skill: %w", err)
			}
			fmt.Printf("Added skill %q at %s\n", name, entry.Location)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Skill name (required)")
	cmd.Flags().BoolVar(&external, "external", false, "Keep skill at its current location instead of copying")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func skillListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			skills, err := skill.List(cfg.Home)
			if err != nil {
				return fmt.Errorf("list skills: %w", err)
			}
			if len(skills) == 0 {
				fmt.Println("No skills registered. Use 'roland skill add <path> --name <name>' to add one.")
				return nil
			}
			for _, s := range skills {
				tags := []string{}
				if len(s.Entry.Personas) > 0 {
					tags = append(tags, "personas:"+strings.Join(s.Entry.Personas, ","))
				}
				if len(s.Entry.TaskTypes) > 0 {
					tags = append(tags, "types:"+strings.Join(s.Entry.TaskTypes, ","))
				}
				if len(s.Entry.Tags) > 0 {
					tags = append(tags, "tags:"+strings.Join(s.Entry.Tags, ","))
				}
				tagStr := ""
				if len(tags) > 0 {
					tagStr = " [" + strings.Join(tags, " ") + "]"
				}
				fmt.Printf("  %-20s %s%s\n", s.Name, s.Entry.Location, tagStr)
			}
			return nil
		},
	}
}

func skillTagCmd() *cobra.Command {
	var (
		personas  string
		taskTypes string
		tags      string
	)

	cmd := &cobra.Command{
		Use:   "tag <skill-name>",
		Short: "Set matching criteria for a skill",
		Long: `Updates the auto-injection criteria for a skill.

Example:
  roland skill tag my-skill --personas builder,researcher --types feature,bug`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var pList, tList, tagList []string
			if personas != "" {
				pList = strings.Split(personas, ",")
			}
			if taskTypes != "" {
				tList = strings.Split(taskTypes, ",")
			}
			if tags != "" {
				tagList = strings.Split(tags, ",")
			}

			if err := skill.SetTags(cfg.Home, args[0], pList, tList, tagList); err != nil {
				return fmt.Errorf("set tags: %w", err)
			}
			fmt.Printf("Updated tags for skill %q\n", args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&personas, "personas", "", "Comma-separated persona names")
	cmd.Flags().StringVar(&taskTypes, "types", "", "Comma-separated task types")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags/labels")

	return cmd
}

func skillInjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inject <skill-name> [task-id]",
		Short: "Inject a skill into a task workspace",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillName := args[0]
			taskArgs := args[1:]

			_, taskDir, err := resolveTaskID(cfg.Home, taskArgs)
			if err != nil {
				return fmt.Errorf("resolve task: %w", err)
			}

			entry, err := skill.Get(cfg.Home, skillName)
			if err != nil {
				return fmt.Errorf("get skill: %w", err)
			}

			if err := skill.Inject(skillName, entry.Location, taskDir); err != nil {
				return fmt.Errorf("inject skill: %w", err)
			}
			fmt.Printf("Injected skill %q into %s\n", skillName, taskDir)
			return nil
		},
	}
}

func skillEjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "eject <skill-name> [task-id]",
		Short: "Remove a skill from a task workspace",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillName := args[0]
			taskArgs := args[1:]

			_, taskDir, err := resolveTaskID(cfg.Home, taskArgs)
			if err != nil {
				return fmt.Errorf("resolve task: %w", err)
			}

			if err := skill.Eject(skillName, taskDir); err != nil {
				return fmt.Errorf("eject skill: %w", err)
			}
			fmt.Printf("Ejected skill %q from %s\n", skillName, taskDir)
			return nil
		},
	}
}
