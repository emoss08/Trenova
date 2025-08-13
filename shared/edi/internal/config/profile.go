package config

import (
	"os"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

type DelimiterConfig struct {
	Element    string `json:"element"`
	Component  string `json:"component"`
	Segment    string `json:"segment"`
	Repetition string `json:"repetition"`
}

type ValidationConfig struct {
	Strictness               string `json:"strictness"` // "strict" or "lenient"
	EnforceSECount           *bool  `json:"enforce_se_count,omitempty"`
	RequirePickupAndDelivery *bool  `json:"require_pickup_and_delivery,omitempty"`
	RequireB2ShipID          *bool  `json:"require_b2_ship_id,omitempty"`
	RequireN1SH              *bool  `json:"require_n1_sh,omitempty"`
	RequireN1ST              *bool  `json:"require_n1_st,omitempty"`
}

type PartnerConfig struct {
	Name       string           `json:"name"`
	SchemaPath string           `json:"schema"`
	Delims     DelimiterConfig  `json:"delimiters"`
	Validation ValidationConfig `json:"validation"`
	// References maps DTO reference keys to a list of L11 qualifiers to pull from.
	// Example: { "customer_po": ["PO"], "bill_of_lading": ["BM"], "shipment_ref": ["SI", "CR"] }
	References map[string][]string `json:"references,omitempty"`
	// PartyRoles maps DTO party roles to N1 entity codes (e.g., SH, ST, BT, SF, CN).
	// Example: { "shipper": ["SH", "SF"], "consignee": ["ST", "CN"], "bill_to": ["BT"] }
	PartyRoles map[string][]string `json:"party_roles,omitempty"`
	// StopTypeMap maps S5 stop type codes to normalized types: pickup|delivery|other.
	// Example: { "LD": "pickup", "CL": "pickup", "UL": "delivery", "CU": "delivery" }
	StopTypeMap map[string]string `json:"stop_type_map,omitempty"`
	// ShipmentIDQuals lists L11 qualifiers to consider as the ShipmentID before falling back to B2-03.
	ShipmentIDQuals []string `json:"shipment_id_quals,omitempty"`
	// ShipmentIDMode chooses how to select shipment_id: ref_first|b2_first|ref_only|b2_only
	ShipmentIDMode string `json:"shipment_id_mode,omitempty"`
	// CarrierSCACFallback provides a fixed SCAC when B2-02 is missing.
	CarrierSCACFallback string `json:"carrier_scac_fallback,omitempty"`
	// IncludeRawL11 controls whether to include all raw L11 references in the Shipment output.
	IncludeRawL11 bool `json:"include_raw_l11,omitempty"`
	// RawL11Filter limits which L11 qualifiers are emitted when IncludeRawL11 is true. Empty means all.
	RawL11Filter []string `json:"raw_l11_filter,omitempty"`
	// EquipmentTypeMap maps raw equipment types to normalized values.
	EquipmentTypeMap map[string]string `json:"equipment_type_map,omitempty"`
	// IncludeSegments controls whether to include raw parsed segments in shipment output.
	IncludeSegments bool `json:"include_segments,omitempty"`
	// EmitISODateTime controls whether to normalize appointment date/time to ISO-8601 in the DTO.
	EmitISODateTime bool `json:"emit_iso_datetime,omitempty"`
	// Timezone specifies the IANA timezone name used for datetime normalization (default UTC).
	Timezone string `json:"timezone,omitempty"`
	// ServiceLevelQuals: L11 qualifiers used to extract service level from header references.
	ServiceLevelQuals []string `json:"service_level_quals,omitempty"`
	// ServiceLevelMap: normalize raw service level values to canonical names.
	ServiceLevelMap map[string]string `json:"service_level_map,omitempty"`
	// AccessorialQuals: L11 qualifiers to treat as accessorial codes.
	AccessorialQuals []string `json:"accessorial_quals,omitempty"`
	// AccessorialMap: normalize accessorial codes to canonical names.
	AccessorialMap map[string]string `json:"accessorial_map,omitempty"`
}

func Load(path string) (*PartnerConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pc PartnerConfig
	if err := sonic.Unmarshal(b, &pc); err != nil {
		return nil, err
	}
	return &pc, nil
}

// ApplyDelimiters applies configured delimiters on top of detected ones.
func (p *PartnerConfig) ApplyDelimiters(d *x12.Delimiters) {
	if len(p.Delims.Element) > 0 {
		d.Element = p.Delims.Element[0]
	}
	if len(p.Delims.Component) > 0 {
		d.Component = p.Delims.Component[0]
	}
	if len(p.Delims.Segment) > 0 {
		d.Segment = p.Delims.Segment[0]
	}
	if len(p.Delims.Repetition) > 0 {
		d.Repetition = p.Delims.Repetition[0]
	}
}

// ApplyValidation overrides fields on a base validation profile.
func (p *PartnerConfig) ApplyValidation(base validation.Profile) validation.Profile {
	prof := base
	switch strings.ToLower(p.Validation.Strictness) {
	case "lenient":
		prof.Strictness = validation.Lenient
	case "strict":
		prof.Strictness = validation.Strict
	}
	if p.Validation.EnforceSECount != nil {
		prof.EnforceSECount = *p.Validation.EnforceSECount
	}
	if p.Validation.RequirePickupAndDelivery != nil {
		prof.RequirePickupAndDelivery = *p.Validation.RequirePickupAndDelivery
	}
	if p.Validation.RequireB2ShipID != nil {
		prof.RequireB2ShipID = *p.Validation.RequireB2ShipID
	}
	if p.Validation.RequireN1SH != nil {
		prof.RequireN1SH = *p.Validation.RequireN1SH
	}
	if p.Validation.RequireN1ST != nil {
		prof.RequireN1ST = *p.Validation.RequireN1ST
	}
	return prof
}
