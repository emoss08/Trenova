package documentparsingruleservice

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/documentparsingrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           *postgres.Connection
	Repo         repositories.DocumentParsingRuleRepository
	Validator    *Validator
	AuditService serviceports.AuditService
}

type Service struct {
	l            *zap.Logger
	db           *postgres.Connection
	repo         repositories.DocumentParsingRuleRepository
	validator    *Validator
	auditService serviceports.AuditService
}

var _ serviceports.DocumentParsingRuleAdminService = (*Service)(nil)
var _ serviceports.DocumentParsingRuleRuntime = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.document-parsing-rule"),
		db:           p.DB,
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) ListRuleSets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	documentKind string,
) ([]*documentparsingrule.RuleSet, error) {
	return s.repo.ListRuleSets(ctx, repositories.ListDocumentParsingRuleSetsRequest{
		TenantInfo:   tenantInfo,
		DocumentKind: strings.TrimSpace(documentKind),
	})
}

func (s *Service) GetRuleSet(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*documentparsingrule.RuleSet, error) {
	return s.repo.GetRuleSet(ctx, repositories.GetDocumentParsingRuleSetRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) CreateRuleSet(
	ctx context.Context,
	entity *documentparsingrule.RuleSet,
	userID pulid.ID,
) (*documentparsingrule.RuleSet, error) {
	entity.PublishedVersionID = nil

	if multiErr := s.validator.ValidateRuleSet(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.CreateRuleSet(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, created, nil, permission.OpCreate, userID, "Document parsing rule set created")
	return created, nil
}

func (s *Service) UpdateRuleSet(
	ctx context.Context,
	entity *documentparsingrule.RuleSet,
	userID pulid.ID,
) (*documentparsingrule.RuleSet, error) {
	original, err := s.GetRuleSet(ctx, entity.ID, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}
	entity.PublishedVersionID = original.PublishedVersionID

	if multiErr := s.validator.ValidateRuleSet(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	updated, err := s.repo.UpdateRuleSet(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, updated, original, permission.OpUpdate, userID, "Document parsing rule set updated")
	return updated, nil
}

func (s *Service) DeleteRuleSet(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) error {
	original, err := s.GetRuleSet(ctx, id, tenantInfo)
	if err != nil {
		return err
	}

	if err = s.repo.DeleteRuleSet(ctx, repositories.GetDocumentParsingRuleSetRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	}); err != nil {
		return err
	}

	s.logAudit(ctx, nil, original, permission.OpDelete, userID, "Document parsing rule set deleted")
	return nil
}

func (s *Service) ListVersions(
	ctx context.Context,
	ruleSetID pulid.ID,
	tenantInfo pagination.TenantInfo,
) ([]*documentparsingrule.RuleVersion, error) {
	return s.repo.ListVersions(ctx, repositories.ListDocumentParsingRuleVersionsRequest{
		RuleSetID:  ruleSetID,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) GetVersion(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*documentparsingrule.RuleVersion, error) {
	return s.repo.GetVersion(ctx, repositories.GetDocumentParsingRuleVersionRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) CreateVersion(
	ctx context.Context,
	entity *documentparsingrule.RuleVersion,
	userID pulid.ID,
) (*documentparsingrule.RuleVersion, error) {
	set, err := s.GetRuleSet(ctx, entity.RuleSetID, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}
	entity.VersionNumber = 0
	entity.Status = documentparsingrule.VersionStatusDraft
	entity.PublishedAt = nil
	entity.PublishedByID = nil
	entity.ValidationSummary = map[string]any{}

	if multiErr := s.validator.ValidateVersion(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.CreateVersion(ctx, entity)
	if err != nil {
		return nil, err
	}

	created.RuleSet = set
	s.logAudit(ctx, created, nil, permission.OpCreate, userID, "Document parsing rule version created")
	return created, nil
}

func (s *Service) UpdateVersion(
	ctx context.Context,
	entity *documentparsingrule.RuleVersion,
	userID pulid.ID,
) (*documentparsingrule.RuleVersion, error) {
	original, _, err := s.repo.GetVersionWithRuleSet(ctx, repositories.GetDocumentParsingRuleVersionRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}
	if original.Status != documentparsingrule.VersionStatusDraft {
		return nil, errortypes.NewBusinessError("Only draft parsing rule versions can be updated")
	}
	entity.Status = original.Status
	entity.VersionNumber = original.VersionNumber
	entity.RuleSetID = original.RuleSetID
	entity.PublishedAt = original.PublishedAt
	entity.PublishedByID = original.PublishedByID
	entity.ValidationSummary = original.ValidationSummary

	if multiErr := s.validator.ValidateVersion(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	updated, err := s.repo.UpdateVersion(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, updated, original, permission.OpUpdate, userID, "Document parsing rule version updated")
	return updated, nil
}

func (s *Service) PublishVersion(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*documentparsingrule.RuleVersion, error) {
	version, set, err := s.repo.GetVersionWithRuleSet(ctx, repositories.GetDocumentParsingRuleVersionRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if version.Status != documentparsingrule.VersionStatusDraft {
		return nil, errortypes.NewBusinessError("Only draft parsing rule versions can be published")
	}

	fixtures, err := s.repo.ListFixtures(ctx, repositories.ListDocumentParsingRuleFixturesRequest{
		RuleSetID:  set.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if len(fixtures) == 0 {
		return nil, errortypes.NewBusinessError("At least one fixture is required before publishing a parsing rule version")
	}

	summary, validationErr := s.validateFixturesAgainstVersion(version, set, fixtures)
	version.ValidationSummary = summary
	if validationErr != nil {
		if _, err = s.repo.UpdateVersion(ctx, version); err != nil {
			s.l.Warn("failed to persist failed validation summary", zap.Error(err))
		}
		return nil, validationErr
	}

	now := time.Now().Unix()
	previousState := jsonutils.MustToJSON(version)
	version.Status = documentparsingrule.VersionStatusPublished
	version.PublishedAt = &now
	version.PublishedByID = &userID

	var updated *documentparsingrule.RuleVersion
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if txErr := s.repo.ArchivePublishedVersions(txCtx, set.ID, tenantInfo.OrgID, tenantInfo.BuID); txErr != nil {
			return txErr
		}

		updated, err = s.repo.UpdateVersion(txCtx, version)
		if err != nil {
			return err
		}

		return s.repo.SetPublishedVersion(txCtx, set.ID, updated.ID, tenantInfo.OrgID, tenantInfo.BuID)
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(err, "The parsing rule is busy. Retry the request.")
	}

	updated.RuleSet = set
	s.logAudit(ctx, updated, previousState, permission.OpActivate, userID, "Document parsing rule version published")
	return updated, nil
}

func (s *Service) ListFixtures(
	ctx context.Context,
	ruleSetID pulid.ID,
	tenantInfo pagination.TenantInfo,
) ([]*documentparsingrule.Fixture, error) {
	return s.repo.ListFixtures(ctx, repositories.ListDocumentParsingRuleFixturesRequest{
		RuleSetID:  ruleSetID,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) GetFixture(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*documentparsingrule.Fixture, error) {
	return s.repo.GetFixture(ctx, repositories.GetDocumentParsingRuleFixtureRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) SaveFixture(
	ctx context.Context,
	entity *documentparsingrule.Fixture,
	userID pulid.ID,
) (*documentparsingrule.Fixture, error) {
	if entity.ID.IsNil() {
		if multiErr := s.validator.ValidateFixture(ctx, entity); multiErr != nil {
			return nil, multiErr
		}

		created, err := s.repo.CreateFixture(ctx, entity)
		if err != nil {
			return nil, err
		}
		s.logAudit(ctx, created, nil, permission.OpCreate, userID, "Document parsing rule fixture created")
		return created, nil
	}

	original, err := s.GetFixture(ctx, entity.ID, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}
	entity.RuleSetID = original.RuleSetID

	if multiErr := s.validator.ValidateFixture(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	updated, err := s.repo.UpdateFixture(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAudit(ctx, updated, original, permission.OpUpdate, userID, "Document parsing rule fixture updated")
	return updated, nil
}

func (s *Service) DeleteFixture(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) error {
	original, err := s.GetFixture(ctx, id, tenantInfo)
	if err != nil {
		return err
	}
	if err = s.repo.DeleteFixture(ctx, repositories.GetDocumentParsingRuleFixtureRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	}); err != nil {
		return err
	}
	s.logAudit(ctx, nil, original, permission.OpDelete, userID, "Document parsing rule fixture deleted")
	return nil
}

func (s *Service) ApplyPublished(
	ctx context.Context,
	input *serviceports.DocumentParsingRuntimeInput,
	baseline *serviceports.DocumentParsingAnalysis,
) (*serviceports.DocumentParsingAnalysis, error) {
	selected, score, providerMatched, err := s.selectPublishedVersion(ctx, input)
	if err != nil || selected == nil {
		return nil, err
	}
	candidate, err := evaluateVersion(selected.RuleSet, selected.Version, input)
	if err != nil {
		return nil, err
	}
	candidate.Metadata = &serviceports.DocumentParsingRuleMetadata{
		RuleSetID:        selected.RuleSet.ID,
		RuleSetName:      selected.RuleSet.Name,
		RuleVersionID:    selected.Version.ID,
		VersionNumber:    selected.Version.VersionNumber,
		ParserMode:       string(selected.Version.ParserMode),
		ProviderMatched:  providerMatched,
		MatchSpecificity: score,
	}
	if selected.Version.ParserMode == documentparsingrule.ParserModeOverrideBase || baseline == nil {
		return candidate, nil
	}
	return mergeAnalyses(baseline, candidate), nil
}

func (s *Service) SimulateVersion(
	ctx context.Context,
	req *serviceports.DocumentParsingSimulationRequest,
) (*serviceports.DocumentParsingSimulationResult, error) {
	version, set, err := s.repo.GetVersionWithRuleSet(ctx, repositories.GetDocumentParsingRuleVersionRequest{
		ID:         req.VersionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	input := &serviceports.DocumentParsingRuntimeInput{
		TenantInfo:          req.TenantInfo,
		DocumentKind:        string(set.DocumentKind),
		FileName:            req.FileName,
		Text:                req.Text,
		Pages:               req.Pages,
		ProviderFingerprint: req.ProviderFingerprint,
	}
	matched, score, providerMatched := matchesVersion(set, version, input)
	result := &serviceports.DocumentParsingSimulationResult{
		Matched:  matched,
		Baseline: req.Baseline,
		Diff:     serviceports.DocumentParsingSimulationDiff{},
		Metadata: &serviceports.DocumentParsingRuleMetadata{
			RuleSetID:        set.ID,
			RuleSetName:      set.Name,
			RuleVersionID:    version.ID,
			VersionNumber:    version.VersionNumber,
			ParserMode:       string(version.ParserMode),
			ProviderMatched:  providerMatched,
			MatchSpecificity: score,
		},
	}
	if !matched {
		result.ValidationErrors = []string{"rule version did not match the provided document input"}
		return result, nil
	}
	candidate, err := evaluateVersion(set, version, input)
	if err != nil {
		return nil, err
	}
	candidate.Metadata = result.Metadata
	if version.ParserMode == documentparsingrule.ParserModeMergeWithBase && req.Baseline != nil {
		candidate = mergeAnalyses(req.Baseline, candidate)
	}
	result.Candidate = candidate
	result.ValidationPassed = true
	result.Diff = diffAnalyses(req.Baseline, candidate)
	return result, nil
}

func (s *Service) selectPublishedVersion(
	ctx context.Context,
	input *serviceports.DocumentParsingRuntimeInput,
) (*repositories.PublishedDocumentParsingRuleVersion, int, string, error) {
	if input == nil || strings.TrimSpace(input.DocumentKind) == "" {
		return nil, 0, "", nil
	}
	published, err := s.repo.ListPublishedVersionsByDocumentKind(ctx, input.TenantInfo, input.DocumentKind)
	if err != nil {
		return nil, 0, "", err
	}
	var selected *repositories.PublishedDocumentParsingRuleVersion
	bestScore := -1
	providerMatched := ""
	for _, candidate := range published {
		matched, score, provider := matchesVersion(candidate.RuleSet, candidate.Version, input)
		if !matched {
			continue
		}
		if score > bestScore {
			selected = candidate
			bestScore = score
			providerMatched = provider
			continue
		}
		if score == bestScore && selected != nil {
			if candidate.RuleSet.Priority > selected.RuleSet.Priority {
				selected = candidate
				providerMatched = provider
			}
		}
	}
	return selected, bestScore, providerMatched, nil
}

func (s *Service) validateFixturesAgainstVersion(
	version *documentparsingrule.RuleVersion,
	set *documentparsingrule.RuleSet,
	fixtures []*documentparsingrule.Fixture,
) (map[string]any, error) {
	failures := make([]map[string]any, 0)
	for _, fixture := range fixtures {
		input := &serviceports.DocumentParsingRuntimeInput{
			DocumentKind:        string(set.DocumentKind),
			FileName:            fixture.FileName,
			Text:                fixture.TextSnapshot,
			Pages:               fixturePages(fixture),
			ProviderFingerprint: fixture.ProviderFingerprint,
		}
		matched, _, _ := matchesVersion(set, version, input)
		if !matched {
			failures = append(failures, map[string]any{
				"fixtureId": fixture.ID.String(),
				"name":      fixture.Name,
				"error":     "fixture did not match rule conditions",
			})
			continue
		}
		analysis, err := evaluateVersion(set, version, input)
		if err != nil {
			failures = append(failures, map[string]any{
				"fixtureId": fixture.ID.String(),
				"name":      fixture.Name,
				"error":     err.Error(),
			})
			continue
		}
		if assertionErr := validateFixtureAssertions(analysis, fixture.Assertions); assertionErr != nil {
			failures = append(failures, map[string]any{
				"fixtureId": fixture.ID.String(),
				"name":      fixture.Name,
				"error":     assertionErr.Error(),
			})
		}
	}

	summary := map[string]any{
		"fixtureCount": len(fixtures),
		"failures":     failures,
	}
	if len(failures) > 0 {
		return summary, errortypes.NewBusinessError("Fixture validation failed for the parsing rule version")
	}
	return summary, nil
}

func (s *Service) logAudit(
	ctx context.Context,
	current any,
	previous any,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
) {
	currentState := map[string]any{}
	previousState := map[string]any{}
	orgID := pulid.Nil
	buID := pulid.Nil

	if entity, ok := current.(interface{ GetOrganizationID() pulid.ID }); ok {
		orgID = entity.GetOrganizationID()
	}
	if entity, ok := current.(interface{ GetBusinessUnitID() pulid.ID }); ok {
		buID = entity.GetBusinessUnitID()
	}
	if current == nil && previous != nil {
		if entity, ok := previous.(interface{ GetOrganizationID() pulid.ID }); ok {
			orgID = entity.GetOrganizationID()
		}
		if entity, ok := previous.(interface{ GetBusinessUnitID() pulid.ID }); ok {
			buID = entity.GetBusinessUnitID()
		}
	}
	if current != nil {
		currentState = jsonutils.MustToJSON(current)
	}
	if previous != nil {
		switch entity := previous.(type) {
		case map[string]any:
			previousState = entity
		default:
			previousState = jsonutils.MustToJSON(previous)
		}
	}

	resourceID := ""
	switch entity := current.(type) {
	case interface{ GetID() pulid.ID }:
		resourceID = entity.GetID().String()
	case nil:
		if entity, ok := previous.(interface{ GetID() pulid.ID }); ok {
			resourceID = entity.GetID().String()
		}
	}

	if err := s.auditService.LogAction(
		&serviceports.LogActionParams{
			Resource:       permission.ResourceDocumentParsingRule,
			ResourceID:     resourceID,
			Operation:      operation,
			UserID:         userID,
			CurrentState:   currentState,
			PreviousState:  previousState,
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		auditservice.WithComment(comment),
		auditservice.WithDiff(previous, current),
	); err != nil {
		s.l.Warn("failed to log document parsing rule audit entry", zap.Error(err))
	}
}

func fixturePages(fixture *documentparsingrule.Fixture) []serviceports.DocumentParsingPage {
	pages := make([]serviceports.DocumentParsingPage, 0, len(fixture.PageSnapshots))
	for _, page := range fixture.PageSnapshots {
		pages = append(pages, serviceports.DocumentParsingPage{
			PageNumber: page.PageNumber,
			Text:       page.Text,
		})
	}
	if len(pages) == 0 {
		pages = append(pages, serviceports.DocumentParsingPage{
			PageNumber: 1,
			Text:       fixture.TextSnapshot,
		})
	}
	return pages
}

func validateFixtureAssertions(
	analysis *serviceports.DocumentParsingAnalysis,
	assertions documentparsingrule.FixtureAssertions,
) error {
	if analysis == nil {
		return fmt.Errorf("analysis is empty")
	}
	for key, expected := range assertions.ExpectedFields {
		if err := validateFieldAssertion(
			analysis,
			key,
			documentparsingrule.FixtureFieldAssertion{
				Operator: documentparsingrule.FixtureFieldAssertionOperatorEquals,
				Value:    expected,
			},
		); err != nil {
			return err
		}
	}
	for key, fieldAssertions := range assertions.FieldAssertions {
		for _, fieldAssertion := range fieldAssertions {
			if err := validateFieldAssertion(analysis, key, fieldAssertion); err != nil {
				return err
			}
		}
	}
	if assertions.MinimumStopCount > 0 && len(analysis.Stops) < assertions.MinimumStopCount {
		return fmt.Errorf("expected at least %d stops, got %d", assertions.MinimumStopCount, len(analysis.Stops))
	}
	for _, role := range assertions.RequiredStopRoles {
		if !analysisHasStopRole(analysis, role) {
			return fmt.Errorf("expected stop role %q was not extracted", role)
		}
	}
	if status := strings.TrimSpace(assertions.ReviewStatus); status != "" && status != analysis.ReviewStatus {
		return fmt.Errorf("expected review status %q, got %q", status, analysis.ReviewStatus)
	}
	return nil
}

func validateFieldAssertion(
	analysis *serviceports.DocumentParsingAnalysis,
	key string,
	assertion documentparsingrule.FixtureFieldAssertion,
) error {
	field, ok := analysis.Fields[key]
	if !ok {
		return fmt.Errorf("expected field %q was not extracted", key)
	}

	value := strings.TrimSpace(field.Value)
	switch assertion.Operator {
	case documentparsingrule.FixtureFieldAssertionOperatorExists:
		return nil
	case documentparsingrule.FixtureFieldAssertionOperatorNotEmpty:
		if value == "" {
			return fmt.Errorf("expected field %q to be non-empty", key)
		}
		return nil
	case documentparsingrule.FixtureFieldAssertionOperatorEquals:
		expected := strings.TrimSpace(assertion.Value)
		if !strings.EqualFold(value, expected) {
			return fmt.Errorf("field %q mismatch: expected %q, got %q", key, expected, field.Value)
		}
		return nil
	case documentparsingrule.FixtureFieldAssertionOperatorMatchesRegex:
		pattern := strings.TrimSpace(assertion.Pattern)
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("field %q has invalid assertion regex: %w", key, err)
		}
		if !re.MatchString(value) {
			return fmt.Errorf("field %q value %q did not match regex %q", key, field.Value, pattern)
		}
		return nil
	case documentparsingrule.FixtureFieldAssertionOperatorOneOf:
		for _, candidate := range assertion.Values {
			if strings.EqualFold(value, strings.TrimSpace(candidate)) {
				return nil
			}
		}
		return fmt.Errorf("field %q value %q did not match any allowed values", key, field.Value)
	default:
		return fmt.Errorf("field %q has unsupported assertion operator %q", key, assertion.Operator)
	}
}

func analysisHasStopRole(analysis *serviceports.DocumentParsingAnalysis, role string) bool {
	for _, stop := range analysis.Stops {
		if strings.EqualFold(stop.Role, role) {
			return true
		}
	}
	return false
}

func diffAnalyses(
	baseline *serviceports.DocumentParsingAnalysis,
	candidate *serviceports.DocumentParsingAnalysis,
) serviceports.DocumentParsingSimulationDiff {
	diff := serviceports.DocumentParsingSimulationDiff{
		AddedFields:      []string{},
		ChangedFields:    []string{},
		AddedStopRoles:   []string{},
		ChangedStopRoles: []string{},
	}
	if candidate == nil {
		return diff
	}
	if baseline == nil {
		for key := range candidate.Fields {
			diff.AddedFields = append(diff.AddedFields, key)
		}
		for _, stop := range candidate.Stops {
			diff.AddedStopRoles = append(diff.AddedStopRoles, stop.Role)
		}
		diff.AddedFields = dedupeStrings(diff.AddedFields)
		diff.AddedStopRoles = dedupeStrings(diff.AddedStopRoles)
		return diff
	}
	for key, field := range candidate.Fields {
		existing, ok := baseline.Fields[key]
		if !ok {
			diff.AddedFields = append(diff.AddedFields, key)
			continue
		}
		if existing.Value != field.Value {
			diff.ChangedFields = append(diff.ChangedFields, key)
		}
	}
	for _, stop := range candidate.Stops {
		baseStop, ok := findStopByRole(baseline.Stops, stop.Role)
		if !ok {
			diff.AddedStopRoles = append(diff.AddedStopRoles, stop.Role)
			continue
		}
		if stop.Name != baseStop.Name || stop.AddressLine1 != baseStop.AddressLine1 || stop.Date != baseStop.Date || stop.TimeWindow != baseStop.TimeWindow {
			diff.ChangedStopRoles = append(diff.ChangedStopRoles, stop.Role)
		}
	}
	diff.AddedFields = dedupeStrings(diff.AddedFields)
	diff.ChangedFields = dedupeStrings(diff.ChangedFields)
	diff.AddedStopRoles = dedupeStrings(diff.AddedStopRoles)
	diff.ChangedStopRoles = dedupeStrings(diff.ChangedStopRoles)
	return diff
}

func findStopByRole(stops []serviceports.DocumentParsingStop, role string) (serviceports.DocumentParsingStop, bool) {
	for _, stop := range stops {
		if strings.EqualFold(stop.Role, role) {
			return stop, true
		}
	}
	return serviceports.DocumentParsingStop{}, false
}

func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || slices.Contains(out, item) {
			continue
		}
		out = append(out, item)
	}
	return out
}
