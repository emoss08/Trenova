package report

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*ReportRun)(nil)

type RunError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type ReportRun struct {
	bun.BaseModel             `bun:"table:report_runs,alias:rrun" json:"-"`
	pagination.CursorValueSet `bun:",embed"                       json:"-"`

	ID                 pulid.ID       `json:"id"                 bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID     pulid.ID       `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID       `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	DefinitionID       pulid.ID       `json:"definitionId"       bun:"definition_id,type:VARCHAR(100),nullzero"`
	RevisionID         pulid.ID       `json:"revisionId"         bun:"revision_id,type:VARCHAR(100),nullzero"`
	CannedKey          string         `json:"cannedKey"          bun:"canned_key,type:VARCHAR(100),nullzero"`
	CannedVersion      string         `json:"cannedVersion"      bun:"canned_version,type:VARCHAR(20),nullzero"`
	ScheduleID         pulid.ID       `json:"scheduleId"         bun:"schedule_id,type:VARCHAR(100),nullzero"`
	RequestedByID      pulid.ID       `json:"requestedById"      bun:"requested_by_id,type:VARCHAR(100),notnull"`
	Trigger            RunTrigger     `json:"trigger"            bun:"trigger,type:VARCHAR(20),notnull,default:'manual'"`
	Params             map[string]any `json:"params"             bun:"params,type:JSONB,nullzero"`
	Format             Format         `json:"format"             bun:"format,type:VARCHAR(10),notnull"`
	Status             RunStatus      `json:"status"             bun:"status,type:VARCHAR(20),notnull,default:'queued'"`
	RowCount           int64          `json:"rowCount"           bun:"row_count,type:BIGINT,nullzero"`
	ByteSize           int64          `json:"byteSize"           bun:"byte_size,type:BIGINT,nullzero"`
	DurationMs         int64          `json:"durationMs"         bun:"duration_ms,type:BIGINT,nullzero"`
	Truncated          bool           `json:"truncated"          bun:"truncated,type:BOOLEAN,notnull,default:false"`
	Error              *RunError      `json:"error"              bun:"error,type:JSONB,nullzero"`
	ArtifactKey        string         `json:"artifactKey"        bun:"artifact_key,type:VARCHAR(512),nullzero"`
	ArtifactExpiresAt  int64          `json:"artifactExpiresAt"  bun:"artifact_expires_at,type:BIGINT,nullzero"`
	CacheHit           bool           `json:"cacheHit"           bun:"cache_hit,type:BOOLEAN,notnull,default:false"`
	TemporalWorkflowID string         `json:"temporalWorkflowId" bun:"temporal_workflow_id,type:VARCHAR(255),nullzero"`
	TemporalRunID      string         `json:"temporalRunId"      bun:"temporal_run_id,type:VARCHAR(255),nullzero"`
	QueuedAt           int64          `json:"queuedAt"           bun:"queued_at,type:BIGINT,nullzero"`
	StartedAt          int64          `json:"startedAt"          bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt        int64          `json:"completedAt"        bun:"completed_at,type:BIGINT,nullzero"`
	Version            int64          `json:"version"            bun:"version,type:BIGINT"`
	CreatedAt          int64          `json:"createdAt"          bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64          `json:"updatedAt"          bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization     *tenant.Organization      `json:"organization,omitempty"     bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit     *tenant.BusinessUnit      `json:"businessUnit,omitempty"     bun:"rel:belongs-to,join:business_unit_id=id"`
	RequestedBy      *tenant.User              `json:"requestedBy,omitempty"      bun:"rel:belongs-to,join:requested_by_id=id"`
	ReportDefinition *ReportDefinition         `json:"reportDefinition,omitempty" bun:"rel:belongs-to,join:definition_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Revision         *ReportDefinitionRevision `json:"revision,omitempty"         bun:"rel:belongs-to,join:revision_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (rr *ReportRun) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rr.ID.IsNil() {
			rr.ID = pulid.MustNew("rrun_")
		}
		rr.CreatedAt = now
		if rr.QueuedAt == 0 {
			rr.QueuedAt = now
		}
	case *bun.UpdateQuery:
		rr.UpdatedAt = now
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
