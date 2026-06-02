package openspec

import (
	"errors"
	"fmt"
	"strings"
)

const (
	minWhyLength       = 50
	maxWhyLength       = 1000
	maxDeltasPerChange = 10
)

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
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("validate spec: %w", err)
	}
	return nil
}

func validateSpec(spec *Spec) error {
	var errs []error
	if strings.TrimSpace(spec.Name) == "" {
		errs = append(errs, ValidationError{"name", errors.New("cannot be empty")})
	}
	if strings.TrimSpace(spec.Overview) == "" {
		errs = append(errs, ValidationError{"overview", errors.New("cannot be empty")})
	}
	if len(spec.Requirements) == 0 {
		errs = append(errs, ValidationError{"requirements", errors.New("must have at least one requirement")})
	}
	for i := range spec.Requirements {
		if err := validateRequirement(spec.Requirements[i]); err != nil {
			errs = append(errs, ValidationError{fmt.Sprintf("requirements[%d]", i), err})
		}
	}
	return errors.Join(errs...)
}

// ValidateChange checks the required OpenSpec change shape.
func ValidateChange(change *Change) error {
	if change == nil {
		return fmt.Errorf("validate change: nil change")
	}
	if err := validateChange(change); err != nil {
		return fmt.Errorf("validate change: %w", err)
	}
	return nil
}

func validateChange(change *Change) error {
	var errs []error
	if strings.TrimSpace(change.Name) == "" {
		errs = append(errs, ValidationError{"name", errors.New("cannot be empty")})
	}
	n := len(strings.TrimSpace(change.Why))
	if n < minWhyLength {
		errs = append(errs, ValidationError{"why", fmt.Errorf("must be at least %d characters", minWhyLength)})
	}
	if n > maxWhyLength {
		errs = append(errs, ValidationError{"why", fmt.Errorf("must not exceed %d characters", maxWhyLength)})
	}
	if strings.TrimSpace(change.WhatChanges) == "" {
		errs = append(errs, ValidationError{"whatChanges", errors.New("cannot be empty")})
	}
	if len(change.Deltas) == 0 {
		errs = append(errs, ValidationError{"deltas", errors.New("must have at least one delta")})
	}
	if len(change.Deltas) > maxDeltasPerChange {
		errs = append(errs, ValidationError{"deltas", fmt.Errorf("must not exceed %d deltas", maxDeltasPerChange)})
	}
	for i := range change.Deltas {
		if err := validateDelta(change.Deltas[i]); err != nil {
			errs = append(errs, ValidationError{fmt.Sprintf("deltas[%d]", i), err})
		}
	}
	return errors.Join(errs...)
}

func validateDelta(delta Delta) error {
	var errs []error
	if strings.TrimSpace(delta.Spec) == "" {
		errs = append(errs, ValidationError{"spec", errors.New("cannot be empty")})
	}
	switch delta.Operation {
	case Added, Modified, Removed, Renamed:
	default:
		errs = append(errs, ValidationError{"operation", errors.New("must be ADDED, MODIFIED, REMOVED, or RENAMED")})
	}
	if strings.TrimSpace(delta.Description) == "" {
		errs = append(errs, ValidationError{"description", errors.New("cannot be empty")})
	}
	for i := range delta.Requirements {
		if err := validateRequirement(delta.Requirements[i]); err != nil {
			errs = append(errs, ValidationError{fmt.Sprintf("requirements[%d]", i), err})
		}
	}
	for i, rename := range delta.Renames {
		if strings.TrimSpace(rename.From) == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("renames[%d].from", i), errors.New("cannot be empty")})
		}
		if strings.TrimSpace(rename.To) == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("renames[%d].to", i), errors.New("cannot be empty")})
		}
	}
	return errors.Join(errs...)
}

func validateRequirement(req Requirement) error {
	var errs []error
	if strings.TrimSpace(req.Text) == "" {
		errs = append(errs, ValidationError{"text", errors.New("cannot be empty")})
	}
	if !strings.Contains(req.Text, "SHALL") && !strings.Contains(req.Text, "MUST") {
		errs = append(errs, ValidationError{"text", errors.New("must contain SHALL or MUST keyword")})
	}
	if len(req.Scenarios) == 0 {
		errs = append(errs, ValidationError{"scenarios", errors.New("must have at least one scenario")})
	}
	for i, scenario := range req.Scenarios {
		if strings.TrimSpace(scenario.RawText) == "" {
			errs = append(errs, ValidationError{fmt.Sprintf("scenarios[%d]", i), errors.New("cannot be empty")})
		}
	}
	return errors.Join(errs...)
}
