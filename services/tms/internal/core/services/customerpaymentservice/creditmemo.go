package customerpaymentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
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
	if err := s.checkCreditRequest(ctx, req, actor); err != nil {
		return nil, err
	}

	var applied []*customerpayment.CreditMemoApplication
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		plan, txErr := s.planCreditApplication(txCtx, req, actor, s.invoiceRepo.LockForUpdate)
		if txErr != nil {
			return txErr
		}

		for idx, target := range plan.targets {
			updatedTarget, updateErr := s.invoiceRepo.Update(txCtx, target)
			if updateErr != nil {
				return updateErr
			}
			s.logInvoiceAudit(plan.targetsBefore[idx], updatedTarget, actor.UserID)
		}

		updatedMemo, txErr := s.invoiceRepo.Update(txCtx, plan.memo)
		if txErr != nil {
			return txErr
		}
		if txErr = s.repo.CreateCreditMemoApplications(txCtx, plan.applications); txErr != nil {
			return txErr
		}
		for _, app := range plan.applications {
			if txErr = serviceports.EnqueueAccountingSync(
				txCtx,
				s.accountingSync,
				serviceports.CreditApplicationSyncRequest(
					app,
					plan.memo.Number,
					accountingsync.SyncOperationCreate,
					accountingsync.SyncSourceCreditMemoApplied,
				),
			); txErr != nil {
				return txErr
			}
		}
		s.logInvoiceAudit(plan.memoBefore, updatedMemo, actor.UserID)
		applied = plan.applications
		return nil
	})
	if err != nil {
		return nil, err
	}

	return applied, nil
}

func (s *Service) UnapplyCreditMemoApplication(
	ctx context.Context,
	req *serviceports.UnapplyCreditMemoApplicationRequest,
	actor *serviceports.RequestActor,
) (*customerpayment.CreditMemoApplication, error) {
	if err := checkUnapplyRequest(req, actor); err != nil {
		return nil, err
	}

	var updated *customerpayment.CreditMemoApplication
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		plan, txErr := s.planUnapplyCredit(txCtx, req, actor, s.invoiceRepo.LockForUpdate)
		if txErr != nil {
			return txErr
		}

		updatedTarget, txErr := s.invoiceRepo.Update(txCtx, plan.targets[0])
		if txErr != nil {
			return txErr
		}
		updatedMemo, txErr := s.invoiceRepo.Update(txCtx, plan.memo)
		if txErr != nil {
			return txErr
		}
		updated, txErr = s.repo.UpdateCreditMemoApplication(txCtx, plan.applications[0])
		if txErr != nil {
			return txErr
		}
		if txErr = serviceports.EnqueueAccountingSync(
			txCtx,
			s.accountingSync,
			serviceports.CreditApplicationSyncRequest(
				updated,
				plan.memo.Number,
				accountingsync.SyncOperationVoid,
				accountingsync.SyncSourceCreditMemoUnapplied,
			),
		); txErr != nil {
			return txErr
		}

		s.logInvoiceAudit(plan.targetsBefore[0], updatedTarget, actor.UserID)
		s.logInvoiceAudit(plan.memoBefore, updatedMemo, actor.UserID)
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
