package sim

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

const specPath = "../../specs/manufacturer-implementation-guide.txt"

func readSpec(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read the manufacturer's guide: %v", err)
	}
	return string(raw)
}

func TestTheGuidesSampleIsTheTenderTheManufacturerSends(t *testing.T) {
	t.Parallel()

	spec := readSpec(t)
	start := strings.Index(spec, "ISA*")
	if start < 0 {
		t.Fatal("the guide has no sample interchange")
	}
	definition := openDefinitions(t, []string{"manufacturer"})[0]
	written, _, err := definition.Interchange(
		"204", 37, partner.Variation{Kind: partner.KindOriginal},
		partner.Controls{Interchange: partner.NewCounter(501), Group: partner.NewCounter(501)},
	)
	if err != nil {
		t.Fatalf("write the tender: %v", err)
	}
	if got := spec[start:]; got != string(written.Bytes) {
		t.Fatalf("the guide's sample is not what the manufacturer sends:\n%s\nwant:\n%s",
			got, written.Bytes)
	}
}

func TestTheGuideStatesTheManufacturersSettingsAndNoSecret(t *testing.T) {
	t.Parallel()

	spec := readSpec(t)
	definition := openDefinitions(t, []string{"manufacturer"})[0]
	p := &definition.Partner
	h := newHarness(t, setup{partners: []string{"manufacturer"}, options: func(o *Options) {
		o.PublicURL = defaultPublicURL
		o.SFTPListen = "127.0.0.1:0"
		o.SFTPRoot = t.TempDir()
	}})
	info := h.sim.info(h.peerOf("manufacturer"))
	for _, value := range []string{
		"Trading Partner Name: " + p.Display,
		"(" + p.Zone + ")",
		"AS2 Identifier: " + p.Transport.AS2.ID,
		"Carrier AS2 Identifier: " + p.Trenova.ID,
		"AS2 URL: " + info.AS2.URL,
		"MDN: synchronous, signed.",
		"Compression: none.",
		defaultPublicURL + "/control/manufacturer",
		"user " + info.SFTP.Username + ".",
		info.SFTP.InboundDirectory, info.SFTP.OutboundDirectory,
		"qualifier " + p.Them.Qualifier + " (mutually defined), ID " + p.Them.ID,
		"qualifier " + p.Trenova.Qualifier + " (mutually defined), ID " + p.Trenova.ID,
		"GS02 Application Sender Code: " + p.Them.Group,
		"GS03 Application Receiver Code: " + p.Trenova.Group,
		"GS08 Version: " + p.Envelope.Version,
		"ISA15 usage indicator: " + p.Envelope.Usage + ".",
		"B2-02 Standard Carrier Alpha Code is " + p.Trenova.SCAC + ".",
		"returns a " + string(p.Acknowledgment.Sends) + " Functional Acknowledgment",
		fmt.Sprintf("within %d seconds", int(p.Acknowledgment.Delay.Seconds())),
		"port 9222",
	} {
		if !strings.Contains(spec, value) {
			t.Errorf("the guide does not state %q", value)
		}
	}
	if !p.Transport.AS2.Signed || !p.Transport.AS2.Encrypted || p.Transport.AS2.Compressed ||
		p.Transport.AS2.MDN != mdnSync {
		t.Errorf("partner.yaml's AS2 settings no longer match the guide: %+v", p.Transport.AS2)
	}
	if strings.Contains(spec, "BEGIN") || strings.Contains(strings.ToLower(spec), "password:") {
		t.Error("the guide carries a key or a password; it must only say where to get them")
	}
}
