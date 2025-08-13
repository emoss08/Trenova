package mapper

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/dto"
	tx204 "github.com/emoss08/trenova/shared/edi/internal/tx/tx204"
)

// ToShipment maps a tx204.LoadTender into a dto.Shipment with basic normalization.
func ToShipment(lt tx204.LoadTender) dto.Shipment {
	return ToShipmentWithOptions(lt, DefaultOptions())
}

// ToShipmentWithRefMap maps a tx204.LoadTender into a dto.Shipment using a partner-provided
// reference mapping (DTO key -> list of L11 qualifiers). If map is nil/empty, defaults are used.
func ToShipmentWithRefMap(lt tx204.LoadTender, refMap map[string][]string) dto.Shipment {
	opts := DefaultOptions()
	if refMap != nil {
		opts.RefMap = refMap
	}
	return ToShipmentWithOptions(lt, opts)
}

// ToShipmentWithOptions maps a tx204.LoadTender into a dto.Shipment using flexible options.
func ToShipmentWithOptions(lt tx204.LoadTender, opts Options) dto.Shipment {
	out := dto.Shipment{
		CarrierSCAC: lt.Header.CarrierSCAC,
		ActionCode:  lt.Header.ActionCode,
		References:  map[string]string{},
		Equipment:   dto.Equipment{Type: lt.Equipment.Type, ID: lt.Equipment.ID},
		Notes:       append([]string{}, lt.Notes...),
	}

	refMap := opts.RefMap
	if refMap == nil {
		refMap = DefaultRefMap()
	}

	for key, quals := range refMap {
		for _, q := range quals {
			if vals, ok := lt.Header.References[q]; ok && len(vals) > 0 {
				if _, exists := out.References[key]; !exists {
					out.References[key] = vals[0]
				}
			}
		}
	}

	out.ShipmentID = pickShipmentID(lt, opts)

	if strings.TrimSpace(out.CarrierSCAC) == "" &&
		strings.TrimSpace(opts.CarrierSCACFallback) != "" {
		out.CarrierSCAC = opts.CarrierSCACFallback
	}

	roles := opts.PartyRoles
	if roles == nil {
		roles = DefaultPartyRoles()
	}

	pick := func(codes []string) *dto.Party {
		for _, code := range codes {
			if p, ok := lt.Parties[code]; ok {
				pp := partyFrom(p)
				return &pp
			}
		}
		return nil
	}
	if v, ok := roles["bill_to"]; ok {
		out.BillTo = pick(v)
	}
	if v, ok := roles["shipper"]; ok {
		out.Shipper = pick(v)
	}
	if v, ok := roles["consignee"]; ok {
		out.Consignee = pick(v)
	}

	out.Stops = make([]dto.Stop, 0, len(lt.Stops))
	for _, s := range lt.Stops {
		stype := mapS5TypeWith(opts.StopTypeMap, s.Type)
		ds := dto.Stop{
			Sequence: s.Sequence,
			Type:     stype,
			Location: partyFrom(s.Location),
			Notes:    append([]string{}, s.Notes...),
		}

		if len(s.Appointments) > 0 {
			ds.Appointments = make([]dto.Appt, 0, len(s.Appointments))
			for _, a := range s.Appointments {
				ap := dto.Appt{Qualifier: a.Qualifier, Date: a.Date, Time: a.Time}
				if opts.EmitISODateTime {
					if iso := normalizeDT(a.Date, a.Time, opts.Timezone); iso != "" {
						ap.DateTime = iso
					}
				}
				ds.Appointments = append(ds.Appointments, ap)
			}
		}
		out.Stops = append(out.Stops, ds)
	}

	if opts.EquipmentTypeMap != nil {
		if norm, ok := opts.EquipmentTypeMap[strings.ToUpper(strings.TrimSpace(out.Equipment.Type))]; ok &&
			norm != "" {
			out.Equipment.Type = norm
		}
	}

	if opts.IncludeRawL11 {
		out.ReferencesRaw = make(map[string][]string, len(lt.Header.References))
		if len(opts.RawL11Filter) == 0 {
			for q, vals := range lt.Header.References {
				out.ReferencesRaw[q] = append([]string{}, vals...)
			}
		} else {
			// Build a set of allowed qualifiers (case-sensitive to match build model usage)
			allow := map[string]struct{}{}
			for _, q := range opts.RawL11Filter {
				allow[q] = struct{}{}
			}
			for q, vals := range lt.Header.References {
				if _, ok := allow[q]; ok {
					out.ReferencesRaw[q] = append([]string{}, vals...)
				}
			}
		}
	}

	out.Totals = dto.Totals{
		Weight:     lt.Totals.Weight,
		WeightUnit: lt.Totals.WeightUnit,
		Pieces:     lt.Totals.Pieces,
	}

	if len(lt.Commodities) > 0 {
		out.Goods = make([]dto.Commodity, 0, len(lt.Commodities))
		for _, c := range lt.Commodities {
			out.Goods = append(out.Goods, dto.Commodity{Description: c.Description, Code: c.Code})
		}
	}

	for _, q := range opts.ServiceLevelQuals {
		if vals, ok := lt.Header.References[q]; ok && len(vals) > 0 {
			out.ServiceLevel = vals[0]
			if name, ok := opts.ServiceLevelMap[out.ServiceLevel]; ok && name != "" {
				out.ServiceLevel = name
			}
			break
		}
	}

	if len(opts.AccessorialQuals) > 0 {
		accs := []dto.Accessorial{}
		for _, q := range opts.AccessorialQuals {
			if vals, ok := lt.Header.References[q]; ok && len(vals) > 0 {
				for _, v := range vals {
					a := dto.Accessorial{Code: v}
					if name, ok := opts.AccessorialMap[v]; ok && name != "" {
						a.Name = name
					}
					accs = append(accs, a)
				}
			}
		}
		if len(accs) > 0 {
			out.Accessorials = accs
		}
	}

	return out
}

func partyFrom(p tx204.Party) dto.Party {
	return dto.Party{
		Code:       p.Code,
		Name:       p.Name,
		IDCodeQual: p.IDCodeQual,
		IDCode:     p.IDCode,
		Address1:   p.Address1,
		Address2:   p.Address2,
		City:       p.City,
		State:      strings.ToUpper(p.State),
		PostalCode: p.PostalCode,
		Country:    strings.ToUpper(p.Country),
		Contacts:   append([]string{}, p.Contacts...),
	}
}

func mapS5Type(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "LD", "CL":
		return "pickup"
	case "UL", "CU":
		return "delivery"
	default:
		return "other"
	}
}

func mapS5TypeWith(m map[string]string, code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	if m != nil {
		if v, ok := m[c]; ok && v != "" {
			return v
		}
	}
	return mapS5Type(c)
}

func pickShipmentID(lt tx204.LoadTender, opts Options) string {
	mode := strings.ToLower(strings.TrimSpace(opts.ShipmentIDMode))
	if mode == "" {
		mode = "ref_first"
	}
	pickFromRefs := func() (string, bool) {
		for _, q := range opts.ShipmentIDQuals {
			if vals, ok := lt.Header.References[q]; ok && len(vals) > 0 &&
				strings.TrimSpace(vals[0]) != "" {
				return vals[0], true
			}
		}
		return "", false
	}

	switch mode {
	case "ref_only":
		if v, ok := pickFromRefs(); ok {
			return v
		}
		return ""
	case "b2_only":
		return strings.TrimSpace(lt.Header.ShipmentID)
	case "b2_first":
		if v := strings.TrimSpace(lt.Header.ShipmentID); v != "" {
			return v
		}
		if v, ok := pickFromRefs(); ok {
			return v
		}
		return ""
	case "ref_first":
		fallthrough
	default:
		if v, ok := pickFromRefs(); ok {
			return v
		}
		return strings.TrimSpace(lt.Header.ShipmentID)
	}
}

func normalizeDT(date, tim, tz string) string {
	if len(date) != 8 || (len(tim) != 4 && len(tim) != 6) {
		return ""
	}
	layout := "200601021504"
	value := date + tim
	if len(tim) == 6 {
		layout = "20060102150405"
	}
	loc := time.UTC
	if tz != "" && tz != "UTC" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	t, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// DefaultRefMap returns a conventional mapping of DTO reference keys to common L11 qualifiers.
func DefaultRefMap() map[string][]string {
	return map[string][]string{
		"customer_po":    {"PO"},
		"bill_of_lading": {"BM"},
		"shipment_ref":   {"SI", "CR"},
	}
}

// MergeRefMaps overlays override onto base by key, replacing conflicting keys.
func MergeRefMaps(base, override map[string][]string) map[string][]string {
	if base == nil && override == nil {
		return nil
	}
	out := make(map[string][]string, len(base)+len(override))
	for k, v := range base {
		vv := make([]string, len(v))
		copy(vv, v)
		out[k] = vv
	}
	for k, v := range override {
		vv := make([]string, len(v))
		copy(vv, v)
		out[k] = vv
	}
	return out
}
