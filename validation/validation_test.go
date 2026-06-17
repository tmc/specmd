package validation

import "testing"

func TestNewSummary(t *testing.T) {
	report := New([]Issue{
		{Level: LevelError},
		{Level: LevelWarning},
		{Level: LevelInfo},
	})
	if report.Valid {
		t.Fatal("Valid = true, want false")
	}
	if report.Summary.Errors != 1 || report.Summary.Warnings != 1 || report.Summary.Info != 1 {
		t.Fatalf("Summary = %+v", report.Summary)
	}
}

func TestNewValidWhenNoErrors(t *testing.T) {
	report := New([]Issue{{Level: LevelWarning}, {Level: LevelInfo}})
	if !report.Valid {
		t.Fatalf("Valid = false, want true: %+v", report)
	}
}

func TestPrefix(t *testing.T) {
	issues := Prefix("concepts.sales", []Issue{
		{Level: LevelError, Path: "type", Message: "cannot be empty"},
		{Level: LevelInfo, Message: "no path"},
	})
	if issues[0].Path != "concepts.sales.type" {
		t.Fatalf("Path = %q, want concepts.sales.type", issues[0].Path)
	}
	if issues[1].Path != "concepts.sales" {
		t.Fatalf("Path = %q, want concepts.sales", issues[1].Path)
	}
}

func TestReportErr(t *testing.T) {
	report := New([]Issue{
		{Level: LevelError, Path: "name", Message: "cannot be empty"},
		{Level: LevelWarning, Path: "overview", Message: "too brief"},
	})
	err := report.Err()
	if err == nil {
		t.Fatal("Err = nil, want error")
	}
	if got := err.Error(); got != "name: cannot be empty" {
		t.Fatalf("Err = %q, want %q", got, "name: cannot be empty")
	}
}
