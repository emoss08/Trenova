package invoicerunservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

// windowKey batches the customers whose open periods happen to coincide.
//
// Most customers on the same cadence share a boundary — everybody billing
// monthly on the 1st in the same zone has the identical window — so batching
// turns "one candidate query per customer" into a handful of queries no matter
// how many customers there are.
type windowKey struct {
	start int64
	end   int64
}

// ListOpenStatements is what every statement-billed customer has accumulated in
// the period they are currently in.
//
// Derived on read rather than persisted. A statement is a view of the billing
// queue through the customer's own schedule, so approving a shipment puts it on
// the statement immediately and there is no job to wait for, no staging table to
// fall out of step, and nothing to reconcile when a biller changes a charge.
func (s *Service) ListOpenStatements(
	ctx context.Context,
	req *servicesports.ListOpenStatementsRequest,
) ([]*servicesports.OpenStatement, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	schedules, err := s.customerRepo.ListDueBillingSchedules(
		ctx,
		&repositories.ListBillingSchedulesRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}

	if !req.CustomerID.IsNil() {
		schedules = filterToCustomer(schedules, req.CustomerID)
	}
	if len(schedules) == 0 {
		return []*servicesports.OpenStatement{}, nil
	}

	now := timeutils.NowUnix()

	statements := make([]*servicesports.OpenStatement, 0, len(schedules))
	byCustomer := make(map[pulid.ID]*servicesports.OpenStatement, len(schedules))
	batches := make(map[windowKey][]pulid.ID)
	windows := make([]windowKey, 0, 4)

	for _, schedule := range schedules {
		period, ok := CurrentPeriod(schedule.Profile(), now)
		if !ok {
			continue
		}

		statement := newOpenStatement(schedule, period)
		statements = append(statements, statement)
		byCustomer[schedule.CustomerID] = statement

		key := windowKey{start: period.Start, end: period.End}
		if _, seen := batches[key]; !seen {
			windows = append(windows, key)
		}
		batches[key] = append(batches[key], schedule.CustomerID)
	}

	for _, window := range windows {
		if err = s.fillStatements(ctx, req, window, batches[window], byCustomer); err != nil {
			return nil, err
		}
	}

	return statements, nil
}

// GetOpenStatement is one customer's open statement with its members filled in.
func (s *Service) GetOpenStatement(
	ctx context.Context,
	req *servicesports.ListOpenStatementsRequest,
) (*servicesports.OpenStatement, error) {
	if req == nil || req.CustomerID.IsNil() {
		return nil, errortypes.NewValidationError(
			"customerId",
			errortypes.ErrRequired,
			"Customer is required",
		)
	}

	req.IncludeShipments = true

	statements, err := s.ListOpenStatements(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(statements) == 0 {
		return nil, errortypes.NewNotFoundError(
			"This customer is not on a statement billing schedule",
		)
	}

	return statements[0], nil
}

// fillStatements resolves one shared period window for every customer in it.
func (s *Service) fillStatements(
	ctx context.Context,
	req *servicesports.ListOpenStatementsRequest,
	window windowKey,
	customerIDs []pulid.ID,
	byCustomer map[pulid.ID]*servicesports.OpenStatement,
) error {
	candidates, err := s.billingQueueRepo.ListConsolidationCandidates(
		ctx,
		&repositories.ListConsolidationCandidatesRequest{
			TenantInfo:  req.TenantInfo,
			CustomerIDs: customerIDs,
			PeriodStart: window.start,
			PeriodEnd:   window.end,
		},
	)
	if err != nil {
		return err
	}

	// Grouping runs through the same splitter the run builder uses, so the
	// invoices a biller is shown accumulating are the invoices the run will cut.
	for _, group := range GroupCandidates(candidates) {
		statement, ok := byCustomer[group.First.CustomerID]
		if !ok {
			continue
		}
		statement.Groups = append(
			statement.Groups,
			newStatementGroup(group, req.IncludeShipments),
		)
	}

	for _, customerID := range customerIDs {
		if statement, ok := byCustomer[customerID]; ok {
			syncStatementTotals(statement)
		}
	}

	return s.fillHeldFreight(ctx, req, window, customerIDs, byCustomer)
}

// fillHeldFreight records the freight that belongs to these periods but has not
// cleared the billing queue yet.
//
// One grouped query for the whole window rather than one per customer, on the
// same principle as the candidate fetch: a deployment with hundreds of statement
// customers must not pay a round trip each to answer "anything still waiting?".
func (s *Service) fillHeldFreight(
	ctx context.Context,
	req *servicesports.ListOpenStatementsRequest,
	window windowKey,
	customerIDs []pulid.ID,
	byCustomer map[pulid.ID]*servicesports.OpenStatement,
) error {
	held, err := s.billingQueueRepo.CountHeldForPeriod(
		ctx,
		&repositories.CountHeldForPeriodRequest{
			TenantInfo:  req.TenantInfo,
			CustomerIDs: customerIDs,
			PeriodStart: window.start,
			PeriodEnd:   window.end,
		},
	)
	if err != nil {
		return err
	}

	for _, row := range held {
		statement, ok := byCustomer[row.CustomerID]
		if !ok {
			continue
		}
		statement.HeldCount = row.ShipmentCount
		statement.HeldAmount = row.TotalAmount
	}

	return nil
}

// BillStatementNow bills an open period before its boundary.
//
// The deviation is deliberate and recorded: the run is stamped with the reason
// and the billing watermark advances to the moment it was billed, so the rest of
// the period keeps accruing to the same boundary the customer expects. Billing
// early never moves the cadence.
func (s *Service) BillStatementNow(
	ctx context.Context,
	req *servicesports.BillStatementNowRequest,
	actor *servicesports.RequestActor,
) (*servicesports.CommitInvoiceRunResult, error) {
	if req == nil || actor == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request and actor are required",
		)
	}
	if req.Reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Say why this statement is being billed before its cycle closes",
		)
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

	// The period is cut at now rather than at the customer's boundary, because
	// everything after this moment still belongs to the period they agreed to.
	now := timeutils.NowUnix()

	run, err := s.Preview(ctx, &servicesports.PreviewInvoiceRunRequest{
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
	if _, err = s.repo.Update(ctx, run); err != nil {
		return nil, err
	}

	if run, err = s.applyStatementExclusions(ctx, run, req, actor); err != nil {
		return nil, err
	}

	return s.Commit(ctx, &servicesports.CommitInvoiceRunRequest{
		TenantInfo: req.TenantInfo,
		RunID:      run.ID,
	}, actor)
}

// applyStatementExclusions translates the biller's picks onto the run that was
// just built.
//
// An open statement has no item ids of its own — nothing is persisted until it
// bills — so the biller works in billing-queue item ids and they are resolved
// here, after the preview has minted the run's items. An id that no longer
// matches anything is dropped rather than failing the whole bill: it means the
// shipment stopped being eligible between the read and the click, which is the
// outcome the biller wanted anyway.
func (s *Service) applyStatementExclusions(
	ctx context.Context,
	run *invoicerun.InvoiceRun,
	req *servicesports.BillStatementNowRequest,
	actor *servicesports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	if len(req.Exclude) == 0 {
		return run, nil
	}

	reasons := make(map[pulid.ID]string, len(req.Exclude))
	for _, exclusion := range req.Exclude {
		reasons[exclusion.BillingQueueItemID] = exclusion.Reason
	}

	exclusions := make([]servicesports.ItemExclusion, 0, len(req.Exclude))
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
				reason = "Held back by the biller before this statement billed"
			}
			exclusions = append(exclusions, servicesports.ItemExclusion{
				ItemID: item.ID,
				Reason: reason,
			})
		}
	}

	if len(exclusions) == 0 {
		return run, nil
	}

	return s.AdjustMembership(ctx, &servicesports.AdjustInvoiceRunMembershipRequest{
		TenantInfo: req.TenantInfo,
		RunID:      run.ID,
		Exclude:    exclusions,
	}, actor)
}

func newOpenStatement(
	schedule *repositories.DueBillingSchedule,
	period Period,
) *servicesports.OpenStatement {
	return &servicesports.OpenStatement{
		CustomerID:            schedule.CustomerID,
		CustomerName:          schedule.CustomerName,
		CustomerCode:          schedule.CustomerCode,
		CustomerStatus:        schedule.CustomerStatus,
		Cycle:                 schedule.BillingCycle,
		BillingCycleAnchorDay: schedule.BillingCycleAnchorDay,
		BillingCycleTimezone:  schedule.BillingCycleTimezone,
		PeriodStart:           period.Start,
		PeriodEnd:             period.End,
		LastBilledPeriodEnd:   schedule.LastBilledPeriodEnd,
		CurrencyCode:          schedule.BillingCurrency,
		SplitBy:               schedule.SplitBy,
		SectionBy:             schedule.SectionBy,
		Detail:                schedule.InvoiceDetail,
		MinimumAmount:         schedule.MinConsolidatedAmount,
		AutoBill:              schedule.AutoBill,
		TotalAmount:           decimal.Zero,
		HeldAmount:            decimal.Zero,
		Groups:                make([]*servicesports.StatementGroup, 0, 1),
	}
}

func newStatementGroup(
	group CandidateGroup,
	includeShipments bool,
) *servicesports.StatementGroup {
	out := &servicesports.StatementGroup{
		Key:           group.Key,
		Label:         group.Label,
		ShipmentCount: len(group.Members),
		TotalAmount:   decimal.Zero,
	}

	if includeShipments {
		out.Shipments = make([]*servicesports.StatementShipment, 0, len(group.Members))
	}

	for _, member := range group.Members {
		amount := member.TotalChargeAmount.Decimal
		out.TotalAmount = out.TotalAmount.Add(amount)

		if !includeShipments {
			continue
		}
		out.Shipments = append(out.Shipments, &servicesports.StatementShipment{
			BillingQueueItemID: member.BillingQueueItemID,
			ShipmentID:         member.ShipmentID,
			OrderID:            member.OrderID,
			ProNumber:          member.ProNumber,
			BOL:                member.ShipmentBOL,
			PONumber:           member.OrderPONumber,
			OrderNumber:        member.OrderNumber,
			ServiceDate:        member.ServiceDate,
			Amount:             amount,
		})
	}

	if minimum := group.First.MinConsolidatedAmount; minimum.Valid {
		out.BelowMinimum = out.TotalAmount.LessThan(minimum.Decimal)
	}

	return out
}

// syncStatementTotals rolls the groups up.
//
// BelowMinimum is true only when every group is under the floor: a statement
// with one billable group and one held group still bills, and saying otherwise
// would tell a biller nothing is going out when something is.
func syncStatementTotals(statement *servicesports.OpenStatement) {
	statement.InvoiceCount = len(statement.Groups)
	statement.ShipmentCount = 0
	statement.TotalAmount = decimal.Zero
	statement.BelowMinimum = len(statement.Groups) > 0

	for _, group := range statement.Groups {
		statement.ShipmentCount += group.ShipmentCount
		statement.TotalAmount = statement.TotalAmount.Add(group.TotalAmount)
		if !group.BelowMinimum {
			statement.BelowMinimum = false
		}
	}
}

func filterToCustomer(
	schedules []*repositories.DueBillingSchedule,
	customerID pulid.ID,
) []*repositories.DueBillingSchedule {
	for _, schedule := range schedules {
		if schedule.CustomerID == customerID {
			return []*repositories.DueBillingSchedule{schedule}
		}
	}

	return nil
}

// StatementCadenceGuard reports whether invoicing a shipment on its own deviates
// from the customer's agreed billing cadence.
//
// It is advisory rather than a bar: a biller who needs to cut one invoice today
// for a customer on a monthly statement is doing something legitimate often
// enough that refusing would just teach them to work around it. What it must not
// be is silent, so the caller has to supply a reason.
func StatementCadenceGuard(
	schedule *repositories.DueBillingSchedule,
) (string, bool) {
	if schedule == nil {
		return "", false
	}

	profile := schedule.Profile()
	if !profile.IsStatementBilled() {
		return "", false
	}

	return fmt.Sprintf(
		"%s is billed on a %s statement. Invoicing this shipment on its own takes it off that statement.",
		schedule.CustomerName,
		profile.BillingCycle.Describe(),
	), true
}
