package extractionshadow

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ pagination.CursorEntity        = (*ShadowResult)(nil)
	_ domaintypes.PostgresSearchable = (*ShadowResult)(nil)
)

const (
	MaxReasonRunes       = 500
	MaxProviderNameRunes = 100
	MaxModelRunes        = 255
)

func WorkflowID(resultID pulid.ID) string {
	return "extraction-shadow:" + resultID.String()
}

type ShadowResult struct {
	keel.Table `keel:"extraction_shadow_results"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:exsr_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	DocumentID   pulid.ID     `json:"documentId" keel:"document_id"`
	ExtractedAt  int64        `json:"extractedAt" keel:"extracted_at"`
	Status       ResultStatus `json:"status" keel:"status"`
	StatusReason string       `json:"statusReason" keel:"status_reason,nullzero"`

	ProviderID           pulid.ID  `json:"providerId" keel:"provider_id"`
	ProviderName         string    `json:"providerName" keel:"provider_name"`
	ServedModel          string    `json:"servedModel" keel:"served_model,nullzero"`
	ProductionProviderID *pulid.ID `json:"productionProviderId" keel:"production_provider_id"`
	ProductionModel      string    `json:"productionModel" keel:"production_model,nullzero"`

	Accepted        bool                   `json:"accepted" keel:"accepted"`
	RejectionReason string                 `json:"rejectionReason" keel:"rejection_reason,nullzero"`
	DraftData       map[string]any         `json:"-" keel:"draft_data,jsonb,nullzero"`
	Predicted       *aicorrection.Snapshot `json:"predicted" keel:"predicted,jsonb"`

	CorrectionID *pulid.ID                  `json:"correctionId" keel:"correction_id"`
	ScoredAt     *int64                     `json:"scoredAt" keel:"scored_at"`
	Verdict      Verdict                    `json:"verdict" keel:"verdict,nullzero"`
	FieldResults []aicorrection.FieldResult `json:"fieldResults" keel:"field_results,jsonb"`

	ScoredCount    int     `json:"scoredCount" keel:"scored_count"`
	CorrectCount   int     `json:"correctCount" keel:"correct_count"`
	CorrectedCount int     `json:"correctedCount" keel:"corrected_count"`
	MissedCount    int     `json:"missedCount" keel:"missed_count"`
	Accuracy       float64 `json:"accuracy" keel:"accuracy"`

	BaselineFieldResults   []aicorrection.FieldResult `json:"baselineFieldResults" keel:"baseline_field_results,jsonb"`
	BaselineScoredCount    int                        `json:"baselineScoredCount" keel:"baseline_scored_count"`
	BaselineCorrectCount   int                        `json:"baselineCorrectCount" keel:"baseline_correct_count"`
	BaselineCorrectedCount int                        `json:"baselineCorrectedCount" keel:"baseline_corrected_count"`
	BaselineMissedCount    int                        `json:"baselineMissedCount" keel:"baseline_missed_count"`
	BaselineAccuracy       float64                    `json:"baselineAccuracy" keel:"baseline_accuracy"`

	LatencyMs    int64           `json:"latencyMs" keel:"latency_ms"`
	InputTokens  int64           `json:"inputTokens" keel:"input_tokens"`
	OutputTokens int64           `json:"outputTokens" keel:"output_tokens"`
	CostUSD      decimal.Decimal `json:"costUsd" keel:"cost_usd"`

	WorkflowID  string `json:"workflowId" keel:"workflow_id,nullzero"`
	StartedAt   *int64 `json:"startedAt" keel:"started_at"`
	CompletedAt *int64 `json:"completedAt" keel:"completed_at"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (r *ShadowResult) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&r.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&r.DocumentID, validation.Required.Error("Document is required")),
		validation.Field(&r.ExtractedAt, validation.Required.Error("Extraction time is required")),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ResultStatus]("Status is invalid"),
		),
		validation.Field(&r.ProviderID, validation.Required.Error("Provider is required")),
		validation.Field(&r.ProviderName,
			validation.Required.Error("Provider name is required"),
			validation.RuneLength(1, MaxProviderNameRunes).
				Error(fmt.Sprintf("Provider name must be at most %d characters", MaxProviderNameRunes)),
		),
	))
}

func (r *ShadowResult) IsScored() bool {
	return r.CorrectionID != nil && r.CorrectionID.IsNotNil()
}

func (r *ShadowResult) Settle(status ResultStatus, reason string, at int64) {
	r.Status = status
	r.StatusReason = reason
	r.CompletedAt = &at
	if r.StartedAt == nil {
		r.StartedAt = &at
	}
}

func (r *ShadowResult) ApplyScore(
	correctionID pulid.ID,
	candidate, baseline []aicorrection.FieldResult,
	at int64,
) {
	c := aicorrection.TallyResults(candidate)
	b := aicorrection.TallyResults(baseline)

	r.CorrectionID = &correctionID
	r.ScoredAt = &at
	r.FieldResults = candidate
	r.ScoredCount = c.Scored
	r.CorrectCount = c.Correct
	r.CorrectedCount = c.Corrected
	r.MissedCount = c.Missed
	r.Accuracy = aicorrection.Accuracy(c.Correct, c.Scored)
	r.BaselineFieldResults = baseline
	r.BaselineScoredCount = b.Scored
	r.BaselineCorrectCount = b.Correct
	r.BaselineCorrectedCount = b.Corrected
	r.BaselineMissedCount = b.Missed
	r.BaselineAccuracy = aicorrection.Accuracy(b.Correct, b.Scored)
	r.Verdict = Compare(c, b)
}

func Compare(candidate, baseline aicorrection.Tally) Verdict {
	switch {
	case candidate.Correct > baseline.Correct:
		return VerdictBetter
	case candidate.Correct < baseline.Correct:
		return VerdictWorse
	}

	candidateErrors := candidate.Corrected + candidate.Missed
	baselineErrors := baseline.Corrected + baseline.Missed
	switch {
	case candidateErrors < baselineErrors:
		return VerdictBetter
	case candidateErrors > baselineErrors:
		return VerdictWorse
	default:
		return VerdictSame
	}
}

func (r *ShadowResult) GetID() pulid.ID { return r.ID }

func (r *ShadowResult) GetCreatedAt() int64 { return r.CreatedAt }

func (r *ShadowResult) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *ShadowResult) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *ShadowResult) GetTableName() string { return "extraction_shadow_results" }

func (r *ShadowResult) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "exsr",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "provider_name", Type: domaintypes.FieldTypeText},
			{Name: "served_model", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "verdict", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (r *ShadowResult) BeforeCreate(_ context.Context) error {
	if r.Status == "" {
		r.Status = ResultStatusPending
	}

	return nil
}
