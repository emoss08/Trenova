package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// ListUnsettledEventsForWorker exposes the pool of accrued, unheld events an
// instant payout can draw from.
func (s *Service) ListUnsettledEventsForWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) ([]*driversettlement.PayEvent, error) {
	return s.payEventRepo.ListAccruedForWorker(ctx, &repositories.ListAccruedPayEventsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
}

type PayWorkerNowRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	// PayEventIDs limits the payout to specific accrued events; empty pays
	// everything the worker has accrued and unheld right now.
	PayEventIDs []pulid.ID
	// ApplyRecurring also runs the period-cycle items (recurring deductions,
	// escrow, advance recovery, carry-forward). Off by default so an instant
	// payout doesn't double-dip deductions the weekly settlement will take.
	ApplyRecurring   bool
	PaymentMethod    string
	PaymentReference string
}

// PayWorkerNow generates an off-cycle settlement from the selected accrued
// events and drives it through the full lifecycle — approve, post to the GL,
// and mark paid — in one call. If a downstream step fails, the settlement is
// left in its last good state for manual follow-up in the workspace.
func (s *Service) PayWorkerNow(
	ctx context.Context,
	req *PayWorkerNowRequest,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := requireActor(actor, "Instant settlement"); err != nil {
		return nil, err
	}
	generation, err := s.instantGeneration(ctx, req)
	if err != nil {
		return nil, err
	}

	settlement, err := s.GenerateForWorker(ctx, generation, actor)
	if err != nil {
		return nil, err
	}
	if settlement == nil {
		return nil, errNoInstantSettlement()
	}

	return s.driveToPaid(ctx, req, settlement, actor)
}

func (s *Service) instantGeneration(
	ctx context.Context,
	req *PayWorkerNowRequest,
) (*GenerateForWorkerRequest, error) {
	if req.PaymentMethod == "" {
		return nil, errortypes.NewValidationError(
			"paymentMethod",
			errortypes.ErrRequired,
			"Payment method is required",
		)
	}

	events, err := s.payEventRepo.ListAccruedForWorker(
		ctx,
		&repositories.ListAccruedPayEventsRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
			EventIDs:   req.PayEventIDs,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(req.PayEventIDs) > 0 && len(events) != len(req.PayEventIDs) {
		return nil, errortypes.NewValidationError(
			"payEventIds",
			errortypes.ErrInvalidOperation,
			"Some selected pay events are no longer payable — they may be settled, held, or voided. Refresh and try again.",
		)
	}
	if len(events) == 0 {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalidOperation,
			"This driver has no payable accrued events. Pay accrues once the load reaches the pay trigger milestone.",
		)
	}

	now := timeutils.NowUnix()
	periodStart := events[0].EventDate
	for _, event := range events {
		if event.EventDate < periodStart {
			periodStart = event.EventDate
		}
	}
	periodEnd := now
	if periodEnd <= periodStart {
		periodEnd = periodStart + 1
	}

	eventIDs := req.PayEventIDs
	if len(eventIDs) == 0 {
		eventIDs = make([]pulid.ID, 0, len(events))
		for _, event := range events {
			eventIDs = append(eventIDs, event.ID)
		}
	}

	return &GenerateForWorkerRequest{
		TenantInfo:    req.TenantInfo,
		WorkerID:      req.WorkerID,
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		PayDate:       now,
		PayEventIDs:   eventIDs,
		SkipRecurring: !req.ApplyRecurring,
	}, nil
}

type InstantPayPlan struct {
	Settlement *driversettlement.Settlement
	Journal    *settlementshared.JournalPlan
	Refusal    error
}

func (s *Service) PlanPayWorkerNow(
	ctx context.Context,
	req *PayWorkerNowRequest,
) (*InstantPayPlan, error) {
	plan := &InstantPayPlan{}
	generation, err := s.instantGeneration(ctx, req)
	if err != nil {
		if settlementshared.IsRefusal(err) {
			plan.Refusal = err
			return plan, nil
		}
		return nil, err
	}

	control, err := s.settlementControl.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	settlement, _, err := s.buildSettlement(ctx, generation, control)
	if err != nil {
		if settlementshared.IsRefusal(err) {
			plan.Refusal = err
			return plan, nil
		}
		return nil, err
	}
	if settlement == nil {
		plan.Refusal = errNoInstantSettlement()
		return plan, nil
	}

	now := timeutils.NowUnix()
	userID := req.TenantInfo.UserID
	settlement.Status = driversettlement.StatusApproved
	draft, err := s.planSettlementJournal(ctx, settlement, userID, false)
	if err != nil {
		if settlementshared.IsRefusal(err) {
			plan.Refusal = err
			return plan, nil
		}
		return nil, err
	}
	if draft != nil {
		plan.Journal = draft.plan
	}
	if err = PlanPost(settlement, userID, now); err != nil {
		return nil, err
	}
	if err = PlanMarkPaid(settlement, &MarkPaidInput{
		PaymentMethod:    req.PaymentMethod,
		PaymentReference: req.PaymentReference,
		PaidAt:           now,
		UserID:           userID,
	}); err != nil {
		return nil, err
	}
	plan.Settlement = settlement
	return plan, nil
}

func errNoInstantSettlement() error {
	return errortypes.NewValidationError(
		"workerId",
		errortypes.ErrInvalidOperation,
		"No settlement could be generated for this driver",
	)
}

// driveToPaid advances an instant settlement through approve → post → paid,
// tolerating states already reached (e.g. auto-approve or auto-post controls).
func (s *Service) driveToPaid(
	ctx context.Context,
	req *PayWorkerNowRequest,
	settlement *driversettlement.Settlement,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	current := settlement

	if current.Status == driversettlement.StatusDraft ||
		current.Status == driversettlement.StatusPendingApproval {
		approved, err := s.approveInternal(ctx, req.TenantInfo, current.ID, actor, true)
		if err != nil {
			return current, s.instantStepError(current, "approve", err)
		}
		current = approved
	}

	if current.Status == driversettlement.StatusApproved {
		posted, err := s.Post(ctx, req.TenantInfo, current.ID, actor)
		if err != nil {
			return current, s.instantStepError(current, "post", err)
		}
		current = posted
	}

	if current.Status == driversettlement.StatusPosted {
		paid, err := s.MarkPaid(ctx, &serviceports.MarkSettlementPaidRequest{
			TenantInfo:       req.TenantInfo,
			SettlementID:     current.ID,
			PaymentMethod:    req.PaymentMethod,
			PaymentReference: req.PaymentReference,
		}, actor)
		if err != nil {
			return current, s.instantStepError(current, "mark paid", err)
		}
		current = paid
	}

	return current, nil
}

func (s *Service) instantStepError(
	settlement *driversettlement.Settlement,
	step string,
	err error,
) error {
	s.l.Error("instant settlement stopped mid-lifecycle",
		zap.String("settlementId", settlement.ID.String()),
		zap.String("step", step),
		zap.Error(err))
	return errortypes.NewValidationError(
		"settlementId",
		errortypes.ErrInvalidOperation,
		"Settlement {0} was created but the {1} step failed: {2} — finish it from the settlement page.",
		settlement.SettlementNumber,
		step,
		err.Error(),
	)
}
