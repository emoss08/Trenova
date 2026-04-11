package manualjournalservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	FiscalRepo    repositories.FiscalPeriodRepository
	GLAccountRepo repositories.GLAccountRepository
}

type Validator struct {
	fiscalRepo    repositories.FiscalPeriodRepository
	glAccountRepo repositories.GLAccountRepository
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{fiscalRepo: p.FiscalRepo, glAccountRepo: p.GLAccountRepo}
}

func (v *Validator) ValidateDraftUpsert(
	ctx context.Context,
	entity *manualjournal.Request,
	accountingControl *tenant.AccountingControl,
) *errortypes.MultiError {
	if accountingControl != nil {
		if strings.TrimSpace(entity.CurrencyCode) == "" {
			entity.CurrencyCode = accountingControl.FunctionalCurrencyCode
		}

		if accountingControl.CurrencyMode == tenant.CurrencyModeSingleCurrency && entity.CurrencyCode != accountingControl.FunctionalCurrencyCode {
			multiErr := errortypes.NewMultiError()
			multiErr.Add("currencyCode", errortypes.ErrInvalid, "Manual journal currency must match the tenant functional currency")
			return multiErr
		}
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	period, err := v.fiscalRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
		Date:  entity.AccountingDate,
	})
	if err != nil {
		multiErr.Add("accountingDate", errortypes.ErrInvalid, "Accounting date must fall within a fiscal period")
		return multiErr
	}

	entity.RequestedFiscalYearID = period.FiscalYearID
	entity.RequestedFiscalPeriodID = period.ID

	if accountingControl != nil && accountingControl.ManualJournalEntryPolicy == tenant.ManualJournalEntryPolicyAdjustmentOnly && period.PeriodType != fiscalperiod.PeriodTypeAdjusting {
		multiErr.Add("accountingDate", errortypes.ErrInvalid, "Manual journals are restricted to adjusting periods by accounting policy")
	}

	if len(entity.Lines) == 0 {
		return multiErrOrNil(multiErr)
	}

	accountIDs := uniqueAccountIDs(entity.Lines)
	accounts, glErr := v.glAccountRepo.GetByIDs(ctx, repositories.GetGLAccountsByIDsRequest{
		TenantInfo:   paginationFromEntity(entity),
		GLAccountIDs: accountIDs,
	})
	if glErr != nil {
		multiErr.Add("lines", errortypes.ErrInvalid, "Failed to validate GL accounts for manual journal lines")
		return multiErr
	}

	accountMap := make(map[pulid.ID]struct{}, len(accounts))
	for _, account := range accounts {
		accountMap[account.ID] = struct{}{}
		if account.Status != domaintypes.StatusActive {
			multiErr.Add("lines", errortypes.ErrInvalid, "Manual journal lines require active GL accounts")
		}
		if !account.AllowManualJE {
			multiErr.Add("lines", errortypes.ErrInvalid, "Selected GL account does not allow manual journal entries")
		}
	}

	for idx, line := range entity.Lines {
		if line == nil {
			continue
		}
		if _, ok := accountMap[line.GLAccountID]; !ok {
			multiErr.WithIndex("lines", idx).Add("glAccountId", errortypes.ErrInvalid, "GL account was not found for this tenant")
		}
	}

	return multiErrOrNil(multiErr)
}

func (v *Validator) ValidateSubmit(entity *manualjournal.Request, accountingControl *tenant.AccountingControl) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if !entity.Status.CanSubmit() {
		multiErr.Add("status", errortypes.ErrInvalid, "Only draft manual journals can be submitted")
	}
	if accountingControl != nil && accountingControl.ManualJournalEntryPolicy == tenant.ManualJournalEntryPolicyDisallow {
		multiErr.Add("status", errortypes.ErrInvalid, "Manual journal entries are disabled by accounting policy")
	}
	if len(entity.Lines) < 2 {
		multiErr.Add("lines", errortypes.ErrInvalid, "Manual journals require at least two lines before submission")
	}
	if !entity.IsBalanced() {
		multiErr.Add("lines", errortypes.ErrInvalid, "Manual journal must be balanced before submission")
	}
	return multiErrOrNil(multiErr)
}

func (v *Validator) ValidateApprove(entity *manualjournal.Request) *errortypes.MultiError {
	if entity.Status.CanApprove() {
		return nil
	}
	me := errortypes.NewMultiError()
	me.Add("status", errortypes.ErrInvalid, "Only pending manual journals can be approved")
	return me
}

func (v *Validator) ValidateReject(reason string, entity *manualjournal.Request) *errortypes.MultiError {
	me := v.ValidateApprove(entity)
	if strings.TrimSpace(reason) == "" {
		if me == nil {
			me = errortypes.NewMultiError()
		}
		me.Add("reason", errortypes.ErrRequired, "Rejection reason is required")
	}
	return multiErrOrNil(me)
}

func (v *Validator) ValidatePost(entity *manualjournal.Request) *errortypes.MultiError {
	if entity.Status == manualjournal.StatusApproved {
		return nil
	}
	me := errortypes.NewMultiError()
	me.Add("status", errortypes.ErrInvalid, "Only approved manual journals can be posted")
	return me
}

func (v *Validator) ValidateCancel(entity *manualjournal.Request, reason string) *errortypes.MultiError {
	me := errortypes.NewMultiError()
	if !entity.Status.CanCancel() {
		me.Add("status", errortypes.ErrInvalid, "Manual journal cannot be cancelled from its current status")
	}
	if strings.TrimSpace(reason) == "" {
		me.Add("reason", errortypes.ErrRequired, "Cancel reason is required")
	}
	return multiErrOrNil(me)
}

func paginationFromEntity(entity *manualjournal.Request) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

func uniqueAccountIDs(lines []*manualjournal.Line) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(lines))
	result := make([]pulid.ID, 0, len(lines))
	for _, line := range lines {
		if line == nil || line.GLAccountID.IsNil() {
			continue
		}
		if _, ok := seen[line.GLAccountID]; ok {
			continue
		}
		seen[line.GLAccountID] = struct{}{}
		result = append(result, line.GLAccountID)
	}
	return result
}

func multiErrOrNil(me *errortypes.MultiError) *errortypes.MultiError {
	if me == nil || !me.HasErrors() {
		return nil
	}
	return me
}
