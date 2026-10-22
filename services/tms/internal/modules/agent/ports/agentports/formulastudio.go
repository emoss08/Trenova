package agentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type FormulaStudioDraft struct {
	TemplateID  pulid.ID `json:"templateId"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Source      string   `json:"source"`
}

type ExplainFormulaRequest struct {
	TenantInfo pagination.TenantInfo
	Draft      FormulaStudioDraft
	Inputs     tarifftypes.Inputs
	ShipmentID pulid.ID
}

type FormulaTraceNote struct {
	Index int                `json:"index"`
	Range *rules.TariffRange `json:"range"`
	Says  string             `json:"says"`
}

type ExplainFormulaResponse struct {
	Preview *formulatemplateports.PreviewResult `json:"preview"`
	Items   []FormulaTraceNote                  `json:"items"`
}

type DescribeFormulaInputsRequest struct {
	TenantInfo pagination.TenantInfo
	Draft      FormulaStudioDraft
}

type DescribeFormulaInputsResponse struct {
	Descriptions map[string]string `json:"descriptions"`
}

type SuggestFormulaTestsRequest struct {
	TenantInfo pagination.TenantInfo
	Draft      FormulaStudioDraft
}

type SuggestedFormulaTest struct {
	Name     string                    `json:"name"`
	Block    string                    `json:"block"`
	Status   rules.TariffTestStatus    `json:"status"`
	Failures []rules.TariffTestFailure `json:"failures,omitempty"`
}

type SuggestFormulaTestsResponse struct {
	Tests []SuggestedFormulaTest `json:"tests"`
}

type FormulaTestVerdict string

const (
	FormulaTestVerdictIntended = FormulaTestVerdict("intended")
	FormulaTestVerdictBug      = FormulaTestVerdict("bug")
	FormulaTestVerdictUnclear  = FormulaTestVerdict("unclear")
)

func (v FormulaTestVerdict) IsValid() bool {
	switch v {
	case FormulaTestVerdictIntended, FormulaTestVerdictBug, FormulaTestVerdictUnclear:
		return true
	default:
		return false
	}
}

type ExplainFormulaTestRequest struct {
	TenantInfo pagination.TenantInfo
	Draft      FormulaStudioDraft
	TestName   string
}

type ExplainFormulaTestResponse struct {
	Verdict   FormulaTestVerdict      `json:"verdict"`
	Why       string                  `json:"why"`
	Generated bool                    `json:"generated"`
	Test      *rules.TariffTestResult `json:"test"`
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
	SuggestTests(
		ctx context.Context,
		req *SuggestFormulaTestsRequest,
		actor *permissionports.RequestActor,
	) (*SuggestFormulaTestsResponse, error)
	ExplainTest(
		ctx context.Context,
		req *ExplainFormulaTestRequest,
		actor *permissionports.RequestActor,
	) (*ExplainFormulaTestResponse, error)
	DraftReviewNote(
		ctx context.Context,
		req *DraftFormulaReviewNoteRequest,
		actor *permissionports.RequestActor,
	) (*DraftFormulaReviewNoteResponse, error)
}
