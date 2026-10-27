package detention

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/shared/money"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
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
	_ validationframework.TenantedEntity = (*DetentionPolicy)(nil)
	_ domaintypes.PostgresSearchable     = (*DetentionPolicy)(nil)
)

type DetentionPolicy struct {
	keel.Table     `keel:"detention_policies"`
	ID             pulid.ID     `keel:"id,pk,autoid:dtp_"          json:"id"`
	BusinessUnitID pulid.ID     `keel:"business_unit_id,pk,tenant" json:"businessUnitId"`
	OrganizationID pulid.ID     `keel:"organization_id,pk,tenant"  json:"organizationId"`
	Name           string       `keel:"name"                       json:"name"`
	Code           string       `keel:"code"                       json:"code"`
	Description    string       `keel:"description,nullzero"       json:"description"`
	Status         PolicyStatus `keel:"status"                     json:"status"`

	IsOrgDefault     bool       `json:"isOrgDefault"     keel:"is_org_default"`
	Priority         int16      `json:"priority"         keel:"priority"`
	SpecificityScore int32      `json:"specificityScore" keel:"specificity_score"`
	CustomerID       *pulid.ID  `json:"customerId"       keel:"customer_id"`
	LocationID       *pulid.ID  `json:"locationId"       keel:"location_id"`
	ShipmentTypeIDs  []pulid.ID `json:"shipmentTypeIds"  keel:"shipment_type_ids,jsonb,nullzero"`
	ServiceTypeIDs   []pulid.ID `json:"serviceTypeIds"   keel:"service_type_ids,jsonb,nullzero"`
	CommodityIDs     []pulid.ID `json:"commodityIds"     keel:"commodity_ids,jsonb,nullzero"`

	StopTypes                  []shipment.StopType `json:"stopTypes"                  keel:"stop_types,jsonb,nullzero"`
	AppointmentStopsOnly       bool                `json:"appointmentStopsOnly"       keel:"appointment_stops_only"`
	EffectiveStartDate         *int64              `json:"effectiveStartDate"         keel:"effective_start_date"`
	EffectiveEndDate           *int64              `json:"effectiveEndDate"           keel:"effective_end_date"`
	ClockStartBasis            ClockStartBasis     `json:"clockStartBasis"            keel:"clock_start_basis"`
	LateArrivalRule            LateArrivalRule     `json:"lateArrivalRule"            keel:"late_arrival_rule"`
	LateArrivalGraceMinutes    int16               `json:"lateArrivalGraceMinutes"    keel:"late_arrival_grace_minutes"`
	BillingFreeMinutes         int32               `json:"billingFreeMinutes"         keel:"billing_free_minutes"`
	PickupFreeMinutes          *int32              `json:"pickupFreeMinutes"          keel:"pickup_free_minutes"`
	DeliveryFreeMinutes        *int32              `json:"deliveryFreeMinutes"        keel:"delivery_free_minutes"`
	PayFreeMinutes             *int32              `json:"payFreeMinutes"             keel:"pay_free_minutes"`
	MinimumBillableMinutes     int32               `json:"minimumBillableMinutes"     keel:"minimum_billable_minutes"`
	BillingIncrementMinutes    int16               `json:"billingIncrementMinutes"    keel:"billing_increment_minutes"`
	RoundingMode               RoundingMode        `json:"roundingMode"               keel:"rounding_mode"`
	RateSource                 RateSource          `json:"rateSource"                 keel:"rate_source"`
	AccessorialChargeID        pulid.ID            `json:"accessorialChargeId"        keel:"accessorial_charge_id"`
	MaxBillableMinutesPerStop  *int32              `json:"maxBillableMinutesPerStop"  keel:"max_billable_minutes_per_stop"`
	MaxChargePerStop           decimal.NullDecimal `json:"maxChargePerStop"           keel:"max_charge_per_stop"`
	MaxChargePerDay            decimal.NullDecimal `json:"maxChargePerDay"            keel:"max_charge_per_day"`
	MaxChargePerShipment       decimal.NullDecimal `json:"maxChargePerShipment"       keel:"max_charge_per_shipment"`
	DayBoundaryMode            CapScope            `json:"dayBoundaryMode"            keel:"day_boundary_mode"`
	ConvertToLayoverAtMinutes  *int32              `json:"convertToLayoverAtMinutes"  keel:"convert_to_layover_at_minutes"`
	LayoverAccessorialChargeID *pulid.ID           `json:"layoverAccessorialChargeId" keel:"layover_accessorial_charge_id"`

	NotificationRequirement     NotificationRequirement `json:"notificationRequirement"     keel:"notification_requirement"`
	NotificationLeadMinutes     int16                   `json:"notificationLeadMinutes"     keel:"notification_lead_minutes"`
	NotificationDeadlineMinutes int16                   `json:"notificationDeadlineMinutes" keel:"notification_deadline_minutes"`
	UnnotifiedBehavior          UnnotifiedBehavior      `json:"unnotifiedBehavior"          keel:"unnotified_behavior"`
	AutoSendNotice              bool                    `json:"autoSendNotice"              keel:"auto_send_notice"`
	AttachNoticePDF             bool                    `json:"attachNoticePdf"             keel:"attach_notice_pdf"`
	SendDepartureSummary        bool                    `json:"sendDepartureSummary"        keel:"send_departure_summary"`
	RequireApprovalOverAmount   decimal.NullDecimal     `json:"requireApprovalOverAmount"   keel:"require_approval_over_amount"`
	AutoApproveUnderAmount      decimal.NullDecimal     `json:"autoApproveUnderAmount"      keel:"auto_approve_under_amount"`
	Currency                    string                  `json:"currency"                    keel:"currency"`
	Comments                    string                  `json:"comments"                    keel:"comments,nullzero"`

	Version      int64  `json:"version"   keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector string `json:"-"         keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-"         keel:"rank,computed"`

	BusinessUnit      *tenant.BusinessUnit                 `json:"-"`
	Organization      *tenant.Organization                 `json:"-"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" keel:"belongs:accessorial_charge_id"`
	Tiers             []*DetentionPolicyTier               `json:"tiers,omitempty"             keel:"many:detention_policy_id"`
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

func (p *DetentionPolicy) BeforeCreate(ctx context.Context) error {
	if p.Status == "" {
		p.Status = PolicyStatusDraft
	}
	if p.ClockStartBasis == "" {
		p.ClockStartBasis = ClockStartBasisLaterOfArrivalOrAppointment
	}
	if p.LateArrivalRule == "" {
		p.LateArrivalRule = LateArrivalRuleNoEffect
	}
	if p.BillingFreeMinutes == 0 {
		p.BillingFreeMinutes = 120
	}
	if p.BillingIncrementMinutes == 0 {
		p.BillingIncrementMinutes = 15
	}
	if p.RoundingMode == "" {
		p.RoundingMode = RoundingModeUp
	}
	if p.RateSource == "" {
		p.RateSource = RateSourceAccessorial
	}
	if p.DayBoundaryMode == "" {
		p.DayBoundaryMode = CapScopePerStop
	}
	if p.NotificationRequirement == "" {
		p.NotificationRequirement = NotificationRequirementNone
	}
	if p.NotificationLeadMinutes == 0 {
		p.NotificationLeadMinutes = 30
	}
	if p.UnnotifiedBehavior == "" {
		p.UnnotifiedBehavior = UnnotifiedBehaviorBill
	}
	if p.Currency == "" {
		p.Currency = money.DefaultCurrencyCode
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

	return nil
}

func (p *DetentionPolicy) BeforeUpdate(ctx context.Context) error {
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

	return nil
}

func (DetentionPolicy) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_detention_policies_code",
			Message: "Detention policy with this code already exists in your organization",
		},
		{
			Name:    "uq_detention_policies_org_default",
			Message: "Your organization already has a default detention policy",
		},
		{
			Name:    "fk_detention_policies_accessorial",
			Message: "Accessorial charge does not exist in your organization",
		},
		{
			Name:    "fk_detention_policies_layover_accessorial",
			Message: "Layover accessorial charge does not exist in your organization",
		},
	}
}
