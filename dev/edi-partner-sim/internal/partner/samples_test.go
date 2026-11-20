package partner_test

import (
	"bytes"
	"flag"
	"os"
	"testing"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

const partnersRoot = "../../partners"

var update = flag.Bool("update", false, "rewrite each partner's generated Lading samples")

func openAll(t *testing.T) []*partner.Definition {
	t.Helper()
	definitions, err := partner.OpenAll(partnersRoot)
	if err != nil {
		t.Fatalf("open the partners:\n%v", err)
	}
	if len(definitions) == 0 {
		t.Fatalf("no partner under %s", partnersRoot)
	}
	return definitions
}

func TestSamplesAreTheGeneratorsOutput(t *testing.T) {
	t.Parallel()

	for _, definition := range openAll(t) {
		t.Run(definition.Name(), func(t *testing.T) {
			t.Parallel()
			if *update {
				if err := definition.WriteSamples(); err != nil {
					t.Fatal(err)
				}
			}
			samples, err := definition.GenerateSamples()
			if err != nil {
				t.Fatal(err)
			}
			for _, sample := range samples {
				onDisk, err := os.ReadFile(sample.Path)
				if err != nil {
					t.Fatalf("%v: run go test ./internal/partner -run TestSamples -update", err)
				}
				if !bytes.Equal(onDisk, sample.Content) {
					t.Errorf(
						"%s differs from what the generator writes for its seed; run go test ./internal/partner -run TestSamples -update",
						sample.Path,
					)
				}
			}
		})
	}
}
