package partner_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/edi-partner-sim/internal/acknowledge"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/pkg/lading"
)

const ruleSeeds = 200

func openPartner(t *testing.T, name string) *partner.Definition {
	t.Helper()
	definition, err := partner.Open(partnersRoot + "/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return definition
}

func controlsAt(start int64) partner.Controls {
	return partner.Controls{
		Interchange: partner.NewCounter(start),
		Group:       partner.NewCounter(start),
	}
}

func heldWith(transaction *lading.TransactionResult, prefix string) bool {
	for _, outcome := range transaction.Outcomes {
		if outcome.Kind == lading.OutcomeHold && strings.HasPrefix(outcome.Message, prefix) {
			return true
		}
	}
	return false
}

func TestTheRulesAgreeWithTheGenerator(t *testing.T) {
	t.Parallel()

	cases := []struct {
		partner string
		hold    string
		truth   func(*partner.Load) bool
	}{
		{
			partner: "retailer",
			hold:    "Over 43,500 lb",
			truth:   func(load *partner.Load) bool { return load.Weight() > 43_500 },
		},
		{
			partner: "grocer",
			hold:    "Meadowridge's ranges don't overlap",
			truth:   (*partner.Load).TemperatureConflict,
		},
	}
	for _, c := range cases {
		t.Run(c.partner, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			definition := openPartner(t, c.partner)
			project := encodeProject(t, definition)
			found := 0
			for seed := uint64(1); seed <= ruleSeeds; seed++ {
				written, load, err := definition.Interchange(
					"204",
					seed,
					partner.Variation{Kind: partner.KindOriginal},
					controlsAt(int64(seed)),
				)
				if err != nil {
					t.Fatal(err)
				}
				result, err := runTenders(ctx, project, "tenderIn", written.Bytes)
				if err != nil {
					t.Fatal(err)
				}
				if len(result.Transactions) != 1 {
					t.Fatalf("seed %d read %d transactions", seed, len(result.Transactions))
				}
				want := c.truth(load)
				if got := heldWith(&result.Transactions[0], c.hold); got != want {
					t.Errorf(
						"seed %d: held %q is %v, the generator says %v",
						seed,
						c.hold,
						got,
						want,
					)
				}
				if want {
					found++
				}
			}
			if found == 0 {
				t.Errorf("no seed of %d gave a load the rule should hold", ruleSeeds)
			}
		})
	}
}

func TestTheRetailerBatchesTendersIntoOneNamedFile(t *testing.T) {
	t.Parallel()

	definition := openPartner(t, "retailer")
	sends := []partner.Send{
		{Seed: 13, Variation: partner.Variation{Kind: partner.KindOriginal}},
		{Seed: 14, Variation: partner.Variation{Kind: partner.KindOriginal}},
		{Seed: 15, Variation: partner.Variation{Kind: partner.KindCancel}},
	}
	written, _, err := definition.Batch("204", sends, controlsAt(610))
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 1 || len(written[0].Groups) != 1 ||
		len(written[0].Groups[0].Transactions) != 3 {
		t.Fatalf("three tenders wrote %+v, want one interchange of one group of three", written)
	}
	name, err := definition.Partner.FileName(written[0])
	if err != nil {
		t.Fatal(err)
	}
	want := "TMB_204_0076543210_" + written[0].At.Format("20060102150405") + "_000000610.edi"
	if name != want {
		t.Errorf("the file is named %q, want %q", name, want)
	}
	if !bytes.Contains(written[0].Bytes, []byte("|00401|000000610|1|T|^~\n")) {
		t.Errorf("the ISA does not ask for a TA1 with ^ components:\n%s", written[0].Bytes)
	}
}

func TestTheRetailerAcknowledgesUnder5010(t *testing.T) {
	t.Parallel()

	p := openPartner(t, "retailer").Partner
	ack, err := acknowledge.Build(&p.Acknowledgment, acknowledge.Input{
		GroupID:      "GF",
		GroupControl: "41",
		Version:      "004010",
		Transactions: []acknowledge.InputTransaction{
			{Set: "990", Control: "0001", Code: acknowledge.Accepted},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	written, err := p.Writer(controlsAt(900)).Write(
		openPartner(t, "retailer").Templates["204"].Load.Generate("retailer", p.Location(), 1).
			TenderedAt,
		[]partner.Transaction{ack},
	)
	if err != nil {
		t.Fatal(err)
	}
	text := string(written[0].Bytes)
	for _, want := range []string{
		"|{|00501|000000900|1|T|^~\n",
		"GS|FA|0076543210|TRNV|",
		"|900|X|005010X231A1~\n",
		"ST|999|0001|005010X231A1~\nAK1|GF|41|004010~\nAK2|990|0001~\nIK5|A~\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the 999 lacks %q:\n%s", want, text)
		}
	}
	if _, err := p.FileName(written[0]); err != nil {
		t.Errorf("a 999 has no file name: %v", err)
	}
}

func TestTheLegacyPartnerResendsTheSameBytes(t *testing.T) {
	t.Parallel()

	definition := openPartner(t, "legacy")
	original, _, err := definition.Interchange(
		"204", 37, partner.Variation{Kind: partner.KindOriginal}, controlsAt(802),
	)
	if err != nil {
		t.Fatal(err)
	}
	resent, _, err := definition.Interchange(
		"204", 37, partner.Variation{Kind: partner.KindDuplicate}, controlsAt(original.Interchange),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !definition.Resends("204") || !bytes.Equal(original.Bytes, resent.Bytes) {
		t.Fatalf("a legacy resend differs from its original:\n%s\n%s", original.Bytes, resent.Bytes)
	}
	text := string(original.Bytes)
	if !strings.HasPrefix(text, "ISA*00**00**01*071284355*02*TRNV*") ||
		!strings.Contains(text, "~\r\nS5*1*ld*") {
		t.Errorf("the legacy tender is not squeezed, CRLF and lowercase:\n%s", text)
	}
	if _, err := definition.Partner.FileName(original); err == nil {
		t.Error("an AS2 partner named a file")
	}
}

func TestTheGrocerSendsARangePerCommodityClass(t *testing.T) {
	t.Parallel()

	definition := openPartner(t, "grocer")
	load, err := definition.Load("204", 8)
	if err != nil {
		t.Fatal(err)
	}
	pickup := load.Stops[0]
	classes := map[partner.TempRange]bool{}
	for _, line := range pickup.Lines {
		if line.Temperature != nil {
			classes[*line.Temperature] = true
		}
	}
	if len(classes) == 0 || len(pickup.Temperatures) != len(classes) {
		t.Fatalf("the pickup carries %v for lines %+v", pickup.Temperatures, pickup.Lines)
	}
	for _, stop := range load.Stops[1:] {
		if len(stop.Temperatures) == 0 {
			t.Errorf("drop %d has no range", stop.Sequence)
		}
	}
	if load.TemperatureConflict() {
		t.Errorf("seed 8 is the grocer's ordinary chilled run, not a conflict")
	}
}
