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

const (
	minDetentionAlertMinutes = int16(15)
	maxDetentionAlertMinutes = int16(1440)
)

// DriverDigestCadence is how a driver hears about what they owe. Immediate
// keeps one notice per obligation, which is what a carrier already relying on
// them expects; Daily and Weekly bundle everything a driver owes into a single
// notice, so a week of renewals arrives once instead of six times.
type DriverDigestCadence string

const (
	DigestImmediate = DriverDigestCadence("Immediate")
	DigestDaily     = DriverDigestCadence("Daily")
	DigestWeekly    = DriverDigestCadence("Weekly")
)

func (c DriverDigestCadence) String() string { return string(c) }

func (c DriverDigestCadence) IsValid() bool {
	switch c {
	case DigestImmediate, DigestDaily, DigestWeekly:
		return true
	default:
		return false
	}
}

// Bundles reports whether the cadence collects obligations rather than sending
// one notice each.
func (c DriverDigestCadence) Bundles() bool {
	return c == DigestDaily || c == DigestWeekly
}

var (
	_ bun.BeforeAppendModelHook          = (*DashControl)(nil)
	_ validationframework.TenantedEntity = (*DashControl)(nil)
)

type DashControl struct {
	bun.BaseModel `bun:"table:dash_controls,alias:dashc" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	RequireLoadAcknowledgment  bool `json:"requireLoadAcknowledgment"  bun:"require_load_acknowledgment,type:BOOLEAN,notnull"`
	AllowLoadRefusals          bool `json:"allowLoadRefusals"          bun:"allow_load_refusals,type:BOOLEAN,notnull"`
	AllowStopActions           bool `json:"allowStopActions"           bun:"allow_stop_actions,type:BOOLEAN,notnull"`
	AllowLoadDocumentUpload    bool `json:"allowLoadDocumentUpload"    bun:"allow_load_document_upload,type:BOOLEAN,notnull"`
	AllowLoadComments          bool `json:"allowLoadComments"          bun:"allow_load_comments,type:BOOLEAN,notnull"`
	ShowLoadPay                bool `json:"showLoadPay"                bun:"show_load_pay,type:BOOLEAN,notnull"`
	ShowPayEstimates           bool `json:"showPayEstimates"           bun:"show_pay_estimates,type:BOOLEAN,notnull"`
	AllowExpenseSubmission     bool `json:"allowExpenseSubmission"     bun:"allow_expense_submission,type:BOOLEAN,notnull"`
	RequireExpenseReceipt      bool `json:"requireExpenseReceipt"      bun:"require_expense_receipt,type:BOOLEAN,notnull"`
	AllowSettlementDisputes    bool `json:"allowSettlementDisputes"    bun:"allow_settlement_disputes,type:BOOLEAN,notnull"`
	AllowProfileDocumentUpload bool `json:"allowProfileDocumentUpload" bun:"allow_profile_document_upload,type:BOOLEAN,notnull"`
	AllowContactInfoEdit       bool `json:"allowContactInfoEdit"       bun:"allow_contact_info_edit,type:BOOLEAN,notnull"`
	AllowPtoRequests           bool `json:"allowPtoRequests"           bun:"allow_pto_requests,type:BOOLEAN,notnull"`
	SendCredentialReminders    bool `json:"sendCredentialReminders"    bun:"send_credential_reminders,type:BOOLEAN,notnull"`
	// RequireContactChangeApproval makes a driver's own contact edits wait on
	// the office instead of landing straight on the record. Off keeps the
	// behaviour every existing carrier relies on.
	RequireContactChangeApproval bool `json:"requireContactChangeApproval" bun:"require_contact_change_approval,type:BOOLEAN,notnull"`

	// DriverDigestCadence bundles a driver's obligations into one notice
	// instead of one each. Immediate is the default so no carrier silently
	// loses notices they already rely on.
	DriverDigestCadence DriverDigestCadence `json:"driverDigestCadence" bun:"driver_digest_cadence,type:driver_digest_cadence_enum,notnull,default:'Immediate'"`
	// DriverDigestWeekday is the day the weekly digest goes out, 0 = Sunday.
	// Ignored unless the cadence is Weekly.
	DriverDigestWeekday int16 `json:"driverDigestWeekday" bun:"driver_digest_weekday,type:SMALLINT,notnull,default:1"`

	EnableDetentionAlerts          bool  `json:"enableDetentionAlerts"          bun:"enable_detention_alerts,type:BOOLEAN,notnull"`
	DetentionAlertThresholdMinutes int16 `json:"detentionAlertThresholdMinutes" bun:"detention_alert_threshold_minutes,type:INTEGER,notnull,default:120"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (dc *DashControl) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(dc,
		validation.Field(&dc.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&dc.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&dc.DetentionAlertThresholdMinutes,
			validation.When(dc.EnableDetentionAlerts,
				validation.Required.Error(
					"Detention alert threshold must be between 15 minutes and 24 hours",
				),
				validation.Min(minDetentionAlertMinutes).Error(
					"Detention alert threshold must be between 15 minutes and 24 hours",
				),
				validation.Max(maxDetentionAlertMinutes).Error(
					"Detention alert threshold must be between 15 minutes and 24 hours",
				),
			),
		),
		// Each of these settings only means anything while the capability it
		// depends on is on, so the pair is refused rather than silently ignored.
		validation.Field(&dc.AllowLoadRefusals,
			validation.When(!dc.RequireLoadAcknowledgment, validation.Empty.Error(
				"Load refusals require load acknowledgment to be enabled",
			)),
		),
		validation.Field(&dc.ShowPayEstimates,
			validation.When(!dc.ShowLoadPay, validation.Empty.Error(
				"Pay estimates require per-load pay visibility to be enabled",
			)),
		),
		validation.Field(&dc.RequireExpenseReceipt,
			validation.When(!dc.AllowExpenseSubmission, validation.Empty.Error(
				"Receipt requirement only applies when expense submission is enabled",
			)),
		),
		validation.Field(&dc.DriverDigestCadence,
			validation.Required.Error("Digest cadence is required"),
			validation.In(DigestImmediate, DigestDaily, DigestWeekly).Error(
				"Digest cadence must be Immediate, Daily or Weekly",
			),
		),
		validation.Field(&dc.DriverDigestWeekday,
			validation.Min(int16(0)).Error("Digest day must be a day of the week"),
			validation.Max(int16(6)).Error("Digest day must be a day of the week"),
		),
	))

	// A digest with reminders switched off would collect obligations and send
	// nothing, which reads as a working setting that quietly does nothing.
	if dc.DriverDigestCadence.Bundles() && !dc.SendCredentialReminders {
		multiErr.Add(
			"driverDigestCadence",
			errortypes.ErrInvalidOperation,
			"A digest needs driver reminders switched on — there would be nothing to bundle",
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
		if dc.DriverDigestCadence == "" {
			dc.DriverDigestCadence = DigestImmediate
		}
		dc.CreatedAt = now
	case *bun.UpdateQuery:
		dc.UpdatedAt = now
	}
	return nil
}
