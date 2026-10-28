package accountingconnection

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
)

func (c *AccountingConnection) Mode() accountingprovider.SyncMode {
	if c.SyncMode.IsValid() {
		return c.SyncMode
	}
	return accountingprovider.SyncModeDocument
}

func (c *AccountingConnection) SendsLedger() bool {
	return c.Mode() == accountingprovider.SyncModeLedger
}

func (c *AccountingConnection) Granularity() LedgerGranularity {
	if !c.SendsLedger() {
		return ""
	}
	if c.LedgerGranularity.IsValid() {
		return c.LedgerGranularity
	}
	return LedgerDetailed
}

func (c *AccountingConnection) SumsByDay() bool {
	return c.Granularity() == LedgerDailySummary
}

func (c *AccountingConnection) Sends(objectType accountingsync.SyncObjectType) bool {
	switch {
	case objectType.IsParty():
		return true
	case objectType == accountingsync.SyncObjectJournalEntry:
		return c.SendsLedger() && !c.SumsByDay()
	case objectType == accountingsync.SyncObjectJournalSummary:
		return c.SendsLedger()
	default:
		return !c.SendsLedger()
	}
}

func (c *AccountingConnection) Backfills(objectType accountingsync.SyncObjectType) bool {
	if objectType == accountingsync.SyncObjectJournalSummary {
		return c.SumsByDay()
	}
	return c.Sends(objectType)
}

func (c *AccountingConnection) ChooseMode(
	mode accountingprovider.SyncMode,
	granularity LedgerGranularity,
) error {
	if c.SyncEnabledAt != nil || c.SetupStep == SetupStepComplete {
		return ErrModeFixed
	}
	if !mode.IsValid() {
		return ErrModeNotRecognized
	}
	if profile, ok := accountingprovider.Profile(c.IntegrationType); ok &&
		!profile.SupportsMode(mode) {
		return ErrModeUnavailable
	}
	if mode == accountingprovider.SyncModeLedger && !granularity.IsValid() {
		return ErrGranularityRequired
	}
	previousMode, previousGranularity := c.Mode(), c.Granularity()
	switch mode {
	case accountingprovider.SyncModeLedger:
		c.LedgerGranularity = granularity
	case accountingprovider.SyncModeDocument:
		c.LedgerGranularity = ""
	}
	c.SyncMode = mode
	changed := c.Mode() != previousMode || c.Granularity() != previousGranularity
	if changed || c.SetupStep == SetupStepMode || c.SetupStep == "" {
		c.SetupStep = SetupStepMappings
	}
	return nil
}

func (c *AccountingConnection) CanMoveStartDate(startDate int64) error {
	if c.SentOpeningBalances() && c.SyncStartDate != nil && *c.SyncStartDate != startDate {
		return ErrStartDateFixed
	}
	return nil
}

func (c *AccountingConnection) SentOpeningBalances() bool {
	return c.LedgerOpeningBalancesSentAt != nil
}

func (c *AccountingConnection) FiscalYearStart(asOf int64, loc *time.Location) int64 {
	if loc == nil {
		loc = time.UTC
	}
	startMonth := time.Month(c.ExternalFiscalYearStartMonth)
	if startMonth < time.January || startMonth > time.December {
		startMonth = time.January
	}
	year, month, _ := time.Unix(asOf, 0).In(loc).Date()
	if month < startMonth {
		year--
	}
	return time.Date(year, startMonth, 1, 0, 0, 0, 0, loc).Unix()
}
