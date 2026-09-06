package ptoledgerservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type AvailabilityRequest struct {
	TenantInfo   pagination.TenantInfo
	WorkerID     pulid.ID
	PTOType      worker.PTOType
	Days         decimal.Decimal
	StartDate    int64
	ExcludePTOID pulid.ID
}

type AvailabilityResult struct {
	Tracked                bool
	Enforced               bool
	Allowed                bool
	Days                   decimal.Decimal
	BalanceDays            decimal.Decimal
	PendingDays            decimal.Decimal
	ProjectedAccrualDays   decimal.Decimal
	ProjectedAvailableDays decimal.Decimal
	FloorDays              decimal.Decimal
	Message                string
	Policy                 *ResolvedPolicy
}

func (r *AvailabilityResult) ValidationError() error {
	if r.Allowed {
		return nil
	}
	return errortypes.NewValidationError("endDate", errortypes.ErrInvalid, r.Message)
}

type BalanceView struct {
	PTOType        worker.PTOType
	Tracked        bool
	Enforced       bool
	BalanceDays    decimal.Decimal
	PendingDays    decimal.Decimal
	AvailableDays  decimal.Decimal
	AccruedYTDDays decimal.Decimal
	UsedYTDDays    decimal.Decimal
	CarriedDays    decimal.Decimal
	MaxBalanceDays decimal.NullDecimal
	NextAccrual    *PlannedEntry
}

func (s *Service) workerCalcContext(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (hire int64, termination *int64, loc *time.Location, err error) {
	wrk, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:             workerID,
		TenantInfo:     tenantInfo,
		IncludeProfile: true,
	})
	if err != nil {
		return 0, nil, nil, err
	}
	if wrk.Profile != nil {
		hire = wrk.Profile.HireDate
		termination = wrk.Profile.TerminationDate
	}
	loc, err = s.OrgLocation(ctx, tenantInfo)
	if err != nil {
		return 0, nil, nil, err
	}
	return hire, termination, loc, nil
}

func (s *Service) CheckAvailability(
	ctx context.Context,
	req *AvailabilityRequest,
) (*AvailabilityResult, error) {
	result := &AvailabilityResult{Allowed: true, Days: req.Days}

	resolved, err := s.ResolvePolicy(ctx, req.TenantInfo, req.WorkerID, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	rule := resolved.RuleFor(req.PTOType)
	if rule == nil {
		return result, nil
	}
	result.Tracked = true
	result.Policy = resolved
	result.Enforced = resolved.Policy.EnforceBalance

	key := &repositories.PTOBalanceKey{
		TenantInfo: req.TenantInfo,
		WorkerID:   req.WorkerID,
		PTOType:    req.PTOType,
	}
	bal, err := s.ledgerRepo.GetBalance(ctx, key)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	if bal != nil {
		result.BalanceDays = bal.BalanceDays
	}

	pending, err := s.ledgerRepo.PendingDays(ctx, &repositories.PendingPTODaysRequest{
		TenantInfo: req.TenantInfo,
		WorkerID:   req.WorkerID,
		PTOType:    req.PTOType,
		ExcludeID:  req.ExcludePTOID,
	})
	if err != nil {
		return nil, err
	}
	result.PendingDays = pending

	if req.StartDate > time.Now().Unix() {
		hire, termination, loc, ctxErr := s.workerCalcContext(ctx, req.TenantInfo, req.WorkerID)
		if ctxErr != nil {
			return nil, ctxErr
		}
		var lastAccrual, lastRollover string
		var carried decimal.Decimal
		if bal != nil {
			lastAccrual = bal.LastAccrualPeriodKey
			lastRollover = bal.LastRolloverKey
			carried = bal.CarriedDays
		}
		plan := Schedule(CalcInput{
			Rule:            *rule,
			YearBasis:       resolved.Policy.YearBasis,
			WaitingDays:     resolved.Policy.WaitingPeriodDays,
			HireDate:        hire,
			TerminationDate: termination,
			Loc:             loc,
			AsOf:            req.StartDate,
			LastAccrualKey:  lastAccrual,
			LastRolloverKey: lastRollover,
		})
		result.ProjectedAccrualDays = ProjectedAccrual(plan, result.BalanceDays, carried, *rule)
	}

	result.ProjectedAvailableDays = result.BalanceDays.Sub(pending).Add(result.ProjectedAccrualDays)
	if resolved.Policy.AllowNegative {
		result.FloorDays = resolved.Policy.NegativeFloorDays
	}

	if !result.Enforced {
		return result, nil
	}

	remaining := result.ProjectedAvailableDays.Sub(req.Days)
	if remaining.LessThan(result.FloorDays) {
		result.Allowed = false
		result.Message = fmt.Sprintf(
			"Requesting %s %s day%s but only %s will be available on %s (balance %s, pending %s, accruing %s)",
			req.Days.StringFixed(2),
			req.PTOType,
			plural(req.Days),
			result.ProjectedAvailableDays.StringFixed(2),
			time.Unix(req.StartDate, 0).UTC().Format("Jan 2, 2006"),
			result.BalanceDays.StringFixed(2),
			pending.StringFixed(2),
			result.ProjectedAccrualDays.StringFixed(2),
		)
	}

	return result, nil
}

func plural(d decimal.Decimal) string {
	if d.Equal(decimal.NewFromInt(1)) {
		return ""
	}
	return "s"
}

func (s *Service) GetBalances(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) ([]*BalanceView, error) {
	resolved, err := s.ResolvePolicy(ctx, tenantInfo, workerID, time.Now().Unix())
	if err != nil {
		return nil, err
	}

	balances, err := s.ledgerRepo.ListBalances(ctx, &repositories.ListPTOBalancesRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return nil, err
	}
	byType := make(map[worker.PTOType]*worker.WorkerPTOBalance, len(balances))
	for _, bal := range balances {
		byType[bal.PTOType] = bal
	}

	types := make([]worker.PTOType, 0, 8)
	seen := make(map[worker.PTOType]struct{}, 8)
	if resolved != nil {
		for _, rule := range resolved.Policy.Rules {
			if rule == nil {
				continue
			}
			types = append(types, rule.PTOType)
			seen[rule.PTOType] = struct{}{}
		}
	}
	for _, bal := range balances {
		if _, ok := seen[bal.PTOType]; !ok {
			types = append(types, bal.PTOType)
			seen[bal.PTOType] = struct{}{}
		}
	}

	var (
		hire        int64
		termination *int64
		loc         *time.Location
		nowUnix     = time.Now().Unix()
	)
	if resolved != nil {
		hire, termination, loc, err = s.workerCalcContext(ctx, tenantInfo, workerID)
		if err != nil {
			return nil, err
		}
	}

	views := make([]*BalanceView, 0, len(types))
	for _, ptoType := range types {
		view := &BalanceView{PTOType: ptoType}
		rule := resolved.RuleFor(ptoType)
		view.Tracked = rule != nil
		view.Enforced = rule != nil && resolved.Policy.EnforceBalance
		if rule != nil {
			view.MaxBalanceDays = rule.MaxBalanceDays
		}

		var lastAccrual, lastRollover string
		if bal, ok := byType[ptoType]; ok {
			view.BalanceDays = bal.BalanceDays
			view.AccruedYTDDays = bal.AccruedYTDDays
			view.UsedYTDDays = bal.UsedYTDDays
			view.CarriedDays = bal.CarriedDays
			lastAccrual = bal.LastAccrualPeriodKey
			lastRollover = bal.LastRolloverKey
		}

		pending, pErr := s.ledgerRepo.PendingDays(ctx, &repositories.PendingPTODaysRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
			PTOType:    ptoType,
		})
		if pErr != nil {
			return nil, pErr
		}
		view.PendingDays = pending
		view.AvailableDays = view.BalanceDays.Sub(pending)

		if rule != nil && rule.Accrues() {
			plan := Schedule(CalcInput{
				Rule:            *rule,
				YearBasis:       resolved.Policy.YearBasis,
				WaitingDays:     resolved.Policy.WaitingPeriodDays,
				HireDate:        hire,
				TerminationDate: termination,
				Loc:             loc,
				AsOf:            time.Unix(nowUnix, 0).AddDate(1, 0, 0).Unix(),
				LastAccrualKey:  lastAccrual,
				LastRolloverKey: lastRollover,
			})
			for i := range plan {
				if plan[i].EntryType == worker.PTOLedgerEntryAccrual &&
					plan[i].EffectiveAt > nowUnix {
					next := plan[i]
					view.NextAccrual = &next
					break
				}
			}
		}

		views = append(views, view)
	}

	return views, nil
}
