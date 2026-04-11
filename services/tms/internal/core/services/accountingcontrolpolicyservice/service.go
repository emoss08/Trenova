package accountingcontrolpolicyservice

import (
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
}

type Service struct{ l *zap.Logger }

func New(p Params) *Service {
	return &Service{l: p.Logger.Named("service.accounting-control-policy")}
}

func (s *Service) CanCreateInvoiceLedgerEntry(
	control *tenant.AccountingControl,
	event tenant.JournalSourceEventType,
) bool {
	if control == nil {
		return false
	}

	if control.AccountingBasis == tenant.AccountingBasisCash {
		return false
	}

	if control.RevenueRecognitionPolicy != tenant.RevenueRecognitionOnInvoicePost {
		return false
	}

	switch event {
	case tenant.JournalSourceEventInvoicePosted,
		tenant.JournalSourceEventCreditMemoPosted,
		tenant.JournalSourceEventDebitMemoPosted:
		return true
	default:
		return false
	}
}

func (s *Service) CanUseAutomaticSourcePosting(
	control *tenant.AccountingControl,
	event tenant.JournalSourceEventType,
) bool {
	if control == nil || control.JournalPostingMode != tenant.JournalPostingModeAutomatic {
		return false
	}

	if !s.CanCreateInvoiceLedgerEntry(control, event) {
		return false
	}

	for _, configured := range control.AutoPostSourceEvents {
		if configured == event {
			return true
		}
	}

	return false
}

func (s *Service) ValidateManualPeriodClose(control *tenant.AccountingControl) error {
	if control == nil {
		return nil
	}

	if control.PeriodCloseMode == tenant.PeriodCloseModeSystemScheduled {
		return errortypes.NewBusinessError("Fiscal periods configured for system-scheduled close cannot be closed manually")
	}

	if control.RequirePeriodCloseApproval {
		return errortypes.NewBusinessError("Fiscal period close approval is required before the period can be closed manually")
	}

	return nil
}
