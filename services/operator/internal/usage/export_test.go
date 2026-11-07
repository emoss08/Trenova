package usage

import (
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

func MapReadings(
	allowances []instanceapi.SpecAllowance,
	in []tms.Reading,
) ([]instanceapi.UsageReading, int) {
	return newMapping(allowances).readings(in)
}

func MapState(allowances []instanceapi.SpecAllowance, ack instanceapi.UsageAck) tms.State {
	return newMapping(allowances).state(ack)
}

func Chunks(readings []instanceapi.UsageReading, size int) [][]instanceapi.UsageReading {
	return chunks(readings, size)
}
