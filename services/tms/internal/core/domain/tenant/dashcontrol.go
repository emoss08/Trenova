package tenant

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*DashControl)(nil)
	_ validationframework.TenantedEntity = (*DashControl)(nil)
)

type DashControl struct {
	bun.BaseModel `bun:"table:dash_controls,alias:dashc" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	RequireLoadAcknowledgment  bool `json:"requireLoadAcknowledgment"  bun:"require_load_acknowledgment,type:BOOLEAN,notnull,default:true"`
	AllowLoadRefusals          bool `json:"allowLoadRefusals"          bun:"allow_load_refusals,type:BOOLEAN,notnull,default:true"`
	AllowStopActions           bool `json:"allowStopActions"           bun:"allow_stop_actions,type:BOOLEAN,notnull,default:true"`
	AllowLoadDocumentUpload    bool `json:"allowLoadDocumentUpload"    bun:"allow_load_document_upload,type:BOOLEAN,notnull,default:true"`
	AllowLoadComments          bool `json:"allowLoadComments"          bun:"allow_load_comments,type:BOOLEAN,notnull,default:true"`
	ShowLoadPay                bool `json:"showLoadPay"                bun:"show_load_pay,type:BOOLEAN,notnull,default:true"`
	ShowPayEstimates           bool `json:"showPayEstimates"           bun:"show_pay_estimates,type:BOOLEAN,notnull,default:true"`
	AllowExpenseSubmission     bool `json:"allowExpenseSubmission"     bun:"allow_expense_submission,type:BOOLEAN,notnull,default:true"`
	RequireExpenseReceipt      bool `json:"requireExpenseReceipt"      bun:"require_expense_receipt,type:BOOLEAN,notnull,default:false"`
	AllowSettlementDisputes    bool `json:"allowSettlementDisputes"    bun:"allow_settlement_disputes,type:BOOLEAN,notnull,default:true"`
	AllowProfileDocumentUpload bool `json:"allowProfileDocumentUpload" bun:"allow_profile_document_upload,type:BOOLEAN,notnull,default:true"`
	AllowContactInfoEdit       bool `json:"allowContactInfoEdit"       bun:"allow_contact_info_edit,type:BOOLEAN,notnull,default:true"`
	AllowPtoRequests           bool `json:"allowPtoRequests"           bun:"allow_pto_requests,type:BOOLEAN,notnull,default:true"`
	SendCredentialReminders    bool `json:"sendCredentialReminders"    bun:"send_credential_reminders,type:BOOLEAN,notnull,default:true"`

	EnableDetentionAlerts          bool  `json:"enableDetentionAlerts"          bun:"enable_detention_alerts,type:BOOLEAN,notnull,default:true"`
	DetentionAlertThresholdMinutes int16 `json:"detentionAlertThresholdMinutes" bun:"detention_alert_threshold_minutes,type:INTEGER,notnull,default:120"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (dc *DashControl) Validate(multiErr *errortypes.MultiError) {
	if dc.EnableDetentionAlerts &&
		(dc.DetentionAlertThresholdMinutes < 15 || dc.DetentionAlertThresholdMinutes > 1440) {
		multiErr.Add(
			"detentionAlertThresholdMinutes",
			errortypes.ErrInvalid,
			"Detention alert threshold must be between 15 minutes and 24 hours",
		)
	}
	if dc.AllowLoadRefusals && !dc.RequireLoadAcknowledgment {
		multiErr.Add(
			"allowLoadRefusals",
			errortypes.ErrInvalid,
			"Load refusals require load acknowledgment to be enabled",
		)
	}
	if dc.ShowPayEstimates && !dc.ShowLoadPay {
		multiErr.Add(
			"showPayEstimates",
			errortypes.ErrInvalid,
			"Pay estimates require per-load pay visibility to be enabled",
		)
	}
	if dc.RequireExpenseReceipt && !dc.AllowExpenseSubmission {
		multiErr.Add(
			"requireExpenseReceipt",
			errortypes.ErrInvalid,
			"Receipt requirement only applies when expense submission is enabled",
		)
	}
}

func (dc *DashControl) GetID() pulid.ID { return dc.ID }

func (dc *DashControl) GetTableName() string { return "dash_controls" }

func (dc *DashControl) GetOrganizationID() pulid.ID { return dc.OrganizationID }

func (dc *DashControl) GetBusinessUnitID() pulid.ID { return dc.BusinessUnitID }

func (dc *DashControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if dc.ID.IsNil() {
			dc.ID = pulid.MustNew("dashc_")
		}
		dc.CreatedAt = now
	case *bun.UpdateQuery:
		dc.UpdatedAt = now
	}
	return nil
}
