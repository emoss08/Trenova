package accountingcontrolpolicyservice

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
}

type Service struct{ l *zap.Logger }

type ResolveFXQuoteDateRequest struct {
	Policy         accountingcontrol.ExchangeRateDatePolicy
	DocumentDate   int64
	AccountingDate int64
}

func New(p Params) *Service {
	return &Service{l: p.Logger.Named("service.accounting-control-policy")}
}

func (s *Service) CanCreateInvoiceLedgerEntry(
	control *accountingcontrol.AccountingControl,
	event accountingcontrol.JournalSourceEventType,
) bool {
	if control == nil {
		return false
	}

	if control.AccountingBasis == accountingcontrol.AccountingBasisCash {
		return false
	}

	if control.RevenueRecognitionPolicy != accountingcontrol.RevenueRecognitionOnInvoicePost {
		return false
	}

	switch event {
	case accountingcontrol.JournalSourceEventInvoicePosted,
		accountingcontrol.JournalSourceEventCreditMemoPosted,
		accountingcontrol.JournalSourceEventDebitMemoPosted:
		return true
	case accountingcontrol.JournalSourceEventCustomerPaymentPosted,
		accountingcontrol.JournalSourceEventCustomerShortPayRecognized,
		accountingcontrol.JournalSourceEventCustomerPaymentReversed,
		accountingcontrol.JournalSourceEventVendorBillPosted,
		accountingcontrol.JournalSourceEventVendorPaymentPosted:
		return false
	default:
		return false
	}
}

func (s *Service) CanUseAutomaticSourcePosting(
	control *accountingcontrol.AccountingControl,
	event accountingcontrol.JournalSourceEventType,
) bool {
	if control == nil ||
		control.JournalPostingMode != accountingcontrol.JournalPostingModeAutomatic {
		return false
	}

	if !s.CanCreateInvoiceLedgerEntry(control, event) {
		return false
	}

	return slices.Contains(control.AutoPostSourceEvents, event)
}

func (s *Service) ValidateManualPeriodClose(control *accountingcontrol.AccountingControl) error {
	if control == nil {
		return nil
	}

	if control.PeriodCloseMode == accountingcontrol.PeriodCloseModeSystemScheduled {
		return errortypes.NewBusinessError(
			"Fiscal periods configured for system-scheduled close cannot be closed manually",
		)
	}

	if control.RequirePeriodCloseApproval {
		return errortypes.NewBusinessError(
			"Fiscal period close approval is required before the period can be closed manually",
		)
	}

	return nil
}

func (s *Service) ResolveFXQuoteDate(req ResolveFXQuoteDateRequest) (int64, error) {
	switch req.Policy {
	case accountingcontrol.ExchangeRateDatePolicyDocumentDate:
		return req.DocumentDate, nil
	case accountingcontrol.ExchangeRateDatePolicyAccountingDate:
		return req.AccountingDate, nil
	default:
		return 0, errortypes.NewBusinessError("unsupported exchange rate date policy")
	}
}
