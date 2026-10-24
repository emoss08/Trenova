package report

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type RunError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// Execution history for reports, schedules, and exports
type ReportRun struct {
	keel.Table `                                            keel:"report_runs,report=Reporting"`

	ID                 pulid.ID       `json:"id" keel:"id,pk,autoid:rrun_"`
	BusinessUnitID     pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	DefinitionID       pulid.ID       `json:"definitionId" keel:"definition_id,nullzero"`
	RevisionID         pulid.ID       `json:"revisionId" keel:"revision_id,nullzero,noreport"`
	CannedKey          string         `json:"cannedKey" keel:"canned_key,nullzero,label=Canned Report Key"`
	CannedVersion      string         `json:"cannedVersion" keel:"canned_version,nullzero,noreport"`
	ScheduleID         pulid.ID       `json:"scheduleId" keel:"schedule_id,nullzero"`
	RequestedByID      pulid.ID       `json:"requestedById" keel:"requested_by_id"`
	Trigger            RunTrigger     `json:"trigger" keel:"trigger"`
	Params             map[string]any `json:"params" keel:"params,jsonb,nullzero,noreport"`
	Format             Format         `json:"format" keel:"format"`
	Status             RunStatus      `json:"status" keel:"status"`
	RowCount           int64          `json:"rowCount" keel:"row_count,nullzero,label=Rows,format=count"`
	ByteSize           int64          `json:"byteSize" keel:"byte_size,nullzero,label=Size (bytes),format=count"`
	DurationMs         int64          `json:"durationMs" keel:"duration_ms,nullzero,label=Duration (ms),format=count"`
	Truncated          bool           `json:"truncated" keel:"truncated"`
	Error              *RunError      `json:"error" keel:"error,jsonb,noreport"`
	ArtifactKey        string         `json:"artifactKey" keel:"artifact_key,nullzero,noreport"`
	RowsKey            string         `json:"rowsKey" keel:"rows_key,nullzero"`
	ArtifactExpiresAt  int64          `json:"artifactExpiresAt" keel:"artifact_expires_at,nullzero,noreport"`
	CacheHit           bool           `json:"cacheHit" keel:"cache_hit"`
	TemporalWorkflowID string         `json:"temporalWorkflowId" keel:"temporal_workflow_id,nullzero,noreport"`
	TemporalRunID      string         `json:"temporalRunId" keel:"temporal_run_id,nullzero,noreport"`
	QueuedAt           int64          `json:"queuedAt" keel:"queued_at,nullzero"`
	StartedAt          int64          `json:"startedAt" keel:"started_at,nullzero"`
	CompletedAt        int64          `json:"completedAt" keel:"completed_at,nullzero"`
	Version            int64          `json:"version" keel:"version,version,noreport"`
	CreatedAt          int64          `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt          int64          `json:"updatedAt" keel:"updated_at,autoupdate,noreport"`

	Organization     *tenant.Organization      `json:"organization,omitempty"`
	BusinessUnit     *tenant.BusinessUnit      `json:"businessUnit,omitempty"`
	RequestedBy      *tenant.User              `json:"requestedBy,omitempty" keel:"belongs:requested_by_id,noreport"`
	ReportDefinition *ReportDefinition         `json:"reportDefinition,omitempty" keel:"belongs:definition_id,label=Report"`
	Revision         *ReportDefinitionRevision `json:"revision,omitempty" keel:"belongs:revision_id,noreport"`
}

func (rr *ReportRun) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if rr.QueuedAt == 0 {
		rr.QueuedAt = now
	}
	if rr.Trigger == "" {
		rr.Trigger = RunTriggerManual
	}
	if rr.Status == "" {
		rr.Status = RunStatusQueued
	}
	if rr.QueuedAt == 0 {
		rr.QueuedAt = rr.CreatedAt
	}

	return nil
}

func (rr *ReportRun) GetCreatedAt() int64 { return rr.CreatedAt }

func (rr *ReportRun) GetID() pulid.ID { return rr.ID }

func (rr *ReportRun) GetOrganizationID() pulid.ID { return rr.OrganizationID }

func (rr *ReportRun) GetBusinessUnitID() pulid.ID { return rr.BusinessUnitID }

func (rr *ReportRun) GetTableName() string { return "report_runs" }

func (rr *ReportRun) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "rrun",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeText},
			{Name: "format", Type: domaintypes.FieldTypeText},
			{Name: "canned_key", Type: domaintypes.FieldTypeText},
		},
	}
}

func (r *ReportRun) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&r.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&r.RequestedByID, validation.Required.Error("Requested by is required")),
		validation.Field(&r.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[RunTrigger]("Trigger is invalid"),
		),
		validation.Field(&r.Format,
			validation.Required.Error("Format is required"),
			domainvalidation.ValidEnum[Format]("Format is invalid"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[RunStatus]("Status is invalid"),
		),
		// A run reports on either a saved definition or a canned report; naming
		// neither leaves nothing to execute.
		validation.Field(&r.CannedKey,
			validation.When(r.DefinitionID.IsNil(), validation.Required.Error(
				"A run must name a report definition or a canned report",
			)),
			validation.Length(0, maxRunCannedKeyLength).
				Error("Canned key cannot be longer than 100 characters"),
		),
		validation.Field(&r.CannedVersion,
			validation.Length(0, maxRunCannedVersionLength).
				Error("Canned version cannot be longer than 20 characters"),
		),
		validation.Field(&r.RowCount,
			validation.Min(int64(0)).Error("Row count cannot be negative"),
		),
		validation.Field(&r.ByteSize,
			validation.Min(int64(0)).Error("Byte size cannot be negative"),
		),
		validation.Field(&r.DurationMs,
			validation.Min(int64(0)).Error("Duration cannot be negative"),
		),
		validation.Field(&r.ArtifactKey,
			validation.Length(0, maxArtifactKeyLength).
				Error("Artifact key cannot be longer than 512 characters"),
		),
		validation.Field(&r.RowsKey,
			validation.Length(0, maxArtifactKeyLength).
				Error("Rows key cannot be longer than 512 characters"),
		),
		validation.Field(&r.TemporalWorkflowID,
			validation.Length(0, maxTemporalFieldLength).
				Error("Workflow cannot be longer than 255 characters"),
		),
		validation.Field(&r.TemporalRunID,
			validation.Length(0, maxTemporalFieldLength).
				Error("Workflow run cannot be longer than 255 characters"),
		),
		// A run that finished before it started would report a negative
		// duration on the runs list.
		validation.Field(&r.CompletedAt,
			validation.Min(r.StartedAt).Error("Completion cannot precede the start"),
		),
	))
}

const (
	maxRunCannedKeyLength     = 100
	maxRunCannedVersionLength = 20
	maxArtifactKeyLength      = 512
	maxTemporalFieldLength    = 255
)
