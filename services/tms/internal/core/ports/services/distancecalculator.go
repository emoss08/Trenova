package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
)

type DistanceCalculationRequest struct {
	MoveID         pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

type DistanceCalculationResult struct {
	Distance float64
	Source   DistanceSource
}

type DistanceSource string

const (
	DistanceSourceOverride   = DistanceSource("override")
	DistanceSourceCalculated = DistanceSource("calculated")
)

type DistanceCalculatorService interface {
	CalculateDistance(
		ctx context.Context,
		tx bun.IDB,
		req *DistanceCalculationRequest,
	) (*DistanceCalculationResult, error)
}
