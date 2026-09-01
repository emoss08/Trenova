package formulaassistantservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxInstructionLength = 4000
	maxExpressionLength  = 8000
	logPreviewLength     = 512
)

type Params struct {
	fx.In

	Logger          *zap.Logger
	Completion      serviceports.CompletionService
	FormulaService  *formula.Service
	TemplateService *formulatemplateservice.Service
	RateMatrixRepo  repositories.RateMatrixRepository
	AILogRepo       repositories.AILogRepository
}

type Service struct {
	l               *zap.Logger
	completion      serviceports.CompletionService
	formulaService  *formula.Service
	templateService *formulatemplateservice.Service
	rateMatrixRepo  repositories.RateMatrixRepository
	aiLogRepo       repositories.AILogRepository
}

func New(p Params) *Service { //nolint:gocritic // fx param structs are passed by value
	return &Service{
		l:               p.Logger.Named("service.formulaassistant"),
		completion:      p.Completion,
		formulaService:  p.FormulaService,
		templateService: p.TemplateService,
		rateMatrixRepo:  p.RateMatrixRepo,
		aiLogRepo:       p.AILogRepo,
	}
}

type GenerateFormulaRequest struct {
	TenantInfo   pagination.TenantInfo        `json:"-"`
	Instruction  string                       `json:"instruction"`
	SchemaID     string                       `json:"schemaId"`
	TemplateType formulatemplate.TemplateType `json:"templateType"`
}

type GeneratedVariable struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	DefaultValue any    `json:"defaultValue"`
}

type GenerateFormulaResponse struct {
	Expression          string                                         `json:"expression"`
	VariableDefinitions []*formulatypes.VariableDefinition             `json:"variableDefinitions"`
	Explanation         string                                         `json:"explanation"`
	Validation          *formulatemplateservice.TestExpressionResponse `json:"validation"`
	ModelIdentifier     string                                         `json:"modelIdentifier"`
}

type ExplainFormulaRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Expression string                `json:"expression"`
	SchemaID   string                `json:"schemaId"`
}

type ExplainFormulaResponse struct {
	Explanation     string `json:"explanation"`
	ModelIdentifier string `json:"modelIdentifier"`
}

type generatedPayload struct {
	Expression  string              `json:"expression"`
	Variables   []GeneratedVariable `json:"variables"`
	Explanation string              `json:"explanation"`
}

type explanationPayload struct {
	Explanation string `json:"explanation"`
}

func (s *Service) GenerateFormula(
	ctx context.Context,
	req *GenerateFormulaRequest,
) (*GenerateFormulaResponse, error) {
	if err := validateGenerateRequest(req); err != nil {
		return nil, err
	}

	schemaID := defaultSchemaID(req.SchemaID)
	description, err := s.formulaService.DescribeSchema(schemaID)
	if err != nil {
		return nil, err
	}

	lookupTables, err := s.listLookupTableCodes(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	templateType := string(req.TemplateType)
	if templateType == "" {
		templateType = string(formulatemplate.TemplateTypeFreightCharge)
	}

	result, err := s.completion.CompleteStructured(ctx, &serviceports.StructuredCompletionRequest{
		TenantInfo:   req.TenantInfo,
		System:       generateSystemPrompt,
		Context:      buildGenerateContext(description, lookupTables, templateType, req.Instruction),
		OutputSchema: generateOutputSchema(),
	})
	if err != nil {
		return nil, err
	}

	var payload generatedPayload
	if err = sonic.Unmarshal([]byte(result.Text), &payload); err != nil {
		return nil, fmt.Errorf("%w: %w", serviceports.ErrModelSchemaValidation, err)
	}

	variables, testValues := mapGeneratedVariables(payload.Variables)

	validation := s.templateService.TestExpression(
		ctx,
		&formulatemplateservice.TestExpressionRequest{
			Expression: payload.Expression,
			SchemaID:   schemaID,
			Variables:  testValues,
			TenantInfo: req.TenantInfo,
		},
	)

	s.logCall(ctx, req.TenantInfo, ailog.OperationFormulaGenerate, schemaID, req.Instruction, result)

	return &GenerateFormulaResponse{
		Expression:          payload.Expression,
		VariableDefinitions: variables,
		Explanation:         payload.Explanation,
		Validation:          validation,
		ModelIdentifier:     result.ModelIdentifier,
	}, nil
}

func (s *Service) ExplainFormula(
	ctx context.Context,
	req *ExplainFormulaRequest,
) (*ExplainFormulaResponse, error) {
	if err := validateExplainRequest(req); err != nil {
		return nil, err
	}

	schemaID := defaultSchemaID(req.SchemaID)
	description, err := s.formulaService.DescribeSchema(schemaID)
	if err != nil {
		return nil, err
	}

	result, err := s.completion.CompleteStructured(ctx, &serviceports.StructuredCompletionRequest{
		TenantInfo:   req.TenantInfo,
		System:       explainSystemPrompt,
		Context:      buildExplainContext(description, req.Expression),
		OutputSchema: explainOutputSchema(),
	})
	if err != nil {
		return nil, err
	}

	var payload explanationPayload
	if err = sonic.Unmarshal([]byte(result.Text), &payload); err != nil {
		return nil, fmt.Errorf("%w: %w", serviceports.ErrModelSchemaValidation, err)
	}

	s.logCall(ctx, req.TenantInfo, ailog.OperationFormulaExplain, schemaID, req.Expression, result)

	return &ExplainFormulaResponse{
		Explanation:     payload.Explanation,
		ModelIdentifier: result.ModelIdentifier,
	}, nil
}

func validateGenerateRequest(req *GenerateFormulaRequest) error {
	multiErr := errortypes.NewMultiError()

	if req.Instruction == "" {
		multiErr.Add("instruction", errortypes.ErrRequired, "An instruction is required")
	}
	if len(req.Instruction) > maxInstructionLength {
		multiErr.Add(
			"instruction",
			errortypes.ErrInvalid,
			fmt.Sprintf("Instruction cannot exceed %d characters", maxInstructionLength),
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func validateExplainRequest(req *ExplainFormulaRequest) error {
	multiErr := errortypes.NewMultiError()

	if req.Expression == "" {
		multiErr.Add("expression", errortypes.ErrRequired, "An expression is required")
	}
	if len(req.Expression) > maxExpressionLength {
		multiErr.Add(
			"expression",
			errortypes.ErrInvalid,
			fmt.Sprintf("Expression cannot exceed %d characters", maxExpressionLength),
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func defaultSchemaID(schemaID string) string {
	if schemaID == "" {
		return "shipment"
	}

	return schemaID
}

func (s *Service) listLookupTableCodes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]string, error) {
	data, err := s.rateMatrixRepo.GetLookupData(
		ctx,
		&repositories.GetRateMatrixLookupDataRequest{TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}

	codes := make([]string, 0, len(data))
	for _, entry := range data {
		if entry == nil || entry.Matrix == nil {
			continue
		}
		switch len(entry.Matrix.Dimensions) {
		case 1:
			codes = append(codes, entry.Matrix.Code+" (single-axis: use lookup/lookupOr)")
		case 2:
			codes = append(codes, entry.Matrix.Code+" (two-axis: use lookup2/lookup2Or)")
		}
	}

	return codes, nil
}

func mapGeneratedVariables(
	generated []GeneratedVariable,
) ([]*formulatypes.VariableDefinition, map[string]any) {
	variables := make([]*formulatypes.VariableDefinition, 0, len(generated))
	testValues := make(map[string]any, len(generated))

	for _, variable := range generated {
		if variable.Name == "" {
			continue
		}

		variableType := formulatypes.VariableValueType(variable.Type)
		switch variableType {
		case formulatypes.VariableValueTypeNumber,
			formulatypes.VariableValueTypeString,
			formulatypes.VariableValueTypeBoolean:
		default:
			variableType = formulatypes.VariableValueTypeNumber
		}

		variables = append(variables, &formulatypes.VariableDefinition{
			Name:         variable.Name,
			Type:         variableType,
			Description:  variable.Description,
			DefaultValue: variable.DefaultValue,
		})

		if variable.DefaultValue != nil {
			testValues[variable.Name] = variable.DefaultValue
		}
	}

	return variables, testValues
}

func (s *Service) logCall(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	operation ailog.Operation,
	object, prompt string,
	result *serviceports.StructuredCompletionResult,
) {
	promptHash := sha256.Sum256([]byte(prompt))
	responseHash := sha256.Sum256([]byte(result.Text))

	promptPreview := prompt
	if len(promptPreview) > logPreviewLength {
		promptPreview = promptPreview[:logPreviewLength]
	}

	entry := &ailog.Log{
		ID:             pulid.MustNew("ail_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		UserID:         tenantInfo.UserID,
		Prompt: fmt.Sprintf(
			"sha256=%s preview=%s",
			hex.EncodeToString(promptHash[:]),
			promptPreview,
		),
		Response:         "sha256=" + hex.EncodeToString(responseHash[:]),
		Model:            ailog.ModelClaudeOpus5,
		Operation:        operation,
		Object:           object,
		PromptTokens:     result.InputTokens,
		CompletionTokens: result.OutputTokens,
		TotalTokens:      result.InputTokens + result.OutputTokens,
		Timestamp:        timeutils.NowUnix(),
	}

	if _, err := s.aiLogRepo.Create(ctx, entry); err != nil {
		s.l.Error("failed to log AI call", zap.Error(err))
	}
}
