package conform

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/reply"
	"github.com/emoss08/trenova/pkg/lading"
)

const (
	WantApplied  = string(reply.OutcomeApplied)
	WantHeld     = string(reply.OutcomeHeld)
	WantRejected = string(reply.OutcomeRejected)
	offerLead    = 24 * time.Hour

	carrierOfferPayment = paymentPrepaid
)

type Offer struct {
	Label    string
	Document any
	At       time.Time
}

type Reply struct {
	Name string
	Text string
	Set  string
	X12  []byte
	At   time.Time
	Want string
	Note string
}

type Conversation struct {
	Label   string
	Title   string
	Replies []Reply
}

type CarrierKit interface {
	Offer(def *partner.Definition, seed uint64) (Offer, error)
	Converse(def *partner.Definition, tender []byte, seed uint64) (Conversation, error)
}

type DefaultKit struct{}

func (DefaultKit) Offer(_ *partner.Definition, seed uint64) (Offer, error) {
	offered := reply.Offer(seed)
	document, at, err := tenderFromOffer(offered)
	if err != nil {
		return Offer{}, err
	}
	return Offer{Label: "tender " + offered.Key, Document: document, At: at}, nil
}

func (DefaultKit) Converse(
	def *partner.Definition,
	tender []byte,
	seed uint64,
) (Conversation, error) {
	carrier, err := reply.Open(def)
	if err != nil {
		return Conversation{}, err
	}
	tenders, _, err := carrier.Receive(tender)
	if err != nil {
		return Conversation{}, err
	}
	if len(tenders) != 1 {
		return Conversation{}, fmt.Errorf("the carrier read %d tenders, want 1", len(tenders))
	}
	scenario := reply.DrawScenario(seed)
	replies, err := carrier.Converse(tenders[0], scenario)
	if err != nil {
		return Conversation{}, err
	}
	start := int64(seed*16%999_999_999) + 1
	controls := partner.Controls{
		Interchange: partner.NewCounter(start),
		Group:       partner.NewCounter(start),
	}
	conversation := Conversation{Label: scenario.String(), Title: scenario.Describe()}
	seen := map[string]int{}
	for i := range replies {
		r := &replies[i]
		written, writeErr := carrier.Write(*r, controls)
		if writeErr != nil {
			return Conversation{}, writeErr
		}
		seen[r.Set]++
		conversation.Replies = append(conversation.Replies, Reply{
			Name: fmt.Sprintf("%s in %d", r.Set, seen[r.Set]),
			Text: replyText(def.Partner.Display, r, scenario),
			Set:  r.Set,
			X12:  written.Bytes,
			At:   r.At,
			Want: string(r.Want),
			Note: r.Note,
		})
	}
	return conversation, nil
}

func tenderFromOffer(offered *reply.OfferedLoad) (loadTender, time.Time, error) {
	total := &money{Amount: offered.Total.Amount, Currency: offered.Total.Currency}
	document := loadTender{
		Key:           offered.Key,
		Purpose:       offered.Purpose,
		BOL:           offered.BOL,
		PONumber:      offered.PONumber,
		Payment:       paymentOf(carrierOfferPayment),
		Equipment:     offered.Equipment,
		Weight:        &measure{Value: offered.Weight.Value, Unit: offered.Weight.Unit},
		Pieces:        count(offered.Pieces),
		Total:         total,
		FreightCharge: total,
	}
	var first time.Time
	for index := range offered.Stops {
		stop := &offered.Stops[index]
		start, err := time.Parse(time.RFC3339, stop.Window.Start.At)
		if err != nil {
			return loadTender{}, time.Time{}, fmt.Errorf("offered stop %d: %w", stop.Sequence, err)
		}
		if index == 0 {
			first = start
		}
		end := dateTime{At: stop.Window.End.At, Zone: stop.Window.End.Zone}
		written := tenderStop{
			Sequence: int64(stop.Sequence),
			Type:     stop.Type,
			Name:     stop.Name,
			Code:     stop.Code,
			Window: window{
				Start: dateTime{At: stop.Window.Start.At, Zone: stop.Window.Start.Zone},
				End:   &end,
			},
			Weight: &measure{Value: stop.Weight.Value, Unit: stop.Weight.Unit},
			Pieces: count(stop.Pieces),
			Address: &address{
				Line1:  stop.Address.Line1,
				City:   stop.Address.City,
				State:  stop.Address.State,
				Postal: stop.Address.Postal,
			},
		}
		if stop.Code != "" || stop.Name != "" {
			written.Location = &ref{
				ID:    trenovaID("loc", uint64(index+1)),
				Code:  stop.Code,
				Label: stop.Name,
			}
		}
		document.Stops = append(document.Stops, written)
	}
	if len(document.Stops) == 0 {
		return loadTender{}, time.Time{}, errors.New("the offered load has no stops")
	}
	return document, first.Add(-offerLead), nil
}

func (r *runner) carrier(ctx context.Context, s *subject, c *Case) {
	offer, err := r.kit.Offer(s.def, c.Seed)
	if err != nil {
		c.fail(c.step("204 out", tenderText(s.def), directionOutbound, tenderSet), &Failure{
			Source:  SourceHarness,
			Rule:    "offer",
			Message: err.Error(),
		})
		return
	}
	if document, ok := offer.Document.(loadTender); ok {
		offer.Document = newVariety(s.def.Name(), s.guides, c.Seed).tender(document)
	}
	c.Variation, c.Title = offer.Label, "Tender to "+s.def.Partner.Display
	tender := r.write(ctx, s, c, namedDocument{
		name:     "204 out",
		text:     tenderText(s.def),
		set:      tenderSet,
		document: offer.Document,
		now:      offer.At,
	}, controlFor(c.Seed, 0))
	if tender.X12 == "" || tender.Status == StatusHeld || tender.Status == StatusDeclined {
		return
	}
	instance := r.offered(ctx, s, c, tender, offer.At)
	conversation, err := r.kit.Converse(s.def, []byte(tender.X12), c.Seed)
	if err != nil {
		c.fail(c.step("replies", "Read the carrier's replies", directionInbound, ""), &Failure{
			Source:  SourceHarness,
			Rule:    "replies",
			Message: err.Error(),
		})
		return
	}
	c.Variation += ", " + conversation.Label
	if conversation.Title != "" {
		c.Title = conversation.Title
	}
	for _, reply := range conversation.Replies {
		step := c.step(reply.Name, cmp.Or(reply.Text, "Read the carrier's "+reply.Set),
			directionInbound, reply.Set)
		step.Note = reply.Note
		r.read(ctx, s, c, step, message{
			text:   string(reply.X12),
			now:    reply.At,
			answer: anyLookup,
			want:   reply.Want,
			flows:  true,
			flow:   instance,
		})
	}
}

func anyLookup(need lading.Need) *lading.Found {
	return &lading.Found{ID: need.Kind + "-" + need.Code, Label: need.Code}
}
