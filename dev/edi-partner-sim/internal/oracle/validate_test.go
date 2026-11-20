package oracle

import (
	"strings"
	"testing"
)

const (
	b1  = "B1*TRNV*SIM000042*20261119*A"
	lx  = "LX*1"
	at7 = "AT7*X3*NS***20261119*1415*LT"
	b10 = "B10*PRO1*SIM000042*TRNV"
	b3  = "B3**INV1*SIM1*PP**20261119*125000****TRNV"
	l3  = "L3*42000*G***125000"
)

func TestTransactionFindings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		group string
		set   string
		body  []string
		level Level
		code  string
		seg   string
		pos   int
		comp  int
		rule  string
		loop  string
	}{
		{
			name: "unrecognized segment", group: "GF", set: "990",
			body:  []string{b1, "Z!*X"},
			level: LevelSegment, code: "1", seg: "Z!",
		},
		{
			name: "not used segment", group: "GF", set: "990",
			body:  []string{b1, "NTE**HELLO"},
			level: LevelSegment, code: "2", seg: "NTE", rule: "NTE usage N",
		},
		{
			name: "mandatory segment missing", group: "GF", set: "990",
			body:  []string{"N9*CN*PRO1"},
			level: LevelSegment, code: "3", seg: "B1", rule: "B1 usage M",
		},
		{
			name: "mandatory loop missing", group: "QM", set: "214",
			body:  []string{b10},
			level: LevelSegment, code: "3", seg: "LX", rule: "loop 0200 usage M", loop: "0200",
		},
		{
			name: "loop over max", group: "IM", set: "210",
			body: []string{
				b3, "C3*USD", "N1*SH*A", "N1*CN*B", "N1*BT*C", "N1*SH*D",
				lx, "L1*1*2.50*PM*100000", l3,
			},
			level: LevelSegment, code: "4", seg: "N1", rule: "loop 0100 max 3", loop: "0100",
		},
		{
			name: "segment over max use", group: "GF", set: "990",
			body:  []string{b1, "K1*A", "K1*B", "K1*C"},
			level: LevelSegment, code: "5", seg: "K1", rule: "K1 max 2",
		},
		{
			name: "segment not in set", group: "GF", set: "990",
			body:  []string{b1, "G62*10*20261119"},
			level: LevelSegment, code: "6", seg: "G62",
		},
		{
			name: "segment out of sequence", group: "GF", set: "990",
			body:  []string{"N9*CN*PRO1", b1},
			level: LevelSegment, code: "7", seg: "B1",
		},
		{
			name: "MS2 in a 214 header is out of sequence", group: "QM", set: "214",
			body:  []string{b10, "MS2*TRNV*T1", lx, at7},
			level: LevelSegment, code: "7", seg: "MS2",
		},
		{
			name: "mandatory element missing", group: "GF", set: "990",
			body:  []string{"B1*TRNV**20261119*A"},
			level: LevelElement, code: "1", seg: "B1", pos: 2, rule: "B1-02 usage M",
		},
		{
			name: "mandatory component missing", group: "GF", set: "990",
			body:  []string{b1, "N9*CN*PRO1*****PO"},
			level: LevelElement, code: "1", seg: "N9", pos: 7, comp: 2, rule: "N9-07-2 usage M",
		},
		{
			name: "required syntax note", group: "GF", set: "990",
			body:  []string{b1, "N9*CN"},
			level: LevelElement, code: "2", seg: "N9", pos: 2, rule: "N9 syntax R0203",
		},
		{
			name: "paired syntax note", group: "QM", set: "214",
			body:  []string{b10, lx, "AT7*X3****20261119*1415"},
			level: LevelElement, code: "2", seg: "AT7", pos: 2, rule: "AT7 syntax P0102",
		},
		{
			name: "conditional syntax note", group: "QM", set: "214",
			body:  []string{b10, lx, "AT7*X3*NS***20261119"},
			level: LevelElement, code: "2", seg: "AT7", pos: 6, rule: "AT7 syntax C0506",
		},
		{
			name: "list conditional syntax note", group: "QM", set: "214",
			body:  []string{b10, lx, at7, "MS1*DALLAS"},
			level: LevelElement, code: "2", seg: "MS1", pos: 2, rule: "MS1 syntax L010203",
		},
		{
			name: "too many elements", group: "GF", set: "990",
			body:  []string{b1 + "*X"},
			level: LevelElement, code: "3", seg: "B1", pos: 5,
		},
		{
			name: "too short", group: "GF", set: "990",
			body:  []string{"B1*T*SIM000042*20261119*A"},
			level: LevelElement, code: "4", seg: "B1", pos: 1, rule: "B1-01 min 2",
		},
		{
			name: "too long", group: "GF", set: "990",
			body:  []string{"B1*TRNVXX*SIM000042*20261119*A"},
			level: LevelElement, code: "5", seg: "B1", pos: 1, rule: "B1-01 max 4",
		},
		{
			name: "invalid character", group: "GF", set: "990",
			body:  []string{"B1*TRNV*SIM\x01042*20261119*A"},
			level: LevelElement, code: "6", seg: "B1", pos: 2, rule: "B1-02 type AN",
		},
		{
			name: "explicit decimal point in N2", group: "IM", set: "210",
			body:  []string{"B3**INV1*SIM1*PP**20261119*1250.00****TRNV", "C3*USD", lx, l3},
			level: LevelElement, code: "6", seg: "B3", pos: 7, rule: "B3-07 type N2",
		},
		{
			name: "bad decimal in R", group: "IM", set: "210",
			body:  []string{b3, "C3*USD", lx, "L1*1*2.5.0*PM*100000", l3},
			level: LevelElement, code: "6", seg: "L1", pos: 2, rule: "L1-02 type R",
			loop: "0400",
		},
		{
			name: "invalid code", group: "GF", set: "990",
			body:  []string{"B1*TRNV*SIM000042*20261119*X"},
			level: LevelElement, code: "7", seg: "B1", pos: 4, rule: "B1-04 codes [A D]",
		},
		{
			name: "invalid code in a loop", group: "QM", set: "214",
			body:  []string{b10, lx, "AT7*ZZ*NS***20261119*1415*LT"},
			level: LevelElement, code: "7", seg: "AT7", pos: 1,
			rule: "AT7-01 codes [AF X3 X1 CD D1 AG]", loop: "0205",
		},
		{
			name: "invalid date", group: "GF", set: "990",
			body:  []string{"B1*TRNV*SIM000042*20261340*A"},
			level: LevelElement, code: "8", seg: "B1", pos: 3, rule: "B1-03 type DT",
		},
		{
			name: "invalid time", group: "GF", set: "990",
			body:  []string{b1, "N9*CN*PRO1**20261119*2575"},
			level: LevelElement, code: "9", seg: "N9", pos: 5, rule: "N9-05 type TM",
		},
		{
			name: "exclusion violated", group: "GF", set: "990",
			body:  []string{b1, "K1*ONE*TWO"},
			level: LevelElement, code: "10", seg: "K1", pos: 2, rule: "K1 syntax E0102",
		},
		{
			name: "components in a simple element", group: "GF", set: "990",
			body:  []string{"B1*TRNV*SIM>1*20261119*A"},
			level: LevelElement, code: "13", seg: "B1", pos: 2,
		},
		{
			name: "too many components", group: "GF", set: "990",
			body:  []string{b1, "N9*CN*PRO1*****PO>1>BN>2>X"},
			level: LevelElement, code: "13", seg: "N9", pos: 7, rule: "N9-07 components 4",
		},
		{
			name: "not used element present", group: "GF", set: "990",
			body:  []string{b1, "N9*CN*PRO1****LT"},
			level: LevelElement, code: "I10", seg: "N9", pos: 6, rule: "N9-06 not used",
		},
		{
			name: "not used element in a 210", group: "IM", set: "210",
			body:  []string{"B3**INV1*SIM1*PP*X*20261119*125000****TRNV", "C3*USD", lx, l3},
			level: LevelElement, code: "I10", seg: "B3", pos: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			j := judge(t, interchange(tt.group, transaction(tt.set, "0001", tt.body...)...))
			r := j.Reports()[0]
			f := findFinding(r.Findings, tt.level, tt.code)
			if f == nil || f.SegmentID != tt.seg || f.ElementPos != tt.pos ||
				f.ComponentPos != tt.comp || (tt.rule != "" && f.Rule != tt.rule) ||
				(tt.loop != "" && f.LoopID != tt.loop) {
				t.Fatalf("want %s %s on %s-%d-%d %q loop %q, got:\n%s",
					tt.level, tt.code, tt.seg, tt.pos, tt.comp, tt.rule, tt.loop,
					errorSummary(r.Findings))
			}
			if f.SegmentIndex < 1 || f.Message == "" {
				t.Fatalf("finding lacks index or message: %+v", f)
			}
			if r.Accepted() || j.Accepted() {
				t.Fatal("a transaction with an error must not be accepted")
			}
		})
	}
}

func TestFindingsAreCapped(t *testing.T) {
	t.Parallel()

	body := []string{b1}
	for range maxFindingsPerTransaction + 5 {
		body = append(body, "G62*1")
	}
	r := judge(t, interchange("GF", transaction("990", "0001", body...)...)).Reports()[0]
	if len(r.Findings) != maxFindingsPerTransaction || r.Omitted == 0 {
		t.Fatalf("got %d findings, %d omitted", len(r.Findings), r.Omitted)
	}
}

func TestFindingsAreInSegmentOrder(t *testing.T) {
	t.Parallel()

	body := []string{"B1*TRNV*SIM000042*20261119*X", "G62*1", "K1*ONE*TWO"}
	r := judge(t, interchange("GF", transaction("990", "0001", body...)...)).Reports()[0]
	last := 0
	for _, f := range r.Findings {
		if f.SegmentIndex < last {
			t.Fatalf("findings out of order:\n%s", errorSummary(r.Findings))
		}
		last = f.SegmentIndex
	}
	if !strings.Contains(errorSummary(r.Findings), "G62") {
		t.Fatalf("missing G62 finding:\n%s", errorSummary(r.Findings))
	}
}
