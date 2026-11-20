package partner

import (
	"slices"
	"time"
)

type StopKind string

const (
	StopPickup   StopKind = "pickup"
	StopDelivery StopKind = "delivery"
)

type Load struct {
	Seed       uint64
	Number     string
	TenderedAt time.Time
	Payment    string
	BOL        string
	PO         string
	Note       string
	BillTo     Party
	Equipment  Equipment
	Stops      []Stop
	Miles      int
	RateCents  int64
	TotalCents int64
}

type Party struct {
	Code    string
	Name    string
	Street  string
	City    string
	State   string
	Zip     string
	Country string
	Zone    string
	Contact string
	Phone   string
}

type Equipment struct {
	Code     string
	LengthFt int
	Reefer   bool
	TempF    int
}

type Window struct {
	Start time.Time
	End   time.Time
}

type Stop struct {
	Sequence     int
	Kind         StopKind
	Party        Party
	Window       Window
	Weight       int
	Pieces       int
	PickupRef    string
	Appointment  string
	Note         string
	Lines        []Line
	Temperatures []TempRange
}

type Line struct {
	Number      int
	Description string
	Code        string
	Weight      int
	Pieces      int
	Hazmat      *Hazmat
	Temperature *TempRange
}

type TempRange struct {
	MinF int
	MaxF int
}

func (r TempRange) overlaps(other TempRange) bool {
	return r.MinF <= other.MaxF && other.MinF <= r.MaxF
}

type Hazmat struct {
	UN    string
	Class string
	Name  string
}

func (l *Load) Weight() int {
	total := 0
	for index := range l.Stops {
		if l.Stops[index].Kind == StopPickup {
			total += l.Stops[index].Weight
		}
	}
	return total
}

func (l *Load) Pieces() int {
	total := 0
	for index := range l.Stops {
		if l.Stops[index].Kind == StopPickup {
			total += l.Stops[index].Pieces
		}
	}
	return total
}

func (l *Load) Hazardous() bool {
	for index := range l.Stops {
		for line := range l.Stops[index].Lines {
			if l.Stops[index].Lines[line].Hazmat != nil {
				return true
			}
		}
	}
	return false
}

func (l *Load) Temperature() (TempRange, bool) {
	var all TempRange
	found := false
	for index := range l.Stops {
		for _, stop := range l.Stops[index].Temperatures {
			if !found {
				all, found = stop, true
				continue
			}
			all.MinF = max(all.MinF, stop.MinF)
			all.MaxF = min(all.MaxF, stop.MaxF)
		}
	}
	return all, found
}

func (l *Load) TemperatureConflict() bool {
	all, found := l.Temperature()
	return found && all.MinF > all.MaxF
}

func (l *Load) Clone() *Load {
	clone := *l
	clone.Stops = make([]Stop, len(l.Stops))
	for index := range l.Stops {
		stop := l.Stops[index]
		stop.Temperatures = slices.Clone(l.Stops[index].Temperatures)
		stop.Lines = make([]Line, len(l.Stops[index].Lines))
		for line := range l.Stops[index].Lines {
			copied := l.Stops[index].Lines[line]
			if copied.Hazmat != nil {
				hazmat := *copied.Hazmat
				copied.Hazmat = &hazmat
			}
			if copied.Temperature != nil {
				temperature := *copied.Temperature
				copied.Temperature = &temperature
			}
			stop.Lines[line] = copied
		}
		clone.Stops[index] = stop
	}
	return &clone
}
