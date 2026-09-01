package formulatemplateservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var defaultTestCaseTolerance = decimal.NewFromFloat(0.01)

type TestCaseInput struct {
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Variables      map[string]any  `json:"variables"`
	ExpectedAmount decimal.Decimal `json:"expectedAmount"`
	Tolerance      decimal.Decimal `json:"tolerance"`
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
	req repositories.ListTestCasesRequest,
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

	if vErr := validateTestCase(entity); vErr != nil {
		return nil, vErr
	}

	created, err := s.testCaseRepo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create test case", zap.Error(err))
		return nil, err
	}

	s.logAuditAction(
		log,
		template,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		nil,
		fmt.Sprintf("Test scenario %q added", created.Name),
	)

	return created, nil
}

func (s *Service) UpdateTestCase(
	ctx context.Context,
	req *UpdateTestCaseRequest,
) (*formulatemplate.TestCase, error) {
	log := s.l.With(
		zap.String("operation", "UpdateTestCase"),
		zap.String("testCaseID", req.TestCaseID.String()),
	)

	existing, err := s.testCaseRepo.GetByID(ctx, repositories.GetTestCaseByIDRequest{
		TenantInfo: req.TenantInfo,
		TestCaseID: req.TestCaseID,
		TemplateID: req.TemplateID,
	})
	if err != nil {
		log.Error("failed to get test case", zap.Error(err))
		return nil, err
	}

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

	if vErr := validateTestCase(existing); vErr != nil {
		return nil, vErr
	}

	updated, err := s.testCaseRepo.Update(ctx, existing)
	if err != nil {
		log.Error("failed to update test case", zap.Error(err))
		return nil, err
	}

	return updated, nil
}

func (s *Service) DeleteTestCase(
	ctx context.Context,
	req repositories.GetTestCaseByIDRequest,
) error {
	log := s.l.With(
		zap.String("operation", "DeleteTestCase"),
		zap.String("testCaseID", req.TestCaseID.String()),
	)

	if err := s.testCaseRepo.Delete(ctx, req); err != nil {
		log.Error("failed to delete test case", zap.Error(err))
		return err
	}

	return nil
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
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// TestCaseCandidate carries unsaved editor content so scenarios can run
// against what the author is typing, not just what is stored.
type TestCaseCandidate struct {
	Expression           string                              `json:"expression"`
	VariableDefinitions  []*formulatypes.VariableDefinition  `json:"variableDefinitions"`
	BreakdownDefinitions []*formulatypes.BreakdownDefinition `json:"breakdownDefinitions"`
	MinCharge            decimal.NullDecimal                 `json:"minCharge"`
	MaxCharge            decimal.NullDecimal                 `json:"maxCharge"`
	RoundingMode         ratetypes.RoundingMode              `json:"roundingMode"`
	RoundingPrecision    int32                               `json:"roundingPrecision"`
}

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

	cases, err := s.testCaseRepo.ListByTemplate(ctx, repositories.ListTestCasesRequest{
		TenantInfo: req.TenantInfo,
		TemplateID: req.TemplateID,
	})
	if err != nil {
		log.Error("failed to list test cases", zap.Error(err))
		return nil, err
	}

	candidate := req.Candidate
	if candidate == nil {
		candidate = &TestCaseCandidate{
			Expression:           template.Expression,
			VariableDefinitions:  template.VariableDefinitions,
			BreakdownDefinitions: template.BreakdownDefinitions,
			MinCharge:            template.MinCharge,
			MaxCharge:            template.MaxCharge,
		}
	}

	return s.runCasesAgainstCandidate(
		ratetablecache.With(ctx),
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

	variables := make(map[string]any, len(testCase.Variables)+len(candidate.VariableDefinitions))
	for _, definition := range candidate.VariableDefinitions {
		if definition != nil && definition.Name != "" && definition.DefaultValue != nil {
			variables[definition.Name] = definition.DefaultValue
		}
	}
	for key, value := range testCase.Variables {
		variables[key] = value
	}

	evaluation := s.TestExpression(ctx, &TestExpressionRequest{
		Expression:        candidate.Expression,
		SchemaID:          schemaID,
		Variables:         variables,
		TenantInfo:        tenantInfo,
		MinCharge:         candidate.MinCharge,
		MaxCharge:         candidate.MaxCharge,
		RoundingMode:      candidate.RoundingMode,
		RoundingPrecision: candidate.RoundingPrecision,
	})

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

// requirePassingTestCases is the approval gate: a template with saved
// scenarios cannot go Active while any of them fail against its content.
// Templates without scenarios pass — the gate enforces what authors pinned,
// it does not force adoption.
func (s *Service) requirePassingTestCases(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
) error {
	cases, err := s.testCaseRepo.ListByTemplate(ctx, repositories.ListTestCasesRequest{
		TenantInfo: tenantInfo,
		TemplateID: template.ID,
	})
	if err != nil {
		return err
	}

	if len(cases) == 0 {
		return nil
	}

	run := s.runCasesAgainstCandidate(
		ratetablecache.With(ctx),
		template.SchemaID,
		tenantInfo,
		cases,
		&TestCaseCandidate{
			Expression:           template.Expression,
			VariableDefinitions:  template.VariableDefinitions,
			BreakdownDefinitions: template.BreakdownDefinitions,
			MinCharge:            template.MinCharge,
			MaxCharge:            template.MaxCharge,
			RoundingMode:         template.RoundingMode,
			RoundingPrecision:    template.RoundingPrecision,
		},
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
		fmt.Sprintf(
			"%d of %d test scenarios fail: %s. Fix the formula or the scenarios before approving.",
			run.Failed,
			run.Total,
			strings.Join(failing, ", "),
		),
	)
}
