package detention

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const (
	// evidenceHeadLength is the width of the hex-encoded SHA-256 digest that
	// heads the occurrence's evidence chain.
	evidenceHeadLength = 64
	// currencyCodeLength is the width of an ISO 4217 code.
	currencyCodeLength = 3
)

// DetentionOccurrence is the detention event at exactly one stop. It is the
// unit that gets billed, disputed, waived, and audited. Once the clock stops
// and the charge is billed the record is frozen: recalculation must never
// silently rewrite a number a customer has already been shown.
type DetentionOccurrence struct {
	keel.Table     `keel:"detention_occurrences"`
	ID             pulid.ID `keel:"id,pk,autoid:dto_"          json:"id"`
	BusinessUnitID pulid.ID `keel:"business_unit_id,pk,tenant" json:"businessUnitId"`
	OrganizationID pulid.ID `keel:"organization_id,pk,tenant"  json:"organizationId"`
	ShipmentID     pulid.ID `keel:"shipment_id"                json:"shipmentId"`
	ShipmentMoveID pulid.ID `keel:"shipment_move_id"           json:"shipmentMoveId"`
	StopID         pulid.ID `keel:"stop_id"                    json:"stopId"`
	CustomerID     pulid.ID `keel:"customer_id"                json:"customerId"`
	LocationID     pulid.ID `keel:"location_id"                json:"locationId"`

	DetentionPolicyID *pulid.ID                       `json:"detentionPolicyId" keel:"detention_policy_id"`
	PolicySnapshot    *detentionpolicy.PolicySnapshot `json:"policySnapshot"    keel:"policy_snapshot,jsonb"`
	CalculationTrace  *CalculationTrace               `json:"calculationTrace"  keel:"calculation_trace,jsonb"`

	StopType           shipment.StopType         `json:"stopType"           keel:"stop_type"`
	ScheduleType       shipment.StopScheduleType `json:"scheduleType"       keel:"schedule_type"`
	AppointmentStart   *int64                    `json:"appointmentStart"   keel:"appointment_start"`
	AppointmentEnd     *int64                    `json:"appointmentEnd"     keel:"appointment_end"`
	ArrivedAt          *int64                    `json:"arrivedAt"          keel:"arrived_at"`
	DepartedAt         *int64                    `json:"departedAt"         keel:"departed_at"`
	ClockStartAt       int64                     `json:"clockStartAt"       keel:"clock_start_at"`
	ClockStopAt        *int64                    `json:"clockStopAt"        keel:"clock_stop_at"`
	FreeTimeExpiresAt  int64                     `json:"freeTimeExpiresAt"  keel:"free_time_expires_at"`
	NoticeDueAt        *int64                    `json:"noticeDueAt"        keel:"notice_due_at"`
	NoticeDeadlineAt   *int64                    `json:"noticeDeadlineAt"   keel:"notice_deadline_at"`
	IsOpen             bool                      `json:"isOpen"             keel:"is_open"`
	ArrivedLate        bool                      `json:"arrivedLate"        keel:"arrived_late"`
	LateByMinutes      int32                     `json:"lateByMinutes"      keel:"late_by_minutes"`
	FreeMinutesGranted int32                     `json:"freeMinutesGranted" keel:"free_minutes_granted"`
	RawDwellMinutes    int32                     `json:"rawDwellMinutes"    keel:"raw_dwell_minutes"`
	BillableMinutes    int32                     `json:"billableMinutes"    keel:"billable_minutes"`
	RoundedMinutes     int32                     `json:"roundedMinutes"     keel:"rounded_minutes"`
	BillableUnits      decimal.Decimal           `json:"billableUnits"      keel:"billable_units"`
	GrossAmount        decimal.Decimal           `json:"grossAmount"        keel:"gross_amount"`
	BillableAmount     decimal.Decimal           `json:"billableAmount"     keel:"billable_amount"`
	DriverPayMinutes   int32                     `json:"driverPayMinutes"   keel:"driver_pay_minutes"`
	DriverPayAmount    decimal.NullDecimal       `json:"driverPayAmount"    keel:"driver_pay_amount,default"`
	NetMargin          decimal.NullDecimal       `json:"netMargin"          keel:"net_margin"`
	CapApplied         CapKind                   `json:"capApplied"         keel:"cap_applied"`
	ConvertedToLayover bool                      `json:"convertedToLayover" keel:"converted_to_layover"`
	Currency           string                    `json:"currency"           keel:"currency"`

	Status             OccurrenceStatus   `json:"status"             keel:"status"`
	NotificationStatus NotificationStatus `json:"notificationStatus" keel:"notification_status"`
	NoticeSentAt       *int64             `json:"noticeSentAt"       keel:"notice_sent_at"`
	SuppressedByGate   bool               `json:"suppressedByGate"   keel:"suppressed_by_gate"`
	RequiresApproval   bool               `json:"requiresApproval"   keel:"requires_approval"`
	ApprovedByID       *pulid.ID          `json:"approvedById"       keel:"approved_by_id"`
	ApprovedAt         *int64             `json:"approvedAt"         keel:"approved_at"`
	WaiverReason       *WaiverReason      `json:"waiverReason"       keel:"waiver_reason"`
	WaiverNote         string             `json:"waiverNote"         keel:"waiver_note,nullzero"`
	WaivedByID         *pulid.ID          `json:"waivedById"         keel:"waived_by_id"`
	WaivedAt           *int64             `json:"waivedAt"           keel:"waived_at"`
	WaivedAmount       decimal.Decimal    `json:"waivedAmount"       keel:"waived_amount"`
	DisputeNote        string             `json:"disputeNote"        keel:"dispute_note,nullzero"`
	DisputedAt         *int64             `json:"disputedAt"         keel:"disputed_at"`

	CollectabilityScore int16     `json:"collectabilityScore" keel:"collectability_score"`
	EvidenceHead        string    `json:"evidenceHead"        keel:"evidence_head,nullzero"`
	AdditionalChargeID  *pulid.ID `json:"additionalChargeId"  keel:"additional_charge_id"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	LocationName      string `json:"locationName,omitempty"      keel:"location_name,computed"`
	CustomerName      string `json:"customerName,omitempty"      keel:"customer_name,computed"`
	ShipmentProNumber string `json:"shipmentProNumber,omitempty" keel:"shipment_pro_number,computed"`

	BusinessUnit    *tenant.BusinessUnit                   `json:"-"`
	Organization    *tenant.Organization                   `json:"-"`
	DetentionPolicy *detentionpolicy.DetentionPolicy       `json:"detentionPolicy,omitempty" keel:"belongs:detention_policy_id"`
	Evidence        []*detentionevidence.DetentionEvidence `json:"evidence,omitempty"        keel:"many:detention_occurrence_id"`
	Notices         []*detentionnotice.DetentionNotice     `json:"notices,omitempty"         keel:"many:detention_occurrence_id"`
}

func (o *DetentionOccurrence) GetID() pulid.ID { return o.ID }

func (o *DetentionOccurrence) GetCreatedAt() int64 { return o.CreatedAt }

func (o *DetentionOccurrence) GetOrganizationID() pulid.ID { return o.OrganizationID }

func (o *DetentionOccurrence) GetBusinessUnitID() pulid.ID { return o.BusinessUnitID }

func (o *DetentionOccurrence) GetTableName() string { return "detention_occurrences" }

// GetPostgresSearchConfig declares no search vector: occurrences are reached by
// shipment, facility, customer, or status rather than by free text.
func (o *DetentionOccurrence) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:       "dto",
		UseSearchVector:  false,
		SearchableFields: []domaintypes.SearchableField{},
	}
}

// IsFrozen reports whether the recalculation engine must leave this record
// alone. A billed, waived, or disputed occurrence represents a number the
// customer has already seen.
func (o *DetentionOccurrence) IsFrozen() bool {
	return o.Status.IsTerminal()
}

// MinutesUntilNoticeDue returns how long dispatch has left to send the required
// notice. Negative means the window has closed.
func (o *DetentionOccurrence) MinutesUntilNoticeDue(now int64) *int32 {
	if o.NoticeDueAt == nil {
		return nil
	}
	remaining := intutils.SafeToInt32((*o.NoticeDueAt - now) / 60)
	return &remaining
}

// NoticeWindowOpen reports whether a notice can still satisfy the policy.
func (o *DetentionOccurrence) NoticeWindowOpen(now int64) bool {
	if o.NoticeDeadlineAt == nil {
		return o.NotificationStatus == NotificationStatusPending
	}
	return now <= *o.NoticeDeadlineAt &&
		o.NotificationStatus == NotificationStatusPending
}

// Waive suppresses the charge with a coded reason so discretionary revenue loss
// stays measurable.
func (o *DetentionOccurrence) Waive(
	reason WaiverReason,
	note string,
	userID pulid.ID,
	now int64,
) error {
	if o.Status == OccurrenceStatusBilled {
		return errors.New("a billed occurrence cannot be waived; issue an adjustment instead")
	}
	if o.Status == OccurrenceStatusWaived {
		return errors.New("occurrence is already waived")
	}

	o.WaivedAmount = o.BillableAmount
	o.Status = OccurrenceStatusWaived
	o.WaiverReason = &reason
	o.WaiverNote = note
	o.WaivedByID = &userID
	o.WaivedAt = &now
	o.BillableAmount = decimal.Zero
	o.NetMargin = decimal.NewNullDecimal(o.BillableAmount.Sub(o.DriverPayAmount.Decimal))

	return nil
}

// Approve clears an occurrence that tripped the approval threshold.
func (o *DetentionOccurrence) Approve(userID pulid.ID, now int64) error {
	if o.Status != OccurrenceStatusPending {
		return errors.New("only a pending occurrence can be approved")
	}

	o.Status = OccurrenceStatusApproved
	o.ApprovedByID = &userID
	o.ApprovedAt = &now
	o.RequiresApproval = false

	return nil
}

// HoldsBilling reports whether the charge must stay off every invoice until a
// person decides it. A pending charge that merely was not auto-approved keeps
// billing; only one the policy or a person put in front of an approver holds.
func (o *DetentionOccurrence) HoldsBilling() bool {
	return o.Status == OccurrenceStatusPending && o.RequiresApproval
}

// BillingHoldReason says why a held charge is waiting on a person.
func (o *DetentionOccurrence) BillingHoldReason() BillingHoldReason {
	if snap := o.PolicySnapshot; snap != nil {
		if snap.RequireApprovalOverAmount.Valid &&
			o.BillableAmount.GreaterThan(snap.RequireApprovalOverAmount.Decimal) {
			return BillingHoldReasonOverApprovalThreshold
		}
		if snap.NotificationRequirement == detentionpolicy.NotificationRequirementRequired &&
			snap.UnnotifiedBehavior == detentionpolicy.UnnotifiedBehaviorFlag &&
			!o.NotificationStatus.SatisfiesRequirement() {
			return BillingHoldReasonNoticeNotSent
		}
	}

	return BillingHoldReasonEscalated
}

// MarkBilled freezes a billable charge once an invoice carries it.
func (o *DetentionOccurrence) MarkBilled() error {
	if o.Status == OccurrenceStatusBilled {
		return errors.New("occurrence is already billed")
	}
	if o.Status != OccurrenceStatusPending && o.Status != OccurrenceStatusApproved {
		return errors.New("only a pending or approved occurrence can be billed")
	}

	o.Status = OccurrenceStatusBilled

	return nil
}

// ReleaseBilling returns a billed charge to approved once no invoice carries
// it any more. It was billable when it was billed, so it stays approved rather
// than going back in front of an approver.
func (o *DetentionOccurrence) ReleaseBilling() error {
	if o.Status != OccurrenceStatusBilled {
		return errors.New("only a billed occurrence can be released")
	}

	o.Status = OccurrenceStatusApproved

	return nil
}

type InvoiceBillingChange uint8

const (
	InvoiceBillingUnchanged InvoiceBillingChange = iota
	InvoiceBillingMarked
	InvoiceBillingReleased
)

func InvoiceSettledStatuses() []OccurrenceStatus {
	return []OccurrenceStatus{
		OccurrenceStatusPending,
		OccurrenceStatusApproved,
		OccurrenceStatusBilled,
		OccurrenceStatusDisputed,
	}
}

func (o *DetentionOccurrence) FollowInvoices(invoiced bool) (InvoiceBillingChange, error) {
	if !o.Status.IsBillable() {
		return InvoiceBillingUnchanged, nil
	}

	switch {
	case invoiced && o.Status != OccurrenceStatusBilled:
		return InvoiceBillingMarked, o.MarkBilled()
	case !invoiced && o.Status == OccurrenceStatusBilled:
		return InvoiceBillingReleased, o.ReleaseBilling()
	default:
		return InvoiceBillingUnchanged, nil
	}
}

// Dispute records a customer rejection so the claim can be worked without
// losing the original computation.
func (o *DetentionOccurrence) Dispute(note string, now int64) error {
	if o.Status == OccurrenceStatusWaived {
		return errors.New("a waived occurrence cannot be disputed")
	}

	o.Status = OccurrenceStatusDisputed
	o.DisputeNote = note
	o.DisputedAt = &now

	return nil
}

// Escalate hands a clock to a person. The money is untouched: escalating
// says the charge cannot be worked automatically from here — the notice
// window has closed, a gate is holding it back, or the customer has no
// recipients on file — not that the figures are wrong. Whoever picks it up
// decides whether to waive, approve or dispute.
func (o *DetentionOccurrence) Escalate(now int64) error {
	switch o.Status {
	case OccurrenceStatusWaived:
		return errors.New("a waived occurrence has nothing left to escalate")
	case OccurrenceStatusDisputed:
		return errors.New("a disputed occurrence is already with a person")
	}
	if o.RequiresApproval {
		return errors.New("this occurrence is already waiting on a person")
	}

	o.RequiresApproval = true
	if o.NoticeDeadlineAt != nil && *o.NoticeDeadlineAt < now &&
		o.NotificationStatus == NotificationStatusPending {
		o.NotificationStatus = NotificationStatusMissed
	}

	return nil
}

func (o *DetentionOccurrence) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(o,
		validation.Field(&o.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&o.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&o.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(&o.ShipmentMoveID, validation.Required.Error("Shipment move is required")),
		validation.Field(&o.StopID, validation.Required.Error("Stop is required")),
		validation.Field(&o.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(&o.LocationID, validation.Required.Error("Location is required")),
		validation.Field(&o.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnumFunc(OccurrenceStatusFromString, "Status is invalid"),
		),
		validation.Field(&o.NotificationStatus,
			domainvalidation.ValidEnumFunc(
				NotificationStatusFromString, "Notification status is invalid",
			),
		),
		validation.Field(&o.CapApplied,
			validation.In(
				CapKindNone,
				CapKindMaxMinutes,
				CapKindPerStopAmount,
				CapKindPerDayAmount,
				CapKindPerShipment,
				CapKindLayoverBoundary,
			).Error("Applied cap is invalid"),
		),
		validation.Field(&o.ClockStartAt,
			validation.Required.Error("Clock start is required"),
			validation.Min(int64(1)).Error("Clock start must be a valid timestamp"),
		),
		// The clock is what the charge is computed from, so an inverted window
		// would bill negative time.
		validation.Field(&o.ClockStopAt,
			validation.Min(o.ClockStartAt).Error("Clock stop cannot precede clock start"),
		),
		validation.Field(&o.LateByMinutes,
			validation.Min(int32(0)).Error("Late by minutes cannot be negative"),
		),
		validation.Field(&o.FreeMinutesGranted,
			validation.Min(int32(0)).Error("Free minutes granted cannot be negative"),
		),
		validation.Field(&o.RawDwellMinutes,
			validation.Min(int32(0)).Error("Raw dwell minutes cannot be negative"),
		),
		validation.Field(&o.BillableMinutes,
			validation.Min(int32(0)).Error("Billable minutes cannot be negative"),
		),
		validation.Field(&o.RoundedMinutes,
			validation.Min(int32(0)).Error("Rounded minutes cannot be negative"),
		),
		validation.Field(&o.DriverPayMinutes,
			validation.Min(int32(0)).Error("Driver pay minutes cannot be negative"),
		),
		validation.Field(&o.Currency,
			validation.Required.Error("Currency is required"),
			validation.Length(currencyCodeLength, currencyCodeLength).
				Error("Currency must be a three letter code"),
		),
		validation.Field(&o.EvidenceHead,
			validation.Length(evidenceHeadLength, evidenceHeadLength).
				Error("Evidence head must be a 64 character digest"),
		),
		// A waiver writes off money, so the reason it was written off is part of
		// the record rather than something reconstructed from an audit trail.
		validation.Field(&o.WaiverReason,
			validation.When(
				o.Status == OccurrenceStatusWaived,
				validation.Required.Error("A waiver requires a coded reason"),
			),
			domainvalidation.ValidEnumFunc(WaiverReasonFromString, "Waiver reason is invalid"),
		),
	))
}

func (o *DetentionOccurrence) BeforeCreate(ctx context.Context) error {
	if o.CapApplied == "" {
		o.CapApplied = CapKindNone
	}
	if o.Currency == "" {
		o.Currency = money.DefaultCurrencyCode
	}
	if o.Status == "" {
		o.Status = OccurrenceStatusAccruing
	}
	if o.NotificationStatus == "" {
		o.NotificationStatus = NotificationStatusNotRequired
	}
	if !o.NetMargin.Valid {
		o.NetMargin = decimal.NewNullDecimal(o.BillableAmount.Sub(o.DriverPayAmount.Decimal))
	}

	return nil
}

func (o *DetentionOccurrence) BeforeUpdate(ctx context.Context) error {
	if o.Currency == "" {
		o.Currency = "USD"
	}
	if o.CapApplied == "" {
		o.CapApplied = CapKindNone
	}
	if o.Status == "" {
		o.Status = OccurrenceStatusAccruing
	}
	if o.NotificationStatus == "" {
		o.NotificationStatus = NotificationStatusNotRequired
	}

	return nil
}

func (DetentionOccurrence) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_detention_occurrences_stop",
			Message: "Stop already has a detention occurrence",
		},
		{
			Name:    "fk_detention_occurrences_shipment",
			Message: "Shipment does not exist in your organization",
		},
		{
			Name:    "fk_detention_occurrences_policy",
			Message: "Detention policy does not exist in your organization",
		},
		{
			Name:    "fk_detention_occurrences_additional_charge",
			Message: "Additional charge does not exist",
		},
	}
}
