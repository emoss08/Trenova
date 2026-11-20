package oracle

import (
	"fmt"
	"strconv"
	"strings"
)

type envelopeReader struct {
	ic         *Interchange
	group      *Group
	report     *Report
	stray      bool
	groupSeen  map[string]bool
	reportSeen map[string]bool
}

func readFile(raw []byte) (*Judgement, error) {
	pos := skipBlank(raw, skipBOM(raw), nil)
	if _, err := readISA(raw, pos); err != nil {
		return nil, err
	}
	j := &Judgement{}
	seen := make(map[string]bool)
	for {
		pos = skipBlank(raw, pos, nil)
		if pos >= len(raw) {
			return j, nil
		}
		if len(raw)-pos < 3 || string(raw[pos:pos+3]) != "ISA" {
			j.Findings = append(j.Findings, controlStructure(
				"", pos, "data after the last IEA is not an interchange",
			))
			return j, nil
		}
		ic, next, stop := readInterchange(raw, pos)
		if seen[ic.ControlNumber] {
			f := errorAt(
				LevelInterchange,
				codeDuplicateInterchange,
				"ISA13 unique in the file",
				fmt.Sprintf("interchange control number %s is used twice", ic.ControlNumber),
			)
			f.SegmentID, f.Offset, f.ElementPos, f.Value = "ISA", pos, 13, ic.ControlNumber
			ic.Findings = append(ic.Findings, f)
		}
		seen[ic.ControlNumber] = true
		j.Interchanges = append(j.Interchanges, ic)
		if stop || next <= pos {
			return j, nil
		}
		pos = next
	}
}

func skipBOM(raw []byte) int {
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		return 3
	}
	return 0
}

func controlStructure(segmentID string, offset int, message string) Finding {
	f := errorAt(LevelInterchange, codeControlStructure, "envelope nesting", message)
	f.SegmentID = segmentID
	f.Offset = offset
	return f
}

func readInterchange(raw []byte, start int) (*Interchange, int, bool) {
	h, err := readISA(raw, start)
	if err != nil {
		ic := &Interchange{Offset: start}
		f := errorAt(LevelInterchange, codeTrailerMissing, "ISA complete", err.Error())
		f.SegmentID, f.Offset = "ISA", start
		ic.Findings = append(ic.Findings, f)
		return ic, len(raw), true
	}
	ic := &Interchange{
		ControlNumber:     h.controlNumber(),
		SenderQualifier:   strings.TrimSpace(h.fields[4]),
		SenderID:          strings.TrimSpace(h.fields[5]),
		ReceiverQualifier: strings.TrimSpace(h.fields[6]),
		ReceiverID:        strings.TrimSpace(h.fields[7]),
		Version:           strings.TrimSpace(h.fields[11]),
		Date:              strings.TrimSpace(h.fields[8]),
		Time:              strings.TrimSpace(h.fields[9]),
		AckRequested:      strings.TrimSpace(h.fields[13]) == "1",
		Offset:            start,
		Header:            newSegment(string(raw[start:h.end-1]), start, h.seps),
	}
	for _, f := range h.findings {
		f.Offset = start
		ic.Findings = append(ic.Findings, f)
	}
	if h.fatal {
		return ic, len(raw), true
	}
	r := &envelopeReader{ic: ic, groupSeen: make(map[string]bool)}
	lx := &lexer{raw: raw, pos: h.end, seps: h.seps}
	for {
		seg, ok := lx.next()
		if !ok {
			r.closeAll(len(raw))
			f := errorAt(
				LevelInterchange,
				codeTrailerMissing,
				"IEA closes the interchange",
				"the file ended before the IEA",
			)
			f.SegmentID, f.Offset = "IEA", len(raw)
			ic.Findings = append(ic.Findings, f)
			return ic, len(raw), true
		}
		switch seg.ID {
		case "ISA":
			r.closeAll(seg.Offset)
			f := errorAt(
				LevelInterchange,
				codeTrailerMissing,
				"IEA closes the interchange",
				"another ISA started before this interchange's IEA",
			)
			f.SegmentID, f.Offset = "IEA", seg.Offset
			ic.Findings = append(ic.Findings, f)
			return ic, seg.Offset, false
		case "IEA":
			r.closeAll(seg.Offset)
			ic.Trailer = seg
			r.iea(&seg)
			return ic, lx.pos, false
		default:
			r.segment(seg)
		}
	}
}

func (r *envelopeReader) segment(seg Segment) {
	switch seg.ID {
	case "GS":
		r.closeAll(seg.Offset)
		r.gs(&seg)
	case "GE":
		r.closeReport(seg.Offset)
		r.ge(&seg)
	case "ST":
		r.closeReport(seg.Offset)
		r.st(seg)
	case "SE":
		r.se(seg)
	default:
		r.body(seg)
	}
}

func (r *envelopeReader) body(seg Segment) {
	if r.report != nil {
		seg.Index = len(r.report.Segments) + 1
		r.report.Segments = append(r.report.Segments, seg)
		return
	}
	if seg.ID == "TA1" && r.group == nil {
		r.ic.TA1 = append(r.ic.TA1, seg)
		return
	}
	if r.stray {
		return
	}
	r.ic.Findings = append(r.ic.Findings, controlStructure(
		seg.ID, seg.Offset, fmt.Sprintf("%s is outside any transaction set", seg.ID),
	))
}

func (r *envelopeReader) closeAll(offset int) {
	r.closeReport(offset)
	if r.group == nil {
		return
	}
	f := errorAt(LevelGroup, codeGroupTrailerMissing, "GE closes the group", "the GE is missing")
	f.SegmentID, f.Offset = "GE", offset
	r.group.Findings = append(r.group.Findings, f)
	r.group = nil
}

func (r *envelopeReader) closeReport(offset int) {
	r.stray = false
	if r.report == nil {
		return
	}
	f := errorAt(
		LevelTransaction,
		codeSetTrailerMissing,
		"SE closes the transaction set",
		"the SE is missing",
	)
	f.SegmentID, f.Offset = "SE", offset
	r.report.add(f)
	r.report = nil
}

func (r *envelopeReader) gs(seg *Segment) {
	g := &Group{
		FunctionalID:  seg.Element(1),
		SenderCode:    seg.Element(2),
		ReceiverCode:  seg.Element(3),
		ControlNumber: seg.Element(6),
		Version:       seg.Element(8),
		Header:        *seg,
	}
	r.ic.Groups = append(r.ic.Groups, g)
	r.group = g
	r.reportSeen = make(map[string]bool)
	if !digitsWithin(g.ControlNumber, 1, 9) {
		r.groupFinding(seg, codeGroupControlSyntax, 6, "GS06 one to nine digits",
			fmt.Sprintf("group control number %q is not one to nine digits", g.ControlNumber))
	}
	if r.groupSeen[g.ControlNumber] {
		r.groupFinding(seg, codeDuplicateGroupControl, 6, "GS06 unique in the interchange",
			fmt.Sprintf("group control number %s is used twice", g.ControlNumber))
	}
	r.groupSeen[g.ControlNumber] = true
}

func (r *envelopeReader) groupFinding(seg *Segment, code string, pos int, rule, msg string) {
	f := errorAt(LevelGroup, code, rule, msg)
	f.SegmentID, f.Offset, f.ElementPos, f.Value = seg.ID, seg.Offset, pos, seg.Element(pos)
	r.group.Findings = append(r.group.Findings, f)
}

func (r *envelopeReader) ge(seg *Segment) {
	if r.group == nil {
		r.ic.Findings = append(r.ic.Findings, controlStructure(
			"GE", seg.Offset, "a GE without a GS",
		))
		return
	}
	r.group.Trailer = *seg
	if seg.Element(2) != r.group.ControlNumber {
		r.groupFinding(seg, codeGroupControlMismatch, 2, "GE02 equals GS06",
			fmt.Sprintf("GE02 %q does not match GS06 %q", seg.Element(2), r.group.ControlNumber))
	}
	if n := strconv.Itoa(len(r.group.Reports)); seg.Element(1) != n {
		r.groupFinding(seg, codeTransactionCountWrong, 1, "GE01 counts the transaction sets",
			fmt.Sprintf("GE01 is %q but the group holds %s transaction sets", seg.Element(1), n))
	}
	r.group = nil
}

func (r *envelopeReader) st(seg Segment) {
	if r.group == nil {
		r.ic.Findings = append(r.ic.Findings, controlStructure(
			"ST", seg.Offset, "an ST outside any functional group",
		))
		r.stray = true
		return
	}
	rep := &Report{
		Set:                      seg.Element(1),
		ControlNumber:            seg.Element(2),
		Version:                  r.group.Version,
		GroupControlNumber:       r.group.ControlNumber,
		InterchangeControlNumber: r.ic.ControlNumber,
	}
	seg.Index = 1
	rep.Segments = append(rep.Segments, seg)
	r.group.Reports = append(r.group.Reports, rep)
	r.report = rep
	if len(rep.Set) != 3 || !allDigits(rep.Set) {
		r.reportFinding(&seg, codeSetIdentifierInvalid, 1, "ST01 three digits",
			fmt.Sprintf("ST01 %q is not a transaction set identifier", rep.Set))
	}
	if n := len(rep.ControlNumber); n < 4 || n > 9 {
		r.reportFinding(&seg, codeSetControlInvalid, 2, "ST02 four to nine characters",
			fmt.Sprintf("ST02 %q is not four to nine characters", rep.ControlNumber))
	}
	if r.reportSeen[rep.ControlNumber] {
		r.reportFinding(&seg, codeDuplicateSetControl, 2, "ST02 unique in the group",
			fmt.Sprintf("transaction control number %s is used twice", rep.ControlNumber))
	}
	r.reportSeen[rep.ControlNumber] = true
}

func (r *envelopeReader) reportFinding(seg *Segment, code string, pos int, rule, msg string) {
	f := errorAt(LevelTransaction, code, rule, msg)
	f.SegmentID, f.SegmentIndex, f.Offset = seg.ID, seg.Index, seg.Offset
	f.ElementPos, f.Value = pos, seg.Element(pos)
	r.report.add(f)
}

func (r *envelopeReader) se(seg Segment) {
	if r.report == nil {
		if !r.stray {
			r.ic.Findings = append(r.ic.Findings, controlStructure(
				"SE", seg.Offset, "an SE without an ST",
			))
		}
		r.stray = false
		return
	}
	seg.Index = len(r.report.Segments) + 1
	r.report.Segments = append(r.report.Segments, seg)
	if seg.Element(2) != r.report.ControlNumber {
		r.reportFinding(&seg, codeSetControlMismatch, 2, "SE02 equals ST02",
			fmt.Sprintf("SE02 %q does not match ST02 %q", seg.Element(2), r.report.ControlNumber))
	}
	if n := strconv.Itoa(seg.Index); seg.Element(1) != n {
		r.reportFinding(&seg, codeSegmentCountWrong, 1, "SE01 counts ST through SE",
			fmt.Sprintf("SE01 is %q but the transaction set has %s segments", seg.Element(1), n))
	}
	r.report = nil
}

func (r *envelopeReader) iea(seg *Segment) {
	add := func(code string, pos int, rule, msg string) {
		f := errorAt(LevelInterchange, code, rule, msg)
		f.SegmentID, f.Offset, f.ElementPos, f.Value = "IEA", seg.Offset, pos, seg.Element(pos)
		r.ic.Findings = append(r.ic.Findings, f)
	}
	if strings.TrimSpace(seg.Element(2)) != r.ic.ControlNumber {
		add(codeControlNumberMismatch, 2, "IEA02 equals ISA13",
			fmt.Sprintf("IEA02 %q does not match ISA13 %q", seg.Element(2), r.ic.ControlNumber))
	}
	if n := strconv.Itoa(len(r.ic.Groups)); seg.Element(1) != n {
		add(codeGroupCountMismatch, 1, "IEA01 counts the groups",
			fmt.Sprintf("IEA01 is %q but the interchange holds %s groups", seg.Element(1), n))
	}
}
