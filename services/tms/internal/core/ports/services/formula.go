package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
)

type FormulaCalculator interface {
	Calculate(
		ctx context.Context,
		req *formulatemplatetypes.CalculateRequest,
	) (*formulatemplatetypes.CalculateResponse, error)
}
