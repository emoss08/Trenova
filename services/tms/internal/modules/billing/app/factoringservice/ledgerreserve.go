package factoringservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	releaseKeyPrefix    = "factoring-reserve-released:"
	chargebackKeyPrefix = "factoring-charged-back:"
	reversalKeyPrefix   = "factoring-chargeback-reversed:"
)

type itemDraft struct {
	plan  *factoringledgerports.ReserveMovementPlan
	input *movementInput
}

type itemHistory struct {
	funding     *factoring.ReserveEntry
	chargebacks int
	open        *factoring.ReserveEntry
}

func (s *LedgerService) PlanRelease(
	ctx context.Context,
	req *factoringledgerports.ReserveMovementRequest,
	actor *permissionports.RequestActor,
) (*factoringledgerports.ReserveMovementPlan, error) {
	draft, err := s.planRelease(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	return draft.plan, nil
}

func (s *LedgerService) ReleaseReserve(
	ctx context.Context,
	req *factoringledgerports.ReserveMovementRequest,
	actor *permissionports.RequestActor,
) (*factoringledgerports.ReserveMovementResult, error) {
	draft, err := s.planRelease(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	return s.recordItemMovement(ctx, req, actor, draft,
		"Reserve released on invoice "+draft.plan.Item.InvoiceNumber)
}

func (s *LedgerService) PlanChargeback(
	ctx context.Context,
	req *factoringledgerports.ReserveMovementRequest,
	actor *permissionports.RequestActor,
) (*factoringledgerports.ReserveMovementPlan, error) {
	draft, err := s.planChargeback(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	return draft.plan, nil
}

func (s *LedgerService) RecordChargeback(
	ctx context.Context,
	req *factoringledgerports.ReserveMovementRequest,
	actor *permissionports.RequestActor,
) (*factoringledgerports.ReserveMovementResult, error) {
	draft, err := s.planChargeback(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	return s.recordItemMovement(ctx, req, actor, draft,
		"Invoice "+draft.plan.Item.InvoiceNumber+" charged back: "+draft.plan.Item.ChargebackReason)
}

func (s *LedgerService) PlanChargebackReversal(
	ctx context.Context,
	req *factoringledgerports.ReserveMovementRequest,
	actor *permissionports.RequestActor,
) (*factoringledgerports.ReserveMovementPlan, error) {
	draft, err := s.planReversal(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	return draft.plan, nil
}

func (s *LedgerService) ReverseChargeback(
	ctx context.Context,
	req *factoringledgerports.ReserveMovementRequest,
	actor *permissionports.RequestActor,
) (*factoringledgerports.ReserveMovementResult, error) {
	draft, err := s.planReversal(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	return s.recordItemMovement(ctx, req, actor, draft,
		"Chargeback of invoice "+draft.plan.Item.InvoiceNumber+" reversed: "+
			strings.TrimSpace(req.Reason))
}

func (s *LedgerService) planRelease(
	ctx context.Context,
	req *factoringledgerports.ReserveMovementRequest,
	actor *permissionports.RequestActor,
) (*itemDraft, error) {
	scope, plan, history, err := s.loadItem(ctx, req, actor, "releasedOn")
	if err != nil {
		return nil, err
	}
	item := plan.Item
	_, _, reserve := item.Actuals()
	cash := reserve
	if req.CashMinor != nil {
		cash = *req.CashMinor
	}
	movement, err := factoring.PlanReleaseMovement(item, history.funding.Treatment, cash)
	if err != nil {
		return nil, err
	}
	if err = item.Settle(req.OccurredOn); err != nil {
		return nil, err
	}
	return s.draftItem(ctx, plan, &movementInput{
		scope:     scope,
		item:      item,
		movement:  movement,
		event:     tenant.JournalSourceEventFactoringReserveReleased,
		key:       releaseKeyPrefix + item.ID.String(),
		date:      req.OccurredOn,
		dateField: "releasedOn",
		memo:      req.Reason,
		remitLine: req.RemittanceLineID,
		actor:     actor,
	})
}

func (s *LedgerService) planChargeback(
	ctx context.Context,
	req *factoringledgerports.ReserveMovementRequest,
	actor *permissionports.RequestActor,
) (*itemDraft, error) {
	scope, plan, history, err := s.loadItem(ctx, req, actor, "chargedBackOn")
	if err != nil {
		return nil, err
	}
	if scope.agreement.Recourse != factoring.RecourseFull {
		return nil, errortypes.NewBusinessError(
			"The agreement is non-recourse: the factor carries the credit risk and cannot charge invoices back",
		)
	}
	item := plan.Item
	cash := factoring.ChargebackRepurchaseMinor(item)
	if req.CashMinor != nil {
		cash = *req.CashMinor
	}
	movement, err := factoring.PlanChargebackMovement(item, history.funding.Treatment, cash)
	if err != nil {
		return nil, err
	}
	if err = item.ChargeBack(req.OccurredOn, req.Reason); err != nil {
		return nil, err
	}
	return s.draftItem(ctx, plan, &movementInput{
		scope:    scope,
		item:     item,
		movement: movement,
		event:    tenant.JournalSourceEventFactoringChargedBack,
		key: chargebackKeyPrefix + item.ID.String() + ":" +
			strconv.Itoa(history.chargebacks+1),
		date:      req.OccurredOn,
		dateField: "chargedBackOn",
		memo:      item.ChargebackReason,
		remitLine: req.RemittanceLineID,
		actor:     actor,
	})
}

func (s *LedgerService) planReversal(
	ctx context.Context,
	req *factoringledgerports.ReserveMovementRequest,
	actor *permissionports.RequestActor,
) (*itemDraft, error) {
	if strings.TrimSpace(req.Reason) == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Say why the chargeback is being reversed",
		)
	}
	if req.CashMinor != nil {
		return nil, errortypes.NewValidationError(
			"cashMinor",
			errortypes.ErrInvalid,
			"A reversal undoes the chargeback exactly; it takes no amount",
		)
	}
	scope, plan, history, err := s.loadItem(ctx, req, actor, "reversedOn")
	if err != nil {
		return nil, err
	}
	if history.open == nil {
		return nil, errortypes.NewBusinessError(
			"Invoice {0} has no chargeback to reverse",
			plan.Item.InvoiceNumber,
		)
	}
	original := history.open.Movement()
	movement := factoring.ReverseMovement(&original)
	item := plan.Item
	if err = item.ReverseChargeback(); err != nil {
		return nil, err
	}
	plan.Reverses = history.open
	return s.draftItem(ctx, plan, &movementInput{
		scope:     scope,
		item:      item,
		movement:  movement,
		event:     tenant.JournalSourceEventFactoringChargebackReversed,
		key:       reversalKeyPrefix + history.open.ID.String(),
		date:      req.OccurredOn,
		dateField: "reversedOn",
		memo:      req.Reason,
		reverses:  history.open.ID,
		remitLine: req.RemittanceLineID,
		actor:     actor,
	})
}

func (s *LedgerService) loadItem(
	ctx context.Context,
	req *factoringledgerports.ReserveMovementRequest,
	actor *permissionports.RequestActor,
	dateField string,
) (*ledgerScope, *factoringledgerports.ReserveMovementPlan, *itemHistory, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, nil, nil, err
	}
	if err := requireExecutor(actor, "Booking a factoring entry"); err != nil {
		return nil, nil, nil, err
	}
	if req.ItemID.IsNil() {
		return nil, nil, nil, errortypes.NewValidationError(
			"itemId",
			errortypes.ErrRequired,
			"Choose the invoice",
		)
	}
	if err := s.requireMovementDate(dateField, req.OccurredOn); err != nil {
		return nil, nil, nil, err
	}
	scope, err := s.loadScope(ctx, req.TenantInfo, req.SubmissionID, req.Version)
	if err != nil {
		return nil, nil, nil, err
	}
	after := cloneSubmission(scope.before)
	item := after.Item(req.ItemID)
	if item == nil {
		return nil, nil, nil, errortypes.NewValidationError(
			"itemId",
			errortypes.ErrInvalid,
			"That invoice is not on this schedule",
		)
	}
	entries, err := s.entries.ListForItem(
		ctx,
		factoringledgerports.ListReserveEntriesForItemRequest{
			TenantInfo:       req.TenantInfo,
			SubmissionItemID: item.ID,
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	history := readHistory(entries)
	if history.funding == nil {
		return nil, nil, nil, errortypes.NewBusinessError(
			"No funding is recorded for invoice {0}; record the factor's funding first",
			item.InvoiceNumber,
		)
	}
	if err = requireAccounts(&scope.accounts, history.funding.Treatment); err != nil {
		return nil, nil, nil, err
	}
	plan := &factoringledgerports.ReserveMovementPlan{
		Before: scope.before,
		After:  after,
		Item:   item,
	}
	return scope, plan, history, nil
}

func (s *LedgerService) draftItem(
	ctx context.Context,
	plan *factoringledgerports.ReserveMovementPlan,
	in *movementInput,
) (*itemDraft, error) {
	if err := s.checkInvoices(ctx, in.tenantInfo(), []*movementInput{in}); err != nil {
		return nil, err
	}
	planned, err := s.planMovement(ctx, in)
	if err != nil {
		return nil, err
	}
	plan.Movement = planned
	return &itemDraft{plan: plan, input: in}, nil
}

func (s *LedgerService) recordItemMovement(
	ctx context.Context,
	req *factoringledgerports.ReserveMovementRequest,
	actor *permissionports.RequestActor,
	draft *itemDraft,
	comment string,
) (*factoringledgerports.ReserveMovementResult, error) {
	var entry *factoring.ReserveEntry
	var updated *factoring.Submission
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		var txErr error
		if entry, txErr = s.postMovement(txCtx, draft.input); txErr != nil {
			return txErr
		}
		updated, txErr = s.submissions.Update(txCtx, draft.plan.After)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	s.finish(ctx, req.TenantInfo, actor, draft.plan.Before, updated,
		comment+" on schedule "+updated.Number,
		[]pulid.ID{draft.plan.Item.InvoiceID},
	)
	return &factoringledgerports.ReserveMovementResult{Submission: updated, Entry: entry}, nil
}

func readHistory(entries []*factoring.ReserveEntry) *itemHistory {
	history := &itemHistory{}
	reversed := make(map[pulid.ID]struct{}, len(entries))
	for _, entry := range entries {
		if entry.Kind == factoring.ReserveEntryChargebackReversal &&
			entry.ReversesEntryID.IsNotNil() {
			reversed[entry.ReversesEntryID] = struct{}{}
		}
	}
	for _, entry := range entries {
		switch entry.Kind {
		case factoring.ReserveEntryFunding:
			if history.funding == nil {
				history.funding = entry
			}
		case factoring.ReserveEntryChargeback:
			history.chargebacks++
			if _, done := reversed[entry.ID]; !done {
				history.open = entry
			}
		case factoring.ReserveEntryRelease, factoring.ReserveEntryChargebackReversal:
		}
	}
	return history
}
