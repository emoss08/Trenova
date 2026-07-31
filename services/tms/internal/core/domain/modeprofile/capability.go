package modeprofile

import (
	"errors"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
)

var (
	ErrInvalidCapability = errors.New("invalid capability")
	ErrUnknownRuleKey    = errors.New("unknown rule key")
)

type Capability string

const (
	CapabilityCore                     = Capability("Core")
	CapabilityTemperatureControl       = Capability("TemperatureControl")
	CapabilityHazmat                   = Capability("Hazmat")
	CapabilityDimensionalCargo         = Capability("DimensionalCargo")
	CapabilityPermitting               = Capability("Permitting")
	CapabilityCompartmentizedCargo     = Capability("CompartmentizedCargo")
	CapabilityMeteredQuantity          = Capability("MeteredQuantity")
	CapabilityPieceLevelTracking       = Capability("PieceLevelTracking")
	CapabilityContainerizedEquipment   = Capability("ContainerizedEquipment")
	CapabilityCrewBased                = Capability("CrewBased")
	CapabilityMultiShipmentConsolidate = Capability("MultiShipmentConsolidation")
	CapabilityAppointmentRequired      = Capability("AppointmentRequired")
)

func (c Capability) String() string { return string(c) }

func (c Capability) IsValid() bool {
	_, ok := capabilityLabels[c]
	return ok
}

func CapabilityFromString(v string) (Capability, error) {
	c := Capability(v)
	if !c.IsValid() {
		return "", ErrInvalidCapability
	}
	return c, nil
}

var capabilityLabels = map[Capability]string{
	CapabilityCore:                     "Core Operations",
	CapabilityTemperatureControl:       "Temperature Control",
	CapabilityHazmat:                   "Hazardous Materials",
	CapabilityDimensionalCargo:         "Dimensional Cargo",
	CapabilityPermitting:               "Permitting",
	CapabilityCompartmentizedCargo:     "Compartmentized Cargo",
	CapabilityMeteredQuantity:          "Metered Quantity",
	CapabilityPieceLevelTracking:       "Piece Level Tracking",
	CapabilityContainerizedEquipment:   "Containerized Equipment",
	CapabilityCrewBased:                "Crew Based Execution",
	CapabilityMultiShipmentConsolidate: "Multi Shipment Consolidation",
	CapabilityAppointmentRequired:      "Appointment Scheduling",
}

func (c Capability) Label() string {
	if label, ok := capabilityLabels[c]; ok {
		return label
	}
	return string(c)
}

func AllCapabilities() []Capability {
	caps := make([]Capability, 0, len(capabilityLabels))
	for c := range capabilityLabels {
		caps = append(caps, c)
	}
	sort.Slice(caps, func(i, j int) bool { return caps[i] < caps[j] })
	return caps
}

type RuleKey string

const (
	RuleKeyHazmatSegregation = RuleKey("hazmat.segregation")
	RuleKeyDuplicateBOL      = RuleKey("documentation.duplicateBol")
	RuleKeyMoveRemoval       = RuleKey("dispatch.moveRemoval")
	RuleKeyMaxShipmentWeight = RuleKey("cargo.maxShipmentWeight")
	RuleKeyTemperatureRange  = RuleKey("cargo.temperatureRange")

	RuleKeyDimensionsRequired       = RuleKey("cargo.dimensionsRequired")
	RuleKeyEnvelopeExceedsEquipment = RuleKey("cargo.envelopeExceedsEquipment")
)

func (r RuleKey) String() string { return string(r) }

const (
	ParamMaxWeight          = "maxWeight"
	ParamRequireBothBounds  = "requireBothBounds"
	ParamMaxOverhangFeet    = "maxOverhangFeet"
	DefaultMaxShipmentLimit = 80000
	DefaultMaxOverhangFeet  = 4
)

type RuleParameter struct {
	Name     string                `json:"name"`
	Label    string                `json:"label"`
	Type     customfield.FieldType `json:"type"`
	Required bool                  `json:"required"`
	Default  any                   `json:"default,omitempty"`
}

type RuleDefinition struct {
	Key                RuleKey                 `json:"key"`
	Capability         Capability              `json:"capability"`
	Label              string                  `json:"label"`
	Rationale          string                  `json:"rationale"`
	DefaultEnforcement tenant.EnforcementLevel `json:"defaultEnforcement"`
	Fields             []string                `json:"fields,omitempty"`
	Parameters         []RuleParameter         `json:"parameters,omitempty"`
}

var ruleDefinitions = map[RuleKey]RuleDefinition{
	RuleKeyHazmatSegregation: {
		Key:        RuleKeyHazmatSegregation,
		Capability: CapabilityHazmat,
		Label:      "Hazmat Segregation",
		Rationale: "Incompatible hazardous materials placed on the same shipment " +
			"violate 49 CFR 177.848 segregation requirements and expose the carrier " +
			"to civil penalties and cargo liability.",
		DefaultEnforcement: tenant.EnforcementLevelBlock,
		Fields:             []string{"commodities"},
	},
	RuleKeyDuplicateBOL: {
		Key:        RuleKeyDuplicateBOL,
		Capability: CapabilityCore,
		Label:      "Duplicate Bill of Lading",
		Rationale: "A bill of lading reused across shipments for the same customer " +
			"usually indicates a double-entered load, which double-bills the customer " +
			"and corrupts revenue reporting.",
		DefaultEnforcement: tenant.EnforcementLevelBlock,
		Fields:             []string{"bol"},
	},
	RuleKeyMoveRemoval: {
		Key:        RuleKeyMoveRemoval,
		Capability: CapabilityCore,
		Label:      "Move Removal",
		Rationale: "Removing a move after dispatch detaches completed stop history " +
			"from the shipment, breaking detention accrual and driver pay traceability.",
		DefaultEnforcement: tenant.EnforcementLevelBlock,
		Fields:             []string{"moves"},
	},
	RuleKeyMaxShipmentWeight: {
		Key:        RuleKeyMaxShipmentWeight,
		Capability: CapabilityCore,
		Label:      "Maximum Shipment Weight",
		Rationale: "Gross weight above the federal 80,000 lb limit requires an " +
			"overweight permit. Dispatching without one risks fines at the scale and " +
			"an out-of-service order.",
		DefaultEnforcement: tenant.EnforcementLevelBlock,
		Fields:             []string{"weight"},
		Parameters: []RuleParameter{
			{
				Name:     ParamMaxWeight,
				Label:    "Maximum Weight (lbs)",
				Type:     customfield.FieldTypeNumber,
				Required: true,
				Default:  DefaultMaxShipmentLimit,
			},
		},
	},
	RuleKeyDimensionsRequired: {
		Key:        RuleKeyDimensionsRequired,
		Capability: CapabilityDimensionalCargo,
		Label:      "Cargo Dimensions",
		Rationale: "Open deck freight cannot be planned or permitted without length, " +
			"width, and height on every line. Deck fit, overhang, and every " +
			"jurisdiction permit threshold are computed from these numbers.",
		DefaultEnforcement: tenant.EnforcementLevelBlock,
		Fields:             []string{"commodities"},
	},
	RuleKeyEnvelopeExceedsEquipment: {
		Key:        RuleKeyEnvelopeExceedsEquipment,
		Capability: CapabilityDimensionalCargo,
		Label:      "Cargo Exceeds Deck",
		Rationale: "Cargo longer than the deck overhangs. Past the allowed overhang " +
			"the load needs different equipment or an over-length permit, and " +
			"discovering that at the shipper costs a day.",
		DefaultEnforcement: tenant.EnforcementLevelBlock,
		Fields:             []string{"trailerTypeId", "commodities"},
		Parameters: []RuleParameter{
			{
				Name:     ParamMaxOverhangFeet,
				Label:    "Allowed Rear Overhang (ft)",
				Type:     customfield.FieldTypeNumber,
				Required: false,
				Default:  DefaultMaxOverhangFeet,
			},
		},
	},
	RuleKeyTemperatureRange: {
		Key:        RuleKeyTemperatureRange,
		Capability: CapabilityTemperatureControl,
		Label:      "Temperature Range",
		Rationale: "Temperature-controlled freight without a declared setpoint range " +
			"cannot be defended against a claim, and FSMA sanitary transport rules " +
			"require the agreed temperature to be recorded before pickup.",
		DefaultEnforcement: tenant.EnforcementLevelBlock,
		Fields:             []string{"temperatureMin", "temperatureMax"},
		Parameters: []RuleParameter{
			{
				Name:     ParamRequireBothBounds,
				Label:    "Require Both Bounds",
				Type:     customfield.FieldTypeBoolean,
				Required: false,
				Default:  true,
			},
		},
	},
}

func RuleDefinitionFor(key RuleKey) (RuleDefinition, error) {
	def, ok := ruleDefinitions[key]
	if !ok {
		return RuleDefinition{}, ErrUnknownRuleKey
	}
	return def, nil
}

func IsKnownRuleKey(key RuleKey) bool {
	_, ok := ruleDefinitions[key]
	return ok
}

func AllRuleDefinitions() []RuleDefinition {
	defs := make([]RuleDefinition, 0, len(ruleDefinitions))
	for _, def := range ruleDefinitions {
		defs = append(defs, def)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Key < defs[j].Key })
	return defs
}

func RuleKeysForCapability(capability Capability) []RuleKey {
	keys := make([]RuleKey, 0, len(ruleDefinitions))
	for key, def := range ruleDefinitions {
		if def.Capability == capability {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
