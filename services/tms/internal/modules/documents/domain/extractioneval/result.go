package extractioneval

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var (
	_ pagination.CursorEntity        = (*ExtractionResult)(nil)
	_ domaintypes.PostgresSearchable = (*ExtractionResult)(nil)
)

type ExtractionResult struct {
	keel.Table `keel:"extraction_eval_results"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:eeres_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	RunID     pulid.ID     `json:"runId" keel:"run_id"`
	CaseID    pulid.ID     `json:"caseId" keel:"case_id"`
	CaseTitle string       `json:"caseTitle" keel:"case_title"`
	Ordinal   int          `json:"ordinal" keel:"ordinal"`
	Status    ResultStatus `json:"status" keel:"status"`

	Model        string                     `json:"model" keel:"model,nullzero"`
	ProviderID   *pulid.ID                  `json:"providerId" keel:"provider_id"`
	Predicted    *aicorrection.Snapshot     `json:"predicted" keel:"predicted,jsonb"`
	FieldResults []aicorrection.FieldResult `json:"fieldResults" keel:"field_results,jsonb"`

	ScoredCount      int     `json:"scoredCount" keel:"scored_count"`
	CorrectCount     int     `json:"correctCount" keel:"correct_count"`
	CorrectedCount   int     `json:"correctedCount" keel:"corrected_count"`
	MissedCount      int     `json:"missedCount" keel:"missed_count"`
	UnconfirmedCount int     `json:"unconfirmedCount" keel:"unconfirmed_count"`
	UnscoredCount    int     `json:"unscoredCount" keel:"unscored_count"`
	Accuracy         float64 `json:"accuracy" keel:"accuracy"`

	LatencyMs    int64           `json:"latencyMs" keel:"latency_ms"`
	InputTokens  int64           `json:"inputTokens" keel:"input_tokens"`
	OutputTokens int64           `json:"outputTokens" keel:"output_tokens"`
	CostUSD      decimal.Decimal `json:"costUsd" keel:"cost_usd"`
	ErrorMessage string          `json:"errorMessage" keel:"error_message,nullzero"`
	StartedAt    *int64          `json:"startedAt" keel:"started_at"`
	CompletedAt  *int64          `json:"completedAt" keel:"completed_at"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (r *ExtractionResult) ApplyScore(results []aicorrection.FieldResult) {
	r.FieldResults = results
	t := aicorrection.TallyResults(results)
	r.ScoredCount = t.Scored
	r.CorrectCount = t.Correct
	r.CorrectedCount = t.Corrected
	r.MissedCount = t.Missed
	r.UnconfirmedCount = t.Unconfirmed
	r.UnscoredCount = t.Unscored
	r.Accuracy = aicorrection.Accuracy(t.Correct, t.Scored)
}

func (r *ExtractionResult) GetID() pulid.ID { return r.ID }

func (r *ExtractionResult) GetCreatedAt() int64 { return r.CreatedAt }

func (r *ExtractionResult) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *ExtractionResult) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *ExtractionResult) GetTableName() string { return "extraction_eval_results" }

func (r *ExtractionResult) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "eeres",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "case_title", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (r *ExtractionResult) BeforeCreate(_ context.Context) error {
	if r.Status == "" {
		r.Status = ResultStatusPending
	}

	return nil
}
