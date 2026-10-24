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
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*ShadowResult)(nil)
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
	bun.BaseModel `bun:"table:extraction_shadow_results,alias:exsr" json:"-"`
	keel.Table    `keel:"extraction_shadow_results"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:exsr_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`

	DocumentID   pulid.ID     `json:"documentId"   bun:"document_id,type:VARCHAR(100),notnull" keel:"document_id"`
	ExtractedAt  int64        `json:"extractedAt"  bun:"extracted_at,type:BIGINT,notnull" keel:"extracted_at"`
	Status       ResultStatus `json:"status"       bun:"status,type:VARCHAR(20),notnull,default:'Pending'" keel:"status"`
	StatusReason string       `json:"statusReason" bun:"status_reason,type:TEXT,nullzero" keel:"status_reason,nullzero"`

	ProviderID           pulid.ID  `json:"providerId"           bun:"provider_id,type:VARCHAR(100),notnull" keel:"provider_id"`
	ProviderName         string    `json:"providerName"         bun:"provider_name,type:VARCHAR(100),notnull" keel:"provider_name"`
	ServedModel          string    `json:"servedModel"          bun:"served_model,type:VARCHAR(255),nullzero" keel:"served_model,nullzero"`
	ProductionProviderID *pulid.ID `json:"productionProviderId" bun:"production_provider_id,type:VARCHAR(100),nullzero" keel:"production_provider_id"`
	ProductionModel      string    `json:"productionModel"      bun:"production_model,type:VARCHAR(255),nullzero" keel:"production_model,nullzero"`

	Accepted        bool                   `json:"accepted"        bun:"accepted,type:BOOLEAN,notnull,default:false" keel:"accepted"`
	RejectionReason string                 `json:"rejectionReason" bun:"rejection_reason,type:TEXT,nullzero" keel:"rejection_reason,nullzero"`
	DraftData       map[string]any         `json:"-"               bun:"draft_data,type:JSONB,nullzero" keel:"draft_data,jsonb,nullzero"`
	Predicted       *aicorrection.Snapshot `json:"predicted"       bun:"predicted,type:JSONB,nullzero" keel:"predicted,jsonb"`

	CorrectionID *pulid.ID                  `json:"correctionId" bun:"correction_id,type:VARCHAR(100),nullzero" keel:"correction_id"`
	ScoredAt     *int64                     `json:"scoredAt"     bun:"scored_at,type:BIGINT,nullzero" keel:"scored_at"`
	Verdict      Verdict                    `json:"verdict"      bun:"verdict,type:VARCHAR(20),nullzero" keel:"verdict,nullzero"`
	FieldResults []aicorrection.FieldResult `json:"fieldResults" bun:"field_results,type:JSONB,notnull,default:'[]'" keel:"field_results,jsonb"`

	ScoredCount    int     `json:"scoredCount"    bun:"scored_count,type:INTEGER,notnull,default:0" keel:"scored_count"`
	CorrectCount   int     `json:"correctCount"   bun:"correct_count,type:INTEGER,notnull,default:0" keel:"correct_count"`
	CorrectedCount int     `json:"correctedCount" bun:"corrected_count,type:INTEGER,notnull,default:0" keel:"corrected_count"`
	MissedCount    int     `json:"missedCount"    bun:"missed_count,type:INTEGER,notnull,default:0" keel:"missed_count"`
	Accuracy       float64 `json:"accuracy"       bun:"accuracy,type:DOUBLE PRECISION,notnull,default:0" keel:"accuracy"`

	BaselineFieldResults   []aicorrection.FieldResult `json:"baselineFieldResults"   bun:"baseline_field_results,type:JSONB,notnull,default:'[]'" keel:"baseline_field_results,jsonb"`
	BaselineScoredCount    int                        `json:"baselineScoredCount"    bun:"baseline_scored_count,type:INTEGER,notnull,default:0" keel:"baseline_scored_count"`
	BaselineCorrectCount   int                        `json:"baselineCorrectCount"   bun:"baseline_correct_count,type:INTEGER,notnull,default:0" keel:"baseline_correct_count"`
	BaselineCorrectedCount int                        `json:"baselineCorrectedCount" bun:"baseline_corrected_count,type:INTEGER,notnull,default:0" keel:"baseline_corrected_count"`
	BaselineMissedCount    int                        `json:"baselineMissedCount"    bun:"baseline_missed_count,type:INTEGER,notnull,default:0" keel:"baseline_missed_count"`
	BaselineAccuracy       float64                    `json:"baselineAccuracy"       bun:"baseline_accuracy,type:DOUBLE PRECISION,notnull,default:0" keel:"baseline_accuracy"`

	LatencyMs    int64           `json:"latencyMs"    bun:"latency_ms,type:BIGINT,notnull,default:0" keel:"latency_ms"`
	InputTokens  int64           `json:"inputTokens"  bun:"input_tokens,type:BIGINT,notnull,default:0" keel:"input_tokens"`
	OutputTokens int64           `json:"outputTokens" bun:"output_tokens,type:BIGINT,notnull,default:0" keel:"output_tokens"`
	CostUSD      decimal.Decimal `json:"costUsd"      bun:"cost_usd,type:NUMERIC(14,6),notnull,default:0" keel:"cost_usd"`

	WorkflowID  string `json:"workflowId"  bun:"workflow_id,type:VARCHAR(255),nullzero" keel:"workflow_id,nullzero"`
	StartedAt   *int64 `json:"startedAt"   bun:"started_at,type:BIGINT,nullzero" keel:"started_at"`
	CompletedAt *int64 `json:"completedAt" bun:"completed_at,type:BIGINT,nullzero" keel:"completed_at"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
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

func (r *ShadowResult) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("exsr_")
		}
		if r.CreatedAt == 0 {
			r.CreatedAt = now
		}
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}
