package tenant

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	_ validationframework.TenantedEntity = (*DashControl)(nil)
)

type DashControl struct {
	keel.Table `keel:"dash_controls"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:dashc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	RequireLoadAcknowledgment  bool `json:"requireLoadAcknowledgment" keel:"require_load_acknowledgment"`
	AllowLoadRefusals          bool `json:"allowLoadRefusals" keel:"allow_load_refusals"`
	AllowStopActions           bool `json:"allowStopActions" keel:"allow_stop_actions"`
	AllowLoadDocumentUpload    bool `json:"allowLoadDocumentUpload" keel:"allow_load_document_upload"`
	AllowLoadComments          bool `json:"allowLoadComments" keel:"allow_load_comments"`
	ShowLoadPay                bool `json:"showLoadPay" keel:"show_load_pay"`
	ShowPayEstimates           bool `json:"showPayEstimates" keel:"show_pay_estimates"`
	AllowExpenseSubmission     bool `json:"allowExpenseSubmission" keel:"allow_expense_submission"`
	RequireExpenseReceipt      bool `json:"requireExpenseReceipt" keel:"require_expense_receipt"`
	AllowSettlementDisputes    bool `json:"allowSettlementDisputes" keel:"allow_settlement_disputes"`
	AllowProfileDocumentUpload bool `json:"allowProfileDocumentUpload" keel:"allow_profile_document_upload"`
	AllowContactInfoEdit       bool `json:"allowContactInfoEdit" keel:"allow_contact_info_edit"`
	AllowPtoRequests           bool `json:"allowPtoRequests" keel:"allow_pto_requests"`
	SendCredentialReminders    bool `json:"sendCredentialReminders" keel:"send_credential_reminders"`
	// RequireContactChangeApproval makes a driver's own contact edits wait on
	// the office instead of landing straight on the record. Off keeps the
	// behaviour every existing carrier relies on.
	RequireContactChangeApproval bool `json:"requireContactChangeApproval" keel:"require_contact_change_approval"`

	// DriverDigestCadence bundles a driver's obligations into one notice
	// instead of one each. Immediate is the default so no carrier silently
	// loses notices they already rely on.
	DriverDigestCadence DriverDigestCadence `json:"driverDigestCadence" keel:"driver_digest_cadence"`
	// DriverDigestWeekday is the day the weekly digest goes out, 0 = Sunday.
	// Ignored unless the cadence is Weekly.
	DriverDigestWeekday int16 `json:"driverDigestWeekday" keel:"driver_digest_weekday"`

	EnableDetentionAlerts          bool  `json:"enableDetentionAlerts" keel:"enable_detention_alerts"`
	DetentionAlertThresholdMinutes int16 `json:"detentionAlertThresholdMinutes" keel:"detention_alert_threshold_minutes"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty"`
	Organization *Organization `json:"organization,omitempty"`
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

func (dc *DashControl) BeforeCreate(_ context.Context) error {
	if dc.DriverDigestCadence == "" {
		dc.DriverDigestCadence = DigestImmediate
	}
	if dc.DriverDigestWeekday == 0 {
		dc.DriverDigestWeekday = 1
	}
	if dc.DetentionAlertThresholdMinutes == 0 {
		dc.DetentionAlertThresholdMinutes = 120
	}

	return nil
}
