package tendervolume

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type tvGuide struct {
	guide    *tender.RoutingGuide
	entries  []*tender.RoutingGuideEntry
	carriers []*tvCarrier
}

func (g *tvGuide) drafts() []tvDraft {
	out := make([]tvDraft, len(g.entries))
	for i, entry := range g.entries {
		out[i] = tvDraft{
			carrier: g.carriers[i],
			method:  entry.RateMethod,
			rate:    entry.Rate,
			ttl:     entry.OfferTTLSeconds,
			channel: entry.Channel,
		}
	}

	return out
}

func (g *tvGuide) rankOf(carrierID pulid.ID) int {
	for i, c := range g.carriers {
		if c.id == carrierID {
			return i
		}
	}

	return -1
}

type tvLane struct {
	key      string
	count    int
	miles    float64
	carriers []pulid.ID
}

func (b *tvBuilder) buildGuides(plans []*tvPlan) []*tvGuide {
	byKey := make(map[string]*tvLane)
	for _, p := range plans {
		lane, ok := byKey[p.lane]
		if !ok {
			lane = &tvLane{key: p.lane}
			byKey[p.lane] = lane
		}
		lane.count++
		lane.miles += moveMiles(p.move)
		if ca := p.move.CarrierAssignment; ca != nil &&
			!slices.Contains(lane.carriers, ca.CarrierID) {
			lane.carriers = append(lane.carriers, ca.CarrierID)
		}
	}

	lanes := make([]*tvLane, 0, len(byKey))
	for _, lane := range byKey {
		if lane.count >= tvGuideLaneThreshold {
			lanes = append(lanes, lane)
		}
	}
	sort.Slice(lanes, func(i, j int) bool {
		if lanes[i].count != lanes[j].count {
			return lanes[i].count > lanes[j].count
		}
		return lanes[i].key < lanes[j].key
	})
	lanes = lanes[:min(len(lanes), volumegen.Clamp(len(b.carriers)/2, 4, 150))]

	f := b.g.Fake
	guides := make([]*tvGuide, 0, len(lanes))
	for _, lane := range lanes {
		origin, dest, _ := strings.Cut(lane.key, "|")
		created := volumegen.Unix(b.g.Start) - f.Between64(5, 120)*volumegen.Day
		guide := &tender.RoutingGuide{
			ID:               volumegen.IDAt("rg_", created),
			BusinessUnitID:   b.g.Refs.BuID,
			OrganizationID:   b.g.Refs.OrgID,
			Name:             fmt.Sprintf("%s to %s", origin, dest),
			Description:      "Contracted carriers for this state lane, in award order",
			Status:           domaintypes.StatusActive,
			OriginState:      origin,
			DestinationState: dest,
			Specificity:      tender.SpecificityStatePair,
			Version:          1,
			CreatedAt:        created,
		}
		tg := &tvGuide{guide: guide}
		miles := lane.miles / float64(lane.count)
		picked := make([]*tvCarrier, 0, 5)
		for _, id := range lane.carriers {
			if c, ok := b.byID[id]; ok && len(picked) < 2 && f.Chance(0.7) {
				picked = append(picked, c)
			}
		}
		want := f.Between(3, 5)
		for try := 0; len(picked) < want && try < want*6; try++ {
			c := volumegen.Pick(f, b.carriers)
			if !slices.ContainsFunc(picked, func(p *tvCarrier) bool { return p.id == c.id }) {
				picked = append(picked, c)
			}
		}
		f.Shuffle(len(picked), func(i, j int) { picked[i], picked[j] = picked[j], picked[i] })
		for rank, c := range picked {
			d := b.draftFor(c, miles)
			entry := &tender.RoutingGuideEntry{
				ID:              volumegen.IDAt("rge_", created),
				BusinessUnitID:  guide.BusinessUnitID,
				OrganizationID:  guide.OrganizationID,
				RoutingGuideID:  guide.ID,
				CarrierID:       c.id,
				Rank:            int16(rank + 1), //nolint:gosec // a handful of ranks
				RateMethod:      d.method,
				Rate:            d.rate,
				OfferTTLSeconds: d.ttl,
				Channel:         d.channel,
				Version:         1,
				CreatedAt:       created,
			}
			tg.entries = append(tg.entries, entry)
			tg.carriers = append(tg.carriers, c)
		}
		b.guides[lane.key] = tg
		guides = append(guides, tg)
	}

	return guides
}

func (b *tvBuilder) writeGuides(ctx context.Context, tx *keel.DB, guides []*tvGuide) error {
	if len(guides) == 0 {
		return nil
	}
	rows := make([]*tender.RoutingGuide, 0, len(guides))
	entries := make([]*tender.RoutingGuideEntry, 0, len(guides)*4)
	guideStamps := make([]volumegen.Stamp, 0, len(guides))
	entryStamps := make([]volumegen.Stamp, 0, len(guides)*4)
	for _, g := range guides {
		rows = append(rows, g.guide)
		guideStamps = append(guideStamps, volumegen.Stamp{
			Id:      g.guide.ID,
			Created: g.guide.CreatedAt,
			Updated: g.guide.CreatedAt,
		})
		for _, entry := range g.entries {
			entries = append(entries, entry)
			entryStamps = append(entryStamps, volumegen.Stamp{
				Id:      entry.ID,
				Created: entry.CreatedAt,
				Updated: entry.CreatedAt,
			})
		}
	}

	if err := volumegen.InsertRows(ctx, tx, tenderdb.RoutingGuides.Handle, rows); err != nil {
		return err
	}
	if err := volumegen.InsertRows(
		ctx,
		tx,
		tenderdb.RoutingGuideEntries.Handle,
		entries,
	); err != nil {
		return err
	}
	if err := volumegen.Backdate(ctx, tx, tenderdb.RoutingGuides.Handle, guideStamps); err != nil {
		return err
	}
	if err := volumegen.Backdate(
		ctx,
		tx,
		tenderdb.RoutingGuideEntries.Handle,
		entryStamps,
	); err != nil {
		return err
	}
	b.counts[tvTableGuides] += len(rows)
	b.counts[tvTableEntries] += len(entries)

	return nil
}
