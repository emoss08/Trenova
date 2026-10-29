package volumegen

import (
	"math"
	_ "time/tzdata"
)

func NearestCity(from, to *CityRecord, share float64) *CityRecord {
	lat := from.Lat + (to.Lat-from.Lat)*share
	lon := from.Lon + (to.Lon-from.Lon)*share
	point := &CityRecord{Lat: lat, Lon: lon}
	best := &Cities[0]
	bestMiles := math.MaxFloat64
	for i := range Cities {
		if d := HaversineMiles(point, &Cities[i]); d < bestMiles {
			best, bestMiles = &Cities[i], d
		}
	}
	return best
}
