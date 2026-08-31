package services

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ManualJournalLineInput struct {
	GLAccountID  pulid.ID `json:"glAccountId"`
	Description  string   `json:"description"`
	DebitAmount  int64    `json:"debitAmount"`
	CreditAmount int64    `json:"creditAmount"`
	CustomerID   pulid.ID `json:"customerId"`
	LocationID   pulid.ID `json:"locationId"`
}

type CreateManualJournalRequest struct {
	Description    string                    `json:"description"`
	Reason         string                    `json:"reason"`
	AccountingDate int64                     `json:"accountingDate"`
	CurrencyCode   string                    `json:"currencyCode"`
	Lines          []*ManualJournalLineInput `json:"lines"`
	TenantInfo     pagination.TenantInfo     `json:"tenantInfo"`
}

type UpdateManualJournalDraftRequest struct {
	RequestID      pulid.ID                  `json:"requestId"`
	Description    string                    `json:"description"`
	Reason         string                    `json:"reason"`
	AccountingDate int64                     `json:"accountingDate"`
	CurrencyCode   string                    `json:"currencyCode"`
	Lines          []*ManualJournalLineInput `json:"lines"`
	TenantInfo     pagination.TenantInfo     `json:"tenantInfo"`
}

type GetManualJournalRequest struct {
	RequestID  pulid.ID              `json:"requestId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type RejectManualJournalRequest struct {
	RequestID  pulid.ID              `json:"requestId"`
	Reason     string                `json:"reason"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CancelManualJournalRequest struct {
	RequestID  pulid.ID              `json:"requestId"`
	Reason     string                `json:"reason"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}
