package customerpaymentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

// ApplyCreditMemo settles open invoices with a posted credit memo of the same
// customer. Both documents already sit on the ledger, so nothing posts here:
// the credit memo's remaining balance goes down, each invoice's open balance
// goes down, and the application rows say which paid which.
func (s *Service) ApplyCreditMemo(
	ctx context.Context,
	req *serviceports.ApplyCreditMemoRequest,
	actor *serviceports.RequestActor,
) ([]*customerpayment.CreditMemoApplication, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Applying a credit memo requires an authenticated user",
		)
	}
	if multiErr := s.validateCreditApplications(req); multiErr != nil {
		return nil, multiErr
	}

	if _, err := s.validator.fiscalPeriodRepo.GetPeriodByDate(
		ctx,
		repositories.GetPeriodByDateRequest{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
			Date:  req.AccountingDate,
		},
	); err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"accountingDate",
			errortypes.ErrInvalid,
			"Accounting date must fall within a fiscal period",
		)
		return nil, multiErr
	}

	applied := make([]*customerpayment.CreditMemoApplication, 0, len(req.Applications))
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		memo, txErr := s.invoiceRepo.LockForUpdate(txCtx, repositories.GetInvoiceByIDRequest{
			ID:         req.CreditMemoID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if multiErr := validateCreditMemoSource(memo); multiErr != nil {
			return multiErr
		}

		var total int64
		for _, app := range req.Applications {
			total += app.AppliedAmountMinor
		}
		if total > memo.CreditRemainingMinor() {
			return errortypes.NewValidationError(
				"applications",
				errortypes.ErrInvalid,
				"Applications exceed the credit memo's remaining balance by {0} minor units",
				total-memo.CreditRemainingMinor(),
			)
		}

		previousMemo := *memo
		for idx, app := range req.Applications {
			target, lockErr := s.invoiceRepo.LockForUpdate(
				txCtx,
				repositories.GetInvoiceByIDRequest{
					ID:         app.InvoiceID,
					TenantInfo: req.TenantInfo,
				},
			)
			if lockErr != nil {
				return lockErr
			}
			if multiErr := validateCreditTarget(
				memo,
				target,
				app.AppliedAmountMinor,
				idx,
			); multiErr != nil {
				return multiErr
			}

			previousTarget := *target
			target.ApplyPaymentMinor(app.AppliedAmountMinor)
			updatedTarget, updateErr := s.invoiceRepo.Update(txCtx, target)
			if updateErr != nil {
				return updateErr
			}
			memo.ApplyCreditMinor(app.AppliedAmountMinor)

			applied = append(applied, &customerpayment.CreditMemoApplication{
				OrganizationID:      req.TenantInfo.OrgID,
				BusinessUnitID:      req.TenantInfo.BuID,
				CreditMemoInvoiceID: memo.ID,
				InvoiceID:           target.ID,
				AppliedAmountMinor:  app.AppliedAmountMinor,
				AccountingDate:      req.AccountingDate,
				LineNumber:          idx + 1,
				Status:              customerpayment.CreditApplicationStatusApplied,
				CreatedByID:         actor.UserID,
			})
			s.logInvoiceAudit(&previousTarget, updatedTarget, actor.UserID)
		}

		updatedMemo, txErr := s.invoiceRepo.Update(txCtx, memo)
		if txErr != nil {
			return txErr
		}
		if txErr = s.repo.CreateCreditMemoApplications(txCtx, applied); txErr != nil {
			return txErr
		}
		for _, app := range applied {
			if txErr = serviceports.EnqueueAccountingSync(
				txCtx,
				s.accountingSync,
				serviceports.CreditApplicationSyncRequest(
					app,
					memo.Number,
					accountingsync.SyncOperationCreate,
					accountingsync.SyncSourceCreditMemoApplied,
				),
			); txErr != nil {
				return txErr
			}
		}
		s.logInvoiceAudit(&previousMemo, updatedMemo, actor.UserID)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return applied, nil
}

// UnapplyCreditMemoApplication takes one application back, restoring the
// invoice's open balance and the credit memo's remaining balance.
func (s *Service) UnapplyCreditMemoApplication(
	ctx context.Context,
	req *serviceports.UnapplyCreditMemoApplicationRequest,
	actor *serviceports.RequestActor,
) (*customerpayment.CreditMemoApplication, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Unapplying a credit memo requires an authenticated user",
		)
	}
	if req.ApplicationID.IsNil() {
		return nil, errortypes.NewValidationError(
			"applicationId",
			errortypes.ErrRequired,
			"Application is required",
		)
	}

	var updated *customerpayment.CreditMemoApplication
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		application, txErr := s.repo.GetCreditMemoApplicationByID(
			txCtx,
			repositories.GetCreditMemoApplicationRequest{
				ID:         req.ApplicationID,
				TenantInfo: req.TenantInfo,
			},
		)
		if txErr != nil {
			return txErr
		}
		if !application.IsApplied() {
			return errortypes.NewValidationError(
				"applicationId",
				errortypes.ErrInvalidOperation,
				"This credit memo application has already been unapplied",
			)
		}

		memo, txErr := s.invoiceRepo.LockForUpdate(txCtx, repositories.GetInvoiceByIDRequest{
			ID:         application.CreditMemoInvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		target, txErr := s.invoiceRepo.LockForUpdate(txCtx, repositories.GetInvoiceByIDRequest{
			ID:         application.InvoiceID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if target.Status == invoice.StatusVoided {
			return errortypes.NewValidationError(
				"applicationId",
				errortypes.ErrInvalidOperation,
				"The invoice this credit settled has been voided",
			)
		}

		previousTarget := *target
		previousMemo := *memo
		target.RemovePaymentMinor(application.AppliedAmountMinor)
		memo.ReleaseCreditMinor(application.AppliedAmountMinor)
		updatedTarget, txErr := s.invoiceRepo.Update(txCtx, target)
		if txErr != nil {
			return txErr
		}
		updatedMemo, txErr := s.invoiceRepo.Update(txCtx, memo)
		if txErr != nil {
			return txErr
		}

		now := timeutils.NowUnix()
		application.Status = customerpayment.CreditApplicationStatusUnapplied
		application.UnappliedAt = &now
		application.UnappliedByID = actor.UserID
		application.UnappliedReason = strings.TrimSpace(req.Reason)
		updated, txErr = s.repo.UpdateCreditMemoApplication(txCtx, application)
		if txErr != nil {
			return txErr
		}
		if txErr = serviceports.EnqueueAccountingSync(
			txCtx,
			s.accountingSync,
			serviceports.CreditApplicationSyncRequest(
				updated,
				memo.Number,
				accountingsync.SyncOperationVoid,
				accountingsync.SyncSourceCreditMemoUnapplied,
			),
		); txErr != nil {
			return txErr
		}

		s.logInvoiceAudit(&previousTarget, updatedTarget, actor.UserID)
		s.logInvoiceAudit(&previousMemo, updatedMemo, actor.UserID)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) validateCreditApplications(
	req *serviceports.ApplyCreditMemoRequest,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req.CreditMemoID.IsNil() {
		multiErr.Add("creditMemoId", errortypes.ErrRequired, "Credit memo is required")
	}
	if req.AccountingDate <= 0 {
		multiErr.Add("accountingDate", errortypes.ErrRequired, "Accounting date is required")
	}
	if len(req.Applications) == 0 {
		multiErr.Add("applications", errortypes.ErrRequired, "At least one application is required")
	}
	seen := make(map[pulid.ID]struct{}, len(req.Applications))
	for idx, app := range req.Applications {
		if app == nil {
			multiErr.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrRequired, "Application is required")
			continue
		}
		if app.InvoiceID.IsNil() {
			multiErr.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrRequired, "Invoice is required")
		}
		if app.AppliedAmountMinor <= 0 {
			multiErr.WithIndex("applications", idx).
				Add("appliedAmountMinor", errortypes.ErrInvalid, "Applied amount must be greater than zero")
		}
		if _, dup := seen[app.InvoiceID]; dup {
			multiErr.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalid, "Each invoice may appear once")
		}
		seen[app.InvoiceID] = struct{}{}
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func validateCreditMemoSource(memo *invoice.Invoice) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	switch {
	case memo.BillType != billingqueue.BillTypeCreditMemo:
		multiErr.Add("creditMemoId", errortypes.ErrInvalid, "Only a credit memo can be applied")
	case memo.Status != invoice.StatusPosted:
		multiErr.Add(
			"creditMemoId",
			errortypes.ErrInvalidOperation,
			"Only a posted credit memo can be applied",
		)
	case memo.CreditRemainingMinor() <= 0:
		multiErr.Add(
			"creditMemoId",
			errortypes.ErrInvalidOperation,
			"This credit memo has nothing left to apply",
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func validateCreditTarget(
	memo *invoice.Invoice,
	target *invoice.Invoice,
	amountMinor int64,
	idx int,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entry := multiErr.WithIndex("applications", idx)
	switch {
	case target.CustomerID != memo.CustomerID:
		entry.Add(
			"invoiceId",
			errortypes.ErrInvalid,
			"Invoice customer must match the credit memo customer",
		)
	case target.Status != invoice.StatusPosted:
		entry.Add(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Only posted invoices can take a credit",
		)
	case target.BillType != billingqueue.BillTypeInvoice && target.BillType != billingqueue.BillTypeDebitMemo:
		entry.Add(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"A credit memo settles invoices and debit memos only",
		)
	case amountMinor > target.OpenBalanceMinor():
		entry.Add(
			"appliedAmountMinor",
			errortypes.ErrInvalid,
			"Applied amount exceeds the invoice open balance by {0} minor units",
			amountMinor-target.OpenBalanceMinor(),
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

var _ = permission.OpUpdate
