package accountingsync

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
	case objectType == SyncObjectJournalEntry:
		return c.SendsLedger() && !c.SumsByDay()
	case objectType == SyncObjectJournalSummary:
		return c.SendsLedger()
	default:
		return !c.SendsLedger()
	}
}

func (c *AccountingConnection) Backfills(objectType SyncObjectType) bool {
	if objectType == SyncObjectJournalSummary {
		return c.SumsByDay()
	}
	return c.Sends(objectType)
}

func (c *AccountingConnection) ChooseMode(mode SyncMode, granularity LedgerGranularity) error {
	if c.SyncEnabledAt != nil || c.SetupStep == SetupStepComplete {
		return ErrModeFixed
	}
	if !mode.IsValid() {
		return ErrModeNotRecognized
	}
	if profile, ok := Profile(c.IntegrationType); ok && !profile.SupportsMode(mode) {
		return ErrModeUnavailable
	}
	if mode == SyncModeLedger && !granularity.IsValid() {
		return ErrGranularityRequired
	}
	previousMode, previousGranularity := c.Mode(), c.Granularity()
	switch mode {
	case SyncModeLedger:
		c.LedgerGranularity = granularity
	case SyncModeDocument:
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

func JournalSendable(entryType, reversesType journalentry.EntryType) bool {
	return entryType != "" && !ClosesFiscalYear(entryType) && !ClosesFiscalYear(reversesType)
}

func ClosesFiscalYear(entryType journalentry.EntryType) bool {
	return entryType == journalentry.EntryTypeClosing || entryType == journalentry.EntryTypeOpening
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
	return JournalDayPrefix + dayKey(day, loc)
}

func JournalOpeningID(startDate int64, loc *time.Location) string {
	return JournalOpeningPrefix + dayKey(startDate, loc)
}

func dayKey(day int64, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return time.Unix(day, 0).In(loc).Format(journalDayLayout)
}

func (r *AccountingSyncRecord) DayUpdate(revision int64) *AccountingSyncRecord {
	return NewAccountingSyncRecord(&NewSyncRecord{
		TenantInfo:   pagination.TenantInfo{OrgID: r.OrganizationID, BuID: r.BusinessUnitID},
		ConnectionID: r.ConnectionID,
		Key: SyncRecordKey{
			ObjectType: r.ObjectType,
			ObjectID:   r.ObjectID,
			Operation:  SyncOperationUpdate,
			Revision:   revision,
		},
		ObjectNumber: r.ObjectNumber,
		SourceEvent:  r.SourceEvent,
		DocumentDate: r.DocumentDate,
		AwaitRelease: r.Status == SyncStatusAwaitingApproval,
		At:           r.QueuedAt,
	})
}

func (r *AccountingSyncRecord) IsOpeningBalances() bool {
	return r.ObjectType == SyncObjectJournalSummary &&
		strings.HasPrefix(r.ObjectID.String(), JournalOpeningPrefix)
}

func LedgerAccountRole(accountID pulid.ID, control *tenant.AccountingControl) string {
	if control == nil || accountID.IsNil() {
		return ""
	}
	switch accountID {
	case control.DefaultARAccountID:
		return AccountRoleAR
	case control.DefaultRevenueAccountID:
		return AccountRoleRevenue
	case control.DefaultCashAccountID:
		return AccountRoleDeposit
	case control.DefaultWriteOffAccountID:
		return AccountRoleWriteOff
	case control.DefaultAPAccountID, control.DefaultSettlementsPayableAccountID:
		return AccountRoleAP
	case control.DefaultPurchasedTransportationAccountID:
		return AccountRolePurchasedTransportation
	default:
		return ""
	}
}
