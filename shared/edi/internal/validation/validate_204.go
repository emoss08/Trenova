package validation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// Validate204 performs basic structural and semantic checks for a 204 transaction
// using defaults inferred from GS08 version.
func Validate204(segs []x12.Segment) []Issue {
	prof := DefaultProfileForVersion(x12.ExtractVersion(segs))
	return Validate204WithProfile(segs, prof)
}

// Validate204WithProfile performs validation under a specific profile.
func Validate204WithProfile(segs []x12.Segment, prof Profile) []Issue {
	issues := make([]Issue, 0)

	// Envelope presence
	if len(segs) == 0 || !strings.EqualFold(segs[0].Tag, "ISA") {
		issues = append(
			issues,
			Issue{
				Severity:     Error,
				Code:         "ISA.MISSING",
				Message:      "ISA segment not found at start",
				SegmentIndex: 0,
			},
		)
	}
	gsIdx := indexOf(segs, "GS")
	geIdx := lastIndexOf(segs, "GE")
	ieaIdx := lastIndexOf(segs, "IEA")
	if gsIdx < 0 {
		issues = append(
			issues,
			Issue{Severity: Error, Code: "GS.MISSING", Message: "GS segment missing"},
		)
	}
	if geIdx < 0 {
		issues = append(
			issues,
			Issue{Severity: Error, Code: "GE.MISSING", Message: "GE segment missing"},
		)
	}
	if ieaIdx < 0 {
		issues = append(
			issues,
			Issue{Severity: Error, Code: "IEA.MISSING", Message: "IEA segment missing"},
		)
	}

	// Transaction boundaries: one ST/SE pair expected for our sample
	stIdxs := indicesOf(segs, "ST")
	seIdxs := indicesOf(segs, "SE")
	if len(stIdxs) == 0 {
		issues = append(
			issues,
			Issue{Severity: Error, Code: "ST.MISSING", Message: "ST segment missing"},
		)
		return issues
	}
	// For each ST, find matching SE by ST02=SE02
	st := segs[stIdxs[0]]
	st02 := get(st, 1, 0)
	seMatchIdx := -1
	for _, idx := range seIdxs {
		if get(segs[idx], 1, 0) == st02 {
			seMatchIdx = idx
			break
		}
	}
	if seMatchIdx < 0 {
		issues = append(
			issues,
			Issue{
				Severity:     Error,
				Code:         "SE.NOT_MATCHING",
				Message:      "No SE matches ST control",
				SegmentIndex: stIdxs[0],
				Tag:          "ST",
				ElementIndex: 2,
				Hint:         "ST02 must equal SE02",
			},
		)
		return issues
	}

	// SE01 count check
	se := segs[seMatchIdx]
	se01 := get(se, 0, 0)
	if n, err := strconv.Atoi(se01); err == nil {
		// Count segments between ST and SE inclusive
		cnt := seMatchIdx - stIdxs[0] + 1
		if n != cnt {
			sev := Error
			if !prof.EnforceSECount || prof.Strictness == Lenient {
				sev = Warning
			}
			issues = append(
				issues,
				Issue{
					Severity:     sev,
					Code:         "SE.COUNT",
					Message:      fmt.Sprintf("SE01=%d does not match actual count %d", n, cnt),
					SegmentIndex: seMatchIdx,
					Tag:          "SE",
					ElementIndex: 1,
					Hint:         "SE01 should count ST..SE inclusive",
				},
			)
		}
	}

	// B2 required
	b2Idx := indexInRange(segs, "B2", stIdxs[0]+1, seMatchIdx-1)
	if b2Idx < 0 {
		issues = append(
			issues,
			Issue{
				Severity:     Error,
				Code:         "B2.MISSING",
				Message:      "B2 segment missing in 204 header",
				SegmentIndex: stIdxs[0],
			},
		)
	}
	if b2Idx >= 0 {
		// B2-02 (SCAC) is required across common guides
		if strings.TrimSpace(get(segs[b2Idx], 1, 0)) == "" {
			issues = append(
				issues,
				Issue{
					Severity:     Error,
					Code:         "B2.SCAC.MISSING",
					Message:      "B2-02 SCAC is required",
					SegmentIndex: b2Idx,
					Tag:          "B2",
					ElementIndex: 2,
					Hint:         "Populate SCAC in B2-02",
				},
			)
		}
		// B2-03 requirement depends on profile.
		if prof.RequireB2ShipID {
			if strings.TrimSpace(get(segs[b2Idx], 2, 0)) == "" {
				issues = append(
					issues,
					Issue{
						Severity:     Error,
						Code:         "B2.SHIPID.MISSING",
						Message:      "B2-03 Shipment Identification is required",
						SegmentIndex: b2Idx,
						Tag:          "B2",
						ElementIndex: 3,
						Hint:         "Provide customer shipment ID in B2-03",
					},
				)
			}
		}
	}

	// Parties SH and ST optionally required depending on profile
	if prof.RequireN1SH {
		if indexInRange(
			segs,
			"N1",
			stIdxs[0]+1,
			seMatchIdx-1,
			func(s x12.Segment) bool { return strings.EqualFold(get(s, 0, 0), "SH") },
		) < 0 {
			issues = append(
				issues,
				Issue{Severity: Error, Code: "N1.SH.MISSING", Message: "N1*SH not found"},
			)
		}
	}
	if prof.RequireN1ST {
		if indexInRange(
			segs,
			"N1",
			stIdxs[0]+1,
			seMatchIdx-1,
			func(s x12.Segment) bool { return strings.EqualFold(get(s, 0, 0), "ST") },
		) < 0 {
			issues = append(
				issues,
				Issue{Severity: Error, Code: "N1.ST.MISSING", Message: "N1*ST not found"},
			)
		}
	}

	// Stops: at least one pickup (LD) and one delivery (UL) if required by profile
	if prof.RequirePickupAndDelivery {
		if indexInRange(
			segs,
			"S5",
			stIdxs[0]+1,
			seMatchIdx-1,
			func(s x12.Segment) bool { return strings.EqualFold(get(s, 1, 0), "LD") },
		) < 0 {
			issues = append(
				issues,
				Issue{
					Severity: Error,
					Code:     "S5.LD.MISSING",
					Message:  "No pickup S5*LD stop found",
				},
			)
		}
		if indexInRange(
			segs,
			"S5",
			stIdxs[0]+1,
			seMatchIdx-1,
			func(s x12.Segment) bool { return strings.EqualFold(get(s, 1, 0), "UL") },
		) < 0 {
			issues = append(
				issues,
				Issue{
					Severity: Error,
					Code:     "S5.UL.MISSING",
					Message:  "No delivery S5*UL stop found",
				},
			)
		}
	}

	// S5 sequence ascending check (type codes validated via schema if provided)
	lastSeq := -1
	for i := stIdxs[0] + 1; i < seMatchIdx && i < len(segs); i++ {
		s := segs[i]
		if !strings.EqualFold(s.Tag, "S5") {
			continue
		}
		// sequence check
		if seq := atoiSafe(get(s, 0, 0)); seq >= 0 {
			if lastSeq >= 0 && seq <= lastSeq {
				issues = append(
					issues,
					Issue{
						Severity:     Error,
						Code:         "S5.SEQ.ORDER",
						Message:      "S5-01 sequence must be increasing",
						SegmentIndex: i,
						Tag:          "S5",
						ElementIndex: 1,
						Hint:         "Number stops 1..n in order",
					},
				)
			}
			lastSeq = seq
		}
	}

	// DTM: basic format check within ST..SE
	for i := stIdxs[0] + 1; i < seMatchIdx && i < len(segs); i++ {
		s := segs[i]
		if !strings.EqualFold(s.Tag, "DTM") {
			continue
		}
		date := strings.TrimSpace(get(s, 1, 0))
		time := strings.TrimSpace(get(s, 2, 0))
		if date != "" && len(date) != 8 {
			issues = append(
				issues,
				Issue{
					Severity:     Error,
					Code:         "DTM.DATE.INVALID",
					Message:      "DTM-02 must be CCYYMMDD",
					SegmentIndex: i,
					Tag:          "DTM",
					ElementIndex: 2,
					Hint:         "Format date as CCYYMMDD",
				},
			)
		}
		if time != "" && !(len(time) == 4 || len(time) == 6) {
			issues = append(
				issues,
				Issue{
					Severity:     Error,
					Code:         "DTM.TIME.INVALID",
					Message:      "DTM-03 must be HHMM or HHMMSS",
					SegmentIndex: i,
					Tag:          "DTM",
					ElementIndex: 3,
					Hint:         "Format time as HHMM or HHMMSS",
				},
			)
		}
	}

	// Control numbers matching at envelope levels where present
	if gsIdx >= 0 && geIdx >= 0 {
		if strings.TrimSpace(get(segs[gsIdx], 5, 0)) != strings.TrimSpace(get(segs[geIdx], 1, 0)) {
			issues = append(
				issues,
				Issue{
					Severity:     Error,
					Code:         "GE.CTRL.MISMATCH",
					Message:      "GE02 does not match GS06",
					SegmentIndex: geIdx,
					Tag:          "GE",
					ElementIndex: 2,
					Hint:         "GE02 must equal GS06",
				},
			)
		}
		// GE01 equals ST/SE pairs count within the group
		geCnt := atoiSafe(get(segs[geIdx], 0, 0))
		pairs := countPairs(segs, "ST", "SE")
		if geCnt != pairs {
			issues = append(
				issues,
				Issue{
					Severity: Error,
					Code:     "GE.COUNT",
					Message: fmt.Sprintf(
						"GE01=%d does not match ST/SE pairs %d",
						geCnt,
						pairs,
					),
					SegmentIndex: geIdx,
					Tag:          "GE",
					ElementIndex: 1,
					Hint:         "GE01 equals number of ST/SE pairs",
				},
			)
		}
	}
	if len(segs) > 0 && ieaIdx >= 0 {
		// ISA13 vs IEA02
		if !strings.EqualFold(get(segs[0], 12, 0), get(segs[ieaIdx], 1, 0)) {
			issues = append(
				issues,
				Issue{
					Severity:     Error,
					Code:         "IEA.CTRL.MISMATCH",
					Message:      "IEA02 does not match ISA13",
					SegmentIndex: ieaIdx,
					Tag:          "IEA",
					ElementIndex: 2,
					Hint:         "IEA02 must equal ISA13",
				},
			)
		}
	}
	return issues
}

func indexOf(segs []x12.Segment, tag string) int {
	for i, s := range segs {
		if strings.EqualFold(s.Tag, tag) {
			return i
		}
	}
	return -1
}

func lastIndexOf(segs []x12.Segment, tag string) int {
	for i := len(segs) - 1; i >= 0; i-- {
		if strings.EqualFold(segs[i].Tag, tag) {
			return i
		}
	}
	return -1
}

func indicesOf(segs []x12.Segment, tag string) []int {
	res := []int{}
	for i, s := range segs {
		if strings.EqualFold(s.Tag, tag) {
			res = append(res, i)
		}
	}
	return res
}

func indexInRange(
	segs []x12.Segment,
	tag string,
	start, end int,
	pred ...func(x12.Segment) bool,
) int {
	for i := start; i <= end && i < len(segs); i++ {
		if !strings.EqualFold(segs[i].Tag, tag) {
			continue
		}
		if len(pred) == 0 || pred[0](segs[i]) {
			return i
		}
	}
	return -1
}

func countPairs(segs []x12.Segment, openTag, closeTag string) int {
	opens := 0
	closes := 0
	for _, s := range segs {
		if strings.EqualFold(s.Tag, openTag) {
			opens++
		} else if strings.EqualFold(s.Tag, closeTag) {
			closes++
		}
	}
	if opens < closes {
		return opens
	}
	return closes
}

func atoiSafe(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
