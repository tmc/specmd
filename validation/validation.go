// Package validation defines the shared validation vocabulary for specmd
// artifacts. Both [github.com/tmc/specmd/openspec] and
// [github.com/tmc/specmd/okf] report conformance findings as a [Report] of
// [Issue] values, so the types live in one place that neither imports the
// other.
package validation

import (
	"errors"
)

// Level is the severity of one validation issue.
type Level string

const (
	LevelError   Level = "ERROR"
	LevelWarning Level = "WARNING"
	LevelInfo    Level = "INFO"
)

// Issue reports one validation problem or advisory.
type Issue struct {
	Level   Level
	Path    string
	Message string
}

// Summary counts validation issues by severity.
type Summary struct {
	Errors   int
	Warnings int
	Info     int
}

// Report reports hard errors and advisory warnings for an artifact.
type Report struct {
	Valid   bool
	Issues  []Issue
	Summary Summary
}

// Error reports one invalid field in a spec, change, or concept.
type Error struct {
	Field string
	Err   error
}

func (e Error) Error() string {
	return e.Field + ": " + e.Err.Error()
}

func (e Error) Unwrap() error {
	return e.Err
}

// Err returns the hard validation errors in report.
func (r Report) Err() error {
	var errs []error
	for _, issue := range r.Issues {
		if issue.Level == LevelError {
			errs = append(errs, Error{issue.Path, errors.New(issue.Message)})
		}
	}
	return errors.Join(errs...)
}

// New builds a Report from issues, computing its summary and validity.
func New(issues []Issue) Report {
	var summary Summary
	for _, issue := range issues {
		switch issue.Level {
		case LevelError:
			summary.Errors++
		case LevelWarning:
			summary.Warnings++
		case LevelInfo:
			summary.Info++
		}
	}
	return Report{Valid: summary.Errors == 0, Issues: issues, Summary: summary}
}

// Prefix returns issues with prefix prepended to each Path, joining with a dot
// when the issue already has a path. It is used to nest a child artifact's
// findings under a parent path.
func Prefix(prefix string, issues []Issue) []Issue {
	out := make([]Issue, len(issues))
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
