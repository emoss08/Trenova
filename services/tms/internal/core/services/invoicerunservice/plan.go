package invoicerunservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	unnumberedRun        = "Unnumbered"
	everyShipmentRemoved = "Every shipment on this group was excluded"
	statementHeldBack    = "Held back by the biller before this statement billed"
)

func planRunHeader(
	req *servicesports.PreviewInvoiceRunRequest,
	actor *servicesports.RequestActor,
) (*invoicerun.InvoiceRun, []pulid.ID, error) {
	if req == nil || actor == nil {
		return nil, nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request and actor are required",
		)
	}

	customerIDs := req.CustomerIDs
	if len(customerIDs) == 0 {
		return nil, nil, errortypes.NewValidationError(
			"customerIds",
			errortypes.ErrRequired,
			"Select at least one customer to bill",
		)
	}

	invoiceDate := req.InvoiceDate
	if invoiceDate == 0 {
		invoiceDate = timeutils.NowUnix()
	}

	source := req.Source
	if source == "" {
		source = invoicerun.SourceManual
	}

	return &invoicerun.InvoiceRun{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Status:         invoicerun.StatusBuilding,
		Source:         source,
		Cycle:          req.Cycle,
		PeriodStart:    req.PeriodStart,
		PeriodEnd:      req.PeriodEnd,
		InvoiceDate:    invoiceDate,
		CustomerIDs:    idsToStrings(customerIDs),
		CurrencyCode:   money.DefaultCurrencyCode,
		BuiltByID:      actor.UserID,
	}, customerIDs, nil
}

func (s *Service) PreviewBuild(
	ctx context.Context,
	req *servicesports.PreviewInvoiceRunRequest,
	actor *servicesports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	run, customerIDs, err := planRunHeader(req, actor)
	if err != nil {
		return nil, err
	}
	run.Number = unnumberedRun
	if multiErr := s.validator.ValidateCreate(run); multiErr != nil {
		return nil, multiErr
	}

	groups, err := s.buildGroups(ctx, run, customerIDs)
	if err != nil {
		return nil, err
	}
	run.Groups = groups
	run.SyncTotals()
	run.Status = invoicerun.StatusReady

	return run, nil
}

func planCancel(
	run *invoicerun.InvoiceRun,
	req *servicesports.CancelInvoiceRunRequest,
	actor *servicesports.RequestActor,
	now int64,
) error {
	if !invoicerun.CanTransition(run.Status, invoicerun.StatusCanceled) {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"An invoice run that is {0} cannot be canceled", run.Status,
		)
	}

	run.Status = invoicerun.StatusCanceled
	run.CanceledByID = actor.UserID
	run.CanceledAt = &now
	run.FailureReason = req.Reason

	return nil
}

func (s *Service) loadRun(
	ctx context.Context,
	runID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*invoicerun.InvoiceRun, error) {
	return s.repo.GetByID(ctx, repositories.GetInvoiceRunByIDRequest{
		ID:            runID,
		TenantInfo:    tenantInfo,
		IncludeGroups: true,
		IncludeItems:  true,
	})
}

func (s *Service) PreviewCancel(
	ctx context.Context,
	req *servicesports.CancelInvoiceRunRequest,
	actor *servicesports.RequestActor,
) (*servicesports.InvoiceRunChangePreview, error) {
	if req == nil || actor == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request and actor are required",
		)
	}

	run, err := s.loadRun(ctx, req.RunID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	after, err := cloneRun(run)
	if err != nil {
		return nil, err
	}
	if err = planCancel(after, req, actor, timeutils.NowUnix()); err != nil {
		return nil, err
	}

	return &servicesports.InvoiceRunChangePreview{Before: run, After: after}, nil
}

func applyMembership(
	run *invoicerun.InvoiceRun,
	req *servicesports.AdjustInvoiceRunMembershipRequest,
) (map[pulid.ID]*invoicerun.InvoiceRunGroupItem, error) {
	index := indexItems(run)
	touched := make(map[pulid.ID]*invoicerun.InvoiceRunGroupItem, len(req.Exclude)+len(req.Moves))

	for _, exclusion := range req.Exclude {
		item, ok := index.items[exclusion.ItemID]
		if !ok {
			return nil, unknownItem(exclusion.ItemID)
		}
		if exclusion.Reason == "" {
			return nil, errortypes.NewValidationError(
				"exclude",
				errortypes.ErrRequired,
				"Say why the shipment is being taken off the invoice",
			)
		}
		item.Excluded = true
		item.ExclusionReason = exclusion.Reason
		touched[item.ID] = item
	}

	for _, itemID := range req.Include {
		item, ok := index.items[itemID]
		if !ok {
			return nil, unknownItem(itemID)
		}
		item.Excluded = false
		item.ExclusionReason = ""
		touched[item.ID] = item
	}

	for _, move := range req.Moves {
		item, ok := index.items[move.ItemID]
		if !ok {
			return nil, unknownItem(move.ItemID)
		}
		target, ok := index.groups[move.TargetGroupID]
		if !ok {
			return nil, unknownGroup(move.TargetGroupID)
		}
		if target.CustomerID != index.groups[item.GroupID].CustomerID {
			return nil, errortypes.NewValidationError(
				"moves",
				errortypes.ErrInvalid,
				"A shipment can only move between groups of the same customer",
			)
		}
		item.GroupID = move.TargetGroupID
		touched[item.ID] = item
	}

	return touched, nil
}

func (s *Service) PreviewMembership(
	ctx context.Context,
	req *servicesports.AdjustInvoiceRunMembershipRequest,
) (*servicesports.InvoiceRunChangePreview, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	run, err := s.loadRun(ctx, req.RunID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateAdjust(run); multiErr != nil {
		return nil, multiErr
	}

	after, err := cloneRun(run)
	if err != nil {
		return nil, err
	}
	if _, err = applyMembership(after, req); err != nil {
		return nil, err
	}
	regroup(after)
	after.SyncTotals()

	return &servicesports.InvoiceRunChangePreview{Before: run, After: after}, nil
}

func regroup(run *invoicerun.InvoiceRun) {
	byGroup := make(map[pulid.ID][]*invoicerun.InvoiceRunGroupItem, len(run.Groups))
	for _, group := range run.Groups {
		if group == nil {
			continue
		}
		for _, item := range group.Items {
			if item != nil {
				byGroup[item.GroupID] = append(byGroup[item.GroupID], item)
			}
		}
	}
	for _, group := range run.Groups {
		if group != nil {
			group.Items = byGroup[group.ID]
		}
	}
}

func cloneRun(run *invoicerun.InvoiceRun) (*invoicerun.InvoiceRun, error) {
	clone := new(invoicerun.InvoiceRun)
	if err := jsonutils.Convert(run, clone); err != nil {
		return nil, err
	}

	return clone, nil
}

type groupDecision struct {
	plan       servicesports.InvoiceRunGroupPlan
	legs       []*shipment.Shipment
	queueItems []*billingqueue.BillingQueueItem
}

func (s *Service) decideGroup(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	group *invoicerun.InvoiceRunGroup,
) (groupDecision, error) {
	included := group.IncludedItems()
	decision := groupDecision{plan: servicesports.InvoiceRunGroupPlan{
		GroupID:       group.ID,
		GroupLabel:    group.GroupLabel,
		CustomerID:    group.CustomerID,
		InvoiceID:     group.InvoiceID,
		ShipmentCount: len(included),
		Total:         includedTotal(included),
		CurrencyCode:  group.CurrencyCode,
		Outcome:       servicesports.InvoiceRunGroupBills,
	}}

	switch {
	case group.Status == invoicerun.GroupStatusCommitted:
		decision.plan.Outcome = servicesports.InvoiceRunGroupAlreadyCommitted
		return decision, nil
	case group.Status == invoicerun.GroupStatusSkipped:
		decision.plan.Outcome = servicesports.InvoiceRunGroupAlreadySkipped
		decision.plan.Reason = group.SkipReason
		return decision, nil
	case len(included) == 0:
		return decision.skip(everyShipmentRemoved), nil
	}
	if reason := belowMinimum(group, included); reason != "" {
		return decision.skip(reason), nil
	}

	legs, queueItems, reason, err := s.resolveGroupLegs(ctx, tenantInfo, group, included)
	if err != nil {
		return groupDecision{}, err
	}
	if reason != "" {
		return decision.skip(reason), nil
	}
	decision.legs, decision.queueItems = legs, queueItems

	return decision, nil
}

func (d groupDecision) skip(reason string) groupDecision {
	d.plan.Outcome = servicesports.InvoiceRunGroupSkips
	d.plan.Reason = reason

	return d
}

func includedTotal(items []*invoicerun.InvoiceRunGroupItem) decimal.Decimal {
	total := decimal.Zero
	for _, item := range items {
		total = total.Add(item.Amount)
	}

	return total
}

func (s *Service) planGroups(
	ctx context.Context,
	run *invoicerun.InvoiceRun,
) ([]servicesports.InvoiceRunGroupPlan, error) {
	tenantInfo := tenantOf(run)
	plans := make([]servicesports.InvoiceRunGroupPlan, 0, len(run.Groups))
	for _, group := range run.Groups {
		if group == nil {
			continue
		}
		decision, err := s.decideGroup(ctx, tenantInfo, group)
		if err != nil {
			return nil, err
		}
		plans = append(plans, decision.plan)
	}

	return plans, nil
}

func (s *Service) PreviewCommit(
	ctx context.Context,
	req *servicesports.CommitInvoiceRunRequest,
) (*servicesports.InvoiceRunCommitPlan, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	run, err := s.loadRun(ctx, req.RunID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	plan := &servicesports.InvoiceRunCommitPlan{Run: run}
	if run.Status == invoicerun.StatusCommitted {
		plan.AlreadyCommitted = true
		return plan, nil
	}
	if multiErr := s.validator.ValidateCommit(run); multiErr != nil {
		return nil, multiErr
	}
	if plan.Groups, err = s.planGroups(ctx, run); err != nil {
		return nil, err
	}

	return plan, nil
}

func validateBillStatementRequest(
	req *servicesports.BillStatementNowRequest,
	actor *servicesports.RequestActor,
) error {
	if req == nil || actor == nil {
		return errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request and actor are required",
		)
	}
	if req.Reason == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Say why this statement is being billed before its cycle closes",
		)
	}

	return nil
}

func matchStatementExclusions(
	run *invoicerun.InvoiceRun,
	req *servicesports.BillStatementNowRequest,
	apply func(item *invoicerun.InvoiceRunGroupItem, reason string),
) {
	if len(req.Exclude) == 0 {
		return
	}

	reasons := make(map[pulid.ID]string, len(req.Exclude))
	for _, exclusion := range req.Exclude {
		reasons[exclusion.BillingQueueItemID] = exclusion.Reason
	}

	for _, group := range run.Groups {
		if group == nil {
			continue
		}
		for _, item := range group.Items {
			reason, excluded := reasons[item.BillingQueueItemID]
			if !excluded || item.Excluded {
				continue
			}
			if reason == "" {
				reason = statementHeldBack
			}
			apply(item, reason)
		}
	}
}

func (s *Service) PreviewBillStatement(
	ctx context.Context,
	req *servicesports.BillStatementNowRequest,
	actor *servicesports.RequestActor,
) (*servicesports.StatementBillPlan, error) {
	if err := validateBillStatementRequest(req, actor); err != nil {
		return nil, err
	}

	statement, err := s.GetOpenStatement(ctx, &servicesports.ListOpenStatementsRequest{
		TenantInfo: req.TenantInfo,
		CustomerID: req.CustomerID,
	})
	if err != nil {
		return nil, err
	}
	if statement.ShipmentCount == 0 {
		return nil, errortypes.NewValidationError(
			"customerId",
			errortypes.ErrInvalidOperation,
			"This statement has no shipments on it yet",
		)
	}

	now := timeutils.NowUnix()
	run, err := s.PreviewBuild(ctx, &servicesports.PreviewInvoiceRunRequest{
		TenantInfo:  req.TenantInfo,
		CustomerIDs: []pulid.ID{req.CustomerID},
		PeriodStart: statement.PeriodStart,
		PeriodEnd:   now,
		InvoiceDate: now,
		Source:      invoicerun.SourceManual,
		Cycle:       statement.Cycle,
	}, actor)
	if err != nil {
		return nil, err
	}
	run.OffCycleReason = req.Reason

	plan := &servicesports.StatementBillPlan{Statement: statement, Run: run}
	matched := make(map[pulid.ID]struct{}, len(req.Exclude))
	matchStatementExclusions(run, req, func(item *invoicerun.InvoiceRunGroupItem, reason string) {
		item.Excluded = true
		item.ExclusionReason = reason
		matched[item.BillingQueueItemID] = struct{}{}
	})
	for _, exclusion := range req.Exclude {
		if _, ok := matched[exclusion.BillingQueueItemID]; !ok {
			plan.UnmatchedExclusions = append(plan.UnmatchedExclusions,
				exclusion.BillingQueueItemID)
		}
	}
	run.SyncTotals()

	if plan.Groups, err = s.planGroups(ctx, run); err != nil {
		return nil, err
	}

	return plan, nil
}
