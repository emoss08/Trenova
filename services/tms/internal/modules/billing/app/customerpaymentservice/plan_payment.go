package customerpaymentservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type paymentChangePlan struct {
	control      *tenant.AccountingControl
	payment      *customerpayment.Payment
	applications []*customerpayment.Application
	invoices     []*invoice.Invoice
	period       *fiscalperiod.FiscalPeriod
}

type paymentJournal struct {
	description string
	lines       []journalentryports.JournalPostingLine
	total       int64
}

func requirePaymentActor(actor *permissionports.RequestActor, action string) error {
	if actor == nil || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError(
			"Customer payment " + action + " requires an authenticated user",
		)
	}

	return nil
}

func requireRequest(missing bool) error {
	if !missing {
		return nil
	}

	return errortypes.NewValidationError("request", errortypes.ErrRequired, "Request is required")
}

func (s *Service) loadPaymentAndControl(
	ctx context.Context,
	paymentID pulid.ID,
	tenantInfo invoiceports.GetInvoiceByIDRequest,
) (*customerpayment.Payment, *tenant.AccountingControl, error) {
	payment, err := s.repo.GetByID(
		ctx,
		customerpaymentports.GetCustomerPaymentByIDRequest{
			ID:         paymentID,
			TenantInfo: tenantInfo.TenantInfo,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	control, err := s.accountingRepo.GetByOrgID(ctx, tenantInfo.TenantInfo.OrgID)
	if err != nil {
		return nil, nil, err
	}

	return payment, control, nil
}

func (s *Service) planApplyUnapplied(
	ctx context.Context,
	req *customerpaymentports.ApplyCustomerPaymentRequest,
	actor *permissionports.RequestActor,
) (*paymentChangePlan, error) {
	if err := requireRequest(req == nil); err != nil {
		return nil, err
	}
	if err := requirePaymentActor(actor, "application"); err != nil {
		return nil, err
	}

	scope := invoiceports.GetInvoiceByIDRequest{TenantInfo: req.TenantInfo}
	payment, control, err := s.loadPaymentAndControl(ctx, req.PaymentID, scope)
	if err != nil {
		return nil, err
	}
	applications := mapApplications(req.Applications)
	invoices, period, me := s.validator.ValidateApplyUnapplied(
		ctx,
		payment,
		req.AccountingDate,
		applications,
		scope,
		control,
	)
	if me != nil {
		return nil, me
	}

	return &paymentChangePlan{
		control:      control,
		payment:      payment,
		applications: applications,
		invoices:     invoices,
		period:       period,
	}, nil
}

func (s *Service) planReverse(
	ctx context.Context,
	req *customerpaymentports.ReverseCustomerPaymentRequest,
	actor *permissionports.RequestActor,
) (*paymentChangePlan, error) {
	if err := requireRequest(req == nil); err != nil {
		return nil, err
	}
	if err := requirePaymentActor(actor, "reversal"); err != nil {
		return nil, err
	}

	scope := invoiceports.GetInvoiceByIDRequest{TenantInfo: req.TenantInfo}
	payment, control, err := s.loadPaymentAndControl(ctx, req.PaymentID, scope)
	if err != nil {
		return nil, err
	}
	invoices, period, me := s.validator.ValidateReverse(
		ctx,
		payment,
		req.AccountingDate,
		control,
		scope,
	)
	if me != nil {
		return nil, me
	}

	return &paymentChangePlan{
		control:  control,
		payment:  payment,
		invoices: invoices,
		period:   period,
	}, nil
}

func cashRecognized(control *tenant.AccountingControl) bool {
	return control.AccountingBasis == tenant.AccountingBasisCash ||
		control.RevenueRecognitionPolicy == tenant.RevenueRecognitionOnCashReceipt
}

func applicationJournal(
	control *tenant.AccountingControl,
	payment *customerpayment.Payment,
	appliedDelta int64,
	applications []*customerpayment.Application,
) paymentJournal {
	description := fmt.Sprintf("Customer payment application %s", payment.DocumentLabel())
	creditAccountID := control.DefaultARAccountID
	if cashRecognized(control) {
		creditAccountID = control.DefaultRevenueAccountID
		description = fmt.Sprintf(
			"Customer payment revenue recognition %s",
			payment.DocumentLabel(),
		)
	}

	lines := make([]journalentryports.JournalPostingLine, 0, len(applications)*3+1)
	lineNumber := int16(1)
	if appliedDelta > 0 {
		lines = append(lines, journalentryports.JournalPostingLine{
			ID:          pulid.MustNew("jel_"),
			GLAccountID: control.DefaultUnappliedCashAccountID,
			LineNumber:  lineNumber,
			Description: description,
			DebitAmount: appliedDelta,
			NetAmount:   appliedDelta,
			CustomerID:  payment.CustomerID,
		})
		lineNumber++
	}

	var total int64
	for _, app := range applications {
		if app == nil {
			continue
		}
		total += app.SettledAmountMinor()
		if app.AppliedAmountMinor > 0 {
			lines = append(lines, journalentryports.JournalPostingLine{
				ID:           pulid.MustNew("jel_"),
				GLAccountID:  creditAccountID,
				LineNumber:   lineNumber,
				Description:  description,
				CreditAmount: app.AppliedAmountMinor,
				NetAmount:    -app.AppliedAmountMinor,
				CustomerID:   payment.CustomerID,
			})
			lineNumber++
		}
		if app.ShortPayAmountMinor > 0 {
			lines = append(lines,
				journalentryports.JournalPostingLine{
					ID:           pulid.MustNew("jel_"),
					GLAccountID:  control.DefaultARAccountID,
					LineNumber:   lineNumber,
					Description:  description,
					CreditAmount: app.ShortPayAmountMinor,
					NetAmount:    -app.ShortPayAmountMinor,
					CustomerID:   payment.CustomerID,
				},
				journalentryports.JournalPostingLine{
					ID:          pulid.MustNew("jel_"),
					GLAccountID: control.DefaultWriteOffAccountID,
					LineNumber:  lineNumber + 1,
					Description: description,
					DebitAmount: app.ShortPayAmountMinor,
					NetAmount:   app.ShortPayAmountMinor,
					CustomerID:  payment.CustomerID,
				},
			)
			lineNumber += 2
		}
	}

	return paymentJournal{description: description, lines: lines, total: total}
}

func reversalJournal(
	control *tenant.AccountingControl,
	payment *customerpayment.Payment,
) paymentJournal {
	description := fmt.Sprintf("Customer payment reversal %s", payment.DocumentLabel())
	debitAccountID := control.DefaultARAccountID
	if cashRecognized(control) {
		debitAccountID = control.DefaultRevenueAccountID
		description = fmt.Sprintf("Customer cash receipt reversal %s", payment.DocumentLabel())
	}

	lines := make([]journalentryports.JournalPostingLine, 0, 5)
	lineNumber := int16(1)
	shortPay := payment.ShortPayAmountMinor()
	if payment.AppliedAmountMinor > 0 {
		lines = append(lines, journalentryports.JournalPostingLine{
			ID:          pulid.MustNew("jel_"),
			GLAccountID: debitAccountID,
			LineNumber:  lineNumber,
			Description: description,
			DebitAmount: payment.AppliedAmountMinor,
			NetAmount:   payment.AppliedAmountMinor,
			CustomerID:  payment.CustomerID,
		})
		lineNumber++
	}
	if payment.UnappliedAmountMinor > 0 {
		lines = append(lines, journalentryports.JournalPostingLine{
			ID:          pulid.MustNew("jel_"),
			GLAccountID: control.DefaultUnappliedCashAccountID,
			LineNumber:  lineNumber,
			Description: description,
			DebitAmount: payment.UnappliedAmountMinor,
			NetAmount:   payment.UnappliedAmountMinor,
			CustomerID:  payment.CustomerID,
		})
		lineNumber++
	}
	lines = append(lines, journalentryports.JournalPostingLine{
		ID:           pulid.MustNew("jel_"),
		GLAccountID:  control.DefaultCashAccountID,
		LineNumber:   lineNumber,
		Description:  description,
		CreditAmount: payment.AmountMinor,
		NetAmount:    -payment.AmountMinor,
		CustomerID:   payment.CustomerID,
	})
	lineNumber++
	// The short pay was written off against the receivable when the payment
	// posted; reversing the payment puts that receivable back.
	if shortPay > 0 {
		lines = append(lines,
			journalentryports.JournalPostingLine{
				ID:          pulid.MustNew("jel_"),
				GLAccountID: control.DefaultARAccountID,
				LineNumber:  lineNumber,
				Description: description,
				DebitAmount: shortPay,
				NetAmount:   shortPay,
				CustomerID:  payment.CustomerID,
			},
			journalentryports.JournalPostingLine{
				ID:           pulid.MustNew("jel_"),
				GLAccountID:  control.DefaultWriteOffAccountID,
				LineNumber:   lineNumber + 1,
				Description:  description,
				CreditAmount: shortPay,
				NetAmount:    -shortPay,
				CustomerID:   payment.CustomerID,
			},
		)
	}

	return paymentJournal{
		description: description,
		lines:       lines,
		total:       payment.AmountMinor + shortPay,
	}
}

func (j paymentJournal) preview(
	control *tenant.AccountingControl,
	accountingDate int64,
	period *fiscalperiod.FiscalPeriod,
	userID pulid.ID,
) *kernelports.JournalPreview {
	entryStatus, _, _, _, requiresApproval, _, _, _ := paymentPostingWorkflow(
		control,
		userID,
		timeutils.NowUnix(),
	)
	journal := &kernelports.JournalPreview{
		AccountingDate:   accountingDate,
		EntryStatus:      entryStatus,
		RequiresApproval: requiresApproval,
		Lines:            make([]kernelports.JournalLinePreview, 0, len(j.lines)),
	}
	if period != nil {
		journal.FiscalPeriodID = period.ID
	}
	for idx := range j.lines {
		line := &j.lines[idx]
		journal.Lines = append(journal.Lines, kernelports.JournalLinePreview{
			GLAccountID: line.GLAccountID,
			Description: line.Description,
			DebitMinor:  line.DebitAmount,
			CreditMinor: line.CreditAmount,
		})
	}

	return journal
}

func markReversed(
	payment *customerpayment.Payment,
	req *customerpaymentports.ReverseCustomerPaymentRequest,
	userID pulid.ID,
	now int64,
) {
	payment.Status = customerpayment.StatusReversed
	payment.ReversedByID = userID
	payment.ReversedAt = &now
	payment.ReversalReason = req.Reason
	payment.UpdatedByID = userID
}

func (s *Service) PreviewApplyUnapplied(
	ctx context.Context,
	req *customerpaymentports.ApplyCustomerPaymentRequest,
	actor *permissionports.RequestActor,
) (*customerpaymentports.CustomerPaymentChangePreview, error) {
	plan, err := s.planApplyUnapplied(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	after := *plan.payment
	after.Applications = append(slices.Clone(plan.payment.Applications), plan.applications...)
	after.SyncAmounts()

	invoicesAfter := cloneInvoices(plan.invoices)
	for idx, inv := range invoicesAfter {
		if inv != nil {
			applyPostedApplication(inv, plan.applications[idx])
		}
	}

	journal := applicationJournal(
		plan.control,
		&after,
		after.AppliedAmountMinor-plan.payment.AppliedAmountMinor,
		plan.applications,
	)

	return &customerpaymentports.CustomerPaymentChangePreview{
		PaymentBefore:  plan.payment,
		PaymentAfter:   &after,
		InvoicesBefore: plan.invoices,
		InvoicesAfter:  invoicesAfter,
		Journal: journal.preview(
			plan.control,
			req.AccountingDate,
			plan.period,
			actor.UserID,
		),
	}, nil
}

func (s *Service) PreviewReverse(
	ctx context.Context,
	req *customerpaymentports.ReverseCustomerPaymentRequest,
	actor *permissionports.RequestActor,
) (*customerpaymentports.CustomerPaymentChangePreview, error) {
	plan, err := s.planReverse(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	after := *plan.payment
	markReversed(&after, req, actor.UserID, timeutils.NowUnix())

	invoicesAfter := cloneInvoices(plan.invoices)
	for idx, app := range plan.payment.Applications {
		if app == nil || idx >= len(invoicesAfter) || invoicesAfter[idx] == nil {
			continue
		}
		invoicesAfter[idx].RemovePaymentMinor(app.SettledAmountMinor())
	}

	journal := reversalJournal(plan.control, plan.payment)

	return &customerpaymentports.CustomerPaymentChangePreview{
		PaymentBefore:  plan.payment,
		PaymentAfter:   &after,
		InvoicesBefore: plan.invoices,
		InvoicesAfter:  invoicesAfter,
		Journal: journal.preview(
			plan.control,
			req.AccountingDate,
			plan.period,
			actor.UserID,
		),
	}, nil
}
