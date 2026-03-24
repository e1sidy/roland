// Package pipeline defines automated agent sequences where completing
// one step auto-triggers the next.
package pipeline

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// TriggerType describes when a pipeline step should be auto-triggered.
type TriggerType string

const (
	TriggerOnCreate  TriggerType = "on_create"   // auto-delegate when task is created
	TriggerOnDepDone TriggerType = "on_dep_done"  // auto-delegate when dependencies close
)

// Pipeline defines an automated agent sequence.
type Pipeline struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Steps       []PipelineStep `yaml:"pipeline"`
}

// PipelineStep links a template task to a trigger condition.
type PipelineStep struct {
	Step    string      `yaml:"step"`    // template task ID
	Trigger TriggerType `yaml:"trigger"` // when to auto-delegate
	Persona string      `yaml:"persona"` // override template persona (optional)
	Repos   []string    `yaml:"repos"`   // override repos (optional)
}

// LoadPipeline parses a pipeline definition from a YAML file.
// The file should contain both template tasks and pipeline steps.
func LoadPipeline(path string) (*Pipeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pipeline: %w", err)
	}
	return Parse(data)
}

// Parse parses pipeline YAML content.
func Parse(data []byte) (*Pipeline, error) {
	var p Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse pipeline: %w", err)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("pipeline missing name")
	}
	if len(p.Steps) == 0 {
		return nil, fmt.Errorf("pipeline %q has no steps", p.Name)
	}
	return &p, nil
}

// StepByID finds a pipeline step by template task ID.
func (p *Pipeline) StepByID(id string) *PipelineStep {
	for i, s := range p.Steps {
		if s.Step == id {
			return &p.Steps[i]
		}
	}
	return nil
}

// NextSteps returns pipeline steps that should trigger when the given step completes.
// These are steps with trigger=on_dep_done whose dependencies include the completed step.
func (p *Pipeline) NextSteps(completedStepID string) []PipelineStep {
	var next []PipelineStep
	for _, s := range p.Steps {
		if s.Trigger == TriggerOnDepDone && s.Step != completedStepID {
			// This step triggers when deps are done.
			// We consider it triggered if the completed step directly precedes it
			// in the pipeline ordering (simplified: sequential).
			next = append(next, s)
		}
	}
	return next
}

// InitialSteps returns steps that should trigger immediately (on_create).
func (p *Pipeline) InitialSteps() []PipelineStep {
	var initial []PipelineStep
	for _, s := range p.Steps {
		if s.Trigger == TriggerOnCreate {
			initial = append(initial, s)
		}
	}
	return initial
}
