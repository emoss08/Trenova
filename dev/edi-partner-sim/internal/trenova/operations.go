package trenova

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

var ErrNotFound = errors.New("not found")

type Stop struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	Sequence             int64  `json:"sequence"`
	Status               string `json:"status"`
	LocationID           string `json:"locationId"`
	ScheduledWindowStart int64  `json:"scheduledWindowStart"`
	ActualArrival        *int64 `json:"actualArrival"`
	ActualDeparture      *int64 `json:"actualDeparture"`
	MoveID               string `json:"-"`
}

type Assignment struct {
	ID string `json:"id"`
}

type Move struct {
	ID         string      `json:"id"`
	Sequence   int64       `json:"sequence"`
	Status     string      `json:"status"`
	Assignment *Assignment `json:"assignment"`
	Stops      []Stop      `json:"stops"`
}

type Shipment struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	ProNumber         string `json:"proNumber"`
	BOL               string `json:"bol"`
	CustomerID        string `json:"customerId"`
	ServiceTypeID     string `json:"serviceTypeId"`
	ShipmentTypeID    string `json:"shipmentTypeId"`
	TotalChargeAmount string `json:"totalChargeAmount"`
	Moves             []Move `json:"moves"`
}

func (s *Shipment) Stops() []Stop {
	moves := slices.Clone(s.Moves)
	slices.SortFunc(moves, func(a, b Move) int { return cmp.Compare(a.Sequence, b.Sequence) })
	var stops []Stop
	for _, move := range moves {
		ordered := slices.Clone(move.Stops)
		slices.SortFunc(ordered, func(a, b Stop) int { return cmp.Compare(a.Sequence, b.Sequence) })
		for index := range ordered {
			ordered[index].MoveID = move.ID
			stops = append(stops, ordered[index])
		}
	}
	return stops
}

func (s *Shipment) TotalCents() (int64, bool) {
	return Cents(s.TotalChargeAmount)
}

func Cents(amount string) (int64, bool) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return 0, false
	}
	whole, fraction, _ := strings.Cut(amount, ".")
	fraction = (fraction + "00")[:2]
	value, err := strconv.ParseInt(whole+fraction, 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

const shipmentFields = `id status proNumber bol customerId serviceTypeId shipmentTypeId
	totalChargeAmount
	moves { id sequence status assignment { id }
		stops { id type sequence status locationId scheduledWindowStart actualArrival
			actualDeparture } }`

func (c *Client) Shipment(ctx context.Context, id string) (*Shipment, error) {
	var out struct {
		Shipment *Shipment `json:"shipment"`
	}
	err := c.GraphQL(ctx, `query EdiSimShipment($id: ID!) {
		shipment(id: $id, expandShipmentDetails: true) { `+shipmentFields+` } }`,
		map[string]any{"id": id}, &out)
	if err != nil {
		return nil, err
	}
	if out.Shipment == nil {
		return nil, fmt.Errorf("the shipment: %w", ErrNotFound)
	}
	return out.Shipment, nil
}

func (c *Client) RecentShipments(ctx context.Context, first int) ([]*Shipment, error) {
	var out struct {
		Shipments struct {
			Edges []struct {
				Node *Shipment `json:"node"`
			} `json:"edges"`
		} `json:"shipments"`
	}
	err := c.GraphQL(ctx, `query EdiSimShipments($input: ShipmentsInput!) {
		shipments(input: $input) { edges { node { `+shipmentFields+` } } } }`,
		map[string]any{"input": map[string]any{"first": first, "expandShipmentDetails": true}},
		&out)
	if err != nil {
		return nil, err
	}
	shipments := make([]*Shipment, 0, len(out.Shipments.Edges))
	for _, edge := range out.Shipments.Edges {
		if edge.Node != nil {
			shipments = append(shipments, edge.Node)
		}
	}
	return shipments, nil
}

type Tractor struct {
	ID               string `json:"id"`
	PrimaryWorkerID  string `json:"primaryWorkerId"`
	InProgressMoveID string `json:"inProgressMoveId"`
}

func (c *Client) AvailableTractors(ctx context.Context, limit int) ([]Tractor, error) {
	var out struct {
		Tractors struct {
			Edges []struct {
				Node Tractor `json:"node"`
			} `json:"edges"`
		} `json:"tractors"`
	}
	err := c.GraphQL(ctx, `query EdiSimTractors($input: DataTableConnectionInput!) {
  tractors(input: $input, status: Available) {
    edges { node { id primaryWorkerId inProgressMoveId } }
  }
}`, map[string]any{"input": map[string]any{"first": limit}}, &out)
	if err != nil {
		return nil, err
	}
	crewed := make([]Tractor, 0, len(out.Tractors.Edges))
	for _, edge := range out.Tractors.Edges {
		if edge.Node.PrimaryWorkerID != "" {
			crewed = append(crewed, edge.Node)
		}
	}
	return crewed, nil
}

func (c *Client) Assign(ctx context.Context, moveID, workerID, tractorID string) error {
	return c.post(ctx, "/shipment-moves/"+url.PathEscape(moveID)+"/assignment/",
		map[string]string{"primaryWorkerId": workerID, "tractorId": tractorID}, nil)
}

const (
	ActionArrive = "Arrive"
	ActionDepart = "Depart"
)

func (c *Client) RecordStop(ctx context.Context, moveID, stopID, action string) error {
	return c.post(ctx,
		"/shipment-moves/"+url.PathEscape(moveID)+"/stops/"+url.PathEscape(stopID)+
			"/record-actual/",
		map[string]any{"action": action, "occurredAt": time.Now().Unix()}, nil)
}

type Invoice struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Number string `json:"number"`
}

func (c *Client) InvoiceShipment(ctx context.Context, shipmentID string) (*Invoice, error) {
	invoice := new(Invoice)
	if err := c.post(ctx, "/billing/invoices/from-shipments/",
		map[string]any{"shipmentIds": []string{shipmentID}}, invoice); err != nil {
		return nil, err
	}
	posted := new(Invoice)
	if err := c.post(ctx, "/billing/invoices/"+url.PathEscape(invoice.ID)+"/post/",
		nil, posted); err != nil {
		return invoice, err
	}
	return posted, nil
}

type CarrierChannel struct {
	ID                     string  `json:"id,omitempty"`
	EDIPartnerID           string  `json:"ediPartnerId"`
	CommunicationProfileID *string `json:"communicationProfileId"`
	IsDefault              bool    `json:"isDefault"`
}

func (c *Client) FindCarrier(ctx context.Context, scac string) (map[string]any, error) {
	var found page[map[string]any]
	query := url.Values{"query": {scac}, "limit": {"50"}, "includeEdiChannels": {"true"}}
	if err := c.get(ctx, listPath("/carriers/", query), &found); err != nil {
		return nil, err
	}
	for _, carrier := range found.Results {
		if value, _ := carrier["scac"].(string); strings.EqualFold(value, scac) {
			id, _ := carrier["id"].(string)
			return c.Carrier(ctx, id)
		}
	}
	return nil, nil
}

func (c *Client) Carrier(ctx context.Context, id string) (map[string]any, error) {
	carrier := map[string]any{}
	err := c.get(ctx, listPath("/carriers/"+url.PathEscape(id),
		url.Values{"includeEdiChannels": {"true"}}), &carrier)
	return carrier, err
}

func (c *Client) CreateCarrier(
	ctx context.Context,
	carrier map[string]any,
) (map[string]any, error) {
	created := map[string]any{}
	err := c.post(ctx, "/carriers/", carrier, &created)
	return created, err
}

func (c *Client) UpdateCarrier(
	ctx context.Context,
	carrier map[string]any,
) (map[string]any, error) {
	id, _ := carrier["id"].(string)
	updated := map[string]any{}
	err := c.put(ctx, "/carriers/"+url.PathEscape(id)+"/", carrier, &updated)
	return updated, err
}

type NewShipment struct {
	CustomerID     string
	ServiceTypeID  string
	ShipmentTypeID string
	BOL            string
	Pickup         string
	Delivery       string
	PickupAt       time.Time
	DeliverAt      time.Time
	Weight         int64
	Pieces         int64
}

func (c *Client) CreateShipment(ctx context.Context, req *NewShipment) (*Shipment, error) {
	window := func(at time.Time) (int64, int64) {
		return at.Unix(), at.Add(2 * time.Hour).Unix()
	}
	pickupStart, pickupEnd := window(req.PickupAt)
	deliverStart, deliverEnd := window(req.DeliverAt)
	stop := func(kind, location string, sequence, start, end int64) map[string]any {
		return map[string]any{
			"locationId":           location,
			"status":               "New",
			"type":                 kind,
			"scheduleType":         "Open",
			"handling":             "Live",
			"sequence":             sequence,
			"scheduledWindowStart": start,
			"scheduledWindowEnd":   end,
			"pieces":               req.Pieces,
			"weight":               req.Weight,
		}
	}
	body := map[string]any{
		"customerId":     req.CustomerID,
		"serviceTypeId":  req.ServiceTypeID,
		"shipmentTypeId": req.ShipmentTypeID,
		"status":         "New",
		"entryMethod":    "Manual",
		"bol":            req.BOL,
		"pieces":         req.Pieces,
		"weight":         req.Weight,
		"moves": []map[string]any{{
			"status":   "New",
			"loaded":   true,
			"sequence": 0,
			"stops": []map[string]any{
				stop("Pickup", req.Pickup, 0, pickupStart, pickupEnd),
				stop("Delivery", req.Delivery, 1, deliverStart, deliverEnd),
			},
		}},
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := c.post(ctx, "/shipments/", body, &created); err != nil {
		return nil, err
	}
	return c.Shipment(ctx, created.ID)
}

type SpotTender struct {
	MoveID    string
	CarrierID string
	RateCents int64
	TTL       time.Duration
}

type Tender struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (c *Client) TenderSpot(ctx context.Context, req *SpotTender) (*Tender, error) {
	tender := new(Tender)
	err := c.post(ctx, "/tenders/spot/", map[string]any{
		"shipmentMoveId": req.MoveID,
		"mode":           "SpotSequential",
		"lines": []map[string]any{{
			"carrierId":       req.CarrierID,
			"rateMethod":      "Flat",
			"rate":            fmt.Sprintf("%d.%02d", req.RateCents/100, req.RateCents%100),
			"offerTtlSeconds": int32(req.TTL / time.Second),
			"channel":         "EDI",
		}},
		"overrideInsuranceWarnings": true,
	}, tender)
	return tender, err
}
