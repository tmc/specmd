package openspec_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/openspec"
)

func Example() {
	spec, err := openspec.ParseSpec("auth", strings.NewReader(`# Auth

## Purpose
Authentication and session management for the application.

## Requirements

### Requirement: User Authentication
The system SHALL issue a token upon successful login.

#### Scenario: Valid credentials
- GIVEN a user with valid credentials
- WHEN the user submits the login form
- THEN a token is returned
`))
	if err != nil {
		panic(err)
	}
	fmt.Println(spec.Name, len(spec.Requirements))
	// Output: auth 1
}

func ExampleParseSpec() {
	f, err := os.Open(filepath.Join("testdata", "specs", "auth", "spec.md"))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	spec, err := openspec.ParseSpec("auth", f)
	if err != nil {
		panic(err)
	}
	fmt.Println(spec.Name)
	fmt.Println(len(spec.Requirements))
	fmt.Println(spec.Requirements[0].Name)
	// Output:
	// auth
	// 2
	// User Authentication
}

func ExampleParseChange() {
	proposal, err := os.Open(filepath.Join("testdata", "changes", "add-2fa", "proposal.md"))
	if err != nil {
		panic(err)
	}
	defer proposal.Close()

	specDelta, err := os.Open(filepath.Join("testdata", "changes", "add-2fa", "specs", "auth", "spec.md"))
	if err != nil {
		panic(err)
	}
	defer specDelta.Close()

	change, err := openspec.ParseChange("add-2fa", proposal, map[string]io.Reader{
		"auth": specDelta,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(change.Name)
	fmt.Println(change.Deltas[0].Operation)
	fmt.Println(change.Deltas[0].Requirements[0].Name)
	// Output:
	// add-2fa
	// ADDED
	// Two-Factor Authentication
}

func ExampleParseProject() {
	project, err := openspec.ParseProject(filepath.Join("testdata", "project", "openspec"))
	if err != nil {
		panic(err)
	}
	fmt.Println(len(project.Specs))
	fmt.Println(project.Specs[0].Name)
	fmt.Println(len(project.Changes))
	fmt.Println(project.Changes[0].Name)
	// Output:
	// 2
	// auth
	// 2
	// add-2fa
}

func ExampleValidateSpec() {
	spec := &openspec.Spec{
		Name:     "auth",
		Overview: "Authentication and session management.",
		Requirements: []openspec.Requirement{{
			Text: "The system SHALL issue a token upon successful login.",
			Scenarios: []openspec.Scenario{{
				RawText: "- WHEN valid credentials are submitted\n- THEN a token is returned",
			}},
		}},
	}
	fmt.Println(openspec.ValidateSpec(spec))
	// Output: <nil>
}

func ExampleValidateChange() {
	change := &openspec.Change{
		Name:        "rename-login",
		Why:         "The existing requirement name uses implementation shorthand and should match user-facing behavior language.",
		WhatChanges: "Rename the login requirement to user authentication.",
		Deltas: []openspec.Delta{{
			Spec:        "auth",
			Operation:   openspec.Renamed,
			Description: "Rename requirement from \"Login\" to \"User Authentication\"",
			Renames: []openspec.Rename{{
				From: "Login",
				To:   "User Authentication",
			}},
		}},
	}
	fmt.Println(openspec.ValidateChange(change))
	// Output: <nil>
}
