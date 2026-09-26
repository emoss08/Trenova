package extractioneval

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
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
	_ bun.BeforeAppendModelHook      = (*ExtractionRun)(nil)
	_ pagination.CursorEntity        = (*ExtractionRun)(nil)
	_ domaintypes.PostgresSearchable = (*ExtractionRun)(nil)
)

const (
	DefaultCaseLimit     = 50
	MaxCaseLimit         = 500
	MaxStopReasonRunes   = 500
	MaxFailureRunes      = 2000
	MaxProviderNameRunes = 100
	MaxModelRunes        = 255
)

func RunWorkflowID(runID pulid.ID) string {
	return "extraction-eval-run:" + runID.String()
}

type ExtractionRun struct {
	bun.BaseModel `bun:"table:extraction_eval_runs,alias:eer" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Task          aicorrection.Task `json:"task"          bun:"task,type:VARCHAR(50),notnull"`
	Status        RunStatus         `json:"status"        bun:"status,type:VARCHAR(20),notnull,default:'Queued'"`
	ProviderID    pulid.ID          `json:"providerId"    bun:"provider_id,type:VARCHAR(100),notnull"`
	ProviderName  string            `json:"providerName"  bun:"provider_name,type:VARCHAR(100),notnull"`
	ProviderModel string            `json:"providerModel" bun:"provider_model,type:VARCHAR(255),nullzero"`
	ServedModel   string            `json:"servedModel"   bun:"served_model,type:VARCHAR(255),nullzero"`
	CaseLimit     int               `json:"caseLimit"     bun:"case_limit,type:INTEGER,notnull"`

	CasesTotal     int `json:"casesTotal"     bun:"cases_total,type:INTEGER,notnull,default:0"`
	CasesCompleted int `json:"casesCompleted" bun:"cases_completed,type:INTEGER,notnull,default:0"`
	CasesFailed    int `json:"casesFailed"    bun:"cases_failed,type:INTEGER,notnull,default:0"`
	CasesSkipped   int `json:"casesSkipped"   bun:"cases_skipped,type:INTEGER,notnull,default:0"`

	ScoredCount      int     `json:"scoredCount"      bun:"scored_count,type:INTEGER,notnull,default:0"`
	CorrectCount     int     `json:"correctCount"     bun:"correct_count,type:INTEGER,notnull,default:0"`
	CorrectedCount   int     `json:"correctedCount"   bun:"corrected_count,type:INTEGER,notnull,default:0"`
	MissedCount      int     `json:"missedCount"      bun:"missed_count,type:INTEGER,notnull,default:0"`
	UnconfirmedCount int     `json:"unconfirmedCount" bun:"unconfirmed_count,type:INTEGER,notnull,default:0"`
	UnscoredCount    int     `json:"unscoredCount"    bun:"unscored_count,type:INTEGER,notnull,default:0"`
	Accuracy         float64 `json:"accuracy"         bun:"accuracy,type:DOUBLE PRECISION,notnull,default:0"`

	FieldAccuracy []aicorrection.FieldAccuracy `json:"fieldAccuracy" bun:"field_accuracy,type:JSONB,notnull,default:'[]'"`

	CostUSD      decimal.Decimal `json:"costUsd"      bun:"cost_usd,type:NUMERIC(14,6),notnull,default:0"`
	InputTokens  int64           `json:"inputTokens"  bun:"input_tokens,type:BIGINT,notnull,default:0"`
	OutputTokens int64           `json:"outputTokens" bun:"output_tokens,type:BIGINT,notnull,default:0"`
	AvgLatencyMs int64           `json:"avgLatencyMs" bun:"avg_latency_ms,type:BIGINT,notnull,default:0"`

	StopReason     string `json:"stopReason"     bun:"stop_reason,type:TEXT,nullzero"`
	FailureMessage string `json:"failureMessage" bun:"failure_message,type:TEXT,nullzero"`

	RequestedByID pulid.ID `json:"requestedById" bun:"requested_by_id,type:VARCHAR(100),notnull"`
	WorkflowID    string   `json:"workflowId"    bun:"workflow_id,type:VARCHAR(255),nullzero"`
	StartedAt     *int64   `json:"startedAt"     bun:"started_at,type:BIGINT,nullzero"`
	FinishedAt    *int64   `json:"finishedAt"    bun:"finished_at,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (r *ExtractionRun) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&r.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&r.Task,
			validation.Required.Error("Task is required"),
			domainvalidation.ValidEnum[aicorrection.Task]("Task is invalid"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[RunStatus]("Status is invalid"),
		),
		validation.Field(&r.ProviderID, validation.Required.Error("Choose the AI provider to evaluate")),
		validation.Field(&r.ProviderName,
			validation.Required.Error("Provider name is required"),
			validation.RuneLength(1, MaxProviderNameRunes).
				Error(fmt.Sprintf("Provider name must be at most %d characters", MaxProviderNameRunes)),
		),
		validation.Field(&r.CaseLimit,
			validation.Min(1).Error("Evaluate at least one case"),
			validation.Max(MaxCaseLimit).
				Error(fmt.Sprintf("A run evaluates at most %d cases", MaxCaseLimit)),
		),
		validation.Field(&r.RequestedByID, validation.Required.Error("Requested by is required")),
	))
}

func (r *ExtractionRun) ApplyResults(results []*ExtractionResult) {
	aggregator := aicorrection.NewAccuracyAggregator()
	var tally aicorrection.Tally
	var latency, timed int64
	r.CasesCompleted, r.CasesFailed, r.CasesSkipped = 0, 0, 0
	r.CostUSD = decimal.Zero
	r.InputTokens, r.OutputTokens = 0, 0

	for _, result := range results {
		switch result.Status {
		case ResultStatusCompleted:
			r.CasesCompleted++
			aggregator.Add(result.FieldResults)
			t := aicorrection.TallyResults(result.FieldResults)
			tally.Scored += t.Scored
			tally.Correct += t.Correct
			tally.Corrected += t.Corrected
			tally.Missed += t.Missed
			tally.Unconfirmed += t.Unconfirmed
			tally.Unscored += t.Unscored
			if result.LatencyMs > 0 {
				latency += result.LatencyMs
				timed++
			}
			if r.ServedModel == "" {
				r.ServedModel = result.Model
			}
		case ResultStatusFailed:
			r.CasesFailed++
		case ResultStatusSkipped:
			r.CasesSkipped++
		case ResultStatusPending:
		}
		r.CostUSD = r.CostUSD.Add(result.CostUSD)
		r.InputTokens += result.InputTokens
		r.OutputTokens += result.OutputTokens
	}

	r.ScoredCount = tally.Scored
	r.CorrectCount = tally.Correct
	r.CorrectedCount = tally.Corrected
	r.MissedCount = tally.Missed
	r.UnconfirmedCount = tally.Unconfirmed
	r.UnscoredCount = tally.Unscored
	r.Accuracy = aicorrection.Accuracy(tally.Correct, tally.Scored)
	r.FieldAccuracy = aggregator.Fields()
	r.AvgLatencyMs = 0
	if timed > 0 {
		r.AvgLatencyMs = latency / timed
	}
}

func (r *ExtractionRun) GetID() pulid.ID { return r.ID }

func (r *ExtractionRun) GetCreatedAt() int64 { return r.CreatedAt }

func (r *ExtractionRun) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *ExtractionRun) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *ExtractionRun) GetTableName() string { return "extraction_eval_runs" }

func (r *ExtractionRun) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "eer",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "provider_name", Type: domaintypes.FieldTypeText},
			{Name: "provider_model", Type: domaintypes.FieldTypeText},
			{Name: "served_model", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (r *ExtractionRun) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("eer_")
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
