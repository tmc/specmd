package openspec

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSpec(t *testing.T) {
	spec, err := ParseSpec("auth", strings.NewReader(`# Auth

## Purpose
Authentication and session management for the application.

## Requirements

### Requirement: User Authentication
The system SHALL issue a token upon successful login.

#### Scenario: Valid credentials
- GIVEN a user with valid credentials
- WHEN the user submits the login form
- THEN a token is returned

#### Scenario: Invalid credentials
- GIVEN invalid credentials
- WHEN the user submits the login form
- THEN an error is displayed
`))
	if err != nil {
		t.Fatal(err)
	}
	if spec.Name != "auth" {
		t.Fatalf("Name = %q, want auth", spec.Name)
	}
	if len(spec.Requirements) != 1 {
		t.Fatalf("len(Requirements) = %d, want 1", len(spec.Requirements))
	}
	req := spec.Requirements[0]
	if req.Name != "User Authentication" {
		t.Fatalf("Requirement.Name = %q", req.Name)
	}
	if len(req.Scenarios) != 2 {
		t.Fatalf("len(Scenarios) = %d, want 2", len(req.Scenarios))
	}
}

func TestParseChange(t *testing.T) {
	change, err := ParseChange("add-2fa", strings.NewReader(`# Proposal

## Why
Users need stronger login protection for sensitive accounts and administrators.

## What Changes
Add two-factor authentication during login.
`), map[string]io.Reader{
		"auth": strings.NewReader(`# Delta for Auth

## ADDED Requirements
This delta adds two-factor authentication behavior.

### Requirement: Two-Factor Authentication
The system MUST require a one-time code after password login.

#### Scenario: OTP challenge
- GIVEN a user with two-factor authentication enabled
- WHEN password login succeeds
- THEN an OTP challenge is shown
`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(change.Deltas) != 1 {
		t.Fatalf("len(Deltas) = %d, want 1", len(change.Deltas))
	}
	if change.Deltas[0].Operation != Added {
		t.Fatalf("Operation = %q, want %q", change.Deltas[0].Operation, Added)
	}
	if err := ValidateChange(change); err != nil {
		t.Fatal(err)
	}
}

func TestParseSpecTestdata(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "specs", "auth", "spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	spec, err := ParseSpec("auth", f)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSpec(spec); err != nil {
		t.Fatal(err)
	}
	if got, want := len(spec.Requirements), 2; got != want {
		t.Fatalf("len(Requirements) = %d, want %d", got, want)
	}
	if got, want := len(spec.Requirements[0].Scenarios), 2; got != want {
		t.Fatalf("len(Scenarios) = %d, want %d", got, want)
	}
}

func TestParseChangeTestdata(t *testing.T) {
	proposal, err := os.Open(filepath.Join("testdata", "changes", "add-2fa", "proposal.md"))
	if err != nil {
		t.Fatal(err)
	}
	defer proposal.Close()

	specDelta, err := os.Open(filepath.Join("testdata", "changes", "add-2fa", "specs", "auth", "spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	defer specDelta.Close()

	change, err := ParseChange("add-2fa", proposal, map[string]io.Reader{"auth": specDelta})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateChange(change); err != nil {
		t.Fatal(err)
	}
	if got, want := len(change.Deltas), 1; got != want {
		t.Fatalf("len(Deltas) = %d, want %d", got, want)
	}
	if got, want := len(change.Deltas[0].Requirements[0].Scenarios), 2; got != want {
		t.Fatalf("len(Scenarios) = %d, want %d", got, want)
	}
}

func TestParseSpecIgnoresFencedHeadings(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "specs", "auth", "spec-with-code.md"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	spec, err := ParseSpec("auth", f)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSpec(spec); err != nil {
		t.Fatal(err)
	}
	if got, want := len(spec.Requirements), 1; got != want {
		t.Fatalf("len(Requirements) = %d, want %d", got, want)
	}
	if got, want := spec.Requirements[0].Name, "Parsed Outside Code Fence"; got != want {
		t.Fatalf("Requirement.Name = %q, want %q", got, want)
	}
}

func TestParseDeltaOperationsTestdata(t *testing.T) {
	tests := []struct {
		name      string
		changeID  string
		operation DeltaOperation
	}{
		{"added", "add-2fa", Added},
		{"modified", "modify-session", Modified},
		{"removed", "remove-legacy-login", Removed},
		{"renamed", "rename-login", Renamed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal, err := os.Open(filepath.Join("testdata", "changes", tt.changeID, "proposal.md"))
			if err != nil {
				t.Fatal(err)
			}
			defer proposal.Close()

			specDelta, err := os.Open(filepath.Join("testdata", "changes", tt.changeID, "specs", "auth", "spec.md"))
			if err != nil {
				t.Fatal(err)
			}
			defer specDelta.Close()

			change, err := ParseChange(tt.changeID, proposal, map[string]io.Reader{"auth": specDelta})
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateChange(change); err != nil {
				t.Fatal(err)
			}
			if got, want := len(change.Deltas), 1; got != want {
				t.Fatalf("len(Deltas) = %d, want %d", got, want)
			}
			if got := change.Deltas[0].Operation; got != tt.operation {
				t.Fatalf("Operation = %q, want %q", got, tt.operation)
			}
		})
	}
}

func TestParseDeltaRenames(t *testing.T) {
	change, err := ParseChange("rename-auth", strings.NewReader(`# Proposal

## Why
The old requirement name is unclear to readers and should use product language.

## What Changes
Rename the login requirement to a clearer authentication requirement.
`), map[string]io.Reader{
		"auth": strings.NewReader(`# Delta for Auth

## renamed Requirements
- FROM: ` + "`### Requirement: Login`" + `
- TO: ` + "`### Requirement: User Authentication`" + `
`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(change.Deltas) != 1 {
		t.Fatalf("len(Deltas) = %d, want 1", len(change.Deltas))
	}
	delta := change.Deltas[0]
	if delta.Operation != Renamed {
		t.Fatalf("Operation = %q, want %q", delta.Operation, Renamed)
	}
	if len(delta.Renames) != 1 {
		t.Fatalf("len(Renames) = %d, want 1", len(delta.Renames))
	}
	if delta.Renames[0].From != "Login" || delta.Renames[0].To != "User Authentication" {
		t.Fatalf("Rename = %+v", delta.Renames[0])
	}
	if err := ValidateChange(change); err != nil {
		t.Fatal(err)
	}
}

func TestParseSpecFile(t *testing.T) {
	path := filepath.Join("testdata", "project", "openspec", "specs", "auth", "spec.md")
	spec, err := ParseSpecFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := spec.Name, "auth"; got != want {
		t.Fatalf("Name = %q, want %q", got, want)
	}
	if got, want := spec.Metadata.SourcePath, path; got != want {
		t.Fatalf("SourcePath = %q, want %q", got, want)
	}
}

func TestParseChangeDir(t *testing.T) {
	path := filepath.Join("testdata", "project", "openspec", "changes", "add-2fa")
	change, err := ParseChangeDir(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := change.Name, "add-2fa"; got != want {
		t.Fatalf("Name = %q, want %q", got, want)
	}
	if got, want := change.Metadata.SourcePath, filepath.Join(path, "proposal.md"); got != want {
		t.Fatalf("SourcePath = %q, want %q", got, want)
	}
	if got, want := change.Deltas[0].Metadata.SourcePath, filepath.Join(path, "specs", "auth", "spec.md"); got != want {
		t.Fatalf("Delta SourcePath = %q, want %q", got, want)
	}
}

func TestParseProjectStableOrder(t *testing.T) {
	project, err := ParseProject(filepath.Join("testdata", "project", "openspec"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(project.Specs), 2; got != want {
		t.Fatalf("len(Specs) = %d, want %d", got, want)
	}
	if got, want := []string{project.Specs[0].Name, project.Specs[1].Name}, []string{"auth", "billing"}; got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Specs = %q, want %q", got, want)
	}
	if got, want := len(project.Changes), 2; got != want {
		t.Fatalf("len(Changes) = %d, want %d", got, want)
	}
	if got, want := []string{project.Changes[0].Name, project.Changes[1].Name}, []string{"add-2fa", "update-billing"}; got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Changes = %q, want %q", got, want)
	}
}
