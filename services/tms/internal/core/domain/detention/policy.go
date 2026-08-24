package detention

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	SpecificityWeightLocation     int32 = 32
	SpecificityWeightCustomer     int32 = 16
	SpecificityWeightCommodity    int32 = 8
	SpecificityWeightServiceType  int32 = 4
	SpecificityWeightShipmentType int32 = 2
	SpecificityWeightStopType     int32 = 1

	MaxFreeMinutes        int32 = 10080
	DefaultFreeMinutes    int32 = 120
	DefaultIncrementMins  int16 = 15
	LayoverBoundaryMinute int32 = 1440
)

var (
	_ bun.BeforeAppendModelHook          = (*DetentionPolicy)(nil)
	_ validationframework.TenantedEntity = (*DetentionPolicy)(nil)
	_ domaintypes.PostgresSearchable     = (*DetentionPolicy)(nil)
)

type DetentionPolicy struct {
	bun.BaseModel             `bun:"table:detention_policies,alias:dtp" json:"-"`
	pagination.CursorValueSet `bun:",embed"                             json:"-"`

	ID             pulid.ID     `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID     `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID     `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Name           string       `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Code           string       `json:"code"           bun:"code,type:VARCHAR(50),notnull"`
	Description    string       `json:"description"    bun:"description,type:TEXT,nullzero"`
	Status         PolicyStatus `json:"status"         bun:"status,type:detention_policy_status_enum,notnull,default:'Draft'"`

	IsOrgDefault     bool       `json:"isOrgDefault"     bun:"is_org_default,type:BOOLEAN,notnull"`
	Priority         int16      `json:"priority"         bun:"priority,type:SMALLINT,notnull"`
	SpecificityScore int32      `json:"specificityScore" bun:"specificity_score,type:INTEGER,notnull"`
	CustomerID       *pulid.ID  `json:"customerId"       bun:"customer_id,type:VARCHAR(100),nullzero"`
	LocationID       *pulid.ID  `json:"locationId"       bun:"location_id,type:VARCHAR(100),nullzero"`
	ShipmentTypeIDs  []pulid.ID `json:"shipmentTypeIds"  bun:"shipment_type_ids,type:JSONB,nullzero"`
	ServiceTypeIDs   []pulid.ID `json:"serviceTypeIds"   bun:"service_type_ids,type:JSONB,nullzero"`
	CommodityIDs     []pulid.ID `json:"commodityIds"     bun:"commodity_ids,type:JSONB,nullzero"`

	StopTypes                  []shipment.StopType `json:"stopTypes"                  bun:"stop_types,type:JSONB,nullzero"`
	AppointmentStopsOnly       bool                `json:"appointmentStopsOnly"       bun:"appointment_stops_only,type:BOOLEAN,notnull"`
	EffectiveStartDate         *int64              `json:"effectiveStartDate"         bun:"effective_start_date,type:BIGINT,nullzero"`
	EffectiveEndDate           *int64              `json:"effectiveEndDate"           bun:"effective_end_date,type:BIGINT,nullzero"`
	ClockStartBasis            ClockStartBasis     `json:"clockStartBasis"            bun:"clock_start_basis,type:detention_clock_start_basis_enum,notnull,default:'LaterOfArrivalOrAppointment'"`
	LateArrivalRule            LateArrivalRule     `json:"lateArrivalRule"            bun:"late_arrival_rule,type:detention_late_arrival_rule_enum,notnull,default:'NoEffect'"`
	LateArrivalGraceMinutes    int16               `json:"lateArrivalGraceMinutes"    bun:"late_arrival_grace_minutes,type:SMALLINT,notnull"`
	BillingFreeMinutes         int32               `json:"billingFreeMinutes"         bun:"billing_free_minutes,type:INTEGER,notnull,default:120"`
	PickupFreeMinutes          *int32              `json:"pickupFreeMinutes"          bun:"pickup_free_minutes,type:INTEGER,nullzero"`
	DeliveryFreeMinutes        *int32              `json:"deliveryFreeMinutes"        bun:"delivery_free_minutes,type:INTEGER,nullzero"`
	PayFreeMinutes             *int32              `json:"payFreeMinutes"             bun:"pay_free_minutes,type:INTEGER,nullzero"`
	MinimumBillableMinutes     int32               `json:"minimumBillableMinutes"     bun:"minimum_billable_minutes,type:INTEGER,notnull"`
	BillingIncrementMinutes    int16               `json:"billingIncrementMinutes"    bun:"billing_increment_minutes,type:SMALLINT,notnull,default:15"`
	RoundingMode               RoundingMode        `json:"roundingMode"               bun:"rounding_mode,type:detention_rounding_mode_enum,notnull,default:'Up'"`
	RateSource                 RateSource          `json:"rateSource"                 bun:"rate_source,type:detention_rate_source_enum,notnull,default:'Accessorial'"`
	AccessorialChargeID        pulid.ID            `json:"accessorialChargeId"        bun:"accessorial_charge_id,type:VARCHAR(100),notnull"`
	MaxBillableMinutesPerStop  *int32              `json:"maxBillableMinutesPerStop"  bun:"max_billable_minutes_per_stop,type:INTEGER,nullzero"`
	MaxChargePerStop           decimal.NullDecimal `json:"maxChargePerStop"           bun:"max_charge_per_stop,type:NUMERIC(19,4),nullzero"`
	MaxChargePerDay            decimal.NullDecimal `json:"maxChargePerDay"            bun:"max_charge_per_day,type:NUMERIC(19,4),nullzero"`
	MaxChargePerShipment       decimal.NullDecimal `json:"maxChargePerShipment"       bun:"max_charge_per_shipment,type:NUMERIC(19,4),nullzero"`
	DayBoundaryMode            CapScope            `json:"dayBoundaryMode"            bun:"day_boundary_mode,type:detention_cap_scope_enum,notnull,default:'PerStop'"`
	ConvertToLayoverAtMinutes  *int32              `json:"convertToLayoverAtMinutes"  bun:"convert_to_layover_at_minutes,type:INTEGER,nullzero"`
	LayoverAccessorialChargeID *pulid.ID           `json:"layoverAccessorialChargeId" bun:"layover_accessorial_charge_id,type:VARCHAR(100),nullzero"`

	NotificationRequirement     NotificationRequirement `json:"notificationRequirement"     bun:"notification_requirement,type:detention_notification_requirement_enum,notnull,default:'None'"`
	NotificationLeadMinutes     int16                   `json:"notificationLeadMinutes"     bun:"notification_lead_minutes,type:SMALLINT,notnull,default:30"`
	NotificationDeadlineMinutes int16                   `json:"notificationDeadlineMinutes" bun:"notification_deadline_minutes,type:SMALLINT,notnull"`
	UnnotifiedBehavior          UnnotifiedBehavior      `json:"unnotifiedBehavior"          bun:"unnotified_behavior,type:detention_unnotified_behavior_enum,notnull,default:'Bill'"`
	AutoSendNotice              bool                    `json:"autoSendNotice"              bun:"auto_send_notice,type:BOOLEAN,notnull"`
	AttachNoticePDF             bool                    `json:"attachNoticePdf"             bun:"attach_notice_pdf,type:BOOLEAN,notnull"`
	SendDepartureSummary        bool                    `json:"sendDepartureSummary"        bun:"send_departure_summary,type:BOOLEAN,notnull"`
	RequireApprovalOverAmount   decimal.NullDecimal     `json:"requireApprovalOverAmount"   bun:"require_approval_over_amount,type:NUMERIC(19,4),nullzero"`
	AutoApproveUnderAmount      decimal.NullDecimal     `json:"autoApproveUnderAmount"      bun:"auto_approve_under_amount,type:NUMERIC(19,4),nullzero"`
	Currency                    string                  `json:"currency"                    bun:"currency,type:VARCHAR(3),notnull,default:'USD'"`
	Comments                    string                  `json:"comments"                    bun:"comments,type:TEXT,nullzero"`

	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`

	BusinessUnit      *tenant.BusinessUnit                 `json:"-"                           bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization      *tenant.Organization                 `json:"-"                           bun:"rel:belongs-to,join:organization_id=id"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" bun:"rel:belongs-to,join:accessorial_charge_id=id"`
	Tiers             []*DetentionPolicyTier               `json:"tiers,omitempty"             bun:"rel:has-many,join:id=detention_policy_id"`
}

func (p *DetentionPolicy) GetID() pulid.ID { return p.ID }

func (p *DetentionPolicy) GetCreatedAt() int64 { return p.CreatedAt }

func (p *DetentionPolicy) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *DetentionPolicy) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *DetentionPolicy) GetTableName() string { return "detention_policies" }

func (p *DetentionPolicy) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dtp",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

// ComputeSpecificity ranks how narrowly a policy is targeted. Resolution sorts
// on Priority first, then this score, so the most specific contract term wins
// without callers having to reason about insertion order.
func (p *DetentionPolicy) ComputeSpecificity() int32 {
	var score int32

	if p.LocationID != nil && !p.LocationID.IsNil() {
		score += SpecificityWeightLocation
	}
	if p.CustomerID != nil && !p.CustomerID.IsNil() {
		score += SpecificityWeightCustomer
	}
	if len(p.CommodityIDs) > 0 {
		score += SpecificityWeightCommodity
	}
	if len(p.ServiceTypeIDs) > 0 {
		score += SpecificityWeightServiceType
	}
	if len(p.ShipmentTypeIDs) > 0 {
		score += SpecificityWeightShipmentType
	}
	if len(p.StopTypes) > 0 {
		score += SpecificityWeightStopType
	}

	return score
}

// FreeMinutesForStopType resolves the free-time allowance, preferring the
// stop-type specific override when the contract distinguishes loading from
// unloading.
func (p *DetentionPolicy) FreeMinutesForStopType(stopType shipment.StopType) int32 {
	switch stopType {
	case shipment.StopTypePickup, shipment.StopTypeSplitPickup:
		if p.PickupFreeMinutes != nil {
			return *p.PickupFreeMinutes
		}
	case shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery:
		if p.DeliveryFreeMinutes != nil {
			return *p.DeliveryFreeMinutes
		}
	}

	return p.BillingFreeMinutes
}

// PayFreeMinutesOrDefault falls back to the billing allowance when the policy
// does not carve out a separate driver-pay threshold.
func (p *DetentionPolicy) PayFreeMinutesOrDefault() int32 {
	if p.PayFreeMinutes != nil {
		return *p.PayFreeMinutes
	}
	return p.BillingFreeMinutes
}

func (p *DetentionPolicy) IsEffectiveAt(timestamp int64) bool {
	if p.EffectiveStartDate != nil && timestamp < *p.EffectiveStartDate {
		return false
	}
	if p.EffectiveEndDate != nil && timestamp > *p.EffectiveEndDate {
		return false
	}
	return true
}

func (p *DetentionPolicy) RequiresNotice() bool {
	return p.NotificationRequirement == NotificationRequirementRequired
}

// MatchInput carries the shipment and stop facts a policy is matched against.
type MatchInput struct {
	CustomerID     pulid.ID
	LocationID     pulid.ID
	ShipmentTypeID pulid.ID
	ServiceTypeID  pulid.ID
	CommodityIDs   []pulid.ID
	StopType       shipment.StopType
	ScheduleType   shipment.StopScheduleType
	Timestamp      int64
}

// Matches reports whether every populated scope dimension on the policy is
// satisfied. An empty dimension is a wildcard.
func (p *DetentionPolicy) Matches(in MatchInput) bool {
	if p.Status != PolicyStatusActive {
		return false
	}

	if !p.IsEffectiveAt(in.Timestamp) {
		return false
	}

	if p.CustomerID != nil && !p.CustomerID.IsNil() && *p.CustomerID != in.CustomerID {
		return false
	}

	if p.LocationID != nil && !p.LocationID.IsNil() && *p.LocationID != in.LocationID {
		return false
	}

	if len(p.ShipmentTypeIDs) > 0 && !slices.Contains(p.ShipmentTypeIDs, in.ShipmentTypeID) {
		return false
	}

	if len(p.ServiceTypeIDs) > 0 && !slices.Contains(p.ServiceTypeIDs, in.ServiceTypeID) {
		return false
	}

	if len(p.CommodityIDs) > 0 && !containsAny(p.CommodityIDs, in.CommodityIDs) {
		return false
	}

	if len(p.StopTypes) > 0 && !slices.Contains(p.StopTypes, in.StopType) {
		return false
	}

	if p.AppointmentStopsOnly && in.ScheduleType != shipment.StopScheduleTypeAppointment {
		return false
	}

	return true
}

func containsAny(want, have []pulid.ID) bool {
	for _, candidate := range have {
		if slices.Contains(want, candidate) {
			return true
		}
	}
	return false
}

func (p *DetentionPolicy) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&p.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 50).Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			validation.In(PolicyStatusActive, PolicyStatusInactive, PolicyStatusDraft).
				Error("Status is invalid"),
		),
		validation.Field(&p.ClockStartBasis,
			validation.Required.Error("Clock start basis is required"),
			validation.In(
				ClockStartBasisArrival,
				ClockStartBasisAppointment,
				ClockStartBasisLaterOfArrivalOrAppointment,
				ClockStartBasisEarlierOfArrivalOrAppointment,
			).Error("Clock start basis is invalid"),
		),
		validation.Field(&p.LateArrivalRule,
			validation.Required.Error("Late arrival rule is required"),
			validation.In(
				LateArrivalRuleNoEffect,
				LateArrivalRuleForfeit,
				LateArrivalRuleClockFromAppointment,
				LateArrivalRuleReduceFreeTime,
			).Error("Late arrival rule is invalid"),
		),
		validation.Field(&p.RoundingMode,
			validation.Required.Error("Rounding mode is required"),
			validation.In(
				RoundingModeUp,
				RoundingModeDown,
				RoundingModeNearest,
				RoundingModeExact,
			).Error("Rounding mode is invalid"),
		),
		validation.Field(&p.RateSource,
			validation.Required.Error("Rate source is required"),
			validation.In(RateSourceAccessorial, RateSourceTiers).Error("Rate source is invalid"),
		),
		validation.Field(&p.NotificationRequirement,
			validation.Required.Error("Notification requirement is required"),
			validation.In(
				NotificationRequirementNone,
				NotificationRequirementAdvisory,
				NotificationRequirementRequired,
			).Error("Notification requirement is invalid"),
		),
		validation.Field(&p.UnnotifiedBehavior,
			validation.Required.Error("Unnotified behavior is required"),
			validation.In(
				UnnotifiedBehaviorBill,
				UnnotifiedBehaviorFlag,
				UnnotifiedBehaviorSuppress,
			).Error("Unnotified behavior is invalid"),
		),
		validation.Field(&p.DayBoundaryMode,
			validation.Required.Error("Day boundary mode is required"),
			validation.In(CapScopePerStop, CapScopePerDay, CapScopePerShipment).
				Error("Day boundary mode is invalid"),
		),
		validation.Field(&p.AccessorialChargeID,
			validation.Required.Error("Accessorial charge is required"),
		),
		validation.Field(&p.Currency,
			validation.Required.Error("Currency is required"),
			validation.Length(3, 3).Error("Currency must be a 3-letter ISO code"),
		),
	))

	p.validateFreeTime(multiErr)
	p.validateIncrement(multiErr)
	p.validateEffectiveDates(multiErr)
	p.validateCaps(multiErr)
	p.validateNotification(multiErr)
	p.validateRateSource(multiErr)
	p.validateScope(multiErr)
	validateTiers(p.Tiers, multiErr)
}

func (p *DetentionPolicy) validateFreeTime(multiErr *errortypes.MultiError) {
	if p.BillingFreeMinutes < 0 || p.BillingFreeMinutes > MaxFreeMinutes {
		multiErr.Add("billingFreeMinutes", errortypes.ErrInvalid,
			"Billing free time must be between 0 minutes and 7 days")
	}

	checkOverride := func(field string, value *int32) {
		if value == nil {
			return
		}
		if *value < 0 || *value > MaxFreeMinutes {
			multiErr.Add(field, errortypes.ErrInvalid,
				"Free time must be between 0 minutes and 7 days")
		}
	}

	checkOverride("pickupFreeMinutes", p.PickupFreeMinutes)
	checkOverride("deliveryFreeMinutes", p.DeliveryFreeMinutes)
	checkOverride("payFreeMinutes", p.PayFreeMinutes)

	if p.MinimumBillableMinutes < 0 {
		multiErr.Add("minimumBillableMinutes", errortypes.ErrInvalid,
			"Minimum billable time cannot be negative")
	}

	if p.LateArrivalGraceMinutes < 0 {
		multiErr.Add("lateArrivalGraceMinutes", errortypes.ErrInvalid,
			"Late arrival grace cannot be negative")
	}
}

func (p *DetentionPolicy) validateIncrement(multiErr *errortypes.MultiError) {
	if p.RoundingMode == RoundingModeExact {
		return
	}

	if p.BillingIncrementMinutes <= 0 {
		multiErr.Add("billingIncrementMinutes", errortypes.ErrInvalid,
			"Billing increment must be greater than zero unless rounding is Exact")
		return
	}

	if p.BillingIncrementMinutes > 1440 {
		multiErr.Add("billingIncrementMinutes", errortypes.ErrInvalid,
			"Billing increment cannot exceed 24 hours")
	}
}

func (p *DetentionPolicy) validateEffectiveDates(multiErr *errortypes.MultiError) {
	if p.EffectiveStartDate == nil || p.EffectiveEndDate == nil {
		return
	}

	if *p.EffectiveEndDate <= *p.EffectiveStartDate {
		multiErr.Add("effectiveEndDate", errortypes.ErrInvalid,
			"Expiration must be after the effective date")
	}
}

func (p *DetentionPolicy) validateCaps(multiErr *errortypes.MultiError) {
	if p.MaxBillableMinutesPerStop != nil && *p.MaxBillableMinutesPerStop <= 0 {
		multiErr.Add("maxBillableMinutesPerStop", errortypes.ErrInvalid,
			"Maximum billable minutes must be greater than zero")
	}

	checkAmount := func(field string, value decimal.NullDecimal) {
		if value.Valid && value.Decimal.LessThanOrEqual(decimal.Zero) {
			multiErr.Add(field, errortypes.ErrInvalid, "Cap must be greater than zero")
		}
	}

	checkAmount("maxChargePerStop", p.MaxChargePerStop)
	checkAmount("maxChargePerDay", p.MaxChargePerDay)
	checkAmount("maxChargePerShipment", p.MaxChargePerShipment)

	if p.MaxChargePerStop.Valid && p.MaxChargePerShipment.Valid &&
		p.MaxChargePerShipment.Decimal.LessThan(p.MaxChargePerStop.Decimal) {
		multiErr.Add("maxChargePerShipment", errortypes.ErrInvalid,
			"Shipment cap cannot be lower than the per-stop cap")
	}

	if p.ConvertToLayoverAtMinutes != nil {
		if *p.ConvertToLayoverAtMinutes <= 0 {
			multiErr.Add("convertToLayoverAtMinutes", errortypes.ErrInvalid,
				"Layover conversion threshold must be greater than zero")
		}
		if p.LayoverAccessorialChargeID == nil || p.LayoverAccessorialChargeID.IsNil() {
			multiErr.Add("layoverAccessorialChargeId", errortypes.ErrRequired,
				"Select the layover charge to apply once detention converts")
		}
	}

	if p.RequireApprovalOverAmount.Valid && p.AutoApproveUnderAmount.Valid &&
		p.AutoApproveUnderAmount.Decimal.GreaterThan(p.RequireApprovalOverAmount.Decimal) {
		multiErr.Add("autoApproveUnderAmount", errortypes.ErrInvalid,
			"Auto-approve threshold cannot exceed the approval-required threshold")
	}
}

func (p *DetentionPolicy) validateNotification(multiErr *errortypes.MultiError) {
	if p.NotificationRequirement == NotificationRequirementNone {
		return
	}

	if p.NotificationLeadMinutes < 0 {
		multiErr.Add("notificationLeadMinutes", errortypes.ErrInvalid,
			"Notification lead time cannot be negative")
	}

	if p.NotificationDeadlineMinutes < 0 {
		multiErr.Add("notificationDeadlineMinutes", errortypes.ErrInvalid,
			"Notification deadline cannot be negative")
	}

	if int32(p.NotificationLeadMinutes) > p.BillingFreeMinutes {
		multiErr.Add("notificationLeadMinutes", errortypes.ErrInvalid,
			"Notification lead time cannot exceed the free time it warns about")
	}

	if p.NotificationRequirement == NotificationRequirementRequired &&
		p.UnnotifiedBehavior == UnnotifiedBehaviorBill {
		multiErr.Add("unnotifiedBehavior", errortypes.ErrInvalid,
			"A required notice must Flag or Suppress billing when it is missed")
	}
}

func (p *DetentionPolicy) validateRateSource(multiErr *errortypes.MultiError) {
	if p.RateSource != RateSourceTiers {
		return
	}

	if len(p.Tiers) == 0 {
		multiErr.Add("tiers", errortypes.ErrRequired,
			"Add at least one rate tier when the rate source is Tiers")
	}
}

func (p *DetentionPolicy) validateScope(multiErr *errortypes.MultiError) {
	if !p.IsOrgDefault {
		return
	}

	if p.ComputeSpecificity() != 0 {
		multiErr.Add("isOrgDefault", errortypes.ErrInvalid,
			"The organization default policy cannot target a customer, facility, or type")
	}
}

func (p *DetentionPolicy) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if p.Currency == "" {
		p.Currency = "USD"
	}
	if p.DayBoundaryMode == "" {
		p.DayBoundaryMode = CapScopePerStop
	}
	if p.RoundingMode == RoundingModeExact {
		p.BillingIncrementMinutes = 0
	}
	if p.RateSource != RateSourceTiers {
		p.Tiers = nil
	}
	if p.ConvertToLayoverAtMinutes == nil {
		p.LayoverAccessorialChargeID = nil
	}
	if p.IsOrgDefault {
		p.CustomerID = nil
		p.LocationID = nil
		p.ShipmentTypeIDs = nil
		p.ServiceTypeIDs = nil
		p.CommodityIDs = nil
		p.StopTypes = nil
	}

	p.SpecificityScore = p.ComputeSpecificity()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("dtp_")
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
