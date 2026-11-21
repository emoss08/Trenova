package partner

import (
	"strings"
	"testing"
	"time"
)

func testPartner() *Partner {
	return &Partner{
		Them:    Identity{Qualifier: "ZZ", ID: "SENDER", Group: "SENDERGS"},
		Trenova: Identity{Qualifier: "01", ID: "123456789", Group: "TRENOVA"},
		Envelope: Envelope{
			Version:    "004010",
			Separators: Separators{Element: "|", Segment: "~", Component: ">", Repetition: "^"},
			Terminator: TerminatorCRLF,
			Usage:      "P",
		},
		Batching:       Batching{TransactionsPerGroup: 2, GroupsPerInterchange: 2},
		Acknowledgment: Acknowledgment{TA1: true},
	}
}

func tender(id string) Transaction {
	return Transaction{Set: "204", Segments: []Segment{
		{ID: "B2", Elements: single("", "TRNV", "", id, "", "PP", "", "")},
		{ID: "N1", Elements: [][]string{{"SH"}, {"A|B~C"}}},
	}}
}

func TestWriterBatchesSqueezesAndTerminates(t *testing.T) {
	t.Parallel()

	p := testPartner()
	p.Envelope.SqueezedISA = true
	at := time.Date(2026, 3, 2, 8, 15, 0, 0, time.UTC)
	controls := Controls{Interchange: NewCounter(7), Group: NewCounter(70)}
	written, err := p.Writer(controls).Write(at, []Transaction{
		tender("A1"), tender("A2"), tender("A3"), tender("A4"), tender("A5"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 2 {
		t.Fatalf(
			"five tenders at two per group and two groups per interchange wrote %d interchanges, want 2",
			len(written),
		)
	}
	first := string(written[0].Bytes)
	wantISA := "ISA|00||00||ZZ|SENDER|01|123456789|260302|0815|^|00401|000000007|1|P|>~\r\n"
	if !strings.HasPrefix(first, wantISA) {
		t.Fatalf("the squeezed ISA is %q, want %q", strings.SplitAfter(first, "\n")[0], wantISA)
	}
	for _, want := range []string{
		"GS|SM|SENDERGS|TRENOVA|20260302|0815|70|X|004010~\r\n",
		"ST|204|0002~\r\nB2||TRNV||A2||PP~\r\nN1|SH|A B C~\r\nSE|4|0002~\r\n",
		"GE|2|71~\r\nIEA|2|000000007~\r\n",
	} {
		if !strings.Contains(first, want) {
			t.Errorf("the interchange lacks %q:\n%s", want, first)
		}
	}
	if got := written[1].Groups; len(got) != 1 || got[0].Control != 72 ||
		len(got[0].Transactions) != 1 {
		t.Errorf("the second interchange holds %+v, want one group 72 with one tender", got)
	}
}

func TestWriterPadsTheISA(t *testing.T) {
	t.Parallel()

	p := testPartner()
	controls := Controls{Interchange: NewCounter(1), Group: NewCounter(1)}
	written, err := p.Writer(controls).Write(time.Now(), []Transaction{tender("A1")})
	if err != nil {
		t.Fatal(err)
	}
	isa := strings.SplitN(string(written[0].Bytes), "~", 2)[0]
	if len(isa) != 105 {
		t.Fatalf("a padded ISA is %d characters, want 105: %q", len(isa), isa)
	}
}

func TestWriterKeepsEachVersionInItsOwnInterchange(t *testing.T) {
	t.Parallel()

	p := testPartner()
	p.Envelope.Versions = map[string]string{"999": "005010X231A1"}
	ack := Transaction{Set: "999", Segments: []Segment{
		{ID: "AK1", Elements: single("SM", "1", "004010")},
	}}
	at := time.Date(2026, 3, 2, 8, 15, 0, 0, time.UTC)
	controls := Controls{Interchange: NewCounter(5), Group: NewCounter(50)}
	written, err := p.Writer(controls).Write(at, []Transaction{tender("A1"), ack})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 2 {
		t.Fatalf("a 204 and a 999 wrote %d interchanges, want 2", len(written))
	}
	first, second := string(written[0].Bytes), string(written[1].Bytes)
	if !strings.Contains(first, "|^|00401|") || !strings.Contains(first, "|X|004010~") ||
		!strings.Contains(first, "ST|204|0001~") {
		t.Errorf("the 204 is not written under 004010:\n%s", first)
	}
	for _, want := range []string{"|^|00501|", "|51|X|005010X231A1~", "ST|999|0001|005010X231A1~"} {
		if !strings.Contains(second, want) {
			t.Errorf("the 999 lacks %q:\n%s", want, second)
		}
	}
	if written[1].Groups[0].Version != "005010X231A1" {
		t.Errorf("the 999's group records version %q", written[1].Groups[0].Version)
	}
}

func TestWriterWritesAnEmptyRepetitionAsU(t *testing.T) {
	t.Parallel()

	p := testPartner()
	p.Envelope.Separators.Repetition = ""
	controls := Controls{Interchange: NewCounter(1), Group: NewCounter(1)}
	written, err := p.Writer(controls).Write(time.Now(), []Transaction{tender("A1")})
	if err != nil {
		t.Fatal(err)
	}
	text := string(written[0].Bytes)
	if !strings.Contains(text, "|U|00401|") || !strings.Contains(text, "B2||TRNV||A1||PP~") {
		t.Errorf("an empty repetition separator garbled the interchange:\n%s", text)
	}
}

func TestFileNamesNameTheirFields(t *testing.T) {
	t.Parallel()

	check := &problems{file: "partner.yaml"}
	validateFileName(check, "transport.sftp.fileName", "{set}_{seq}/x.edi")
	got := check.list.Error()
	for _, want := range []string{
		"{seq} is not one of",
		"needs {control}",
		"may hold only letters",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the problems lack %q:\n%s", want, got)
		}
	}
	p := testPartner()
	p.Transport.SFTP = &SFTP{FileName: "{sender}_{set}_{date}{time}_{control}.x12"}
	name, err := p.FileName(Written{
		At:          time.Date(2026, 3, 2, 8, 15, 9, 0, time.UTC),
		Interchange: 42,
		Groups:      []WrittenGroup{{Transactions: []WrittenTransaction{{Set: "204"}}}},
	})
	if err != nil || name != "SENDER_204_20260302081509_000000042.x12" {
		t.Errorf("FileName = %q, %v", name, err)
	}
}
