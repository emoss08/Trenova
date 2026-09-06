package driverportalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/schedulingservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// PortalShiftDay is one day of the signed-in driver's own week. It carries
// what the driver needs to decide whether to ask for a swap and nothing about
// anybody else: the office board shows the roster, and a driver seeing who
// else is off is a personnel matter rather than a scheduling one.
type PortalShiftDay struct {
	Date            int64               `json:"date"`
	State           worker.RotaDayState `json:"state"`
	Scheduled       bool                `json:"scheduled"`
	StartMinute     int16               `json:"startMinute"`
	DurationMinutes int16               `json:"durationMinutes"`
	// Preference is nil when the driver has said nothing about that weekday,
	// which is not the same as having said they are available.
	Preference      *worker.AvailabilityPreference `json:"preference"`
	AssignmentCount int                            `json:"assignmentCount"`
}

// PortalSchedule is the driver's own rota.
type PortalSchedule struct {
	WeekStart     int64             `json:"weekStart"`
	WeekEnd       int64             `json:"weekEnd"`
	ShiftName     string            `json:"shiftName"`
	ShiftCode     string            `json:"shiftCode"`
	ShiftColor    string            `json:"shiftColor"`
	ScheduledDays int               `json:"scheduledDays"`
	Days          []*PortalShiftDay `json:"days"`
}

func (s *Service) requireScheduling() error {
	if s.scheduling == nil {
		return errortypes.NewValidationError(
			"feature",
			errortypes.ErrInvalidOperation,
			"Scheduling is not available",
		)
	}
	return nil
}

// MySchedule is the signed-in driver's own week, composed the same way the
// office board is so the two can never disagree about what they are working.
func (s *Service) MySchedule(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	at int64,
	weeks int,
) (*PortalSchedule, error) {
	if err := s.requireScheduling(); err != nil {
		return nil, err
	}
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	rota, err := s.scheduling.Rota(ctx, &schedulingservice.RotaRequest{
		TenantInfo: tenantInfo,
		At:         at,
		Weeks:      weeks,
		WorkerIDs:  []pulid.ID{wrk.ID},
	})
	if err != nil {
		return nil, err
	}

	out := &PortalSchedule{
		WeekStart: rota.WeekStart,
		WeekEnd:   rota.WeekEnd,
		Days:      make([]*PortalShiftDay, 0, len(rota.Rows)),
	}
	if len(rota.Rows) == 0 {
		return out, nil
	}

	row := rota.Rows[0]
	out.ShiftName = row.ShiftName
	out.ShiftCode = row.ShiftCode
	out.ShiftColor = row.ShiftColor
	out.ScheduledDays = row.ScheduledDays
	for _, day := range row.Days {
		shiftDay := &PortalShiftDay{
			Date:            day.Date,
			State:           day.State,
			Scheduled:       day.Scheduled,
			StartMinute:     day.StartMinute,
			DurationMinutes: day.DurationMinutes,
			AssignmentCount: day.AssignmentCount,
		}
		if day.Preference != "" {
			preference := day.Preference
			shiftDay.Preference = &preference
		}
		out.Days = append(out.Days, shiftDay)
	}

	return out, nil
}

func (s *Service) MyAvailability(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.WorkerAvailabilityPreference, error) {
	if err := s.requireScheduling(); err != nil {
		return nil, err
	}
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return s.scheduling.ListPreferences(ctx, &repositories.ListAvailabilityPreferencesRequest{
		TenantInfo: tenantInfo,
		WorkerID:   wrk.ID,
	})
}

// SetMyAvailabilityRequest is one weekday's statement.
type SetMyAvailabilityRequest struct {
	DayOfWeek  int16
	Preference worker.AvailabilityPreference
	Note       string
}

// SetMyAvailability records what the driver would rather work. The worker is
// resolved from the session rather than taken from the request: an id supplied
// by a driver is an id they can change.
func (s *Service) SetMyAvailability(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *SetMyAvailabilityRequest,
) (*worker.WorkerAvailabilityPreference, error) {
	if err := s.requireScheduling(); err != nil {
		return nil, err
	}
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return s.scheduling.SetPreference(ctx, &schedulingservice.SetPreferenceRequest{
		Entity: &worker.WorkerAvailabilityPreference{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			WorkerID:       wrk.ID,
			DayOfWeek:      req.DayOfWeek,
			Preference:     req.Preference,
			Note:           req.Note,
		},
		TenantInfo: tenantInfo,
	})
}

// PortalShiftSwap is a swap from the driver's own side of it. Outgoing is
// computed here rather than on the client: which side you are on decides
// whether the app offers "take it" or "withdraw", and a client that had to
// work that out would first have to be told its own worker id.
type PortalShiftSwap struct {
	ID                    string                 `json:"id"`
	Status                worker.ShiftSwapStatus `json:"status"`
	Outgoing              bool                   `json:"outgoing"`
	CounterpartyName      string                 `json:"counterpartyName"`
	ShiftDate             int64                  `json:"shiftDate"`
	CounterpartyShiftDate *int64                 `json:"counterpartyShiftDate"`
	Reason                string                 `json:"reason"`
	ResponseNote          string                 `json:"responseNote"`
	RespondedAt           *int64                 `json:"respondedAt"`
	DecidedAt             *int64                 `json:"decidedAt"`
}

// MyShiftSwaps is every swap the driver is on either side of.
func (s *Service) MyShiftSwaps(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*PortalShiftSwap, error) {
	if err := s.requireScheduling(); err != nil {
		return nil, err
	}
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	swaps, err := s.scheduling.ListSwaps(ctx, &repositories.ListShiftSwapsRequest{
		TenantInfo:     tenantInfo,
		WorkerID:       wrk.ID,
		IncludeWorkers: true,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*PortalShiftSwap, 0, len(swaps))
	for _, swap := range swaps {
		outgoing := swap.RequestingWorkerID == wrk.ID
		other := swap.CounterpartyWorker
		if !outgoing {
			other = swap.RequestingWorker
		}
		name := ""
		if other != nil {
			name = other.FirstName + " " + other.LastName
		}
		out = append(out, &PortalShiftSwap{
			ID:                    swap.ID.String(),
			Status:                swap.Status,
			Outgoing:              outgoing,
			CounterpartyName:      name,
			ShiftDate:             swap.ShiftDate,
			CounterpartyShiftDate: swap.CounterpartyShiftDate,
			Reason:                swap.Reason,
			ResponseNote:          swap.ResponseNote,
			RespondedAt:           swap.RespondedAt,
			DecidedAt:             swap.DecidedAt,
		})
	}

	return out, nil
}

// ProposeMyShiftSwapRequest is the driver asking a colleague to take a day.
type ProposeMyShiftSwapRequest struct {
	CounterpartyWorkerID  pulid.ID
	ShiftDate             int64
	CounterpartyShiftDate *int64
	Reason                string
}

func (s *Service) ProposeMyShiftSwap(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *ProposeMyShiftSwapRequest,
) (*worker.ShiftSwapRequest, error) {
	if err := s.requireScheduling(); err != nil {
		return nil, err
	}
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return s.scheduling.ProposeSwap(ctx, &schedulingservice.ProposeSwapRequest{
		Entity: &worker.ShiftSwapRequest{
			OrganizationID:        tenantInfo.OrgID,
			BusinessUnitID:        tenantInfo.BuID,
			RequestingWorkerID:    wrk.ID,
			CounterpartyWorkerID:  req.CounterpartyWorkerID,
			ShiftDate:             req.ShiftDate,
			CounterpartyShiftDate: req.CounterpartyShiftDate,
			Reason:                req.Reason,
		},
		TenantInfo: tenantInfo,
	})
}

// RespondToMyShiftSwap is the driver's own answer — accepting or declining one
// offered to them, or withdrawing one they made. The acting worker is carried
// through so the service can refuse an answer on somebody else's behalf, and
// the office half of the decision is not reachable from here at all.
func (s *Service) RespondToMyShiftSwap(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	swapID pulid.ID,
	status worker.ShiftSwapStatus,
	note string,
) (*worker.ShiftSwapRequest, error) {
	if err := s.requireScheduling(); err != nil {
		return nil, err
	}
	switch status {
	case worker.SwapAccepted, worker.SwapDeclined, worker.SwapWithdrawn:
	default:
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"A swap is decided by the office, not from the app",
		)
	}

	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return s.scheduling.TransitionSwap(ctx, &schedulingservice.TransitionSwapRequest{
		ID:            swapID,
		Status:        status,
		Note:          note,
		ActorWorkerID: wrk.ID,
		TenantInfo:    tenantInfo,
	})
}

// scheduleVisible is on once the driver is actually on a shift. An empty rota
// reads as a roster nobody has filled in rather than as a job with no fixed
// pattern, which is what an owner-operator has.
func (s *Service) scheduleVisible(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (bool, error) {
	if s.scheduling == nil {
		return false, nil
	}

	assignments, err := s.scheduling.ListAssignments(
		ctx,
		&repositories.ListShiftAssignmentsRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
			ActiveAt:   timeutils.NowUnix(),
		},
	)
	if err != nil {
		return false, err
	}

	return len(assignments) > 0, nil
}
