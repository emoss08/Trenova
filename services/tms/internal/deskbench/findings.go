package deskbench

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type FindingStatus string

const (
	FindingOpen    = FindingStatus("open")
	FindingFixed   = FindingStatus("fixed")
	FindingWontFix = FindingStatus("wontfix")
)

type FindingSeverity string

const (
	SeverityCritical = FindingSeverity("critical")
	SeverityHigh     = FindingSeverity("high")
	SeverityMedium   = FindingSeverity("medium")
	SeverityLow      = FindingSeverity("low")
)

var severityOrder = []FindingSeverity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow}

type Finding struct {
	ID        string          `yaml:"id"                  json:"id"`
	Title     string          `yaml:"title"               json:"title"`
	Status    FindingStatus   `yaml:"status"              json:"status"`
	Severity  FindingSeverity `yaml:"severity"            json:"severity"`
	Area      string          `yaml:"area"                json:"area"`
	Found     string          `yaml:"found"               json:"found"`
	Models    []string        `yaml:"models,omitempty"    json:"models,omitempty"`
	Scenarios []string        `yaml:"scenarios,omitempty" json:"scenarios,omitempty"`
	Problem   string          `yaml:"problem"             json:"problem"`
	Evidence  string          `yaml:"evidence,omitempty"  json:"evidence,omitempty"`
	Cause     string          `yaml:"cause,omitempty"     json:"cause,omitempty"`
	Fix       string          `yaml:"fix,omitempty"       json:"fix,omitempty"`
	Files     []string        `yaml:"files,omitempty"     json:"files,omitempty"`
	FixedIn   string          `yaml:"fixedIn,omitempty"   json:"fixedIn,omitempty"`
}

type FindingFilter struct {
	Statuses   []FindingStatus
	Severities []FindingSeverity
	Area       string
}

func (f *FindingFilter) keeps(finding *Finding) bool {
	if len(f.Statuses) > 0 && !slices.Contains(f.Statuses, finding.Status) {
		return false
	}
	if len(f.Severities) > 0 && !slices.Contains(f.Severities, finding.Severity) {
		return false
	}
	if f.Area != "" && !strings.Contains(strings.ToLower(finding.Area), strings.ToLower(f.Area)) {
		return false
	}

	return true
}

func LoadFindings(path string) ([]*Finding, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var doc struct {
		Findings []*Finding `yaml:"findings"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err = decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	ids := make(map[string]bool, len(doc.Findings))
	for idx, finding := range doc.Findings {
		if finding == nil {
			return nil, fmt.Errorf("%s: finding %d is empty", path, idx+1)
		}
		if err = finding.validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if ids[finding.ID] {
			return nil, fmt.Errorf("%s: finding %s is listed twice", path, finding.ID)
		}
		ids[finding.ID] = true
	}

	return doc.Findings, nil
}

func (f *Finding) validate() error {
	switch {
	case strings.TrimSpace(f.ID) == "":
		return errors.New("a finding has no id")
	case strings.TrimSpace(f.Title) == "":
		return fmt.Errorf("finding %s has no title", f.ID)
	case strings.TrimSpace(f.Problem) == "":
		return fmt.Errorf("finding %s does not say what the problem is", f.ID)
	case !slices.Contains([]FindingStatus{FindingOpen, FindingFixed, FindingWontFix}, f.Status):
		return fmt.Errorf("finding %s: status must be open, fixed or wontfix, not %q", f.ID, f.Status)
	case !slices.Contains(severityOrder, f.Severity):
		return fmt.Errorf("finding %s: severity must be critical, high, medium or low, not %q", f.ID, f.Severity)
	case f.Status == FindingFixed && strings.TrimSpace(f.Fix) == "":
		return fmt.Errorf("finding %s is fixed but does not say how", f.ID)
	}

	return nil
}

func FilterFindings(findings []*Finding, filter FindingFilter) []*Finding {
	kept := make([]*Finding, 0, len(findings))
	for _, finding := range findings {
		if filter.keeps(finding) {
			kept = append(kept, finding)
		}
	}
	slices.SortStableFunc(kept, func(a, b *Finding) int {
		if a.Status != b.Status {
			return statusRank(a.Status) - statusRank(b.Status)
		}

		return slices.Index(severityOrder, a.Severity) - slices.Index(severityOrder, b.Severity)
	})

	return kept
}

func statusRank(status FindingStatus) int {
	switch status {
	case FindingOpen:
		return 0
	case FindingFixed:
		return 1
	default:
		return 2
	}
}

func RenderFindingsMarkdown(findings []*Finding) string {
	var b strings.Builder
	b.WriteString("# Desk bench findings\n\n")

	counts := make(map[FindingStatus]int, 3)
	for _, finding := range findings {
		counts[finding.Status]++
	}
	fmt.Fprintf(&b, "%d open, %d fixed, %d won't fix.\n\n", counts[FindingOpen], counts[FindingFixed],
		counts[FindingWontFix])

	b.WriteString("| ID | Status | Severity | Area | Title |\n|---|---|---|---|---|\n")
	for _, finding := range findings {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", finding.ID, finding.Status, finding.Severity,
			finding.Area, strings.ReplaceAll(finding.Title, "|", "\\|"))
	}
	b.WriteString("\n")

	for _, finding := range findings {
		fmt.Fprintf(&b, "## %s: %s\n\n", finding.ID, finding.Title)
		fmt.Fprintf(&b, "- Status: %s", finding.Status)
		if finding.FixedIn != "" {
			fmt.Fprintf(&b, " (%s)", finding.FixedIn)
		}
		fmt.Fprintf(&b, "\n- Severity: %s\n- Area: %s\n- Found: %s\n", finding.Severity, finding.Area,
			finding.Found)
		if len(finding.Models) > 0 {
			fmt.Fprintf(&b, "- Models: %s\n", strings.Join(finding.Models, ", "))
		}
		if len(finding.Scenarios) > 0 {
			fmt.Fprintf(&b, "- Scenarios: %s\n", strings.Join(finding.Scenarios, ", "))
		}
		if len(finding.Files) > 0 {
			fmt.Fprintf(&b, "- Files: %s\n", strings.Join(finding.Files, ", "))
		}
		b.WriteString("\n")
		writeSection(&b, "Problem", finding.Problem)
		writeSection(&b, "Evidence", finding.Evidence)
		writeSection(&b, "Cause", finding.Cause)
		writeSection(&b, "Fix", finding.Fix)
	}

	return b.String()
}

func writeSection(b *strings.Builder, heading, body string) {
	if strings.TrimSpace(body) == "" {
		return
	}
	fmt.Fprintf(b, "**%s.** %s\n\n", heading, strings.TrimSpace(body))
}

func RenderFindingsCSV(findings []*Finding) (string, error) {
	var b strings.Builder
	writer := csv.NewWriter(&b)
	if err := writer.Write([]string{
		"id", "status", "severity", "area", "title", "found", "models", "scenarios",
		"problem", "evidence", "cause", "fix", "files", "fixedIn",
	}); err != nil {
		return "", err
	}
	for _, f := range findings {
		if err := writer.Write([]string{
			f.ID, string(f.Status), string(f.Severity), f.Area, f.Title, f.Found,
			strings.Join(f.Models, "; "), strings.Join(f.Scenarios, "; "),
			strings.TrimSpace(f.Problem), strings.TrimSpace(f.Evidence), strings.TrimSpace(f.Cause),
			strings.TrimSpace(f.Fix), strings.Join(f.Files, "; "), f.FixedIn,
		}); err != nil {
			return "", err
		}
	}
	writer.Flush()

	return b.String(), writer.Error()
}

func RenderFindingsJSON(findings []*Finding) (string, error) {
	encoded, err := reportJSON.MarshalIndent(findings, "", "  ")
	if err != nil {
		return "", err
	}

	return string(encoded) + "\n", nil
}
