package conform

import (
	"context"
	"hash/fnv"
	"math/rand/v2"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/spoken"
	"github.com/emoss08/trenova/pkg/lading"
)

const (
	tenderSet   = "204"
	responseSet = "990"
	statusSet   = "214"
	invoiceSet  = "210"
	respondIn   = 20 * time.Minute
)

func Variation(def *partner.Definition, seed uint64) (partner.Variation, bool) {
	template, ok := def.Templates[tenderSet]
	if !ok {
		return partner.Variation{}, false
	}
	kinds := make([]partner.Kind, 0, len(template.Purposes))
	for _, kind := range partner.Kinds {
		if _, sent := template.Purposes[kind]; sent {
			kinds = append(kinds, kind)
		}
	}
	rng := rand.New( //nolint:gosec // a seeded generator: the same seed must replay the same case
		rand.NewPCG(seed, hash64("conform "+def.Name())),
	)
	variation := partner.Variation{
		Kind:   kinds[seed%uint64(len(kinds))],
		Sparse: rng.IntN(4) == 0,
	}
	if variation.Kind != partner.KindChange && variation.Kind != partner.KindReplace {
		return variation, true
	}
	switch rng.IntN(4) {
	case 0:
		variation.Delta.RatePercent = float64(rng.IntN(31) - 10)
	case 1:
		variation.Delta.AddStop = true
	case 2:
		variation.Delta.Move = &partner.Move{By: time.Duration(1+rng.IntN(24)) * time.Hour}
	}
	return variation, true
}

func (r *runner) shipper(ctx context.Context, s *subject, c *Case) {
	step := c.step("204 in", "Read "+possessive(s.def.Partner.Display)+" 204",
		directionInbound, tenderSet)
	variation, ok := Variation(s.def, c.Seed)
	if !ok {
		c.fail(step, &Failure{
			Source:  SourceHarness,
			Rule:    "generate/204.yaml",
			Message: "a shipper partner needs a 204 template to tender from",
		})
		return
	}
	c.Variation, c.Title = variation.String(), spoken.Variation(variation)
	controls := partner.Controls{
		Interchange: partner.NewCounter(int64(c.Seed % 999_999_999)),
		Group:       partner.NewCounter(int64(c.Seed % 999_999_999)),
	}
	written, load, err := s.def.Interchange(tenderSet, c.Seed, variation, controls)
	if err != nil {
		c.fail(step, &Failure{Source: SourceHarness, Rule: "generate", Message: err.Error()})
		return
	}
	sentAt := variation.SentAt(load)
	transaction := r.read(ctx, s, c, step, message{
		text:   string(written.Bytes),
		now:    sentAt,
		answer: loadLookups(load),
	})
	if transaction == nil || step.Status == StatusFailed {
		return
	}
	var tender readTender
	if err = sonic.Unmarshal(transaction.Document, &tender); err != nil {
		c.fail(step, &Failure{Source: SourceHarness, Rule: "document", Message: err.Error()})
		return
	}
	if tender.Purpose == "Cancel" {
		step.Note = "a cancellation: Trenova writes nothing back"
		return
	}
	r.answer(ctx, s, c, load, &tender, sentAt)
}

func (r *runner) answer(
	ctx context.Context,
	s *subject,
	c *Case,
	load *partner.Load,
	tender *readTender,
	sentAt time.Time,
) {
	zone := s.def.Partner.Zone
	var plan []namedDocument
	add := func(set string, docs ...namedDocument) {
		if s.outboundMap(set) == "" {
			return
		}
		for _, doc := range docs {
			doc.set = set
			plan = append(plan, doc)
		}
	}
	v := newVariety(s.def.Name(), s.guides, c.Seed)
	built := v.shipment(tender)
	add(responseSet, responses(tender.Key, sentAt.Add(respondIn), zone, v)...)
	add(statusSet, statuses(load, built, s.sendsOn(statusSet), v)...)
	add(invoiceSet, invoice(load, built, s.def.Partner.Location(), v))
	var index uint64
	for i := range plan {
		r.write(ctx, s, c, plan[i], controlFor(c.Seed, index))
		index++
	}
}

func controlFor(seed, index uint64) lading.Control {
	base := (seed*16 + index + 1) % 999_999_999
	return lading.Control{Interchange: base, Group: base, Transaction: index + 1}
}

func loadLookups(load *partner.Load) answerFunc {
	found := map[lading.Need]*lading.Found{}
	put := func(kind, code, label, zone string) {
		if code == "" {
			return
		}
		found[lading.Need{Kind: kind, Code: code}] = &lading.Found{
			ID:    kind + "-" + code,
			Label: label,
			Zone:  zone,
		}
	}
	put("Customer", load.BillTo.Code, load.BillTo.Name, load.BillTo.Zone)
	for index := range load.Stops {
		stop := &load.Stops[index]
		put("Location", stop.Party.Code, stop.Party.Name, stop.Party.Zone)
		for _, line := range stop.Lines {
			put("Commodity", line.Code, line.Description, "")
		}
	}
	return func(need lading.Need) *lading.Found {
		return found[need]
	}
}

func hash64(text string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(text))
	return h.Sum64()
}
