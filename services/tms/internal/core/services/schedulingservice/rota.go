package schedulingservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"golang.org/x/sync/errgroup"
)

const (
	daysInWeek = 7
	// maxRotaWeeks bounds how far ahead the board can be drawn in one read. A
	// year of weeks is a report, not a rota, and would be a cheap way to make
	// the four queries expensive.
	maxRotaWeeks = 8
)

// RotaRequest is the week being drawn.
type RotaRequest struct {
	TenantInfo pagination.TenantInfo
	// At is any instant in the week wanted; it is resolved back to the Sunday
	// that starts it, so the caller can pass "today" without doing calendar
	// arithmetic on the client.
	At          int64
	Weeks       int
	FleetCodeID pulid.ID
	// ManagerIDs narrows the board to the people a manager answers for, which
	// is what the team-scoped view passes.
	ManagerIDs []pulid.ID
	WorkerIDs  []pulid.ID
	Limit      int
}

// Rota draws the board. It is four grouped queries and a pure composition:
// the pattern says what was planned, and time off, leave, dispatch and the
// driver's own preference each say what happened to it.
func (s *Service) Rota(ctx context.Context, req *RotaRequest) (*worker.Rota, error) {
	at := req.At
	if at <= 0 {
		at = timeutils.NowUnix()
	}
	weeks := req.Weeks
	if weeks <= 0 {
		weeks = 1
	}
	if weeks > maxRotaWeeks {
		weeks = maxRotaWeeks
	}

	weekStart := worker.StartOfWeekUTC(at, time.UTC)
	weekEnd := time.Unix(weekStart, 0).UTC().AddDate(0, 0, daysInWeek*weeks).Unix()

	query := &repositories.RotaQuery{
		TenantInfo:  req.TenantInfo,
		WeekStart:   weekStart,
		WeekEnd:     weekEnd,
		FleetCodeID: req.FleetCodeID,
		ManagerIDs:  req.ManagerIDs,
		Limit:       req.Limit,
	}

	var (
		workers   []repositories.RotaWorkerRow
		timeOff   []repositories.RotaRangeRow
		leave     []repositories.RotaRangeRow
		assigned  []repositories.RotaDayRow
		templates map[pulid.ID]*worker.ShiftTemplate
	)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		rows, err := s.repo.RotaWorkers(groupCtx, query)
		if err != nil {
			return err
		}
		workers = rows
		return nil
	})
	group.Go(func() error {
		rows, err := s.repo.RotaTimeOffRanges(groupCtx, query)
		if err != nil {
			return err
		}
		timeOff = rows
		return nil
	})
	group.Go(func() error {
		rows, err := s.repo.RotaLeaveRanges(groupCtx, query)
		if err != nil {
			return err
		}
		leave = rows
		return nil
	})
	group.Go(func() error {
		rows, err := s.repo.RotaAssignedDays(groupCtx, query)
		if err != nil {
			return err
		}
		assigned = rows
		return nil
	})
	// The patterns are read whole rather than joined per worker: a carrier has
	// a handful of them and every roster line points at one of the same few.
	group.Go(func() error {
		rows, err := s.repo.ListTemplates(groupCtx, &repositories.ListShiftTemplatesRequest{
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return err
		}
		templates = make(map[pulid.ID]*worker.ShiftTemplate, len(rows))
		for _, template := range rows {
			templates[template.ID] = template
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("gather rota: %w", err)
	}

	if len(req.WorkerIDs) > 0 {
		workers = filterWorkers(workers, req.WorkerIDs)
	}

	preferences, err := s.gatherPreferences(ctx, req.TenantInfo, workers)
	if err != nil {
		return nil, err
	}

	timeOffDays := expandRanges(timeOff, weekStart, weekEnd)
	leaveDays := expandRanges(leave, weekStart, weekEnd)
	assignedDays := indexDays(assigned)

	inputs := make([]worker.RotaWorkerInput, 0, len(workers))
	for _, row := range workers {
		inputs = append(inputs, worker.RotaWorkerInput{
			WorkerID:         row.WorkerID,
			Name:             fmt.Sprintf("%s %s", row.FirstName, row.LastName),
			FleetCode:        row.FleetCode,
			FleetColor:       row.FleetColor,
			Template:         templates[row.ShiftTemplateID],
			CycleOffsetWeeks: row.CycleOffsetWeeks,
			Preferences:      preferences[row.WorkerID],
			TimeOffDays:      timeOffDays[row.WorkerID],
			LeaveDays:        leaveDays[row.WorkerID],
			AssignedDays:     assignedDays[row.WorkerID],
		})
	}

	rota := worker.BuildRota(worker.RotaInput{
		WeekStart: weekStart,
		Weeks:     weeks,
		Location:  time.UTC,
		Workers:   inputs,
	})

	return rota, nil
}

// gatherPreferences reads the whole roster's stated availability in one query.
// A preference is seven rows a worker at most, so the board's worth is a small
// read rather than a per-worker one.
func (s *Service) gatherPreferences(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workers []repositories.RotaWorkerRow,
) (map[pulid.ID]map[int16]worker.AvailabilityPreference, error) {
	out := make(map[pulid.ID]map[int16]worker.AvailabilityPreference, len(workers))
	if len(workers) == 0 {
		return out, nil
	}

	rows, err := s.repo.ListPreferences(ctx, &repositories.ListAvailabilityPreferencesRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		byDay, ok := out[row.WorkerID]
		if !ok {
			byDay = make(map[int16]worker.AvailabilityPreference, daysInWeek)
			out[row.WorkerID] = byDay
		}
		byDay[row.DayOfWeek] = row.Preference
	}

	return out, nil
}

func filterWorkers(
	rows []repositories.RotaWorkerRow,
	ids []pulid.ID,
) []repositories.RotaWorkerRow {
	wanted := make(map[pulid.ID]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}

	out := make([]repositories.RotaWorkerRow, 0, len(ids))
	for _, row := range rows {
		if _, ok := wanted[row.WorkerID]; ok {
			out = append(out, row)
		}
	}

	return out
}

// expandRanges turns stored spans into the days of the window they cover. A
// zero end is an open span — a leave case with no return date — and runs to the
// end of the window rather than stopping on its start day.
func expandRanges(
	rows []repositories.RotaRangeRow,
	windowStart, windowEnd int64,
) map[pulid.ID]map[int64]bool {
	out := make(map[pulid.ID]map[int64]bool, len(rows))

	for _, row := range rows {
		start := startOfDayUTC(row.StartsAt)
		if start < windowStart {
			start = windowStart
		}
		end := row.EndsAt
		if end <= 0 || end >= windowEnd {
			end = windowEnd - 1
		}
		end = startOfDayUTC(end)
		if end < start {
			continue
		}

		days, ok := out[row.WorkerID]
		if !ok {
			days = make(map[int64]bool, daysInWeek)
			out[row.WorkerID] = days
		}
		for day := start; day <= end; day += secondsPerDay {
			days[day] = true
		}
	}

	return out
}

func indexDays(rows []repositories.RotaDayRow) map[pulid.ID]map[int64]int {
	out := make(map[pulid.ID]map[int64]int, len(rows))

	for _, row := range rows {
		days, ok := out[row.WorkerID]
		if !ok {
			days = make(map[int64]int, daysInWeek)
			out[row.WorkerID] = days
		}
		days[row.DayStart] += row.Count
	}

	return out
}
