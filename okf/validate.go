package okf

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tmc/specmd/validation"
)

// ValidateConcept checks the hard OKF v0.1 concept document rules.
func ValidateConcept(concept *Concept) error {
	if concept == nil {
		return fmt.Errorf("validate okf concept: nil concept")
	}
	if err := ValidateConceptReport(concept).Err(); err != nil {
		return fmt.Errorf("validate okf concept: %w", err)
	}
	return nil
}

// ValidateConceptReport checks an OKF concept and returns errors plus warnings.
func ValidateConceptReport(concept *Concept) validation.Report {
	if concept == nil {
		return validation.New([]validation.Issue{{Level: validation.LevelError, Path: "concept", Message: "cannot be nil"}})
	}
	var issues []validation.Issue
	if strings.TrimSpace(concept.ID) == "" {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: "id", Message: "cannot be empty"})
	}
	if strings.TrimSpace(concept.Type) == "" {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: "type", Message: "cannot be empty"})
	}
	if strings.TrimSpace(concept.Title) == "" {
		issues = append(issues, validation.Issue{Level: validation.LevelInfo, Path: "title", Message: "recommended field is missing"})
	}
	if strings.TrimSpace(concept.Description) == "" {
		issues = append(issues, validation.Issue{Level: validation.LevelInfo, Path: "description", Message: "recommended field is missing"})
	}
	return validation.New(issues)
}

// ValidateBundle checks the hard OKF v0.1 bundle rules.
func ValidateBundle(bundle *Bundle) error {
	if bundle == nil {
		return fmt.Errorf("validate okf bundle: nil bundle")
	}
	if err := ValidateBundleReport(bundle).Err(); err != nil {
		return fmt.Errorf("validate okf bundle: %w", err)
	}
	return nil
}

// ValidateBundleReport checks an OKF bundle and returns errors plus warnings.
func ValidateBundleReport(bundle *Bundle) validation.Report {
	if bundle == nil {
		return validation.New([]validation.Issue{{Level: validation.LevelError, Path: "bundle", Message: "cannot be nil"}})
	}
	var issues []validation.Issue
	for i := range bundle.Concepts {
		report := ValidateConceptReport(&bundle.Concepts[i])
		issues = append(issues, validation.Prefix("concepts."+bundle.Concepts[i].ID, report.Issues)...)
	}
	for _, bad := range bundle.Invalid {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: "concepts." + bad.ID, Message: "unparseable frontmatter: " + bad.Err.Error()})
	}
	for i := range bundle.Index {
		issues = append(issues, validateIndexIssues(fmt.Sprintf("index[%d]", i), bundle.Index[i])...)
	}
	for i := range bundle.Logs {
		issues = append(issues, validateLogIssues(fmt.Sprintf("logs[%d]", i), bundle.Logs[i])...)
	}
	return validation.New(issues)
}

func validateIndexIssues(path string, file ReservedFile) []validation.Issue {
	var issues []validation.Issue
	if len(file.FrontMatter) > 0 && !file.Root {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: path, Message: "frontmatter is only permitted in root index.md"})
	}
	if !hasMarkdownHeading(file.Body, "# ") {
		issues = append(issues, validation.Issue{Level: validation.LevelWarning, Path: path, Message: "should contain at least one section heading"})
	}
	if !hasMarkdownListLink(file.Body) {
		issues = append(issues, validation.Issue{Level: validation.LevelWarning, Path: path, Message: "should contain at least one linked list entry"})
	}
	return issues
}

func validateLogIssues(path string, file ReservedFile) []validation.Issue {
	var issues []validation.Issue
	if len(file.FrontMatter) > 0 {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: path, Message: "log.md must not contain frontmatter"})
	}
	if !hasMarkdownHeading(file.Body, "# ") {
		issues = append(issues, validation.Issue{Level: validation.LevelWarning, Path: path, Message: "should contain a title heading"})
	}
	for _, bad := range malformedDateHeadings(file.Body) {
		issues = append(issues, validation.Issue{Level: validation.LevelError, Path: path, Message: "date heading must use ISO 8601 form: " + bad})
	}
	return issues
}

func hasMarkdownHeading(text, prefix string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return true
		}
	}
	return false
}

func hasMarkdownListLink(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if (strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "- ")) && strings.Contains(line, "](") {
			return true
		}
	}
	return false
}

var isoDateHeadingRE = regexp.MustCompile(`^## [0-9]{4}-[0-9]{2}-[0-9]{2}$`)
var dateLikeHeadingRE = regexp.MustCompile(`^## [0-9]{4}[-/][0-9]{1,2}[-/][0-9]{1,2}$`)

func malformedDateHeadings(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if dateLikeHeadingRE.MatchString(line) && !isoDateHeadingRE.MatchString(line) {
			out = append(out, strings.TrimSpace(strings.TrimPrefix(line, "##")))
		}
	}
	return out
}
