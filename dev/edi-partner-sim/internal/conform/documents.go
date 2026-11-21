package conform

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	unitPounds           = "lb"
	unitFahrenheit       = "°F"
	currencyUSD          = "USD"
	freightCode          = "400"
	paymentPrepaid       = "Prepaid"
	stopPickup           = "Pickup"
	stopDelivery         = "Delivery"
	eventStopArrived     = "StopArrived"
	eventStopDeparted    = "StopDeparted"
	eventDelivered       = "Delivered"
	maxExternalReference = 100
	maxPONumber          = 100
	refBillOfLading      = "BM"
	refCarrierPRO        = "CN"
	refPurchaseOrder     = "PO"
	otherZone            = "America/Denver"
)

var payments = map[string]bool{
	"Prepaid":    true,
	"Collect":    true,
	"ThirdParty": true,
}

var statusKinds = map[string]string{
	"AF": "DepartedPickup",
	"X3": "ArrivedPickup",
	"X1": "ArrivedDelivery",
	"D1": "Delivered",
	"CD": "DepartedDelivery",
}

type money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency,omitempty"`
}

type measure struct {
	Value string `json:"value"`
	Unit  string `json:"unit"`
}

type dateTime struct {
	At   string `json:"at"`
	Zone string `json:"zone,omitempty"`
}

type ref struct {
	ID    string `json:"id,omitempty"`
	Code  string `json:"code,omitempty"`
	Label string `json:"label,omitempty"`
}

type address struct {
	Line1   string `json:"line1"`
	Line2   string `json:"line2,omitempty"`
	City    string `json:"city"`
	State   string `json:"state,omitempty"`
	Postal  string `json:"postal,omitempty"`
	Country string `json:"country,omitempty"`
}

type window struct {
	Start dateTime  `json:"start"`
	End   *dateTime `json:"end,omitempty"`
}

type temperatureRange struct {
	Min *measure `json:"min,omitempty"`
	Max *measure `json:"max,omitempty"`
}

type tenderStop struct {
	Sequence    int64             `json:"sequence"`
	Type        string            `json:"type"`
	Location    *ref              `json:"location,omitempty"`
	Name        string            `json:"name,omitempty"`
	Code        string            `json:"code,omitempty"`
	Address     *address          `json:"address,omitempty"`
	Window      window            `json:"window"`
	Weight      *measure          `json:"weight,omitempty"`
	Pieces      *int64            `json:"pieces,omitempty"`
	Temperature *temperatureRange `json:"temperature,omitempty"`
}

type commodityLine struct {
	Commodity   *ref              `json:"commodity,omitempty"`
	Description string            `json:"description"`
	Weight      *measure          `json:"weight,omitempty"`
	Pieces      *int64            `json:"pieces,omitempty"`
	Temperature *temperatureRange `json:"temperature,omitempty"`
	Hazmat      *bool             `json:"hazmat,omitempty"`
}

type loadTender struct {
	Key           string            `json:"key"`
	Purpose       string            `json:"purpose,omitempty"`
	BOL           string            `json:"bol,omitempty"`
	PONumber      string            `json:"poNumber,omitempty"`
	Payment       string            `json:"payment,omitempty"`
	Equipment     string            `json:"equipment,omitempty"`
	Weight        *measure          `json:"weight,omitempty"`
	Pieces        *int64            `json:"pieces,omitempty"`
	Temperature   *temperatureRange `json:"temperature,omitempty"`
	Total         *money            `json:"total,omitempty"`
	FreightCharge *money            `json:"freightCharge,omitempty"`
	Stops         []tenderStop      `json:"stops"`
	Commodities   []commodityLine   `json:"commodities,omitempty"`
}

type tenderResponse struct {
	Key         string   `json:"key"`
	Decision    string   `json:"decision"`
	Reason      string   `json:"reason,omitempty"`
	RespondedAt dateTime `json:"respondedAt"`
}

type statusEvent struct {
	Kind    string   `json:"kind"`
	At      dateTime `json:"at"`
	Stop    *int64   `json:"stop,omitempty"`
	City    string   `json:"city,omitempty"`
	State   string   `json:"state,omitempty"`
	Country string   `json:"country,omitempty"`
}

type shipmentStatus struct {
	Key              string            `json:"key"`
	BOL              string            `json:"bol,omitempty"`
	ProNumber        string            `json:"proNumber,omitempty"`
	ShipperReference string            `json:"shipperReference,omitempty"`
	Weight           *measure          `json:"weight,omitempty"`
	Pieces           *int64            `json:"pieces,omitempty"`
	Refs             map[string]string `json:"refs,omitempty"`
	Events           []statusEvent     `json:"events"`
}

type party struct {
	Name    string   `json:"name"`
	Address *address `json:"address,omitempty"`
}

type invoiceLine struct {
	Sequence    int64    `json:"sequence"`
	Code        string   `json:"code,omitempty"`
	Description string   `json:"description,omitempty"`
	Amount      money    `json:"amount"`
	Rate        string   `json:"rate,omitempty"`
	Weight      *measure `json:"weight,omitempty"`
}

type freightInvoice struct {
	Number       string            `json:"number"`
	Date         string            `json:"date"`
	DeliveryDate string            `json:"deliveryDate,omitempty"`
	Key          string            `json:"key,omitempty"`
	BOL          string            `json:"bol,omitempty"`
	ProNumber    string            `json:"proNumber,omitempty"`
	Weight       *measure          `json:"weight,omitempty"`
	BillTo       *party            `json:"billTo,omitempty"`
	Payment      string            `json:"payment,omitempty"`
	Total        money             `json:"total"`
	Lines        []invoiceLine     `json:"lines,omitempty"`
	Refs         map[string]string `json:"refs,omitempty"`
}

type readStop struct {
	Sequence int64  `json:"sequence"`
	Type     string `json:"type"`
}

type readTender struct {
	Key      string     `json:"key"`
	Purpose  string     `json:"purpose"`
	BOL      string     `json:"bol"`
	PONumber string     `json:"poNumber"`
	Payment  string     `json:"payment"`
	Weight   *measure   `json:"weight"`
	Pieces   *int64     `json:"pieces"`
	Total    *money     `json:"total"`
	Stops    []readStop `json:"stops"`
}

func at(instant time.Time, zone string) dateTime {
	if location, err := time.LoadLocation(zone); zone != "" && err == nil {
		return dateTime{At: instant.In(location).Format(time.RFC3339), Zone: zone}
	}
	return dateTime{At: instant.UTC().Format(time.RFC3339)}
}

func day(instant time.Time, zone *time.Location) string {
	return instant.In(zone).Format(time.DateOnly)
}

func pounds(weight int) *measure {
	if weight <= 0 {
		return nil
	}
	return &measure{Value: strconv.Itoa(weight), Unit: unitPounds}
}

func count(value int) *int64 {
	if value <= 0 {
		return nil
	}
	n := int64(value)
	return &n
}

func cents(total int64) money {
	return money{Amount: fmt.Sprintf("%d.%02d", total/100, total%100), Currency: currencyUSD}
}

func trenovaID(prefix string, seed uint64) string {
	return fmt.Sprintf("%s_01J%023d", prefix, seed)
}

func proNumber(seed uint64) string {
	return fmt.Sprintf("PRO%07d", seed%10_000_000)
}

func wholePounds(weight *measure) *measure {
	if weight == nil {
		return nil
	}
	amount, err := decimal.NewFromString(weight.Value)
	if err != nil || !amount.IsPositive() {
		return nil
	}
	return &measure{Value: strconv.FormatInt(amount.Round(0).IntPart(), 10), Unit: unitPounds}
}

func paymentOf(payment string) string {
	if payments[payment] {
		return payment
	}
	return ""
}

func freightTerms(tendered string) string {
	return stringutils.FirstNonEmpty(paymentOf(tendered), paymentPrepaid)
}

func partnerReference(tender *readTender) string {
	external := stringutils.TruncateRunes(strings.TrimSpace(tender.Key), maxExternalReference)
	return stringutils.FirstNonEmpty(external, tender.BOL)
}

func purchaseOrder(tender *readTender) string {
	return stringutils.TruncateRunes(strings.TrimSpace(tender.PONumber), maxPONumber)
}

func partnerRefs(bol, pro string) map[string]string {
	refs := make(map[string]string, 3)
	if bol = strings.TrimSpace(bol); bol != "" {
		refs[refBillOfLading] = bol
	}
	if pro = strings.TrimSpace(pro); pro != "" {
		refs[refCarrierPRO] = pro
	}
	return refs
}

func shipmentKey(tender *readTender, pro string) string {
	return stringutils.FirstNonEmpty(partnerReference(tender), tender.BOL, pro)
}

func stopStatusCode(pickup, departed, completesMove bool) string {
	switch {
	case pickup && departed:
		return "AF"
	case pickup:
		return "X3"
	case !departed:
		return "X1"
	case completesMove:
		return "D1"
	default:
		return "CD"
	}
}

func flowEventOf(kind string) string {
	switch kind {
	case "ArrivedPickup", "ArrivedDelivery":
		return eventStopArrived
	case "DepartedPickup", "DepartedDelivery":
		return eventStopDeparted
	case "Delivered":
		return eventDelivered
	default:
		return ""
	}
}

func responses(key string, respondedAt time.Time, zone string, v *variety) []namedDocument {
	respondedAt = v.instant(respondedAt, location(zone))
	when := at(respondedAt, v.zone(zone, otherZone))
	return []namedDocument{
		{
			name: "990 accept",
			text: "Write the 990 accepting it",
			now:  respondedAt,
			document: tenderResponse{
				Key: key, Decision: "Accept", RespondedAt: when,
			},
		},
		{
			name: "990 decline",
			text: "Write the 990 declining it",
			now:  respondedAt,
			document: tenderResponse{
				Key: key, Decision: "Decline", RespondedAt: when,
			},
		},
		{name: "990 decline with a reason", text: "Write the 990 declining it, with a reason",
			now: respondedAt, document: tenderResponse{
				Key:      key,
				Decision: "Decline",
				Reason: v.text(
					"No trucks near the pickup on the requested date",
					declineReason,
				),
				RespondedAt: when,
			}},
	}
}

type stopEvent struct {
	name      string
	text      string
	eventType string
	departed  bool
	after     time.Duration
}

var stopEvents = []stopEvent{
	{
		name:      "arrived",
		text:      "Write the 214 for arriving at stop %d",
		eventType: eventStopArrived,
		after:     30 * time.Minute,
	},
	{
		name:      "departed",
		text:      "Write the 214 for leaving stop %d",
		eventType: eventStopDeparted,
		departed:  true,
		after:     90 * time.Minute,
	},
}

type shipmentStop struct {
	sequence int64
	pickup   bool
}

func shipmentStops(load *partner.Load, tender *readTender) []shipmentStop {
	stops := make([]shipmentStop, len(load.Stops))
	for index := range load.Stops {
		stop := shipmentStop{
			sequence: int64(load.Stops[index].Sequence),
			pickup:   load.Stops[index].Kind == partner.StopPickup,
		}
		if index < len(tender.Stops) {
			read := &tender.Stops[index]
			stop.sequence, stop.pickup = read.Sequence, read.Type == stopPickup
			if stop.sequence == 0 {
				stop.sequence = int64(index + 1)
			}
		}
		stops[index] = stop
	}
	return stops
}

func finalStop(stops []shipmentStop) int {
	last := -1
	for index := range stops {
		if last < 0 || stops[index].sequence >= stops[last].sequence {
			last = index
		}
	}
	return last
}

func statuses(
	load *partner.Load,
	tender *readTender,
	sends func(event string) bool,
	v *variety,
) []namedDocument {
	var out []namedDocument
	pro := proNumber(load.Seed)
	stops := shipmentStops(load, tender)
	final := finalStop(stops)
	fallback := load.TenderedAt.Location().String()
	for index := range load.Stops {
		stop := &load.Stops[index]
		times := stopTimes(stop, v)
		city := v.text(stop.Party.City, statusCity)
		for event, e := range stopEvents {
			code := stopStatusCode(stops[index].pickup, e.departed, index == final)
			kind := statusKinds[code]
			if !sends(flowEventOf(kind)) {
				continue
			}
			sequence := stops[index].sequence
			occurred := times[event]
			status := shipmentStatus{
				Key:              shipmentKey(tender, pro),
				BOL:              tender.BOL,
				ProNumber:        pro,
				ShipperReference: partnerReference(tender),
				Weight:           wholePounds(tender.Weight),
				Pieces:           tender.Pieces,
				Refs:             partnerRefs(tender.BOL, pro),
				Events: []statusEvent{{
					Kind:  kind,
					At:    at(occurred, v.zone(stop.Party.Zone, fallback)),
					Stop:  &sequence,
					City:  city,
					State: stop.Party.State,
				}},
			}
			if v.chance(2) {
				status.Events[0].Country = "US"
			}
			out = append(out, namedDocument{
				name:     fmt.Sprintf("214 stop %d %s", sequence, e.name),
				text:     fmt.Sprintf(e.text, sequence),
				document: status,
				now:      occurred.Add(time.Minute),
			})
		}
	}
	return out
}

func stopTimes(stop *partner.Stop, v *variety) []time.Time {
	arrived := v.instant(stop.Window.Start.Add(stopEvents[0].after), location(stop.Party.Zone))
	dwell := stopEvents[1].after - stopEvents[0].after
	if v != nil {
		dwell = time.Duration(20+v.rng.IntN(160))*time.Minute +
			time.Duration(v.rng.IntN(60))*time.Second
	}
	return []time.Time{arrived, arrived.Add(dwell)}
}

func invoice(
	load *partner.Load,
	tender *readTender,
	zone *time.Location,
	v *variety,
) namedDocument {
	last := load.Stops[len(load.Stops)-1].Window.End
	issued := v.instant(time.Date(last.Year(), last.Month(), last.Day()+1, 12, 0, 0, 0, zone), zone)
	total := cents(load.TotalCents)
	if tender.Total != nil && tender.Total.Amount != "" {
		total = money{Amount: tender.Total.Amount, Currency: currencyUSD}
	}
	pro := proNumber(load.Seed)
	bol := tender.BOL
	refs := partnerRefs(bol, pro)
	if po := purchaseOrder(tender); po != "" {
		refs[refPurchaseOrder] = po
	}
	billed := freightInvoice{
		Number:    fmt.Sprintf("INV-%06d", load.Seed%1_000_000),
		Date:      day(issued, zone),
		Key:       shipmentKey(tender, pro),
		BOL:       bol,
		ProNumber: pro,
		BillTo:    billTo(load, v),
		Payment:   freightTerms(tender.Payment),
		Total:     total,
		Lines:     invoiceLines(load, total, v),
		Refs:      refs,
	}
	if v != nil && !v.chance(3) {
		billed.Weight = pounds(load.Weight())
	}
	if v != nil && !v.chance(5) {
		billed.DeliveryDate = day(last, zone)
	}
	return namedDocument{
		name:     "210",
		text:     "Write the 210 at the tendered total",
		document: billed,
		now:      issued,
	}
}

func billTo(load *partner.Load, v *variety) *party {
	billed := &party{Name: v.text(load.BillTo.Name, invoiceName)}
	if load.BillTo.Street == "" || load.BillTo.City == "" || v.chance(7) {
		return billed
	}
	billed.Address = &address{
		Line1:   v.text(load.BillTo.Street, invoiceStreet),
		City:    v.text(load.BillTo.City, invoiceCity),
		State:   load.BillTo.State,
		Postal:  load.BillTo.Zip,
		Country: load.BillTo.Country,
	}
	if v.chance(4) {
		billed.Address.Line2 = v.text("SUITE 200", invoiceSuite)
	}
	return billed
}

func invoiceLines(load *partner.Load, total money, v *variety) []invoiceLine {
	totalCents := centsOf(total.Amount)
	extras, amounts := v.charges(totalCents)
	linehaul := totalCents
	for _, amount := range amounts {
		linehaul -= amount
	}
	first := invoiceLine{
		Sequence:    1,
		Code:        freightCode,
		Description: v.text("Linehaul", invoiceLineFor),
		Amount:      total,
	}
	if v == nil {
		return []invoiceLine{first}
	}
	first.Amount = cents(linehaul)
	if !v.chance(3) {
		first.Rate = decimal.New(load.RateCents, -2).String()
	}
	if v.chance(2) {
		first.Weight = pounds(load.Weight())
	}
	if v.chance(10) {
		first.Description = ""
	}
	lines := []invoiceLine{first}
	for index, extra := range extras {
		lines = append(lines, invoiceLine{
			Sequence:    int64(index + 2),
			Code:        extra.code,
			Description: v.text(extra.description, invoiceLineFor),
			Amount:      cents(amounts[index]),
		})
	}
	return lines
}

func centsOf(amount string) int64 {
	value, err := decimal.NewFromString(amount)
	if err != nil {
		return 0
	}
	return value.Shift(2).Round(0).IntPart()
}

func location(zone string) *time.Location {
	if loaded, err := time.LoadLocation(zone); zone != "" && err == nil {
		return loaded
	}
	return time.UTC
}
