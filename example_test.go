package openspec_test

import (
	"fmt"
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
