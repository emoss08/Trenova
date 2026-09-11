package invoicerunservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/invoiceservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Commit turns a reviewed proposal into invoices.
//
// One transaction per group rather than one for the run: a four-hundred-customer
// monthly run in a single transaction would hold locks for minutes and roll every
// invoice back because one customer was misconfigured. A group that cannot be
// billed is skipped with a reason and the rest still bill.
//
// Re-entrant by design. A run already Committed returns what it produced; a run
// left Committing by a dead worker resumes at the first group without an invoice,
// which is what stops a Temporal retry from billing the same freight twice.
func (s *Service) Commit(
	ctx context.Context,
	req *servicesports.CommitInvoiceRunRequest,
	actor *servicesports.RequestActor,
) (*servicesports.CommitInvoiceRunResult, error) {
	if req == nil || actor == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request and actor are required",
		)
	}

	run, err := s.repo.GetByID(ctx, repositories.GetInvoiceRunByIDRequest{
		ID:            req.RunID,
		TenantInfo:    req.TenantInfo,
		IncludeGroups: true,
		IncludeItems:  true,
	})
	if err != nil {
		return nil, err
	}

	if run.Status == invoicerun.StatusCommitted {
		return s.resultFromRun(run), nil
	}
	if multiErr := s.validator.ValidateCommit(run); multiErr != nil {
		return nil, multiErr
	}

	if run.Status != invoicerun.StatusCommitting {
		run.Status = invoicerun.StatusCommitting
		if run, err = s.repo.Update(ctx, run); err != nil {
			return nil, err
		}
		if run, err = s.repo.GetByID(ctx, repositories.GetInvoiceRunByIDRequest{
			ID:            req.RunID,
			TenantInfo:    req.TenantInfo,
			IncludeGroups: true,
			IncludeItems:  true,
		}); err != nil {
			return nil, err
		}
	}

	result := &servicesports.CommitInvoiceRunResult{
		Results: make([]servicesports.CommitGroupResult, 0, len(run.Groups)),
	}

	for _, group := range run.Groups {
		if group == nil {
			continue
		}
		result.Results = append(result.Results, s.commitGroup(ctx, run, group, actor))
	}

	return s.finishRun(ctx, run, result, actor)
}

func (s *Service) commitGroup(
	ctx context.Context,
	run *invoicerun.InvoiceRun,
	group *invoicerun.InvoiceRunGroup,
	actor *servicesports.RequestActor,
) servicesports.CommitGroupResult {
	outcome := servicesports.CommitGroupResult{
		GroupID:    group.ID,
		GroupLabel: group.GroupLabel,
	}

	// A group committed by an earlier attempt is left alone; that is what makes a
	// retry safe.
	if group.Status == invoicerun.GroupStatusCommitted {
		outcome.Success = true
		outcome.InvoiceID = group.InvoiceID
		return outcome
	}
	if group.Status == invoicerun.GroupStatusSkipped {
		outcome.Skipped = true
		outcome.Error = group.SkipReason
		return outcome
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: run.OrganizationID,
		BuID:  run.BusinessUnitID,
	}

	included := group.IncludedItems()
	if len(included) == 0 {
		return s.skipGroup(ctx, group, &outcome, "Every shipment on this group was excluded")
	}

	// A customer's floor is a deliberate instruction not to send them a trivial
	// invoice. The shipments stay billable, so they roll into the next period
	// rather than being lost.
	if reason := belowMinimum(group, included); reason != "" {
		return s.skipGroup(ctx, group, &outcome, reason)
	}

	legs, reason, err := s.resolveGroupLegs(ctx, tenantInfo, included)
	if err != nil {
		return s.failGroup(ctx, group, &outcome, err)
	}
	if reason != "" {
		return s.skipGroup(ctx, group, &outcome, reason)
	}

	invoiceEntity, err := s.consolidatedMaker.CreateConsolidated(
		ctx,
		&invoiceservice.ConsolidatedInvoiceParams{
			TenantInfo:  tenantInfo,
			Legs:        legs,
			RunID:       run.ID,
			InvoiceDate: run.InvoiceDate,
			PeriodStart: run.PeriodStart,
			PeriodEnd:   run.PeriodEnd,
		},
		actor,
	)
	if err != nil {
		return s.failGroup(ctx, group, &outcome, err)
	}

	group.Status = invoicerun.GroupStatusCommitted
	group.InvoiceID = invoiceEntity.ID
	group.TotalAmount = invoiceEntity.TotalAmount
	group.TotalAmountMinor = invoiceEntity.TotalAmountMinor
	if _, err = s.repo.UpdateGroup(ctx, group); err != nil {
		s.l.Error("failed to mark invoice run group committed", zap.Error(err))
	}

	outcome.Success = true
	outcome.InvoiceID = invoiceEntity.ID
	outcome.InvoiceNumber = invoiceEntity.Number

	return outcome
}

// resolveGroupLegs loads the shipments a group bills and re-checks that every
// queue item is still billable.
//
// The re-check is the point: between preview and commit another operator can
// invoice one of these shipments. Detecting it here turns that into a skipped
// group with a reason the biller can read, instead of a unique-index violation
// from the partial index that names nothing useful.
func (s *Service) resolveGroupLegs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	items []*invoicerun.InvoiceRunGroupItem,
) ([]*shipment.Shipment, string, error) {
	legs := make([]*shipment.Shipment, 0, len(items))

	for _, item := range items {
		queueItem, err := s.billingQueueRepo.GetByID(
			ctx,
			&repositories.GetBillingQueueItemByIDRequest{
				ItemID:     item.BillingQueueItemID,
				TenantInfo: tenantInfo,
			},
		)
		if err != nil {
			return nil, "", err
		}
		if !queueItem.InvoiceID.IsNil() {
			return nil, "Some shipments were invoiced elsewhere after this run was built", nil
		}
		if queueItem.Status != billingqueue.StatusApproved {
			return nil, "Some shipments are no longer approved for billing", nil
		}

		shp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
			ID:         item.ShipmentID,
			TenantInfo: tenantInfo,
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		if err != nil {
			return nil, "", err
		}
		legs = append(legs, shp)
	}

	return legs, "", nil
}

func (s *Service) skipGroup(
	ctx context.Context,
	group *invoicerun.InvoiceRunGroup,
	outcome *servicesports.CommitGroupResult,
	reason string,
) servicesports.CommitGroupResult {
	group.Status = invoicerun.GroupStatusSkipped
	group.SkipReason = reason
	if _, err := s.repo.UpdateGroup(ctx, group); err != nil {
		s.l.Error("failed to mark invoice run group skipped", zap.Error(err))
	}

	outcome.Skipped = true
	outcome.Error = reason

	return *outcome
}

func (s *Service) failGroup(
	ctx context.Context,
	group *invoicerun.InvoiceRunGroup,
	outcome *servicesports.CommitGroupResult,
	cause error,
) servicesports.CommitGroupResult {
	group.Status = invoicerun.GroupStatusFailed
	group.SkipReason = cause.Error()
	if _, err := s.repo.UpdateGroup(ctx, group); err != nil {
		s.l.Error("failed to mark invoice run group failed", zap.Error(err))
	}

	outcome.Error = cause.Error()

	return *outcome
}

// finishRun rolls the group outcomes up and advances the customers' billing
// watermark.
func (s *Service) finishRun(
	ctx context.Context,
	run *invoicerun.InvoiceRun,
	result *servicesports.CommitInvoiceRunResult,
	actor *servicesports.RequestActor,
) (*servicesports.CommitInvoiceRunResult, error) {
	for _, outcome := range result.Results {
		result.TotalCount++
		switch {
		case outcome.Success:
			result.SuccessCount++
		case outcome.Skipped:
			result.SkippedCount++
		default:
			result.ErrorCount++
		}
	}

	// Only a run where nothing at all succeeded is a failure; a partial commit is
	// a real outcome the operator acts on, not an error to roll back.
	if result.SuccessCount == 0 && result.ErrorCount > 0 {
		run.Status = invoicerun.StatusFailed
		run.FailureReason = "No group on this run could be invoiced"
	} else {
		run.Status = invoicerun.StatusCommitted
	}

	committedAt := timeutils.NowUnix()
	run.CommittedAt = &committedAt
	run.CommittedByID = actor.UserID
	run.InvoiceCount = result.SuccessCount

	updated, err := s.repo.Update(ctx, run)
	if err != nil {
		return nil, err
	}

	if run.Status == invoicerun.StatusCommitted {
		s.advanceBillingWatermark(ctx, run)
	}

	s.audit(ctx, updated, actor, permission.OpApprove, "Invoice run committed")
	result.Run = updated

	return result, nil
}

// advanceBillingWatermark moves each customer past the period just billed.
//
// Advancing only on a committed run is deliberate: a run that failed outright
// must be re-billable for the same period, and leaving the watermark where it is
// is what lets the next tick pick it up rather than skipping the month.
func (s *Service) advanceBillingWatermark(ctx context.Context, run *invoicerun.InvoiceRun) {
	for _, raw := range run.CustomerIDs {
		customerID, err := pulidFromString(raw)
		if err != nil {
			continue
		}

		if err = s.customerRepo.AdvanceBilledPeriod(
			ctx,
			&repositories.AdvanceBilledPeriodRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: run.OrganizationID,
					BuID:  run.BusinessUnitID,
				},
				CustomerID: customerID,
				PeriodEnd:  run.PeriodEnd,
			},
		); err != nil {
			s.l.Error("failed to advance billed period", zap.Error(err))
		}
	}
}

func (s *Service) resultFromRun(
	run *invoicerun.InvoiceRun,
) *servicesports.CommitInvoiceRunResult {
	result := &servicesports.CommitInvoiceRunResult{
		Run:     run,
		Results: make([]servicesports.CommitGroupResult, 0, len(run.Groups)),
	}

	for _, group := range run.Groups {
		if group == nil {
			continue
		}
		outcome := servicesports.CommitGroupResult{
			GroupID:    group.ID,
			GroupLabel: group.GroupLabel,
			InvoiceID:  group.InvoiceID,
		}
		switch group.Status {
		case invoicerun.GroupStatusCommitted:
			outcome.Success = true
			result.SuccessCount++
		case invoicerun.GroupStatusSkipped:
			outcome.Skipped = true
			outcome.Error = group.SkipReason
			result.SkippedCount++
		default:
			outcome.Error = group.SkipReason
			result.ErrorCount++
		}
		result.TotalCount++
		result.Results = append(result.Results, outcome)
	}

	return result
}

// belowMinimum reports why a group should defer, or empty when it should bill.
func belowMinimum(
	group *invoicerun.InvoiceRunGroup,
	included []*invoicerun.InvoiceRunGroupItem,
) string {
	if !group.MinimumAmount.Valid {
		return ""
	}

	total := decimal.Zero
	for _, item := range included {
		total = total.Add(item.Amount)
	}
	if total.GreaterThanOrEqual(group.MinimumAmount.Decimal) {
		return ""
	}

	return fmt.Sprintf(
		"Below this customer's %s minimum — held for the next period",
		group.MinimumAmount.Decimal.StringFixed(2),
	)
}
