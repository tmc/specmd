package openspec

import (
	"strings"
	"testing"

	"github.com/tmc/specmd/validation"
)

func TestValidateSpec(t *testing.T) {
	tests := []struct {
		name string
		spec *Spec
		want string
	}{
		{
			name: "valid",
			spec: &Spec{
				Name:     "auth",
				Overview: "Authentication and session management.",
				Requirements: []Requirement{{
					Text:      "The system SHALL issue a token.",
					Scenarios: []Scenario{{RawText: "- WHEN login succeeds\n- THEN a token is returned"}},
				}},
			},
		},
		{
			name: "missing keyword",
			spec: &Spec{
				Name:     "auth",
				Overview: "Authentication and session management.",
				Requirements: []Requirement{{
					Text:      "The system issues a token.",
					Scenarios: []Scenario{{RawText: "- WHEN login succeeds\n- THEN a token is returned"}},
				}},
			},
			want: "SHALL or MUST",
		},
		{
			name: "missing scenario",
			spec: &Spec{
				Name:     "auth",
				Overview: "Authentication and session management.",
				Requirements: []Requirement{{
					Text: "The system SHALL issue a token.",
				}},
			},
			want: "scenario",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSpec(tt.spec)
			if tt.want == "" && err != nil {
				t.Fatalf("ValidateSpec() = %v", err)
			}
			if tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Fatalf("ValidateSpec() = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestValidateSpecAllowsSoftWarningThresholds(t *testing.T) {
	spec := &Spec{
		Name:     "auth",
		Overview: "short",
		Requirements: []Requirement{{
			Text: "The system SHALL " + strings.Repeat("preserve documented behavior ", 30),
			Scenarios: []Scenario{{
				RawText: "- WHEN behavior is documented\n- THEN validation accepts the spec",
			}},
		}},
	}
	if err := ValidateSpec(spec); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSpecReport(t *testing.T) {
	spec := &Spec{
		Name:     "auth",
		Overview: "short",
		Requirements: []Requirement{{
			Text: "The system SHALL " + strings.Repeat("preserve documented behavior ", 30),
			Scenarios: []Scenario{{
				RawText: "- WHEN behavior is documented\n- THEN validation accepts the spec",
			}},
		}},
	}
	report := ValidateSpecReport(spec)
	if !report.Valid {
		t.Fatalf("Valid = false, issues: %+v", report.Issues)
	}
	if report.Summary.Errors != 0 {
		t.Fatalf("Errors = %d, want 0", report.Summary.Errors)
	}
	if report.Summary.Warnings != 1 {
		t.Fatalf("Warnings = %d, want 1", report.Summary.Warnings)
	}
	if report.Summary.Info != 1 {
		t.Fatalf("Info = %d, want 1", report.Summary.Info)
	}
	if err := report.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSpecReportBoundaries(t *testing.T) {
	requirementPrefix := "The system SHALL "
	tests := []struct {
		name     string
		overview string
		text     string
		want     validation.Summary
	}{
		{
			name:     "purpose exactly minimum",
			overview: strings.Repeat("a", minPurposeLength),
			text:     "The system SHALL issue a token.",
		},
		{
			name:     "purpose one short warns",
			overview: strings.Repeat("a", minPurposeLength-1),
			text:     "The system SHALL issue a token.",
			want:     validation.Summary{Warnings: 1},
		},
		{
			name:     "requirement text exactly maximum",
			overview: strings.Repeat("a", minPurposeLength),
			text:     requirementPrefix + strings.Repeat("x", maxRequirementTextLength-len(requirementPrefix)),
		},
		{
			name:     "requirement text one long is info",
			overview: strings.Repeat("a", minPurposeLength),
			text:     requirementPrefix + strings.Repeat("x", maxRequirementTextLength-len(requirementPrefix)+1),
			want:     validation.Summary{Info: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &Spec{
				Name:     "auth",
				Overview: tt.overview,
				Requirements: []Requirement{{
					Text:      tt.text,
					Scenarios: []Scenario{{RawText: "- WHEN login succeeds\n- THEN a token is returned"}},
				}},
			}
			report := ValidateSpecReport(spec)
			if report.Summary != tt.want {
				t.Fatalf("Summary = %+v, want %+v; issues: %+v", report.Summary, tt.want, report.Issues)
			}
			if err := report.Err(); err != nil {
				t.Fatalf("Err() = %v", err)
			}
		})
	}
}

func TestValidateChangeReport(t *testing.T) {
	change := &Change{
		Name:        "add-empty",
		Why:         "This change exists to check advisory validation behavior for sparse delta data.",
		WhatChanges: "Add a placeholder delta.",
		Deltas: []Delta{{
			Spec:        "auth",
			Operation:   Added,
			Description: "brief",
		}},
	}
	report := ValidateChangeReport(change)
	if !report.Valid {
		t.Fatalf("Valid = false, issues: %+v", report.Issues)
	}
	if report.Summary.Errors != 0 {
		t.Fatalf("Errors = %d, want 0", report.Summary.Errors)
	}
	if report.Summary.Warnings != 2 {
		t.Fatalf("Warnings = %d, want 2", report.Summary.Warnings)
	}
	if err := ValidateChange(change); err != nil {
		t.Fatal(err)
	}
}

func TestValidateChangeReportBoundaries(t *testing.T) {
	validDelta := Delta{
		Spec:        "auth",
		Operation:   Renamed,
		Description: strings.Repeat("d", minDeltaDescriptionLength),
		Renames:     []Rename{{From: "Login", To: "User Authentication"}},
	}
	tests := []struct {
		name string
		why  string
		edit func(*Change)
		want validation.Summary
	}{
		{
			name: "why exactly minimum",
			why:  strings.Repeat("w", minWhyLength),
		},
		{
			name: "why one short is error",
			why:  strings.Repeat("w", minWhyLength-1),
			want: validation.Summary{Errors: 1},
		},
		{
			name: "why exactly maximum",
			why:  strings.Repeat("w", maxWhyLength),
		},
		{
			name: "why one long is error",
			why:  strings.Repeat("w", maxWhyLength+1),
			want: validation.Summary{Errors: 1},
		},
		{
			name: "delta description one short warns",
			why:  strings.Repeat("w", minWhyLength),
			edit: func(change *Change) {
				change.Deltas[0].Description = strings.Repeat("d", minDeltaDescriptionLength-1)
			},
			want: validation.Summary{Warnings: 1},
		},
		{
			name: "missing added requirements warns",
			why:  strings.Repeat("w", minWhyLength),
			edit: func(change *Change) {
				change.Deltas[0] = Delta{
					Spec:        "auth",
					Operation:   Added,
					Description: strings.Repeat("d", minDeltaDescriptionLength),
				}
			},
			want: validation.Summary{Warnings: 1},
		},
		{
			name: "deltas exactly maximum",
			why:  strings.Repeat("w", minWhyLength),
			edit: func(change *Change) {
				change.Deltas = repeatDelta(validDelta, maxDeltasPerChange)
			},
		},
		{
			name: "deltas one too many is error",
			why:  strings.Repeat("w", minWhyLength),
			edit: func(change *Change) {
				change.Deltas = repeatDelta(validDelta, maxDeltasPerChange+1)
			},
			want: validation.Summary{Errors: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := &Change{
				Name:        "change",
				Why:         tt.why,
				WhatChanges: "Rename the login requirement.",
				Deltas:      []Delta{validDelta},
			}
			if tt.edit != nil {
				tt.edit(change)
			}
			report := ValidateChangeReport(change)
			if report.Summary != tt.want {
				t.Fatalf("Summary = %+v, want %+v; issues: %+v", report.Summary, tt.want, report.Issues)
			}
		})
	}
}

func repeatDelta(delta Delta, n int) []Delta {
	deltas := make([]Delta, n)
	for i := range deltas {
		deltas[i] = delta
	}
	return deltas
}
