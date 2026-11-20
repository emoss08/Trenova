package transcript

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
)

const tender = "ISA*00*          *00*          *ZZ*KESTRELVW      *ZZ*TRNVLOGISTICS  " +
	"*261119*1402*U*00401*000000503*0*T*>~\n" +
	"GS*SM*KESTRELVW*TRNVLOGISTICS*20261119*1402*503*X*004010~\n" +
	"ST*204*0001~\nB2**TRNV**KVW-48213**PP~\nB2A*04*LT~\nL3*42000*G***240240******24*L~\n" +
	"SE*5*0001~\nGE*1*503~\nIEA*1*000000503~\n"

const ack = "ISA*00*          *00*          *ZZ*TRNVLOGISTICS  *ZZ*KESTRELVW      " +
	"*261119*1402*U*00401*000000914*1*T*>~\n" +
	"TA1*000000503*261119*1402*A*000~\n" +
	"GS*FA*TRNVLOGISTICS*KESTRELVW*20261119*1402*914*X*004010~\n" +
	"ST*997*0001~\nAK1*SM*503~\nAK2*204*0001~\nAK5*R*5~\nAK9*R*1*1*0~\nSE*6*0001~\n" +
	"GE*1*914~\nIEA*1*000000914~\n"

func judge(t *testing.T, raw string) *oracle.Judgement {
	t.Helper()
	j, err := new(oracle.GuideSet).Judge([]byte(raw))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return j
}

func TestATenderBecomesADocumentWithItsEnvelopeAndRateChange(t *testing.T) {
	t.Parallel()

	documents := Build(judge(t, tender), Options{Earlier: func(key string) (int64, bool) {
		return 214500, key == "KVW-48213"
	}})
	if len(documents) != 1 {
		t.Fatalf("documents %+v", documents)
	}
	d := documents[0]
	if d.Set != "204" || d.Control != "0001" || d.GroupID != "SM" || d.GroupControl != "503" ||
		d.Interchange != "000000503" || !slices.Equal(d.Keys, []string{"KVW-48213"}) ||
		d.ChargeCents != 240240 || d.Title != "Change: rate +12%" || d.Judged ||
		d.Accepted != nil {
		t.Fatalf("the document %+v", d)
	}
	if d.Separator != "*" || d.Terminator != "~" || len(d.Header) != 2 ||
		d.Header[0][0] != "ISA" || d.Header[1][0] != "GS" || len(d.Trailer) != 2 ||
		d.Trailer[0][0] != "GE" || d.Trailer[1][0] != "IEA" || len(d.Segments) != 5 ||
		d.Segments[1][4] != "KVW-48213" {
		t.Fatalf("the text %+v", d)
	}
}

func TestAnAcknowledgmentNamesWhatItAnswers(t *testing.T) {
	t.Parallel()

	documents := Build(judge(t, ack), Options{Judged: true})
	if len(documents) != 2 {
		t.Fatalf("documents %+v", documents)
	}
	answer, ta1 := documents[0], documents[1]
	if answer.Title != "Rejected 204 503" || answer.AckCode != "R" ||
		len(answer.Acknowledges) != 1 || answer.Acknowledges[0].Control != "0001" ||
		answer.Acknowledges[0].GroupControl != "503" || answer.Accepted == nil {
		t.Fatalf("the 997 %+v", answer)
	}
	if ta1.Set != "TA1" || ta1.Title != "Interchange accepted 503" ||
		ta1.Acknowledges[0].Interchange != "000000503" || len(ta1.Header) != 1 {
		t.Fatalf("the TA1 %+v", ta1)
	}
}

func TestStatusAndAnswerTitles(t *testing.T) {
	t.Parallel()

	for _, c := range []struct{ body, want string }{
		{"ST*214*0001~B10*KVW-1*BOL*TRNV~AT7*X3*NS***20261120*0802*CT~AT7*AF~SE*4*0001",
			"X3 arrived pickup, AF"},
		{"ST*990*0001~B1*TRNV*KVW-1*20261119*D~SE*3*0001", "Decline"},
		{"ST*210*0001~B3**INV-1001*KVW-1*PP~SE*3*0001", "Invoice INV-1001"},
	} {
		raw := "ISA*00*          *00*          *ZZ*A              *ZZ*B              " +
			"*261119*1402*U*00401*000000001*0*T*>~GS*XX*A*B*20261119*1402*1*X*004010~" +
			c.body + "~GE*1*1~IEA*1*000000001~"
		documents := Build(judge(t, raw), Options{})
		if len(documents) != 1 || documents[0].Title != c.want {
			t.Errorf("%s: %+v", c.want, documents)
		}
	}
}
