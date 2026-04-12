package customerpaymentservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"golang.org/x/text/currency"
)

type ValidatorParams struct {
	fx.In

	InvoiceRepo      repositories.InvoiceRepository
	FiscalPeriodRepo repositories.FiscalPeriodRepository
}

type Validator struct {
	invoiceRepo      repositories.InvoiceRepository
	fiscalPeriodRepo repositories.FiscalPeriodRepository
}

type invoiceKey struct {
	ID pulid.ID
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{invoiceRepo: p.InvoiceRepo, fiscalPeriodRepo: p.FiscalPeriodRepo}
}

func (v *Validator) ValidatePostAndApply(
	ctx context.Context,
	entity *customerpayment.Payment,
	tenantInfo repositories.GetInvoiceByIDRequest,
	control *tenant.AccountingControl,
) ([]*invoice.Invoice, *fiscalperiod.FiscalPeriod, *errortypes.MultiError) {
	me := errortypes.NewMultiError()
	entity.Validate(me)
	if strings.TrimSpace(entity.CurrencyCode) != "" {
		if _, err := currency.ParseISO(strings.ToUpper(entity.CurrencyCode)); err != nil {
			me.Add("currencyCode", errortypes.ErrInvalid, "Currency code must be a valid ISO 4217 code")
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
		if control.CurrencyMode == tenant.CurrencyModeSingleCurrency && !strings.EqualFold(entity.CurrencyCode, control.FunctionalCurrencyCode) {
			me.Add("currencyCode", errortypes.ErrInvalid, "Customer payment currency must match the tenant functional currency")
		}
		if control.DefaultCashAccountID.IsNil() {
			me.Add("defaultCashAccountId", errortypes.ErrRequired, "Default cash account is required for customer payment posting")
		}
		if control.DefaultUnappliedCashAccountID.IsNil() {
			me.Add("defaultUnappliedCashAccountId", errortypes.ErrRequired, "Default unapplied cash account is required for customer payment posting")
		}
		if hasShortPay && control.DefaultWriteOffAccountID.IsNil() {
			me.Add("defaultWriteOffAccountId", errortypes.ErrRequired, "Default write-off account is required for customer short-pay recognition")
		}
		if control.AccountingBasis == tenant.AccountingBasisCash || control.RevenueRecognitionPolicy == tenant.RevenueRecognitionOnCashReceipt {
			for idx, app := range entity.Applications {
				if app != nil && app.ShortPayAmountMinor > 0 {
					me.WithIndex("applications", idx).Add("shortPayAmountMinor", errortypes.ErrInvalidOperation, "Short-pay recognition is not supported for cash-basis customer payments in this slice")
				}
			}
		}
	}
	if me.HasErrors() {
		return nil, nil, me
	}

	period, err := v.fiscalPeriodRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID, Date: entity.AccountingDate})
	if err != nil {
		me.Add("accountingDate", errortypes.ErrInvalid, "Accounting date must fall within a fiscal period")
		return nil, nil, me
	}

	resolvedInvoices := make([]*invoice.Invoice, 0, len(entity.Applications))
	var totalApplied int64
	seenInvoices := make(map[invoiceKey]struct{}, len(entity.Applications))
	for idx, app := range entity.Applications {
		key := invoiceKey{ID: app.InvoiceID}
		if _, ok := seenInvoices[key]; ok {
			me.WithIndex("applications", idx).Add("invoiceId", errortypes.ErrDuplicate, "Invoice can only appear once in a payment application set")
			continue
		}
		seenInvoices[key] = struct{}{}
		inv, getErr := v.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{ID: app.InvoiceID, TenantInfo: tenantInfo.TenantInfo})
		if getErr != nil {
			me.WithIndex("applications", idx).Add("invoiceId", errortypes.ErrInvalid, "Invoice was not found")
			continue
		}
		if inv.CustomerID != entity.CustomerID {
			me.WithIndex("applications", idx).Add("invoiceId", errortypes.ErrInvalid, "Invoice customer must match payment customer")
		}
		if inv.Status != invoice.StatusPosted {
			me.WithIndex("applications", idx).Add("invoiceId", errortypes.ErrInvalidOperation, "Only posted invoices can accept customer payments")
		}
		if inv.BillType != billingqueue.BillTypeInvoice && inv.BillType != billingqueue.BillTypeDebitMemo {
			me.WithIndex("applications", idx).Add("invoiceId", errortypes.ErrInvalidOperation, "Customer payments only support invoices and debit memos in this slice")
		}
		openBalanceMinor := inv.OpenBalanceMinor()
		settlementMinor := app.AppliedAmountMinor + app.ShortPayAmountMinor
		if settlementMinor > openBalanceMinor {
			me.WithIndex("applications", idx).Add("appliedAmountMinor", errortypes.ErrInvalid, fmt.Sprintf("Applied amount plus short pay exceeds invoice open balance by %d minor units", settlementMinor-openBalanceMinor))
		}
		totalApplied += app.AppliedAmountMinor
		resolvedInvoices = append(resolvedInvoices, inv)
	}
	if totalApplied > entity.AmountMinor {
		me.Add("amountMinor", errortypes.ErrInvalid, "Payment amount must be greater than or equal to the total applied amount")
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

func (v *Validator) ValidateApplyUnapplied(
	ctx context.Context,
	payment *customerpayment.Payment,
	accountingDate int64,
	applications []*customerpayment.Application,
	tenantInfo repositories.GetInvoiceByIDRequest,
	control *tenant.AccountingControl,
) ([]*invoice.Invoice, *fiscalperiod.FiscalPeriod, *errortypes.MultiError) {
	me := errortypes.NewMultiError()
	if payment == nil {
		me.Add("paymentId", errortypes.ErrRequired, "Customer payment is required")
		return nil, nil, me
	}
	if payment.Status != customerpayment.StatusPosted {
		me.Add("paymentId", errortypes.ErrInvalidOperation, "Only posted customer payments can be applied")
	}
	if payment.UnappliedAmountMinor <= 0 {
		me.Add("paymentId", errortypes.ErrInvalidOperation, "Customer payment has no unapplied amount remaining")
	}
	if len(applications) == 0 {
		me.Add("applications", errortypes.ErrRequired, "At least one payment application is required")
	}
	for idx, app := range applications {
		if app == nil {
			me.WithIndex("applications", idx).Add("", errortypes.ErrInvalid, "Payment applications must not contain null values")
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
			me.Add("defaultUnappliedCashAccountId", errortypes.ErrRequired, "Default unapplied cash account is required for customer payment application")
		}
		if hasShortPay && control.DefaultWriteOffAccountID.IsNil() {
			me.Add("defaultWriteOffAccountId", errortypes.ErrRequired, "Default write-off account is required for customer short-pay recognition")
		}
		if control.AccountingBasis == tenant.AccountingBasisAccrual && control.DefaultARAccountID.IsNil() {
			me.Add("defaultArAccountId", errortypes.ErrRequired, "Default AR account is required for accrual customer payment application")
		}
		if (control.AccountingBasis == tenant.AccountingBasisCash || control.RevenueRecognitionPolicy == tenant.RevenueRecognitionOnCashReceipt) && control.DefaultRevenueAccountID.IsNil() {
			me.Add("defaultRevenueAccountId", errortypes.ErrRequired, "Default revenue account is required for cash-basis customer payment application")
		}
		if control.AccountingBasis == tenant.AccountingBasisCash || control.RevenueRecognitionPolicy == tenant.RevenueRecognitionOnCashReceipt {
			for idx, app := range applications {
				if app != nil && app.ShortPayAmountMinor > 0 {
					me.WithIndex("applications", idx).Add("shortPayAmountMinor", errortypes.ErrInvalidOperation, "Short-pay recognition is not supported for cash-basis customer payments in this slice")
				}
			}
		}
	}
	if me.HasErrors() {
		return nil, nil, me
	}

	period, err := v.fiscalPeriodRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{OrgID: payment.OrganizationID, BuID: payment.BusinessUnitID, Date: accountingDate})
	if err != nil {
		me.Add("accountingDate", errortypes.ErrInvalid, "Accounting date must fall within a fiscal period")
		return nil, nil, me
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
			me.WithIndex("applications", idx).Add("invoiceId", errortypes.ErrDuplicate, "Invoice can only appear once in a payment application set")
			continue
		}
		seenInvoices[key] = struct{}{}
		inv, getErr := v.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{ID: app.InvoiceID, TenantInfo: tenantInfo.TenantInfo})
		if getErr != nil {
			me.WithIndex("applications", idx).Add("invoiceId", errortypes.ErrInvalid, "Invoice was not found")
			continue
		}
		if inv.CustomerID != payment.CustomerID {
			me.WithIndex("applications", idx).Add("invoiceId", errortypes.ErrInvalid, "Invoice customer must match payment customer")
		}
		if inv.Status != invoice.StatusPosted {
			me.WithIndex("applications", idx).Add("invoiceId", errortypes.ErrInvalidOperation, "Only posted invoices can accept customer payment application")
		}
		if inv.BillType != billingqueue.BillTypeInvoice && inv.BillType != billingqueue.BillTypeDebitMemo {
			me.WithIndex("applications", idx).Add("invoiceId", errortypes.ErrInvalidOperation, "Customer payments only support invoices and debit memos in this slice")
		}
		openBalanceMinor := inv.OpenBalanceMinor()
		settlementMinor := app.AppliedAmountMinor + app.ShortPayAmountMinor
		if settlementMinor > openBalanceMinor {
			me.WithIndex("applications", idx).Add("appliedAmountMinor", errortypes.ErrInvalid, fmt.Sprintf("Applied amount plus short pay exceeds invoice open balance by %d minor units", settlementMinor-openBalanceMinor))
		}
		totalApplied += app.AppliedAmountMinor
		resolvedInvoices = append(resolvedInvoices, inv)
	}
	if totalApplied > payment.UnappliedAmountMinor {
		me.Add("applications", errortypes.ErrInvalid, "Total applied amount exceeds payment unapplied amount")
	}
	if me.HasErrors() {
		return nil, nil, me
	}
	return resolvedInvoices, period, nil
}

func (v *Validator) ValidateReverse(
	ctx context.Context,
	payment *customerpayment.Payment,
	accountingDate int64,
	control *tenant.AccountingControl,
	tenantInfo repositories.GetInvoiceByIDRequest,
) ([]*invoice.Invoice, *fiscalperiod.FiscalPeriod, *errortypes.MultiError) {
	me := errortypes.NewMultiError()
	if payment == nil {
		me.Add("paymentId", errortypes.ErrRequired, "Customer payment is required")
		return nil, nil, me
	}
	if !payment.CanReverse() {
		me.Add("paymentId", errortypes.ErrInvalidOperation, "Only posted customer payments can be reversed")
	}
	if accountingDate == 0 {
		me.Add("accountingDate", errortypes.ErrRequired, "Accounting date is required")
	}
	if control != nil {
		if control.DefaultCashAccountID.IsNil() {
			me.Add("defaultCashAccountId", errortypes.ErrRequired, "Default cash account is required for customer payment reversal")
		}
		if control.DefaultUnappliedCashAccountID.IsNil() {
			me.Add("defaultUnappliedCashAccountId", errortypes.ErrRequired, "Default unapplied cash account is required for customer payment reversal")
		}
		if payment.AppliedAmountMinor > 0 {
			if control.AccountingBasis == tenant.AccountingBasisCash || control.RevenueRecognitionPolicy == tenant.RevenueRecognitionOnCashReceipt {
				requireAccountField(me, control.DefaultRevenueAccountID, "defaultRevenueAccountId", "Default revenue account is required for customer payment reversal")
			} else {
				requireAccountField(me, control.DefaultARAccountID, "defaultArAccountId", "Default AR account is required for customer payment reversal")
			}
		}
	}
	if me.HasErrors() {
		return nil, nil, me
	}
	period, err := v.fiscalPeriodRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{OrgID: payment.OrganizationID, BuID: payment.BusinessUnitID, Date: accountingDate})
	if err != nil {
		me.Add("accountingDate", errortypes.ErrInvalid, "Accounting date must fall within a fiscal period")
		return nil, nil, me
	}
	invoices := make([]*invoice.Invoice, 0, len(payment.Applications))
	for idx, app := range payment.Applications {
		if app == nil {
			continue
		}
		inv, getErr := v.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{ID: app.InvoiceID, TenantInfo: tenantInfo.TenantInfo})
		if getErr != nil {
			me.WithIndex("applications", idx).Add("invoiceId", errortypes.ErrInvalid, "Invoice was not found")
			continue
		}
		if inv.CustomerID != payment.CustomerID {
			me.WithIndex("applications", idx).Add("invoiceId", errortypes.ErrInvalid, "Invoice customer must match payment customer")
		}
		invoices = append(invoices, inv)
	}
	if me.HasErrors() {
		return nil, nil, me
	}
	return invoices, period, nil
}

func requireAccountField(me *errortypes.MultiError, id pulid.ID, field, message string) {
	if id.IsNil() {
		me.Add(field, errortypes.ErrRequired, message)
	}
}
