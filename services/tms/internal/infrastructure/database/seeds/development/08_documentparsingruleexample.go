package development

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documentparsingrule"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type DocumentParsingRuleExampleSeed struct {
	seedhelpers.BaseSeed
}

func NewDocumentParsingRuleExampleSeed() *DocumentParsingRuleExampleSeed {
	seed := &DocumentParsingRuleExampleSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"DocumentParsingRuleExample",
		"1.0.0",
		"Creates DocumentParsingRuleExample data",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)
	seed.SetDependencies(seedhelpers.SeedAdminAccount)
	return seed
}

func (s *DocumentParsingRuleExampleSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetOrganization("default_org")
			if err != nil {
				org, err = sc.GetDefaultOrganization(ctx)
				if err != nil {
					return fmt.Errorf("get organization: %w", err)
				}
			}

			data, err := loadDocumentParsingRuleExampleData()
			if err != nil {
				return err
			}

			exists, err := tx.NewSelect().
				Model((*documentparsingrule.RuleSet)(nil)).
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Where("name = ?", data.RuleSet.Name).
				Exists(ctx)
			if err != nil {
				return fmt.Errorf("check existing parsing rule example: %w", err)
			}
			if exists {
				return nil
			}
			pages, combinedText, err := loadDocumentParsingRuleExamplePages()
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
			if _, err = tx.NewInsert().Model(ruleSet).Exec(ctx); err != nil {
				return fmt.Errorf("insert parsing rule set: %w", err)
			}
			if err = sc.TrackCreated(ctx, "document_parsing_rule_sets", ruleSet.ID, s.Name()); err != nil {
				return fmt.Errorf("track parsing rule set: %w", err)
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
			if _, err = tx.NewInsert().Model(version).Exec(ctx); err != nil {
				return fmt.Errorf("insert parsing rule version: %w", err)
			}
			if err = sc.TrackCreated(ctx, "document_parsing_rule_versions", version.ID, s.Name()); err != nil {
				return fmt.Errorf("track parsing rule version: %w", err)
			}

			ruleSet.PublishedVersionID = &version.ID
			if _, err = tx.NewUpdate().
				Model(ruleSet).
				Column("published_version_id", "updated_at").
				WherePK().
				Exec(ctx); err != nil {
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
			if _, err = tx.NewInsert().Model(fixture).Exec(ctx); err != nil {
				return fmt.Errorf("insert parsing rule fixture: %w", err)
			}
			if err = sc.TrackCreated(ctx, "document_parsing_rule_fixtures", fixture.ID, s.Name()); err != nil {
				return fmt.Errorf("track parsing rule fixture: %w", err)
			}

			return nil
		},
	)
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

func loadDocumentParsingRuleExampleData() (*documentParsingRuleExampleSeedData, error) {
	loader := seedhelpers.NewDataLoader("./internal/infrastructure/database/seeds/development/data")
	data := new(documentParsingRuleExampleSeedData)
	if err := loader.LoadYAML("document_parsing_rule_example.yaml", data); err != nil {
		return nil, fmt.Errorf("load document parsing rule example seed data: %w", err)
	}
	return data, nil
}

func loadDocumentParsingRuleExamplePages() ([]documentparsingrule.PageSnapshot, string, error) {
	path := filepath.Join(
		".",
		"internal",
		"infrastructure",
		"database",
		"seeds",
		"development",
		"data",
		"document_parsing_rule_example_ch_robinson.txt",
	)
	raw, err := os.ReadFile(path)
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

func (s *DocumentParsingRuleExampleSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *DocumentParsingRuleExampleSeed) CanRollback() bool {
	return true
}
