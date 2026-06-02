package openspec

import (
	"errors"
	"fmt"
	"strings"
)

const (
	minWhyLength              = 50
	minPurposeLength          = 50
	maxWhyLength              = 1000
	maxDeltasPerChange        = 10
	maxRequirementTextLength  = 500
	minDeltaDescriptionLength = 10
)

// ValidationLevel is the severity of one validation issue.
type ValidationLevel string

const (
	Error   ValidationLevel = "ERROR"
	Warning ValidationLevel = "WARNING"
	Info    ValidationLevel = "INFO"
)

// ValidationIssue reports one validation problem or advisory.
type ValidationIssue struct {
	Level   ValidationLevel
	Path    string
	Message string
}

// ValidationSummary counts validation issues by severity.
type ValidationSummary struct {
	Errors   int
	Warnings int
	Info     int
}

// ValidationReport reports hard errors and advisory warnings for an artifact.
type ValidationReport struct {
	Valid   bool
	Issues  []ValidationIssue
	Summary ValidationSummary
}

// ValidationError reports one invalid field in a spec or change.
type ValidationError struct {
	Field string
	Err   error
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Err.Error()
}

func (e ValidationError) Unwrap() error {
	return e.Err
}

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
func ValidateSpecReport(spec *Spec) ValidationReport {
	if spec == nil {
		return validationReport([]ValidationIssue{{Error, "spec", "cannot be nil"}})
	}
	var issues []ValidationIssue
	if strings.TrimSpace(spec.Name) == "" {
		issues = append(issues, ValidationIssue{Error, "name", "cannot be empty"})
	}
	if strings.TrimSpace(spec.Overview) == "" {
		issues = append(issues, ValidationIssue{Error, "overview", "cannot be empty"})
	} else if len(spec.Overview) < minPurposeLength {
		issues = append(issues, ValidationIssue{Warning, "overview", fmt.Sprintf("purpose section is too brief (less than %d characters)", minPurposeLength)})
	}
	if len(spec.Requirements) == 0 {
		issues = append(issues, ValidationIssue{Error, "requirements", "must have at least one requirement"})
	}
	for i := range spec.Requirements {
		issues = append(issues, validateRequirementIssues(fmt.Sprintf("requirements[%d]", i), spec.Requirements[i])...)
	}
	return validationReport(issues)
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
func ValidateChangeReport(change *Change) ValidationReport {
	if change == nil {
		return validationReport([]ValidationIssue{{Error, "change", "cannot be nil"}})
	}
	var issues []ValidationIssue
	if strings.TrimSpace(change.Name) == "" {
		issues = append(issues, ValidationIssue{Error, "name", "cannot be empty"})
	}
	n := len(strings.TrimSpace(change.Why))
	if n < minWhyLength {
		issues = append(issues, ValidationIssue{Error, "why", fmt.Sprintf("must be at least %d characters", minWhyLength)})
	}
	if n > maxWhyLength {
		issues = append(issues, ValidationIssue{Error, "why", fmt.Sprintf("must not exceed %d characters", maxWhyLength)})
	}
	if strings.TrimSpace(change.WhatChanges) == "" {
		issues = append(issues, ValidationIssue{Error, "whatChanges", "cannot be empty"})
	}
	if len(change.Deltas) == 0 {
		issues = append(issues, ValidationIssue{Error, "deltas", "must have at least one delta"})
	}
	if len(change.Deltas) > maxDeltasPerChange {
		issues = append(issues, ValidationIssue{Error, "deltas", fmt.Sprintf("must not exceed %d deltas", maxDeltasPerChange)})
	}
	for i := range change.Deltas {
		issues = append(issues, validateDeltaIssues(fmt.Sprintf("deltas[%d]", i), change.Deltas[i])...)
	}
	return validationReport(issues)
}

func validateDeltaIssues(path string, delta Delta) []ValidationIssue {
	var issues []ValidationIssue
	if strings.TrimSpace(delta.Spec) == "" {
		issues = append(issues, ValidationIssue{Error, path + ".spec", "cannot be empty"})
	}
	switch delta.Operation {
	case Added, Modified, Removed, Renamed:
	default:
		issues = append(issues, ValidationIssue{Error, path + ".operation", "must be ADDED, MODIFIED, REMOVED, or RENAMED"})
	}
	if strings.TrimSpace(delta.Description) == "" {
		issues = append(issues, ValidationIssue{Error, path + ".description", "cannot be empty"})
	} else if len(delta.Description) < minDeltaDescriptionLength {
		issues = append(issues, ValidationIssue{Warning, path + ".description", "delta description is too brief"})
	}
	if (delta.Operation == Added || delta.Operation == Modified) && len(delta.Requirements) == 0 {
		issues = append(issues, ValidationIssue{Warning, path + ".requirements", string(delta.Operation) + " delta should include requirements"})
	}
	for i := range delta.Requirements {
		issues = append(issues, validateRequirementIssues(fmt.Sprintf("%s.requirements[%d]", path, i), delta.Requirements[i])...)
	}
	for i, rename := range delta.Renames {
		if strings.TrimSpace(rename.From) == "" {
			issues = append(issues, ValidationIssue{Error, fmt.Sprintf("%s.renames[%d].from", path, i), "cannot be empty"})
		}
		if strings.TrimSpace(rename.To) == "" {
			issues = append(issues, ValidationIssue{Error, fmt.Sprintf("%s.renames[%d].to", path, i), "cannot be empty"})
		}
	}
	return issues
}

func validateRequirementIssues(path string, req Requirement) []ValidationIssue {
	var issues []ValidationIssue
	if strings.TrimSpace(req.Text) == "" {
		issues = append(issues, ValidationIssue{Error, path + ".text", "cannot be empty"})
	}
	if !strings.Contains(req.Text, "SHALL") && !strings.Contains(req.Text, "MUST") {
		issues = append(issues, ValidationIssue{Error, path + ".text", "must contain SHALL or MUST keyword"})
	}
	if len(req.Text) > maxRequirementTextLength {
		issues = append(issues, ValidationIssue{Info, path, fmt.Sprintf("requirement text is very long (>%d characters)", maxRequirementTextLength)})
	}
	if len(req.Scenarios) == 0 {
		issues = append(issues, ValidationIssue{Error, path + ".scenarios", "must have at least one scenario"})
	}
	for i, scenario := range req.Scenarios {
		if strings.TrimSpace(scenario.RawText) == "" {
			issues = append(issues, ValidationIssue{Error, fmt.Sprintf("%s.scenarios[%d]", path, i), "cannot be empty"})
		}
	}
	return issues
}

func validationReport(issues []ValidationIssue) ValidationReport {
	var summary ValidationSummary
	for _, issue := range issues {
		switch issue.Level {
		case Error:
			summary.Errors++
		case Warning:
			summary.Warnings++
		case Info:
			summary.Info++
		}
	}
	return ValidationReport{Valid: summary.Errors == 0, Issues: issues, Summary: summary}
}

// Err returns the hard validation errors in report.
func (r ValidationReport) Err() error {
	var errs []error
	for _, issue := range r.Issues {
		if issue.Level == Error {
			errs = append(errs, ValidationError{issue.Path, errors.New(issue.Message)})
		}
	}
	return errors.Join(errs...)
}
