package learning

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/e1sidy/slate"
)

// CalibrateParams controls the calibration scope.
type CalibrateParams struct {
	Since   *time.Time     // only tasks closed after this date
	Type    *slate.TaskType // filter by task type
	Persona string          // filter by persona_used attr
}

// CalibrationReport contains estimation accuracy data.
type CalibrationReport struct {
	Entries []CalibrationEntry `json:"entries"`
	Total   int                `json:"total"` // total tasks analyzed
}

// CalibrationEntry represents calibration data for one group.
type CalibrationEntry struct {
	GroupBy        string        `json:"group_by"`        // "type:bug", "persona:builder"
	MedianEstimate time.Duration `json:"median_estimate"` // from task.estimate field (hours)
	MedianActual   time.Duration `json:"median_actual"`   // from events timestamps
	Ratio          float64       `json:"ratio"`           // actual / estimate
	SampleSize     int           `json:"sample_size"`
}

// calibrationData holds a single task's estimation data.
type calibrationData struct {
	task     *slate.Task
	estimate time.Duration
	actual   time.Duration
	persona  string
}

// Calibrate compares task estimates vs actual cycle times.
func Calibrate(ctx context.Context, store *slate.Store, params CalibrateParams) (*CalibrationReport, error) {
	closedStatus := slate.StatusClosed
	tasks, err := store.List(ctx, slate.ListParams{Status: &closedStatus})
	if err != nil {
		return nil, fmt.Errorf("list closed tasks: %w", err)
	}

	var data []calibrationData
	for _, t := range tasks {
		// Date filter.
		if params.Since != nil && t.ClosedAt != nil && t.ClosedAt.Before(*params.Since) {
			continue
		}
		// Type filter.
		if params.Type != nil && t.Type != *params.Type {
			continue
		}
		// Persona filter.
		personaUsed := getPersonaAttr(ctx, store, t.ID)
		if params.Persona != "" && personaUsed != params.Persona {
			continue
		}
		// Need estimate > 0 and valid close time.
		if t.Estimate <= 0 || t.ClosedAt == nil {
			continue
		}

		estimate := time.Duration(t.Estimate) * time.Hour
		actual := t.ClosedAt.Sub(t.CreatedAt)

		data = append(data, calibrationData{
			task:     t,
			estimate: estimate,
			actual:   actual,
			persona:  personaUsed,
		})
	}

	if len(data) == 0 {
		return &CalibrationReport{Total: 0}, nil
	}

	report := &CalibrationReport{Total: len(data)}

	// Group by type.
	byType := make(map[slate.TaskType][]calibrationData)
	for _, d := range data {
		byType[d.task.Type] = append(byType[d.task.Type], d)
	}
	for tp, group := range byType {
		entry := buildEntry("type:"+string(tp), group)
		report.Entries = append(report.Entries, entry)
	}

	// Group by persona.
	byPersona := make(map[string][]calibrationData)
	for _, d := range data {
		if d.persona != "" {
			byPersona[d.persona] = append(byPersona[d.persona], d)
		}
	}
	for p, group := range byPersona {
		entry := buildEntry("persona:"+p, group)
		report.Entries = append(report.Entries, entry)
	}

	// Sort entries by group name.
	sort.Slice(report.Entries, func(i, j int) bool {
		return report.Entries[i].GroupBy < report.Entries[j].GroupBy
	})

	return report, nil
}

func buildEntry(groupBy string, data []calibrationData) CalibrationEntry {
	var estimates, actuals []time.Duration
	for _, d := range data {
		estimates = append(estimates, d.estimate)
		actuals = append(actuals, d.actual)
	}

	medEstimate := median(estimates)
	medActual := median(actuals)

	var ratio float64
	if medEstimate > 0 {
		ratio = float64(medActual) / float64(medEstimate)
	}

	return CalibrationEntry{
		GroupBy:        groupBy,
		MedianEstimate: medEstimate,
		MedianActual:   medActual,
		Ratio:          ratio,
		SampleSize:     len(data),
	}
}

func median(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	mid := len(durations) / 2
	if len(durations)%2 == 0 {
		return (durations[mid-1] + durations[mid]) / 2
	}
	return durations[mid]
}
