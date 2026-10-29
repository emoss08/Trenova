package factoringservice

import (
	"context"
	"errors"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const fundingKeyPrefix = "factoring-funded:"

type fundingDraft struct {
	plan   *factoringledgerports.FundingPlan
	inputs []*movementInput
}

func (s *LedgerService) PlanFunding(
	ctx context.Context,
	req *factoringledgerports.RecordFundingRequest,
	actor *permissionports.RequestActor,
) (*factoringledgerports.FundingPlan, error) {
	draft, err := s.planFunding(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	return draft.plan, nil
}

func (s *LedgerService) RecordFunding(
	ctx context.Context,
	req *factoringledgerports.RecordFundingRequest,
	actor *permissionports.RequestActor,
) (*factoringledgerports.FundingResult, error) {
	draft, err := s.planFunding(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	entries := make([]*factoringledger.ReserveEntry, 0, len(draft.inputs))
	var updated *factoringsubmission.Submission
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		for _, in := range draft.inputs {
			entry, postErr := s.postMovement(txCtx, in)
			if postErr != nil {
				return postErr
			}
			if entry != nil {
				entries = append(entries, entry)
			}
		}
		var updateErr error
		updated, updateErr = s.submissions.Update(txCtx, draft.plan.After)
		return updateErr
	})
	if err != nil {
		return nil, err
	}

	invoiceIDs := make([]pulid.ID, 0, len(req.Items))
	for _, input := range req.Items {
		if item := draft.plan.After.Item(input.ItemID); item != nil {
			invoiceIDs = append(invoiceIDs, item.InvoiceID)
		}
	}
	s.finish(ctx, req.TenantInfo, actor, draft.plan.Before, updated,
		"Funding recorded on schedule "+updated.Number+": "+
			strconv.Itoa(draft.plan.FundedCount)+" funded, "+
			strconv.Itoa(draft.plan.RejectCount)+" rejected",
		invoiceIDs,
	)
	if draft.plan.FundedCount > 0 {
		s.publishEvent(ctx, req.TenantInfo, agenttrigger.EventFactoringSubmissionFunded, updated.ID,
			[]pulid.ID{updated.FactoringCompanyID},
			"Schedule "+updated.Number+" funded: "+strconv.Itoa(draft.plan.FundedCount)+
				" invoices funded, "+strconv.Itoa(draft.plan.RejectCount)+" rejected",
		)
	}
	return &factoringledgerports.FundingResult{Submission: updated, Entries: entries}, nil
}

func (s *LedgerService) planFunding(
	ctx context.Context,
	req *factoringledgerports.RecordFundingRequest,
	actor *permissionports.RequestActor,
) (*fundingDraft, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireExecutor(actor, "Recording a factor's funding"); err != nil {
		return nil, err
	}
	if err := s.validateFundingRequest(req); err != nil {
		return nil, err
	}
	scope, err := s.loadScope(ctx, req.TenantInfo, req.SubmissionID, req.Version)
	if err != nil {
		return nil, err
	}
	if !scope.before.CanFund() {
		return nil, errortypes.NewBusinessError(
			"Schedule {0} is not waiting on the factor's funding",
			scope.before.Number,
		)
	}

	after := cloneSubmission(scope.before)
	plan := &factoringledgerports.FundingPlan{Before: scope.before, After: after}
	funded, err := applyFundingOutcomes(after, req, plan)
	if err != nil {
		return nil, err
	}
	after.Resolve(req.FundedOn)

	draft := &fundingDraft{plan: plan, inputs: make([]*movementInput, 0, len(funded))}
	if len(funded) == 0 {
		return draft, nil
	}
	treatment := scope.agreement.AccountingTreatment
	if err = requireAccounts(&scope.accounts, treatment); err != nil {
		return nil, err
	}
	remitLines := make(map[pulid.ID]pulid.ID, len(req.Items))
	for idx := range req.Items {
		remitLines[req.Items[idx].ItemID] = req.Items[idx].RemittanceLineID
	}
	for _, item := range funded {
		movement, planErr := factoringledger.PlanFundingMovement(item, treatment)
		if planErr != nil {
			return nil, planErr
		}
		draft.inputs = append(draft.inputs, &movementInput{
			scope:     scope,
			item:      item,
			movement:  movement,
			event:     accountingcontrol.JournalSourceEventFactoringFunded,
			key:       fundingKeyPrefix + item.ID.String(),
			date:      req.FundedOn,
			dateField: "fundedOn",
			memo:      req.Memo,
			remitLine: remitLines[item.ID],
			actor:     actor,
		})
	}
	if err = s.checkInvoices(ctx, req.TenantInfo, draft.inputs); err != nil {
		return nil, err
	}
	plan.Movements = make([]*factoringledgerports.PlannedMovement, 0, len(draft.inputs))
	for _, in := range draft.inputs {
		planned, planErr := s.planMovement(ctx, in)
		if planErr != nil {
			return nil, planErr
		}
		plan.Movements = append(plan.Movements, planned)
	}
	return draft, nil
}

func (s *LedgerService) validateFundingRequest(
	req *factoringledgerports.RecordFundingRequest,
) error {
	if err := s.requireMovementDate("fundedOn", req.FundedOn); err != nil {
		return err
	}
	if len(req.Items) == 0 {
		return errortypes.NewValidationError(
			"items",
			errortypes.ErrRequired,
			"Say which invoices the factor funded or rejected",
		)
	}
	multiErr := errortypes.NewMultiError()
	seen := make(map[pulid.ID]struct{}, len(req.Items))
	for idx := range req.Items {
		input := &req.Items[idx]
		field := "items[" + strconv.Itoa(idx) + "]"
		if input.ItemID.IsNil() {
			multiErr.Add(field+".itemId", errortypes.ErrRequired, "Choose the invoice")
			continue
		}
		if _, dup := seen[input.ItemID]; dup {
			multiErr.Add(field+".itemId", errortypes.ErrDuplicate, "This invoice is listed twice")
			continue
		}
		seen[input.ItemID] = struct{}{}
		if !input.Outcome.IsValid() {
			multiErr.Add(field+".outcome", errortypes.ErrInvalid, "Choose funded or rejected")
		}
		if input.Outcome == factoringledger.FundingOutcomeRejected &&
			(input.AdvanceMinor != nil || input.FeeMinor != nil) {
			multiErr.Add(
				field+".advanceMinor",
				errortypes.ErrInvalid,
				"A rejected invoice has no advance or fee",
			)
		}
	}
	return validationOf(multiErr)
}

func applyFundingOutcomes(
	after *factoringsubmission.Submission,
	req *factoringledgerports.RecordFundingRequest,
	plan *factoringledgerports.FundingPlan,
) ([]*factoringsubmission.SubmissionItem, error) {
	funded := make([]*factoringsubmission.SubmissionItem, 0, len(req.Items))
	multiErr := errortypes.NewMultiError()
	for idx := range req.Items {
		input := &req.Items[idx]
		field := "items[" + strconv.Itoa(idx) + "]"
		item := after.Item(input.ItemID)
		if item == nil {
			multiErr.Add(
				field+".itemId",
				errortypes.ErrInvalid,
				"That invoice is not on this schedule",
			)
			continue
		}
		var err error
		switch input.Outcome {
		case factoringledger.FundingOutcomeFunded:
			funding := item.ExpectedFunding()
			if input.AdvanceMinor != nil {
				funding.AdvanceMinor = *input.AdvanceMinor
			}
			if input.FeeMinor != nil {
				funding.FeeMinor = *input.FeeMinor
			}
			if err = item.Fund(req.FundedOn, funding); err == nil {
				funded = append(funded, item)
				advance, fee, reserve := item.Actuals()
				plan.FundedCount++
				plan.AdvanceMinor += advance
				plan.FeeMinor += fee
				plan.ReserveMinor += reserve
			}
		case factoringledger.FundingOutcomeRejected:
			if err = item.Reject(input.Reason); err == nil {
				plan.RejectCount++
			}
		}
		if err != nil {
			if addErr := addItemError(multiErr, field, err); addErr != nil {
				return nil, addErr
			}
		}
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return funded, nil
}

func addItemError(multiErr *errortypes.MultiError, field string, err error) error {
	var validationErr *errortypes.Error
	if errors.As(err, &validationErr) {
		multiErr.Add(field, validationErr.Code, validationErr.Message, validationErr.Args...)
		return nil
	}
	var businessErr *errortypes.BusinessError
	if errors.As(err, &businessErr) {
		multiErr.Add(
			field,
			errortypes.ErrInvalidOperation,
			businessErr.Message,
			businessErr.Args...)
		return nil
	}
	return err
}
