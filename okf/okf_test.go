package okf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmc/specmd/validation"
)

func TestParseOKFBundle(t *testing.T) {
	bundle, err := ParseBundle("testdata/okf")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Concepts) != 2 {
		t.Fatalf("len(Concepts) = %d, want 2", len(bundle.Concepts))
	}
	if len(bundle.Index) != 1 || len(bundle.Logs) != 1 {
		t.Fatalf("reserved files: index=%d logs=%d", len(bundle.Index), len(bundle.Logs))
	}
	if bundle.Version != "0.1" {
		t.Fatalf("Version = %q, want 0.1", bundle.Version)
	}
	c := bundle.Concepts[0]
	if c.ID != "datasets/sales" {
		t.Fatalf("first concept ID = %q, want datasets/sales", c.ID)
	}
	if c.Type != "BigQuery Dataset" || c.Title != "Sales" {
		t.Fatalf("concept = %+v", c)
	}
	if got, want := strings.Join(c.Tags, ","), "sales"; got != want {
		t.Fatalf("Tags = %q, want %q", got, want)
	}
	if c.Metadata.SourcePath == "" {
		t.Fatal("SourcePath is empty")
	}
}

func TestValidateOKFReservedFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"index.md":        "# Catalog\n\n* [Good](good.md) - good concept\n",
		"log.md":          "# Directory Update Log\n\n## 2026-05-28\n\n* **Creation**: Created bundle.\n",
		"good.md":         "---\ntype: Metric\ntitle: Good\ndescription: Good metric.\n---\n\nBody.\n",
		"nested/index.md": "---\nokf_version: \"0.1\"\n---\n\n# Nested\n\nNo entries.\n",
	}
	for name, text := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	bundle, err := ParseBundle(dir)
	if err != nil {
		t.Fatal(err)
	}
	report := ValidateBundleReport(bundle)
	for _, want := range []string{
		"frontmatter is only permitted in root index.md",
		"should contain at least one linked list entry",
	} {
		if !hasIssue(report, want) {
			t.Fatalf("missing issue %q: %+v", want, report.Issues)
		}
	}
	if report.Summary.Errors != 1 {
		t.Fatalf("Errors = %d, want 1: %+v", report.Summary.Errors, report.Issues)
	}
	if report.Summary.Warnings == 0 {
		t.Fatalf("Warnings = 0, want at least one: %+v", report.Issues)
	}
}

func TestValidateOKFLogDateHeadingFormat(t *testing.T) {
	report := ValidateBundleReport(&Bundle{
		Logs: []ReservedFile{{
			Name: "log.md",
			Body: "# Directory Update Log\n\n## 2026/5/28\n\n* Created.\n",
		}},
	})
	if !hasIssue(report, "date heading must use ISO 8601 form") {
		t.Fatalf("missing malformed date issue: %+v", report.Issues)
	}
	if report.Summary.Errors != 1 {
		t.Fatalf("Errors = %d, want 1: %+v", report.Summary.Errors, report.Issues)
	}
}

func TestParseOKFConcept(t *testing.T) {
	concept, err := ParseConcept("metrics/weekly_active_users", strings.NewReader(`---
type: Metric
title: Weekly active users
description: Count of unique active users per week.
tags: [growth, engagement]
extra: kept
---

# Definition

Users active in a seven-day window.
`))
	if err != nil {
		t.Fatal(err)
	}
	if concept.Type != "Metric" {
		t.Fatalf("Type = %q, want Metric", concept.Type)
	}
	if got, want := strings.Join(concept.Tags, ","), "growth,engagement"; got != want {
		t.Fatalf("Tags = %q, want %q", got, want)
	}
	if len(concept.FrontMatter) != 5 {
		t.Fatalf("len(FrontMatter) = %d, want 5", len(concept.FrontMatter))
	}
}

func TestParseOKFConceptBlockListTags(t *testing.T) {
	concept, err := ParseConcept("metrics/wau", strings.NewReader(`---
type: Metric
title: Weekly active users
tags:
  - growth
  - "engagement"
---

Body.
`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(concept.Tags, ","), "growth,engagement"; got != want {
		t.Fatalf("Tags = %q, want %q", got, want)
	}
	if len(concept.FrontMatter) != 3 {
		t.Fatalf("len(FrontMatter) = %d, want 3: %+v", len(concept.FrontMatter), concept.FrontMatter)
	}
}

func TestParseOKFBundleTolerantOfBadConcept(t *testing.T) {
	dir := t.TempDir()
	good := "---\ntype: Metric\ntitle: Good\n---\n\nBody.\n"
	bad := "---\ntype: Metric\nno colon here\n---\n\nBody.\n"
	if err := os.WriteFile(filepath.Join(dir, "good.md"), []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.md"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	bundle, err := ParseBundle(dir)
	if err != nil {
		t.Fatalf("ParseBundle returned error for a malformed concept: %v", err)
	}
	if len(bundle.Concepts) != 1 || bundle.Concepts[0].ID != "good" {
		t.Fatalf("Concepts = %+v, want one with ID good", bundle.Concepts)
	}
	if len(bundle.Invalid) != 1 || bundle.Invalid[0].ID != "bad" {
		t.Fatalf("Invalid = %+v, want one with ID bad", bundle.Invalid)
	}
	report := ValidateBundleReport(bundle)
	if report.Valid || report.Summary.Errors == 0 {
		t.Fatalf("bundle with a bad concept should report errors: %+v", report.Issues)
	}
}

func TestParseOKFConceptFrontmatterErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"missing", "# No frontmatter", "missing frontmatter"},
		{"unterminated", "---\ntype: Metric\n", "unterminated frontmatter"},
		{"bad field", "---\ntype Metric\n---\n", "missing colon"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConcept("x", strings.NewReader(tt.input))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ParseConcept error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateOKFConceptReport(t *testing.T) {
	concept := &Concept{ID: "metrics/wau", Type: "Metric"}
	report := ValidateConceptReport(concept)
	if !report.Valid {
		t.Fatalf("Valid = false: %+v", report.Issues)
	}
	if report.Summary.Info != 2 {
		t.Fatalf("Info = %d, want 2: %+v", report.Summary.Info, report.Issues)
	}
	if err := ValidateConcept(concept); err != nil {
		t.Fatal(err)
	}
}

func TestValidateOKFConceptRequiresType(t *testing.T) {
	report := ValidateConceptReport(&Concept{ID: "metrics/wau"})
	if report.Valid {
		t.Fatalf("Valid = true, want false")
	}
	if report.Summary.Errors != 1 {
		t.Fatalf("Errors = %d, want 1: %+v", report.Summary.Errors, report.Issues)
	}
}

func hasIssue(report validation.Report, msg string) bool {
	for _, issue := range report.Issues {
		if strings.Contains(issue.Message, msg) {
			return true
		}
	}
	return false
}
