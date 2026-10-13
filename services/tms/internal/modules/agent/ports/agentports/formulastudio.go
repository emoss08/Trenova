package agentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type FormulaStudioDraft struct {
	TemplateID  pulid.ID
	Name        string
	Description string
	Type        formulatemplate.TemplateType
	SchemaID    string
	Content     formulatemplate.ScenarioContent
}

type ExplainFormulaRequest struct {
	TenantInfo   pagination.TenantInfo
	Draft        FormulaStudioDraft
	SampleValues map[string]any
	ShipmentID   pulid.ID
}

type FormulaStatementNote struct {
	StatementIndex int    `json:"statementIndex"`
	Says           string `json:"says"`
}

type ExplainFormulaResponse struct {
	Statements   []formulatemplatetypes.Statement      `json:"statements"`
	Items        []FormulaStatementNote                `json:"items"`
	RawAmount    *string                               `json:"rawAmount"`
	BilledAmount *string                               `json:"billedAmount"`
	Guardrail    *formulatemplatetypes.GuardrailResult `json:"guardrail"`
}

type DescribeFormulaInputsRequest struct {
	TenantInfo pagination.TenantInfo
	Draft      FormulaStudioDraft
}

type DescribeFormulaInputsResponse struct {
	Descriptions map[string]string `json:"descriptions"`
}

type ExistingFormulaScenario struct {
	Name      string         `json:"name"`
	Variables map[string]any `json:"variables"`
}

type SuggestFormulaScenariosRequest struct {
	TenantInfo pagination.TenantInfo
	Draft      FormulaStudioDraft
	Existing   []ExistingFormulaScenario
}

type SuggestFormulaScenariosResponse struct {
	Scenarios []pagedraft.PricedScenario `json:"scenarios"`
}

type ExplainFormulaScenarioRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
	TestCaseID pulid.ID
	Candidate  formulatemplate.ScenarioContent
}

type ExplainFormulaScenarioResponse struct {
	Verdict           formulatemplateports.ScenarioVerdict    `json:"verdict"`
	Why               string                                  `json:"why"`
	Generated         bool                                    `json:"generated"`
	Expected          string                                  `json:"expected"`
	Actual            *string                                 `json:"actual"`
	Delta             *string                                 `json:"delta"`
	BaselineLabel     string                                  `json:"baselineLabel"`
	RecordedOnVersion *int64                                  `json:"recordedOnVersion"`
	RecordedBy        *formulatemplateports.ScenarioRecorder  `json:"recordedBy"`
	Error             string                                  `json:"error,omitempty"`
	Rows              []formulatemplateports.ScenarioTraceRow `json:"rows"`
}

type DraftFormulaReviewNoteRequest struct {
	TenantInfo      pagination.TenantInfo
	TemplateID      pulid.ID
	IncludeBacktest bool
}

type DraftFormulaReviewNoteResponse struct {
	Note      string                                `json:"note"`
	Generated bool                                  `json:"generated"`
	Backtest  *formulatemplateports.BacktestSummary `json:"backtest"`
}

type FormulaStudioAssist interface {
	ExplainFormula(
		ctx context.Context,
		req *ExplainFormulaRequest,
		actor *permissionports.RequestActor,
	) (*ExplainFormulaResponse, error)
	DescribeInputs(
		ctx context.Context,
		req *DescribeFormulaInputsRequest,
		actor *permissionports.RequestActor,
	) (*DescribeFormulaInputsResponse, error)
	SuggestScenarios(
		ctx context.Context,
		req *SuggestFormulaScenariosRequest,
		actor *permissionports.RequestActor,
	) (*SuggestFormulaScenariosResponse, error)
	ExplainScenario(
		ctx context.Context,
		req *ExplainFormulaScenarioRequest,
		actor *permissionports.RequestActor,
	) (*ExplainFormulaScenarioResponse, error)
	DraftReviewNote(
		ctx context.Context,
		req *DraftFormulaReviewNoteRequest,
		actor *permissionports.RequestActor,
	) (*DraftFormulaReviewNoteResponse, error)
}
