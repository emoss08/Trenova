package agentquality

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*SuiteRun)(nil)
	_ validationframework.TenantedEntity = (*SuiteRun)(nil)
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
	bun.BaseModel `bun:"table:agent_suite_runs,alias:asr" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	AgentDefinitionID  pulid.ID                  `json:"agentDefinitionId"  bun:"agent_definition_id,type:VARCHAR(100),notnull"`
	Trigger            SuiteRunTrigger           `json:"trigger"            bun:"trigger,type:VARCHAR(20),notnull,default:'Scheduled'"`
	SweepKey           string                    `json:"sweepKey"           bun:"sweep_key,type:VARCHAR(200),nullzero"`
	Fingerprint        *agent.Fingerprint        `json:"fingerprint"        bun:"fingerprint,type:JSONB,nullzero"`
	FingerprintHash    string                    `json:"fingerprintHash"    bun:"fingerprint_hash,type:VARCHAR(64),notnull"`
	FingerprintChanges []agent.FingerprintChange `json:"fingerprintChanges" bun:"fingerprint_changes,type:JSONB,notnull,default:'[]'"`
	SuiteRevision      string                    `json:"suiteRevision"      bun:"suite_revision,type:VARCHAR(64),notnull"`
	SampleSeed         int64                     `json:"sampleSeed"         bun:"sample_seed,type:BIGINT,notnull"`
	Status             SuiteRunStatus            `json:"status"             bun:"status,type:VARCHAR(20),notnull,default:'Running'"`

	CasesTotal   int `json:"casesTotal"   bun:"cases_total,type:INTEGER,notnull,default:0"`
	CasesPassed  int `json:"casesPassed"  bun:"cases_passed,type:INTEGER,notnull,default:0"`
	CasesFailed  int `json:"casesFailed"  bun:"cases_failed,type:INTEGER,notnull,default:0"`
	CasesSkipped int `json:"casesSkipped" bun:"cases_skipped,type:INTEGER,notnull,default:0"`
	HardFailures int `json:"hardFailures" bun:"hard_failures,type:INTEGER,notnull,default:0"`

	DeterministicScore *float64  `json:"deterministicScore" bun:"deterministic_score,type:DOUBLE PRECISION,nullzero"`
	JudgeScore         *float64  `json:"judgeScore"         bun:"judge_score,type:DOUBLE PRECISION,nullzero"`
	QualityScore       *float64  `json:"qualityScore"       bun:"quality_score,type:DOUBLE PRECISION,nullzero"`
	BaselineScore      *float64  `json:"baselineScore"      bun:"baseline_score,type:DOUBLE PRECISION,nullzero"`
	BaselineRunID      *pulid.ID `json:"baselineRunId"      bun:"baseline_run_id,type:VARCHAR(100),nullzero"`
	Regression         bool      `json:"regression"         bun:"regression,type:BOOLEAN,notnull,default:false"`

	CostUSD           decimal.Decimal `json:"costUsd"           bun:"cost_usd,type:NUMERIC(14,6),notnull,default:0"`
	StartedAt         int64           `json:"startedAt"         bun:"started_at,type:BIGINT,notnull"`
	FinishedAt        *int64          `json:"finishedAt"        bun:"finished_at,type:BIGINT,nullzero"`
	Comments          string          `json:"comments"          bun:"comments,type:TEXT,nullzero"`
	WorkflowID        string          `json:"workflowId"        bun:"workflow_id,type:VARCHAR(255),nullzero"`
	RequestedByUserID *pulid.ID       `json:"requestedByUserId" bun:"requested_by_user_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func NewSuiteRunID() pulid.ID {
	return pulid.MustNew(SuiteRunIDPrefix)
}

func (r *SuiteRun) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&r.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&r.AgentDefinitionID, validation.Required.Error("Agent is required")),
		validation.Field(&r.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[SuiteRunTrigger]("Trigger is invalid"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[SuiteRunStatus]("Status is invalid"),
		),
		validation.Field(&r.FingerprintHash,
			validation.Required.Error("Fingerprint is required"),
			validation.Length(SuiteRevisionChars, SuiteRevisionChars).
				Error("Fingerprint is invalid"),
		),
		validation.Field(&r.SuiteRevision,
			validation.Required.Error("Suite revision is required"),
			validation.Length(SuiteRevisionChars, SuiteRevisionChars).
				Error("Suite revision is invalid"),
		),
		validation.Field(&r.SweepKey,
			validation.Length(0, MaxSweepKeyChars).Error("Sweep key is at most 200 characters"),
		),
		validation.Field(&r.Comments,
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

func (r *SuiteRun) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
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
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}
