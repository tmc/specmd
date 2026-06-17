package openspec

import (
	"errors"
	"fmt"
	"regexp"
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
	ValidationLevelError   ValidationLevel = "ERROR"
	ValidationLevelWarning ValidationLevel = "WARNING"
	ValidationLevelInfo    ValidationLevel = "INFO"
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
		return validationReport([]ValidationIssue{{ValidationLevelError, "spec", "cannot be nil"}})
	}
	var issues []ValidationIssue
	if strings.TrimSpace(spec.Name) == "" {
		issues = append(issues, ValidationIssue{ValidationLevelError, "name", "cannot be empty"})
	}
	if strings.TrimSpace(spec.Overview) == "" {
		issues = append(issues, ValidationIssue{ValidationLevelError, "overview", "cannot be empty"})
	} else if len(spec.Overview) < minPurposeLength {
		issues = append(issues, ValidationIssue{ValidationLevelWarning, "overview", fmt.Sprintf("purpose section is too brief (less than %d characters)", minPurposeLength)})
	}
	if len(spec.Requirements) == 0 {
		issues = append(issues, ValidationIssue{ValidationLevelError, "requirements", "must have at least one requirement"})
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
		return validationReport([]ValidationIssue{{ValidationLevelError, "change", "cannot be nil"}})
	}
	var issues []ValidationIssue
	if strings.TrimSpace(change.Name) == "" {
		issues = append(issues, ValidationIssue{ValidationLevelError, "name", "cannot be empty"})
	}
	n := len(strings.TrimSpace(change.Why))
	if n < minWhyLength {
		issues = append(issues, ValidationIssue{ValidationLevelError, "why", fmt.Sprintf("must be at least %d characters", minWhyLength)})
	}
	if n > maxWhyLength {
		issues = append(issues, ValidationIssue{ValidationLevelError, "why", fmt.Sprintf("must not exceed %d characters", maxWhyLength)})
	}
	if strings.TrimSpace(change.WhatChanges) == "" {
		issues = append(issues, ValidationIssue{ValidationLevelError, "whatChanges", "cannot be empty"})
	}
	if len(change.Deltas) == 0 {
		issues = append(issues, ValidationIssue{ValidationLevelError, "deltas", "must have at least one delta"})
	}
	if len(change.Deltas) > maxDeltasPerChange {
		issues = append(issues, ValidationIssue{ValidationLevelError, "deltas", fmt.Sprintf("must not exceed %d deltas", maxDeltasPerChange)})
	}
	for i := range change.Deltas {
		issues = append(issues, validateDeltaIssues(fmt.Sprintf("deltas[%d]", i), change.Deltas[i])...)
	}
	return validationReport(issues)
}

// ValidateOKFConcept checks the hard OKF v0.1 concept document rules.
func ValidateOKFConcept(concept *OKFConcept) error {
	if concept == nil {
		return fmt.Errorf("validate okf concept: nil concept")
	}
	if err := ValidateOKFConceptReport(concept).Err(); err != nil {
		return fmt.Errorf("validate okf concept: %w", err)
	}
	return nil
}

// ValidateOKFConceptReport checks an OKF concept and returns errors plus warnings.
func ValidateOKFConceptReport(concept *OKFConcept) ValidationReport {
	if concept == nil {
		return validationReport([]ValidationIssue{{ValidationLevelError, "concept", "cannot be nil"}})
	}
	var issues []ValidationIssue
	if strings.TrimSpace(concept.ID) == "" {
		issues = append(issues, ValidationIssue{ValidationLevelError, "id", "cannot be empty"})
	}
	if strings.TrimSpace(concept.Type) == "" {
		issues = append(issues, ValidationIssue{ValidationLevelError, "type", "cannot be empty"})
	}
	if strings.TrimSpace(concept.Title) == "" {
		issues = append(issues, ValidationIssue{ValidationLevelInfo, "title", "recommended field is missing"})
	}
	if strings.TrimSpace(concept.Description) == "" {
		issues = append(issues, ValidationIssue{ValidationLevelInfo, "description", "recommended field is missing"})
	}
	return validationReport(issues)
}

// ValidateOKFBundle checks the hard OKF v0.1 bundle rules.
func ValidateOKFBundle(bundle *OKFBundle) error {
	if bundle == nil {
		return fmt.Errorf("validate okf bundle: nil bundle")
	}
	if err := ValidateOKFBundleReport(bundle).Err(); err != nil {
		return fmt.Errorf("validate okf bundle: %w", err)
	}
	return nil
}

// ValidateOKFBundleReport checks an OKF bundle and returns errors plus warnings.
func ValidateOKFBundleReport(bundle *OKFBundle) ValidationReport {
	if bundle == nil {
		return validationReport([]ValidationIssue{{ValidationLevelError, "bundle", "cannot be nil"}})
	}
	var issues []ValidationIssue
	for i := range bundle.Concepts {
		report := ValidateOKFConceptReport(&bundle.Concepts[i])
		issues = append(issues, prefixValidationIssues("concepts."+bundle.Concepts[i].ID, report.Issues)...)
	}
	for _, bad := range bundle.Invalid {
		issues = append(issues, ValidationIssue{ValidationLevelError, "concepts." + bad.ID, "unparseable frontmatter: " + bad.Err.Error()})
	}
	for i := range bundle.Index {
		issues = append(issues, validateOKFIndexIssues(fmt.Sprintf("index[%d]", i), bundle.Index[i])...)
	}
	for i := range bundle.Logs {
		issues = append(issues, validateOKFLogIssues(fmt.Sprintf("logs[%d]", i), bundle.Logs[i])...)
	}
	return validationReport(issues)
}

func validateOKFIndexIssues(path string, file OKFReservedFile) []ValidationIssue {
	var issues []ValidationIssue
	if len(file.FrontMatter) > 0 && !file.Root {
		issues = append(issues, ValidationIssue{ValidationLevelError, path, "frontmatter is only permitted in root index.md"})
	}
	if !hasMarkdownHeading(file.Body, "# ") {
		issues = append(issues, ValidationIssue{ValidationLevelWarning, path, "should contain at least one section heading"})
	}
	if !hasMarkdownListLink(file.Body) {
		issues = append(issues, ValidationIssue{ValidationLevelWarning, path, "should contain at least one linked list entry"})
	}
	return issues
}

func validateOKFLogIssues(path string, file OKFReservedFile) []ValidationIssue {
	var issues []ValidationIssue
	if len(file.FrontMatter) > 0 {
		issues = append(issues, ValidationIssue{ValidationLevelError, path, "log.md must not contain frontmatter"})
	}
	if !hasMarkdownHeading(file.Body, "# ") {
		issues = append(issues, ValidationIssue{ValidationLevelWarning, path, "should contain a title heading"})
	}
	for _, bad := range malformedDateHeadings(file.Body) {
		issues = append(issues, ValidationIssue{ValidationLevelError, path, "date heading must use ISO 8601 form: " + bad})
	}
	return issues
}

func hasMarkdownHeading(text, prefix string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return true
		}
	}
	return false
}

func hasMarkdownListLink(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if (strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "- ")) && strings.Contains(line, "](") {
			return true
		}
	}
	return false
}

var isoDateHeadingRE = regexp.MustCompile(`^## [0-9]{4}-[0-9]{2}-[0-9]{2}$`)
var dateLikeHeadingRE = regexp.MustCompile(`^## [0-9]{4}[-/][0-9]{1,2}[-/][0-9]{1,2}$`)

func malformedDateHeadings(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if dateLikeHeadingRE.MatchString(line) && !isoDateHeadingRE.MatchString(line) {
			out = append(out, strings.TrimSpace(strings.TrimPrefix(line, "##")))
		}
	}
	return out
}

func prefixValidationIssues(prefix string, issues []ValidationIssue) []ValidationIssue {
	out := make([]ValidationIssue, len(issues))
	for i, issue := range issues {
		out[i] = issue
		if issue.Path == "" {
			out[i].Path = prefix
		} else {
			out[i].Path = prefix + "." + issue.Path
		}
	}
	return out
}

func validateDeltaIssues(path string, delta Delta) []ValidationIssue {
	var issues []ValidationIssue
	if strings.TrimSpace(delta.Spec) == "" {
		issues = append(issues, ValidationIssue{ValidationLevelError, path + ".spec", "cannot be empty"})
	}
	switch delta.Operation {
	case Added, Modified, Removed, Renamed:
	default:
		issues = append(issues, ValidationIssue{ValidationLevelError, path + ".operation", "must be ADDED, MODIFIED, REMOVED, or RENAMED"})
	}
	if strings.TrimSpace(delta.Description) == "" {
		issues = append(issues, ValidationIssue{ValidationLevelError, path + ".description", "cannot be empty"})
	} else if len(delta.Description) < minDeltaDescriptionLength {
		issues = append(issues, ValidationIssue{ValidationLevelWarning, path + ".description", "delta description is too brief"})
	}
	if (delta.Operation == Added || delta.Operation == Modified) && len(delta.Requirements) == 0 {
		issues = append(issues, ValidationIssue{ValidationLevelWarning, path + ".requirements", string(delta.Operation) + " delta should include requirements"})
	}
	for i := range delta.Requirements {
		issues = append(issues, validateRequirementIssues(fmt.Sprintf("%s.requirements[%d]", path, i), delta.Requirements[i])...)
	}
	for i, rename := range delta.Renames {
		if strings.TrimSpace(rename.From) == "" {
			issues = append(issues, ValidationIssue{ValidationLevelError, fmt.Sprintf("%s.renames[%d].from", path, i), "cannot be empty"})
		}
		if strings.TrimSpace(rename.To) == "" {
			issues = append(issues, ValidationIssue{ValidationLevelError, fmt.Sprintf("%s.renames[%d].to", path, i), "cannot be empty"})
		}
	}
	return issues
}

func validateRequirementIssues(path string, req Requirement) []ValidationIssue {
	var issues []ValidationIssue
	if strings.TrimSpace(req.Text) == "" {
		issues = append(issues, ValidationIssue{ValidationLevelError, path + ".text", "cannot be empty"})
	}
	if !strings.Contains(req.Text, "SHALL") && !strings.Contains(req.Text, "MUST") {
		issues = append(issues, ValidationIssue{ValidationLevelError, path + ".text", "must contain SHALL or MUST keyword"})
	}
	if len(req.Text) > maxRequirementTextLength {
		issues = append(issues, ValidationIssue{ValidationLevelInfo, path, fmt.Sprintf("requirement text is very long (>%d characters)", maxRequirementTextLength)})
	}
	if len(req.Scenarios) == 0 {
		issues = append(issues, ValidationIssue{ValidationLevelError, path + ".scenarios", "must have at least one scenario"})
	}
	for i, scenario := range req.Scenarios {
		if strings.TrimSpace(scenario.RawText) == "" {
			issues = append(issues, ValidationIssue{ValidationLevelError, fmt.Sprintf("%s.scenarios[%d]", path, i), "cannot be empty"})
		}
	}
	return issues
}

func validationReport(issues []ValidationIssue) ValidationReport {
	var summary ValidationSummary
	for _, issue := range issues {
		switch issue.Level {
		case ValidationLevelError:
			summary.Errors++
		case ValidationLevelWarning:
			summary.Warnings++
		case ValidationLevelInfo:
			summary.Info++
		}
	}
	return ValidationReport{Valid: summary.Errors == 0, Issues: issues, Summary: summary}
}

// Err returns the hard validation errors in report.
func (r ValidationReport) Err() error {
	var errs []error
	for _, issue := range r.Issues {
		if issue.Level == ValidationLevelError {
			errs = append(errs, ValidationError{issue.Path, errors.New(issue.Message)})
		}
	}
	return errors.Join(errs...)
}
