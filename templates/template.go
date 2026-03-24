// Package templates provides task template definitions and application.
package templates

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

//go:embed builtin/*.yaml
var builtinFS embed.FS

// Template defines a reusable task structure.
type Template struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Vars        []TemplateVar  `yaml:"vars"`
	Tasks       []TemplateTask `yaml:"tasks"`
}

// TemplateVar defines a template variable.
type TemplateVar struct {
	Name     string `yaml:"name"`
	Required bool   `yaml:"required"`
	Default  string `yaml:"default"`
}

// TemplateTask defines a single task within a template.
type TemplateTask struct {
	ID            string   `yaml:"id"`
	TitleTemplate string   `yaml:"title"`
	Type          string   `yaml:"type"`
	Priority      int      `yaml:"priority"`
	Persona       string   `yaml:"persona"`
	Repos         []string `yaml:"repos"`
	Deps          []string `yaml:"deps"`
	Labels        []string `yaml:"labels"`
}

// Load parses a template from a YAML file.
func Load(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}
	return parse(data)
}

// parse parses template YAML content.
func parse(data []byte) (*Template, error) {
	var tmpl Template
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	if tmpl.Name == "" {
		return nil, fmt.Errorf("template missing name")
	}
	if len(tmpl.Tasks) == 0 {
		return nil, fmt.Errorf("template %q has no tasks", tmpl.Name)
	}
	return &tmpl, nil
}

// List returns all available templates (built-in + custom).
func List(home string) ([]*Template, error) {
	var templates []*Template

	// Load built-in templates.
	entries, err := builtinFS.ReadDir("builtin")
	if err == nil {
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			data, err := builtinFS.ReadFile("builtin/" + e.Name())
			if err != nil {
				continue
			}
			tmpl, err := parse(data)
			if err != nil {
				continue
			}
			templates = append(templates, tmpl)
		}
	}

	// Load custom templates from ~/.roland/templates/.
	customDir := filepath.Join(home, "templates")
	customEntries, err := os.ReadDir(customDir)
	if err == nil {
		for _, e := range customEntries {
			if !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			tmpl, err := Load(filepath.Join(customDir, e.Name()))
			if err != nil {
				continue
			}
			templates = append(templates, tmpl)
		}
	}

	return templates, nil
}

// Get finds a template by name from built-in or custom templates.
func Get(home, name string) (*Template, error) {
	// Check custom first.
	customPath := filepath.Join(home, "templates", name+".yaml")
	if tmpl, err := Load(customPath); err == nil {
		return tmpl, nil
	}

	// Check built-in.
	data, err := builtinFS.ReadFile("builtin/" + name + ".yaml")
	if err != nil {
		return nil, fmt.Errorf("template %q not found", name)
	}
	return parse(data)
}

// RenderTitle renders a template task title with variable substitution.
// Uses custom delimiters <<.var>> to avoid markdown {{ }} conflicts.
func RenderTitle(titleTemplate string, vars map[string]string) (string, error) {
	tmpl, err := template.New("title").Delims("<<", ">>").Parse(titleTemplate)
	if err != nil {
		return "", fmt.Errorf("parse title template: %w", err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("render title: %w", err)
	}
	return buf.String(), nil
}

// ValidateVars checks that all required variables are provided.
func (t *Template) ValidateVars(vars map[string]string) error {
	for _, v := range t.Vars {
		if v.Required {
			if _, ok := vars[v.Name]; !ok {
				if v.Default == "" {
					return fmt.Errorf("required variable %q not provided", v.Name)
				}
			}
		}
	}
	return nil
}

// MergeVars fills in default values for missing variables.
func (t *Template) MergeVars(vars map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range vars {
		merged[k] = v
	}
	for _, v := range t.Vars {
		if _, ok := merged[v.Name]; !ok && v.Default != "" {
			merged[v.Name] = v.Default
		}
	}
	return merged
}
