package telematicsports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type StopVisitReport struct {
	TenantInfo   pagination.TenantInfo
	Event        *telematics.TelematicsEvent
	MoveID       pulid.ID
	StopID       pulid.ID
	Departure    bool
	ProviderStop bool
}

type StopVisitVerdict struct {
	Outcome telematics.StopOutcome
	Reason  string
}

type StopVisitRecorder interface {
	RecordStopVisit(ctx context.Context, report *StopVisitReport) (*StopVisitVerdict, error)
}
