package x12

import "strings"

// ExtractVersion returns the X12 version/release (e.g., "004010", "006010")
// as communicated in GS08. Returns empty string if not found.
func ExtractVersion(segs []Segment) string {
	for _, s := range segs {
		if strings.EqualFold(s.Tag, "GS") {
			if len(s.Elements) >= 8 {
				return s.Elements[7][0]
			}
		}
	}
	return ""
}
