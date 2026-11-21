package play

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/poll"
	"github.com/emoss08/trenova/edi-partner-sim/internal/provision"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/sim"
	"github.com/emoss08/trenova/edi-partner-sim/internal/simclient"
	"github.com/emoss08/trenova/edi-partner-sim/internal/trenova"
)

const (
	tenderTTL          = 4 * time.Hour
	transferProcessing = "Processing"
	transferFailed     = "Failed"
)

var (
	ErrNoKey      = errors.New("the conversation has no key yet")
	ErrNoFlow     = errors.New("no flow instance follows the conversation")
	ErrNoShipment = errors.New("the conversation has no shipment")
)

type Conversation struct {
	Name          string
	Party         *Party
	Seed          uint64
	Key           string
	Original      *partner.Load
	Tendered      *partner.Load
	Need          *partner.LoadNeed
	TenderedCents int64
	StartedAt     time.Time
	LastSent      []string
	BOL           string
	mu            sync.Mutex
	instanceID    string
	transferID    string
	shipmentID    string
	claimed       map[string]bool
	states        []Snapshot
	trace         *Trace
}

func (c *Conversation) claim(ids ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.claimed == nil {
		c.claimed = map[string]bool{}
	}
	for _, id := range ids {
		c.claimed[id] = true
	}
}

func (c *Conversation) isClaimed(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.claimed[id]
}

func (c *Conversation) tenderedCents() int64 {
	if c.Party.Carrier() {
		return c.TenderedCents
	}
	if c.Tendered != nil {
		return c.Tendered.TotalCents
	}
	return 0
}

func (c *Conversation) originalCents() int64 {
	if c.Party.Carrier() {
		return c.TenderedCents
	}
	if c.Original != nil {
		return c.Original.TotalCents
	}
	return 0
}

func (e *Env) SendTender(
	ctx context.Context,
	party *Party,
	conversations []*Conversation,
	variation partner.Variation,
) ([]journal.Message, error) {
	if len(conversations) == 0 {
		return nil, errors.New("nothing to send")
	}
	req := &sim.SendRequest{
		Set:       "204",
		Seed:      conversations[0].Seed,
		Variation: string(variation.Kind),
		Sparse:    variation.Sparse,
		Delta: sim.DeltaRequest{
			RatePercent: variation.Delta.RatePercent,
			AddStop:     variation.Delta.AddStop,
		},
		Load: variation.Need,
	}
	if move := variation.Delta.Move; move != nil {
		req.Delta.Move = &sim.MoveRequest{Stop: move.Stop, By: move.By.String()}
	}
	for _, c := range conversations[1:] {
		req.With = append(req.With, c.Seed)
	}
	now := time.Now()
	for _, c := range conversations {
		_, load, err := party.Def.Transaction("204", c.Seed, variation)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.Name, err)
		}
		if variation.Kind == partner.KindOriginal {
			c.Key, c.Original, c.StartedAt = load.Number, load, now
		}
		if variation.Kind != partner.KindCancel && variation.Kind != partner.KindDuplicate {
			c.Tendered = load
		}
	}
	messages, err := e.Sim.Send(ctx, party.Name(), req)
	if err != nil {
		return nil, err
	}
	e.Watch.Observe(messages)
	for index := range messages {
		for _, c := range conversations {
			c.noteMatch(runlog.Match{MessageID: messages[index].ID, Key: c.Key})
		}
	}
	ids := make([]string, 0, len(messages))
	var failed []string
	for index := range messages {
		ids = append(ids, messages[index].ID)
		if messages[index].State == journal.StateFailed {
			failed = append(failed, firstNonEmpty(messages[index].Error, "delivery failed"))
		}
	}
	for _, c := range conversations {
		c.LastSent = ids
	}
	if len(failed) > 0 {
		return messages, fmt.Errorf("the simulator could not deliver the 204: %s",
			strings.Join(failed, "; "))
	}
	if party.Setup.Transport == provision.MethodSFTP {
		if err = e.Trenova.PollProfile(ctx, party.Setup.ProfileID); err != nil {
			return messages, fmt.Errorf("ask Trenova to poll the SFTP mailbox: %w", err)
		}
	}
	return messages, nil
}

func (e *Env) Instance(
	ctx context.Context,
	c *Conversation,
	within time.Duration,
) (*trenova.FlowInstance, error) {
	if c.Key == "" {
		return nil, ErrNoKey
	}
	var found *trenova.FlowInstance
	err := poll.Until(ctx, within, func(ctx context.Context) (bool, error) {
		instances, err := e.Trenova.FlowInstances(ctx, c.Party.Setup.PartnerID, c.Key)
		if err != nil {
			return false, err
		}
		if len(instances) == 0 {
			return false, nil
		}
		found = instances[0]
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w (key %s): %w", ErrNoFlow, c.Key, err)
	}
	c.mu.Lock()
	c.instanceID = found.ID
	if found.TransferID != "" {
		c.transferID = found.TransferID
	}
	if found.ShipmentID != "" {
		c.shipmentID = found.ShipmentID
	}
	c.mu.Unlock()
	return found, nil
}

func (e *Env) transfer(ctx context.Context, c *Conversation, within time.Duration) (string, error) {
	var transferID string
	err := poll.Until(ctx, within, func(ctx context.Context) (bool, error) {
		instance, err := e.Instance(ctx, c, within)
		if err != nil {
			return false, poll.Stop(err)
		}
		transferID = instance.TransferID
		return transferID != "", nil
	})
	if err != nil {
		return "", fmt.Errorf("the flow for %s has no tender transfer to decide: %w", c.Key, err)
	}
	return transferID, nil
}

func (e *Env) ShipmentOf(
	ctx context.Context,
	c *Conversation,
	within time.Duration,
) (string, error) {
	c.mu.Lock()
	known := c.shipmentID
	c.mu.Unlock()
	if known != "" {
		return known, nil
	}
	var shipmentID string
	err := poll.Until(ctx, within, func(ctx context.Context) (bool, error) {
		instance, err := e.Instance(ctx, c, within)
		if err != nil {
			return false, poll.Stop(err)
		}
		shipmentID = instance.ShipmentID
		if shipmentID == "" && instance.TransferID != "" {
			transfer, transferErr := e.Trenova.Transfer(ctx, instance.TransferID)
			if transferErr != nil {
				return false, transferErr
			}
			shipmentID = transfer.TargetShipmentID
		}
		return shipmentID != "", nil
	})
	if err != nil {
		return "", fmt.Errorf("%w (key %s): %w", ErrNoShipment, c.Key, err)
	}
	c.mu.Lock()
	c.shipmentID = shipmentID
	c.mu.Unlock()
	return shipmentID, nil
}

func (e *Env) Decide(
	ctx context.Context,
	c *Conversation,
	accept bool,
	reason string,
	within time.Duration,
) error {
	transferID, err := e.transfer(ctx, c, within)
	if err != nil {
		return err
	}
	event := runlog.Event{Label: "Planner accepted", Who: e.email}
	if accept {
		mappings, mapErr := e.mapper.Resolve(ctx, c.Party.Setup, transferID)
		if mapErr != nil {
			return mapErr
		}
		_, err = e.Trenova.ApproveTransfer(ctx, transferID, mappings)
		if err == nil {
			err = e.approved(ctx, transferID, within)
		}
		if len(mappings) > 0 && err == nil {
			event.Detail = fmt.Sprintf("mapped %d of the partner's codes", len(mappings))
		}
	} else {
		event.Label, event.Detail = "Planner declined", reason
		_, err = e.Trenova.RejectTransfer(ctx, transferID, firstNonEmpty(reason,
			"Declined by the EDI simulator"))
	}
	if err == nil {
		c.noteEvent(event)
	}
	return err
}

func (e *Env) approved(ctx context.Context, transferID string, within time.Duration) error {
	var transfer *trenova.Transfer
	err := poll.Until(ctx, within, func(ctx context.Context) (bool, error) {
		current, err := e.Trenova.Transfer(ctx, transferID)
		if err != nil {
			return false, err
		}
		transfer = current
		return current.Status != transferProcessing, nil
	})
	if err != nil {
		return fmt.Errorf("the accepted transfer is still processing: %w", err)
	}
	if transfer.Status == transferFailed {
		return fmt.Errorf("Trenova accepted the transfer but could not build its shipment: %s",
			firstNonEmpty(transfer.FailureReason, "no reason given"))
	}
	return nil
}

func (e *Env) Dispatch(ctx context.Context, c *Conversation, within time.Duration) error {
	shipmentID, err := e.ShipmentOf(ctx, c, within)
	if err != nil {
		return err
	}
	shipment, err := e.Trenova.Shipment(ctx, shipmentID)
	if err != nil {
		return err
	}
	e.fleet.drive(shipmentID)
	for _, move := range shipment.Moves {
		if move.Assignment != nil && move.Assignment.ID != "" {
			continue
		}
		if err = e.assign(ctx, move.ID); err != nil {
			return err
		}
	}
	c.noteEvent(runlog.Event{Label: "Dispatched", Who: e.email})
	return nil
}

func (e *Env) assign(ctx context.Context, moveID string) error {
	var last error
	for range assignAttempts {
		worker, tractor, err := e.pair(ctx)
		if err != nil {
			return err
		}
		if last = e.Trenova.Assign(ctx, moveID, worker, tractor); last == nil {
			return nil
		}
	}
	return fmt.Errorf("assign move %s: %w", moveID, last)
}

func (e *Env) RecordStop(
	ctx context.Context,
	c *Conversation,
	place int,
	action string,
	within time.Duration,
) error {
	shipmentID, err := e.ShipmentOf(ctx, c, within)
	if err != nil {
		return err
	}
	shipment, err := e.Trenova.Shipment(ctx, shipmentID)
	if err != nil {
		return err
	}
	stops := shipment.Stops()
	if place < 1 || place > len(stops) {
		return fmt.Errorf("the shipment has %d stops, not a stop %d", len(stops), place)
	}
	stop := stops[place-1]
	if err = e.Trenova.RecordStop(ctx, stop.MoveID, stop.ID, action); err != nil {
		return err
	}
	label := fmt.Sprintf("Arrived at stop %d", place)
	if action == trenova.ActionDepart {
		label = fmt.Sprintf("Departed stop %d", place)
	}
	c.noteEvent(runlog.Event{Label: label, Who: e.email})
	return nil
}

func (e *Env) Deliver(ctx context.Context, c *Conversation, within time.Duration) error {
	shipmentID, err := e.ShipmentOf(ctx, c, within)
	if err != nil {
		return err
	}
	shipment, err := e.Trenova.Shipment(ctx, shipmentID)
	if err != nil {
		return err
	}
	if err = e.finishStops(ctx, shipment); err != nil {
		return err
	}
	c.noteEvent(runlog.Event{Label: "Delivered", Detail: "every stop arrived and departed",
		Who: e.email})
	return nil
}

func (e *Env) Invoice(ctx context.Context, c *Conversation, within time.Duration) error {
	shipmentID, err := e.ShipmentOf(ctx, c, within)
	if err != nil {
		return err
	}
	if _, err = e.Trenova.InvoiceShipment(ctx, shipmentID); err != nil {
		return err
	}
	c.noteEvent(runlog.Event{Label: "Invoiced", Who: e.email})
	return nil
}

func (c *Conversation) shipment() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.shipmentID
}

func (e *Env) Release(ctx context.Context, c *Conversation, within time.Duration) error {
	instance, err := e.Instance(ctx, c, within)
	if err != nil {
		return err
	}
	if err = e.Trenova.ReleaseFlow(ctx, instance.ID); err != nil {
		return err
	}
	c.noteEvent(runlog.Event{Label: "Planner released", Who: e.email})
	return nil
}

func (e *Env) Advance(
	ctx context.Context,
	c *Conversation,
	by time.Duration,
	within time.Duration,
) (*trenova.Advance, error) {
	instance, err := e.Instance(ctx, c, within)
	if err != nil {
		return nil, err
	}
	return e.CLI.Advance(ctx, instance.ID, by)
}

func (e *Env) TenderOut(
	ctx context.Context,
	c *Conversation,
	rateCents int64,
	within time.Duration,
) error {
	template, err := e.shipmentTemplate(ctx)
	if err != nil {
		return err
	}
	now := time.Now().Truncate(time.Hour)
	template.BOL = c.BOL
	template.PickupAt = now.Add(24 * time.Hour)
	template.DeliverAt = now.Add(48 * time.Hour)
	shipment, err := e.Trenova.CreateShipment(ctx, &template)
	if err != nil {
		return fmt.Errorf("create the shipment to tender: %w", err)
	}
	c.mu.Lock()
	c.shipmentID = shipment.ID
	c.mu.Unlock()
	if len(shipment.Moves) == 0 {
		return errors.New("the created shipment has no move to tender")
	}
	c.TenderedCents, c.StartedAt = rateCents, time.Now()
	if _, err = e.Trenova.TenderSpot(ctx, &trenova.SpotTender{
		MoveID:    shipment.Moves[0].ID,
		CarrierID: c.Party.Setup.CarrierID,
		RateCents: rateCents,
		TTL:       tenderTTL,
	}); err != nil {
		return fmt.Errorf("tender the shipment to %s: %w", c.Party.Setup.CarrierID, err)
	}
	if err = e.bindTender(ctx, c, shipment, within); err != nil {
		return err
	}
	c.noteEvent(runlog.Event{Label: "Tendered to the carrier", Detail: money(rateCents),
		Who: e.email})
	return nil
}

func (e *Env) bindTender(
	ctx context.Context,
	c *Conversation,
	shipment *trenova.Shipment,
	within time.Duration,
) error {
	candidates := []string{shipment.ProNumber, shipment.BOL, c.BOL}
	name := c.Party.Name()
	err := e.Watch.Until(ctx, time.Now().Add(within), func(view simclient.View) bool {
		tenders := view.Partner(name, fromTrenova("204"))
		for index := range tenders {
			for _, key := range tenders[index].Keys {
				if key != "" && slices.Contains(candidates, key) {
					c.Key = key
					return true
				}
			}
		}
		return false
	})
	if err != nil {
		return fmt.Errorf("no 204 for the shipment (pro %s, bol %s) reached the carrier "+
			"within %s: %w", shipment.ProNumber, c.BOL, within, err)
	}
	return nil
}

func rateCents(text string, fallback int64) int64 {
	if text == "" {
		return fallback
	}
	value, ok := trenova.Cents(text)
	if !ok {
		return fallback
	}
	return value
}

func percentOf(cents int64, percent float64) int64 {
	return int64(math.Round(float64(cents) * (100 + percent) / 100))
}
