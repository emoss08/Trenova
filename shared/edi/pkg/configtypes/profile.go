package configtypes

// DelimiterConfig describes X12 separators for an interchange/partner.
type DelimiterConfig struct {
	Element    string `json:"element"`
	Component  string `json:"component"`
	Segment    string `json:"segment"`
	Repetition string `json:"repetition"`
}

// ValidationConfig controls validation behavior and requiredness per partner/version.
type ValidationConfig struct {
	Strictness               Strictness `json:"strictness"` // "strict" or "lenient"
	EnforceSECount           *bool      `json:"enforceSeCount,omitempty"`
	RequirePickupAndDelivery *bool      `json:"requirePickupAndDelivery,omitempty"`
	RequireB2ShipID          *bool      `json:"requireB2ShipID,omitempty"`
	RequireN1SH              *bool      `json:"requireN1SH,omitempty"`
	RequireN1ST              *bool      `json:"requireN1ST,omitempty"`
}

// PartnerConfig is the portable profile shape intended for sharing across services.
// It mirrors the JSON used by the CLI and provides an import-friendly type for the TMS.
type PartnerConfig struct {
	Name       string           `json:"name"`
	SchemaPath string           `json:"schema"`
	Delims     DelimiterConfig  `json:"delimiters"`
	Validation ValidationConfig `json:"validation"`

	// Mapping options
	References          map[string][]string `json:"references,omitempty"`
	PartyRoles          map[string][]string `json:"party_roles,omitempty"`
	StopTypeMap         map[string]string   `json:"stop_type_map,omitempty"`
	ShipmentIDQuals     []string            `json:"shipment_id_quals,omitempty"`
	ShipmentIDMode      string              `json:"shipment_id_mode,omitempty"`
	CarrierSCACFallback string              `json:"carrier_scac_fallback,omitempty"`
	IncludeRawL11       bool                `json:"include_raw_l11,omitempty"`
	RawL11Filter        []string            `json:"raw_l11_filter,omitempty"`
	EquipmentTypeMap    map[string]string   `json:"equipment_type_map,omitempty"`
	IncludeSegments     bool                `json:"include_segments,omitempty"`
	EmitISODateTime     bool                `json:"emit_iso_datetime,omitempty"`
	Timezone            string              `json:"timezone,omitempty"`
	ServiceLevelQuals   []string            `json:"service_level_quals,omitempty"`
	ServiceLevelMap     map[string]string   `json:"service_level_map,omitempty"`
	AccessorialQuals    []string            `json:"accessorial_quals,omitempty"`
	AccessorialMap      map[string]string   `json:"accessorial_map,omitempty"`
}
