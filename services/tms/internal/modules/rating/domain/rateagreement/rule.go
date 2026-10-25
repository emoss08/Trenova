package rateagreement

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/postgis"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const (
	maxLabelLength = 150
	// DaysOfWeekAll is the bitmask that matches every day, which is also what
	// an unset field means.
	DaysOfWeekAll = int16(0)
)

// The lane, applicability and price of one line of a contract
type RateAgreementRule struct {
	keel.Table `keel:"rate_agreement_rules,report=Billing,label=Lane Rule"`

	ID              pulid.ID `json:"id"              keel:"id,pk,autoid:ragr_"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID `json:"organizationId"  keel:"organization_id,pk,tenant"`
	RateAgreementID pulid.ID `json:"rateAgreementId" keel:"rate_agreement_id"`

	PartyType PartyType  `json:"partyType" keel:"party_type"`
	PartyID   pulid.ID   `json:"partyId"   keel:"party_id"`
	Label     string     `json:"label"     keel:"label,nullzero"`
	Status    RuleStatus `json:"status"    keel:"status"`

	OriginScopeType  rategeo.ScopeType `json:"originScopeType"  keel:"origin_scope_type,label=Origin Scope"`
	OriginScopeValue string            `json:"originScopeValue" keel:"origin_scope_value,nullzero"`
	OriginCity       string            `json:"originCity"       keel:"origin_city,nullzero"`

	DestinationScopeType  rategeo.ScopeType `json:"destinationScopeType"  keel:"destination_scope_type,label=Destination Scope"`
	DestinationScopeValue string            `json:"destinationScopeValue" keel:"destination_scope_value,nullzero"`
	DestinationCity       string            `json:"destinationCity"       keel:"destination_city,nullzero"`

	LaneKey   string    `json:"laneKey"   keel:"lane_key"`
	Direction Direction `json:"direction" keel:"direction"`

	OriginRadiusMeters      *float64       `json:"originRadiusMeters"      keel:"origin_radius_meters"`
	DestinationRadiusMeters *float64       `json:"destinationRadiusMeters" keel:"destination_radius_meters"`
	OriginLatitude          *float64       `json:"originLatitude"          keel:"origin_latitude"`
	OriginLongitude         *float64       `json:"originLongitude"         keel:"origin_longitude"`
	DestinationLatitude     *float64       `json:"destinationLatitude"     keel:"destination_latitude"`
	DestinationLongitude    *float64       `json:"destinationLongitude"    keel:"destination_longitude"`
	OriginCenter            *postgis.Point `json:"-"                       keel:"origin_center,readonly"`
	DestinationCenter       *postgis.Point `json:"-"                       keel:"destination_center,readonly"`

	ServiceTypeIDs   []pulid.ID                   `json:"serviceTypeIds"   keel:"service_type_ids,jsonb,nullzero"`
	ShipmentTypeIDs  []pulid.ID                   `json:"shipmentTypeIds"  keel:"shipment_type_ids,jsonb,nullzero"`
	TractorTypeIDs   []pulid.ID                   `json:"tractorTypeIds"   keel:"tractor_type_ids,jsonb,nullzero"`
	TrailerTypeIDs   []pulid.ID                   `json:"trailerTypeIds"   keel:"trailer_type_ids,jsonb,nullzero"`
	CommodityIDs     []pulid.ID                   `json:"commodityIds"     keel:"commodity_ids,jsonb,nullzero"`
	FreightClasses   []commodity.FreightClass     `json:"freightClasses"   keel:"freight_classes,jsonb,nullzero"`
	ServiceModels    []modeprofile.ServiceModel   `json:"serviceModels"    keel:"service_models,jsonb,nullzero"`
	EquipmentClasses []modeprofile.EquipmentClass `json:"equipmentClasses" keel:"equipment_classes,jsonb,nullzero"`

	MinWeight   decimal.NullDecimal `json:"minWeight"   keel:"min_weight"`
	MaxWeight   decimal.NullDecimal `json:"maxWeight"   keel:"max_weight"`
	MinDistance decimal.NullDecimal `json:"minDistance" keel:"min_distance"`
	MaxDistance decimal.NullDecimal `json:"maxDistance" keel:"max_distance"`
	MinStops    *int16              `json:"minStops"    keel:"min_stops,dimension"`
	MaxStops    *int16              `json:"maxStops"    keel:"max_stops,dimension"`

	DaysOfWeek      int16 `json:"daysOfWeek"      keel:"days_of_week,dimension"`
	HazmatOnly      bool  `json:"hazmatOnly"      keel:"hazmat_only"`
	TempControlOnly bool  `json:"tempControlOnly" keel:"temp_control_only"`

	FormulaTemplateID pulid.ID           `json:"formulaTemplateId" keel:"formula_template_id"`
	Inputs            tarifftypes.Inputs `json:"inputs"            keel:"inputs,jsonb"`
	Currency          string             `json:"currency"          keel:"currency,nullzero"`

	Priority         int16 `json:"priority"         keel:"priority,dimension"`
	SpecificityScore int32 `json:"specificityScore" keel:"specificity_score,dimension"`

	EffectiveFrom     int64     `json:"effectiveFrom"     keel:"effective_from,type=epoch"`
	EffectiveTo       *int64    `json:"effectiveTo"       keel:"effective_to,type=epoch"`
	SupersedesRuleID  *pulid.ID `json:"supersedesRuleId"  keel:"supersedes_rule_id"`
	SourceImportRowID *pulid.ID `json:"sourceImportRowId" keel:"source_import_row_id"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Agreement *RateAgreement `json:"-" keel:"belongs:rate_agreement_id"`

	lane laneState
}

func (rar *RateAgreementRule) BeforeUpdate(_ context.Context) error {
	rar.applyDefaults()
	if rar.LaneKey == "" {
		return ErrLaneNotApplied
	}

	return nil
}

// OriginScope and DestinationScope read the two ends back as the geography
// value objects the key and weight helpers operate on.
func (rar *RateAgreementRule) OriginScope() rategeo.Scope {
	return rategeo.Scope{
		Type:  rar.OriginScopeType,
		Value: rar.OriginScopeValue,
		City:  rar.OriginCity,
	}
}

func (rar *RateAgreementRule) DestinationScope() rategeo.Scope {
	return rategeo.Scope{
		Type:  rar.DestinationScopeType,
		Value: rar.DestinationScopeValue,
		City:  rar.DestinationCity,
	}
}

// IsRadiusLane reports whether either end is described by a circle. Those rules
// are found through the geospatial index rather than by lane key, so the
// resolver has to fetch them with a second query.
func (rar *RateAgreementRule) IsRadiusLane() bool {
	return rar.OriginScopeType == rategeo.ScopeTypeRadius ||
		rar.DestinationScopeType == rategeo.ScopeTypeRadius
}

// IsEffectiveAt reports whether the rule's own window covers a moment. The
// window is half open so an amendment can begin on the instant its predecessor
// ends, with neither a gap nor an overlap.
func (rar *RateAgreementRule) IsEffectiveAt(timestamp int64) bool {
	if timestamp < rar.EffectiveFrom {
		return false
	}

	if rar.EffectiveTo != nil && timestamp >= *rar.EffectiveTo {
		return false
	}

	return true
}

// MatchesDayOfWeek reports whether the rule prices shipments on this weekday,
// where Sunday is zero. A zero mask means every day.
func (rar *RateAgreementRule) MatchesDayOfWeek(weekday int) bool {
	if rar.DaysOfWeek == DaysOfWeekAll {
		return true
	}

	return rar.DaysOfWeek&(1<<uint(weekday)) != 0
}

// EffectiveCurrency falls back to the agreement's currency, which is the usual
// case: a rule only names one when a single lane is priced in another currency.
func (rar *RateAgreementRule) EffectiveCurrency(agreement *RateAgreement) string {
	if rar.Currency != "" {
		return rar.Currency
	}

	if agreement != nil {
		return agreement.Currency
	}

	return ""
}

func (rar *RateAgreementRule) applyDefaults() {
	if rar.Status == "" {
		rar.Status = RuleStatusActive
	}
	if rar.Direction == "" {
		rar.Direction = DirectionDirectional
	}
	rar.Inputs = rar.Inputs.OrEmpty()
	if rar.OriginScopeType == "" {
		rar.OriginScopeType = rategeo.ScopeTypeAny
	}
	if rar.DestinationScopeType == "" {
		rar.DestinationScopeType = rategeo.ScopeTypeAny
	}
}

// ValidateWithin checks the rule against the agreement it belongs to, which is
// the only place the two can be compared — a rule's currency defaults from the
// header, and its effective window must sit inside the agreement's.
func (rar *RateAgreementRule) ValidateWithin(
	agreement *RateAgreement,
	multiErr *errortypes.MultiError,
) {
	rar.Validate(multiErr)

	if agreement == nil {
		return
	}

	// A rule that starts before its agreement does, or runs past its end, would
	// be reachable on dates the contract does not cover.
	if rar.EffectiveFrom < agreement.EffectiveFrom {
		multiErr.Add(
			"effectiveFrom",
			errortypes.ErrInvalid,
			"A rule cannot start before its agreement",
		)
	}

	if agreement.EffectiveTo != nil &&
		(rar.EffectiveTo == nil || *rar.EffectiveTo > *agreement.EffectiveTo) {
		multiErr.Add(
			"effectiveTo",
			errortypes.ErrInvalid,
			"A rule cannot outlast its agreement",
		)
	}
}

func (rar *RateAgreementRule) Validate(multiErr *errortypes.MultiError) {
	rar.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(rar,
		validation.Field(&rar.Label,
			validation.Length(0, maxLabelLength).
				Error("Label cannot be longer than 150 characters"),
		),
		validation.Field(&rar.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[RuleStatus]("Status is invalid"),
		),
		validation.Field(&rar.Direction,
			validation.Required.Error("Direction is required"),
			domainvalidation.ValidEnum[Direction]("Direction is invalid"),
		),
		validation.Field(&rar.OriginScopeType,
			validation.Required.Error("Origin scope type is required"),
			domainvalidation.ValidEnum[rategeo.ScopeType]("Origin scope type is invalid"),
		),
		validation.Field(&rar.DestinationScopeType,
			validation.Required.Error("Destination scope type is required"),
			domainvalidation.ValidEnum[rategeo.ScopeType]("Destination scope type is invalid"),
		),
		validation.Field(&rar.FormulaTemplateID,
			validation.Required.Error("A rule prices through a formula template"),
		),
		validation.Field(&rar.FreightClasses,
			validation.Each(
				domainvalidation.ValidEnum[commodity.FreightClass]("Freight class is invalid"),
			),
		),
		validation.Field(&rar.ServiceModels,
			validation.Each(
				domainvalidation.ValidEnum[modeprofile.ServiceModel]("Service model is invalid"),
			),
		),
		validation.Field(&rar.EquipmentClasses,
			validation.Each(
				domainvalidation.ValidEnum[modeprofile.EquipmentClass](
					"Equipment class is invalid",
				),
			),
		),
		validation.Field(&rar.Currency,
			validation.Length(0, 3).Error("Currency must be a three letter code"),
		),
		validation.Field(&rar.EffectiveFrom,
			validation.Required.Error("Effective from is required"),
		),
		validation.Field(&rar.DaysOfWeek,
			validation.Min(int16(0)).Error("Days of week is invalid"),
			validation.Max(int16(127)).Error("Days of week is invalid"),
		),
	))

	rar.validateScopes(multiErr)
	rar.validateRanges(multiErr)
}

func (rar *RateAgreementRule) validateScopes(multiErr *errortypes.MultiError) {
	validateScopeSide(multiErr, &scopeSide{
		field:        "originScopeValue",
		radiusField:  "originRadiusMeters",
		scope:        rar.OriginScope(),
		keyed:        !rar.lane.applied || rar.lane.origin,
		radiusMeters: rar.OriginRadiusMeters,
		latitude:     rar.OriginLatitude,
		longitude:    rar.OriginLongitude,
		label:        "Origin",
	})

	validateScopeSide(multiErr, &scopeSide{
		field:        "destinationScopeValue",
		radiusField:  "destinationRadiusMeters",
		scope:        rar.DestinationScope(),
		keyed:        !rar.lane.applied || rar.lane.destination,
		radiusMeters: rar.DestinationRadiusMeters,
		latitude:     rar.DestinationLatitude,
		longitude:    rar.DestinationLongitude,
		label:        "Destination",
	})
}

type scopeSide struct {
	field        string
	radiusField  string
	scope        rategeo.Scope
	keyed        bool
	radiusMeters *float64
	latitude     *float64
	longitude    *float64
	label        string
}

func validateScopeSide(multiErr *errortypes.MultiError, side *scopeSide) {
	if side.scope.Type == rategeo.ScopeTypeRadius {
		if side.radiusMeters == nil || *side.radiusMeters <= 0 {
			multiErr.Add(
				side.radiusField,
				errortypes.ErrRequired,
				"{0} radius must be greater than zero", side.label,
			)
		}
		if side.latitude == nil || side.longitude == nil {
			multiErr.Add(
				side.field,
				errortypes.ErrRequired,
				"{0} radius needs a centre point", side.label,
			)
		}
		return
	}

	if side.radiusMeters != nil {
		multiErr.Add(
			side.radiusField,
			errortypes.ErrInvalid,
			"{0} radius only applies to a radius lane", side.label,
		)
	}

	if !side.scope.Type.RequiresValue() {
		return
	}

	if side.scope.Value == "" {
		multiErr.Add(side.field, errortypes.ErrRequired, "{0} value is required", side.label)
		return
	}

	// A scope that cannot be reduced to a key would be stored and never
	// matched, which reads as a rate that silently never applies.
	if !side.keyed {
		multiErr.Add(
			side.field,
			errortypes.ErrInvalid,
			"{0} value is not valid for a {1} lane", side.label, side.scope.Type.String(),
		)
	}
}

func (rar *RateAgreementRule) validateRanges(multiErr *errortypes.MultiError) {
	validateDecimalRange(multiErr, "maxWeight", rar.MinWeight, rar.MaxWeight, "weight")
	validateDecimalRange(multiErr, "maxDistance", rar.MinDistance, rar.MaxDistance, "distance")

	if rar.MinStops != nil && rar.MaxStops != nil && *rar.MaxStops < *rar.MinStops {
		multiErr.Add(
			"maxStops",
			errortypes.ErrInvalid,
			"Maximum stops cannot be less than minimum stops",
		)
	}

	if rar.EffectiveTo != nil && *rar.EffectiveTo <= rar.EffectiveFrom {
		multiErr.Add(
			"effectiveTo",
			errortypes.ErrInvalid,
			"Effective to must be after effective from",
		)
	}
}

func validateDecimalRange(
	multiErr *errortypes.MultiError,
	field string,
	minimum, maximum decimal.NullDecimal,
	label string,
) {
	if minimum.Valid && minimum.Decimal.IsNegative() {
		multiErr.Add(field, errortypes.ErrInvalid, "Minimum {0} cannot be negative", label)
	}

	if minimum.Valid && maximum.Valid && maximum.Decimal.LessThanOrEqual(minimum.Decimal) {
		multiErr.Add(
			field,
			errortypes.ErrInvalid,
			"Maximum {0} must be greater than minimum {1}", label, label,
		)
	}
}

func (rar *RateAgreementRule) BeforeCreate(ctx context.Context) error {
	if rar.Status == "" {
		rar.Status = RuleStatusActive
	}
	if rar.Direction == "" {
		rar.Direction = DirectionDirectional
	}
	rar.applyDefaults()
	if rar.LaneKey == "" {
		return ErrLaneNotApplied
	}

	return nil
}

func (rar *RateAgreementRule) GetID() pulid.ID {
	return rar.ID
}

func (rar *RateAgreementRule) GetCreatedAt() int64 {
	return rar.CreatedAt
}

func (rar *RateAgreementRule) GetOrganizationID() pulid.ID {
	return rar.OrganizationID
}

func (rar *RateAgreementRule) GetBusinessUnitID() pulid.ID {
	return rar.BusinessUnitID
}

func (rar *RateAgreementRule) GetTableName() string {
	return "rate_agreement_rules"
}
