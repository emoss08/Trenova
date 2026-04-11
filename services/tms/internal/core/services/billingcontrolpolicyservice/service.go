package billingcontrolpolicyservice

import (
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const AutoPostInvoiceTrigger = "auto-post-workflow"

type Params struct {
	fx.In

	Logger *zap.Logger
}

type Service struct{ l *zap.Logger }

func New(p Params) *Service {
	return &Service{l: p.Logger.Named("service.billing-control-policy")}
}

func (s *Service) CanAutoCreateInvoiceDraft(control *tenant.BillingControl) bool {
	return control != nil && control.InvoiceDraftCreationMode == tenant.InvoiceDraftCreationModeAutomaticWhenTransferred
}

func (s *Service) CanAutoPostInvoice(control *tenant.BillingControl, cus *customer.Customer) bool {
	if control == nil {
		return false
	}

	if !s.CanAutoCreateInvoiceDraft(control) {
		return false
	}

	if control.InvoicePostingMode != tenant.InvoicePostingModeAutomaticWhenNoBlockingExceptions {
		return false
	}

	if cus == nil || cus.BillingProfile == nil {
		return true
	}

	return cus.BillingProfile.AutoBill
}

func (s *Service) ValidateInvoicePosting(control *tenant.BillingControl, triggeredBy string) error {
	if triggeredBy != AutoPostInvoiceTrigger {
		return nil
	}

	if control == nil {
		return errortypes.NewBusinessError("Billing control is required for invoice auto-posting")
	}

	if control.InvoicePostingMode != tenant.InvoicePostingModeAutomaticWhenNoBlockingExceptions {
		return errortypes.NewBusinessError("Invoice auto-posting is disabled by billing control")
	}

	if !s.CanAutoCreateInvoiceDraft(control) {
		return errortypes.NewBusinessError("Invoice auto-posting requires automatic invoice draft creation mode")
	}

	return nil
}
