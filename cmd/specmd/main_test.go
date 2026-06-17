package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	var out, err bytes.Buffer
	if e := run([]string{"help"}, &out, &err); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(out.String(), "validate [path]") || err.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", out.String(), err.String())
	}
}

func TestValidateSpecWarningsDoNotFail(t *testing.T) {
	path := filepath.Join("..", "..", "openspec", "testdata", "specs", "auth", "spec.md")
	var out, err bytes.Buffer
	if e := run([]string{"validate", path}, &out, &err); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(out.String(), "spec: "+path+": valid") {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestValidateStrictFailsOnWarnings(t *testing.T) {
	path := writeSpec(t, "spec.md", `# Auth

## Purpose
short

## Requirements

### Requirement: Login
The system SHALL issue a token.

#### Scenario: Valid
- WHEN login succeeds
- THEN a token is returned
`)
	var out, err bytes.Buffer
	e := run([]string{"validate", "-strict", path}, &out, &err)
	if e == nil || !strings.Contains(e.Error(), "warning") {
		t.Fatalf("run strict = %v, stdout=%q stderr=%q", e, out.String(), err.String())
	}
}

func TestLSPRejectsArgs(t *testing.T) {
	var out, err bytes.Buffer
	e := run([]string{"lsp", "extra"}, &out, &err)
	if e == nil || !strings.Contains(e.Error(), "too many arguments") {
		t.Fatalf("run lsp = %v", e)
	}
}

func TestValidateJSON(t *testing.T) {
	path := filepath.Join("..", "..", "openspec", "testdata", "project", "openspec")
	var out, err bytes.Buffer
	if e := run([]string{"validate", "-json", path}, &out, &err); e != nil {
		t.Fatal(e)
	}
	var result validationResult
	if e := json.Unmarshal(out.Bytes(), &result); e != nil {
		t.Fatalf("json: %v\n%s", e, out.String())
	}
	if result.Kind != "project" {
		t.Fatalf("Kind = %q, want project", result.Kind)
	}
	if result.Summary.Errors != 0 {
		t.Fatalf("Errors = %d, issues=%+v", result.Summary.Errors, result.Issues)
	}
}

func TestValidateOKFBundle(t *testing.T) {
	path := filepath.Join("..", "..", "okf", "testdata", "okf")
	var out, err bytes.Buffer
	if e := run([]string{"validate", path}, &out, &err); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(out.String(), "okf-bundle: "+path+": valid") {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestValidateOKFConceptJSON(t *testing.T) {
	path := filepath.Join("..", "..", "okf", "testdata", "okf", "tables", "orders.md")
	var out, err bytes.Buffer
	if e := run([]string{"validate", "-json", path}, &out, &err); e != nil {
		t.Fatal(e)
	}
	var result validationResult
	if e := json.Unmarshal(out.Bytes(), &result); e != nil {
		t.Fatalf("json: %v\n%s", e, out.String())
	}
	if result.Kind != "okf-concept" {
		t.Fatalf("Kind = %q, want okf-concept", result.Kind)
	}
	if result.Summary.Errors != 0 {
		t.Fatalf("Errors = %d, issues=%+v", result.Summary.Errors, result.Issues)
	}
}

func TestValidateOKFConceptMissingTypeFails(t *testing.T) {
	path := writeSpec(t, "metric.md", `---
title: Missing type
---

Body.
`)
	var out, err bytes.Buffer
	e := run([]string{"validate", path}, &out, &err)
	if e == nil || !strings.Contains(e.Error(), "1 error") {
		t.Fatalf("run = %v, stdout=%q stderr=%q", e, out.String(), err.String())
	}
	if !strings.Contains(out.String(), "error type: cannot be empty") {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestValidateOKFConceptCRLF(t *testing.T) {
	// A concept file with Windows CRLF newlines must still be detected as
	// frontmatter and routed to OKF concept validation, not the spec parser.
	path := writeSpec(t, "metric.md", "---\r\ntype: Metric\r\ntitle: CRLF\r\n---\r\n\r\nBody.\r\n")
	var out, err bytes.Buffer
	if e := run([]string{"validate", "-json", path}, &out, &err); e != nil {
		t.Fatalf("run = %v, stderr=%q", e, err.String())
	}
	var result validationResult
	if e := json.Unmarshal(out.Bytes(), &result); e != nil {
		t.Fatalf("json: %v\n%s", e, out.String())
	}
	if result.Kind != "okf-concept" {
		t.Fatalf("Kind = %q, want okf-concept", result.Kind)
	}
}

func writeSpec(t *testing.T, name, text string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(text), 0666); err != nil {
		t.Fatal(err)
	}
	return path
}
