package shipmentwatchlistservice

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const detentionTopLimit = 3

var secondsPerHour = decimal.NewFromInt(3600)

func AccruedAmount(rate decimal.Decimal, billableSince, now int64) decimal.Decimal {
	if !rate.IsPositive() || now <= billableSince {
		return decimal.Zero
	}
	return rate.Mul(decimal.NewFromInt(now - billableSince)).Div(secondsPerHour).Round(2)
}

func accessorialTierUnit(unit accessorialcharge.RateUnit) detention.TierRateUnit {
	switch unit {
	case accessorialcharge.RateUnitHour:
		return detention.TierRateUnitHour
	case accessorialcharge.RateUnitDay:
		return detention.TierRateUnitDay
	case accessorialcharge.RateUnitMile, accessorialcharge.RateUnitStop:
		return detention.TierRateUnitFlat
	default:
		return detention.TierRateUnitFlat
	}
}

func (s *Service) detentionWatch(
	ctx context.Context,
	req *repositories.ShipmentWatchlistRequest,
	control *tenant.ShipmentControl,
) (services.ShipmentDetentionWatch, error) {
	now := req.Basis.Now.Unix()

	var (
		accruals []services.DetentionAccrual
		moveIDs  []pulid.ID
		err      error
	)
	if req.Basis.Detention.UsePolicyEngine {
		accruals, moveIDs, err = s.engineAccruals(ctx, req.TenantInfo, now)
	} else {
		accruals, moveIDs, err = s.dwellAccruals(ctx, req, control, now)
	}
	if err != nil {
		return services.ShipmentDetentionWatch{}, err
	}

	if err = s.nameCoverage(ctx, req.TenantInfo, accruals, moveIDs); err != nil {
		return services.ShipmentDetentionWatch{}, err
	}

	return SummarizeDetention(accruals, now), nil
}

func SummarizeDetention(
	accruals []services.DetentionAccrual,
	now int64,
) services.ShipmentDetentionWatch {
	watch := services.ShipmentDetentionWatch{
		StopCount:   len(accruals),
		Amount:      decimal.Zero,
		RatePerHour: decimal.Zero,
		SnapshotAt:  now,
	}
	for i := range accruals {
		watch.Amount = watch.Amount.Add(accruals[i].Amount)
		watch.RatePerHour = watch.RatePerHour.Add(accruals[i].RatePerHour)
	}

	sorted := slices.Clone(accruals)
	slices.SortStableFunc(sorted, func(a, b services.DetentionAccrual) int {
		if c := b.Amount.Cmp(a.Amount); c != 0 {
			return c
		}
		return cmp.Compare(a.BillableSince, b.BillableSince)
	})
	watch.Top = sorted[:min(len(sorted), detentionTopLimit)]

	return watch
}

func (s *Service) engineAccruals(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	now int64,
) ([]services.DetentionAccrual, []pulid.ID, error) {
	entries, err := s.detention.ListDesk(ctx, tenantInfo)
	if err != nil {
		return nil, nil, err
	}

	accruals := make([]services.DetentionAccrual, 0, len(entries))
	moveIDs := make([]pulid.ID, 0, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.Occurrence == nil {
			continue
		}
		occurrence := entry.Occurrence
		if occurrence.FreeTimeExpiresAt > now {
			continue
		}

		rate := occurrence.PolicySnapshot.HourlyRateAt(occurrence.BillableMinutes)
		amount := AccruedAmount(rate, occurrence.FreeTimeExpiresAt, now)
		if !rate.IsPositive() {
			amount = occurrence.BillableAmount
		}
		occurrenceID := occurrence.ID
		accruals = append(accruals, services.DetentionAccrual{
			ShipmentID:    occurrence.ShipmentID,
			StopID:        occurrence.StopID,
			OccurrenceID:  &occurrenceID,
			FacilityName:  occurrence.LocationName,
			BillableSince: occurrence.FreeTimeExpiresAt,
			RatePerHour:   rate,
			Amount:        amount,
		})
		moveIDs = append(moveIDs, occurrence.ShipmentMoveID)
	}

	return accruals, moveIDs, nil
}

func (s *Service) dwellAccruals(
	ctx context.Context,
	req *repositories.ShipmentWatchlistRequest,
	control *tenant.ShipmentControl,
	now int64,
) ([]services.DetentionAccrual, []pulid.ID, error) {
	rows, err := s.watchlist.ListDwellingStops(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return []services.DetentionAccrual{}, nil, nil
	}

	rate, err := s.defaultDetentionRate(ctx, req.TenantInfo, control)
	if err != nil {
		return nil, nil, err
	}

	threshold := req.Basis.Detention.ThresholdMinutes * secondsPerMinute
	accruals := make([]services.DetentionAccrual, 0, len(rows))
	moveIDs := make([]pulid.ID, 0, len(rows))
	for _, row := range rows {
		since := row.ActualArrival + threshold
		accruals = append(accruals, services.DetentionAccrual{
			ShipmentID:    row.ShipmentID,
			StopID:        row.StopID,
			FacilityName:  row.FacilityName,
			BillableSince: since,
			RatePerHour:   rate,
			Amount:        AccruedAmount(rate, since, now),
		})
		moveIDs = append(moveIDs, row.MoveID)
	}

	return accruals, moveIDs, nil
}

func (s *Service) defaultDetentionRate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *tenant.ShipmentControl,
) (decimal.Decimal, error) {
	if control.DetentionChargeID == nil || control.DetentionChargeID.IsNil() {
		return decimal.Zero, nil
	}

	charge, err := s.accessorials.GetByID(ctx, repositories.GetAccessorialChargeByIDRequest{
		ID:         *control.DetentionChargeID,
		TenantInfo: &tenantInfo,
	})
	if err != nil {
		return decimal.Zero, err
	}

	return detention.HourlyRate(charge.Amount, accessorialTierUnit(charge.RateUnit)), nil
}

func (s *Service) nameCoverage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	accruals []services.DetentionAccrual,
	moveIDs []pulid.ID,
) error {
	if len(moveIDs) == 0 || s.moves == nil {
		return nil
	}

	moves, err := s.moves.ListBoardMoves(ctx, &repositories.DispatchBoardFilter{
		TenantInfo:     tenantInfo,
		MoveIDs:        moveIDs,
		IncludeCovered: true,
		Limit:          len(moveIDs),
	})
	if err != nil {
		return err
	}

	names := make(map[pulid.ID]string, len(moves))
	for _, move := range moves {
		if move == nil {
			continue
		}
		if move.AssignedWorkerName != "" {
			names[move.MoveID] = move.AssignedWorkerName
		} else if move.AssignedCarrierName != "" {
			names[move.MoveID] = move.AssignedCarrierName
		}
	}
	for i := range accruals {
		accruals[i].CoverageName = names[moveIDs[i]]
	}

	return nil
}
