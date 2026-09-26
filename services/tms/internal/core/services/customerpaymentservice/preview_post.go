package customerpaymentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/exchangeratestamp"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// PreviewPostAndApply is PostAndApply up to the write: the payment as it
// would be recorded, with its applied and unapplied amounts, and each invoice
// it pays before and after the payment is applied to it.
func (s *Service) PreviewPostAndApply(
	ctx context.Context,
	req *serviceports.PostCustomerPaymentRequest,
	actor *serviceports.RequestActor,
) (*serviceports.CustomerPaymentPostPreview, error) {
	plan, err := s.planPost(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	after := cloneInvoices(plan.invoices)
	for idx, inv := range after {
		applyPostedApplication(inv, plan.entity.Applications[idx])
	}

	return &serviceports.CustomerPaymentPostPreview{
		Payment:        plan.entity,
		InvoicesBefore: plan.invoices,
		InvoicesAfter:  after,
	}, nil
}

type postPlan struct {
	control  *tenant.AccountingControl
	entity   *customerpayment.Payment
	invoices []*invoice.Invoice
	period   *fiscalperiod.FiscalPeriod
}

// planPost builds the payment a post would record and validates it against
// the accounting controls, the open period and the invoices it applies to.
func (s *Service) planPost(
	ctx context.Context,
	req *serviceports.PostCustomerPaymentRequest,
	actor *serviceports.RequestActor,
) (*postPlan, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Customer payment posting requires an authenticated user",
		)
	}
	control, err := s.accountingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	entity := &customerpayment.Payment{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		CustomerID:      req.CustomerID,
		PaymentDate:     req.PaymentDate,
		AccountingDate:  req.AccountingDate,
		AmountMinor:     req.AmountMinor,
		Status:          customerpayment.StatusPosted,
		PaymentMethod:   req.PaymentMethod,
		ReferenceNumber: req.ReferenceNumber,
		Memo:            req.Memo,
		CurrencyCode:    req.CurrencyCode,
		CreatedByID:     actor.UserID,
		UpdatedByID:     actor.UserID,
		Applications:    mapApplications(req.Applications),
	}
	if err = s.stamper.StampInto(ctx, &exchangeratestamp.Request{
		TenantInfo:     req.TenantInfo,
		CurrencyCode:   entity.CurrencyCode,
		DocumentDate:   entity.PaymentDate,
		AccountingDate: entity.AccountingDate,
	}, &entity.ExchangeRate, &entity.ExchangeRateDate); err != nil {
		return nil, err
	}

	invoices, period, me := s.validator.ValidatePostAndApply(
		ctx,
		entity,
		repositories.GetInvoiceByIDRequest{TenantInfo: req.TenantInfo},
		control,
	)
	if me != nil {
		return nil, me
	}

	return &postPlan{control: control, entity: entity, invoices: invoices, period: period}, nil
}

// applyPostedApplication settles one application on its invoice: the amount
// applied and the amount written off as a short pay both close the balance.
func applyPostedApplication(inv *invoice.Invoice, app *customerpayment.Application) {
	inv.ApplyPaymentMinor(app.AppliedAmountMinor + app.ShortPayAmountMinor)
}
