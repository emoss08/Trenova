package schedulingresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func shiftTemplateFromInput(
	input gqlmodel.ShiftTemplateInput,
	tenant pagination.TenantInfo,
) *workerschedule.ShiftTemplate {
	status := domaintypes.StatusActive
	if input.Status != nil {
		status = *input.Status
	}

	return &workerschedule.ShiftTemplate{
		OrganizationID:  tenant.OrgID,
		BusinessUnitID:  tenant.BuID,
		Status:          status,
		Code:            input.Code,
		Name:            input.Name,
		Description:     gqlscalar.StringValue(input.Description),
		Color:           gqlscalar.StringValue(input.Color),
		DaysOfWeek:      input.DaysOfWeek,
		StartMinute:     int16(input.StartMinute),     //nolint:gosec // validated 0..1439
		DurationMinutes: int16(input.DurationMinutes), //nolint:gosec // validated 1..1440
		CycleWeeks:      int16(input.CycleWeeks),      //nolint:gosec // validated 1..8
	}
}

func shiftAssignmentFromInput(
	input gqlmodel.AssignShiftInput,
	tenant pagination.TenantInfo,
) (*workerschedule.WorkerShiftAssignment, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, gqlid.InvalidIDError("workerId", "Worker is invalid")
	}
	templateID, err := pulid.MustParse(input.ShiftTemplateID)
	if err != nil {
		return nil, gqlid.InvalidIDError("shiftTemplateId", "Shift is invalid")
	}

	return &workerschedule.WorkerShiftAssignment{
		OrganizationID:   tenant.OrgID,
		BusinessUnitID:   tenant.BuID,
		WorkerID:         workerID,
		ShiftTemplateID:  templateID,
		EffectiveFrom:    int64(input.EffectiveFrom),
		CycleOffsetWeeks: intutils.SafeToInt16(gqlscalar.IntValue(input.CycleOffsetWeeks)),
		Notes:            gqlscalar.StringValue(input.Notes),
	}, nil
}

func availabilityPreferenceFromInput(
	input gqlmodel.SetAvailabilityPreferenceInput,
	tenant pagination.TenantInfo,
) (*workerschedule.WorkerAvailabilityPreference, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, gqlid.InvalidIDError("workerId", "Worker is invalid")
	}

	return &workerschedule.WorkerAvailabilityPreference{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		WorkerID:       workerID,
		DayOfWeek:      int16(input.DayOfWeek), //nolint:gosec // validated 0..6
		Preference:     input.Preference,
		Note:           gqlscalar.StringValue(input.Note),
	}, nil
}

func shiftSwapFromInput(
	input gqlmodel.ProposeShiftSwapInput,
	tenant pagination.TenantInfo,
) (*workerschedule.ShiftSwapRequest, error) {
	requestingID, err := pulid.MustParse(input.RequestingWorkerID)
	if err != nil {
		return nil, gqlid.InvalidIDError("requestingWorkerId", "Worker is invalid")
	}
	counterpartyID, err := gqlid.OptionalID(input.CounterpartyWorkerID)
	if err != nil {
		return nil, gqlid.InvalidIDError("counterpartyWorkerId", "Worker is invalid")
	}

	entity := &workerschedule.ShiftSwapRequest{
		OrganizationID:       tenant.OrgID,
		BusinessUnitID:       tenant.BuID,
		RequestingWorkerID:   requestingID,
		CounterpartyWorkerID: counterpartyID,
		ShiftDate:            int64(input.ShiftDate),
		Reason:               gqlscalar.StringValue(input.Reason),
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
func rotaToGQL(rota *workerschedule.Rota) *gqlmodel.Rota {
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
			FleetCode:     gqlscalar.EmptyToNil(row.FleetCode),
			FleetColor:    gqlscalar.EmptyToNil(row.FleetColor),
			ShiftCode:     gqlscalar.EmptyToNil(row.ShiftCode),
			ShiftName:     gqlscalar.EmptyToNil(row.ShiftName),
			ShiftColor:    gqlscalar.EmptyToNil(row.ShiftColor),
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

func rotaDayToGQL(day workerschedule.RotaDay) *gqlmodel.RotaDay {
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
func swapPermissionFor(status workerschedule.ShiftSwapStatus) permission.Operation {
	switch status {
	case workerschedule.SwapApproved:
		return permission.OpApprove
	case workerschedule.SwapRejected:
		return permission.OpReject
	case workerschedule.SwapWithdrawn:
		return permission.OpCancel
	case workerschedule.SwapAccepted, workerschedule.SwapDeclined, workerschedule.SwapProposed:
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
