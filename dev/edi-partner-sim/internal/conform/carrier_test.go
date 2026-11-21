package conform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/acknowledge"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/stringutils"
)

const fixturePartner = `name: swift
display: Swift Prairie Freight
role: carrier
zone: America/Chicago
them: { qualifier: ZZ, id: SWIFTPF, group: SWIFTPF, scac: SWPF }
trenova: { qualifier: ZZ, id: TRENOVA, group: TRENOVA, scac: TRNV }
envelope:
  version: "004010"
  separators: { element: "*", segment: "~", component: ">", repetition: "^" }
  terminator: lf
  squeezedIsa: false
  usage: T
transport:
  as2: { id: SWIFTPF, signed: true, encrypted: true, compressed: false, mdn: sync }
batching: { transactionsPerGroup: 1, groupsPerInterchange: 1 }
acknowledgment:
  sends: "997"
  expects: "997"
  ta1: false
  delay: 30s
  reject: { when: errors, sets: [] }
  never: 0
answers: { policy: accept, pace: 2s }
sends: ["990", "214", "210", "997"]
receives: ["204", "997"]
`

const fixtureGuide = `set: "204"
version: "004010"
group: SM
title: "Swift Prairie Load Tender Guide 1.0"
segments:
  ST:
    elements:
      - { pos: 1, type: ID, min: 3, max: 3, usage: M, fixed: "204" }
      - { pos: 2, type: AN, min: 4, max: 9, usage: M }
  B2:
    elements:
      - { pos: 2, type: ID, min: 2, max: 4, usage: M, fixed: TRNV }
      - { pos: 4, type: AN, min: 1, max: 30, usage: M }
      - { pos: 6, type: ID, min: 2, max: 2, usage: M, codes: [PP, CC, TP] }
  B2A:
    elements:
      - { pos: 1, type: ID, min: 2, max: 2, usage: M, codes: ["00", "01", "04"] }
  L11:
    elements:
      - { pos: 1, type: AN, min: 1, max: 30, usage: M }
      - { pos: 2, type: ID, min: 2, max: 3, usage: M, codes: [BM, PO] }
  S5:
    elements:
      - { pos: 1, type: N0, min: 1, max: 3, usage: M }
      - { pos: 2, type: ID, min: 2, max: 2, usage: M, codes: [LD, UL] }
  G62:
    elements:
      - { pos: 1, type: ID, min: 2, max: 2, usage: M, codes: ["37", "53"] }
      - { pos: 2, type: DT, min: 8, max: 8, usage: M }
      - { pos: 3, type: ID, min: 1, max: 2, usage: M }
      - { pos: 4, type: TM, min: 4, max: 4, usage: M }
      - { pos: 5, type: ID, min: 2, max: 2, usage: O }
  N1:
    elements:
      - { pos: 1, type: ID, min: 2, max: 3, usage: M, codes: [SH, CN] }
      - { pos: 2, type: AN, min: 1, max: 60, usage: M }
  N3:
    elements:
      - { pos: 1, type: AN, min: 1, max: 55, usage: M }
      - { pos: 2, type: AN, min: 1, max: 55, usage: O }
  N4:
    elements:
      - { pos: 1, type: AN, min: 2, max: 30, usage: M }
      - { pos: 2, type: ID, min: 2, max: 2, usage: M }
      - { pos: 3, type: ID, min: 3, max: 15, usage: M }
      - { pos: 4, type: ID, min: 2, max: 3, usage: O }
  L3:
    elements:
      - { pos: 1, type: R, min: 1, max: 10, usage: O }
      - { pos: 2, type: ID, min: 1, max: 2, usage: O }
      - { pos: 5, type: N2, min: 1, max: 12, usage: M }
      - { pos: 12, type: ID, min: 1, max: 1, usage: O }
    syntax: [P0102]
  SE:
    elements:
      - { pos: 1, type: N0, min: 1, max: 10, usage: M }
      - { pos: 2, type: AN, min: 4, max: 9, usage: M }
structure:
  - { seg: ST, usage: M }
  - { seg: B2, usage: M }
  - { seg: B2A, usage: M }
  - { seg: L11, usage: O, max: 2 }
  - loop: "0300"
    usage: M
    max: 10
    body:
      - { seg: S5, usage: M }
      - { seg: G62, usage: M }
      - loop: "0310"
        usage: M
        body:
          - { seg: N1, usage: M }
          - { seg: N3, usage: O }
          - { seg: N4, usage: O }
  - { seg: L3, usage: M }
  - { seg: SE, usage: M }
`

type fixtureKit struct {
	zone *time.Location
}

func (k fixtureKit) Offer(_ *partner.Definition, seed uint64) (Offer, error) {
	load := k.load(seed)
	return Offer{Label: "tender " + load.Number, Document: tenderOf(load), At: load.TenderedAt}, nil
}

func (k fixtureKit) load(seed uint64) *partner.Load {
	start := time.Date(2026, 3, 2, 8, 0, 0, 0, k.zone)
	party := func(code, name, street, city, state, zip string) partner.Party {
		return partner.Party{
			Code: code, Name: name, Street: street, City: city, State: state, Zip: zip,
			Country: "US", Zone: "America/Chicago",
		}
	}
	return &partner.Load{
		Seed:       seed,
		Number:     fmt.Sprintf("TRN-%04d", seed),
		TenderedAt: start.Add(-24 * time.Hour),
		BOL:        fmt.Sprintf("BOL%06d", seed),
		Stops: []partner.Stop{
			{
				Sequence: 1,
				Kind:     partner.StopPickup,
				Party:    party("DC-1", "Origin DC", "100 Main St", "Dallas", "TX", "75201"),
				Window:   partner.Window{Start: start, End: start.Add(2 * time.Hour)},
				Weight:   42_000,
				Pieces:   20,
				Lines: []partner.Line{
					{Number: 1, Description: "Valves", Weight: 42_000, Pieces: 20},
				},
			},
			{
				Sequence: 2,
				Kind:     partner.StopDelivery,
				Party:    party("ST-7", "Destination", "5 Elm St", "Chicago", "IL", "60601"),
				Window: partner.Window{
					Start: start.Add(30 * time.Hour),
					End:   start.Add(32 * time.Hour),
				},
				Weight: 42_000,
				Pieces: 20,
			},
		},
		Miles:      920,
		RateCents:  250,
		TotalCents: 230_000 + int64(seed),
	}
}

func (k fixtureKit) Converse(
	def *partner.Definition,
	_ []byte,
	seed uint64,
) (Conversation, error) {
	load := k.load(seed)
	key := proNumber(seed)
	at := load.TenderedAt.Add(time.Hour)
	controls := partner.Controls{
		Interchange: partner.NewCounter(int64(seed) * 10),
		Group:       partner.NewCounter(int64(seed) * 10),
	}
	segment := func(id string, values ...string) partner.Segment {
		elements := make([][]string, 0, len(values))
		for _, value := range values {
			elements = append(elements, []string{value})
		}
		return partner.Segment{ID: id, Elements: elements}
	}
	transactions := []struct {
		name string
		tx   partner.Transaction
	}{
		{"990 in accept", partner.Transaction{Set: "990", Segments: []partner.Segment{
			segment("B1", "SWPF", key, at.Format("20060102"), "A"),
		}}},
		{"214 in arrived", partner.Transaction{Set: "214", Segments: []partner.Segment{
			segment("B10", "PRO-9", key, "SWPF"),
			segment("LX", "1"),
			segment("AT7", "X1", "NS", "", "", "20260303", "1430", "CT"),
			segment("MS1", "CHICAGO", "IL", "US"),
		}}},
	}
	replies := make([]Reply, 0, len(transactions))
	for _, item := range transactions {
		written, err := def.Partner.Writer(controls).Write(at, []partner.Transaction{item.tx})
		if err != nil {
			return Conversation{}, err
		}
		replies = append(replies, Reply{
			Name: item.name,
			Set:  item.tx.Set,
			X12:  written[0].Bytes,
			At:   at,
			Want: WantApplied,
		})
	}
	return Conversation{Label: "accept and arrive", Replies: replies}, nil
}

func writeCarrierFixture(t *testing.T) string {
	t.Helper()
	starter, err := lading.Default().Starter(context.Background(), &lading.StarterRequest{
		Name:  "swift",
		Label: "Swift Prairie Freight",
		Role:  lading.RoleBroker,
		Them:  &[2]string{"ZZ", "SWIFTPF"},
		Us:    &[2]string{"ZZ", "TRENOVA"},
		Group: &[2]string{"SWIFTPF", "TRENOVA"},
		Zone:  "America/Chicago",
		Sets:  []string{"204", "990", "214"},
	})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	dir := filepath.Join(root, "swift")
	write := func(path, content string) {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("partner.yaml", fixturePartner)
	write("guide/204.yaml", fixtureGuide)
	for _, file := range starter.Files {
		write(filepath.Join("lading", file.Path), file.Source)
	}
	for _, sample := range starter.Samples {
		write(filepath.Join("lading", sample.Path), sample.Content)
	}
	return root
}

func TestACarrierPartnerIsTenderedAndReadBack(t *testing.T) {
	t.Parallel()

	zone, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	root := writeCarrierFixture(t)
	report := run(t, Options{Root: root, Count: 2, Workers: 1, Kit: fixtureKit{zone: zone}})
	p := report.Partners[0]
	if p.Role != string(partner.RoleCarrier) || len(p.Failures) > 0 {
		t.Fatalf("project: %+v", p)
	}
	for _, c := range p.Cases {
		var b strings.Builder
		c.Write(&b)
		if len(c.Failures) > 0 {
			t.Fatalf("seed %d failed:\n%s", c.Seed, b.String())
		}
		names := make([]string, 0, len(c.Steps))
		for _, s := range c.Steps {
			names = append(names, s.Name+"="+s.Status)
		}
		want := "204 out=sent,990 in accept=read,214 in arrived=read"
		if got := strings.Join(names, ","); got != want {
			t.Fatalf("seed %d steps %s, want %s\n%s", c.Seed, got, want, b.String())
		}
		if ack := c.Steps[0].Ack; ack == nil || ack.Code != acknowledge.Accepted {
			t.Fatalf("seed %d: the carrier's 997 for the 204: %+v", c.Seed, ack)
		}
	}
}

func TestACarrierTenderBreakingItsGuideFails(t *testing.T) {
	t.Parallel()

	zone, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	root := writeCarrierFixture(t)
	edit(t, filepath.Join(root, "swift/guide/204.yaml"),
		"{ pos: 4, type: AN, min: 1, max: 30, usage: M }",
		"{ pos: 4, type: AN, min: 1, max: 8, usage: M }")
	report := run(t, Options{Root: root, Count: 1, Workers: 1, Kit: fixtureKit{zone: zone}})
	failures := failuresOf(report, "204 out")
	if len(failures) == 0 || failures[0].Rule != "B2-04 max 8" {
		t.Fatalf("want B2-04 max 8, got %+v", failures)
	}
}

func tenderOf(load *partner.Load) loadTender {
	document := loadTender{
		Key:     proNumber(load.Seed),
		Purpose: "Original",
		BOL:     load.BOL,
		Weight:  pounds(load.Weight()),
		Pieces:  count(load.Pieces()),
	}
	if load.TotalCents > 0 {
		total := cents(load.TotalCents)
		document.Total = &total
	}
	if load.Equipment.Reefer {
		temp := &measure{Value: strconv.Itoa(load.Equipment.TempF), Unit: unitFahrenheit}
		document.Temperature = &temperatureRange{Min: temp, Max: temp}
	}
	for index := range load.Stops {
		stop := &load.Stops[index]
		kind := "Pickup"
		if stop.Kind == partner.StopDelivery {
			kind = "Delivery"
		}
		end := at(stop.Window.End, stop.Party.Zone)
		written := tenderStop{
			Sequence: int64(stop.Sequence),
			Type:     kind,
			Location: &ref{
				ID:    trenovaID("loc", load.Seed*100+uint64(index)),
				Code:  stop.Party.Code,
				Label: stop.Party.Name,
			},
			Name:   stop.Party.Name,
			Code:   stop.Party.Code,
			Window: window{Start: at(stop.Window.Start, stop.Party.Zone), End: &end},
			Weight: pounds(stop.Weight),
			Pieces: count(stop.Pieces),
		}
		if stop.Party.Street != "" && stop.Party.City != "" {
			written.Address = &address{
				Line1:  stop.Party.Street,
				City:   stop.Party.City,
				State:  stop.Party.State,
				Postal: stop.Party.Zip,
			}
		}
		document.Stops = append(document.Stops, written)
		for _, line := range stop.Lines {
			document.Commodities = append(document.Commodities, commodityLine{
				Description: stringutils.FirstNonEmpty(line.Description, "Freight"),
				Weight:      pounds(line.Weight),
				Pieces:      count(line.Pieces),
			})
		}
	}
	return document
}
