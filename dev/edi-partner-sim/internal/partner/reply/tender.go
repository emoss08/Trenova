package reply

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

var (
	errNoKey   = errors.New("reply: the tender has no shipment ID (B2-04)")
	errNoStops = errors.New("reply: the tender has fewer than two stops")
)

var zoneCodes = map[string]string{
	"ET": "America/New_York",
	"CT": "America/Chicago",
	"MT": "America/Denver",
	"PT": "America/Los_Angeles",
	"UT": "UTC",
}

type Tender struct {
	Interchange string
	Group       string
	Control     string
	SCAC        string
	Key         string
	Purpose     string
	Payment     string
	BOL         string
	PO          string
	Equipment   string
	LengthFt    int
	WeightLb    int
	Pieces      int
	TotalCents  int64
	Stops       []TenderStop
}

type TenderStop struct {
	Sequence int
	Kind     partner.StopKind
	Name     string
	Code     string
	Street   string
	City     string
	State    string
	Zip      string
	Country  string
	Start    time.Time
	End      time.Time
	WeightLb int
	Pieces   int
}

func (c *Carrier) Receive(raw []byte) ([]*Tender, *oracle.Judgement, error) {
	judgement, err := c.Guides.Judge(raw)
	if err != nil {
		return nil, nil, err
	}
	var tenders []*Tender
	for _, report := range judgement.Reports() {
		if report.Set != "204" {
			continue
		}
		tender, readErr := ReadTender(report, c.Definition.Partner.Location())
		if readErr != nil {
			return nil, judgement, readErr
		}
		tenders = append(tenders, tender)
	}
	return tenders, judgement, nil
}

func ReadTender(report *oracle.Report, zone *time.Location) (*Tender, error) {
	if report.Set != "204" {
		return nil, fmt.Errorf("reply: set %s is not a load tender", report.Set)
	}
	tender := &Tender{
		Interchange: report.InterchangeControlNumber,
		Group:       report.GroupControlNumber,
		Control:     report.ControlNumber,
	}
	var stop *TenderStop
	for index := range report.Segments {
		segment := &report.Segments[index]
		if segment.ID == "S5" {
			tender.Stops = append(tender.Stops, TenderStop{
				Sequence: atoi(segment.Element(1)),
				Kind:     stopKind(segment.Element(2)),
				WeightLb: atoi(segment.Element(3)),
				Pieces:   atoi(segment.Element(5)),
			})
			stop = &tender.Stops[len(tender.Stops)-1]
			continue
		}
		if segment.ID == "L3" {
			stop = nil
		}
		if stop != nil {
			readStopSegment(stop, segment, zone)
			continue
		}
		readHeadingSegment(tender, segment)
	}
	if tender.Key == "" {
		return nil, errNoKey
	}
	if len(tender.Stops) < 2 {
		return nil, fmt.Errorf("%w: %s", errNoStops, tender.Key)
	}
	return tender, nil
}

func readHeadingSegment(tender *Tender, segment *oracle.Segment) {
	switch segment.ID {
	case "B2":
		tender.SCAC = segment.Element(2)
		tender.Key = segment.Element(4)
		tender.Payment = segment.Element(6)
	case "B2A":
		tender.Purpose = segment.Element(1)
	case "L11":
		switch segment.Element(2) {
		case "BM":
			tender.BOL = segment.Element(1)
		case "PO":
			tender.PO = segment.Element(1)
		}
	case "N7":
		tender.Equipment = segment.Element(11)
		tender.LengthFt = atoi(segment.Element(15)) / 100
	case "L3":
		tender.WeightLb = atoi(segment.Element(1))
		tender.TotalCents = int64(atoi(segment.Element(5)))
		tender.Pieces = atoi(segment.Element(11))
	}
}

func readStopSegment(stop *TenderStop, segment *oracle.Segment, zone *time.Location) {
	switch segment.ID {
	case "G62":
		at, ok := readTime(segment, zone)
		if !ok {
			return
		}
		switch segment.Element(1) {
		case "37", "53":
			stop.Start = at
		case "38", "54":
			stop.End = at
		}
	case "N1":
		stop.Name = segment.Element(2)
		stop.Code = segment.Element(4)
	case "N3":
		stop.Street = segment.Element(1)
	case "N4":
		stop.City = segment.Element(1)
		stop.State = segment.Element(2)
		stop.Zip = segment.Element(3)
		stop.Country = segment.Element(4)
	}
}

func readTime(segment *oracle.Segment, zone *time.Location) (time.Time, bool) {
	location := zone
	if name, ok := zoneCodes[segment.Element(5)]; ok {
		if loaded, err := time.LoadLocation(name); err == nil {
			location = loaded
		}
	}
	at, err := time.ParseInLocation(
		"200601021504",
		segment.Element(2)+segment.Element(4),
		location,
	)
	return at, err == nil
}

func stopKind(code string) partner.StopKind {
	if code == "UL" || code == "CU" {
		return partner.StopDelivery
	}
	return partner.StopPickup
}

func atoi(value string) int {
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return number
}

func (t *Tender) rng(seed uint64, what string) *rand.Rand {
	h := fnv.New64a()
	_, _ = h.Write([]byte(t.Key + " " + what))
	return rand.New(rand.NewPCG(seed, h.Sum64()))
}

func (t *Tender) first(kind partner.StopKind) *TenderStop {
	for index := range t.Stops {
		if t.Stops[index].Kind == kind {
			return &t.Stops[index]
		}
	}
	return &t.Stops[0]
}

func (t *Tender) last(kind partner.StopKind) *TenderStop {
	for index := len(t.Stops) - 1; index >= 0; index-- {
		if t.Stops[index].Kind == kind {
			return &t.Stops[index]
		}
	}
	return &t.Stops[len(t.Stops)-1]
}

func (t *Tender) view() map[string]any {
	return map[string]any{
		"key":       t.Key,
		"scac":      t.SCAC,
		"payment":   t.Payment,
		"bol":       t.BOL,
		"po":        t.PO,
		"equipment": t.Equipment,
		"weight":    t.WeightLb,
		"pieces":    t.Pieces,
		"total":     t.TotalCents,
	}
}
