package extractioneval

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
	keel.Table `keel:"extraction_eval_runs"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:eer_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Task          aicorrection.Task `json:"task"          keel:"task"`
	Status        RunStatus         `json:"status"        keel:"status"`
	ProviderID    pulid.ID          `json:"providerId"    keel:"provider_id"`
	ProviderName  string            `json:"providerName"  keel:"provider_name"`
	ProviderModel string            `json:"providerModel" keel:"provider_model,nullzero"`
	ServedModel   string            `json:"servedModel"   keel:"served_model,nullzero"`
	CaseLimit     int               `json:"caseLimit"     keel:"case_limit"`

	CasesTotal     int `json:"casesTotal"     keel:"cases_total"`
	CasesCompleted int `json:"casesCompleted" keel:"cases_completed"`
	CasesFailed    int `json:"casesFailed"    keel:"cases_failed"`
	CasesSkipped   int `json:"casesSkipped"   keel:"cases_skipped"`

	ScoredCount      int     `json:"scoredCount"      keel:"scored_count"`
	CorrectCount     int     `json:"correctCount"     keel:"correct_count"`
	CorrectedCount   int     `json:"correctedCount"   keel:"corrected_count"`
	MissedCount      int     `json:"missedCount"      keel:"missed_count"`
	UnconfirmedCount int     `json:"unconfirmedCount" keel:"unconfirmed_count"`
	UnscoredCount    int     `json:"unscoredCount"    keel:"unscored_count"`
	Accuracy         float64 `json:"accuracy"         keel:"accuracy"`

	FieldAccuracy []aicorrection.FieldAccuracy `json:"fieldAccuracy" keel:"field_accuracy,jsonb"`

	CostUSD      decimal.Decimal `json:"costUsd"      keel:"cost_usd"`
	InputTokens  int64           `json:"inputTokens"  keel:"input_tokens"`
	OutputTokens int64           `json:"outputTokens" keel:"output_tokens"`
	AvgLatencyMs int64           `json:"avgLatencyMs" keel:"avg_latency_ms"`

	StopReason     string `json:"stopReason"     keel:"stop_reason,nullzero"`
	FailureMessage string `json:"failureMessage" keel:"failure_message,nullzero"`

	RequestedByID pulid.ID `json:"requestedById" keel:"requested_by_id"`
	WorkflowID    string   `json:"workflowId"    keel:"workflow_id,nullzero"`
	StartedAt     *int64   `json:"startedAt"     keel:"started_at"`
	FinishedAt    *int64   `json:"finishedAt"    keel:"finished_at"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (r *ExtractionRun) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		r,
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
		validation.Field(
			&r.ProviderID,
			validation.Required.Error("Choose the AI provider to evaluate"),
		),
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

func (r *ExtractionRun) BeforeCreate(_ context.Context) error {
	if r.Status == "" {
		r.Status = RunStatusQueued
	}

	return nil
}
