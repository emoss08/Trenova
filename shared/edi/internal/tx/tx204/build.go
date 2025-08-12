package tx204

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// BuildFromSegments converts raw segments into a minimal 204 LoadTender model.
// MVP: captures B2/B2A/L11 header refs, header N1 parties, basic S5 stops + DTM, and N7 equipment.
func BuildFromSegments(segs []x12.Segment) LoadTender {
	lt := LoadTender{
		Header:  Header{References: map[string][]string{}},
		Parties: map[string]Party{},
	}

	inHeader := true
	var curStop *Stop
	for _, s := range segs {
		switch strings.ToUpper(s.Tag) {
		case "ST":
			if val := get(s, 1, 0); val != "" {
				lt.Control.STControl = val
			}
		case "B2":
			// B2-02 = SCAC, B2-03 = Shipment ID
			lt.Header.CarrierSCAC = get(s, 1, 0)
			lt.Header.ShipmentID = get(s, 2, 0)
		case "B2A":
			lt.Header.ActionCode = get(s, 0, 0)
		case "L11":
			v := get(s, 0, 0)
			q := get(s, 1, 0)
			if q == "" {
				q = "L11"
			}
			lt.Header.References[q] = append(lt.Header.References[q], v)
		case "N1":
			p := Party{
				Code:       get(s, 0, 0),
				Name:       get(s, 1, 0),
				IDCodeQual: get(s, 2, 0),
				IDCode:     get(s, 3, 0),
			}
			// lookahead location details via N3/N4 following this N1
			// only attach for header parties; for stops we'll attach to stop's Location.
			if inHeader {
				lt.Parties[p.Code] = p
			} else if curStop != nil {
				curStop.Location = p
			}
		case "N3":
			if inHeader {
				// attach to most recent party by code if available
				// this is a simplification for MVP; later we'll manage N1 loops explicitly
				// try common party precedence: ST, SH, BT
				for _, code := range []string{"ST", "SH", "BT"} {
					if pp, ok := lt.Parties[code]; ok {
						pp.Address1 = get(s, 0, 0)
						pp.Address2 = get(s, 1, 0)
						lt.Parties[code] = pp
						break
					}
				}
			} else if curStop != nil {
				curStop.Location.Address1 = get(s, 0, 0)
				curStop.Location.Address2 = get(s, 1, 0)
			}
		case "N4":
			if inHeader {
				for _, code := range []string{"ST", "SH", "BT"} {
					if pp, ok := lt.Parties[code]; ok {
						pp.City = get(s, 0, 0)
						pp.State = get(s, 1, 0)
						pp.PostalCode = get(s, 2, 0)
						pp.Country = get(s, 3, 0)
						lt.Parties[code] = pp
						break
					}
				}
			} else if curStop != nil {
				curStop.Location.City = get(s, 0, 0)
				curStop.Location.State = get(s, 1, 0)
				curStop.Location.PostalCode = get(s, 2, 0)
				curStop.Location.Country = get(s, 3, 0)
			}
		case "S5":
			inHeader = false
			st := Stop{}
			if seq := get(s, 0, 0); seq != "" {
				if n, err := strconv.Atoi(seq); err == nil {
					st.Sequence = n
				}
			}
			st.Type = get(s, 1, 0)
			lt.Stops = append(lt.Stops, st)
			curStop = &lt.Stops[len(lt.Stops)-1]
		case "DTM":
			ap := Appt{Qualifier: get(s, 0, 0), Date: get(s, 1, 0), Time: get(s, 2, 0)}
			if curStop != nil {
				curStop.Appointments = append(curStop.Appointments, ap)
			}
		case "N7":
			// N7-01 Equipment Initial (or type), N7-02 Equipment Number, N7-03 Type Code, N7-04 Description
			lt.Equipment.Type = get(s, 0, 0)
			lt.Equipment.ID = get(s, 1, 0)
			lt.Equipment.TypeCode = get(s, 2, 0)
			lt.Equipment.Description = get(s, 3, 0)
			// Dimensional data appears in various positions across guides; capture common spots when present
			// N7-08 length, N7-09 width, N7-10 height, N7-11 unit (varies by guide)
			if v := get(s, 7, 0); v != "" {
				lt.Equipment.Length = v
			}
			if v := get(s, 8, 0); v != "" {
				lt.Equipment.Width = v
			}
			if v := get(s, 9, 0); v != "" {
				lt.Equipment.Height = v
			}
			if v := get(s, 10, 0); v != "" {
				lt.Equipment.DimUnit = v
			}
		case "NTE":
			if inHeader {
				lt.Notes = append(lt.Notes, get(s, 1, 0))
			} else if curStop != nil {
				curStop.Notes = append(curStop.Notes, get(s, 1, 0))
			}
		case "AT8":
			// AT8-02 weight unit, AT8-03 weight, AT8-04 lading qty (pieces)
			if v := get(s, 1, 0); v != "" {
				lt.Totals.WeightUnit = v
			}
			if v := get(s, 2, 0); v != "" {
				lt.Totals.Weight = v
			}
			if v := get(s, 3, 0); v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					lt.Totals.Pieces = n
				}
			}
		case "L3":
			// L3-01 total weight, last element sometimes weight unit
			if v := get(s, 0, 0); v != "" {
				lt.Totals.Weight = v
			}
			// Try last element for unit if present
			if len(s.Elements) > 1 {
				last := s.Elements[len(s.Elements)-1]
				if len(last) > 0 && last[0] != "" {
					lt.Totals.WeightUnit = last[0]
				}
			}
		case "L5":
			// L5-02 description, L5-03 code
			c := Commodity{Description: get(s, 1, 0), Code: get(s, 2, 0)}
			if c.Description != "" || c.Code != "" {
				lt.Commodities = append(lt.Commodities, c)
			}
		}
	}
	return lt
}

// get safely returns element i, component j (0-based), or "".
func get(s x12.Segment, i, j int) string {
	if i < 0 || i >= len(s.Elements) {
		return ""
	}
	if j < 0 || j >= len(s.Elements[i]) {
		return ""
	}
	return s.Elements[i][j]
}
