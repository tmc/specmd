package openspec

import (
	"fmt"
	"strings"

	"github.com/tmc/specmd/validation"
)

const (
	minWhyLength              = 50
	minPurposeLength          = 50
	maxWhyLength              = 1000
	maxDeltasPerChange        = 10
	maxRequirementTextLength  = 500
	minDeltaDescriptionLength = 10
)

// ValidateSpec checks the required OpenSpec spec shape.
func ValidateSpec(spec *Spec) error {
	if spec == nil {
		return fmt.Errorf("validate spec: nil spec")
	}
	if err := ValidateSpecReport(spec).Err(); err != nil {
		return fmt.Errorf("validate spec: %w", err)
	}
	return nil
}

// ValidateSpecReport checks an OpenSpec spec and returns errors plus warnings.
func ValidateSpecReport(spec *Spec) validation.Report {
	if spec == nil {
		return validation.New([]validation.Issue{{Level: validation.LevelError, Path: "spec", Message: "cannot be nil"}})
	}
	var issues []validation.Issue
	if strings.TrimSpace(spec.Name) == "" {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: "name", Message: "cannot be empty"})
	}
	if strings.TrimSpace(spec.Overview) == "" {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: "overview", Message: "cannot be empty"})
	} else if len(spec.Overview) < minPurposeLength {
		issues = append(issues, validation.Issue{Level: validation.LevelWarning, Path: "overview", Message: fmt.Sprintf("purpose section is too brief (less than %d characters)", minPurposeLength)})
	}
	if len(spec.Requirements) == 0 {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: "requirements", Message: "must have at least one requirement"})
	}
	for i := range spec.Requirements {
		issues = append(issues, validateRequirementIssues(fmt.Sprintf("requirements[%d]", i), spec.Requirements[i])...)
	}
	return validation.New(issues)
}

// ValidateChange checks the required OpenSpec change shape.
func ValidateChange(change *Change) error {
	if change == nil {
		return fmt.Errorf("validate change: nil change")
	}
	if err := ValidateChangeReport(change).Err(); err != nil {
		return fmt.Errorf("validate change: %w", err)
	}
	return nil
}

// ValidateChangeReport checks an OpenSpec change and returns errors plus warnings.
func ValidateChangeReport(change *Change) validation.Report {
	if change == nil {
		return validation.New([]validation.Issue{{Level: validation.LevelError, Path: "change", Message: "cannot be nil"}})
	}
	var issues []validation.Issue
	if strings.TrimSpace(change.Name) == "" {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: "name", Message: "cannot be empty"})
	}
	n := len(strings.TrimSpace(change.Why))
	if n < minWhyLength {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: "why", Message: fmt.Sprintf("must be at least %d characters", minWhyLength)})
	}
	if n > maxWhyLength {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: "why", Message: fmt.Sprintf("must not exceed %d characters", maxWhyLength)})
	}
	if strings.TrimSpace(change.WhatChanges) == "" {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: "whatChanges", Message: "cannot be empty"})
	}
	if len(change.Deltas) == 0 {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: "deltas", Message: "must have at least one delta"})
	}
	if len(change.Deltas) > maxDeltasPerChange {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: "deltas", Message: fmt.Sprintf("must not exceed %d deltas", maxDeltasPerChange)})
	}
	for i := range change.Deltas {
		issues = append(issues, validateDeltaIssues(fmt.Sprintf("deltas[%d]", i), change.Deltas[i])...)
	}
	return validation.New(issues)
}

func validateDeltaIssues(path string, delta Delta) []validation.Issue {
	var issues []validation.Issue
	if strings.TrimSpace(delta.Spec) == "" {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: path + ".spec", Message: "cannot be empty"})
	}
	switch delta.Operation {
	case Added, Modified, Removed, Renamed:
	default:
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: path + ".operation", Message: "must be ADDED, MODIFIED, REMOVED, or RENAMED"})
	}
	if strings.TrimSpace(delta.Description) == "" {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: path + ".description", Message: "cannot be empty"})
	} else if len(delta.Description) < minDeltaDescriptionLength {
		issues = append(issues, validation.Issue{Level: validation.LevelWarning, Path: path + ".description", Message: "delta description is too brief"})
	}
	if (delta.Operation == Added || delta.Operation == Modified) && len(delta.Requirements) == 0 {
		issues = append(issues, validation.Issue{Level: validation.LevelWarning, Path: path + ".requirements", Message: string(delta.Operation) + " delta should include requirements"})
	}
	for i := range delta.Requirements {
		issues = append(issues, validateRequirementIssues(fmt.Sprintf("%s.requirements[%d]", path, i), delta.Requirements[i])...)
	}
	for i, rename := range delta.Renames {
		if strings.TrimSpace(rename.From) == "" {
			issues = append(issues, validation.Issue{Level: validation.LevelError, Path: fmt.Sprintf("%s.renames[%d].from", path, i), Message: "cannot be empty"})
		}
		if strings.TrimSpace(rename.To) == "" {
			issues = append(issues, validation.Issue{Level: validation.LevelError, Path: fmt.Sprintf("%s.renames[%d].to", path, i), Message: "cannot be empty"})
		}
	}
	return issues
}

func validateRequirementIssues(path string, req Requirement) []validation.Issue {
	var issues []validation.Issue
	if strings.TrimSpace(req.Text) == "" {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: path + ".text", Message: "cannot be empty"})
	}
	if !strings.Contains(req.Text, "SHALL") && !strings.Contains(req.Text, "MUST") {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: path + ".text", Message: "must contain SHALL or MUST keyword"})
	}
	if len(req.Text) > maxRequirementTextLength {
		issues = append(issues, validation.Issue{Level: validation.LevelInfo, Path: path, Message: fmt.Sprintf("requirement text is very long (>%d characters)", maxRequirementTextLength)})
	}
	if len(req.Scenarios) == 0 {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: path + ".scenarios", Message: "must have at least one scenario"})
	}
	for i, scenario := range req.Scenarios {
		if strings.TrimSpace(scenario.RawText) == "" {
			issues = append(issues, validation.Issue{Level: validation.LevelError, Path: fmt.Sprintf("%s.scenarios[%d]", path, i), Message: "cannot be empty"})
		}
	}
	return issues
}
