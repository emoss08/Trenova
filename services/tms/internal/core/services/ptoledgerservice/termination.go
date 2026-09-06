package ptoledgerservice

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// TerminationSettlement is what closing out a worker's balances did: which
// types were paid out, which were forfeited, and the entries that recorded it.
type TerminationSettlement struct {
	PaidOutDays   decimal.Decimal
	ForfeitedDays decimal.Decimal
	Entries       []*worker.WorkerPTOLedgerEntry
}

// PlannedSettlement is one balance to close, decided by the rule's
// termination action. Pure so the planner is testable without a database.
type PlannedSettlement struct {
	PTOType   worker.PTOType
	EntryType worker.PTOLedgerEntryType
	Days      decimal.Decimal
}

// PlanTerminationSettlement pairs positive balances with their rule's
// termination action. Untracked types (no rule) and zero or negative balances
// produce nothing; a negative balance is a receivable for payroll, not a
// ledger event.
func PlanTerminationSettlement(
	balances []*worker.WorkerPTOBalance,
	policy *worker.PTOPolicy,
) []PlannedSettlement {
	plan := make([]PlannedSettlement, 0, len(balances))
	for _, bal := range balances {
		if bal == nil || !bal.BalanceDays.IsPositive() {
			continue
		}
		var rule *worker.PTOPolicyRule
		if policy != nil {
			rule = policy.RuleFor(bal.PTOType)
		}
		if rule == nil {
			continue
		}
		entryType := worker.PTOLedgerEntryForfeiture
		if rule.TerminationAction() == worker.PTOTerminationPayOut {
			entryType = worker.PTOLedgerEntryPayout
		}
		plan = append(plan, PlannedSettlement{
			PTOType:   bal.PTOType,
			EntryType: entryType,
			Days:      bal.BalanceDays,
		})
	}
	sort.SliceStable(plan, func(i, j int) bool { return plan[i].PTOType < plan[j].PTOType })
	return plan
}

// SettleOnTermination zeroes every tracked balance as of the termination
// date, posting a Payout or Forfeiture per the policy rule. The period key
// `T:<effective>` makes a repeated call for the same termination a no-op.
func (s *Service) SettleOnTermination(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	effectiveAt int64,
	actor Actor,
) (*TerminationSettlement, error) {
	log := s.l.With(
		zap.String("operation", "SettleOnTermination"),
		zap.String("workerId", workerID.String()),
	)
	result := &TerminationSettlement{}

	resolved, err := s.ResolvePolicy(ctx, tenantInfo, workerID, effectiveAt)
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return result, nil
	}
	balances, err := s.ledgerRepo.ListBalances(ctx, &repositories.ListPTOBalancesRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return nil, err
	}

	periodKey := fmt.Sprintf("%s%d", periodKeyTermination, effectiveAt)
	for _, planned := range PlanTerminationSettlement(balances, resolved.Policy) {
		entry := &worker.WorkerPTOLedgerEntry{
			EntryType:    planned.EntryType,
			EffectiveAt:  effectiveAt,
			PeriodKey:    periodKey,
			AssignmentID: resolved.Assignment.ID,
			PTOPolicyID:  resolved.Policy.ID,
			Note:         "Employment ended",
		}
		posted, postErr := s.post(ctx, postParams{
			key: repositories.PTOBalanceKey{
				TenantInfo: tenantInfo,
				WorkerID:   workerID,
				PTOType:    planned.PTOType,
			},
			entry: entry,
			actor: actor,
			resolve: func(bal *worker.WorkerPTOBalance) (decimal.Decimal, bool) {
				if !bal.BalanceDays.IsPositive() {
					return decimal.Zero, false
				}
				return bal.BalanceDays.Neg(), true
			},
		})
		switch {
		case postErr == nil:
			result.Entries = append(result.Entries, posted)
			if planned.EntryType == worker.PTOLedgerEntryPayout {
				result.PaidOutDays = result.PaidOutDays.Add(posted.AmountDays.Neg())
			} else {
				result.ForfeitedDays = result.ForfeitedDays.Add(posted.AmountDays.Neg())
			}
		case errors.Is(postErr, repositories.ErrDuplicatePTOLedgerEntry):
			log.Info("termination settlement already posted", zap.String("ptoType", string(planned.PTOType)))
		default:
			return nil, postErr
		}
	}

	return result, nil
}

// LiabilityRow is one worker × type balance with its termination treatment.
type LiabilityRow struct {
	WorkerID      pulid.ID
	Worker        *worker.Worker
	PTOType       worker.PTOType
	BalanceDays   decimal.Decimal
	PendingDays   decimal.Decimal
	AccruedYTD    decimal.Decimal
	UsedYTD       decimal.Decimal
	OnTermination worker.PTOTerminationAction
	// LiabilityDays is the balance the organisation would owe if the worker
	// left today: the positive balance for PayOut rules, zero otherwise.
	LiabilityDays decimal.Decimal
}

type LiabilityReport struct {
	AsOf             int64
	WorkersTracked   int
	TotalBalanceDays decimal.Decimal
	LiabilityDays    decimal.Decimal
	ForfeitableDays  decimal.Decimal
	Rows             []*LiabilityRow
}

// AggregateLiability folds balances into the report, given each worker's
// resolved policy. Rows are sorted by liability, then balance, descending.
func AggregateLiability(
	asOf int64,
	balances []*worker.WorkerPTOBalance,
	policies map[pulid.ID]*worker.PTOPolicy,
) *LiabilityReport {
	report := &LiabilityReport{AsOf: asOf, Rows: make([]*LiabilityRow, 0, len(balances))}
	workers := make(map[pulid.ID]struct{}, len(balances))
	for _, bal := range balances {
		if bal == nil {
			continue
		}
		row := &LiabilityRow{
			WorkerID:      bal.WorkerID,
			Worker:        bal.Worker,
			PTOType:       bal.PTOType,
			BalanceDays:   bal.BalanceDays,
			AccruedYTD:    bal.AccruedYTDDays,
			UsedYTD:       bal.UsedYTDDays,
			OnTermination: worker.PTOTerminationForfeit,
		}
		if policy := policies[bal.WorkerID]; policy != nil {
			if rule := policy.RuleFor(bal.PTOType); rule != nil {
				row.OnTermination = rule.TerminationAction()
			}
		}
		if bal.BalanceDays.IsPositive() {
			if row.OnTermination == worker.PTOTerminationPayOut {
				row.LiabilityDays = bal.BalanceDays
				report.LiabilityDays = report.LiabilityDays.Add(bal.BalanceDays)
			} else {
				report.ForfeitableDays = report.ForfeitableDays.Add(bal.BalanceDays)
			}
		}
		report.TotalBalanceDays = report.TotalBalanceDays.Add(bal.BalanceDays)
		workers[bal.WorkerID] = struct{}{}
		report.Rows = append(report.Rows, row)
	}
	report.WorkersTracked = len(workers)
	sort.SliceStable(report.Rows, func(i, j int) bool {
		if !report.Rows[i].LiabilityDays.Equal(report.Rows[j].LiabilityDays) {
			return report.Rows[i].LiabilityDays.GreaterThan(report.Rows[j].LiabilityDays)
		}
		return report.Rows[i].BalanceDays.GreaterThan(report.Rows[j].BalanceDays)
	})
	return report
}

// LiabilityReport values every tracked balance in the organisation against
// its policy's termination treatment.
func (s *Service) LiabilityReport(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	asOf int64,
) (*LiabilityReport, error) {
	balances, err := s.ledgerRepo.ListOrgBalances(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	policies := make(map[pulid.ID]*worker.PTOPolicy, 16)
	seen := make(map[pulid.ID]bool, 16)
	for _, bal := range balances {
		if seen[bal.WorkerID] {
			continue
		}
		seen[bal.WorkerID] = true
		resolved, resolveErr := s.ResolvePolicy(ctx, tenantInfo, bal.WorkerID, asOf)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if resolved != nil {
			policies[bal.WorkerID] = resolved.Policy
		}
	}

	return AggregateLiability(asOf, balances, policies), nil
}
