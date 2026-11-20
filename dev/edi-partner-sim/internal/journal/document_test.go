package journal

import "testing"

func TestLinkAckJoinsAnAcknowledgmentToWhatItAnswers(t *testing.T) {
	t.Parallel()

	j := New(10, NewBroker(10, fixedNow), fixedNow)
	tender := j.Add("message.sent", &Message{Partner: "a", Direction: ToTrenova,
		Documents: []Document{
			{Set: "204", GroupID: "SM", GroupControl: "501", Control: "0001",
				Interchange: "000000501", Keys: []string{"K1"}, Title: "Original tender"},
			{Set: "204", GroupID: "SM", GroupControl: "501", Control: "0002",
				Interchange: "000000501", Keys: []string{"K2"}},
		}})
	if tender.Documents[0].Number != 1 || tender.Documents[1].Number != 2 {
		t.Fatalf("numbers %+v", tender.Documents)
	}
	ack := j.Add("message.received", &Message{Partner: "a", Direction: FromTrenova,
		Documents: []Document{{Set: "997", Interchange: "000000912", Acknowledges: []Acked{
			{Set: "204", GroupID: "SM", GroupControl: "000000501", Control: "1", Code: "R"},
		}}}})
	j.LinkAck(ack.ID, tender.ID)
	linked, _ := j.Get(tender.ID)
	if by := linked.Documents[0].AnsweredBy; by == nil || by.Number != 3 || by.Code != "R" ||
		by.MessageID != ack.ID {
		t.Fatalf("answered by %+v", by)
	}
	if linked.Documents[1].AnsweredBy != nil {
		t.Fatal("the 997 answered a transaction it does not name")
	}
	answer, _ := j.Get(ack.ID)
	if refs := answer.Documents[0].Answers; len(refs) != 1 || refs[0].Number != 1 ||
		refs[0].Title != "Original tender" || answer.Documents[0].Keys[0] != "K1" {
		t.Fatalf("answers %+v", answer.Documents[0])
	}
	restored := New(10, NewBroker(10, fixedNow), fixedNow)
	restored.Restore(j.Snapshot())
	next := restored.Add("message.sent", &Message{Partner: "a", Documents: []Document{{}}})
	if next.Documents[0].Number != 4 {
		t.Fatalf("document numbers restart at %d", next.Documents[0].Number)
	}
}
