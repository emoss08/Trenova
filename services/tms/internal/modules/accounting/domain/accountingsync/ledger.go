package accountingsync

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func JournalSendable(entryType, reversesType journalentry.EntryType) bool {
	return entryType != "" && !ClosesFiscalYear(entryType) && !ClosesFiscalYear(reversesType)
}

func ClosesFiscalYear(entryType journalentry.EntryType) bool {
	return entryType == journalentry.EntryTypeClosing || entryType == journalentry.EntryTypeOpening
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
