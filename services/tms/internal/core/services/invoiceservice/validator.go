package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ValidatorParams struct {
	fx.In

	DB               *postgres.Connection
	Logger           *zap.Logger
	AccountingRepo   repositories.AccountingControlRepository
	FiscalPeriodRepo repositories.FiscalPeriodRepository
	ShipmentRepo     repositories.ShipmentRepository
	AccountingPolicy *accountingcontrolpolicyservice.Service
}

type Validator struct {
	validator        *validationframework.TenantedValidator[*invoice.Invoice]
	l                *zap.Logger
	accountingRepo   repositories.AccountingControlRepository
	fiscalPeriodRepo repositories.FiscalPeriodRepository
	shipmentRepo     repositories.ShipmentRepository
	accountingPolicy *accountingcontrolpolicyservice.Service
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		l: p.Logger.Named("validator.invoice"),
		validator: validationframework.NewTenantedValidatorBuilder[*invoice.Invoice]().
			WithModelName("Invoice").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			Build(),
		accountingRepo:   p.AccountingRepo,
		fiscalPeriodRepo: p.FiscalPeriodRepo,
		shipmentRepo:     p.ShipmentRepo,
		accountingPolicy: p.AccountingPolicy,
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *invoice.Invoice,
) *errortypes.MultiError {
	multiErr := v.validator.ValidateCreate(ctx, entity)
	return validateLineDerivedTotals(entity, multiErr)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *invoice.Invoice,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}

func (v *Validator) ValidatePost(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
	postedAt int64,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	v.validatePostingPeriodPolicy(ctx, entity, postedAt, multiErr)
	v.validatePostingReconciliation(ctx, entity, tenantInfo, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func validateLineDerivedTotals(
	entity *invoice.Invoice,
	multiErr *errortypes.MultiError,
) *errortypes.MultiError {
	if entity == nil || len(entity.Lines) == 0 {
		return multiErr
	}

	if multiErr == nil {
		multiErr = errortypes.NewMultiError()
	}

	expectedSubtotal := sumLinesByType(entity.Lines, invoice.InvoiceLineTypeFreight)
	expectedOther := sumLinesByType(entity.Lines, invoice.InvoiceLineTypeAccessorial)
	expectedTotal := sumLinesByType(entity.Lines, "")

	if !entity.SubtotalAmount.Equal(expectedSubtotal) {
		multiErr.Add(
			"subtotalAmount",
			errortypes.ErrInvalid,
			"Invoice subtotal must equal the sum of freight lines",
		)
	}
	if !entity.OtherAmount.Equal(expectedOther) {
		multiErr.Add(
			"otherAmount",
			errortypes.ErrInvalid,
			"Invoice other amount must equal the sum of accessorial lines",
		)
	}
	if !entity.TotalAmount.Equal(expectedTotal) {
		multiErr.Add(
			"totalAmount",
			errortypes.ErrInvalid,
			"Invoice total must equal the sum of invoice lines",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) validatePostingPeriodPolicy(
	ctx context.Context,
	entity *invoice.Invoice,
	postedAt int64,
	multiErr *errortypes.MultiError,
) {
	if v.accountingRepo == nil || v.fiscalPeriodRepo == nil {
		return
	}

	control, err := v.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return
		}
		multiErr.Add("postedAt", errortypes.ErrSystemError, "Failed to load accounting control")
		return
	}
	if v.accountingPolicyService().
		CanCreateInvoiceLedgerEntry(control, invoicePostingSourceEvent(entity.BillType)) &&
		!invoicePostingHasRequiredAccounts(control) {
		multiErr.Add(
			"accountingControl",
			errortypes.ErrRequired,
			"Invoice posting requires default Accounts Receivable and revenue accounts",
		)
		return
	}

	period, err := v.fiscalPeriodRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
		Date:  postedAt,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			multiErr.Add(
				"postedAt",
				errortypes.ErrRequired,
				"No fiscal period covers the invoice posting date",
			)
			return
		}
		multiErr.Add("postedAt", errortypes.ErrSystemError, "Failed to resolve accounting period")
		return
	}

	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch period.Status {
	case fiscalperiod.StatusLocked:
		if control.LockedPeriodPostingPolicy == tenant.LockedPeriodPostingPolicyBlockSubledgerAllowManualJe {
			multiErr.Add(
				"postedAt",
				errortypes.ErrInvalidOperation,
				"Invoice posting is blocked because the accounting period is locked",
			)
		}
	case fiscalperiod.StatusClosed, fiscalperiod.StatusPermanentlyClosed:
		if control.ClosedPeriodPostingPolicy == tenant.ClosedPeriodPostingPolicyRequireReopen {
			multiErr.Add(
				"postedAt",
				errortypes.ErrInvalidOperation,
				"Invoice posting is blocked because the accounting period is closed and must be reopened",
			)
		}
	}
}

func (v *Validator) accountingPolicyService() *accountingcontrolpolicyservice.Service {
	if v.accountingPolicy != nil {
		return v.accountingPolicy
	}
	return accountingcontrolpolicyservice.New(
		accountingcontrolpolicyservice.Params{Logger: zap.NewNop()},
	)
}

func (v *Validator) validatePostingReconciliation(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
	multiErr *errortypes.MultiError,
) {
	if v.accountingRepo == nil || v.shipmentRepo == nil {
		return
	}

	control, err := v.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return
		}
		multiErr.Add("totalAmount", errortypes.ErrSystemError, "Failed to load accounting control")
		return
	}

	if control.ReconciliationMode == tenant.ReconciliationModeDisabled {
		return
	}

	legs, err := loadInvoiceLegs(ctx, v.shipmentRepo, entity, tenantInfo)
	if err != nil {
		multiErr.Add(
			"shipmentId",
			errortypes.ErrSystemError,
			"Failed to load invoice legs for reconciliation",
		)
		return
	}
	if len(legs) == 0 {
		return
	}

	expectedTotal := reconciliationExpectedTotal(entity, legs)
	discrepancy := entity.TotalAmount.Sub(expectedTotal).Abs()
	if !discrepancy.GreaterThan(control.ReconciliationToleranceAmount) {
		return
	}

	if control.ReconciliationMode == tenant.ReconciliationModeBlockPosting {
		multiErr.Add(
			"totalAmount",
			errortypes.ErrInvalidOperation,
			"Invoice posting is blocked because the invoice total exceeds the reconciliation tolerance",
		)
		return
	}

	v.l.Warn("invoice reconciliation discrepancy exceeded tolerance during posting",
		zap.String("invoiceId", entity.ID.String()),
		zap.String("shipmentId", entity.ShipmentID.String()),
		zap.String("orderId", entity.OrderID.String()),
		zap.String("invoiceTotal", entity.TotalAmount.String()),
		zap.String("expectedTotal", expectedTotal.String()),
		zap.String("toleranceAmount", control.ReconciliationToleranceAmount.String()),
		zap.String("discrepancyAmount", discrepancy.String()),
	)
}
