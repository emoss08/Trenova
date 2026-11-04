package shipmentvolume

import (
	"hash/fnv"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	shpDropShare        = 0.17
	shpVerySlowDocks    = 6
	shpSlowDocks        = 22
	shpDriftingDocks    = 12
	shpDriftBase        = 50 * shpMinute
	shpMorningFrom      = 5
	shpMorningUntil     = 10
	shpPaceBuckets      = 100
	shpDriftBuckets     = 100
	shpDriftWorseShare  = 6
	shpPaceSpreadMinute = 150
)

type shpPricing struct {
	miles     float64
	stops     int
	detention decimal.Decimal
}

type shpDockPace struct {
	delay int64
	drift float64
}

func shpPaceOf(loc *volumegen.LocationRecord) shpDockPace {
	h := fnv.New32a()
	_, _ = h.Write([]byte(loc.Code))
	sum := h.Sum32()
	spread := int64((sum>>8)%shpPaceSpreadMinute) * shpMinute

	pace := shpDockPace{}
	switch bucket := sum % shpPaceBuckets; {
	case bucket < shpVerySlowDocks:
		pace.delay = 2*shpHour + 30*shpMinute + spread
	case bucket < shpSlowDocks:
		pace.delay = 45*shpMinute + spread/2
	}

	switch drift := (sum >> 16) % shpDriftBuckets; {
	case drift < shpDriftWorseShare:
		pace.drift = 1
	case drift < shpDriftingDocks:
		pace.drift = -1
	}
	if pace.drift != 0 {
		pace.delay += shpDriftBase
	}

	return pace
}

func (p shpDockPace) at(arrival, start, now int64) int64 {
	if p.drift == 0 || now <= start {
		return p.delay
	}
	progress := min(max(float64(arrival-start)/float64(now-start), 0), 1)

	return max(int64(float64(p.delay)*(1+p.drift*(progress-0.5)*1.6)), 0)
}

func (b *shpBuilder) handling(sp shpStopPlan) shipmentvocab.StopHandling {
	if sp.typ == shipmentvocab.StopTypeDelivery && b.g.Fake.Chance(shpDropShare) {
		return shipmentvocab.StopHandlingDrop
	}
	return shipmentvocab.StopHandlingLive
}

func (b *shpBuilder) dwell(ts *shpTimedStop, zone *time.Location, held bool) int64 {
	f := b.g.Fake
	if ts.handling == shipmentvocab.StopHandlingDrop {
		return f.Between64(15*shpMinute, 50*shpMinute)
	}

	dwell := f.Between64(40*shpMinute, 2*shpHour+15*shpMinute)
	if delay := shpPaceOf(ts.plan.loc).at(ts.arrival, volumegen.Unix(b.g.Start), b.now); delay > 0 {
		dwell += delay/2 + f.Between64(0, delay)
	}
	local := time.Unix(ts.arrival, 0).In(zone)
	if hour := local.Hour(); hour >= shpMorningFrom && hour < shpMorningUntil {
		dwell += f.Between64(10*shpMinute, 55*shpMinute)
	}
	if local.Weekday() == time.Friday {
		dwell += f.Between64(0, 30*shpMinute)
	}
	if held {
		dwell = max(dwell, f.Between64(3*shpHour+15*shpMinute, 6*shpHour))
	}

	return dwell
}

func shpDelivered(status shipmentvocab.Status) bool {
	return status == shipmentvocab.StatusCompleted ||
		status == shipmentvocab.StatusReadyToInvoice ||
		status == shipmentvocab.StatusInvoiced
}

func shpCommodityIDs(p *shpPlan) []pulid.ID {
	ids := make([]pulid.ID, 0, len(p.commodities))
	for _, c := range p.commodities {
		ids = append(ids, c.commodity.ID)
	}
	return ids
}

func shpDetentionCharge(rec *volumegen.ShipmentRecord) pulid.ID {
	for _, c := range rec.Charges {
		if c.Code == "DET" {
			return c.AdditionalChargeID
		}
	}
	return pulid.Nil
}
