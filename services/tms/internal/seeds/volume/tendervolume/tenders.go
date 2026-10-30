package tendervolume

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	tvChunk              = 2_000
	tvTenderedCarrier    = 0.75
	tvDriverRetender     = 0.03
	tvCanceledTendered   = 0.25
	tvPriorExhausted     = 0.08
	tvLiveEscalated      = 0.15
	tvGuideLaneThreshold = 3
	tvTableTenders       = "tenders"
	tvTableOffers        = "tender_offers"
	tvTableTokens        = "tender_offer_tokens"
	tvTableGuides        = "routing_guides"
	tvTableEntries       = "routing_guide_entries"
)

type tvKind int

const (
	tvKindAccepted tvKind = iota
	tvKindLive
	tvKindExhausted
	tvKindCovered
	tvKindShipmentCanceled
)

type tvPlan struct {
	rec  *volumegen.ShipmentRecord
	move *volumegen.MoveRecord
	kind tvKind
	lane string
}

type tvCarrier struct {
	id      pulid.ID
	email   string
	partner *pulid.ID
}

type tvBuilder struct {
	g        *volumegen.Generator
	now      int64
	action   dispatchcontrol.TenderExhaustedAction
	carriers []*tvCarrier
	byID     map[pulid.ID]*tvCarrier
	guides   map[string]*tvGuide
	batch    *tvBatch
	statuses map[tender.Status]int
	counts   map[string]int
}

type tvBatch struct {
	tenders []*tender.Tender
	offers  []*tender.TenderOffer
	tokens  []*tender.TenderOfferToken
	stamps  map[string][]volumegen.Stamp
}

func newBatch() *tvBatch {
	return &tvBatch{stamps: make(map[string][]volumegen.Stamp)}
}

func (b *tvBatch) stamp(table string, id pulid.ID, created, updated int64) {
	b.stamps[table] = append(b.stamps[table], volumegen.Stamp{
		Id:      id,
		Created: created,
		Updated: max(updated, created),
	})
}

type tvPhase struct {
	*volumegen.Generator
}

func Seed(ctx context.Context, g *volumegen.Generator) error {
	return (&tvPhase{Generator: g}).seed(ctx)
}

func (g *tvPhase) seed(ctx context.Context) error {
	if len(g.Carriers) == 0 {
		return nil
	}
	control, err := g.DispatchControl(ctx)
	if err != nil {
		return err
	}
	b := &tvBuilder{
		g:        g.Generator,
		now:      volumegen.Unix(g.Now),
		action:   dispatchcontrol.TenderExhaustedActionNotify,
		byID:     make(map[pulid.ID]*tvCarrier),
		guides:   make(map[string]*tvGuide),
		statuses: make(map[tender.Status]int),
		counts:   make(map[string]int),
	}
	if control != nil && control.TenderExhaustedAction != "" {
		b.action = control.TenderExhaustedAction
	}
	if err = b.loadCarriers(ctx); err != nil {
		return err
	}
	if len(b.carriers) < 2 {
		return nil
	}

	plans := b.plan()
	guides := b.buildGuides(plans)
	if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		return b.writeGuides(ctx, tx, guides)
	}); err != nil {
		return err
	}

	for begin := 0; begin < len(plans); begin += tvChunk {
		end := min(begin+tvChunk, len(plans))
		b.batch = newBatch()
		for _, p := range plans[begin:end] {
			b.build(p)
		}
		if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
			return b.write(ctx, tx)
		}); err != nil {
			return err
		}
		g.Progress("tenders", end, len(plans))
	}

	for _, entity := range []struct{ table, label string }{
		{tvTableGuides, "Routing guides"},
		{tvTableEntries, "Routing guide entries"},
		{tvTableTenders, "Tenders"},
		{tvTableOffers, "Tender offers"},
		{tvTableTokens, "Tender offer tokens"},
	} {
		g.Report.Add(entity.label, b.counts[entity.table])
	}
	for _, status := range []tender.Status{
		tender.StatusActive, tender.StatusNeedsReview, tender.StatusAccepted,
		tender.StatusExhausted, tender.StatusCanceled,
	} {
		g.Opts.Logf("  • tenders %s: %d", status, b.statuses[status])
	}

	return nil
}

func (b *tvBuilder) loadCarriers(ctx context.Context) error {
	c := carrierdb.Carriers
	rows, err := c.Pick(c.ID, c.Name, c.Email).
		Where(c.ID.In(b.g.Carriers...)).
		Order(c.ID.Asc()).
		All(ctx, b.g.DB)
	if err != nil {
		return fmt.Errorf("load carriers: %w", err)
	}
	ch := carrierdb.CarrierEDIChannels
	channels, err := ch.Pick(ch.CarrierID, ch.EDIPartnerID).
		Where(ch.IsDefault.IsTrue(), ch.CarrierID.In(b.g.Carriers...)).
		All(ctx, b.g.DB)
	if err != nil {
		return fmt.Errorf("load carrier edi channels: %w", err)
	}
	partners := make(map[pulid.ID]pulid.ID, len(channels))
	for _, channel := range channels {
		partners[channel.CarrierID] = channel.EDIPartnerID
	}

	for _, row := range rows {
		email := strings.TrimSpace(row.Email)
		if email == "" {
			email = "dispatch@" + volumegen.Slug(row.Name) + ".com"
		}
		tc := &tvCarrier{id: row.ID, email: strings.ToLower(email)}
		if partner, ok := partners[row.ID]; ok {
			tc.partner = volumegen.Ptr(partner)
		}
		b.carriers = append(b.carriers, tc)
		b.byID[row.ID] = tc
	}

	return nil
}

func (b *tvBuilder) plan() []*tvPlan {
	f := b.g.Fake
	plans := make([]*tvPlan, 0, len(b.g.Shipments)/5)
	for _, rec := range b.g.Shipments {
		for _, move := range rec.Moves {
			if len(move.Stops) == 0 {
				continue
			}
			p := &tvPlan{rec: rec, move: move, lane: laneOf(move)}
			switch {
			case rec.Status == shipmentvocab.StatusCanceled:
				if !f.Chance(tvCanceledTendered) {
					continue
				}
				p.kind = tvKindShipmentCanceled
			case move.CarrierAssignment != nil:
				if !f.Chance(tvTenderedCarrier) {
					continue
				}
				p.kind = tvKindAccepted
			case move.Crew != nil:
				if !f.Chance(tvDriverRetender) {
					continue
				}
				p.kind = tvKindCovered
			default:
				kind, ok := b.openKind(move)
				if !ok {
					continue
				}
				p.kind = kind
			}
			plans = append(plans, p)
		}
	}

	return plans
}

func (b *tvBuilder) openKind(move *volumegen.MoveRecord) (tvKind, bool) {
	if move.Status != shipmentvocab.MoveStatusNew || move.Stops[0].Arrived() {
		return 0, false
	}
	f := b.g.Fake
	lead := move.Stops[0].WindowStart - b.now
	switch {
	case lead < -volumegen.Day:
		return tvKindExhausted, f.Chance(0.3)
	case lead < 0:
		return 0, false
	case lead <= 3*volumegen.Day:
		switch f.Weighted([]int{45, 14, 41}) {
		case 0:
			return tvKindLive, true
		case 1:
			return tvKindExhausted, true
		default:
			return 0, false
		}
	case lead <= 10*volumegen.Day:
		switch f.Weighted([]int{22, 6, 72}) {
		case 0:
			return tvKindLive, true
		case 1:
			return tvKindExhausted, true
		default:
			return 0, false
		}
	default:
		return tvKindLive, f.Chance(0.05)
	}
}

func laneOf(move *volumegen.MoveRecord) string {
	first, last := move.Stops[0], move.Stops[len(move.Stops)-1]
	return first.Location.City.State + "|" + last.Location.City.State
}

func (b *tvBuilder) build(p *tvPlan) {
	f := b.g.Fake
	switch p.kind {
	case tvKindAccepted:
		ca := p.move.CarrierAssignment
		var from *tender.Tender
		if f.Chance(tvPriorExhausted) {
			from = b.priorExhausted(p, ca.CreatedAt)
		}
		b.accepted(p, ca, from)
	case tvKindLive:
		var from *tender.Tender
		if f.Chance(tvLiveEscalated) {
			from = b.priorExhausted(p, b.now-f.Between64(30*volumegen.Minute, 6*volumegen.Hour))
		}
		b.live(p, from)
	case tvKindExhausted:
		end := max(
			b.now-f.Between64(20*volumegen.Minute, 5*volumegen.Day),
			p.rec.CreatedAt+f.Between64(3*volumegen.Hour, 30*volumegen.Hour),
		)
		b.exhaustedTender(p, min(end, b.now-10*volumegen.Minute))
	case tvKindCovered:
		at := p.rec.CreatedAt + f.Between64(10*volumegen.Minute, 3*volumegen.Hour)
		b.canceled(p, min(at, b.now-volumegen.Hour), "Covered with a company driver")
	case tvKindShipmentCanceled:
		at := p.rec.CreatedAt + f.Between64(10*volumegen.Minute, 2*volumegen.Hour)
		b.canceled(p, min(at, b.now-volumegen.Hour), "Shipment canceled by the customer")
	}
}

func (b *tvBuilder) newTender(p *tvPlan, mode tender.Mode, created int64) *tender.Tender {
	t := &tender.Tender{
		ID:             volumegen.IDAt("tnd_", created),
		BusinessUnitID: b.g.Refs.BuID,
		OrganizationID: b.g.Refs.OrgID,
		ShipmentID:     p.rec.ID,
		ShipmentMoveID: p.move.ID,
		Mode:           mode,
		Status:         tender.StatusActive,
		CreatedByID:    volumegen.Ptr(b.g.PickUser(b.g.Refs.UserIDs)),
		Version:        1,
		CreatedAt:      created,
	}
	t.WorkflowID = workflowID(t.ID, 0)

	return t
}

func (b *tvBuilder) escalate(t, from *tender.Tender) {
	if from == nil {
		return
	}
	switch b.action {
	case dispatchcontrol.TenderExhaustedActionRebroadcast,
		dispatchcontrol.TenderExhaustedActionRewaterfall:
		t.EscalatedFromID = volumegen.Ptr(from.ID)
		t.EscalationRound = 1
		t.WorkflowID = workflowID(t.ID, 1)
	case dispatchcontrol.TenderExhaustedActionNotify:
	}
}

func (b *tvBuilder) finish(t *tender.Tender, offers []*tender.TenderOffer, updated int64) {
	for _, offer := range offers {
		if offer.Rank > t.CurrentRank && offer.SentAt != nil {
			t.CurrentRank = offer.Rank
		}
		offerUpdated := offer.CreatedAt
		for _, at := range []*int64{
			offer.SentAt, offer.RespondedAt, offer.CounteredAt, offer.RevokedAt,
		} {
			if at != nil {
				offerUpdated = max(offerUpdated, *at)
			}
		}
		if offer.Status == tender.OfferStatusExpired && offer.ExpiresAt != nil {
			offerUpdated = max(offerUpdated, *offer.ExpiresAt)
		}
		switch offer.Status {
		case tender.OfferStatusSkipped, tender.OfferStatusSuperseded, tender.OfferStatusWithdrawn:
			offerUpdated = max(offerUpdated, min(updated, b.now))
		case tender.OfferStatusPending, tender.OfferStatusSent, tender.OfferStatusAccepted,
			tender.OfferStatusDeclined, tender.OfferStatusExpired, tender.OfferStatusDeliveryFailed,
			tender.OfferStatusCountered, tender.OfferStatusCounterRejected, tender.OfferStatusRevoked:
		}
		b.batch.offers = append(b.batch.offers, offer)
		b.batch.stamp(tvTableOffers, offer.ID, offer.CreatedAt, min(offerUpdated, b.now))
		b.token(offer, updated)
	}
	b.batch.tenders = append(b.batch.tenders, t)
	b.batch.stamp(tvTableTenders, t.ID, t.CreatedAt, min(updated, b.now))
	b.statuses[t.Status]++
}

func (b *tvBuilder) rateFor(miles float64) (shipmentvocab.CarrierRateMethod, decimal.Decimal) {
	f := b.g.Fake
	rpm := 1.7 + 120/(miles+40) + f.Float()*0.35
	if f.Chance(0.4) && miles >= 150 {
		return shipmentvocab.CarrierRateMethodPerMile, decimal.NewFromFloat(rpm).Round(2)
	}
	return shipmentvocab.CarrierRateMethodFlat,
		decimal.NewFromFloat(math.Round(math.Max(miles*rpm, 300)/5) * 5)
}

func (b *tvBuilder) write(ctx context.Context, tx *keel.DB) error {
	batch := b.batch
	if err := volumegen.InsertRows(ctx, tx, tenderdb.Tenders.Handle, batch.tenders); err != nil {
		return err
	}
	if err := volumegen.InsertRows(
		ctx,
		tx,
		tenderdb.TenderOffers.Handle,
		batch.offers,
	); err != nil {
		return err
	}
	if err := volumegen.InsertRows(
		ctx,
		tx,
		tenderdb.TenderOfferTokens.Handle,
		batch.tokens,
	); err != nil {
		return err
	}
	if err := volumegen.Backdate(
		ctx,
		tx,
		tenderdb.Tenders.Handle,
		batch.stamps[tvTableTenders],
	); err != nil {
		return err
	}
	if err := volumegen.Backdate(
		ctx,
		tx,
		tenderdb.TenderOffers.Handle,
		batch.stamps[tvTableOffers],
	); err != nil {
		return err
	}
	if err := volumegen.Backdate(
		ctx,
		tx,
		tenderdb.TenderOfferTokens.Handle,
		batch.stamps[tvTableTokens],
	); err != nil {
		return err
	}
	b.counts[tvTableTenders] += len(batch.tenders)
	b.counts[tvTableOffers] += len(batch.offers)
	b.counts[tvTableTokens] += len(batch.tokens)

	return nil
}

func moveMiles(move *volumegen.MoveRecord) float64 {
	if move.Miles > 0 {
		return move.Miles
	}
	return 250
}
