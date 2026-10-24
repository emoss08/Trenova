package shipmentstate

const DefaultAutoCancelThresholdDays int16 = 30

func ResolveAutoCancelThresholdDays(thresholdDays int16) int16 {
	if thresholdDays <= 0 {
		return DefaultAutoCancelThresholdDays
	}

	return thresholdDays
}
