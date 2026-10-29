package floatutils

func MeanPtr(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	mean := total / float64(len(values))

	return &mean
}
