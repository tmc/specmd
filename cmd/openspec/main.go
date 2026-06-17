package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/specmd/internal/lsp"
	"github.com/tmc/specmd/okf"
	"github.com/tmc/specmd/openspec"
	"github.com/tmc/specmd/validation"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		usage(stderr)
		return fmt.Errorf("missing command")
	}
	switch args[0] {
	case "help", "-h", "--help":
		usage(stdout)
		return nil
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "lsp":
		return runLSP(args[1:])
	default:
		usage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: openspec command [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  validate [path]   validate an openspec or OKF path")
	fmt.Fprintln(w, "  lsp               run stdio language server")
}

func runValidate(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "write validation report as json")
	strict := fs.Bool("strict", false, "fail when warnings or info issues are present")
	if err := fs.Parse(args); err != nil {
		return err
	}
	path := "."
	switch fs.NArg() {
	case 0:
	case 1:
		path = fs.Arg(0)
	default:
		return fmt.Errorf("validate: too many arguments")
	}
	result, err := validatePath(path)
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return fmt.Errorf("validate: %w", err)
		}
	} else {
		printValidation(stdout, result)
	}
	if result.Summary.Errors > 0 {
		return fmt.Errorf("validate: %d error(s)", result.Summary.Errors)
	}
	if *strict && (result.Summary.Warnings > 0 || result.Summary.Info > 0) {
		return fmt.Errorf("validate: %d warning(s), %d info", result.Summary.Warnings, result.Summary.Info)
	}
	return nil
}

func runLSP(args []string) error {
	fs := flag.NewFlagSet("lsp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("lsp: too many arguments")
	}
	if err := lsp.NewServer(os.Stdin, os.Stdout).Run(); err != nil {
		return fmt.Errorf("lsp: %w", err)
	}
	return nil
}

type validationResult struct {
	Path    string            `json:"path"`
	Kind    string            `json:"kind"`
	Valid   bool              `json:"valid"`
	Issues  []validationIssue `json:"issues,omitempty"`
	Summary validationSummary `json:"summary"`
	report  validation.Report
}

type validationIssue struct {
	Level   string `json:"level"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type validationSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

func validatePath(path string) (validationResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return validationResult{}, fmt.Errorf("validate: %w", err)
	}
	if info.IsDir() {
		if fileExists(filepath.Join(path, "proposal.md")) {
			return validateChangeDir(path)
		}
		if fileExists(filepath.Join(path, "openspec")) {
			return validateProject(filepath.Join(path, "openspec"))
		}
		if fileExists(filepath.Join(path, "specs")) || fileExists(filepath.Join(path, "changes")) {
			return validateProject(path)
		}
		return validateOKFBundle(path)
	}
	if strings.EqualFold(filepath.Base(path), "spec.md") {
		spec, err := openspec.ParseSpecFile(path)
		if err != nil {
			return validationResult{}, fmt.Errorf("validate: %w", err)
		}
		return result(path, "spec", openspec.ValidateSpecReport(spec)), nil
	}
	if hasFrontMatter(path) {
		return validateOKFConceptFile(path)
	}
	spec, err := openspec.ParseSpecFile(path)
	if err == nil {
		report := openspec.ValidateSpecReport(spec)
		return result(path, "spec", report), nil
	}
	return validateOKFConceptFile(path)
}

func validateProject(path string) (validationResult, error) {
	project, err := openspec.ParseProject(path)
	if err != nil {
		return validationResult{}, fmt.Errorf("validate: %w", err)
	}
	var issues []validation.Issue
	for i := range project.Specs {
		report := openspec.ValidateSpecReport(&project.Specs[i])
		issues = append(issues, validation.Prefix("specs."+project.Specs[i].Name, report.Issues)...)
	}
	for i := range project.Changes {
		report := openspec.ValidateChangeReport(&project.Changes[i])
		issues = append(issues, validation.Prefix("changes."+project.Changes[i].Name, report.Issues)...)
	}
	return result(path, "project", validation.New(issues)), nil
}

func validateChangeDir(path string) (validationResult, error) {
	change, err := openspec.ParseChangeDir(path)
	if err != nil {
		return validationResult{}, fmt.Errorf("validate: %w", err)
	}
	report := openspec.ValidateChangeReport(change)
	return result(path, "change", report), nil
}

func validateOKFBundle(path string) (validationResult, error) {
	bundle, err := okf.ParseBundle(path)
	if err != nil {
		return validationResult{}, fmt.Errorf("validate: %w", err)
	}
	report := okf.ValidateBundleReport(bundle)
	return result(path, "okf-bundle", report), nil
}

func validateOKFConceptFile(path string) (validationResult, error) {
	concept, err := okf.ParseConceptFile(path)
	if err != nil {
		return validationResult{}, fmt.Errorf("validate: %w", err)
	}
	report := okf.ValidateConceptReport(concept)
	return result(path, "okf-concept", report), nil
}

func result(path, kind string, report validation.Report) validationResult {
	return validationResult{
		Path:    path,
		Kind:    kind,
		Valid:   report.Valid,
		Issues:  jsonIssues(report.Issues),
		Summary: jsonSummary(report.Summary),
		report:  report,
	}
}

func jsonIssues(issues []validation.Issue) []validationIssue {
	out := make([]validationIssue, len(issues))
	for i, issue := range issues {
		out[i] = validationIssue{
			Level:   strings.ToLower(string(issue.Level)),
			Path:    issue.Path,
			Message: issue.Message,
		}
	}
	return out
}

func jsonSummary(summary validation.Summary) validationSummary {
	return validationSummary{Errors: summary.Errors, Warnings: summary.Warnings, Info: summary.Info}
}

func printValidation(w io.Writer, result validationResult) {
	fmt.Fprintf(w, "%s: %s", result.Kind, result.Path)
	if result.Valid {
		fmt.Fprintln(w, ": valid")
	} else {
		fmt.Fprintln(w, ": invalid")
	}
	for _, issue := range result.Issues {
		fmt.Fprintf(w, "%s %s: %s\n", issue.Level, issue.Path, issue.Message)
	}
	fmt.Fprintf(w, "summary: %d error(s), %d warning(s), %d info\n", result.Summary.Errors, result.Summary.Warnings, result.Summary.Info)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasFrontMatter(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	line, err := bufio.NewReader(f).ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}
	return strings.TrimRight(line, "\r\n") == "---"
}
