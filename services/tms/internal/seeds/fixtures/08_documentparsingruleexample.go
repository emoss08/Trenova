package fixtures

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentparsingrule"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentparsingrule/documentparsingruledb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	parsingRuleExampleFile      = "data/document_parsing_rule_example.yaml"
	parsingRuleExamplePagesFile = "data/document_parsing_rule_example_ch_robinson.txt"
)

type DocumentParsingRuleExampleSeed struct {
}

var DocumentParsingRuleExample = Seeds.Add(seed.Seed{
	Name:        "document_parsing_rule_example",
	Kind:        seed.Fixture,
	Description: "An example parsing rule set, its published version and a fixture document",
	After:       []*seed.Seed{AdminAccount},
	Inputs:      seed.Files(data, parsingRuleExampleFile, parsingRuleExamplePagesFile),
	Run:         (&DocumentParsingRuleExampleSeed{}).Run,
})

func (s *DocumentParsingRuleExampleSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	data, err := loadDocumentParsingRuleExampleData(sr)
	if err != nil {
		return err
	}

	pages, combinedText, err := loadDocumentParsingRuleExamplePages(sr)
	if err != nil {
		return err
	}

	ruleSetID := pulid.MustNew("dprs_")
	versionID := pulid.MustNew("dprv_")
	fixtureID := pulid.MustNew("dprf_")
	now := timeutils.NowUnix()

	ruleSet := &documentparsingrule.RuleSet{
		ID:             ruleSetID,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Name:           data.RuleSet.Name,
		Description:    data.RuleSet.Description,
		DocumentKind:   data.RuleSet.DocumentKind,
		Priority:       data.RuleSet.Priority,
	}
	if err = documentparsingruledb.RuleSets.Create(ctx, db, ruleSet); err != nil {
		return fmt.Errorf("insert parsing rule set: %w", err)
	}

	version := &documentparsingrule.RuleVersion{
		ID:                versionID,
		RuleSetID:         ruleSet.ID,
		OrganizationID:    org.ID,
		BusinessUnitID:    org.BusinessUnitID,
		VersionNumber:     1,
		Status:            documentparsingrule.VersionStatusPublished,
		Label:             data.Version.Label,
		ParserMode:        data.Version.ParserMode,
		MatchConfig:       data.Version.MatchConfig,
		RuleDocument:      data.Version.RuleDocument,
		ValidationSummary: map[string]any{"fixtureCount": 1, "failures": []map[string]any{}},
		PublishedAt:       &now,
	}
	if err = documentparsingruledb.RuleVersions.Create(ctx, db, version); err != nil {
		return fmt.Errorf("insert parsing rule version: %w", err)
	}

	ruleSet.PublishedVersionID = &version.ID
	rs := documentparsingruledb.RuleSets
	if err = rs.Save(ctx, db, ruleSet, rs.PublishedVersionID); err != nil {
		return fmt.Errorf("set published parsing rule version: %w", err)
	}

	fixture := &documentparsingrule.Fixture{
		ID:                  fixtureID,
		RuleSetID:           ruleSet.ID,
		OrganizationID:      org.ID,
		BusinessUnitID:      org.BusinessUnitID,
		Name:                data.Fixture.Name,
		Description:         data.Fixture.Description,
		FileName:            data.Fixture.FileName,
		ProviderFingerprint: data.Fixture.ProviderFingerprint,
		TextSnapshot:        combinedText,
		PageSnapshots:       pages,
		Assertions:          data.Fixture.Assertions,
	}
	if err = documentparsingruledb.Fixtures.Create(ctx, db, fixture); err != nil {
		return fmt.Errorf("insert parsing rule fixture: %w", err)
	}

	return nil
}

type documentParsingRuleExampleSeedData struct {
	RuleSet struct {
		Name         string                           `json:"name"`
		Description  string                           `json:"description"`
		DocumentKind documentparsingrule.DocumentKind `json:"documentKind"`
		Priority     int                              `json:"priority"`
	} `json:"ruleSet"`
	Version struct {
		Label        string                           `json:"label"`
		ParserMode   documentparsingrule.ParserMode   `json:"parserMode"`
		MatchConfig  documentparsingrule.MatchConfig  `json:"matchConfig"`
		RuleDocument documentparsingrule.RuleDocument `json:"ruleDocument"`
	} `json:"version"`
	Fixture struct {
		Name                string                                `json:"name"`
		Description         string                                `json:"description"`
		FileName            string                                `json:"fileName"`
		ProviderFingerprint string                                `json:"providerFingerprint"`
		Assertions          documentparsingrule.FixtureAssertions `json:"assertions"`
	} `json:"fixture"`
}

func loadDocumentParsingRuleExampleData(sr *seed.Run) (*documentParsingRuleExampleSeedData, error) {
	data := new(documentParsingRuleExampleSeedData)
	if err := yamlAsJSON(sr, parsingRuleExampleFile, data); err != nil {
		return nil, fmt.Errorf("load document parsing rule example seed data: %w", err)
	}
	return data, nil
}

func loadDocumentParsingRuleExamplePages(
	sr *seed.Run,
) ([]documentparsingrule.PageSnapshot, string, error) {
	raw, err := sr.ReadFile(parsingRuleExamplePagesFile)
	if err != nil {
		return nil, "", fmt.Errorf("read parsing rule example fixture text: %w", err)
	}

	pagePattern := regexp.MustCompile(`(?m)^--- PAGE (\d+) ---\n`)
	matches := pagePattern.FindAllStringSubmatchIndex(string(raw), -1)
	if len(matches) == 0 {
		return nil, "", fmt.Errorf("fixture text file does not contain page markers")
	}

	pages := make([]documentparsingrule.PageSnapshot, 0, len(matches))
	textParts := make([]string, 0, len(matches))
	for idx, match := range matches {
		pageNumber, convErr := strconv.Atoi(string(raw[match[2]:match[3]]))
		if convErr != nil {
			return nil, "", fmt.Errorf("parse page number: %w", convErr)
		}
		start := match[1]
		end := len(raw)
		if idx+1 < len(matches) {
			end = matches[idx+1][0]
		}
		pageText := strings.TrimSpace(string(raw[start:end]))
		pages = append(pages, documentparsingrule.PageSnapshot{
			PageNumber: pageNumber,
			Text:       pageText,
		})
		textParts = append(textParts, pageText)
	}

	return pages, strings.TrimSpace(strings.Join(textParts, "\n\n")), nil
}
