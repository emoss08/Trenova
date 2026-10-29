package documentanalysis

import (
	"github.com/emoss08/trenova/shared/floatutils"
)

func ClampConfidence(value float64) float64 {
	return floatutils.Clamp(value, 0, maxConfidence)
}
