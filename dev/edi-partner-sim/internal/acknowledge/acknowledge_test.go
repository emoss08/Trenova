package acknowledge

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

func TestOracleFindingsBecomeAckNotes(t *testing.T) {
	t.Parallel()

	findings := []oracle.Finding{
		{Severity: oracle.SeverityError, Level: oracle.LevelElement, Code: "1",
			SegmentID: "B1", SegmentIndex: 2, ElementPos: 4},
		{Severity: oracle.SeverityError, Level: oracle.LevelElement, Code: "I10",
			SegmentID: "B1", SegmentIndex: 2, ElementPos: 5, Value: "X"},
		{Severity: oracle.SeverityError, Level: oracle.LevelSegment, Code: "3",
			SegmentID: "N9", SegmentIndex: 3, LoopID: "0100"},
		{Severity: oracle.SeverityWarning, Level: oracle.LevelElement, Code: "7",
			SegmentID: "N9", SegmentIndex: 4, ElementPos: 1},
		{Severity: oracle.SeverityError, Level: oracle.LevelTransaction, Code: "4"},
	}
	notes := Notes(findings, partner.Ack997)
	if len(notes) != 2 {
		t.Fatalf("want 2 notes, got %+v", notes)
	}
	b1 := notes[0]
	if b1.ID != "B1" || b1.Position != 2 || b1.Code != SegmentHasElementErrors ||
		len(b1.Elements) != 2 || b1.Elements[0].Code != "1" || b1.Elements[1].Code != "3" ||
		b1.Elements[1].Copy != "X" {
		t.Fatalf("B1 note: %+v", b1)
	}
	if n9 := notes[1]; n9.ID != "N9" || n9.Code != "3" || n9.Loop != "0100" ||
		len(n9.Elements) != 0 {
		t.Fatalf("N9 note: %+v", n9)
	}
	if code := Notes(findings, partner.Ack999)[0].Elements[1].Code; code != "I10" {
		t.Fatalf("a 999 keeps I10, got %s", code)
	}
}

func TestPlanAcknowledgesEveryDocumentGroupButAcks(t *testing.T) {
	t.Parallel()

	judgement := &oracle.Judgement{Interchanges: []*oracle.Interchange{{
		ControlNumber: "000000101",
		Groups: []*oracle.Group{
			{FunctionalID: "GF", ControlNumber: "101", Version: "004010", Reports: []*oracle.Report{
				{Set: "990", ControlNumber: "0001"},
				{Set: "990", ControlNumber: "0002", Findings: []oracle.Finding{{
					Severity: oracle.SeverityError, Level: oracle.LevelElement, Code: "7",
					SegmentID: "B1", SegmentIndex: 2, ElementPos: 4, Value: "Q",
				}}},
			}},
			{FunctionalID: "FA", ControlNumber: "102", Version: "004010", Reports: []*oracle.Report{
				{Set: "997", ControlNumber: "0001"},
			}},
		},
	}}}
	rules := &partner.Acknowledgment{
		Sends:  partner.Ack997,
		Reject: partner.RejectRules{When: partner.RejectErrors},
	}
	plan := Plan(rules, judgement)
	if len(plan) != 1 || plan[0].GroupControl != "101" || len(plan[0].Transactions) != 2 {
		t.Fatalf("plan %+v", plan)
	}
	if plan[0].Transactions[0].Code != partner.AckAccepted ||
		plan[0].Transactions[1].Code != partner.AckRejected {
		t.Fatalf("codes %+v", plan[0].Transactions)
	}
	built, err := rules.Build(plan[0])
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if TransactionCode(built) != partner.AckAccepted {
		t.Fatalf("the first AK5 says %s", TransactionCode(built))
	}
	if Plan(&partner.Acknowledgment{Sends: partner.AckNone}, judgement) != nil {
		t.Fatal("a partner that sends no acknowledgment planned one")
	}
}

func TestTA1AcceptsOrNamesTheEnvelopeError(t *testing.T) {
	t.Parallel()

	clean := &oracle.Interchange{ControlNumber: "000000101", Date: "260302", Time: "0900"}
	if ta1 := TA1(clean); ta1.Code != partner.TA1Accepted || ta1.Note != partner.TA1NoError ||
		ta1.Date != "260302" {
		t.Fatalf("clean: %+v", ta1)
	}
	broken := &oracle.Interchange{ControlNumber: "7", Findings: []oracle.Finding{
		{Severity: oracle.SeverityError, Level: oracle.LevelInterchange, Code: "001"},
	}}
	if ta1 := TA1(broken); ta1.Code != partner.TA1Rejected || ta1.Note != "001" {
		t.Fatalf("broken: %+v", ta1)
	}
}

func TestAWrittenTA1IsReadBack(t *testing.T) {
	t.Parallel()

	p := partner.Partner{
		Them:    partner.Identity{Qualifier: "ZZ", ID: "PARTNER", Group: "PARTNER"},
		Trenova: partner.Identity{Qualifier: "ZZ", ID: "TRENOVA", Group: "TRENOVA"},
		Envelope: partner.Envelope{
			Version: "004010",
			Separators: partner.Separators{
				Element:    "|",
				Segment:    "~",
				Component:  ":",
				Repetition: "^",
			},
			Terminator: partner.TerminatorCRLF,
			Usage:      "T",
		},
		Acknowledgment: partner.Acknowledgment{TA1: true},
	}
	writer := p.Writer(partner.Controls{
		Interchange: partner.NewCounter(40),
		Group:       partner.NewCounter(40),
	})
	written := writer.WriteTA1(time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC), partner.TA1{
		Control: "77", Date: "260302", Time: "0900", Code: partner.TA1Rejected, Note: "001",
	})
	judgement, err := new(oracle.GuideSet).Judge(written.Bytes)
	if err != nil {
		t.Fatalf("judge: %v", err)
	}
	if len(judgement.Interchanges) != 1 || len(judgement.Interchanges[0].Findings) != 0 ||
		judgement.Interchanges[0].AckRequested {
		t.Fatalf("the TA1 interchange: %+v", judgement.Interchanges)
	}
	read := ReadTA1(judgement.Interchanges[0])
	if len(read) != 1 || read[0].Control != "000000077" || read[0].Code != partner.TA1Rejected ||
		read[0].Note != "001" {
		t.Fatalf("read back %+v from\n%s", read, written.Bytes)
	}
}
