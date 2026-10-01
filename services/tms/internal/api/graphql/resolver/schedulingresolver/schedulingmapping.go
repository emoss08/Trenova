package schedulingresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func shiftTemplateFromInput(
	input gqlmodel.ShiftTemplateInput,
	tenant pagination.TenantInfo,
) *worker.ShiftTemplate {
	status := domaintypes.StatusActive
	if input.Status != nil {
		status = *input.Status
	}

	return &worker.ShiftTemplate{
		OrganizationID:  tenant.OrgID,
		BusinessUnitID:  tenant.BuID,
		Status:          status,
		Code:            input.Code,
		Name:            input.Name,
		Description:     base.StringValue(input.Description),
		Color:           base.StringValue(input.Color),
		DaysOfWeek:      input.DaysOfWeek,
		StartMinute:     int16(input.StartMinute),     //nolint:gosec // validated 0..1439
		DurationMinutes: int16(input.DurationMinutes), //nolint:gosec // validated 1..1440
		CycleWeeks:      int16(input.CycleWeeks),      //nolint:gosec // validated 1..8
	}
}

func shiftAssignmentFromInput(
	input gqlmodel.AssignShiftInput,
	tenant pagination.TenantInfo,
) (*worker.WorkerShiftAssignment, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, base.InvalidIDError("workerId", "Worker is invalid")
	}
	templateID, err := pulid.MustParse(input.ShiftTemplateID)
	if err != nil {
		return nil, base.InvalidIDError("shiftTemplateId", "Shift is invalid")
	}

	return &worker.WorkerShiftAssignment{
		OrganizationID:   tenant.OrgID,
		BusinessUnitID:   tenant.BuID,
		WorkerID:         workerID,
		ShiftTemplateID:  templateID,
		EffectiveFrom:    int64(input.EffectiveFrom),
		CycleOffsetWeeks: intutils.SafeToInt16(base.IntValue(input.CycleOffsetWeeks)),
		Notes:            base.StringValue(input.Notes),
	}, nil
}

func availabilityPreferenceFromInput(
	input gqlmodel.SetAvailabilityPreferenceInput,
	tenant pagination.TenantInfo,
) (*worker.WorkerAvailabilityPreference, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, base.InvalidIDError("workerId", "Worker is invalid")
	}

	return &worker.WorkerAvailabilityPreference{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		WorkerID:       workerID,
		DayOfWeek:      int16(input.DayOfWeek), //nolint:gosec // validated 0..6
		Preference:     input.Preference,
		Note:           base.StringValue(input.Note),
	}, nil
}

func shiftSwapFromInput(
	input gqlmodel.ProposeShiftSwapInput,
	tenant pagination.TenantInfo,
) (*worker.ShiftSwapRequest, error) {
	requestingID, err := pulid.MustParse(input.RequestingWorkerID)
	if err != nil {
		return nil, base.InvalidIDError("requestingWorkerId", "Worker is invalid")
	}
	counterpartyID, err := base.OptionalID(input.CounterpartyWorkerID)
	if err != nil {
		return nil, base.InvalidIDError("counterpartyWorkerId", "Worker is invalid")
	}

	entity := &worker.ShiftSwapRequest{
		OrganizationID:       tenant.OrgID,
		BusinessUnitID:       tenant.BuID,
		RequestingWorkerID:   requestingID,
		CounterpartyWorkerID: counterpartyID,
		ShiftDate:            int64(input.ShiftDate),
		Reason:               base.StringValue(input.Reason),
	}
	if input.CounterpartyShiftDate != nil {
		counterDate := int64(*input.CounterpartyShiftDate)
		entity.CounterpartyShiftDate = &counterDate
	}

	return entity, nil
}

// rotaToGQL carries the composed board out. The state is a domain string and
// the enum is a generated one; they are spelled the same on purpose, so the
// conversion stays a cast rather than a switch that drifts.
func rotaToGQL(rota *worker.Rota) *gqlmodel.Rota {
	if rota == nil {
		return &gqlmodel.Rota{Rows: []*gqlmodel.RotaRow{}}
	}

	out := &gqlmodel.Rota{
		WeekStart:     int(rota.WeekStart),
		WeekEnd:       int(rota.WeekEnd),
		Weeks:         rota.Weeks,
		Rows:          make([]*gqlmodel.RotaRow, 0, len(rota.Rows)),
		ScheduledDays: rota.ScheduledDays,
		Conflicts:     rota.Conflicts,
	}

	for _, row := range rota.Rows {
		gqlRow := &gqlmodel.RotaRow{
			WorkerID:      row.WorkerID.String(),
			Name:          row.Name,
			FleetCode:     base.EmptyToNil(row.FleetCode),
			FleetColor:    base.EmptyToNil(row.FleetColor),
			ShiftCode:     base.EmptyToNil(row.ShiftCode),
			ShiftName:     base.EmptyToNil(row.ShiftName),
			ShiftColor:    base.EmptyToNil(row.ShiftColor),
			Days:          make([]*gqlmodel.RotaDay, 0, len(row.Days)),
			ScheduledDays: row.ScheduledDays,
			Conflicts:     row.Conflicts,
		}
		for _, day := range row.Days {
			gqlRow.Days = append(gqlRow.Days, rotaDayToGQL(day))
		}
		out.Rows = append(out.Rows, gqlRow)
	}

	return out
}

func rotaDayToGQL(day worker.RotaDay) *gqlmodel.RotaDay {
	out := &gqlmodel.RotaDay{
		Date:            int(day.Date),
		State:           day.State,
		Scheduled:       day.Scheduled,
		StartMinute:     int(day.StartMinute),
		DurationMinutes: int(day.DurationMinutes),
		AssignmentCount: day.AssignmentCount,
		IsConflict:      day.IsConflict(),
	}
	// A worker who has said nothing about a weekday has no preference, which
	// is not the same as having said they are available.
	if day.Preference != "" {
		preference := day.Preference
		out.Preference = &preference
	}

	return out
}

// swapPermissionFor maps the answer onto the grant it needs. Accepting and
// declining are the counterparty's own act and are gated by read, because the
// service refuses an answer from anybody but the driver it was offered to.
func swapPermissionFor(status worker.ShiftSwapStatus) permission.Operation {
	switch status {
	case worker.SwapApproved:
		return permission.OpApprove
	case worker.SwapRejected:
		return permission.OpReject
	case worker.SwapWithdrawn:
		return permission.OpCancel
	case worker.SwapAccepted, worker.SwapDeclined, worker.SwapProposed:
		return permission.OpRead
	default:
		return permission.OpApprove
	}
}

func parseIDList(values []string) ([]pulid.ID, error) {
	if len(values) == 0 {
		return nil, nil
	}

	out := make([]pulid.ID, 0, len(values))
	for _, value := range values {
		id, err := pulid.MustParse(value)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}

	return out, nil
}
