//nolint:gocognit // existing legacy workflow/API shape is intentionally kept stable
package customerpaymentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/journalposting"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"golang.org/x/text/currency"
)

type ValidatorParams struct {
	fx.In

	InvoiceRepo        invoiceports.InvoiceRepository
	FiscalPeriodRepo   fiscalperiodports.FiscalPeriodRepository
	AdjustmentCtrlRepo tenantports.InvoiceAdjustmentControlRepository
}

type Validator struct {
	invoiceRepo        invoiceports.InvoiceRepository
	fiscalPeriodRepo   fiscalperiodports.FiscalPeriodRepository
	adjustmentCtrlRepo tenantports.InvoiceAdjustmentControlRepository
}

type invoiceKey struct {
	ID pulid.ID
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		invoiceRepo:        p.InvoiceRepo,
		fiscalPeriodRepo:   p.FiscalPeriodRepo,
		adjustmentCtrlRepo: p.AdjustmentCtrlRepo,
	}
}

func (v *Validator) checkShortPays(
	ctx context.Context,
	orgID pulid.ID,
	applications []*customerpayment.Application,
	me *errortypes.MultiError,
) error {
	var control *invoiceadjustmentcontrol.InvoiceAdjustmentControl
	for idx, app := range applications {
		if app == nil || app.ShortPayAmountMinor <= 0 {
			continue
		}
		if control == nil {
			var err error
			if control, err = v.adjustmentCtrlRepo.GetByOrgID(ctx, orgID); err != nil {
				return err
			}
		}
		switch shortPayPolicy(control, app.ShortPayAmountMinor) {
		case shortPayRefused:
			me.WithIndex("applications", idx).Add(
				"shortPayAmountMinor",
				errortypes.ErrInvalidOperation,
				"Write-offs are disallowed by policy, so this invoice cannot take a short pay",
			)
		case shortPayNeedsApproval:
			me.WithIndex("applications", idx).Add(
				"shortPayAmountMinor",
				errortypes.ErrInvalidOperation,
				"This short pay needs approval: apply what the customer paid and write off the rest with an invoice adjustment",
			)
		case shortPayAllowed:
		}
	}
	return nil
}

type shortPayOutcome int

const (
	shortPayAllowed shortPayOutcome = iota
	shortPayNeedsApproval
	shortPayRefused
)

func shortPayPolicy(
	control *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
	amountMinor int64,
) shortPayOutcome {
	switch control.WriteOffApprovalPolicy {
	case invoiceadjustmentcontrol.WriteOffApprovalPolicyDisallow:
		return shortPayRefused
	case invoiceadjustmentcontrol.WriteOffApprovalPolicyAlwaysRequireApproval:
		return shortPayNeedsApproval
	case invoiceadjustmentcontrol.WriteOffApprovalPolicyRequireApprovalAboveThreshold:
		if money.DecimalFromMinor(amountMinor).
			GreaterThanOrEqual(control.WriteOffApprovalThreshold) {
			return shortPayNeedsApproval
		}
		return shortPayAllowed
	default:
		return shortPayRefused
	}
}

//nolint:nestif // existing validation flow mirrors business rule nesting
func (v *Validator) ValidatePostAndApply( //nolint:cyclop,funlen,gocyclo // legacy workflow
	ctx context.Context,
	entity *customerpayment.Payment,
	tenantInfo invoiceports.GetInvoiceByIDRequest,
	control *accountingcontrol.AccountingControl,
) ([]*invoice.Invoice, *journalposting.ResolvedPeriod, error) {
	me := errortypes.NewMultiError()
	entity.Validate(me)
	if strings.TrimSpace(entity.CurrencyCode) != "" {
		if _, err := currency.ParseISO(strings.ToUpper(entity.CurrencyCode)); err != nil {
			me.Add(
				"currencyCode",
				errortypes.ErrInvalid,
				"Currency code must be a valid ISO 4217 code",
			)
		}
	}
	if me.HasErrors() {
		return nil, nil, me
	}
	if strings.TrimSpace(entity.CurrencyCode) != "" {
		entity.CurrencyCode = strings.ToUpper(entity.CurrencyCode)
	}
	if control != nil {
		hasShortPay := false
		for _, app := range entity.Applications {
			if app != nil && app.ShortPayAmountMinor > 0 {
				hasShortPay = true
				break
			}
		}
		if control.CurrencyMode == accountingcontrol.CurrencyModeSingleCurrency &&
			!strings.EqualFold(entity.CurrencyCode, control.FunctionalCurrencyCode) {
			me.Add(
				"currencyCode",
				errortypes.ErrInvalid,
				"Customer payment currency must match the tenant functional currency",
			)
		}
		if control.DefaultCashAccountID.IsNil() {
			me.Add(
				"defaultCashAccountId",
				errortypes.ErrRequired,
				"Default cash account is required for customer payment posting",
			)
		}
		if control.DefaultUnappliedCashAccountID.IsNil() {
			me.Add(
				"defaultUnappliedCashAccountId",
				errortypes.ErrRequired,
				"Default unapplied cash account is required for customer payment posting",
			)
		}
		if hasShortPay && control.DefaultWriteOffAccountID.IsNil() {
			me.Add(
				"defaultWriteOffAccountId",
				errortypes.ErrRequired,
				"Default write-off account is required for customer short-pay recognition",
			)
		}
		if control.AccountingBasis == accountingcontrol.AccountingBasisCash ||
			control.RevenueRecognitionPolicy == accountingcontrol.RevenueRecognitionOnCashReceipt {
			for idx, app := range entity.Applications {
				if app != nil && app.ShortPayAmountMinor > 0 {
					me.WithIndex("applications", idx).
						Add("shortPayAmountMinor", errortypes.ErrInvalidOperation, "Short-pay recognition is not supported for cash-basis customer payments in this slice")
				}
			}
		}
	}
	if me.HasErrors() {
		return nil, nil, me
	}

	period, err := resolvePaymentPeriod(ctx, v.fiscalPeriodRepo, &paymentPeriodRequest{
		tenant:  pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		date:    entity.AccountingDate,
		control: control,
	})
	if err != nil {
		return nil, nil, err
	}

	resolvedInvoices := make([]*invoice.Invoice, 0, len(entity.Applications))
	var totalApplied int64
	seenInvoices := make(map[invoiceKey]struct{}, len(entity.Applications))
	for idx, app := range entity.Applications {
		key := invoiceKey{ID: app.InvoiceID}
		if _, ok := seenInvoices[key]; ok {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrDuplicate, "Invoice can only appear once in a payment application set")
			continue
		}
		seenInvoices[key] = struct{}{}
		inv, getErr := v.invoiceRepo.GetByID(
			ctx,
			invoiceports.GetInvoiceByIDRequest{
				ID:         app.InvoiceID,
				TenantInfo: tenantInfo.TenantInfo,
			},
		)
		if getErr != nil {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalid, "Invoice was not found")
			continue
		}
		if inv.CustomerID != entity.CustomerID {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalid, "Invoice customer must match payment customer")
		}
		if !inv.BilledIn(entity.CurrencyCode) {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalid, "Invoice {0} is billed in {1}, but this payment is in {2}", inv.Number, inv.CurrencyCode, entity.CurrencyCode)
		}
		if inv.Status != invoice.StatusPosted {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalidOperation, "Only posted invoices can accept customer payments")
		}
		if inv.BillType != billingvocab.BillTypeInvoice &&
			inv.BillType != billingvocab.BillTypeDebitMemo {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalidOperation, "Customer payments only support invoices and debit memos in this slice")
		}
		openBalanceMinor := inv.OpenBalanceMinor()
		settlementMinor := app.AppliedAmountMinor + app.ShortPayAmountMinor
		if settlementMinor > openBalanceMinor {
			me.WithIndex("applications", idx).
				Add("appliedAmountMinor", errortypes.ErrInvalid, "Applied amount plus short pay exceeds invoice open balance by {0} minor units", settlementMinor-openBalanceMinor)
		}
		totalApplied += app.AppliedAmountMinor
		resolvedInvoices = append(resolvedInvoices, inv)
	}
	if err := v.checkShortPays(
		ctx,
		tenantInfo.TenantInfo.OrgID,
		entity.Applications,
		me,
	); err != nil {
		return nil, nil, err
	}
	if totalApplied > entity.AmountMinor {
		me.Add(
			"amountMinor",
			errortypes.ErrInvalid,
			"Payment amount must be greater than or equal to the total applied amount",
		)
	}
	if me.HasErrors() {
		return nil, nil, me
	}
	entity.SyncAmounts()
	entity.Memo = strings.TrimSpace(entity.Memo)
	entity.ReferenceNumber = strings.TrimSpace(entity.ReferenceNumber)
	if strings.TrimSpace(entity.CurrencyCode) == "" {
		entity.CurrencyCode = money.DefaultCurrencyCode
	}
	return resolvedInvoices, period, nil
}

//nolint:nestif // existing validation flow mirrors business rule nesting
func (v *Validator) ValidateApplyUnapplied( //nolint:cyclop,funlen,gocyclo // legacy workflow
	ctx context.Context,
	payment *customerpayment.Payment,
	accountingDate int64,
	applications []*customerpayment.Application,
	tenantInfo invoiceports.GetInvoiceByIDRequest,
	control *accountingcontrol.AccountingControl,
) ([]*invoice.Invoice, *journalposting.ResolvedPeriod, error) {
	me := errortypes.NewMultiError()
	if payment == nil {
		me.Add("paymentId", errortypes.ErrRequired, "Customer payment is required")
		return nil, nil, me
	}
	if payment.Status != customerpayment.StatusPosted {
		me.Add(
			"paymentId",
			errortypes.ErrInvalidOperation,
			"Only posted customer payments can be applied",
		)
	}
	if payment.UnappliedAmountMinor <= 0 {
		me.Add(
			"paymentId",
			errortypes.ErrInvalidOperation,
			"Customer payment has no unapplied amount remaining",
		)
	}
	if len(applications) == 0 {
		me.Add(
			"applications",
			errortypes.ErrRequired,
			"At least one payment application is required",
		)
	}
	for idx, app := range applications {
		if app == nil {
			me.WithIndex("applications", idx).
				Add("", errortypes.ErrInvalid, "Payment applications must not contain null values")
			continue
		}
		app.Validate(me.WithIndex("applications", idx))
	}
	if control != nil {
		hasShortPay := false
		for _, app := range applications {
			if app != nil && app.ShortPayAmountMinor > 0 {
				hasShortPay = true
				break
			}
		}
		if control.DefaultUnappliedCashAccountID.IsNil() {
			me.Add(
				"defaultUnappliedCashAccountId",
				errortypes.ErrRequired,
				"Default unapplied cash account is required for customer payment application",
			)
		}
		if hasShortPay && control.DefaultWriteOffAccountID.IsNil() {
			me.Add(
				"defaultWriteOffAccountId",
				errortypes.ErrRequired,
				"Default write-off account is required for customer short-pay recognition",
			)
		}
		if control.AccountingBasis == accountingcontrol.AccountingBasisAccrual &&
			control.DefaultARAccountID.IsNil() {
			me.Add(
				"defaultArAccountId",
				errortypes.ErrRequired,
				"Default AR account is required for accrual customer payment application",
			)
		}
		if (control.AccountingBasis == accountingcontrol.AccountingBasisCash || control.RevenueRecognitionPolicy == accountingcontrol.RevenueRecognitionOnCashReceipt) &&
			control.DefaultRevenueAccountID.IsNil() {
			me.Add(
				"defaultRevenueAccountId",
				errortypes.ErrRequired,
				"Default revenue account is required for cash-basis customer payment application",
			)
		}
		if control.AccountingBasis == accountingcontrol.AccountingBasisCash ||
			control.RevenueRecognitionPolicy == accountingcontrol.RevenueRecognitionOnCashReceipt {
			for idx, app := range applications {
				if app != nil && app.ShortPayAmountMinor > 0 {
					me.WithIndex("applications", idx).
						Add("shortPayAmountMinor", errortypes.ErrInvalidOperation, "Short-pay recognition is not supported for cash-basis customer payments in this slice")
				}
			}
		}
	}
	if me.HasErrors() {
		return nil, nil, me
	}

	period, err := resolvePaymentPeriod(ctx, v.fiscalPeriodRepo, &paymentPeriodRequest{
		tenant:  pagination.TenantInfo{OrgID: payment.OrganizationID, BuID: payment.BusinessUnitID},
		date:    accountingDate,
		control: control,
	})
	if err != nil {
		return nil, nil, err
	}

	seenInvoices := make(map[invoiceKey]struct{}, len(applications))
	resolvedInvoices := make([]*invoice.Invoice, 0, len(applications))
	var totalApplied int64
	for idx, app := range applications {
		if app == nil {
			continue
		}
		key := invoiceKey{ID: app.InvoiceID}
		if _, ok := seenInvoices[key]; ok {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrDuplicate, "Invoice can only appear once in a payment application set")
			continue
		}
		seenInvoices[key] = struct{}{}
		inv, getErr := v.invoiceRepo.GetByID(
			ctx,
			invoiceports.GetInvoiceByIDRequest{
				ID:         app.InvoiceID,
				TenantInfo: tenantInfo.TenantInfo,
			},
		)
		if getErr != nil {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalid, "Invoice was not found")
			continue
		}
		if inv.CustomerID != payment.CustomerID {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalid, "Invoice customer must match payment customer")
		}
		if !inv.BilledIn(payment.CurrencyCode) {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalid, "Invoice {0} is billed in {1}, but this payment is in {2}", inv.Number, inv.CurrencyCode, payment.CurrencyCode)
		}
		if inv.Status != invoice.StatusPosted {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalidOperation, "Only posted invoices can accept customer payment application")
		}
		if inv.BillType != billingvocab.BillTypeInvoice &&
			inv.BillType != billingvocab.BillTypeDebitMemo {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalidOperation, "Customer payments only support invoices and debit memos in this slice")
		}
		openBalanceMinor := inv.OpenBalanceMinor()
		settlementMinor := app.AppliedAmountMinor + app.ShortPayAmountMinor
		if settlementMinor > openBalanceMinor {
			me.WithIndex("applications", idx).
				Add("appliedAmountMinor", errortypes.ErrInvalid, "Applied amount plus short pay exceeds invoice open balance by {0} minor units", settlementMinor-openBalanceMinor)
		}
		totalApplied += app.AppliedAmountMinor
		resolvedInvoices = append(resolvedInvoices, inv)
	}
	if err := v.checkShortPays(ctx, tenantInfo.TenantInfo.OrgID, applications, me); err != nil {
		return nil, nil, err
	}
	if totalApplied > payment.UnappliedAmountMinor {
		me.Add(
			"applications",
			errortypes.ErrInvalid,
			"Total applied amount exceeds payment unapplied amount",
		)
	}
	if me.HasErrors() {
		return nil, nil, me
	}
	return resolvedInvoices, period, nil
}

//nolint:nestif // existing validation flow mirrors business rule nesting
func (v *Validator) ValidateReverse( //nolint:funlen // legacy workflow
	ctx context.Context,
	payment *customerpayment.Payment,
	accountingDate int64,
	control *accountingcontrol.AccountingControl,
	tenantInfo invoiceports.GetInvoiceByIDRequest,
) ([]*invoice.Invoice, *journalposting.ResolvedPeriod, error) {
	me := errortypes.NewMultiError()
	if payment == nil {
		me.Add("paymentId", errortypes.ErrRequired, "Customer payment is required")
		return nil, nil, me
	}
	if !payment.CanReverse() {
		me.Add(
			"paymentId",
			errortypes.ErrInvalidOperation,
			"Only posted customer payments can be reversed",
		)
	}
	if accountingDate == 0 {
		me.Add("accountingDate", errortypes.ErrRequired, "Accounting date is required")
	}
	if control != nil {
		if control.DefaultCashAccountID.IsNil() {
			me.Add(
				"defaultCashAccountId",
				errortypes.ErrRequired,
				"Default cash account is required for customer payment reversal",
			)
		}
		if control.DefaultUnappliedCashAccountID.IsNil() {
			me.Add(
				"defaultUnappliedCashAccountId",
				errortypes.ErrRequired,
				"Default unapplied cash account is required for customer payment reversal",
			)
		}
		if payment.AppliedAmountMinor > 0 {
			if control.AccountingBasis == accountingcontrol.AccountingBasisCash ||
				control.RevenueRecognitionPolicy == accountingcontrol.RevenueRecognitionOnCashReceipt {
				requireAccountField(
					me,
					control.DefaultRevenueAccountID,
					"defaultRevenueAccountId",
					"Default revenue account is required for customer payment reversal",
				)
			} else {
				requireAccountField(
					me,
					control.DefaultARAccountID,
					"defaultArAccountId",
					"Default AR account is required for customer payment reversal",
				)
			}
		}
	}
	if me.HasErrors() {
		return nil, nil, me
	}
	period, err := resolvePaymentPeriod(ctx, v.fiscalPeriodRepo, &paymentPeriodRequest{
		tenant:  pagination.TenantInfo{OrgID: payment.OrganizationID, BuID: payment.BusinessUnitID},
		date:    accountingDate,
		control: control,
	})
	if err != nil {
		return nil, nil, err
	}
	invoices := make([]*invoice.Invoice, 0, len(payment.Applications))
	for idx, app := range payment.Applications {
		if app == nil {
			continue
		}
		inv, getErr := v.invoiceRepo.GetByID(
			ctx,
			invoiceports.GetInvoiceByIDRequest{
				ID:         app.InvoiceID,
				TenantInfo: tenantInfo.TenantInfo,
			},
		)
		if getErr != nil {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalid, "Invoice was not found")
			continue
		}
		if inv.CustomerID != payment.CustomerID {
			me.WithIndex("applications", idx).
				Add("invoiceId", errortypes.ErrInvalid, "Invoice customer must match payment customer")
		}
		invoices = append(invoices, inv)
	}
	if me.HasErrors() {
		return nil, nil, me
	}
	return invoices, period, nil
}

type paymentPeriodRequest struct {
	tenant  pagination.TenantInfo
	date    int64
	control *accountingcontrol.AccountingControl
}

func resolvePaymentPeriod(
	ctx context.Context,
	periods journalposting.PeriodReader,
	req *paymentPeriodRequest,
) (*journalposting.ResolvedPeriod, error) {
	return journalposting.ResolvePeriod(ctx, periods, &journalposting.ResolvePeriodRequest{
		OrganizationID: req.tenant.OrgID,
		BusinessUnitID: req.tenant.BuID,
		Date:           req.date,
		Policy:         journalposting.ClosedPeriodPolicy(req.control),
		RefuseLocked:   journalposting.RefusesLockedSubledger(req.control),
		Subject:        "customer payment",
	})
}

func requireAccountField(me *errortypes.MultiError, id pulid.ID, field, message string) {
	if id.IsNil() {
		me.Add(field, errortypes.ErrRequired, message)
	}
}
