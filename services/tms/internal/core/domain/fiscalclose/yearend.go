package fiscalclose

import (
	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/shared/pulid"
)

// EntryKind names the two journal entries a year-end close produces. The closing
// entry empties the income statement into retained earnings; the opening entry
// re-establishes every balance-sheet account in the first period of the next
// fiscal year so the new year starts from the old year's ending position.
type EntryKind string

const (
	EntryKindClosing = EntryKind("Closing")
	EntryKindOpening = EntryKind("Opening")
)

func (k EntryKind) String() string { return string(k) }

// PlanLine is one prospective journal line. Exactly one of DebitMinor and
// CreditMinor carries an amount, matching the journal-line invariant.
type PlanLine struct {
	GLAccountID     pulid.ID             `json:"glAccountId"`
	AccountCode     string               `json:"accountCode"`
	AccountName     string               `json:"accountName"`
	AccountCategory accounttype.Category `json:"accountCategory"`
	DebitMinor      int64                `json:"debitMinor"`
	CreditMinor     int64                `json:"creditMinor"`
	IsRetainedEarn  bool                 `json:"isRetainedEarnings"`
}

// PlanEntry is one prospective journal entry. FiscalPeriodID is nil when the
// close will have to create the period it posts into — the year-end adjusting
// period is created as part of the close itself.
type PlanEntry struct {
	Kind             EntryKind   `json:"kind"`
	FiscalYearID     pulid.ID    `json:"fiscalYearId"`
	FiscalPeriodID   pulid.ID    `json:"fiscalPeriodId"`
	FiscalPeriodName string      `json:"fiscalPeriodName"`
	CreatesPeriod    bool        `json:"createsPeriod"`
	AccountingDate   int64       `json:"accountingDate"`
	Description      string      `json:"description"`
	TotalDebitMinor  int64       `json:"totalDebitMinor"`
	TotalCreditMinor int64       `json:"totalCreditMinor"`
	Lines            []*PlanLine `json:"lines"`
}

func (e *PlanEntry) IsEmpty() bool { return e == nil || len(e.Lines) == 0 }

// SubledgerCheck reconciles a general-ledger control account against the
// subledger that carries its detail. The GL holds one total per control account
// — customer-level balances live in the AR subledger, which is append-only and
// has no year boundary — so the close verifies that the total still agrees with
// the detail rather than carrying customer tags forward into the ledger.
type SubledgerCheck struct {
	Key                   string   `json:"key"`
	Label                 string   `json:"label"`
	GLAccountID           pulid.ID `json:"glAccountId"`
	AccountCode           string   `json:"accountCode"`
	AccountName           string   `json:"accountName"`
	GLBalanceMinor        int64    `json:"glBalanceMinor"`
	SubledgerBalanceMinor int64    `json:"subledgerBalanceMinor"`
	DifferenceMinor       int64    `json:"differenceMinor"`
	ToleranceMinor        int64    `json:"toleranceMinor"`
	Reconciled            bool     `json:"reconciled"`
	// Enforced reports whether a mismatch blocks the close. The check is always
	// computed and always shown; whether it stops the close is the carrier's
	// call, through RequireReconciliationToClose and ReconciliationMode.
	Enforced bool `json:"enforced"`
}

// Plan is the full accounting consequence of closing a fiscal year, computed
// without writing anything. The close endpoint posts it; the preview endpoint
// returns it so a controller can see the numbers before committing.
type Plan struct {
	FiscalYearID              pulid.ID `json:"fiscalYearId"`
	FiscalYearName            string   `json:"fiscalYearName"`
	NextFiscalYearID          pulid.ID `json:"nextFiscalYearId"`
	NextFiscalYearName        string   `json:"nextFiscalYearName"`
	RetainedEarningsAccountID pulid.ID `json:"retainedEarningsAccountId"`
	RetainedEarningsCode      string   `json:"retainedEarningsAccountCode"`
	RetainedEarningsName      string   `json:"retainedEarningsAccountName"`

	RevenueMinor          int64 `json:"revenueMinor"`
	CostOfRevenueMinor    int64 `json:"costOfRevenueMinor"`
	OperatingExpenseMinor int64 `json:"operatingExpenseMinor"`
	NetIncomeMinor        int64 `json:"netIncomeMinor"`

	ClosingEntry    *PlanEntry        `json:"closingEntry"`
	OpeningEntry    *PlanEntry        `json:"openingEntry"`
	SubledgerChecks []*SubledgerCheck `json:"subledgerChecks"`

	Revision int        `json:"revision"`
	CanClose bool       `json:"canClose"`
	Blockers []*Blocker `json:"blockers"`
}

// PostedEntry records what a close actually wrote for one of its two entries.
type PostedEntry struct {
	Kind             EntryKind `json:"kind"`
	JournalEntryID   pulid.ID  `json:"journalEntryId"`
	JournalBatchID   pulid.ID  `json:"journalBatchId"`
	EntryNumber      string    `json:"entryNumber"`
	FiscalYearID     pulid.ID  `json:"fiscalYearId"`
	FiscalPeriodID   pulid.ID  `json:"fiscalPeriodId"`
	TotalDebitMinor  int64     `json:"totalDebitMinor"`
	TotalCreditMinor int64     `json:"totalCreditMinor"`
	LineCount        int       `json:"lineCount"`
}

// PostResult is what a completed close wrote. Entries holds one item per entry
// that carried lines; a year with no activity produces none.
type PostResult struct {
	FiscalYearID   pulid.ID       `json:"fiscalYearId"`
	NetIncomeMinor int64          `json:"netIncomeMinor"`
	Revision       int            `json:"revision"`
	Entries        []*PostedEntry `json:"entries"`
}

// ReversedEntry records one closing-side entry unwound by a reopen.
type ReversedEntry struct {
	Kind              EntryKind `json:"kind"`
	OriginalEntryID   pulid.ID  `json:"originalEntryId"`
	ReversalEntryID   pulid.ID  `json:"reversalEntryId"`
	ReversalEntryNum  string    `json:"reversalEntryNumber"`
	FiscalPeriodID    pulid.ID  `json:"fiscalPeriodId"`
	TotalDebitMinor   int64     `json:"totalDebitMinor"`
	TotalCreditMinor  int64     `json:"totalCreditMinor"`
	ReversedLineCount int       `json:"reversedLineCount"`
}

// ReverseResult is what a reopen wrote.
type ReverseResult struct {
	FiscalYearID pulid.ID         `json:"fiscalYearId"`
	Entries      []*ReversedEntry `json:"entries"`
}
