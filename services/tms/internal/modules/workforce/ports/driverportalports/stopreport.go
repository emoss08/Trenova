package driverportalports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type StopReport struct {
	TenantInfo    pagination.TenantInfo
	MoveID        pulid.ID
	StopID        pulid.ID
	Departure     bool
	OccurredAt    *int64
	ClientEventID string
	ReportedByID  pulid.ID
}

type StopReporter interface {
	ReportDriverStop(ctx context.Context, report *StopReport) error
}
