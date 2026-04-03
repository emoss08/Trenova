package documentparsingruleservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/documentparsingrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestEvaluateVersionExtractsFieldsAndStops(t *testing.T) {
	set := &documentparsingrule.RuleSet{
		ID:           pulid.MustNew("dprs_"),
		Name:         "Rate Confirmation Layout",
		DocumentKind: documentparsingrule.DocumentKindRateConfirmation,
		Priority:     200,
	}
	version := &documentparsingrule.RuleVersion{
		ID:            pulid.MustNew("dprv_"),
		VersionNumber: 2,
		ParserMode:    documentparsingrule.ParserModeMergeWithBase,
		MatchConfig: documentparsingrule.MatchConfig{
			ProviderFingerprints: []string{"GenericBroker"},
			RequiresAll:          []string{"rate confirmation"},
		},
		RuleDocument: documentparsingrule.RuleDocument{
			Sections: []documentparsingrule.SectionRule{
				{Name: "shipper_block", StartAnchors: []string{"shipper #1"}, AllowMultiple: true},
				{Name: "receiver_block", StartAnchors: []string{"receiver #1"}, AllowMultiple: true},
			},
			Fields: []documentparsingrule.FieldRule{
				{Key: "rate", Label: "Rate", Aliases: []string{"line haul"}, Normalizer: "currency", Required: true, Confidence: 0.91},
			},
			Stops: []documentparsingrule.StopRule{
				{
					Role:         "pickup",
					Required:     true,
					SectionNames: []string{"shipper_block"},
					Extractors: []documentparsingrule.StopFieldRule{
						{FieldKey: "name", Aliases: []string{"shipper #1"}, Confidence: 0.9},
						{FieldKey: "addressLine1", Patterns: []string{`(?im)shipper #1\s+(.+)\n([0-9]{1,6}.+)`}, Confidence: 0.9},
						{FieldKey: "date", Aliases: []string{"pickup date"}, Confidence: 0.88},
					},
				},
				{
					Role:         "delivery",
					Required:     true,
					SectionNames: []string{"receiver_block"},
					Extractors: []documentparsingrule.StopFieldRule{
						{FieldKey: "name", Aliases: []string{"receiver #1"}, Confidence: 0.9},
						{FieldKey: "addressLine1", Patterns: []string{`(?im)receiver #1\s+(.+)\n([0-9]{1,6}.+)`}, Confidence: 0.9},
						{FieldKey: "date", Aliases: []string{"delivery date"}, Confidence: 0.88},
					},
				},
			},
		},
	}
	input := &serviceports.DocumentParsingRuntimeInput{
		DocumentKind:        "RateConfirmation",
		FileName:            "rate_confirmation.pdf",
		ProviderFingerprint: "GenericBroker",
		Text:                "Rate Confirmation\nLine Haul: $1,250.00\nSHIPPER #1 Alpha Foods\n123 Market St\nDallas, TX 75201\nPickup Date: 04/10/2026\nRECEIVER #1 Beta Stores\n890 Harbor Rd\nAtlanta, GA 30301\nDelivery Date: 04/11/2026",
		Pages: []serviceports.DocumentParsingPage{
			{
				PageNumber: 1,
				Text:       "Rate Confirmation\nLine Haul: $1,250.00\nSHIPPER #1 Alpha Foods\n123 Market St\nDallas, TX 75201\nPickup Date: 04/10/2026\nRECEIVER #1 Beta Stores\n890 Harbor Rd\nAtlanta, GA 30301\nDelivery Date: 04/11/2026",
			},
		},
	}

	analysis, err := evaluateVersion(set, version, input)
	require.NoError(t, err)
	require.Equal(t, "Ready", analysis.ReviewStatus)
	require.Equal(t, "$1250.00", analysis.Fields["rate"].Value)
	require.Len(t, analysis.Stops, 2)
	require.Equal(t, "pickup", analysis.Stops[0].Role)
	require.Equal(t, "delivery", analysis.Stops[1].Role)
	require.Equal(t, "Dallas", analysis.Stops[0].City)
	require.Equal(t, "Atlanta", analysis.Stops[1].City)
}

func TestMergeAnalysesPrefersCandidateWhenBaselineIsIncomplete(t *testing.T) {
	baseline := &serviceports.DocumentParsingAnalysis{
		Fields: map[string]serviceports.DocumentParsingField{
			"rate": {Key: "rate", Label: "Rate", Value: "", Confidence: 0.2, ReviewRequired: true},
		},
		Stops:             []serviceports.DocumentParsingStop{},
		Conflicts:         []serviceports.DocumentParsingConflict{},
		MissingFields:     []string{"Rate", "Pickup Stop"},
		Signals:           []string{"baseline"},
		ReviewStatus:      "NeedsReview",
		OverallConfidence: 0.55,
	}
	candidate := &serviceports.DocumentParsingAnalysis{
		Fields: map[string]serviceports.DocumentParsingField{
			"rate": {Key: "rate", Label: "Rate", Value: "$800.00", Confidence: 0.9},
		},
		Stops: []serviceports.DocumentParsingStop{
			{Role: "pickup", Name: "Origin", Confidence: 0.91},
		},
		Conflicts:         []serviceports.DocumentParsingConflict{},
		MissingFields:     []string{},
		Signals:           []string{"candidate"},
		ReviewStatus:      "Ready",
		OverallConfidence: 0.91,
	}

	merged := mergeAnalyses(baseline, candidate)
	require.Equal(t, "$800.00", merged.Fields["rate"].Value)
	require.Len(t, merged.Stops, 1)
	require.Equal(t, "Ready", merged.ReviewStatus)
	require.NotContains(t, merged.MissingFields, "Rate")
	require.NotContains(t, merged.MissingFields, "Pickup Stop")
}

func TestValidateFixturesAgainstVersionFailsOnAssertionMismatch(t *testing.T) {
	svc := &Service{}
	set := &documentparsingrule.RuleSet{
		ID:           pulid.MustNew("dprs_"),
		Name:         "Fixture Rule",
		DocumentKind: documentparsingrule.DocumentKindRateConfirmation,
	}
	version := &documentparsingrule.RuleVersion{
		ID:            pulid.MustNew("dprv_"),
		VersionNumber: 1,
		ParserMode:    documentparsingrule.ParserModeOverrideBase,
		RuleDocument: documentparsingrule.RuleDocument{
			Fields: []documentparsingrule.FieldRule{
				{Key: "rate", Label: "Rate", Aliases: []string{"rate"}, Normalizer: "currency", Required: true},
			},
		},
	}
	fixture := &documentparsingrule.Fixture{
		ID:           pulid.MustNew("dprf_"),
		RuleSetID:    set.ID,
		Name:         "bad expectation",
		TextSnapshot: "Rate: $100.00",
		Assertions: documentparsingrule.FixtureAssertions{
			ExpectedFields: map[string]string{"rate": "$200.00"},
		},
	}

	summary, err := svc.validateFixturesAgainstVersion(version, set, []*documentparsingrule.Fixture{fixture})
	require.Error(t, err)
	require.NotNil(t, summary["failures"])
}

func TestValidateFixtureAssertionsSupportsPatternAndNotEmpty(t *testing.T) {
	analysis := &serviceports.DocumentParsingAnalysis{
		Fields: map[string]serviceports.DocumentParsingField{
			"referenceNumber": {
				Key:   "referenceNumber",
				Label: "Reference Number",
				Value: "123456789",
			},
			"equipment": {
				Key:   "equipment",
				Label: "Equipment",
				Value: "Van - Min L=53",
			},
		},
		Stops: []serviceports.DocumentParsingStop{
			{Role: "pickup", Date: "7/13/21"},
			{Role: "delivery", Date: "7/15/21"},
		},
		ReviewStatus: "Ready",
	}

	err := validateFixtureAssertions(analysis, documentparsingrule.FixtureAssertions{
		FieldAssertions: map[string][]documentparsingrule.FixtureFieldAssertion{
			"referenceNumber": {
				{
					Operator: documentparsingrule.FixtureFieldAssertionOperatorMatchesRegex,
					Pattern:  `^[0-9]+$`,
				},
			},
			"equipment": {
				{
					Operator: documentparsingrule.FixtureFieldAssertionOperatorNotEmpty,
				},
			},
		},
		RequiredStopRoles: []string{"pickup", "delivery"},
		MinimumStopCount:  2,
		ReviewStatus:      "Ready",
	})
	require.NoError(t, err)
}

func TestValidateFixtureAssertionsFailsWhenPatternDoesNotMatch(t *testing.T) {
	analysis := &serviceports.DocumentParsingAnalysis{
		Fields: map[string]serviceports.DocumentParsingField{
			"referenceNumber": {
				Key:   "referenceNumber",
				Label: "Reference Number",
				Value: "ABC-123",
			},
		},
	}

	err := validateFixtureAssertions(analysis, documentparsingrule.FixtureAssertions{
		FieldAssertions: map[string][]documentparsingrule.FixtureFieldAssertion{
			"referenceNumber": {
				{
					Operator: documentparsingrule.FixtureFieldAssertionOperatorMatchesRegex,
					Pattern:  `^[0-9]+$`,
				},
			},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "did not match regex")
}

func TestServiceApplyPublishedSelectsMostSpecificRule(t *testing.T) {
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	genericSet := &documentparsingrule.RuleSet{
		ID:           pulid.MustNew("dprs_"),
		Name:         "Generic",
		DocumentKind: documentparsingrule.DocumentKindRateConfirmation,
		Priority:     100,
	}
	specificSet := &documentparsingrule.RuleSet{
		ID:           pulid.MustNew("dprs_"),
		Name:         "Specific",
		DocumentKind: documentparsingrule.DocumentKindRateConfirmation,
		Priority:     100,
	}
	genericVersion := &documentparsingrule.RuleVersion{
		ID:            pulid.MustNew("dprv_"),
		VersionNumber: 1,
		ParserMode:    documentparsingrule.ParserModeOverrideBase,
		MatchConfig: documentparsingrule.MatchConfig{
			RequiresAll: []string{"rate confirmation"},
		},
		RuleDocument: documentparsingrule.RuleDocument{
			Fields: []documentparsingrule.FieldRule{
				{Key: "referenceNumber", Label: "Reference", Aliases: []string{"reference"}, Required: true},
			},
		},
	}
	specificVersion := &documentparsingrule.RuleVersion{
		ID:            pulid.MustNew("dprv_"),
		VersionNumber: 2,
		ParserMode:    documentparsingrule.ParserModeOverrideBase,
		MatchConfig: documentparsingrule.MatchConfig{
			ProviderFingerprints: []string{"GenericBroker"},
			RequiresAll:          []string{"rate confirmation"},
		},
		RuleDocument: documentparsingrule.RuleDocument{
			Fields: []documentparsingrule.FieldRule{
				{Key: "rate", Label: "Rate", Aliases: []string{"line haul"}, Normalizer: "currency", Required: true},
			},
		},
	}

	svc := &Service{
		repo: stubDocumentParsingRuleRepo{
			listPublishedVersionsByDocumentKindFn: func(_ context.Context, _ pagination.TenantInfo, _ string) ([]*repositories.PublishedDocumentParsingRuleVersion, error) {
				return []*repositories.PublishedDocumentParsingRuleVersion{
					{RuleSet: genericSet, Version: genericVersion},
					{RuleSet: specificSet, Version: specificVersion},
				}, nil
			},
		},
	}

	result, err := svc.ApplyPublished(context.Background(), &serviceports.DocumentParsingRuntimeInput{
		TenantInfo:          tenantInfo,
		DocumentKind:        "RateConfirmation",
		FileName:            "rate_confirmation.pdf",
		Text:                "Rate Confirmation\nLine Haul: $500.00\nReference: REF-1",
		ProviderFingerprint: "GenericBroker",
		Pages: []serviceports.DocumentParsingPage{
			{PageNumber: 1, Text: "Rate Confirmation\nLine Haul: $500.00\nReference: REF-1"},
		},
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "Specific", result.Metadata.RuleSetName)
	require.Equal(t, "$500.00", result.Fields["rate"].Value)
}

type stubDocumentParsingRuleRepo struct {
	getRuleSetFn                          func(context.Context, repositories.GetDocumentParsingRuleSetRequest) (*documentparsingrule.RuleSet, error)
	createRuleSetFn                       func(context.Context, *documentparsingrule.RuleSet) (*documentparsingrule.RuleSet, error)
	updateRuleSetFn                       func(context.Context, *documentparsingrule.RuleSet) (*documentparsingrule.RuleSet, error)
	listPublishedVersionsByDocumentKindFn func(context.Context, pagination.TenantInfo, string) ([]*repositories.PublishedDocumentParsingRuleVersion, error)
	getFixtureFn                          func(context.Context, repositories.GetDocumentParsingRuleFixtureRequest) (*documentparsingrule.Fixture, error)
	createFixtureFn                       func(context.Context, *documentparsingrule.Fixture) (*documentparsingrule.Fixture, error)
	updateFixtureFn                       func(context.Context, *documentparsingrule.Fixture) (*documentparsingrule.Fixture, error)
}

func (s stubDocumentParsingRuleRepo) ListRuleSets(context.Context, repositories.ListDocumentParsingRuleSetsRequest) ([]*documentparsingrule.RuleSet, error) {
	panic("not implemented")
}

func (s stubDocumentParsingRuleRepo) GetRuleSet(ctx context.Context, req repositories.GetDocumentParsingRuleSetRequest) (*documentparsingrule.RuleSet, error) {
	if s.getRuleSetFn == nil {
		panic("not implemented")
	}
	return s.getRuleSetFn(ctx, req)
}

func (s stubDocumentParsingRuleRepo) CreateRuleSet(ctx context.Context, entity *documentparsingrule.RuleSet) (*documentparsingrule.RuleSet, error) {
	if s.createRuleSetFn == nil {
		panic("not implemented")
	}
	return s.createRuleSetFn(ctx, entity)
}

func (s stubDocumentParsingRuleRepo) UpdateRuleSet(ctx context.Context, entity *documentparsingrule.RuleSet) (*documentparsingrule.RuleSet, error) {
	if s.updateRuleSetFn == nil {
		panic("not implemented")
	}
	return s.updateRuleSetFn(ctx, entity)
}

func (s stubDocumentParsingRuleRepo) DeleteRuleSet(context.Context, repositories.GetDocumentParsingRuleSetRequest) error {
	panic("not implemented")
}

func (s stubDocumentParsingRuleRepo) ListVersions(context.Context, repositories.ListDocumentParsingRuleVersionsRequest) ([]*documentparsingrule.RuleVersion, error) {
	panic("not implemented")
}

func (s stubDocumentParsingRuleRepo) GetVersion(context.Context, repositories.GetDocumentParsingRuleVersionRequest) (*documentparsingrule.RuleVersion, error) {
	panic("not implemented")
}

func (s stubDocumentParsingRuleRepo) GetVersionWithRuleSet(context.Context, repositories.GetDocumentParsingRuleVersionRequest) (*documentparsingrule.RuleVersion, *documentparsingrule.RuleSet, error) {
	panic("not implemented")
}

func (s stubDocumentParsingRuleRepo) CreateVersion(context.Context, *documentparsingrule.RuleVersion) (*documentparsingrule.RuleVersion, error) {
	panic("not implemented")
}

func (s stubDocumentParsingRuleRepo) UpdateVersion(context.Context, *documentparsingrule.RuleVersion) (*documentparsingrule.RuleVersion, error) {
	panic("not implemented")
}

func (s stubDocumentParsingRuleRepo) ArchivePublishedVersions(context.Context, pulid.ID, pulid.ID, pulid.ID) error {
	panic("not implemented")
}

func (s stubDocumentParsingRuleRepo) SetPublishedVersion(context.Context, pulid.ID, pulid.ID, pulid.ID, pulid.ID) error {
	panic("not implemented")
}

func (s stubDocumentParsingRuleRepo) NextVersionNumber(context.Context, pulid.ID, pulid.ID, pulid.ID) (int, error) {
	panic("not implemented")
}

func (s stubDocumentParsingRuleRepo) ListPublishedVersionsByDocumentKind(ctx context.Context, tenantInfo pagination.TenantInfo, documentKind string) ([]*repositories.PublishedDocumentParsingRuleVersion, error) {
	return s.listPublishedVersionsByDocumentKindFn(ctx, tenantInfo, documentKind)
}

func (s stubDocumentParsingRuleRepo) ListFixtures(context.Context, repositories.ListDocumentParsingRuleFixturesRequest) ([]*documentparsingrule.Fixture, error) {
	panic("not implemented")
}

func (s stubDocumentParsingRuleRepo) GetFixture(ctx context.Context, req repositories.GetDocumentParsingRuleFixtureRequest) (*documentparsingrule.Fixture, error) {
	if s.getFixtureFn == nil {
		panic("not implemented")
	}
	return s.getFixtureFn(ctx, req)
}

func (s stubDocumentParsingRuleRepo) CreateFixture(ctx context.Context, entity *documentparsingrule.Fixture) (*documentparsingrule.Fixture, error) {
	if s.createFixtureFn == nil {
		panic("not implemented")
	}
	return s.createFixtureFn(ctx, entity)
}

func (s stubDocumentParsingRuleRepo) UpdateFixture(ctx context.Context, entity *documentparsingrule.Fixture) (*documentparsingrule.Fixture, error) {
	if s.updateFixtureFn == nil {
		panic("not implemented")
	}
	return s.updateFixtureFn(ctx, entity)
}

func (s stubDocumentParsingRuleRepo) DeleteFixture(context.Context, repositories.GetDocumentParsingRuleFixtureRequest) error {
	panic("not implemented")
}

func TestCreateRuleSetClearsPublishedVersionID(t *testing.T) {
	t.Parallel()

	var saved *documentparsingrule.RuleSet
	svc := &Service{
		repo: stubDocumentParsingRuleRepo{
			createRuleSetFn: func(_ context.Context, entity *documentparsingrule.RuleSet) (*documentparsingrule.RuleSet, error) {
				saved = entity
				return entity, nil
			},
		},
		validator:    NewValidator(),
		auditService: noopAuditService{},
	}
	publishedVersionID := pulid.MustNew("dprv_")
	entity := &documentparsingrule.RuleSet{
		ID:                 pulid.MustNew("dprs_"),
		OrganizationID:     pulid.MustNew("org_"),
		BusinessUnitID:     pulid.MustNew("bu_"),
		Name:               "RC Rules",
		DocumentKind:       documentparsingrule.DocumentKindRateConfirmation,
		Priority:           100,
		PublishedVersionID: &publishedVersionID,
	}

	created, err := svc.CreateRuleSet(t.Context(), entity, pulid.MustNew("usr_"))
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotNil(t, saved)
	require.Nil(t, saved.PublishedVersionID)
}

func TestUpdateRuleSetPreservesPublishedVersionID(t *testing.T) {
	t.Parallel()

	preservedVersionID := pulid.MustNew("dprv_")
	svc := &Service{
		repo: stubDocumentParsingRuleRepo{
			getRuleSetFn: func(_ context.Context, _ repositories.GetDocumentParsingRuleSetRequest) (*documentparsingrule.RuleSet, error) {
				return &documentparsingrule.RuleSet{
					ID:                 pulid.MustNew("dprs_"),
					OrganizationID:     pulid.MustNew("org_"),
					BusinessUnitID:     pulid.MustNew("bu_"),
					Name:               "Existing",
					DocumentKind:       documentparsingrule.DocumentKindRateConfirmation,
					Priority:           100,
					PublishedVersionID: &preservedVersionID,
				}, nil
			},
			updateRuleSetFn: func(_ context.Context, entity *documentparsingrule.RuleSet) (*documentparsingrule.RuleSet, error) {
				return entity, nil
			},
		},
		validator:    NewValidator(),
		auditService: noopAuditService{},
	}
	incomingVersionID := pulid.MustNew("dprv_")
	entity := &documentparsingrule.RuleSet{
		ID:                 pulid.MustNew("dprs_"),
		OrganizationID:     pulid.MustNew("org_"),
		BusinessUnitID:     pulid.MustNew("bu_"),
		Name:               "Updated",
		DocumentKind:       documentparsingrule.DocumentKindRateConfirmation,
		Priority:           200,
		PublishedVersionID: &incomingVersionID,
	}

	updated, err := svc.UpdateRuleSet(t.Context(), entity, pulid.MustNew("usr_"))
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.NotNil(t, updated.PublishedVersionID)
	require.Equal(t, preservedVersionID, *updated.PublishedVersionID)
}

func TestSaveFixtureUpdateUsesStoredRuleSetIDBeforeValidation(t *testing.T) {
	t.Parallel()

	ruleSetID := pulid.MustNew("dprs_")
	fixtureID := pulid.MustNew("dprf_")
	svc := &Service{
		repo: stubDocumentParsingRuleRepo{
			getFixtureFn: func(_ context.Context, _ repositories.GetDocumentParsingRuleFixtureRequest) (*documentparsingrule.Fixture, error) {
				return &documentparsingrule.Fixture{
					ID:             fixtureID,
					RuleSetID:      ruleSetID,
					OrganizationID: pulid.MustNew("org_"),
					BusinessUnitID: pulid.MustNew("bu_"),
					Name:           "Fixture",
					TextSnapshot:   "Rate Confirmation",
				}, nil
			},
			updateFixtureFn: func(_ context.Context, entity *documentparsingrule.Fixture) (*documentparsingrule.Fixture, error) {
				return entity, nil
			},
		},
		validator:    NewValidator(),
		auditService: noopAuditService{},
	}
	entity := &documentparsingrule.Fixture{
		ID:             fixtureID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Updated Fixture",
		TextSnapshot:   "Rate Confirmation\nLine Haul: $100.00",
		Assertions: documentparsingrule.FixtureAssertions{
			ExpectedFields: map[string]string{"rate": "$100.00"},
		},
	}

	updated, err := svc.SaveFixture(t.Context(), entity, pulid.MustNew("usr_"))
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, ruleSetID, updated.RuleSetID)
}

type noopAuditService struct{}

func (noopAuditService) List(context.Context, *repositories.ListAuditEntriesRequest) (*pagination.ListResult[*audit.Entry], error) {
	return nil, nil
}

func (noopAuditService) ListByResourceID(context.Context, *repositories.ListByResourceIDRequest) (*pagination.ListResult[*audit.Entry], error) {
	return nil, nil
}

func (noopAuditService) GetByID(context.Context, repositories.GetAuditEntryByIDOptions) (*audit.Entry, error) {
	return nil, nil
}

func (noopAuditService) LogAction(*serviceports.LogActionParams, ...serviceports.LogOption) error {
	return nil
}

func (noopAuditService) LogActions([]serviceports.BulkLogEntry) error {
	return nil
}

func (noopAuditService) RegisterSensitiveFields(permission.Resource, []serviceports.SensitiveField) error {
	return nil
}
