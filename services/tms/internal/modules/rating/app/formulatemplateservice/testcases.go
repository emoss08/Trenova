package formulatemplateservice

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/contextvariablecache"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var defaultTestCaseTolerance = decimal.NewFromFloat(0.01)

const (
	maxScenarioVariableNameLength = 100
	maxScenarioVariableTextLength = 1000
)

type TestCaseInput struct {
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	Variables      map[string]any     `json:"variables"`
	ExpectedAmount decimal.Decimal    `json:"expectedAmount"`
	Tolerance      decimal.Decimal    `json:"tolerance"`
	Candidate      *TestCaseCandidate `json:"candidate,omitempty"`
}

type CreateTestCaseRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TemplateID pulid.ID              `json:"-"`
	TestCaseInput
}

type UpdateTestCaseRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TemplateID pulid.ID              `json:"-"`
	TestCaseID pulid.ID              `json:"-"`
	TestCaseInput
	Version int64 `json:"version"`
}

func (s *Service) ListTestCases(
	ctx context.Context,
	req formulatemplateports.ListTestCasesRequest,
) ([]*formulatemplate.TestCase, error) {
	return s.testCaseRepo.ListByTemplate(ctx, req)
}

func (s *Service) CreateTestCase(
	ctx context.Context,
	req *CreateTestCaseRequest,
) (*formulatemplate.TestCase, error) {
	log := s.l.With(
		zap.String("operation", "CreateTestCase"),
		zap.String("templateID", req.TemplateID.String()),
	)

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get template", zap.Error(err))
		return nil, err
	}

	entity := buildTestCaseEntity(template.ID, req.TenantInfo, &req.TestCaseInput)

	if vErr := validateTestCaseWithCandidate(entity, req.Candidate); vErr != nil {
		return nil, vErr
	}

	entity.Record(
		recordedContent(template, req.Candidate),
		s.recordingVersion(ctx, template, req.TenantInfo),
		req.TenantInfo.UserID,
		timeutils.NowUnix(),
	)

	created, err := s.testCaseRepo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create test case", zap.Error(err))
		return nil, err
	}

	s.logAuditAction(
		ctx,
		log,
		template,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		nil,
		fmt.Sprintf("Test scenario %q added", created.Name),
		auditservice.WithMetadata(map[string]any{"scenarioAfter": scenarioRecord(created)}),
	)

	return s.reloadTestCase(ctx, created, req.TenantInfo), nil
}

func (s *Service) UpdateTestCase(
	ctx context.Context,
	req *UpdateTestCaseRequest,
) (*formulatemplate.TestCase, error) {
	log := s.l.With(
		zap.String("operation", "UpdateTestCase"),
		zap.String("testCaseID", req.TestCaseID.String()),
	)

	existing, err := s.testCaseRepo.GetByID(ctx, formulatemplateports.GetTestCaseByIDRequest{
		TenantInfo: req.TenantInfo,
		TestCaseID: req.TestCaseID,
		TemplateID: req.TemplateID,
	})
	if err != nil {
		log.Error("failed to get test case", zap.Error(err))
		return nil, err
	}

	before := scenarioRecord(existing)

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Variables = req.Variables
	existing.ExpectedAmount = req.ExpectedAmount
	existing.Tolerance = req.Tolerance
	existing.Version = req.Version
	if existing.Variables == nil {
		existing.Variables = map[string]any{}
	}
	if existing.Tolerance.IsZero() {
		existing.Tolerance = defaultTestCaseTolerance
	}

	if vErr := validateTestCaseWithCandidate(existing, req.Candidate); vErr != nil {
		return nil, vErr
	}

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get template", zap.Error(err))
		return nil, err
	}

	if req.Candidate != nil {
		existing.Record(
			recordedContent(template, req.Candidate),
			s.recordingVersion(ctx, template, req.TenantInfo),
			req.TenantInfo.UserID,
			timeutils.NowUnix(),
		)
	}

	updated, err := s.testCaseRepo.Update(ctx, existing)
	if err != nil {
		log.Error("failed to update test case", zap.Error(err))
		return nil, err
	}

	s.logAuditAction(
		ctx,
		log,
		template,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		nil,
		fmt.Sprintf("Test scenario %q updated", updated.Name),
		auditservice.WithMetadata(map[string]any{
			"scenarioBefore": before,
			"scenarioAfter":  scenarioRecord(updated),
		}),
	)

	return s.reloadTestCase(ctx, updated, req.TenantInfo), nil
}

func (s *Service) DeleteTestCase(
	ctx context.Context,
	req formulatemplateports.GetTestCaseByIDRequest,
) error {
	log := s.l.With(
		zap.String("operation", "DeleteTestCase"),
		zap.String("testCaseID", req.TestCaseID.String()),
	)

	existing, err := s.testCaseRepo.GetByID(ctx, req)
	if err != nil {
		log.Error("failed to get test case", zap.Error(err))
		return err
	}

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get template", zap.Error(err))
		return err
	}

	if err = s.testCaseRepo.Delete(ctx, req); err != nil {
		log.Error("failed to delete test case", zap.Error(err))
		return err
	}

	s.logAuditAction(
		ctx,
		log,
		template,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		nil,
		fmt.Sprintf("Test scenario %q deleted", existing.Name),
		auditservice.WithMetadata(map[string]any{"scenarioBefore": scenarioRecord(existing)}),
	)

	return nil
}

func scenarioRecord(testCase *formulatemplate.TestCase) map[string]any {
	return map[string]any{
		"id":             testCase.ID.String(),
		"name":           testCase.Name,
		"description":    testCase.Description,
		"variables":      maps.Clone(testCase.Variables),
		"expectedAmount": testCase.ExpectedAmount.String(),
		"tolerance":      testCase.Tolerance.String(),
	}
}

// buildTestCaseEntity is the single place a scenario's storage shape and
// defaults are decided, so the authoring path and the import path cannot
// drift apart.
func buildTestCaseEntity(
	templateID pulid.ID,
	tenantInfo pagination.TenantInfo,
	input *TestCaseInput,
) *formulatemplate.TestCase {
	entity := &formulatemplate.TestCase{
		TemplateID:     templateID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           input.Name,
		Description:    input.Description,
		Variables:      input.Variables,
		ExpectedAmount: input.ExpectedAmount,
		Tolerance:      input.Tolerance,
		CreatedByID:    tenantInfo.UserID,
	}
	if entity.Variables == nil {
		entity.Variables = map[string]any{}
	}
	if entity.Tolerance.IsZero() {
		entity.Tolerance = defaultTestCaseTolerance
	}

	return entity
}

func validateTestCase(entity *formulatemplate.TestCase) error {
	return validateTestCaseWithCandidate(entity, nil)
}

func validateTestCaseWithCandidate(
	entity *formulatemplate.TestCase,
	candidate *TestCaseCandidate,
) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	ValidateScenarioContent(multiErr, "candidate", candidate)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) reloadTestCase(
	ctx context.Context,
	entity *formulatemplate.TestCase,
	tenantInfo pagination.TenantInfo,
) *formulatemplate.TestCase {
	reloaded, err := s.testCaseRepo.GetByID(ctx, formulatemplateports.GetTestCaseByIDRequest{
		TenantInfo: tenantInfo,
		TestCaseID: entity.ID,
		TemplateID: entity.TemplateID,
	})
	if err != nil {
		s.l.Warn("failed to reload test case after saving it",
			zap.String("testCaseID", entity.ID.String()),
			zap.Error(err),
		)
		return entity
	}

	return reloaded
}

func (s *Service) recordingVersion(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
) int64 {
	if s.latestApprovedVersionNumber(ctx, template, tenantInfo) == 0 {
		return 0
	}

	return template.CurrentVersionNumber
}

func recordedContent(
	template *formulatemplate.FormulaTemplate,
	candidate *TestCaseCandidate,
) *formulatemplate.ScenarioContent {
	if candidate == nil {
		return template.ScenarioContent()
	}

	content := *candidate
	if content.VariableDefinitions == nil {
		content.VariableDefinitions = []*formulatypes.VariableDefinition{}
	}
	if content.BreakdownDefinitions == nil {
		content.BreakdownDefinitions = []*formulatypes.BreakdownDefinition{}
	}

	return &content
}

func ValidateScenarioContent(
	multiErr *errortypes.MultiError,
	prefix string,
	content *formulatemplate.ScenarioContent,
) {
	if content == nil {
		return
	}

	if utf8.RuneCountInString(content.Expression) > formulatemplate.MaxExpressionLength {
		multiErr.Add(
			scenarioField(prefix, "expression"),
			errortypes.ErrInvalid,
			"Expression cannot exceed 10,000 characters",
		)
	}
	if len(content.VariableDefinitions) > formulatemplate.MaxVariableDefinitions {
		multiErr.Add(
			scenarioField(prefix, "variableDefinitions"),
			errortypes.ErrInvalid,
			"A template cannot declare more than 50 custom variables",
		)
	}
	if content.MinCharge.Valid && content.MinCharge.Decimal.IsNegative() {
		multiErr.Add(
			scenarioField(prefix, "minCharge"),
			errortypes.ErrInvalid,
			"Minimum charge cannot be negative",
		)
	}
	if content.MaxCharge.Valid && content.MaxCharge.Decimal.IsNegative() {
		multiErr.Add(
			scenarioField(prefix, "maxCharge"),
			errortypes.ErrInvalid,
			"Maximum charge cannot be negative",
		)
	}
	if content.MinCharge.Valid && content.MaxCharge.Valid &&
		content.MinCharge.Decimal.GreaterThan(content.MaxCharge.Decimal) {
		multiErr.Add(
			scenarioField(prefix, "minCharge"),
			errortypes.ErrInvalid,
			"Minimum charge cannot exceed maximum charge",
		)
	}
	if content.RoundingMode != "" && !content.RoundingMode.IsValid() {
		multiErr.Add(
			scenarioField(prefix, "roundingMode"),
			errortypes.ErrInvalid,
			"Rounding mode is invalid",
		)
	}
	if content.RoundingPrecision < 0 ||
		content.RoundingPrecision > formulatypes.MaxRoundingPrecision {
		multiErr.Add(
			scenarioField(prefix, "roundingPrecision"),
			errortypes.ErrInvalid,
			"Rounding precision must be between 0 and 4",
		)
	}

	for index, definition := range content.VariableDefinitions {
		if definition == nil {
			continue
		}
		if utf8.RuneCountInString(definition.Name) > maxScenarioVariableNameLength ||
			utf8.RuneCountInString(definition.Description) > maxScenarioVariableTextLength {
			multiErr.Add(
				scenarioField(prefix, fmt.Sprintf("variableDefinitions[%d]", index)),
				errortypes.ErrInvalid,
				"Variable name or description is too long",
			)
		}
	}
	for index, definition := range content.BreakdownDefinitions {
		if definition == nil {
			continue
		}
		if utf8.RuneCountInString(definition.Expression) > formulatemplate.MaxExpressionLength ||
			utf8.RuneCountInString(definition.Label) > maxScenarioVariableNameLength {
			multiErr.Add(
				scenarioField(prefix, fmt.Sprintf("breakdownDefinitions[%d]", index)),
				errortypes.ErrInvalid,
				"Breakdown label or expression is too long",
			)
		}
	}

	breakdownErrs := errortypes.NewMultiError()
	formulatypes.ValidateBreakdownDefinitions(content.BreakdownDefinitions, breakdownErrs)
	for _, fieldErr := range breakdownErrs.Errors {
		multiErr.Add(
			scenarioField(prefix, fieldErr.Field),
			fieldErr.Code,
			fieldErr.Message,
			fieldErr.Args...)
	}
}

func scenarioField(prefix, name string) string {
	if prefix == "" {
		return name
	}

	return prefix + "." + name
}

type TestCaseCandidate = formulatemplate.ScenarioContent

type RunTestCasesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TemplateID pulid.ID              `json:"-"`
	Candidate  *TestCaseCandidate    `json:"candidate"`
}

type TestCaseResult struct {
	TestCaseID     pulid.ID        `json:"testCaseId"`
	Name           string          `json:"name"`
	Passed         bool            `json:"passed"`
	ExpectedAmount decimal.Decimal `json:"expectedAmount"`
	ActualAmount   decimal.Decimal `json:"actualAmount"`
	Difference     decimal.Decimal `json:"difference"`
	Tolerance      decimal.Decimal `json:"tolerance"`
	Error          string          `json:"error,omitempty"`
}

type RunTestCasesResponse struct {
	Results []*TestCaseResult `json:"results"`
	Total   int               `json:"total"`
	Passed  int               `json:"passed"`
	Failed  int               `json:"failed"`
}

func (s *Service) RunTestCases(
	ctx context.Context,
	req *RunTestCasesRequest,
) (*RunTestCasesResponse, error) {
	log := s.l.With(
		zap.String("operation", "RunTestCases"),
		zap.String("templateID", req.TemplateID.String()),
	)

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get template", zap.Error(err))
		return nil, err
	}

	cases, err := s.testCaseRepo.ListByTemplate(ctx, formulatemplateports.ListTestCasesRequest{
		TenantInfo: req.TenantInfo,
		TemplateID: req.TemplateID,
	})
	if err != nil {
		log.Error("failed to list test cases", zap.Error(err))
		return nil, err
	}

	candidate := req.Candidate
	if candidate == nil {
		candidate = template.ScenarioContent()
	}

	return s.runCasesAgainstCandidate(
		contextvariablecache.With(ratetablecache.With(ctx)),
		template.SchemaID,
		req.TenantInfo,
		cases,
		candidate,
	), nil
}

func (s *Service) runCasesAgainstCandidate(
	ctx context.Context,
	schemaID string,
	tenantInfo pagination.TenantInfo,
	cases []*formulatemplate.TestCase,
	candidate *TestCaseCandidate,
) *RunTestCasesResponse {
	response := &RunTestCasesResponse{
		Results: make([]*TestCaseResult, 0, len(cases)),
		Total:   len(cases),
	}

	for _, testCase := range cases {
		result := s.runSingleCase(ctx, schemaID, tenantInfo, testCase, candidate)
		if result.Passed {
			response.Passed++
		} else {
			response.Failed++
		}
		response.Results = append(response.Results, result)
	}

	return response
}

func (s *Service) runSingleCase(
	ctx context.Context,
	schemaID string,
	tenantInfo pagination.TenantInfo,
	testCase *formulatemplate.TestCase,
	candidate *TestCaseCandidate,
) *TestCaseResult {
	result := &TestCaseResult{
		TestCaseID:     testCase.ID,
		Name:           testCase.Name,
		ExpectedAmount: testCase.ExpectedAmount,
		Tolerance:      testCase.Tolerance,
	}

	evaluation := s.TestExpression(
		ctx,
		candidateRequest(schemaID, tenantInfo, candidate, testCase.Variables),
	)

	if !evaluation.Valid {
		result.Error = evaluation.Error
		if result.Error == "" {
			result.Error = evaluation.Message
		}
		return result
	}

	actual, ok := evaluation.Result.(decimal.Decimal)
	if !ok {
		result.Error = fmt.Sprintf(
			"expression produced a non-numeric result: %v",
			evaluation.Result,
		)
		return result
	}

	result.ActualAmount = actual
	result.Difference = actual.Sub(testCase.ExpectedAmount)
	result.Passed = result.Difference.Abs().LessThanOrEqual(testCase.Tolerance)

	return result
}

func scenarioVariables(
	content *formulatemplate.ScenarioContent,
	inputs map[string]any,
) map[string]any {
	variables := content.VariableDefaults()
	for key, value := range inputs {
		variables[key] = value
	}

	return variables
}

func candidateRequest(
	schemaID string,
	tenantInfo pagination.TenantInfo,
	content *formulatemplate.ScenarioContent,
	inputs map[string]any,
) *TestExpressionRequest {
	return &TestExpressionRequest{
		Expression:        content.Expression,
		SchemaID:          schemaID,
		Variables:         scenarioVariables(content, inputs),
		TenantInfo:        tenantInfo,
		MinCharge:         content.MinCharge,
		MaxCharge:         content.MaxCharge,
		RoundingMode:      content.RoundingMode,
		RoundingPrecision: content.RoundingPrecision,
	}
}

func (s *Service) requirePassingTestCases(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
) error {
	cases, err := s.testCaseRepo.ListByTemplate(ctx, formulatemplateports.ListTestCasesRequest{
		TenantInfo: tenantInfo,
		TemplateID: template.ID,
	})
	if err != nil {
		return err
	}

	if len(cases) == 0 {
		return errortypes.NewValidationError(
			"testCases",
			errortypes.ErrRequired,
			"Add at least one test scenario before approving, so the template's charge is pinned by an example.",
		)
	}

	run := s.runCasesAgainstCandidate(
		contextvariablecache.With(ratetablecache.With(ctx)),
		template.SchemaID,
		tenantInfo,
		cases,
		template.ScenarioContent(),
	)

	if run.Failed == 0 {
		return nil
	}

	failing := make([]string, 0, run.Failed)
	for _, result := range run.Results {
		if !result.Passed {
			failing = append(failing, result.Name)
		}
	}

	return errortypes.NewValidationError(
		"testCases",
		errortypes.ErrInvalid,
		"{0} of {1} test scenarios fail: {2}. Fix the formula or the scenarios before approving.",
		run.Failed,
		run.Total,
		strings.Join(failing, ", "),
	)
}
