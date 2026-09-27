package accountingsync

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/shared/timeutils"
)

type SyncMode string

const (
	SyncModeDocument = SyncMode("Document")
	SyncModeLedger   = SyncMode("Ledger")
)

func (m SyncMode) String() string { return string(m) }

func (m SyncMode) IsValid() bool {
	switch m {
	case SyncModeDocument, SyncModeLedger:
		return true
	default:
		return false
	}
}

func AllSyncModes() []SyncMode {
	return []SyncMode{SyncModeDocument, SyncModeLedger}
}

type LedgerGranularity string

const (
	LedgerDetailed     = LedgerGranularity("Detailed")
	LedgerDailySummary = LedgerGranularity("DailySummary")
)

func (g LedgerGranularity) String() string { return string(g) }

func (g LedgerGranularity) IsValid() bool {
	switch g {
	case LedgerDetailed, LedgerDailySummary:
		return true
	default:
		return false
	}
}

func AllLedgerGranularities() []LedgerGranularity {
	return []LedgerGranularity{LedgerDetailed, LedgerDailySummary}
}

var (
	ErrModeFixed = errors.New(
		"the mode is fixed once sync is enabled; disconnect and connect again to change it",
	)
	ErrModeNotRecognized   = errors.New("the mode is not recognized")
	ErrGranularityRequired = errors.New(
		"journal entries are sent detailed or as a daily summary",
	)
	ErrOpeningBalancesNeedLedger = errors.New(
		"opening balances are sent only when journal entries are",
	)
)

const (
	journalDayPrefix     = "jday_"
	journalOpeningPrefix = "jopen_"
	journalDayLayout     = "20060102"
)

func (c *AccountingConnection) Mode() SyncMode {
	if c.SyncMode.IsValid() {
		return c.SyncMode
	}
	return SyncModeDocument
}

func (c *AccountingConnection) SendsLedger() bool {
	return c.Mode() == SyncModeLedger
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

func (c *AccountingConnection) Sends(objectType SyncObjectType) bool {
	switch {
	case objectType.IsParty():
		return true
	case objectType.IsLedger():
		return c.SendsLedger()
	default:
		return !c.SendsLedger()
	}
}

func (c *AccountingConnection) ChooseMode(mode SyncMode, granularity LedgerGranularity) error {
	if c.SyncEnabledAt != nil || c.SetupStep == SetupStepComplete {
		return ErrModeFixed
	}
	if !mode.IsValid() {
		return ErrModeNotRecognized
	}
	switch mode {
	case SyncModeLedger:
		if !granularity.IsValid() {
			return ErrGranularityRequired
		}
		c.LedgerGranularity = granularity
	case SyncModeDocument:
		c.LedgerGranularity = ""
	}
	c.SyncMode = mode
	if c.SetupStep == SetupStepMode || c.SetupStep == "" {
		c.SetupStep = SetupStepMappings
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

func JournalDayID(day int64, loc *time.Location) string {
	return journalDayPrefix + dayKey(day, loc)
}

func JournalOpeningID(startDate int64, loc *time.Location) string {
	return journalOpeningPrefix + dayKey(startDate, loc)
}

func JournalDayNumber(day int64, loc *time.Location) string {
	return timeutils.FormatCalendarDate(day, loc)
}

func dayKey(day int64, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return time.Unix(day, 0).In(loc).Format(journalDayLayout)
}
