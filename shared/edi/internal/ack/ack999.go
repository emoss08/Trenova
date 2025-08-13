package ack

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// Generate999 builds a simple 999 Implementation Acknowledgment for 005010+.
// txAccepted is a parallel slice to txs indicating whether each transaction was accepted (true) or had errors (false).
func Generate999(
	segs []x12.Segment,
	delims x12.Delimiters,
	txs []x12.TxBlock,
	txAccepted []bool,
) string {
	elem := string([]byte{delims.Element})
	seg := string([]byte{delims.Segment})
	// Extract ISA sender/receiver and swap for ACK
	isa := x12.FindSegments(segs, "ISA")
	isaSender, isaReceiver := "SENDER", "RECEIVER"
	if len(isa) > 0 {
		if len(isa[0].Elements) >= 6 {
			isaSender = isa[0].Elements[5][0]
		}
		if len(isa[0].Elements) >= 8 {
			isaReceiver = isa[0].Elements[7][0]
		}
	}
	now := time.Now().UTC()
	date := now.Format("060102")
	timeHM := now.Format("1504")
	ctrl := "000000002"
	gsCtrl := "2"
	stCtrl := "0002"

	out := ""
	out += fmt.Sprintf(
		"ISA%[1]s00%[1]s          %[1]s00%[1]s          %[1]sZZ%[1]s%-15s%[1]sZZ%[1]s%-15s%[1]s%[4]s%[1]s%[5]s%[1]sU%[1]s00501%[1]s%[6]s%[1]s0%[1]sP%[1]s>%[7]s",
		elem,
		isaReceiver,
		isaSender,
		date,
		timeHM,
		ctrl,
		seg,
	)
	out += fmt.Sprintf(
		"GS%[1]sFA%[1]s%[2]s%[1]s%[3]s%[1]s%[4]s%[1]s%[5]s%[1]sX%[1]s005010%[7]s",
		elem,
		isaSender,
		isaReceiver,
		now.Format("20060102"),
		timeHM,
		gsCtrl,
		seg,
	)
	out += fmt.Sprintf("ST%[1]s999%[1]s%[2]s%[3]s", elem, stCtrl, seg)
	// AK1 for Functional Group SM
	out += fmt.Sprintf("AK1%[1]sSM%[1]s1%[2]s", elem, seg)
	// Per-transaction AK2/IK5
	acceptedCount := 0
	for i, tx := range txs {
		// Only include 204s
		if tx.SetID != "204" {
			continue
		}
		out += fmt.Sprintf("AK2%[1]s%[2]s%[1]s%[3]s%[4]s", elem, tx.SetID, tx.Control, seg)
		code := "A"
		if i < len(txAccepted) && !txAccepted[i] {
			code = "E"
		} else {
			acceptedCount++
		}
		out += fmt.Sprintf("IK5%[1]s%[2]s%[3]s", elem, code, seg)
	}
	// AK9 summary for group
	total := 0
	for _, tx := range txs {
		if tx.SetID == "204" {
			total++
		}
	}
	groupCode := "A"
	if acceptedCount != total {
		groupCode = "E"
	}
	out += fmt.Sprintf(
		"AK9%[1]s%[2]s%[1]s%[3]d%[1]s%[3]d%[1]s%[3]d%[4]s",
		elem,
		groupCode,
		total,
		seg,
	)
	// naive segment count for ST/SE: ST, AK1, (AK2+IK5)*total, AK9, SE
	seCount := 4 + 2*total
	out += fmt.Sprintf("SE%[1]s%[2]d%[1]s%[3]s%[4]s", elem, seCount, stCtrl, seg)
	out += fmt.Sprintf("GE%[1]s1%[1]s%[2]s%[3]s", elem, gsCtrl, seg)
	out += fmt.Sprintf("IEA%[1]s1%[1]s%[2]s%[3]s", elem, ctrl, seg)
	return out
}
