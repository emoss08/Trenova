package ladder

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
)

var start = time.Date(2026, 3, 2, 14, 0, 0, 0, time.UTC)

func at(minutes int) time.Time {
	return start.Add(time.Duration(minutes) * time.Minute)
}

func accepted(value bool) *bool {
	return &value
}

func messages() []journal.Message {
	delivered := at(0)
	return []journal.Message{
		{ID: "m-1", Direction: journal.ToTrenova, State: journal.StateDelivered, At: at(0),
			DeliveredAt: &delivered, Documents: []journal.Document{{Number: 1, Set: "204",
				Control: "0001", GroupControl: "501", Interchange: "000000501",
				Title: "Original tender", Keys: []string{"K1"},
				AnsweredBy: &journal.DocRef{Number: 2, Set: "997", Code: "A"}}}},
		{ID: "m-2", Direction: journal.FromTrenova, State: journal.StateReceived, At: at(1),
			Documents: []journal.Document{{Number: 2, Set: "997", Interchange: "000000912",
				Keys: []string{"K1"}, Judged: true, Accepted: accepted(true),
				AckCode: "A"}}},
		{ID: "m-3", Direction: journal.FromTrenova, State: journal.StateReceived, At: at(9),
			Documents: []journal.Document{
				{Number: 3, Set: "210", Control: "0001", GroupControl: "916",
					Interchange: "000000916", Keys: []string{"K1"}, Judged: true,
					Accepted: accepted(false), Findings: []journal.Note{
						{Severity: "error", Segment: 2, Element: 2, Message: "too long"},
					}},
				{Number: 4, Set: "210", Control: "0002", GroupControl: "916",
					Interchange: "000000916", Keys: []string{"K2"}, Judged: true,
					Accepted: accepted(true)},
			}},
		{ID: "m-4", Direction: journal.ToTrenova, State: journal.StateDelivered, At: at(10),
			Documents: []journal.Document{{Number: 5, Set: "997", Interchange: "000000505",
				Keys: []string{"K1"}, AckCode: "R"}}},
	}
}

func TestComposeDrawsDocumentsEventsAndExpectations(t *testing.T) {
	t.Parallel()

	scenario := &runlog.Scenario{Steps: []runlog.Step{
		{Index: 0, Kind: "send", Status: runlog.StatusPassed,
			Matches: []runlog.Match{{MessageID: "m-1", Key: "K1"}}},
		{Index: 1, Kind: "trenova", Status: runlog.StatusPassed,
			Events: []runlog.Event{{Key: "K1", At: at(2), Label: "Planner accepted"}}},
		{Index: 2, Kind: "advance", Status: runlog.StatusPassed, Ended: at(3),
			Keys: []string{"K1"}, Gap: "advanced 2h"},
		{Index: 3, Kind: "expect", Status: runlog.StatusPassed,
			Events: []runlog.Event{{Key: "K1", At: at(4), Label: "Flow held",
				Tone: runlog.ToneHold}}},
		{Index: 4, Kind: "expect", Status: runlog.StatusFailed,
			Matches: []runlog.Match{{MessageID: "m-3", GroupControl: "916", Control: "0001",
				Key: "K1"}}},
		{Index: 5, Kind: "expect", Status: runlog.StatusPassed,
			Events: []runlog.Event{{Key: "K2", At: at(5), Label: "elsewhere"}}},
	}}
	rows := Compose("K1", messages(), scenario)
	want := []struct {
		kind, set, label, expect, tone, gap string
	}{
		{kind: KindMessage, set: "204", expect: ToneOK, tone: ToneOK},
		{kind: KindMessage, set: "997", tone: ToneOK},
		{kind: KindEvent, label: "Planner accepted", tone: runlog.ToneTrenova},
		{kind: KindEvent, label: "Flow held", expect: ToneOK, tone: runlog.ToneHold,
			gap: "advanced 2h"},
		{kind: KindMessage, set: "210", expect: ToneBad, tone: ToneBad},
		{kind: KindMessage, set: "997", tone: ToneBad},
	}
	if len(rows) != len(want) {
		t.Fatalf("rows %+v", rows)
	}
	for index, w := range want {
		got := rows[index]
		if got.Kind != w.kind || got.Set != w.set || got.Label != w.label ||
			got.Expect != w.expect || got.Tone != w.tone || got.Gap != w.gap {
			t.Errorf("row %d: %+v, want %+v", index, got, w)
		}
	}
	if rows[0].Verdict != VerdictRead || rows[4].Note != "oracle: 1 error" ||
		rows[5].Verdict != VerdictRejects || rows[5].Note != "AK5 R" || !rows[1].Ack {
		t.Fatalf("verdicts: %+v %+v %+v", rows[0], rows[4], rows[5])
	}
}

func TestAConversationWithoutARunHasNoMarks(t *testing.T) {
	t.Parallel()

	rows := Compose("K2", messages(), nil)
	if len(rows) != 1 || rows[0].Number != 4 || rows[0].Expect != "" ||
		rows[0].Verdict != VerdictPasses {
		t.Fatalf("rows %+v", rows)
	}
}
