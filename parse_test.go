package openspec

import (
	"io"
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
