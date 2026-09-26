package customerpaymentservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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
	lines       []repositories.JournalPostingLine
	total       int64
}

func requirePaymentActor(actor *serviceports.RequestActor, action string) error {
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
	tenantInfo repositories.GetInvoiceByIDRequest,
) (*customerpayment.Payment, *tenant.AccountingControl, error) {
	payment, err := s.repo.GetByID(
		ctx,
		repositories.GetCustomerPaymentByIDRequest{ID: paymentID, TenantInfo: tenantInfo.TenantInfo},
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
	req *serviceports.ApplyCustomerPaymentRequest,
	actor *serviceports.RequestActor,
) (*paymentChangePlan, error) {
	if err := requireRequest(req == nil); err != nil {
		return nil, err
	}
	if err := requirePaymentActor(actor, "application"); err != nil {
		return nil, err
	}

	scope := repositories.GetInvoiceByIDRequest{TenantInfo: req.TenantInfo}
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
	req *serviceports.ReverseCustomerPaymentRequest,
	actor *serviceports.RequestActor,
) (*paymentChangePlan, error) {
	if err := requireRequest(req == nil); err != nil {
		return nil, err
	}
	if err := requirePaymentActor(actor, "reversal"); err != nil {
		return nil, err
	}

	scope := repositories.GetInvoiceByIDRequest{TenantInfo: req.TenantInfo}
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

	lines := make([]repositories.JournalPostingLine, 0, len(applications)*3+1)
	lineNumber := int16(1)
	lines = append(lines, repositories.JournalPostingLine{
		ID:          pulid.MustNew("jel_"),
		GLAccountID: control.DefaultUnappliedCashAccountID,
		LineNumber:  lineNumber,
		Description: description,
		DebitAmount: appliedDelta,
		NetAmount:   appliedDelta,
		CustomerID:  payment.CustomerID,
	})
	lineNumber++

	var total int64
	for _, app := range applications {
		if app == nil {
			continue
		}
		total += app.AppliedAmountMinor
		if app.AppliedAmountMinor > 0 {
			lines = append(lines, repositories.JournalPostingLine{
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
				repositories.JournalPostingLine{
					ID:           pulid.MustNew("jel_"),
					GLAccountID:  control.DefaultARAccountID,
					LineNumber:   lineNumber,
					Description:  description,
					CreditAmount: app.ShortPayAmountMinor,
					NetAmount:    -app.ShortPayAmountMinor,
					CustomerID:   payment.CustomerID,
				},
				repositories.JournalPostingLine{
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

	lines := make([]repositories.JournalPostingLine, 0, 3)
	lineNumber := int16(1)
	if payment.AppliedAmountMinor > 0 {
		lines = append(lines, repositories.JournalPostingLine{
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
		lines = append(lines, repositories.JournalPostingLine{
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
	lines = append(lines, repositories.JournalPostingLine{
		ID:           pulid.MustNew("jel_"),
		GLAccountID:  control.DefaultCashAccountID,
		LineNumber:   lineNumber,
		Description:  description,
		CreditAmount: payment.AmountMinor,
		NetAmount:    -payment.AmountMinor,
		CustomerID:   payment.CustomerID,
	})

	return paymentJournal{description: description, lines: lines, total: payment.AmountMinor}
}

func (j paymentJournal) preview(
	control *tenant.AccountingControl,
	accountingDate int64,
	period *fiscalperiod.FiscalPeriod,
	userID pulid.ID,
) *serviceports.JournalPreview {
	entryStatus, _, _, _, requiresApproval, _, _, _ := paymentPostingWorkflow(
		control,
		userID,
		timeutils.NowUnix(),
	)
	journal := &serviceports.JournalPreview{
		AccountingDate:   accountingDate,
		EntryStatus:      entryStatus,
		RequiresApproval: requiresApproval,
		Lines:            make([]serviceports.JournalLinePreview, 0, len(j.lines)),
	}
	if period != nil {
		journal.FiscalPeriodID = period.ID
	}
	for idx := range j.lines {
		line := &j.lines[idx]
		journal.Lines = append(journal.Lines, serviceports.JournalLinePreview{
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
	req *serviceports.ReverseCustomerPaymentRequest,
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
	req *serviceports.ApplyCustomerPaymentRequest,
	actor *serviceports.RequestActor,
) (*serviceports.CustomerPaymentChangePreview, error) {
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

	return &serviceports.CustomerPaymentChangePreview{
		PaymentBefore:  plan.payment,
		PaymentAfter:   &after,
		InvoicesBefore: plan.invoices,
		InvoicesAfter:  invoicesAfter,
		Journal:        journal.preview(plan.control, req.AccountingDate, plan.period, actor.UserID),
	}, nil
}

func (s *Service) PreviewReverse(
	ctx context.Context,
	req *serviceports.ReverseCustomerPaymentRequest,
	actor *serviceports.RequestActor,
) (*serviceports.CustomerPaymentChangePreview, error) {
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
		invoicesAfter[idx].RemovePaymentMinor(app.AppliedAmountMinor)
	}

	journal := reversalJournal(plan.control, plan.payment)

	return &serviceports.CustomerPaymentChangePreview{
		PaymentBefore:  plan.payment,
		PaymentAfter:   &after,
		InvoicesBefore: plan.invoices,
		InvoicesAfter:  invoicesAfter,
		Journal:        journal.preview(plan.control, req.AccountingDate, plan.period, actor.UserID),
	}, nil
}
