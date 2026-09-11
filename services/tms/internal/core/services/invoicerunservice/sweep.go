package invoicerunservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// periodKey groups the customers that share a tenant, cadence and period, so one
// run covers them all rather than one run per customer.
type periodKey struct {
	orgID     pulid.ID
	buID      pulid.ID
	cycle     customer.BillingCycle
	periodEnd int64
}

// SweepDueSchedules bills every period that has closed and not yet been billed.
//
// Driven off each customer's own schedule rather than the cron's cadence: one
// hourly tick asks every statement customer which of their periods have closed,
// so an outage that skips ticks still bills the periods it missed instead of
// losing them.
func (s *Service) SweepDueSchedules(
	ctx context.Context,
	actor *servicesports.RequestActor,
) (*servicesports.InvoiceRunSweepResult, error) {
	schedules, err := s.customerRepo.ListDueBillingSchedules(ctx)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	batches := make(map[periodKey][]pulid.ID)
	order := make([]periodKey, 0)
	starts := make(map[periodKey]int64)

	for _, schedule := range schedules {
		for _, period := range PeriodsDue(schedule.Profile(), now) {
			key := periodKey{
				orgID:     schedule.OrganizationID,
				buID:      schedule.BusinessUnitID,
				cycle:     schedule.BillingCycle,
				periodEnd: period.End,
			}
			if _, seen := batches[key]; !seen {
				order = append(order, key)
				starts[key] = period.Start
			}
			batches[key] = append(batches[key], schedule.CustomerID)
		}
	}

	result := &servicesports.InvoiceRunSweepResult{}
	for _, key := range order {
		result.SchedulesDue += len(batches[key])
		s.sweepPeriod(ctx, key, starts[key], batches[key], actor, result)
	}

	return result, nil
}

func (s *Service) sweepPeriod(
	ctx context.Context,
	key periodKey,
	periodStart int64,
	customerIDs []pulid.ID,
	actor *servicesports.RequestActor,
	result *servicesports.InvoiceRunSweepResult,
) {
	tenantInfo := pagination.TenantInfo{OrgID: key.orgID, BuID: key.buID}

	// A tick that re-fires, or two workers racing the same schedule, must not
	// build the period twice. The unique index would refuse the second insert;
	// finding the existing run first turns that into a resume.
	run, err := s.repo.GetOpenScheduledRun(ctx, repositories.GetOpenScheduledRunRequest{
		TenantInfo: tenantInfo,
		Cycle:      string(key.cycle),
		PeriodEnd:  key.periodEnd,
	})
	if err == nil {
		// Re-read with groups: whether the run may bill without review is a
		// property of its groups, not of the run row.
		run, err = s.repo.GetByID(ctx, repositories.GetInvoiceRunByIDRequest{
			ID:            run.ID,
			TenantInfo:    tenantInfo,
			IncludeGroups: true,
		})
	}
	if err != nil {
		run, err = s.Preview(ctx, &servicesports.PreviewInvoiceRunRequest{
			TenantInfo:  tenantInfo,
			CustomerIDs: customerIDs,
			PeriodStart: periodStart,
			PeriodEnd:   key.periodEnd,
			Source:      invoicerun.SourceScheduled,
			Cycle:       key.cycle,
		}, actor)
		if err != nil {
			s.l.Error("failed to build scheduled invoice run", zap.Error(err))
			result.Failed++
			return
		}
		result.RunsBuilt++
	}

	if run.Status.IsTerminal() {
		return
	}

	// A customer who has not asked to be billed without review keeps their run at
	// Ready, where a biller sees it waiting. Committing everything automatically
	// would take the decision away from the people who asked to make it.
	if !runIsAutoBillable(run) {
		return
	}

	commit, err := s.Commit(ctx, &servicesports.CommitInvoiceRunRequest{
		TenantInfo: tenantInfo,
		RunID:      run.ID,
	}, actor)
	if err != nil {
		s.l.Error("failed to commit scheduled invoice run", zap.Error(err))
		result.Failed++
		return
	}

	result.InvoicesCreated += commit.SuccessCount
	result.GroupsSkipped += commit.SkippedCount
	result.Failed += commit.ErrorCount
}

// runIsAutoBillable reports whether every group on a scheduled run belongs to a
// customer who asked to be billed without review.
//
// All-or-nothing on purpose: a run mixing auto-bill and review customers is
// better left whole for a biller than half-committed, which would leave them
// looking at a run whose numbers no longer match what it billed.
func runIsAutoBillable(run *invoicerun.InvoiceRun) bool {
	if len(run.Groups) == 0 {
		return false
	}

	for _, group := range run.Groups {
		if group == nil {
			continue
		}
		if group.Status == invoicerun.GroupStatusPending && !group.AutoBill {
			return false
		}
	}

	return true
}
