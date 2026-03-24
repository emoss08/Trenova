package shipmentstate

const DefaultAutoCancelThresholdDays int8 = 30

func ResolveAutoCancelThresholdDays(thresholdDays int8) int8 {
	if thresholdDays <= 0 {
		return DefaultAutoCancelThresholdDays
	}

	return thresholdDays
}
