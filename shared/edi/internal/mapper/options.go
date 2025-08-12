package mapper

// Options configures how the 204 -> Shipment mapping behaves.
// It enables partner-level customization without code changes.
type Options struct {
    // RefMap: DTO key -> ordered L11 qualifiers to pull first value from.
    RefMap map[string][]string
    // PartyRoles: DTO party role -> ordered N1 codes to consider.
    // Supported roles: bill_to, shipper, consignee (others can be added later).
    PartyRoles map[string][]string
    // StopTypeMap: S5 type code -> normalized type (pickup|delivery|other).
    StopTypeMap map[string]string
    // ShipmentIDQuals: ordered L11 qualifiers to use for ShipmentID before falling back to B2-03.
    ShipmentIDQuals []string
    // ShipmentIDMode: ref_first|b2_first|ref_only|b2_only
    ShipmentIDMode string
    // CarrierSCACFallback: fixed SCAC if header B2-02 is empty
    CarrierSCACFallback string
    // IncludeRawL11: when true, emit references_raw with filtered qualifiers if provided.
    IncludeRawL11 bool
    RawL11Filter  []string
    // EquipmentTypeMap: raw equipment type -> normalized value
    EquipmentTypeMap map[string]string
    // Date/time normalization
    EmitISODateTime bool
    Timezone        string // IANA name or 'UTC'; default UTC
    // Service level mapping
    ServiceLevelQuals []string
    ServiceLevelMap   map[string]string
    // Accessorials mapping
    AccessorialQuals []string
    AccessorialMap   map[string]string
}

// DefaultOptions builds a baseline set of options.
func DefaultOptions() Options {
    return Options{
        RefMap:       DefaultRefMap(),
        PartyRoles:   DefaultPartyRoles(),
        StopTypeMap:  DefaultStopTypeMap(),
        ShipmentIDQuals: []string{"SI", "CR"},
        ShipmentIDMode:  "ref_first",
        IncludeRawL11:    false,
        EmitISODateTime:  false,
        Timezone:         "UTC",
    }
}

func DefaultPartyRoles() map[string][]string {
    return map[string][]string{
        "bill_to":  {"BT"},
        "shipper":  {"SH", "SF"},
        "consignee": {"ST", "CN"},
    }
}

func DefaultStopTypeMap() map[string]string {
    return map[string]string{
        "LD": "pickup",
        "CL": "pickup",
        "UL": "delivery",
        "CU": "delivery",
    }
}

// MergeOptions overlays non-empty fields from o onto base.
func MergeOptions(base, o Options) Options {
    out := base
    if o.RefMap != nil && len(o.RefMap) > 0 {
        out.RefMap = MergeRefMaps(out.RefMap, o.RefMap)
    }
    if o.PartyRoles != nil && len(o.PartyRoles) > 0 {
        if out.PartyRoles == nil {
            out.PartyRoles = map[string][]string{}
        }
        for k, v := range o.PartyRoles {
            vv := make([]string, len(v))
            copy(vv, v)
            out.PartyRoles[k] = vv
        }
    }
    if o.StopTypeMap != nil && len(o.StopTypeMap) > 0 {
        if out.StopTypeMap == nil {
            out.StopTypeMap = map[string]string{}
        }
        for k, v := range o.StopTypeMap {
            out.StopTypeMap[k] = v
        }
    }
    if len(o.ShipmentIDQuals) > 0 {
        out.ShipmentIDQuals = append([]string{}, o.ShipmentIDQuals...)
    }
    if o.ShipmentIDMode != "" {
        out.ShipmentIDMode = o.ShipmentIDMode
    }
    if o.CarrierSCACFallback != "" {
        out.CarrierSCACFallback = o.CarrierSCACFallback
    }
    if o.IncludeRawL11 {
        out.IncludeRawL11 = true
    }
    if len(o.RawL11Filter) > 0 {
        out.RawL11Filter = append([]string{}, o.RawL11Filter...)
    }
    if o.EquipmentTypeMap != nil && len(o.EquipmentTypeMap) > 0 {
        if out.EquipmentTypeMap == nil {
            out.EquipmentTypeMap = map[string]string{}
        }
        for k, v := range o.EquipmentTypeMap { out.EquipmentTypeMap[k] = v }
    }
    if o.EmitISODateTime { out.EmitISODateTime = true }
    if o.Timezone != "" { out.Timezone = o.Timezone }
    if len(o.ServiceLevelQuals) > 0 { out.ServiceLevelQuals = append([]string{}, o.ServiceLevelQuals...) }
    if o.ServiceLevelMap != nil && len(o.ServiceLevelMap) > 0 {
        if out.ServiceLevelMap == nil { out.ServiceLevelMap = map[string]string{} }
        for k, v := range o.ServiceLevelMap { out.ServiceLevelMap[k] = v }
    }
    if len(o.AccessorialQuals) > 0 { out.AccessorialQuals = append([]string{}, o.AccessorialQuals...) }
    if o.AccessorialMap != nil && len(o.AccessorialMap) > 0 {
        if out.AccessorialMap == nil { out.AccessorialMap = map[string]string{} }
        for k, v := range o.AccessorialMap { out.AccessorialMap[k] = v }
    }
    return out
}
