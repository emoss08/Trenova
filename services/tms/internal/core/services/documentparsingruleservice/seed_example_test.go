package documentparsingruleservice

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documentparsingrule"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/stretchr/testify/require"
)

type parsingRuleExampleSeedData struct {
	RuleSet struct {
		Name         string                           `json:"name"`
		DocumentKind documentparsingrule.DocumentKind `json:"documentKind"`
		Priority     int                              `json:"priority"`
	} `json:"ruleSet"`
	Version struct {
		ParserMode   documentparsingrule.ParserMode   `json:"parserMode"`
		MatchConfig  documentparsingrule.MatchConfig  `json:"matchConfig"`
		RuleDocument documentparsingrule.RuleDocument `json:"ruleDocument"`
	} `json:"version"`
	Fixture struct {
		FileName            string                                `json:"fileName"`
		ProviderFingerprint string                                `json:"providerFingerprint"`
		Assertions          documentparsingrule.FixtureAssertions `json:"assertions"`
	} `json:"fixture"`
}

func TestSeededDocumentParsingRuleExampleMatchesFixture(t *testing.T) {
	t.Parallel()

	data := loadParsingRuleExampleSeedDataForTest(t)
	pages, text := loadParsingRuleExamplePagesForTest(t)

	set := &documentparsingrule.RuleSet{
		Name:         data.RuleSet.Name,
		DocumentKind: data.RuleSet.DocumentKind,
		Priority:     data.RuleSet.Priority,
	}
	version := &documentparsingrule.RuleVersion{
		VersionNumber: 1,
		ParserMode:    data.Version.ParserMode,
		MatchConfig:   data.Version.MatchConfig,
		RuleDocument:  data.Version.RuleDocument,
	}
	input := &serviceports.DocumentParsingRuntimeInput{
		DocumentKind:        string(data.RuleSet.DocumentKind),
		FileName:            data.Fixture.FileName,
		ProviderFingerprint: data.Fixture.ProviderFingerprint,
		Text:                text,
		Pages:               pages,
	}

	matched, _, providerMatched := matchesVersion(set, version, input)
	require.True(t, matched)
	require.Equal(t, "CHRobinson", providerMatched)

	analysis, err := evaluateVersion(set, version, input)
	require.NoError(t, err)
	require.NoError(t, validateFixtureAssertions(analysis, data.Fixture.Assertions))
	require.Equal(t, "Ready", analysis.ReviewStatus)
	require.NotEmpty(t, analysis.Fields["referenceNumber"].Value)
	require.NotEmpty(t, analysis.Fields["rate"].Value)
	require.NotEmpty(t, analysis.Fields["equipment"].Value)
	require.Len(t, analysis.Stops, 2)
	require.Equal(t, "pickup", analysis.Stops[0].Role)
	require.Equal(t, "delivery", analysis.Stops[1].Role)
	require.NotEmpty(t, analysis.Stops[0].Date)
	require.NotEmpty(t, analysis.Stops[1].Date)
}

func loadParsingRuleExampleSeedDataForTest(t *testing.T) *parsingRuleExampleSeedData {
	t.Helper()

	loader := seedhelpers.NewDataLoader("../../../infrastructure/database/seeds/development/data")
	data := new(parsingRuleExampleSeedData)
	require.NoError(t, loader.LoadYAML("document_parsing_rule_example.yaml", data))
	return data
}

func loadParsingRuleExamplePagesForTest(t *testing.T) ([]serviceports.DocumentParsingPage, string) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(
		"..",
		"..",
		"..",
		"infrastructure",
		"database",
		"seeds",
		"development",
		"data",
		"document_parsing_rule_example_ch_robinson.txt",
	))
	require.NoError(t, err)

	pagePattern := regexp.MustCompile(`(?m)^--- PAGE (\d+) ---\n`)
	matches := pagePattern.FindAllStringSubmatchIndex(string(raw), -1)
	require.NotEmpty(t, matches)

	pages := make([]serviceports.DocumentParsingPage, 0, len(matches))
	textParts := make([]string, 0, len(matches))
	for idx, match := range matches {
		pageNumber, convErr := strconv.Atoi(string(raw[match[2]:match[3]]))
		require.NoError(t, convErr)

		start := match[1]
		end := len(raw)
		if idx+1 < len(matches) {
			end = matches[idx+1][0]
		}

		pageText := strings.TrimSpace(string(raw[start:end]))
		pages = append(pages, serviceports.DocumentParsingPage{
			PageNumber: pageNumber,
			Text:       pageText,
		})
		textParts = append(textParts, pageText)
	}

	return pages, strings.TrimSpace(strings.Join(textParts, "\n\n"))
}
