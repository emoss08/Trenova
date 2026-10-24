package detention

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
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
	keel.Table                `keel:"detention_policies"`
	pagination.CursorValueSet `bun:",embed"                             json:"-"`

	ID             pulid.ID     `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:dtp_"`
	BusinessUnitID pulid.ID     `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID     `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	Name           string       `json:"name"           bun:"name,type:VARCHAR(100),notnull" keel:"name"`
	Code           string       `json:"code"           bun:"code,type:VARCHAR(50),notnull" keel:"code"`
	Description    string       `json:"description"    bun:"description,type:TEXT,nullzero" keel:"description,nullzero"`
	Status         PolicyStatus `json:"status"         bun:"status,type:detention_policy_status_enum,notnull,default:'Draft'" keel:"status"`

	IsOrgDefault     bool       `json:"isOrgDefault"     bun:"is_org_default,type:BOOLEAN,notnull" keel:"is_org_default"`
	Priority         int16      `json:"priority"         bun:"priority,type:SMALLINT,notnull" keel:"priority"`
	SpecificityScore int32      `json:"specificityScore" bun:"specificity_score,type:INTEGER,notnull" keel:"specificity_score"`
	CustomerID       *pulid.ID  `json:"customerId"       bun:"customer_id,type:VARCHAR(100),nullzero" keel:"customer_id"`
	LocationID       *pulid.ID  `json:"locationId"       bun:"location_id,type:VARCHAR(100),nullzero" keel:"location_id"`
	ShipmentTypeIDs  []pulid.ID `json:"shipmentTypeIds"  bun:"shipment_type_ids,type:JSONB,nullzero" keel:"shipment_type_ids,jsonb,nullzero"`
	ServiceTypeIDs   []pulid.ID `json:"serviceTypeIds"   bun:"service_type_ids,type:JSONB,nullzero" keel:"service_type_ids,jsonb,nullzero"`
	CommodityIDs     []pulid.ID `json:"commodityIds"     bun:"commodity_ids,type:JSONB,nullzero" keel:"commodity_ids,jsonb,nullzero"`

	StopTypes                  []shipment.StopType `json:"stopTypes"                  bun:"stop_types,type:JSONB,nullzero" keel:"stop_types,jsonb,nullzero"`
	AppointmentStopsOnly       bool                `json:"appointmentStopsOnly"       bun:"appointment_stops_only,type:BOOLEAN,notnull" keel:"appointment_stops_only"`
	EffectiveStartDate         *int64              `json:"effectiveStartDate"         bun:"effective_start_date,type:BIGINT,nullzero" keel:"effective_start_date"`
	EffectiveEndDate           *int64              `json:"effectiveEndDate"           bun:"effective_end_date,type:BIGINT,nullzero" keel:"effective_end_date"`
	ClockStartBasis            ClockStartBasis     `json:"clockStartBasis"            bun:"clock_start_basis,type:detention_clock_start_basis_enum,notnull,default:'LaterOfArrivalOrAppointment'" keel:"clock_start_basis"`
	LateArrivalRule            LateArrivalRule     `json:"lateArrivalRule"            bun:"late_arrival_rule,type:detention_late_arrival_rule_enum,notnull,default:'NoEffect'" keel:"late_arrival_rule"`
	LateArrivalGraceMinutes    int16               `json:"lateArrivalGraceMinutes"    bun:"late_arrival_grace_minutes,type:SMALLINT,notnull" keel:"late_arrival_grace_minutes"`
	BillingFreeMinutes         int32               `json:"billingFreeMinutes"         bun:"billing_free_minutes,type:INTEGER,notnull,default:120" keel:"billing_free_minutes"`
	PickupFreeMinutes          *int32              `json:"pickupFreeMinutes"          bun:"pickup_free_minutes,type:INTEGER,nullzero" keel:"pickup_free_minutes"`
	DeliveryFreeMinutes        *int32              `json:"deliveryFreeMinutes"        bun:"delivery_free_minutes,type:INTEGER,nullzero" keel:"delivery_free_minutes"`
	PayFreeMinutes             *int32              `json:"payFreeMinutes"             bun:"pay_free_minutes,type:INTEGER,nullzero" keel:"pay_free_minutes"`
	MinimumBillableMinutes     int32               `json:"minimumBillableMinutes"     bun:"minimum_billable_minutes,type:INTEGER,notnull" keel:"minimum_billable_minutes"`
	BillingIncrementMinutes    int16               `json:"billingIncrementMinutes"    bun:"billing_increment_minutes,type:SMALLINT,notnull,default:15" keel:"billing_increment_minutes"`
	RoundingMode               RoundingMode        `json:"roundingMode"               bun:"rounding_mode,type:detention_rounding_mode_enum,notnull,default:'Up'" keel:"rounding_mode"`
	RateSource                 RateSource          `json:"rateSource"                 bun:"rate_source,type:detention_rate_source_enum,notnull,default:'Accessorial'" keel:"rate_source"`
	AccessorialChargeID        pulid.ID            `json:"accessorialChargeId"        bun:"accessorial_charge_id,type:VARCHAR(100),notnull" keel:"accessorial_charge_id"`
	MaxBillableMinutesPerStop  *int32              `json:"maxBillableMinutesPerStop"  bun:"max_billable_minutes_per_stop,type:INTEGER,nullzero" keel:"max_billable_minutes_per_stop"`
	MaxChargePerStop           decimal.NullDecimal `json:"maxChargePerStop"           bun:"max_charge_per_stop,type:NUMERIC(19,4),nullzero" keel:"max_charge_per_stop"`
	MaxChargePerDay            decimal.NullDecimal `json:"maxChargePerDay"            bun:"max_charge_per_day,type:NUMERIC(19,4),nullzero" keel:"max_charge_per_day"`
	MaxChargePerShipment       decimal.NullDecimal `json:"maxChargePerShipment"       bun:"max_charge_per_shipment,type:NUMERIC(19,4),nullzero" keel:"max_charge_per_shipment"`
	DayBoundaryMode            CapScope            `json:"dayBoundaryMode"            bun:"day_boundary_mode,type:detention_cap_scope_enum,notnull,default:'PerStop'" keel:"day_boundary_mode"`
	ConvertToLayoverAtMinutes  *int32              `json:"convertToLayoverAtMinutes"  bun:"convert_to_layover_at_minutes,type:INTEGER,nullzero" keel:"convert_to_layover_at_minutes"`
	LayoverAccessorialChargeID *pulid.ID           `json:"layoverAccessorialChargeId" bun:"layover_accessorial_charge_id,type:VARCHAR(100),nullzero" keel:"layover_accessorial_charge_id"`

	NotificationRequirement     NotificationRequirement `json:"notificationRequirement"     bun:"notification_requirement,type:detention_notification_requirement_enum,notnull,default:'None'" keel:"notification_requirement"`
	NotificationLeadMinutes     int16                   `json:"notificationLeadMinutes"     bun:"notification_lead_minutes,type:SMALLINT,notnull,default:30" keel:"notification_lead_minutes"`
	NotificationDeadlineMinutes int16                   `json:"notificationDeadlineMinutes" bun:"notification_deadline_minutes,type:SMALLINT,notnull" keel:"notification_deadline_minutes"`
	UnnotifiedBehavior          UnnotifiedBehavior      `json:"unnotifiedBehavior"          bun:"unnotified_behavior,type:detention_unnotified_behavior_enum,notnull,default:'Bill'" keel:"unnotified_behavior"`
	AutoSendNotice              bool                    `json:"autoSendNotice"              bun:"auto_send_notice,type:BOOLEAN,notnull" keel:"auto_send_notice"`
	AttachNoticePDF             bool                    `json:"attachNoticePdf"             bun:"attach_notice_pdf,type:BOOLEAN,notnull" keel:"attach_notice_pdf"`
	SendDepartureSummary        bool                    `json:"sendDepartureSummary"        bun:"send_departure_summary,type:BOOLEAN,notnull" keel:"send_departure_summary"`
	RequireApprovalOverAmount   decimal.NullDecimal     `json:"requireApprovalOverAmount"   bun:"require_approval_over_amount,type:NUMERIC(19,4),nullzero" keel:"require_approval_over_amount"`
	AutoApproveUnderAmount      decimal.NullDecimal     `json:"autoApproveUnderAmount"      bun:"auto_approve_under_amount,type:NUMERIC(19,4),nullzero" keel:"auto_approve_under_amount"`
	Currency                    string                  `json:"currency"                    bun:"currency,type:VARCHAR(3),notnull,default:'USD'" keel:"currency"`
	Comments                    string                  `json:"comments"                    bun:"comments,type:TEXT,nullzero" keel:"comments,nullzero"`

	Version      int64  `json:"version"   bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly" keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly" keel:"rank,computed"`

	BusinessUnit      *tenant.BusinessUnit                 `json:"-"                           bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization      *tenant.Organization                 `json:"-"                           bun:"rel:belongs-to,join:organization_id=id"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" bun:"rel:belongs-to,join:accessorial_charge_id=id" keel:"belongs:accessorial_charge_id"`
	Tiers             []*DetentionPolicyTier               `json:"tiers,omitempty"             bun:"rel:has-many,join:id=detention_policy_id" keel:"many:detention_policy_id"`
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
