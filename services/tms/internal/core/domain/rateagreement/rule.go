package rateagreement

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/postgis"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateAgreementRule)(nil)

const (
	maxLabelLength = 150
	// DaysOfWeekAll is the bitmask that matches every day, which is also what
	// an unset field means.
	DaysOfWeekAll = int16(0)
)

// RateAgreementRule is one priced lane within an agreement.
//
// Everything the engine needs to decide whether this rule applies, and what it
// charges if it does, lives on this row. LaneKey and SpecificityScore are both
// derived from the other fields and recomputed on every write, so the values
// the database indexes and orders by can never disagree with the values a
// person edited.
type RateAgreementRule struct {
	bun.BaseModel `bun:"table:rate_agreement_rules,alias:ragr" json:"-"`

	ID              pulid.ID `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID  pulid.ID `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RateAgreementID pulid.ID `json:"rateAgreementId" bun:"rate_agreement_id,type:VARCHAR(100),notnull"`

	// PartyType duplicates the agreement's own value so the resolution index
	// can lead with it without joining. It is stamped from the parent, never
	// supplied.
	PartyType PartyType  `json:"partyType" bun:"party_type,type:rate_agreement_party_type_enum,notnull"`
	Label     string     `json:"label"     bun:"label,type:VARCHAR(150),nullzero"`
	Status    RuleStatus `json:"status"    bun:"status,type:rate_agreement_rule_status_enum,notnull,default:'Active'"`

	OriginScopeType  rategeo.ScopeType `json:"originScopeType"  bun:"origin_scope_type,type:rate_geo_scope_type_enum,notnull"`
	OriginScopeValue string            `json:"originScopeValue" bun:"origin_scope_value,type:VARCHAR(120),nullzero"`
	OriginCity       string            `json:"originCity"       bun:"origin_city,type:VARCHAR(100),nullzero"`

	DestinationScopeType  rategeo.ScopeType `json:"destinationScopeType"  bun:"destination_scope_type,type:rate_geo_scope_type_enum,notnull"`
	DestinationScopeValue string            `json:"destinationScopeValue" bun:"destination_scope_value,type:VARCHAR(120),nullzero"`
	DestinationCity       string            `json:"destinationCity"       bun:"destination_city,type:VARCHAR(100),nullzero"`

	// LaneKey is the origin and destination keys joined. A rule stores one; a
	// shipment produces the handful it could have stored. That turns lane
	// matching into a set membership test the database answers from an index.
	LaneKey   string    `json:"laneKey"   bun:"lane_key,type:VARCHAR(255),notnull"`
	Direction Direction `json:"direction" bun:"direction,type:rate_agreement_direction_enum,notnull,default:'Directional'"`

	OriginRadiusMeters      *float64       `json:"originRadiusMeters"      bun:"origin_radius_meters,type:DOUBLE PRECISION,nullzero"`
	DestinationRadiusMeters *float64       `json:"destinationRadiusMeters" bun:"destination_radius_meters,type:DOUBLE PRECISION,nullzero"`
	OriginLatitude          *float64       `json:"originLatitude"          bun:"origin_latitude,type:FLOAT,nullzero"`
	OriginLongitude         *float64       `json:"originLongitude"         bun:"origin_longitude,type:FLOAT,nullzero"`
	DestinationLatitude     *float64       `json:"destinationLatitude"     bun:"destination_latitude,type:FLOAT,nullzero"`
	DestinationLongitude    *float64       `json:"destinationLongitude"    bun:"destination_longitude,type:FLOAT,nullzero"`
	OriginCenter            *postgis.Point `json:"-"                       bun:"origin_center,type:geography,scanonly"`
	DestinationCenter       *postgis.Point `json:"-"                       bun:"destination_center,type:geography,scanonly"`

	// An empty applicability set means the rule does not care about that
	// dimension. This is the same convention detention policies and fuel
	// surcharge programs already use, so a rule that names nothing matches
	// everything rather than nothing.
	ServiceTypeIDs   []pulid.ID                   `json:"serviceTypeIds"   bun:"service_type_ids,type:JSONB,nullzero"`
	ShipmentTypeIDs  []pulid.ID                   `json:"shipmentTypeIds"  bun:"shipment_type_ids,type:JSONB,nullzero"`
	TractorTypeIDs   []pulid.ID                   `json:"tractorTypeIds"   bun:"tractor_type_ids,type:JSONB,nullzero"`
	TrailerTypeIDs   []pulid.ID                   `json:"trailerTypeIds"   bun:"trailer_type_ids,type:JSONB,nullzero"`
	CommodityIDs     []pulid.ID                   `json:"commodityIds"     bun:"commodity_ids,type:JSONB,nullzero"`
	FreightClasses   []commodity.FreightClass     `json:"freightClasses"   bun:"freight_classes,type:JSONB,nullzero"`
	ServiceModels    []modeprofile.ServiceModel   `json:"serviceModels"    bun:"service_models,type:JSONB,nullzero"`
	EquipmentClasses []modeprofile.EquipmentClass `json:"equipmentClasses" bun:"equipment_classes,type:JSONB,nullzero"`

	MinWeight   decimal.NullDecimal `json:"minWeight"   bun:"min_weight,type:NUMERIC(12,2),nullzero"`
	MaxWeight   decimal.NullDecimal `json:"maxWeight"   bun:"max_weight,type:NUMERIC(12,2),nullzero"`
	MinDistance decimal.NullDecimal `json:"minDistance" bun:"min_distance,type:NUMERIC(12,2),nullzero"`
	MaxDistance decimal.NullDecimal `json:"maxDistance" bun:"max_distance,type:NUMERIC(12,2),nullzero"`
	MinStops    *int16              `json:"minStops"    bun:"min_stops,type:SMALLINT,nullzero"`
	MaxStops    *int16              `json:"maxStops"    bun:"max_stops,type:SMALLINT,nullzero"`

	// DaysOfWeek is a bitmask with Sunday at bit zero. Zero means every day.
	DaysOfWeek      int16 `json:"daysOfWeek"      bun:"days_of_week,type:SMALLINT,notnull,default:0"`
	HazmatOnly      bool  `json:"hazmatOnly"      bun:"hazmat_only,type:BOOLEAN,notnull,default:false"`
	TempControlOnly bool  `json:"tempControlOnly" bun:"temp_control_only,type:BOOLEAN,notnull,default:false"`

	RatingBasis       RatingBasis         `json:"ratingBasis"       bun:"rating_basis,type:rate_agreement_basis_enum,notnull"`
	Rate              decimal.NullDecimal `json:"rate"              bun:"rate,type:NUMERIC(19,6),nullzero"`
	RateMatrixID      *pulid.ID           `json:"rateMatrixId"      bun:"rate_matrix_id,type:VARCHAR(100),nullzero"`
	FormulaTemplateID *pulid.ID           `json:"formulaTemplateId" bun:"formula_template_id,type:VARCHAR(100),nullzero"`
	PercentBasis      PercentBasis        `json:"percentBasis"      bun:"percent_basis,type:rate_agreement_percent_basis_enum,nullzero"`
	Currency          string              `json:"currency"          bun:"currency,type:VARCHAR(3),nullzero"`

	FreightClassSource FreightClassSource     `json:"freightClassSource" bun:"freight_class_source,type:rate_freight_class_source_enum,notnull,default:'Commodity'"`
	FixedFreightClass  commodity.FreightClass `json:"fixedFreightClass"  bun:"fixed_freight_class,type:freight_class_enum,nullzero"`
	DensityScaleID     *pulid.ID              `json:"densityScaleId"     bun:"density_scale_id,type:VARCHAR(100),nullzero"`
	DiscountPercent    decimal.NullDecimal    `json:"discountPercent"    bun:"discount_percent,type:NUMERIC(9,4),nullzero"`
	AbsoluteMinCharge  decimal.NullDecimal    `json:"absoluteMinCharge"  bun:"absolute_min_charge,type:NUMERIC(19,4),nullzero"`
	AllowDeficitRating bool                   `json:"allowDeficitRating" bun:"allow_deficit_rating,type:BOOLEAN,notnull,default:true"`

	MinCharge           decimal.NullDecimal    `json:"minCharge"           bun:"min_charge,type:NUMERIC(19,4),nullzero"`
	MaxCharge           decimal.NullDecimal    `json:"maxCharge"           bun:"max_charge,type:NUMERIC(19,4),nullzero"`
	MinBillableDistance decimal.NullDecimal    `json:"minBillableDistance" bun:"min_billable_distance,type:NUMERIC(12,2),nullzero"`
	RoundingMode        ratetypes.RoundingMode `json:"roundingMode"        bun:"rounding_mode,type:rate_rounding_mode_enum,nullzero"`

	Priority         int16 `json:"priority"         bun:"priority,type:SMALLINT,notnull,default:0"`
	SpecificityScore int32 `json:"specificityScore" bun:"specificity_score,type:INTEGER,notnull,default:0"`

	EffectiveFrom     int64     `json:"effectiveFrom"     bun:"effective_from,type:BIGINT,notnull"`
	EffectiveTo       *int64    `json:"effectiveTo"       bun:"effective_to,type:BIGINT,nullzero"`
	SupersedesRuleID  *pulid.ID `json:"supersedesRuleId"  bun:"supersedes_rule_id,type:VARCHAR(100),nullzero"`
	SourceImportRowID *pulid.ID `json:"sourceImportRowId" bun:"source_import_row_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Agreement  *RateAgreement            `json:"-"                    bun:"rel:belongs-to,join:rate_agreement_id=id"`
	RateMatrix *ratematrix.RateMatrix    `json:"rateMatrix,omitempty" bun:"rel:belongs-to,join:rate_matrix_id=id"`
	Breaks     []*RateAgreementRuleBreak `json:"breaks,omitempty"     bun:"rel:has-many,join:id=rate_agreement_rule_id"`
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

// ApplyLaneKey recomputes the stored lane key from the two scopes.
//
// A radius lane has no key. It still needs a non-empty value because the column
// is not null, so it gets a sentinel that no shipment will ever generate — the
// rule is unreachable by key on purpose, and reachable only through the
// geospatial query.
func (rar *RateAgreementRule) ApplyLaneKey() {
	originKey, originOK := rar.OriginScope().Key()
	destinationKey, destinationOK := rar.DestinationScope().Key()

	if !originOK || !destinationOK {
		rar.LaneKey = RadiusLaneKey
		return
	}

	rar.LaneKey = rategeo.LaneKey(originKey, destinationKey)
}

// RadiusLaneKey is the placeholder stored on lanes that are matched
// geospatially. It cannot collide with a real key because no scope encodes to
// this string.
const RadiusLaneKey = "RADIUS"

// ReverseLaneKey is the lane key with the two ends swapped, which is what a
// bidirectional rule must also be reachable by.
func (rar *RateAgreementRule) ReverseLaneKey() (string, bool) {
	if rar.Direction != DirectionBidirectional {
		return "", false
	}

	originKey, originOK := rar.OriginScope().Key()
	destinationKey, destinationOK := rar.DestinationScope().Key()
	if !originOK || !destinationOK {
		return "", false
	}

	return rategeo.LaneKey(destinationKey, originKey), true
}

// ComputeSpecificity scores how narrowly the rule is written.
//
// Geography is scaled clear of every condition, so a tighter lane always wins
// regardless of how many conditions a broader rule piles on. See specificity.go
// for why the weights are fixed constants.
func (rar *RateAgreementRule) ComputeSpecificity() int32 {
	score := rategeo.LaneSpecificity(rar.OriginScopeType, rar.DestinationScopeType) * GeographyScale

	if len(rar.CommodityIDs) > 0 {
		score += ConditionWeightCommodity
	}
	if len(rar.FreightClasses) > 0 {
		score += ConditionWeightFreightClass
	}
	if len(rar.TractorTypeIDs) > 0 || len(rar.TrailerTypeIDs) > 0 ||
		len(rar.EquipmentClasses) > 0 {
		score += ConditionWeightEquipmentType
	}
	if len(rar.ServiceTypeIDs) > 0 {
		score += ConditionWeightServiceType
	}
	if len(rar.ShipmentTypeIDs) > 0 {
		score += ConditionWeightShipmentType
	}
	if rar.MinWeight.Valid || rar.MaxWeight.Valid {
		score += ConditionWeightWeightRange
	}
	if rar.MinDistance.Valid || rar.MaxDistance.Valid {
		score += ConditionWeightDistanceRange
	}
	if len(rar.ServiceModels) > 0 {
		score += ConditionWeightServiceModel
	}
	if rar.HazmatOnly {
		score += ConditionWeightHazmat
	}
	if rar.TempControlOnly {
		score += ConditionWeightTempControl
	}

	return score
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

// EffectiveRoundingMode falls back to the agreement's mode when the rule does
// not override it.
func (rar *RateAgreementRule) EffectiveRoundingMode(
	agreement *RateAgreement,
) ratetypes.RoundingMode {
	if rar.RoundingMode != "" {
		return rar.RoundingMode
	}

	if agreement != nil && agreement.RoundingMode != "" {
		return agreement.RoundingMode
	}

	return ratetypes.RoundingModeHalfUp
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
	if rar.FreightClassSource == "" {
		rar.FreightClassSource = FreightClassSourceCommodity
	}
	if rar.OriginScopeType == "" {
		rar.OriginScopeType = rategeo.ScopeTypeAny
	}
	if rar.DestinationScopeType == "" {
		rar.DestinationScopeType = rategeo.ScopeTypeAny
	}

	rar.ApplyLaneKey()
	rar.SpecificityScore = rar.ComputeSpecificity()
}

// ValidateWithin checks the rule against the agreement it belongs to, which is
// the only place the two can be compared — a rule's currency defaults from the
// header, and a sell-side rule may not be written as a percentage of the sell
// rate.
func (rar *RateAgreementRule) ValidateWithin(
	agreement *RateAgreement,
	multiErr *errortypes.MultiError,
) {
	rar.Validate(multiErr)

	if agreement == nil {
		return
	}

	if rar.PercentBasis == PercentBasisSellRate && agreement.PartyType != PartyTypeCarrier {
		multiErr.Add(
			"percentBasis",
			errortypes.ErrInvalid,
			"Only a carrier agreement can be priced as a share of the sell rate",
		)
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
		validation.Field(&rar.RatingBasis,
			validation.Required.Error("Rating basis is required"),
			domainvalidation.ValidEnum[RatingBasis]("Rating basis is invalid"),
		),
		validation.Field(&rar.PercentBasis,
			domainvalidation.ValidEnum[PercentBasis]("Percent basis is invalid"),
		),
		validation.Field(&rar.FreightClassSource,
			validation.Required.Error("Freight class source is required"),
			domainvalidation.ValidEnum[FreightClassSource]("Freight class source is invalid"),
		),
		validation.Field(&rar.FixedFreightClass,
			domainvalidation.ValidEnum[commodity.FreightClass]("Fixed freight class is invalid"),
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
				domainvalidation.ValidEnum[modeprofile.EquipmentClass]("Equipment class is invalid"),
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
	rar.validatePricing(multiErr)
	rar.validateFreightClass(multiErr)
	rar.validateRanges(multiErr)
	rar.validateBreaks(multiErr)
}

func (rar *RateAgreementRule) validateScopes(multiErr *errortypes.MultiError) {
	validateScopeSide(multiErr, scopeSide{
		field:        "originScopeValue",
		radiusField:  "originRadiusMeters",
		scope:        rar.OriginScope(),
		radiusMeters: rar.OriginRadiusMeters,
		latitude:     rar.OriginLatitude,
		longitude:    rar.OriginLongitude,
		label:        "Origin",
	})

	validateScopeSide(multiErr, scopeSide{
		field:        "destinationScopeValue",
		radiusField:  "destinationRadiusMeters",
		scope:        rar.DestinationScope(),
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
	radiusMeters *float64
	latitude     *float64
	longitude    *float64
	label        string
}

func validateScopeSide(multiErr *errortypes.MultiError, side scopeSide) {
	if side.scope.Type == rategeo.ScopeTypeRadius {
		if side.radiusMeters == nil || *side.radiusMeters <= 0 {
			multiErr.Add(
				side.radiusField,
				errortypes.ErrRequired,
				side.label+" radius must be greater than zero",
			)
		}
		if side.latitude == nil || side.longitude == nil {
			multiErr.Add(
				side.field,
				errortypes.ErrRequired,
				side.label+" radius needs a centre point",
			)
		}
		return
	}

	if side.radiusMeters != nil {
		multiErr.Add(
			side.radiusField,
			errortypes.ErrInvalid,
			side.label+" radius only applies to a radius lane",
		)
	}

	if !side.scope.Type.RequiresValue() {
		return
	}

	if side.scope.Value == "" {
		multiErr.Add(side.field, errortypes.ErrRequired, side.label+" value is required")
		return
	}

	// A scope that cannot be reduced to a key would be stored and never
	// matched, which reads as a rate that silently never applies.
	if _, ok := side.scope.Key(); !ok {
		multiErr.Add(
			side.field,
			errortypes.ErrInvalid,
			side.label+" value is not valid for a "+side.scope.Type.String()+" lane",
		)
	}
}

// validatePricing insists on exactly the one input the chosen basis reads.
// A rule carrying both a rate and a matrix leaves the engine to guess, and a
// rule carrying neither prices at nothing.
func (rar *RateAgreementRule) validatePricing(multiErr *errortypes.MultiError) {
	hasMatrix := rar.RateMatrixID != nil && !rar.RateMatrixID.IsNil()
	hasFormula := rar.FormulaTemplateID != nil && !rar.FormulaTemplateID.IsNil()
	hasBreaks := len(rar.Breaks) > 0

	switch rar.RatingBasis {
	case RatingBasisMatrix:
		if !hasMatrix {
			multiErr.Add("rateMatrixId", errortypes.ErrRequired, "Rate matrix is required")
		}
	case RatingBasisFormula:
		if !hasFormula {
			multiErr.Add(
				"formulaTemplateId",
				errortypes.ErrRequired,
				"Formula template is required",
			)
		}
	case RatingBasisPercent:
		if rar.PercentBasis == "" {
			multiErr.Add("percentBasis", errortypes.ErrRequired, "Percent basis is required")
		}
		if !rar.Rate.Valid {
			multiErr.Add("rate", errortypes.ErrRequired, "Rate is required")
		}
	default:
		// Banded rules carry their rates on the breaks instead of the rule.
		if !rar.Rate.Valid && !hasBreaks {
			multiErr.Add("rate", errortypes.ErrRequired, "Rate is required")
		}
	}

	if rar.RatingBasis != RatingBasisMatrix && hasMatrix {
		multiErr.Add(
			"rateMatrixId",
			errortypes.ErrInvalid,
			"A rate matrix only applies to a matrix rated rule",
		)
	}

	if rar.RatingBasis != RatingBasisFormula && hasFormula {
		multiErr.Add(
			"formulaTemplateId",
			errortypes.ErrInvalid,
			"A formula template only applies to a formula rated rule",
		)
	}

	if hasBreaks && !rar.RatingBasis.SupportsWeightBreaks() {
		multiErr.Add(
			"breaks",
			errortypes.ErrInvalid,
			"Weight breaks only apply to a hundredweight or flat rated rule",
		)
	}

	if rar.Rate.Valid && rar.Rate.Decimal.IsNegative() {
		multiErr.Add("rate", errortypes.ErrInvalid, "Rate cannot be negative")
	}
}

func (rar *RateAgreementRule) validateFreightClass(multiErr *errortypes.MultiError) {
	switch rar.FreightClassSource {
	case FreightClassSourceFixed:
		if rar.FixedFreightClass == "" {
			multiErr.Add(
				"fixedFreightClass",
				errortypes.ErrRequired,
				"A fixed class rule needs a freight class",
			)
		}
	case FreightClassSourceDensity:
		if rar.FixedFreightClass != "" {
			multiErr.Add(
				"fixedFreightClass",
				errortypes.ErrInvalid,
				"A density rated rule derives its class rather than fixing it",
			)
		}
	case FreightClassSourceCommodity:
		if rar.FixedFreightClass != "" {
			multiErr.Add(
				"fixedFreightClass",
				errortypes.ErrInvalid,
				"A commodity rated rule reads its class from the commodity",
			)
		}
	}

	if rar.DiscountPercent.Valid {
		if rar.DiscountPercent.Decimal.IsNegative() {
			multiErr.Add(
				"discountPercent",
				errortypes.ErrInvalid,
				"Discount cannot be negative",
			)
		}
		if rar.DiscountPercent.Decimal.GreaterThan(decimal.NewFromInt(hundredPercent)) {
			multiErr.Add(
				"discountPercent",
				errortypes.ErrInvalid,
				"Discount cannot exceed one hundred percent",
			)
		}
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

	if rar.MinCharge.Valid && rar.MaxCharge.Valid &&
		rar.MaxCharge.Decimal.LessThan(rar.MinCharge.Decimal) {
		multiErr.Add(
			"maxCharge",
			errortypes.ErrInvalid,
			"Maximum charge cannot be less than the minimum charge",
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
		multiErr.Add(field, errortypes.ErrInvalid, "Minimum "+label+" cannot be negative")
	}

	if minimum.Valid && maximum.Valid && maximum.Decimal.LessThanOrEqual(minimum.Decimal) {
		multiErr.Add(
			field,
			errortypes.ErrInvalid,
			"Maximum "+label+" must be greater than minimum "+label,
		)
	}
}

func (rar *RateAgreementRule) BeforeAppendModel(_ context.Context, query bun.Query) error {
	rar.applyDefaults()

	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rar.ID.IsNil() {
			rar.ID = pulid.MustNew("ragr_")
		}
		rar.CreatedAt = now
		rar.UpdatedAt = now
	case *bun.UpdateQuery:
		rar.UpdatedAt = now
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
