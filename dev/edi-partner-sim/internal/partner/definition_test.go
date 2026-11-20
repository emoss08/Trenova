package partner_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

func TestTheSameSeedWritesTheSameBytes(t *testing.T) {
	t.Parallel()

	definition, err := partner.Open(partnersRoot + "/manufacturer")
	if err != nil {
		t.Fatal(err)
	}
	write := func(seed uint64) []byte {
		controls := partner.Controls{
			Interchange: partner.NewCounter(1),
			Group:       partner.NewCounter(1),
		}
		written, _, err := definition.Interchange(
			"204",
			seed,
			partner.Variation{Kind: partner.KindOriginal},
			controls,
		)
		if err != nil {
			t.Fatal(err)
		}
		return written.Bytes
	}
	if !bytes.Equal(write(42), write(42)) {
		t.Fatal("seed 42 wrote different bytes twice")
	}
	if bytes.Equal(write(42), write(43)) {
		t.Fatal("seeds 42 and 43 wrote the same bytes")
	}
}

func TestVariationsChangeWhatTheyName(t *testing.T) {
	t.Parallel()

	definition, err := partner.Open(partnersRoot + "/manufacturer")
	if err != nil {
		t.Fatal(err)
	}
	_, original, err := definition.Transaction(
		"204",
		37,
		partner.Variation{Kind: partner.KindOriginal},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, raised, err := definition.Transaction("204", 37, partner.Variation{
		Kind:  partner.KindChange,
		Delta: partner.Delta{RatePercent: 10, AddStop: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(raised.Stops) != len(original.Stops)+1 {
		t.Errorf("an added stop made %d stops from %d", len(raised.Stops), len(original.Stops))
	}
	if raised.TotalCents <= original.TotalCents {
		t.Errorf(
			"a 10%% raise and a stop-off made %d from %d",
			raised.TotalCents,
			original.TotalCents,
		)
	}
	if _, _, err := definition.Transaction("204", 37, partner.Variation{Kind: "resend"}); err == nil {
		t.Error("an unknown variation kind was written")
	}
}

func TestProblemsNameTheFileAndPath(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile(partnersRoot + "/manufacturer/partner.yaml")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "manufacturer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(source), `sends: "997"`, `sends: "998"`, 1)
	broken = strings.Replace(broken, "terminator: lf", "terminator: cr", 1)
	broken += "\nanswers:\n  policy: accept\n"
	if err := os.WriteFile(filepath.Join(dir, "partner.yaml"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = partner.Open(dir)
	var problems partner.Problems
	if !errors.As(err, &problems) {
		t.Fatalf("a broken partner.yaml opened: %v", err)
	}
	text := problems.Error()
	for _, want := range []string{
		filepath.Join(dir, "partner.yaml") + ": acknowledgment.sends: \"998\" is not one of 997, 999, none",
		"envelope.terminator: \"cr\" is not one of none, lf, crlf",
		"answers: only a carrier answers tenders",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the problems lack %q:\n%s", want, text)
		}
	}
}
