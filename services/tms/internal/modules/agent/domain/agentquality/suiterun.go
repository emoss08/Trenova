package agentquality

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*SuiteRun)(nil)
	_ pagination.CursorEntity            = (*SuiteRun)(nil)
)

const (
	SuiteRunIDPrefix   = "asr_"
	MaxSuiteComments   = 2000
	MaxSweepKeyChars   = 200
	SuiteRevisionChars = 64
)

type SuiteRunStatus string

const (
	SuiteRunStatusRunning       = SuiteRunStatus("Running")
	SuiteRunStatusCompleted     = SuiteRunStatus("Completed")
	SuiteRunStatusSkipped       = SuiteRunStatus("Skipped")
	SuiteRunStatusBudgetStopped = SuiteRunStatus("BudgetStopped")
	SuiteRunStatusFailed        = SuiteRunStatus("Failed")
)

func AllSuiteRunStatuses() []SuiteRunStatus {
	return []SuiteRunStatus{
		SuiteRunStatusRunning,
		SuiteRunStatusCompleted,
		SuiteRunStatusSkipped,
		SuiteRunStatusBudgetStopped,
		SuiteRunStatusFailed,
	}
}

func (s SuiteRunStatus) IsValid() bool {
	return slices.Contains(AllSuiteRunStatuses(), s)
}

func (s SuiteRunStatus) Terminal() bool {
	return s.IsValid() && s != SuiteRunStatusRunning
}

func (s SuiteRunStatus) Scored() bool {
	return s == SuiteRunStatusCompleted || s == SuiteRunStatusBudgetStopped
}

type SuiteRunTrigger string

const (
	SuiteRunTriggerScheduled = SuiteRunTrigger("Scheduled")
	SuiteRunTriggerManual    = SuiteRunTrigger("Manual")
)

func AllSuiteRunTriggers() []SuiteRunTrigger {
	return []SuiteRunTrigger{SuiteRunTriggerScheduled, SuiteRunTriggerManual}
}

func (t SuiteRunTrigger) IsValid() bool {
	return slices.Contains(AllSuiteRunTriggers(), t)
}

type SuiteRun struct {
	keel.Table     `keel:"agent_suite_runs"`
	ID             pulid.ID `keel:"id,pk"                      json:"id"`
	BusinessUnitID pulid.ID `keel:"business_unit_id,pk,tenant" json:"businessUnitId"`
	OrganizationID pulid.ID `keel:"organization_id,pk,tenant"  json:"organizationId"`

	AgentDefinitionID  pulid.ID                  `json:"agentDefinitionId"  keel:"agent_definition_id"`
	Trigger            SuiteRunTrigger           `json:"trigger"            keel:"trigger"`
	SweepKey           string                    `json:"sweepKey"           keel:"sweep_key,nullzero"`
	Fingerprint        *agent.Fingerprint        `json:"fingerprint"        keel:"fingerprint,jsonb"`
	FingerprintHash    string                    `json:"fingerprintHash"    keel:"fingerprint_hash"`
	FingerprintChanges []agent.FingerprintChange `json:"fingerprintChanges" keel:"fingerprint_changes,jsonb"`
	SuiteRevision      string                    `json:"suiteRevision"      keel:"suite_revision"`
	SampleSeed         int64                     `json:"sampleSeed"         keel:"sample_seed"`
	Status             SuiteRunStatus            `json:"status"             keel:"status"`

	CasesTotal   int `json:"casesTotal"   keel:"cases_total"`
	CasesPassed  int `json:"casesPassed"  keel:"cases_passed"`
	CasesFailed  int `json:"casesFailed"  keel:"cases_failed"`
	CasesSkipped int `json:"casesSkipped" keel:"cases_skipped"`
	HardFailures int `json:"hardFailures" keel:"hard_failures"`

	DeterministicScore *float64  `json:"deterministicScore" keel:"deterministic_score"`
	JudgeScore         *float64  `json:"judgeScore"         keel:"judge_score"`
	QualityScore       *float64  `json:"qualityScore"       keel:"quality_score"`
	BaselineScore      *float64  `json:"baselineScore"      keel:"baseline_score"`
	BaselineRunID      *pulid.ID `json:"baselineRunId"      keel:"baseline_run_id"`
	Regression         bool      `json:"regression"         keel:"regression"`

	CostUSD           decimal.Decimal `json:"costUsd"           keel:"cost_usd"`
	StartedAt         int64           `json:"startedAt"         keel:"started_at"`
	FinishedAt        *int64          `json:"finishedAt"        keel:"finished_at"`
	Comments          string          `json:"comments"          keel:"comments,nullzero"`
	WorkflowID        string          `json:"workflowId"        keel:"workflow_id,nullzero"`
	RequestedByUserID *pulid.ID       `json:"requestedByUserId" keel:"requested_by_user_id"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (r *SuiteRun) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if r.ID.IsNil() {
		r.ID = NewSuiteRunID()
	}
	if r.Status == "" {
		r.Status = SuiteRunStatusRunning
	}
	if r.Trigger == "" {
		r.Trigger = SuiteRunTriggerScheduled
	}
	if r.FingerprintChanges == nil {
		r.FingerprintChanges = []agent.FingerprintChange{}
	}
	if r.StartedAt == 0 {
		r.StartedAt = now
	}

	return nil
}

func NewSuiteRunID() pulid.ID {
	return pulid.MustNew(SuiteRunIDPrefix)
}

func (r *SuiteRun) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		r,
		validation.Field(&r.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&r.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&r.AgentDefinitionID, validation.Required.Error("Agent is required")),
		validation.Field(
			&r.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[SuiteRunTrigger]("Trigger is invalid"),
		),
		validation.Field(
			&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[SuiteRunStatus]("Status is invalid"),
		),
		validation.Field(
			&r.FingerprintHash,
			validation.Required.Error("Fingerprint is required"),
			validation.Length(SuiteRevisionChars, SuiteRevisionChars).
				Error("Fingerprint is invalid"),
		),
		validation.Field(
			&r.SuiteRevision,
			validation.Required.Error("Suite revision is required"),
			validation.Length(SuiteRevisionChars, SuiteRevisionChars).
				Error("Suite revision is invalid"),
		),
		validation.Field(
			&r.SweepKey,
			validation.Length(0, MaxSweepKeyChars).Error("Sweep key is at most 200 characters"),
		),
		validation.Field(
			&r.Comments,
			validation.Length(0, MaxSuiteComments).Error("Comments are at most 2000 characters"),
		),
		validation.Field(&r.StartedAt, validation.Required.Error("Start time is required")),
	))

	if r.CasesTotal < 0 || r.CasesPassed < 0 || r.CasesFailed < 0 || r.CasesSkipped < 0 {
		multiErr.Add("casesTotal", errortypes.ErrInvalid, "Case counts cannot be negative")
	}
	if r.CasesPassed+r.CasesFailed+r.CasesSkipped > r.CasesTotal {
		multiErr.Add(
			"casesTotal",
			errortypes.ErrInvalid,
			"Passed, failed and skipped cases cannot add up to more than the suite holds",
		)
	}
	if r.CostUSD.IsNegative() {
		multiErr.Add("costUsd", errortypes.ErrInvalid, "Cost cannot be negative")
	}
	scores := [...]struct {
		field string
		value *float64
	}{
		{"deterministicScore", r.DeterministicScore},
		{"judgeScore", r.JudgeScore},
		{"qualityScore", r.QualityScore},
		{"baselineScore", r.BaselineScore},
	}
	for _, score := range scores {
		if score.value != nil && (*score.value < 0 || *score.value > 1) {
			multiErr.Add(score.field, errortypes.ErrInvalid, "A score is between 0 and 1")
		}
	}
}

func (r *SuiteRun) Finish(status SuiteRunStatus, comments string, at int64) {
	r.Status = status
	r.FinishedAt = &at
	if comments != "" {
		r.Comments = comments
	}
}

func (r *SuiteRun) GetID() pulid.ID { return r.ID }

func (r *SuiteRun) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *SuiteRun) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *SuiteRun) GetCreatedAt() int64 { return r.CreatedAt }

func (r *SuiteRun) GetTableName() string { return "agent_suite_runs" }

func (r *SuiteRun) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "asr",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "comments", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "trigger", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (SuiteRun) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_agent_suite_runs_one_running",
			Message: "This agent's suite is already running",
		},
		{
			Name:    "uq_agent_suite_runs_sweep_key",
			Message: "This sweep already opened a run for the agent",
		},
	}
}
