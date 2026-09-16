package fiscalyearservice

import (
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

func validateEditable(original, updated *fiscalyear.FiscalYear) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if original.Status == fiscalyear.StatusPermanentlyClosed {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Permanently closed fiscal years cannot be changed",
		)
		return multiErr
	}

	if updated.Year != original.Year {
		addStructureLocked(multiErr, "year")
	}
	if updated.StartDate != original.StartDate {
		addStructureLocked(multiErr, "startDate")
	}
	if updated.EndDate != original.EndDate {
		addStructureLocked(multiErr, "endDate")
	}
	if updated.IsCalendarYear != original.IsCalendarYear {
		addStructureLocked(multiErr, "isCalendarYear")
	}
	if original.Status == fiscalyear.StatusClosed && updated.Name != original.Name {
		multiErr.Add(
			"name",
			errortypes.ErrInvalid,
			"The name of a closed fiscal year cannot be changed",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func preserveLifecycle(original, updated *fiscalyear.FiscalYear) {
	updated.Status = original.Status
	updated.IsCurrent = original.IsCurrent
	updated.ClosedAt = original.ClosedAt
	updated.ClosedByID = original.ClosedByID
	updated.LockedAt = original.LockedAt
	updated.LockedByID = original.LockedByID
	updated.ReopenedAt = original.ReopenedAt
	updated.ReopenedByID = original.ReopenedByID
	updated.ReopenReason = original.ReopenReason
	updated.CreatedAt = original.CreatedAt
	updated.Periods = nil
}

func addStructureLocked(multiErr *errortypes.MultiError, field string) {
	multiErr.Add(
		field,
		errortypes.ErrInvalid,
		"This field cannot be changed after the fiscal year is created",
	)
}

func normalizeDateBounds(entity *fiscalyear.FiscalYear) error {
	if entity.StartDate > 0 {
		start, err := timeutils.DayStartUnix(entity.StartDate, "UTC")
		if err != nil {
			return err
		}
		entity.StartDate = start
	}

	if entity.EndDate > 0 {
		end, err := timeutils.DayEndUnix(entity.EndDate, "UTC")
		if err != nil {
			return err
		}
		entity.EndDate = end
	}

	return nil
}
