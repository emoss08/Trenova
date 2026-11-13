package demandforecast

var LeadHours = [...]int64{0, 2, 4, 8, 12, 24, 36, 48, 72, 96, 120, 168, 336}

func BookedShare(shares []float64, leadHours float64) float64 {
	if len(shares) != len(LeadHours) || leadHours <= 0 {
		if leadHours <= 0 {
			return 1
		}
		return 0
	}
	last := len(LeadHours) - 1
	if leadHours >= float64(LeadHours[last]) {
		return shares[last]
	}
	for i := 1; i <= last; i++ {
		upper := float64(LeadHours[i])
		if leadHours > upper {
			continue
		}
		lower := float64(LeadHours[i-1])
		t := (leadHours - lower) / (upper - lower)
		return shares[i-1] + t*(shares[i]-shares[i-1])
	}

	return shares[last]
}

func LeadBucket(leadSeconds int64) int {
	hours := leadSeconds / 3600
	bucket := 0
	for i, edge := range LeadHours {
		if hours >= edge {
			bucket = i
		}
	}

	return bucket
}
