package configproto

import (
	"github.com/emoss08/trenova/shared/edi/pkg/configtypes"
	configpb "github.com/emoss08/trenova/shared/edi/proto/config/v1"
)

// ToProto converts a configtypes.PartnerConfig into the gRPC protobuf message.
func ToProto(in *configtypes.PartnerConfig) *configpb.PartnerConfig {
	if in == nil {
		return nil
	}
	out := &configpb.PartnerConfig{
		Name:   in.Name,
		Schema: in.SchemaPath,
		Delimiters: &configpb.DelimiterConfig{
			Element:    in.Delims.Element,
			Component:  in.Delims.Component,
			Segment:    in.Delims.Segment,
			Repetition: in.Delims.Repetition,
		},
		Validation: &configpb.ValidationConfig{
			Strictness: toProtoStrictness(in.Validation.Strictness),
		},
		StopTypeMap:         map[string]string{},
		ShipmentIdMode:      in.ShipmentIDMode,
		CarrierScacFallback: in.CarrierSCACFallback,
		IncludeRawL11:       in.IncludeRawL11,
		RawL11Filter:        append([]string{}, in.RawL11Filter...),
		EquipmentTypeMap:    map[string]string{},
		IncludeSegments:     in.IncludeSegments,
		EmitIsoDatetime:     in.EmitISODateTime,
		Timezone:            in.Timezone,
		ServiceLevelQuals:   append([]string{}, in.ServiceLevelQuals...),
		ServiceLevelMap:     map[string]string{},
		AccessorialQuals:    append([]string{}, in.AccessorialQuals...),
		AccessorialMap:      map[string]string{},
		ShipmentIdQuals:     append([]string{}, in.ShipmentIDQuals...),
	}
	// Optional validation flags
	out.Validation.EnforceSeCount = in.Validation.EnforceSECount
	out.Validation.RequirePickupAndDelivery = in.Validation.RequirePickupAndDelivery
	out.Validation.RequireB2ShipId = in.Validation.RequireB2ShipID
	out.Validation.RequireN1Sh = in.Validation.RequireN1SH
	out.Validation.RequireN1St = in.Validation.RequireN1ST

	// References map[string][]string -> repeated entries
	if len(in.References) > 0 {
		out.References = make([]*configpb.ReferencesEntry, 0, len(in.References))
		for k, vs := range in.References {
			out.References = append(
				out.References,
				&configpb.ReferencesEntry{
					Key:  k,
					List: &configpb.StringList{Values: append([]string{}, vs...)},
				},
			)
		}
	}
	// Party roles map
	if len(in.PartyRoles) > 0 {
		out.PartyRoles = make([]*configpb.PartyRolesEntry, 0, len(in.PartyRoles))
		for role, codes := range in.PartyRoles {
			out.PartyRoles = append(
				out.PartyRoles,
				&configpb.PartyRolesEntry{Role: role, N1Codes: append([]string{}, codes...)},
			)
		}
	}
	// Simple maps copy
	for k, v := range in.StopTypeMap {
		out.StopTypeMap[k] = v
	}
	for k, v := range in.EquipmentTypeMap {
		out.EquipmentTypeMap[k] = v
	}
	for k, v := range in.ServiceLevelMap {
		out.ServiceLevelMap[k] = v
	}
	for k, v := range in.AccessorialMap {
		out.AccessorialMap[k] = v
	}
	return out
}

// FromProto converts the proto PartnerConfig into configtypes.PartnerConfig.
func FromProto(in *configpb.PartnerConfig) *configtypes.PartnerConfig {
	if in == nil {
		return nil
	}
	out := &configtypes.PartnerConfig{
		Name:       in.GetName(),
		SchemaPath: in.GetSchema(),
		Delims: configtypes.DelimiterConfig{
			Element:    in.GetDelimiters().GetElement(),
			Component:  in.GetDelimiters().GetComponent(),
			Segment:    in.GetDelimiters().GetSegment(),
			Repetition: in.GetDelimiters().GetRepetition(),
		},
		Validation: configtypes.ValidationConfig{
			Strictness: fromProtoStrictness(in.GetValidation().GetStrictness()),
		},
		References:          map[string][]string{},
		PartyRoles:          map[string][]string{},
		StopTypeMap:         map[string]string{},
		ShipmentIDQuals:     append([]string{}, in.GetShipmentIdQuals()...),
		ShipmentIDMode:      in.GetShipmentIdMode(),
		CarrierSCACFallback: in.GetCarrierScacFallback(),
		IncludeRawL11:       in.GetIncludeRawL11(),
		RawL11Filter:        append([]string{}, in.GetRawL11Filter()...),
		EquipmentTypeMap:    map[string]string{},
		IncludeSegments:     in.GetIncludeSegments(),
		EmitISODateTime:     in.GetEmitIsoDatetime(),
		Timezone:            in.GetTimezone(),
		ServiceLevelQuals:   append([]string{}, in.GetServiceLevelQuals()...),
		ServiceLevelMap:     map[string]string{},
		AccessorialQuals:    append([]string{}, in.GetAccessorialQuals()...),
		AccessorialMap:      map[string]string{},
	}
	// Optional validation flags
	if v := in.GetValidation().GetEnforceSeCount(); v != false {
		out.Validation.EnforceSECount = &v
	}
	if v := in.GetValidation().GetRequirePickupAndDelivery(); v != false {
		out.Validation.RequirePickupAndDelivery = &v
	}
	if v := in.GetValidation().GetRequireB2ShipId(); v != false {
		out.Validation.RequireB2ShipID = &v
	}
	if v := in.GetValidation().GetRequireN1Sh(); v != false {
		out.Validation.RequireN1SH = &v
	}
	if v := in.GetValidation().GetRequireN1St(); v != false {
		out.Validation.RequireN1ST = &v
	}

	// References
	for _, e := range in.GetReferences() {
		if e == nil || e.List == nil {
			continue
		}
		out.References[e.Key] = append([]string{}, e.List.Values...)
	}
	// Party roles
	for _, pr := range in.GetPartyRoles() {
		if pr == nil {
			continue
		}
		out.PartyRoles[pr.Role] = append([]string{}, pr.N1Codes...)
	}
	// Maps
	for k, v := range in.GetStopTypeMap() {
		out.StopTypeMap[k] = v
	}
	for k, v := range in.GetEquipmentTypeMap() {
		out.EquipmentTypeMap[k] = v
	}
	for k, v := range in.GetServiceLevelMap() {
		out.ServiceLevelMap[k] = v
	}
	for k, v := range in.GetAccessorialMap() {
		out.AccessorialMap[k] = v
	}
	return out
}

func toProtoStrictness(s configtypes.Strictness) configpb.Strictness {
	switch s {
	case configtypes.Lenient:
		return configpb.Strictness_LENIENT
	default:
		return configpb.Strictness_STRICT
	}
}

func fromProtoStrictness(s configpb.Strictness) configtypes.Strictness {
	switch s {
	case configpb.Strictness_LENIENT:
		return configtypes.Lenient
	default:
		return configtypes.Strict
	}
}
