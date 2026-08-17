package detention

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/shared/intutils"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*DetentionOccurrence)(nil)

// DetentionOccurrence is the detention event at exactly one stop. It is the
// unit that gets billed, disputed, waived, and audited. Once the clock stops
// and the charge is billed the record is frozen: recalculation must never
// silently rewrite a number a customer has already been shown.
type DetentionOccurrence struct {
	bun.BaseModel             `bun:"table:detention_occurrences,alias:dto" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ShipmentID     pulid.ID `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull"`
	ShipmentMoveID pulid.ID `json:"shipmentMoveId" bun:"shipment_move_id,type:VARCHAR(100),notnull"`
	StopID         pulid.ID `json:"stopId"         bun:"stop_id,type:VARCHAR(100),notnull"`
	CustomerID     pulid.ID `json:"customerId"     bun:"customer_id,type:VARCHAR(100),notnull"`
	LocationID     pulid.ID `json:"locationId"     bun:"location_id,type:VARCHAR(100),notnull"`

	DetentionPolicyID *pulid.ID         `json:"detentionPolicyId" bun:"detention_policy_id,type:VARCHAR(100),nullzero"`
	PolicySnapshot    *PolicySnapshot   `json:"policySnapshot"    bun:"policy_snapshot,type:JSONB,nullzero"`
	CalculationTrace  *CalculationTrace `json:"calculationTrace"  bun:"calculation_trace,type:JSONB,nullzero"`

	StopType           shipment.StopType         `json:"stopType"           bun:"stop_type,type:stop_type_enum,notnull"`
	ScheduleType       shipment.StopScheduleType `json:"scheduleType"       bun:"schedule_type,type:stop_schedule_type_enum,notnull"`
	AppointmentStart   *int64                    `json:"appointmentStart"   bun:"appointment_start,type:BIGINT,nullzero"`
	AppointmentEnd     *int64                    `json:"appointmentEnd"     bun:"appointment_end,type:BIGINT,nullzero"`
	ArrivedAt          *int64                    `json:"arrivedAt"          bun:"arrived_at,type:BIGINT,nullzero"`
	DepartedAt         *int64                    `json:"departedAt"         bun:"departed_at,type:BIGINT,nullzero"`
	ClockStartAt       int64                     `json:"clockStartAt"       bun:"clock_start_at,type:BIGINT,notnull"`
	ClockStopAt        *int64                    `json:"clockStopAt"        bun:"clock_stop_at,type:BIGINT,nullzero"`
	FreeTimeExpiresAt  int64                     `json:"freeTimeExpiresAt"  bun:"free_time_expires_at,type:BIGINT,notnull"`
	NoticeDueAt        *int64                    `json:"noticeDueAt"        bun:"notice_due_at,type:BIGINT,nullzero"`
	NoticeDeadlineAt   *int64                    `json:"noticeDeadlineAt"   bun:"notice_deadline_at,type:BIGINT,nullzero"`
	IsOpen             bool                      `json:"isOpen"             bun:"is_open,type:BOOLEAN,notnull,default:true"`
	ArrivedLate        bool                      `json:"arrivedLate"        bun:"arrived_late,type:BOOLEAN,notnull"`
	LateByMinutes      int32                     `json:"lateByMinutes"      bun:"late_by_minutes,type:INTEGER,notnull"`
	FreeMinutesGranted int32                     `json:"freeMinutesGranted" bun:"free_minutes_granted,type:INTEGER,notnull"`
	RawDwellMinutes    int32                     `json:"rawDwellMinutes"    bun:"raw_dwell_minutes,type:INTEGER,notnull"`
	BillableMinutes    int32                     `json:"billableMinutes"    bun:"billable_minutes,type:INTEGER,notnull"`
	RoundedMinutes     int32                     `json:"roundedMinutes"     bun:"rounded_minutes,type:INTEGER,notnull"`
	BillableUnits      decimal.Decimal           `json:"billableUnits"      bun:"billable_units,type:NUMERIC(12,4),notnull,default:0"`
	GrossAmount        decimal.Decimal           `json:"grossAmount"        bun:"gross_amount,type:NUMERIC(19,4),notnull,default:0"`
	BillableAmount     decimal.Decimal           `json:"billableAmount"     bun:"billable_amount,type:NUMERIC(19,4),notnull,default:0"`
	DriverPayMinutes   int32                     `json:"driverPayMinutes"   bun:"driver_pay_minutes,type:INTEGER,notnull"`
	DriverPayAmount    decimal.Decimal           `json:"driverPayAmount"    bun:"driver_pay_amount,type:NUMERIC(19,4),notnull,default:0"`
	NetMargin          decimal.Decimal           `json:"netMargin"          bun:"net_margin,type:NUMERIC(19,4),notnull,default:0"`
	CapApplied         CapKind                   `json:"capApplied"         bun:"cap_applied,type:detention_cap_kind_enum,notnull,default:'None'"`
	ConvertedToLayover bool                      `json:"convertedToLayover" bun:"converted_to_layover,type:BOOLEAN,notnull"`
	Currency           string                    `json:"currency"           bun:"currency,type:VARCHAR(3),notnull,default:'USD'"`

	Status             OccurrenceStatus   `json:"status"             bun:"status,type:detention_occurrence_status_enum,notnull,default:'Accruing'"`
	NotificationStatus NotificationStatus `json:"notificationStatus" bun:"notification_status,type:detention_notification_status_enum,notnull,default:'NotRequired'"`
	NoticeSentAt       *int64             `json:"noticeSentAt"       bun:"notice_sent_at,type:BIGINT,nullzero"`
	SuppressedByGate   bool               `json:"suppressedByGate"   bun:"suppressed_by_gate,type:BOOLEAN,notnull"`
	RequiresApproval   bool               `json:"requiresApproval"   bun:"requires_approval,type:BOOLEAN,notnull"`
	ApprovedByID       *pulid.ID          `json:"approvedById"       bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	ApprovedAt         *int64             `json:"approvedAt"         bun:"approved_at,type:BIGINT,nullzero"`
	WaiverReason       *WaiverReason      `json:"waiverReason"       bun:"waiver_reason,type:detention_waiver_reason_enum,nullzero"`
	WaiverNote         string             `json:"waiverNote"         bun:"waiver_note,type:TEXT,nullzero"`
	WaivedByID         *pulid.ID          `json:"waivedById"         bun:"waived_by_id,type:VARCHAR(100),nullzero"`
	WaivedAt           *int64             `json:"waivedAt"           bun:"waived_at,type:BIGINT,nullzero"`
	WaivedAmount       decimal.Decimal    `json:"waivedAmount"       bun:"waived_amount,type:NUMERIC(19,4),notnull,default:0"`
	DisputeNote        string             `json:"disputeNote"        bun:"dispute_note,type:TEXT,nullzero"`
	DisputedAt         *int64             `json:"disputedAt"         bun:"disputed_at,type:BIGINT,nullzero"`

	CollectabilityScore int16     `json:"collectabilityScore" bun:"collectability_score,type:SMALLINT,notnull"`
	EvidenceHead        string    `json:"evidenceHead"        bun:"evidence_head,type:VARCHAR(64),nullzero"`
	AdditionalChargeID  *pulid.ID `json:"additionalChargeId"  bun:"additional_charge_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	LocationName      string `json:"locationName,omitempty"      bun:"location_name,scanonly"`
	CustomerName      string `json:"customerName,omitempty"      bun:"customer_name,scanonly"`
	ShipmentProNumber string `json:"shipmentProNumber,omitempty" bun:"shipment_pro_number,scanonly"`

	BusinessUnit    *tenant.BusinessUnit `json:"-"                         bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization    *tenant.Organization `json:"-"                         bun:"rel:belongs-to,join:organization_id=id"`
	DetentionPolicy *DetentionPolicy     `json:"detentionPolicy,omitempty" bun:"rel:belongs-to,join:detention_policy_id=id"`
	Evidence        []*DetentionEvidence `json:"evidence,omitempty"        bun:"rel:has-many,join:id=detention_occurrence_id"`
	Notices         []*DetentionNotice   `json:"notices,omitempty"         bun:"rel:has-many,join:id=detention_occurrence_id"`
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
	o.NetMargin = o.BillableAmount.Sub(o.DriverPayAmount)

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
			validation.Length(hashLength, hashLength).
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

func (o *DetentionOccurrence) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

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

	switch query.(type) {
	case *bun.InsertQuery:
		if o.ID.IsNil() {
			o.ID = pulid.MustNew("dto_")
		}
		o.CreatedAt = now
		o.UpdatedAt = now
	case *bun.UpdateQuery:
		o.UpdatedAt = now
	}

	return nil
}
