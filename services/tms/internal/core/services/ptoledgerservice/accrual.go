package ptoledgerservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type AccrualRunResult struct {
	WorkersProcessed int `json:"workersProcessed"`
	WorkersSkipped   int `json:"workersSkipped"`
	EntriesPosted    int `json:"entriesPosted"`
	EntriesCapped    int `json:"entriesCapped"`
	EntriesSkipped   int `json:"entriesSkipped"`
}

func (r *AccrualRunResult) Add(other *AccrualRunResult) {
	if other == nil {
		return
	}
	r.WorkersProcessed += other.WorkersProcessed
	r.WorkersSkipped += other.WorkersSkipped
	r.EntriesPosted += other.EntriesPosted
	r.EntriesCapped += other.EntriesCapped
	r.EntriesSkipped += other.EntriesSkipped
}

func (s *Service) PreviewAccrual(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	asOf int64,
) ([]PlannedEntry, error) {
	resolved, err := s.ResolvePolicy(ctx, tenantInfo, workerID, asOf)
	if err != nil || resolved == nil {
		return nil, err
	}
	hire, termination, loc, err := s.workerCalcContext(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	payPeriod, err := s.payPeriodFor(ctx, tenantInfo, resolved.Policy)
	if err != nil {
		return nil, err
	}

	plan := make([]PlannedEntry, 0, 8)
	for _, rule := range resolved.Policy.Rules {
		if rule == nil {
			continue
		}
		bal, bErr := s.ledgerRepo.GetBalance(ctx, &repositories.PTOBalanceKey{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
			PTOType:    rule.PTOType,
		})
		if bErr != nil && !errortypes.IsNotFoundError(bErr) {
			return nil, bErr
		}
		var lastAccrual, lastRollover string
		if bal != nil {
			lastAccrual = bal.LastAccrualPeriodKey
			lastRollover = bal.LastRolloverKey
		}
		for _, entry := range Schedule(CalcInput{
			Rule:            *rule,
			YearBasis:       resolved.Policy.YearBasis,
			WaitingDays:     resolved.Policy.WaitingPeriodDays,
			HireDate:        hire,
			TerminationDate: termination,
			Loc:             loc,
			AsOf:            asOf,
			LastAccrualKey:  lastAccrual,
			LastRolloverKey: lastRollover,
			PayPeriod:       payPeriod,
		}) {
			entry.PeriodKey = string(rule.PTOType) + "/" + entry.PeriodKey
			plan = append(plan, entry)
		}
	}
	return plan, nil
}

func (s *Service) RunAccrualForWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	asOf int64,
	actor Actor,
) (*AccrualRunResult, error) {
	log := s.l.With(
		zap.String("operation", "RunAccrualForWorker"),
		zap.String("workerId", workerID.String()),
	)
	result := &AccrualRunResult{}

	resolved, err := s.ResolvePolicy(ctx, tenantInfo, workerID, asOf)
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		result.WorkersSkipped = 1
		return result, nil
	}
	hire, termination, loc, err := s.workerCalcContext(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	if hire == 0 {
		log.Warn("worker has no hire date; skipping accrual")
		result.WorkersSkipped = 1
		return result, nil
	}
	payPeriod, err := s.payPeriodFor(ctx, tenantInfo, resolved.Policy)
	if err != nil {
		return nil, err
	}

	for _, rule := range resolved.Policy.Rules {
		if rule == nil {
			continue
		}
		key := repositories.PTOBalanceKey{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
			PTOType:    rule.PTOType,
		}
		if err = s.ledgerRepo.EnsureBalance(ctx, &key); err != nil {
			return nil, err
		}
		bal, bErr := s.ledgerRepo.GetBalance(ctx, &key)
		if bErr != nil {
			return nil, bErr
		}

		plan := Schedule(CalcInput{
			Rule:            *rule,
			YearBasis:       resolved.Policy.YearBasis,
			WaitingDays:     resolved.Policy.WaitingPeriodDays,
			HireDate:        hire,
			TerminationDate: termination,
			Loc:             loc,
			AsOf:            asOf,
			LastAccrualKey:  bal.LastAccrualPeriodKey,
			LastRolloverKey: bal.LastRolloverKey,
			PayPeriod:       payPeriod,
		})

		for _, planned := range plan {
			if postErr := s.postPlanned(ctx, key, resolved, rule, planned, actor, result); postErr != nil {
				log.Error("failed to post planned PTO entry",
					zap.String("periodKey", planned.PeriodKey), zap.Error(postErr))
				return result, postErr
			}
		}
	}

	result.WorkersProcessed = 1
	if actor.Type == worker.PTOLedgerActorUser && !actor.UserID.IsNil() {
		if err = s.auditService.LogAction(&services.LogActionParams{
			Resource:       permission.ResourceWorkerPTO,
			ResourceID:     workerID.String(),
			Operation:      permission.OpManage,
			UserID:         actor.UserID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
		}, auditservice.WithComment(fmt.Sprintf(
			"Accrual run: %d posted, %d capped, %d skipped",
			result.EntriesPosted, result.EntriesCapped, result.EntriesSkipped,
		))); err != nil {
			log.Error("failed to log audit action", zap.Error(err))
		}
	}

	return result, nil
}

func (s *Service) postPlanned(
	ctx context.Context,
	key repositories.PTOBalanceKey,
	resolved *ResolvedPolicy,
	rule *worker.PTOPolicyRule,
	planned PlannedEntry,
	actor Actor,
	result *AccrualRunResult,
) error {
	entry := &worker.WorkerPTOLedgerEntry{
		EntryType:    planned.EntryType,
		AmountDays:   planned.NominalDays,
		EffectiveAt:  planned.EffectiveAt,
		PeriodKey:    planned.PeriodKey,
		AssignmentID: resolved.Assignment.ID,
		PTOPolicyID:  resolved.Policy.ID,
	}

	capRule := *rule
	if planned.MaxBalanceDays.Valid {
		capRule.MaxBalanceDays = planned.MaxBalanceDays
	}
	params := postParams{
		key:   key,
		entry: entry,
		actor: actor,
		rule:  &capRule,
		onApplied: func(bal *worker.WorkerPTOBalance) {
			advanceCursor(bal, planned, rule)
		},
	}

	if planned.Deferred {
		params.resolve = func(bal *worker.WorkerPTOBalance) (decimal.Decimal, bool) {
			return resolveExpiry(bal, planned, rule)
		}
	}

	_, err := s.post(ctx, params)
	switch {
	case err == nil:
		result.EntriesPosted++
		return nil
	case errors.Is(err, ErrCapped):
		result.EntriesCapped++
		return nil
	case errors.Is(err, repositories.ErrDuplicatePTOLedgerEntry):
		result.EntriesSkipped++
		return s.advanceCursorOnly(ctx, key, planned, rule)
	default:
		return err
	}
}

// payPeriodFor loads the settlement calendar only when a rule needs it.
func (s *Service) payPeriodFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	policy *worker.PTOPolicy,
) (*PayPeriodSpec, error) {
	for _, rule := range policy.Rules {
		if rule != nil && rule.AccrualMethod == worker.PTOAccrualMethodPerPayPeriod {
			return s.PayPeriodSpec(ctx, tenantInfo)
		}
	}
	return nil, nil //nolint:nilnil // no rule accrues per pay period
}

func resolveExpiry(
	bal *worker.WorkerPTOBalance,
	planned PlannedEntry,
	rule *worker.PTOPolicyRule,
) (decimal.Decimal, bool) {
	switch {
	case strings.HasPrefix(planned.PeriodKey, periodKeyCarryover):
		if !rule.CarryoverCapDays.Valid {
			return decimal.Zero, false
		}
		excess := bal.BalanceDays.Sub(rule.CarryoverCapDays.Decimal)
		if !excess.IsPositive() {
			return decimal.Zero, false
		}
		return excess.Neg(), true
	case strings.HasPrefix(planned.PeriodKey, periodKeyExpiry):
		expire := decimal.Min(bal.CarriedDays, bal.BalanceDays)
		if !expire.IsPositive() {
			return decimal.Zero, false
		}
		return expire.Neg(), true
	default:
		return planned.NominalDays, true
	}
}

func advanceCursor(bal *worker.WorkerPTOBalance, planned PlannedEntry, rule *worker.PTOPolicyRule) {
	switch {
	case strings.HasPrefix(planned.PeriodKey, periodKeyCarryover):
		bal.LastRolloverKey = planned.PeriodKey
		bal.YearStartedAt = &planned.EffectiveAt
		carried := bal.BalanceDays
		if rule.CarryoverCapDays.Valid && carried.GreaterThan(rule.CarryoverCapDays.Decimal) {
			carried = rule.CarryoverCapDays.Decimal
		}
		if carried.IsNegative() {
			carried = decimal.Zero
		}
		bal.CarriedDays = carried
		bal.AccruedYTDDays = decimal.Zero
		bal.UsedYTDDays = decimal.Zero
	case strings.HasPrefix(planned.PeriodKey, periodKeyExpiry):
		if planned.PeriodKey > bal.LastRolloverKey {
			bal.LastRolloverKey = planned.PeriodKey
		}
		bal.CarriedDays = decimal.Zero
	default:
		if planned.PeriodKey > bal.LastAccrualPeriodKey {
			bal.LastAccrualPeriodKey = planned.PeriodKey
		}
	}
}

func (s *Service) advanceCursorOnly(
	ctx context.Context,
	key repositories.PTOBalanceKey,
	planned PlannedEntry,
	rule *worker.PTOPolicyRule,
) error {
	_, err := s.post(ctx, postParams{
		key:   key,
		entry: &worker.WorkerPTOLedgerEntry{EntryType: planned.EntryType},
		actor: SystemActor(),
		resolve: func(_ *worker.WorkerPTOBalance) (decimal.Decimal, bool) {
			return decimal.Zero, false
		},
		onApplied: func(bal *worker.WorkerPTOBalance) {
			advanceCursor(bal, planned, rule)
		},
	})
	if errors.Is(err, ErrCapped) {
		return nil
	}
	return err
}

type TenantAccrualPage struct {
	Result    *AccrualRunResult
	NextAfter pulid.ID
}

func (s *Service) RunAccrualForTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	asOf int64,
	afterID pulid.ID,
	limit int,
	actor Actor,
) (*TenantAccrualPage, error) {
	assignments, err := s.policyRepo.ListOpenAssignments(
		ctx,
		&repositories.ListOpenPTOAssignmentsRequest{
			TenantInfo: tenantInfo,
			AfterID:    afterID,
			Limit:      limit,
		},
	)
	if err != nil {
		return nil, err
	}

	page := &TenantAccrualPage{Result: &AccrualRunResult{}}
	for _, assignment := range assignments {
		page.NextAfter = assignment.ID
		res, runErr := s.RunAccrualForWorker(ctx, tenantInfo, assignment.WorkerID, asOf, actor)
		if runErr != nil {
			s.l.Error("accrual failed for worker",
				zap.String("workerId", assignment.WorkerID.String()), zap.Error(runErr))
			page.Result.WorkersSkipped++
			continue
		}
		page.Result.Add(res)
	}
	if len(assignments) < limit || limit <= 0 {
		page.NextAfter = pulid.Nil
	}

	return page, nil
}

func (s *Service) ListAccrualTenants(ctx context.Context) ([]pagination.TenantInfo, error) {
	return s.policyRepo.ListTenantsWithOpenAssignments(ctx)
}
