package ack

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// Generate997 builds a simple 997 functional acknowledgment for a single interchange/group
// using detected delimiters and the incoming envelope values. It collapses results to group-level
// AK1/AK9 with A (accepted) or E (accepted with errors).
func Generate997(segs []x12.Segment, delims x12.Delimiters, issues []validation.Issue) string {
	elem := string([]byte{delims.Element})
	seg := string([]byte{delims.Segment})
	// comp not used in this minimal 997
	// Extract ISA/GS envelope values
	isa := x12.FindSegments(segs, "ISA")
	// gs := x12.FindSegments(segs, "GS")
	isaSender, isaReceiver := "SENDER", "RECEIVER"
	if len(isa) > 0 {
		if len(isa[0].Elements) >= 6 {
			isaSender = isa[0].Elements[5][0]
		}
		if len(isa[0].Elements) >= 8 {
			isaReceiver = isa[0].Elements[7][0]
		}
	}
	// gsSender, gsReceiver not currently used
	// if len(gs) > 0 {
	//     if len(gs[0].Elements) >= 2 { gsSender = gs[0].Elements[1][0] }
	//     if len(gs[0].Elements) >= 3 { gsReceiver = gs[0].Elements[2][0] }
	// }
	now := time.Now().UTC()
	date := now.Format("060102")
	timeHM := now.Format("1504")
	ctrl := "000000001"
	gsCtrl := "1"
	stCtrl := "0001"
	// Functional group (SM) acknowledgment (FA)
	// ISA (swap sender/receiver for ack)
	out := ""
	out += fmt.Sprintf(
		"ISA%[1]s00%[1]s          %[1]s00%[1]s          %[1]sZZ%[1]s%-15s%[1]sZZ%[1]s%-15s%[1]s%[4]s%[1]s%[5]s%[1]sU%[1]s00401%[1]s%[6]s%[1]s0%[1]sP%[1]s>%[7]s",
		elem,
		isaReceiver,
		isaSender,
		date,
		timeHM,
		ctrl,
		seg,
	)
	out += fmt.Sprintf(
		"GS%[1]sFA%[1]s%[2]s%[1]s%[3]s%[1]s%[4]s%[1]s%[5]s%[1]sX%[1]s004010%[7]s",
		elem,
		isaSender,
		isaReceiver,
		now.Format("20060102"),
		timeHM,
		gsCtrl,
		seg,
	)
	out += fmt.Sprintf("ST%[1]s997%[1]s%[2]s%[3]s", elem, stCtrl, seg)
	// AK1: Functional group response (SM)
	out += fmt.Sprintf("AK1%[1]sSM%[1]s1%[2]s", elem, seg)
	// AK9: Group response code
	code := "A"
	if hasErrors(issues) {
		code = "E"
	}
	// AK9 with counts: here we collapse to 1/1/1 for simplicity
	out += fmt.Sprintf("AK9%[1]s%[2]s%[1]s1%[1]s1%[1]s1%[3]s", elem, code, seg)
	out += fmt.Sprintf("SE%[1]s4%[1]s%[2]s%[3]s", elem, stCtrl, seg)
	out += fmt.Sprintf("GE%[1]s1%[1]s%[2]s%[3]s", elem, gsCtrl, seg)
	out += fmt.Sprintf("IEA%[1]s1%[1]s%[2]s%[3]s", elem, ctrl, seg)
	return out
}

func hasErrors(issues []validation.Issue) bool {
	for _, is := range issues {
		if is.Severity == validation.Error {
			return true
		}
	}
	return false
}
