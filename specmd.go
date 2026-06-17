package specmd

import "github.com/tmc/specmd/validation"

// Shared validation vocabulary, re-exported from
// [github.com/tmc/specmd/validation] so callers can use one set of names across
// every Markdown family.
type (
	ValidationLevel   = validation.Level
	ValidationIssue   = validation.Issue
	ValidationSummary = validation.Summary
	ValidationReport  = validation.Report
	ValidationError   = validation.Error
)

// Validation severity levels, re-exported from
// [github.com/tmc/specmd/validation].
const (
	ValidationLevelError   = validation.LevelError
	ValidationLevelWarning = validation.LevelWarning
	ValidationLevelInfo    = validation.LevelInfo
)
