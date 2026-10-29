package driversettlementjournal

import (
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/seqgen"
)

type Params struct {
	AccountingRepo   tenantports.AccountingControlRepository
	PayCodeRepo      driverpayports.PayCodeRepository
	JournalRepo      journalentryports.JournalPostingRepository
	FiscalPeriodRepo fiscalperiodports.FiscalPeriodRepository
	Generator        seqgen.Generator
}

type Writer struct {
	accountingRepo   tenantports.AccountingControlRepository
	payCodeRepo      driverpayports.PayCodeRepository
	journalRepo      journalentryports.JournalPostingRepository
	fiscalPeriodRepo fiscalperiodports.FiscalPeriodRepository
	generator        seqgen.Generator
}

func New(p *Params) *Writer {
	return &Writer{
		accountingRepo:   p.AccountingRepo,
		payCodeRepo:      p.PayCodeRepo,
		journalRepo:      p.JournalRepo,
		fiscalPeriodRepo: p.FiscalPeriodRepo,
		generator:        p.Generator,
	}
}
