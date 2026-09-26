package tenant

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*DataRetention)(nil)
	_ validationframework.TenantedEntity = (*DataRetention)(nil)
)

const (
	DefaultAIFeedbackRetentionDays    = 730
	DefaultAIFeedbackRetentionMinDays = 30
	DefaultAICorrectionRetentionDays  = 730
	MinAICorrectionRetentionDays      = 30
	DefaultAgentEvalCaseRetentionDays = 365
	DefaultAIAuditRetentionDays       = 2555
	MinAIAuditRetentionDays           = 365
)

type DataRetention struct {
	bun.BaseModel `bun:"table:data_retention,alias:dr" json:"-"`

	ID                            pulid.ID `json:"id"                                 bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID                pulid.ID `json:"businessUnitId"                     bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID                pulid.ID `json:"organizationId"                     bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	AuditRetentionPeriod          int      `json:"auditRetentionPeriod"               bun:"audit_retention_period,type:INTEGER,notnull,default:120"` // In days
	EDIInboundFileRetentionPeriod int      `json:"ediInboundFileRetentionPeriod"      bun:"edi_inbound_file_retention_period,type:INTEGER,notnull"`  // In days, 0 disables purging
	EDIMessageRetentionPeriod     int      `json:"ediMessageRetentionPeriod"          bun:"edi_message_retention_period,type:INTEGER,notnull"`       // In days, 0 disables purging
	// DriverQualificationRetentionPeriod is how long a driver qualification
	// file is held past termination before it is eligible for purge, in days.
	// 1095 is the three years 49 CFR 391.51(d) requires; zero means the reader
	// falls back to that, so an unset value can never make every terminated
	// file look purgeable. Files are only ever flagged; nothing deletes one.
	DriverQualificationRetentionPeriod int   `json:"driverQualificationRetentionPeriod" bun:"driver_qualification_retention_period,type:INTEGER,notnull,default:1095"`
	AgentEvalCaseRetentionPeriod       int   `json:"agentEvalCaseRetentionPeriod"       bun:"agent_eval_case_retention_period,type:INTEGER,notnull,default:365"`
	AIFeedbackRetentionPeriod          int   `json:"aiFeedbackRetentionPeriod"          bun:"ai_feedback_retention_period,type:INTEGER,notnull,default:730"`
	AIAuditRetentionPeriod             int   `json:"aiAuditRetentionPeriod"             bun:"ai_audit_retention_period,type:INTEGER,notnull,default:2555"` // In days, at least MinAIAuditRetentionDays
	AICorrectionRetentionPeriod        int   `json:"aiCorrectionRetentionPeriod"        bun:"ai_correction_retention_period,type:INTEGER,notnull,default:730"`
	Version                            int64 `json:"version"                            bun:"version,type:BIGINT"`
	CreatedAt                          int64 `json:"createdAt"                          bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                          int64 `json:"updatedAt"                          bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (dr *DataRetention) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(dr,
		validation.Field(&dr.AuditRetentionPeriod,
			validation.Required.Error("Audit retention period is required"),
			validation.Min(1).Error("Audit retention period must be greater than 0"),
		),
		validation.Field(&dr.EDIInboundFileRetentionPeriod,
			validation.Min(0).Error("EDI inbound file retention period cannot be negative"),
		),
		validation.Field(&dr.EDIMessageRetentionPeriod,
			validation.Min(0).Error("EDI message retention period cannot be negative"),
		),
		validation.Field(&dr.DriverQualificationRetentionPeriod,
			validation.Min(0).
				Error("Driver qualification retention period cannot be negative"),
		),
		validation.Field(&dr.AgentEvalCaseRetentionPeriod,
			validation.Min(0).Error("Agent evaluation case retention period cannot be negative"),
		),
		validation.Field(&dr.AIFeedbackRetentionPeriod,
			validation.Min(0).Error("AI feedback retention period cannot be negative"),
		),
		validation.Field(&dr.AICorrectionRetentionPeriod,
			validation.Min(0).Error("AI correction retention period cannot be negative"),
		),
		validation.Field(&dr.AIAuditRetentionPeriod,
			validation.Required.Error("AI audit trail retention period is required"),
			validation.Min(MinAIAuditRetentionDays).
				Error("AI audit trail retention period must be at least 365 days"),
		),
	))

	if dr.AIFeedbackRetentionPeriod > 0 &&
		dr.AIFeedbackRetentionPeriod < DefaultAIFeedbackRetentionMinDays {
		multiErr.Add(
			"aiFeedbackRetentionPeriod",
			errortypes.ErrInvalid,
			"AI feedback retention period must be at least 30 days",
		)
	}

	if dr.AICorrectionRetentionPeriod > 0 &&
		dr.AICorrectionRetentionPeriod < MinAICorrectionRetentionDays {
		multiErr.Add(
			"aiCorrectionRetentionPeriod",
			errortypes.ErrInvalid,
			"AI correction retention period must be at least 30 days",
		)
	}
}

func (dr *DataRetention) AICorrectionRetentionDays() int {
	if dr == nil || dr.AICorrectionRetentionPeriod <= 0 {
		return DefaultAICorrectionRetentionDays
	}

	return dr.AICorrectionRetentionPeriod
}

func (dr *DataRetention) AIFeedbackRetentionDays() int {
	if dr == nil || dr.AIFeedbackRetentionPeriod <= 0 {
		return DefaultAIFeedbackRetentionDays
	}

	return dr.AIFeedbackRetentionPeriod
}

func (dr *DataRetention) AIAuditRetentionDays() int {
	if dr == nil || dr.AIAuditRetentionPeriod <= 0 {
		return DefaultAIAuditRetentionDays
	}

	return dr.AIAuditRetentionPeriod
}

func (dr *DataRetention) GetID() pulid.ID {
	return dr.ID
}

func (dr *DataRetention) GetTableName() string {
	return "data_retention"
}

func (dr *DataRetention) GetOrganizationID() pulid.ID {
	return dr.OrganizationID
}

func (dr *DataRetention) GetBusinessUnitID() pulid.ID {
	return dr.BusinessUnitID
}

func (dr *DataRetention) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if dr.ID.IsNil() {
			dr.ID = pulid.MustNew("dr_")
		}

		dr.CreatedAt = now
	case *bun.UpdateQuery:
		dr.UpdatedAt = now
	}

	return nil
}
