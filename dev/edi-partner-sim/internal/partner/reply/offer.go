package reply

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
)

type place struct {
	name   string
	code   string
	street string
	city   string
	state  string
	postal string
	zone   string
}

var origins = []place{
	{"Front Range Mill", "FRM-1", "4100 Brighton Blvd", "Denver", "CO", "80216", "America/Denver"},
	{"Wasatch Paper Converting", "WPC-2", "2250 S 900 W", "Salt Lake City", "UT", "84119",
		"America/Denver"},
	{"Rio Grande Fasteners", "RGF-1", "3301 Princeton Dr NE", "Albuquerque", "NM", "87107",
		"America/Denver"},
	{"High Plains Cooperative", "HPC-4", "1200 W Lincolnway", "Cheyenne", "WY", "82001",
		"America/Denver"},
}

var consignees = []place{
	{"Prairie Supply", "", "900 N Broadway", "Wichita", "KS", "67214", "America/Chicago"},
	{"Big Muddy Distribution", "BMD-7", "4800 Gateway Dr", "Omaha", "NE", "68110",
		"America/Chicago"},
	{"Panhandle Grocers", "", "1501 Amarillo Blvd E", "Amarillo", "TX", "79107",
		"America/Chicago"},
	{"Copper State Building Supply", "CSB-3", "2828 W Buckeye Rd", "Phoenix", "AZ", "85009",
		"America/Phoenix"},
	{"Snake River Hardware", "", "655 W Amity Rd", "Boise", "ID", "83705", "America/Boise"},
	{"Black Hills Feed", "BHF-2", "1400 E North St", "Rapid City", "SD", "57701",
		"America/Denver"},
	{"Tulsa Pipe and Valve", "TPV-5", "5310 S 33rd W Ave", "Tulsa", "OK", "74107",
		"America/Chicago"},
	{"Las Vegas Restaurant Supply", "", "3950 W Tompkins Ave", "Las Vegas", "NV", "89103",
		"America/Los_Angeles"},
}

var equipmentChoices = []struct {
	code   string
	weight int
}{{"VAN", 6}, {"REEFER", 2}, {"FLATBED", 2}}

type measure struct {
	Value string `json:"value"`
	Unit  string `json:"unit"`
}

type money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type instant struct {
	At   string `json:"at"`
	Zone string `json:"zone"`
}

type window struct {
	Start instant `json:"start"`
	End   instant `json:"end"`
}

type address struct {
	Line1   string `json:"line1"`
	City    string `json:"city"`
	State   string `json:"state"`
	Postal  string `json:"postal"`
	Country string `json:"country"`
}

type offeredStop struct {
	Sequence int     `json:"sequence"`
	Type     string  `json:"type"`
	Name     string  `json:"name"`
	Code     string  `json:"code,omitempty"`
	Address  address `json:"address"`
	Window   window  `json:"window"`
	Weight   measure `json:"weight"`
	Pieces   int     `json:"pieces"`
	start    time.Time
}

type OfferedLoad struct {
	Key       string        `json:"key"`
	Purpose   string        `json:"purpose"`
	BOL       string        `json:"bol"`
	PONumber  string        `json:"poNumber,omitempty"`
	Payment   string        `json:"payment"`
	Equipment string        `json:"equipment"`
	Terms     string        `json:"-"`
	Weight    measure       `json:"weight"`
	Pieces    int           `json:"pieces"`
	Total     money         `json:"total"`
	Stops     []offeredStop `json:"stops"`
}

func Offer(seed uint64) *OfferedLoad {
	rng := rand.New(rand.NewPCG(seed, 204204))
	load := &OfferedLoad{
		Key:       "S2603" + digits(rng, 7),
		Purpose:   "Original",
		BOL:       "BOL-" + digits(rng, 6),
		Payment:   "Prepaid",
		Equipment: pickEquipment(rng),
		Terms:     "Prepaid",
	}
	if rng.IntN(10) < 7 {
		load.PONumber = "PO-" + digits(rng, 5)
	}
	if rng.IntN(10) == 0 {
		load.Terms = "ThirdParty"
	}
	weight := 10 * (800 + rng.IntN(3_601))
	pieces := 4 + rng.IntN(23)
	load.Weight = pounds(weight)
	load.Pieces = pieces
	load.Total = dollars(int64(500 * (180 + rng.IntN(661))))
	origin := origins[rng.IntN(len(origins))]
	count := 1 + rng.IntN(3)
	order := rng.Perm(len(consignees))[:count]
	day := time.Date(2026, time.March, 2, 0, 0, 0, 0, zone(origin.zone)).
		AddDate(0, 0, rng.IntN(45))
	start := time.Date(day.Year(), day.Month(), day.Day(), 6+rng.IntN(8), 0, 0, 0, day.Location())
	load.Stops = append(load.Stops, stopAt(1, "Pickup", origin, start, weight, pieces))
	weights := share(weight, count, rng)
	shares := share(pieces, count, rng)
	for index, chosen := range order {
		destination := consignees[chosen]
		previous := load.Stops[len(load.Stops)-1].start
		next := previous.Add(time.Duration(9+rng.IntN(20)) * time.Hour).In(zone(destination.zone))
		next = workingHour(next.Truncate(time.Hour))
		load.Stops = append(load.Stops,
			stopAt(index+2, "Delivery", destination, next, weights[index], shares[index]))
	}
	return load
}

func (l *OfferedLoad) JSON() ([]byte, error) {
	encoded, err := sonic.ConfigStd.Marshal(l)
	if err != nil {
		return nil, fmt.Errorf("reply: encode the offered load %s: %w", l.Key, err)
	}
	return encoded, nil
}

func stopAt(sequence int, kind string, at place, start time.Time, weight, pieces int) offeredStop {
	return offeredStop{
		Sequence: sequence,
		Type:     kind,
		Name:     at.name,
		Code:     at.code,
		Address: address{
			Line1:   at.street,
			City:    at.city,
			State:   at.state,
			Postal:  at.postal,
			Country: "US",
		},
		Window: window{
			Start: instant{At: start.Format(time.RFC3339), Zone: at.zone},
			End:   instant{At: start.Add(2 * time.Hour).Format(time.RFC3339), Zone: at.zone},
		},
		Weight: pounds(weight),
		Pieces: pieces,
		start:  start,
	}
}

func workingHour(at time.Time) time.Time {
	switch {
	case at.Hour() < 6:
		return time.Date(at.Year(), at.Month(), at.Day(), 7, 0, 0, 0, at.Location())
	case at.Hour() >= 17:
		next := at.AddDate(0, 0, 1)
		return time.Date(next.Year(), next.Month(), next.Day(), 7, 0, 0, 0, at.Location())
	default:
		return at
	}
}

func share(total, parts int, rng *rand.Rand) []int {
	shares := make([]int, parts)
	remaining := total
	for index := range parts - 1 {
		left := parts - index
		even := remaining / left
		part := max(even/2+rng.IntN(max(even, 1)), 1)
		part = min(part, remaining-(left-1))
		shares[index] = part
		remaining -= part
	}
	shares[parts-1] = remaining
	return shares
}

func pickEquipment(rng *rand.Rand) string {
	total := 0
	for _, choice := range equipmentChoices {
		total += choice.weight
	}
	roll := rng.IntN(total)
	for _, choice := range equipmentChoices {
		if roll < choice.weight {
			return choice.code
		}
		roll -= choice.weight
	}
	return equipmentChoices[0].code
}

func pounds(weight int) measure {
	return measure{Value: strconv.Itoa(weight), Unit: "lb"}
}

func dollars(cents int64) money {
	return money{Amount: fmt.Sprintf("%d.%02d", cents/100, cents%100), Currency: "USD"}
}

func zone(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return location
}
