package trackingeventservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/driverportalports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const maxClientEventIDLength = 128

func (s *Service) ReportDriverStop(
	ctx context.Context,
	report *driverportalports.StopReport,
) error {
	if report == nil {
		return errortypes.NewValidationError("", errortypes.ErrInvalid, "Request is required")
	}

	clientEventID := strings.TrimSpace(report.ClientEventID)
	if clientEventID != "" && !stringutils.IsCompactToken(clientEventID, maxClientEventIDLength) {
		return errortypes.NewValidationError(
			"clientEventId",
			errortypes.ErrInvalid,
			"Client event ID must be 128 characters or fewer with no spaces",
		)
	}

	action := shipmentmoveports.StopActualActionArrive
	if report.Departure {
		action = shipmentmoveports.StopActualActionDepart
	}
	sourceKey := ""
	if clientEventID != "" {
		sourceKey = report.ReportedByID.String() + ":" + clientEventID
	}

	_, err := s.RecordReported(ctx, &trackingeventports.RecordReportedStopEventParams{
		TenantInfo:   report.TenantInfo,
		MoveID:       report.MoveID,
		StopID:       report.StopID,
		Action:       action,
		OccurredAt:   tapTime(report.OccurredAt, timeutils.NowUnix()),
		Source:       trackingevent.SourceDriver,
		SourceKey:    sourceKey,
		ReportedByID: report.ReportedByID,
	})
	return err
}

func tapTime(occurredAt *int64, now int64) *int64 {
	if occurredAt == nil || *occurredAt <= now {
		return occurredAt
	}
	return &now
}
