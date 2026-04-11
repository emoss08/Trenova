package journalreversalservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	FiscalRepo repositories.FiscalPeriodRepository
}

type Validator struct {
	fiscalRepo repositories.FiscalPeriodRepository
}

func NewValidator(p ValidatorParams) *Validator { return &Validator{fiscalRepo: p.FiscalRepo} }

func (v *Validator) ValidateCreate(ctx context.Context, entry *journalentry.Entry, reqAccountingDate int64, reasonCode, reasonText string) *errortypes.MultiError {
	me := errortypes.NewMultiError()
	if entry == nil {
		me.Add("originalJournalEntryId", errortypes.ErrRequired, "Original journal entry is required")
		return me
	}
	if !entry.IsPosted || entry.Status != "Posted" {
		me.Add("originalJournalEntryId", errortypes.ErrInvalidOperation, "Only posted journal entries can be reversed")
	}
	if entry.IsReversal {
		me.Add("originalJournalEntryId", errortypes.ErrInvalidOperation, "Reversal journal entries cannot be reversed again")
	}
	if !entry.ReversedByID.IsNil() {
		me.Add("originalJournalEntryId", errortypes.ErrInvalidOperation, "Journal entry has already been reversed")
	}
	if reqAccountingDate == 0 {
		me.Add("requestedAccountingDate", errortypes.ErrRequired, "Requested accounting date is required")
	}
	if strings.TrimSpace(reasonCode) == "" {
		me.Add("reasonCode", errortypes.ErrRequired, "Reason code is required")
	}
	if strings.TrimSpace(reasonText) == "" {
		me.Add("reasonText", errortypes.ErrRequired, "Reason text is required")
	}
	return multiErrOrNil(me)
}

func (v *Validator) ResolvePostingPeriod(ctx context.Context, orgID, buID pulid.ID, accountingDate int64, control *tenant.AccountingControl) (*fiscalperiod.FiscalPeriod, int64, *errortypes.MultiError) {
	me := errortypes.NewMultiError()
	period, err := v.fiscalRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: accountingDate})
	if err != nil {
		me.Add("requestedAccountingDate", errortypes.ErrInvalid, "Accounting date must resolve to a fiscal period")
		return nil, 0, me
	}
	if period.Status == fiscalperiod.StatusClosed || period.Status == fiscalperiod.StatusPermanentlyClosed {
		if control == nil || control.JournalReversalPolicy != tenant.JournalReversalPolicyNextOpenPeriod {
			me.Add("requestedAccountingDate", errortypes.ErrInvalidOperation, "Closed periods require next-open-period reversal policy")
			return nil, 0, me
		}
		periods, listErr := v.fiscalRepo.ListByFiscalYearID(ctx, repositories.ListByFiscalYearIDRequest{FiscalYearID: period.FiscalYearID, OrgID: orgID, BuID: buID})
		if listErr != nil {
			me.Add("requestedAccountingDate", errortypes.ErrSystemError, "Failed to resolve next open fiscal period")
			return nil, 0, me
		}
		for _, candidate := range periods {
			if candidate == nil || candidate.PeriodNumber <= period.PeriodNumber {
				continue
			}
			if candidate.Status == fiscalperiod.StatusOpen || candidate.Status == fiscalperiod.StatusLocked {
				return candidate, candidate.StartDate, nil
			}
		}
		me.Add("requestedAccountingDate", errortypes.ErrInvalidOperation, "No next open fiscal period is available for reversal posting")
		return nil, 0, me
	}
	return period, accountingDate, nil
}

func (v *Validator) ValidateApprove(entity *journalreversal.Reversal) *errortypes.MultiError {
	if entity != nil && entity.Status.CanApprove() {
		return nil
	}
	me := errortypes.NewMultiError()
	me.Add("status", errortypes.ErrInvalid, "Only requested or pending reversals can be approved")
	return me
}
func (v *Validator) ValidateReject(entity *journalreversal.Reversal, reason string) *errortypes.MultiError {
	me := v.ValidateApprove(entity)
	if strings.TrimSpace(reason) == "" {
		if me == nil {
			me = errortypes.NewMultiError()
		}
		me.Add("reason", errortypes.ErrRequired, "Rejection reason is required")
	}
	return multiErrOrNil(me)
}
func (v *Validator) ValidateCancel(entity *journalreversal.Reversal, reason string) *errortypes.MultiError {
	me := errortypes.NewMultiError()
	if entity == nil || !entity.Status.CanCancel() {
		me.Add("status", errortypes.ErrInvalid, "Journal reversal cannot be cancelled from its current status")
	}
	if strings.TrimSpace(reason) == "" {
		me.Add("reason", errortypes.ErrRequired, "Cancel reason is required")
	}
	return multiErrOrNil(me)
}
func (v *Validator) ValidatePost(entity *journalreversal.Reversal) *errortypes.MultiError {
	if entity != nil && entity.Status.CanPost() {
		return nil
	}
	me := errortypes.NewMultiError()
	me.Add("status", errortypes.ErrInvalid, "Only approved journal reversals can be posted")
	return me
}

func multiErrOrNil(me *errortypes.MultiError) *errortypes.MultiError {
	if me == nil || !me.HasErrors() {
		return nil
	}
	return me
}
