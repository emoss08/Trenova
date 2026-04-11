package journalentry

import "github.com/emoss08/trenova/shared/pulid"

type Entry struct {
	ID             pulid.ID `json:"id"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	BatchID        pulid.ID `json:"batchId"`
	FiscalYearID   pulid.ID `json:"fiscalYearId"`
	FiscalPeriodID pulid.ID `json:"fiscalPeriodId"`
	EntryNumber    string   `json:"entryNumber"`
	EntryType      string   `json:"entryType"`
	Status         string   `json:"status"`
	AccountingDate int64    `json:"accountingDate"`
	Description    string   `json:"description"`
	ReferenceType  string   `json:"referenceType"`
	ReferenceID    string   `json:"referenceId"`
	TotalDebit     int64    `json:"totalDebit"`
	TotalCredit    int64    `json:"totalCredit"`
	IsPosted       bool     `json:"isPosted"`
	IsReversal     bool     `json:"isReversal"`
	ReversalOfID   pulid.ID `json:"reversalOfId"`
	ReversedByID   pulid.ID `json:"reversedById"`
	ReversalDate   *int64   `json:"reversalDate"`
	ReversalReason string   `json:"reversalReason"`
	Lines          []*Line  `json:"lines,omitempty"`
}

type Line struct {
	ID             pulid.ID `json:"id"`
	JournalEntryID pulid.ID `json:"journalEntryId"`
	GLAccountID    pulid.ID `json:"glAccountId"`
	LineNumber     int16    `json:"lineNumber"`
	Description    string   `json:"description"`
	DebitAmount    int64    `json:"debitAmount"`
	CreditAmount   int64    `json:"creditAmount"`
	NetAmount      int64    `json:"netAmount"`
	CustomerID     pulid.ID `json:"customerId"`
	LocationID     pulid.ID `json:"locationId"`
}
