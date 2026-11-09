package dataretention

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.Entity = (*DataRetention)(nil)
)

const (
	DefaultAIFeedbackRetentionDays    = 730
	DefaultAIFeedbackRetentionMinDays = 30
	DefaultAICorrectionRetentionDays  = 730
	MinAICorrectionRetentionDays      = 30
	DefaultAgentEvalCaseRetentionDays = 365
	DefaultAIAuditRetentionDays       = 2555
	MinAIAuditRetentionDays           = 365

	MinAuditAccessRetentionDays      = 90
	MinAuditOperationalRetentionDays = 730
	MinAuditFinancialRetentionDays   = 2555
	MinAuditSecurityRetentionDays    = 730
	MaxAuditRetentionDays            = 3650
)

type DataRetention struct {
	keel.Table `keel:"data_retention"`

	ID                            pulid.ID `json:"id"                            keel:"id,pk,autoid:dr_"`
	BusinessUnitID                pulid.ID `json:"businessUnitId"                keel:"business_unit_id,pk,tenant"`
	OrganizationID                pulid.ID `json:"organizationId"                keel:"organization_id,pk,tenant"`
	AuditAccessRetentionDays      int      `json:"auditAccessRetentionDays"      keel:"audit_access_retention_days"`
	AuditOperationalRetentionDays int      `json:"auditOperationalRetentionDays" keel:"audit_operational_retention_days"`
	AuditFinancialRetentionDays   int      `json:"auditFinancialRetentionDays"   keel:"audit_financial_retention_days"`
	AuditSecurityRetentionDays    int      `json:"auditSecurityRetentionDays"    keel:"audit_security_retention_days"`
	EDIInboundFileRetentionPeriod int      `json:"ediInboundFileRetentionPeriod" keel:"edi_inbound_file_retention_period"` // In days, 0 disables purging
	EDIMessageRetentionPeriod     int      `json:"ediMessageRetentionPeriod"     keel:"edi_message_retention_period"`      // In days, 0 disables purging
	// DriverQualificationRetentionPeriod is how long a driver qualification
	// file is held past termination before it is eligible for purge, in days.
	// 1095 is the three years 49 CFR 391.51(d) requires; zero means the reader
	// falls back to that, so an unset value can never make every terminated
	// file look purgeable. Files are only ever flagged; nothing deletes one.
	DriverQualificationRetentionPeriod int   `json:"driverQualificationRetentionPeriod" keel:"driver_qualification_retention_period"`
	AgentEvalCaseRetentionPeriod       int   `json:"agentEvalCaseRetentionPeriod"       keel:"agent_eval_case_retention_period"`
	AIFeedbackRetentionPeriod          int   `json:"aiFeedbackRetentionPeriod"          keel:"ai_feedback_retention_period"`
	AIAuditRetentionPeriod             int   `json:"aiAuditRetentionPeriod"             keel:"ai_audit_retention_period"` // In days, at least MinAIAuditRetentionDays
	AICorrectionRetentionPeriod        int   `json:"aiCorrectionRetentionPeriod"        keel:"ai_correction_retention_period"`
	Version                            int64 `json:"version"                            keel:"version,version"`
	CreatedAt                          int64 `json:"createdAt"                          keel:"created_at,autocreate"`
	UpdatedAt                          int64 `json:"updatedAt"                          keel:"updated_at,autoupdate"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
}

func (dr *DataRetention) BeforeCreate(_ context.Context) error {
	dr.applyAuditDefaults()
	if dr.DriverQualificationRetentionPeriod == 0 {
		dr.DriverQualificationRetentionPeriod = 1095
	}
	if dr.AgentEvalCaseRetentionPeriod == 0 {
		dr.AgentEvalCaseRetentionPeriod = 365
	}
	if dr.AIFeedbackRetentionPeriod == 0 {
		dr.AIFeedbackRetentionPeriod = 730
	}
	if dr.AICorrectionRetentionPeriod == 0 {
		dr.AICorrectionRetentionPeriod = 730
	}

	return nil
}

func (dr *DataRetention) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(dr,
		validation.Field(&dr.AuditAccessRetentionDays,
			validation.Min(MinAuditAccessRetentionDays).
				Error("Reads and downloads are kept for at least 90 days"),
			validation.Max(MaxAuditRetentionDays).
				Error("Audit entries are kept for at most 3,650 days"),
		),
		validation.Field(&dr.AuditOperationalRetentionDays,
			validation.Min(MinAuditOperationalRetentionDays).
				Error("Operational changes are kept for at least 730 days"),
			validation.Max(MaxAuditRetentionDays).
				Error("Audit entries are kept for at most 3,650 days"),
		),
		validation.Field(&dr.AuditFinancialRetentionDays,
			validation.Min(MinAuditFinancialRetentionDays).
				Error("Financial changes are kept for at least 2,555 days (seven years)"),
			validation.Max(MaxAuditRetentionDays).
				Error("Audit entries are kept for at most 3,650 days"),
		),
		validation.Field(&dr.AuditSecurityRetentionDays,
			validation.Min(MinAuditSecurityRetentionDays).
				Error("Security entries are kept for at least 730 days"),
			validation.Max(MaxAuditRetentionDays).
				Error("Audit entries are kept for at most 3,650 days"),
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

func (dr *DataRetention) applyAuditDefaults() {
	if dr.AuditAccessRetentionDays == 0 {
		dr.AuditAccessRetentionDays = MinAuditAccessRetentionDays
	}
	if dr.AuditOperationalRetentionDays == 0 {
		dr.AuditOperationalRetentionDays = MinAuditOperationalRetentionDays
	}
	if dr.AuditFinancialRetentionDays == 0 {
		dr.AuditFinancialRetentionDays = MinAuditFinancialRetentionDays
	}
	if dr.AuditSecurityRetentionDays == 0 {
		dr.AuditSecurityRetentionDays = MinAuditSecurityRetentionDays
	}
}

func (dr *DataRetention) AuditRetentionDays(class string) int {
	floors := map[string]int{
		"Access":      MinAuditAccessRetentionDays,
		"Operational": MinAuditOperationalRetentionDays,
		"Financial":   MinAuditFinancialRetentionDays,
		"Security":    MinAuditSecurityRetentionDays,
	}
	floor, ok := floors[class]
	if !ok {
		return MaxAuditRetentionDays
	}
	if dr == nil {
		return floor
	}
	chosen := map[string]int{
		"Access":      dr.AuditAccessRetentionDays,
		"Operational": dr.AuditOperationalRetentionDays,
		"Financial":   dr.AuditFinancialRetentionDays,
		"Security":    dr.AuditSecurityRetentionDays,
	}[class]
	return min(max(chosen, floor), MaxAuditRetentionDays)
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
