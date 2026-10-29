package carriersettlementjournal

import (
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/seqgen"
)

type Params struct {
	AccountingRepo    tenantports.AccountingControlRepository
	SettlementControl tenantports.CarrierSettlementControlRepository
	JournalRepo       journalentryports.JournalPostingRepository
	FiscalPeriodRepo  fiscalperiodports.FiscalPeriodRepository
	Generator         seqgen.Generator
}

type Writer struct {
	accountingRepo    tenantports.AccountingControlRepository
	settlementControl tenantports.CarrierSettlementControlRepository
	journalRepo       journalentryports.JournalPostingRepository
	fiscalPeriodRepo  fiscalperiodports.FiscalPeriodRepository
	generator         seqgen.Generator
}

func New(p *Params) *Writer {
	return &Writer{
		accountingRepo:    p.AccountingRepo,
		settlementControl: p.SettlementControl,
		journalRepo:       p.JournalRepo,
		fiscalPeriodRepo:  p.FiscalPeriodRepo,
		generator:         p.Generator,
	}
}
