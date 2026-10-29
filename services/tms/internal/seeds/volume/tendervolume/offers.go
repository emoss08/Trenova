package tendervolume

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/tenderjobs/tendercontract"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	tvMinGap         = 5 * volumegen.Minute
	tvDeclineRate    = 0.55
	tvDeliveryFailed = 0.02
	tvCounterShare   = 0.2
)

var (
	tvTTLs        = []int32{900, 1800, 3600, 3600, 7200, 7200, 14400}
	tvDeclineWhys = []string{
		"No trucks available in the area",
		"Rate too low for the lane",
		"Cannot make the pickup appointment",
		"Equipment not available",
		"Driver out of hours",
		"Lane is outside our network",
	}
	tvDeliveryErrors = []string{
		"Mailbox unavailable: 550 5.1.1 recipient address rejected",
		"Connection to the carrier's mail server timed out",
	}
)

type tvDraft struct {
	carrier *tvCarrier
	method  shipmentvocab.CarrierRateMethod
	rate    decimal.Decimal
	ttl     int32
	channel tender.Channel
}

type tvLayout struct {
	mode   tender.Mode
	guide  *tvGuide
	drafts []tvDraft
}

func workflowID(id pulid.ID, round int16) string {
	return tendercontract.WorkflowID(id, round)
}

func (b *tvBuilder) spotDrafts(p *tvPlan, n int, exclude pulid.ID) []tvDraft {
	f := b.g.Fake
	taken := map[pulid.ID]struct{}{exclude: {}}
	drafts := make([]tvDraft, 0, n)
	for try := 0; len(drafts) < n && try < n*6; try++ {
		c := volumegen.Pick(f, b.carriers)
		if _, ok := taken[c.id]; ok {
			continue
		}
		taken[c.id] = struct{}{}
		drafts = append(drafts, b.draftFor(c, moveMiles(p.move)))
	}

	return drafts
}

func (b *tvBuilder) draftFor(c *tvCarrier, miles float64) tvDraft {
	f := b.g.Fake
	method, rate := b.rateFor(miles)
	d := tvDraft{
		carrier: c,
		method:  method,
		rate:    rate,
		ttl:     volumegen.Pick(f, tvTTLs),
		channel: tender.ChannelEmail,
	}
	if c.partner != nil && f.Chance(0.6) {
		d.channel = tender.ChannelEDI
	}

	return d
}

func (b *tvBuilder) layout(p *tvPlan, n int, broadcast bool) tvLayout {
	f := b.g.Fake
	if guide, ok := b.guides[p.lane]; ok && !broadcast && f.Chance(0.7) {
		return tvLayout{mode: tender.ModeWaterfall, guide: guide, drafts: guide.drafts()}
	}
	mode := tender.ModeSpotSequential
	if broadcast {
		mode = tender.ModeSpotBroadcast
	}
	return tvLayout{mode: mode, drafts: b.spotDrafts(p, n, pulid.Nil)}
}

func (b *tvBuilder) wantsBroadcast(from *tender.Tender) bool {
	if from != nil && b.action == dispatchcontrol.TenderExhaustedActionRebroadcast {
		return true
	}
	return b.g.Fake.Chance(0.3)
}

func (b *tvBuilder) newOffer(t *tender.Tender, rank int, d tvDraft) *tender.TenderOffer {
	offer := &tender.TenderOffer{
		ID:              volumegen.IDAt("tof_", t.CreatedAt),
		BusinessUnitID:  t.BusinessUnitID,
		OrganizationID:  t.OrganizationID,
		TenderID:        t.ID,
		CarrierID:       d.carrier.id,
		Rank:            int16(rank + 1), //nolint:gosec // a handful of ranks
		RateMethod:      d.method,
		Rate:            d.rate,
		OfferTTLSeconds: d.ttl,
		Channel:         d.channel,
		Status:          tender.OfferStatusPending,
		Version:         1,
		CreatedAt:       t.CreatedAt,
	}
	if d.channel == tender.ChannelEDI {
		offer.EDIPartnerID = d.carrier.partner
	} else {
		offer.RecipientEmail = d.carrier.email
	}

	return offer
}

func (b *tvBuilder) send(offer *tender.TenderOffer, at int64) {
	offer.Status = tender.OfferStatusSent
	offer.SentAt = volumegen.Ptr(at)
	offer.ExpiresAt = volumegen.Ptr(at + int64(offer.OfferTTLSeconds))
}

func (b *tvBuilder) respond(offer *tender.TenderOffer, status tender.OfferStatus, at int64) {
	offer.Status = status
	offer.RespondedAt = volumegen.Ptr(at)
	offer.ResponseSource = tender.ResponseSourceEmail
	switch {
	case offer.Channel == tender.ChannelEDI:
		offer.ResponseSource = tender.ResponseSourceEDI
	case b.g.Fake.Chance(0.25):
		offer.ResponseSource = tender.ResponseSourceManual
	}
	if status == tender.OfferStatusDeclined {
		offer.DeclineReason = volumegen.Pick(b.g.Fake, tvDeclineWhys)
	}
}

func (b *tvBuilder) fail(offer *tender.TenderOffer) bool {
	if offer.Channel != tender.ChannelEmail || !b.g.Fake.Chance(tvDeliveryFailed) {
		return false
	}
	offer.Status = tender.OfferStatusDeliveryFailed
	offer.DeliveryError = volumegen.Pick(b.g.Fake, tvDeliveryErrors)
	return true
}

func (b *tvBuilder) closeOut(offer *tender.TenderOffer, sent int64) int64 {
	f := b.g.Fake
	b.send(offer, sent)
	if b.fail(offer) {
		return sent + volumegen.Minute
	}
	if f.Chance(tvDeclineRate) {
		at := sent + f.Between64(3*volumegen.Minute, int64(offer.OfferTTLSeconds)*9/10)
		b.respond(offer, tender.OfferStatusDeclined, at)
		return at
	}
	offer.Status = tender.OfferStatusExpired
	return *offer.ExpiresAt
}

func (b *tvBuilder) accepted(p *tvPlan, ca *shipment.CarrierAssignment, from *tender.Tender) {
	f := b.g.Fake
	assigned := ca.CreatedAt
	floor := p.rec.CreatedAt
	if from != nil && from.ExhaustedAt != nil {
		floor = max(floor, *from.ExhaustedAt)
	}
	if assigned-floor < 2*tvMinGap {
		return
	}
	carrier, ok := b.byID[ca.CarrierID]
	if !ok {
		carrier = &tvCarrier{id: ca.CarrierID, email: "dispatch@carrier.example.com"}
	}
	winner := b.draftFor(carrier, moveMiles(p.move))
	winner.method = ca.RateMethod
	winner.rate = ca.BaseRate

	broadcast := b.wantsBroadcast(from)
	lay := tvLayout{mode: tender.ModeSpotSequential}
	won := -1
	if guide, hasGuide := b.guides[p.lane]; hasGuide && !broadcast {
		if idx := guide.rankOf(ca.CarrierID); idx >= 0 {
			lay = tvLayout{mode: tender.ModeWaterfall, guide: guide, drafts: guide.drafts()}
			won = idx
		}
	}
	if won < 0 {
		n := 1 + f.Weighted([]int{50, 26, 15, 9})
		if broadcast {
			lay.mode = tender.ModeSpotBroadcast
			n = f.Between(3, 6)
		}
		others := b.spotDrafts(p, n-1, ca.CarrierID)
		won = f.Intn(len(others) + 1)
		if !broadcast {
			won = len(others)
			others = append(others, b.spotDrafts(p, f.Between(0, 2), ca.CarrierID)...)
		}
		lay.drafts = append(append(append([]tvDraft{}, others[:won]...), winner), others[won:]...)
	}
	lay.drafts[won].method = winner.method
	lay.drafts[won].rate = winner.rate

	if lay.mode == tender.ModeSpotBroadcast {
		b.acceptBroadcast(p, lay, won, assigned, floor, from)
		return
	}
	b.acceptSequential(p, lay, won, assigned, floor, from)
}

func (b *tvBuilder) acceptSequential(
	p *tvPlan,
	lay tvLayout,
	won int,
	assigned, floor int64,
	from *tender.Tender,
) {
	f := b.g.Fake
	sent := make([]int64, won+1)
	ends := make([]tvEnd, won+1)
	ttl := int64(lay.drafts[won].ttl)
	sent[won] = assigned - min(f.Between64(5*volumegen.Minute, min(ttl*8/10, 3*volumegen.Hour)),
		assigned-floor-volumegen.Minute)
	first := won
	for i := won - 1; i >= 0; i-- {
		d := lay.drafts[i]
		dur := int64(d.ttl)
		ends[i] = tvEnd{status: tender.OfferStatusExpired}
		if f.Chance(tvDeclineRate) {
			dur = f.Between64(3*volumegen.Minute, int64(d.ttl)*9/10)
			ends[i] = tvEnd{status: tender.OfferStatusDeclined, at: sent[i+1]}
		}
		if sent[i+1]-dur < floor {
			break
		}
		sent[i] = sent[i+1] - dur
		first = i
	}
	if first > 0 && lay.mode == tender.ModeWaterfall {
		lay = tvLayout{mode: tender.ModeSpotSequential, drafts: []tvDraft{lay.drafts[won]}}
		won, first = 0, 0
		sent = []int64{sent[len(sent)-1]}
		ends = []tvEnd{{}}
	}

	t := b.newTender(p, lay.mode, sent[first])
	b.attachGuide(t, lay)
	b.escalate(t, from)
	offers := make([]*tender.TenderOffer, 0, len(lay.drafts)-first)
	for i := first; i < len(lay.drafts); i++ {
		offer := b.newOffer(t, i-first, lay.drafts[i])
		switch {
		case i < won:
			b.send(offer, sent[i])
			if ends[i].status == tender.OfferStatusDeclined {
				b.respond(offer, tender.OfferStatusDeclined, ends[i].at)
			} else {
				offer.Status = tender.OfferStatusExpired
			}
		case i == won:
			b.send(offer, sent[i])
			b.respond(offer, tender.OfferStatusAccepted, assigned)
			t.AcceptedOfferID = volumegen.Ptr(offer.ID)
		default:
			offer.Status = tender.OfferStatusSkipped
		}
		offers = append(offers, offer)
	}
	t.Status = tender.StatusAccepted
	t.AcceptedAt = volumegen.Ptr(assigned)
	b.finish(t, offers, assigned)
}

type tvEnd struct {
	status tender.OfferStatus
	at     int64
}

func (b *tvBuilder) acceptBroadcast(
	p *tvPlan,
	lay tvLayout,
	won int,
	assigned, floor int64,
	from *tender.Tender,
) {
	f := b.g.Fake
	start := max(assigned-f.Between64(5*volumegen.Minute, 4*volumegen.Hour), floor)
	t := b.newTender(p, tender.ModeSpotBroadcast, start)
	b.escalate(t, from)
	window := assigned - start
	offers := make([]*tender.TenderOffer, 0, len(lay.drafts))
	for i, d := range lay.drafts {
		if i == won {
			d.ttl = ttlCovering(d.ttl, window)
		}
		offer := b.newOffer(t, i, d)
		b.send(offer, start)
		switch {
		case i == won:
			b.respond(offer, tender.OfferStatusAccepted, assigned)
			t.AcceptedOfferID = volumegen.Ptr(offer.ID)
		case *offer.ExpiresAt <= assigned:
			offer.Status = tender.OfferStatusExpired
		case f.Chance(0.4) && window > 2*volumegen.Minute:
			b.respond(offer, tender.OfferStatusDeclined,
				start+f.Between64(volumegen.Minute, window-volumegen.Minute))
		default:
			offer.Status = tender.OfferStatusSuperseded
		}
		offers = append(offers, offer)
	}
	t.Status = tender.StatusAccepted
	t.AcceptedAt = volumegen.Ptr(assigned)
	b.finish(t, offers, assigned)
}

func ttlCovering(ttl int32, window int64) int32 {
	need := window + 10*volumegen.Minute
	if int64(ttl) >= need {
		return ttl
	}
	return int32(min(need, int64(tender.MaxOfferTTLSeconds))) //nolint:gosec // bounded above
}

func (b *tvBuilder) priorExhausted(p *tvPlan, before int64) *tender.Tender {
	end := before - b.g.Fake.Between64(5*volumegen.Minute, 2*volumegen.Hour)
	return b.exhaustedTender(p, end)
}

func (b *tvBuilder) exhaustedTender(p *tvPlan, end int64) *tender.Tender {
	f := b.g.Fake
	floor := p.rec.CreatedAt + tvMinGap
	if end-floor < 2*tvMinGap {
		return nil
	}
	lay := b.layout(p, f.Between(2, 5), false)
	durations := make([]int64, len(lay.drafts))
	declined := make([]bool, len(lay.drafts))
	total := int64(0)
	for i, d := range lay.drafts {
		durations[i] = int64(d.ttl)
		if f.Chance(tvDeclineRate) {
			declined[i] = true
			durations[i] = f.Between64(3*volumegen.Minute, int64(d.ttl)*9/10)
		}
		total += durations[i]
	}
	for total > end-floor && len(lay.drafts) > 1 {
		last := len(lay.drafts) - 1
		total -= durations[last]
		lay.drafts, durations, declined = lay.drafts[:last], durations[:last], declined[:last]
	}
	if total > end-floor {
		return nil
	}

	start := end - total
	t := b.newTender(p, lay.mode, start)
	b.attachGuide(t, lay)
	offers := make([]*tender.TenderOffer, 0, len(lay.drafts))
	at := start
	for i, d := range lay.drafts {
		offer := b.newOffer(t, i, d)
		b.send(offer, at)
		switch {
		case b.fail(offer):
		case declined[i]:
			b.respond(offer, tender.OfferStatusDeclined, at+durations[i])
		default:
			offer.Status = tender.OfferStatusExpired
		}
		at += durations[i]
		offers = append(offers, offer)
	}
	t.Status = tender.StatusExhausted
	t.ExhaustedAt = volumegen.Ptr(end)
	b.finish(t, offers, end)

	return t
}

func (b *tvBuilder) live(p *tvPlan, from *tender.Tender) {
	f := b.g.Fake
	floor := p.rec.CreatedAt + tvMinGap
	if from != nil && from.ExhaustedAt != nil {
		floor = max(floor, *from.ExhaustedAt+volumegen.Minute)
	}
	if floor > b.now-tvMinGap {
		return
	}
	start := max(b.now-f.Between64(10*volumegen.Minute, 18*volumegen.Hour), floor)
	broadcast := b.wantsBroadcast(from)
	lay := b.layout(p, f.Between(3, 6), broadcast)
	if len(lay.drafts) == 0 {
		return
	}
	t := b.newTender(p, lay.mode, start)
	b.attachGuide(t, lay)
	b.escalate(t, from)

	counter := f.Chance(tvCounterShare)
	offers := make([]*tender.TenderOffer, 0, len(lay.drafts))
	if lay.mode == tender.ModeSpotBroadcast {
		window := b.now - start
		for i, d := range lay.drafts {
			d.ttl = ttlCovering(d.ttl, window)
			offer := b.newOffer(t, i, d)
			b.send(offer, start)
			if f.Chance(0.3) && window > 2*volumegen.Minute {
				b.respond(offer, tender.OfferStatusDeclined,
					start+f.Between64(volumegen.Minute, window-volumegen.Minute))
			}
			offers = append(offers, offer)
		}
	} else {
		at := start
		current := -1
		for i, d := range lay.drafts {
			offer := b.newOffer(t, i, d)
			offers = append(offers, offer)
			if current >= 0 {
				continue
			}
			end := b.closeOut(offer, at)
			if end < b.now {
				at = end
				continue
			}
			current = i
			offer.Status = tender.OfferStatusSent
			offer.RespondedAt, offer.ResponseSource, offer.DeclineReason = nil, "", ""
			offer.DeliveryError = ""
			if *offer.ExpiresAt <= b.now {
				offer.OfferTTLSeconds = ttlCovering(offer.OfferTTLSeconds, b.now-at)
				offer.ExpiresAt = volumegen.Ptr(at + int64(offer.OfferTTLSeconds))
			}
		}
		if current < 0 {
			t.Status = tender.StatusExhausted
			t.ExhaustedAt = volumegen.Ptr(at)
			b.finish(t, offers, at)
			return
		}
	}

	if counter {
		for _, offer := range offers {
			if offer.Status != tender.OfferStatusSent {
				continue
			}
			sentAt := *offer.SentAt
			at := min(
				sentAt+f.Between64(2*volumegen.Minute, 40*volumegen.Minute),
				b.now-volumegen.Minute,
			)
			if at <= sentAt {
				break
			}
			bump := 1.05 + f.Float()*0.15
			offer.Status = tender.OfferStatusCountered
			offer.CounterRate = decimal.NewNullDecimal(
				offer.Rate.Mul(decimal.NewFromFloat(bump)).Round(2),
			)
			offer.CounterNote = "Can cover at this rate; truck empty nearby"
			offer.CounteredAt = volumegen.Ptr(at)
			offer.CounterMargin = decimal.NewNullDecimal(
				decimal.NewFromFloat(4 + f.Float()*8).Round(2),
			)
			offer.CounterNeedsApproval = true
			t.Status = tender.StatusNeedsReview
			break
		}
	}

	b.finish(t, offers, b.now-volumegen.Minute)
}

func (b *tvBuilder) canceled(p *tvPlan, at int64, reason string) {
	f := b.g.Fake
	if at <= p.rec.CreatedAt {
		return
	}
	lay := b.layout(p, f.Between(2, 4), false)
	if len(lay.drafts) == 0 {
		return
	}
	t := b.newTender(p, lay.mode, at)
	b.attachGuide(t, lay)
	ttl := int64(lay.drafts[0].ttl)
	canceledAt := min(
		at+f.Between64(10*volumegen.Minute, min(ttl-volumegen.Minute, 4*volumegen.Hour)),
		b.now-volumegen.Minute,
	)
	offers := make([]*tender.TenderOffer, 0, len(lay.drafts))
	for i, d := range lay.drafts {
		offer := b.newOffer(t, i, d)
		if i == 0 {
			b.send(offer, at)
			offer.Status = tender.OfferStatusWithdrawn
		} else {
			offer.Status = tender.OfferStatusSkipped
		}
		offers = append(offers, offer)
	}
	t.Status = tender.StatusCanceled
	t.CanceledAt = volumegen.Ptr(max(canceledAt, at))
	t.CanceledByID = volumegen.Ptr(b.g.PickUser(b.g.Refs.UserIDs))
	t.CancellationReason = reason
	b.finish(t, offers, *t.CanceledAt)
}

func (b *tvBuilder) attachGuide(t *tender.Tender, lay tvLayout) {
	if lay.guide != nil && lay.mode == tender.ModeWaterfall {
		t.RoutingGuideID = volumegen.Ptr(lay.guide.guide.ID)
	}
}

func (b *tvBuilder) token(offer *tender.TenderOffer, settled int64) {
	if offer.Channel != tender.ChannelEmail || offer.SentAt == nil ||
		offer.Status == tender.OfferStatusDeliveryFailed {
		return
	}
	sum := sha256.Sum256([]byte("tender-offer-token:" + offer.ID.String()))
	tok := &tender.TenderOfferToken{
		ID:             volumegen.IDAt("tot_", *offer.SentAt),
		BusinessUnitID: offer.BusinessUnitID,
		OrganizationID: offer.OrganizationID,
		TenderOfferID:  offer.ID,
		TokenHash:      hex.EncodeToString(sum[:]),
		Email:          offer.RecipientEmail,
		ExpiresAt:      *offer.ExpiresAt,
		CreatedAt:      *offer.SentAt,
	}
	updated := *offer.SentAt
	switch offer.Status {
	case tender.OfferStatusAccepted, tender.OfferStatusDeclined, tender.OfferStatusCountered:
		if offer.ResponseSource == tender.ResponseSourceEmail {
			used := *offer.SentAt
			if offer.RespondedAt != nil {
				used = *offer.RespondedAt
			}
			if offer.CounteredAt != nil {
				used = *offer.CounteredAt
			}
			tok.UsedAt = volumegen.Ptr(used)
			updated = used
		}
	case tender.OfferStatusSuperseded, tender.OfferStatusWithdrawn:
		at := min(max(settled, *offer.SentAt), b.now)
		tok.RevokedAt = volumegen.Ptr(at)
		updated = at
	case tender.OfferStatusPending, tender.OfferStatusSent, tender.OfferStatusExpired,
		tender.OfferStatusSkipped, tender.OfferStatusDeliveryFailed,
		tender.OfferStatusCounterRejected, tender.OfferStatusRevoked:
	}
	b.batch.tokens = append(b.batch.tokens, tok)
	b.batch.stamp(tvTableTokens, tok.ID, tok.CreatedAt, updated)
}
