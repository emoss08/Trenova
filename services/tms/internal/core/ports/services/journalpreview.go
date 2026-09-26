package services

import "github.com/emoss08/trenova/shared/pulid"

type JournalPreview struct {
	AccountingDate   int64
	FiscalPeriodID   pulid.ID
	EntryStatus      string
	RequiresApproval bool
	Lines            []JournalLinePreview
}

type JournalLinePreview struct {
	GLAccountID pulid.ID
	Description string
	DebitMinor  int64
	CreditMinor int64
}
