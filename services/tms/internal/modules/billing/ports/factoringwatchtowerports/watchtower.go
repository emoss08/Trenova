package factoringwatchtowerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListInvoiceLabelsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceIDs []pulid.ID            `json:"invoiceIds"`
}

type InvoiceLabel struct {
	InvoiceID    pulid.ID `json:"invoiceId"`
	Number       string   `json:"number"`
	CurrencyCode string   `json:"currencyCode"`
	CustomerName string   `json:"customerName"`
	FactorName   string   `json:"factorName"`
}

type ListCustomersMissingNoaRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	At         int64                 `json:"at"`
	Limit      int                   `json:"limit"`
}

type CustomerMissingNoa struct {
	CustomerID         pulid.ID            `json:"customerId"`
	CustomerCode       string              `json:"customerCode"`
	CustomerName       string              `json:"customerName"`
	FactoringCompanyID pulid.ID            `json:"factoringCompanyId"`
	FactorName         string              `json:"factorName"`
	ProfileUpdatedAt   int64               `json:"profileUpdatedAt"`
	NoaID              pulid.ID            `json:"noaId"`
	NoaStatus          factoring.NoaStatus `json:"noaStatus"`
	NoaSentAt          *int64              `json:"noaSentAt"`
	NoaExpiresAt       *int64              `json:"noaExpiresAt"`
	NoaCreatedAt       *int64              `json:"noaCreatedAt"`
}

type ListNoasExpiringSoonRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	At         int64                 `json:"at"`
	Before     int64                 `json:"before"`
	Limit      int                   `json:"limit"`
}

type ExpiringNoa struct {
	NoaID              pulid.ID `json:"noaId"`
	CustomerID         pulid.ID `json:"customerId"`
	CustomerName       string   `json:"customerName"`
	FactoringCompanyID pulid.ID `json:"factoringCompanyId"`
	FactorName         string   `json:"factorName"`
	AcknowledgedAt     *int64   `json:"acknowledgedAt"`
	ExpiresAt          int64    `json:"expiresAt"`
}

type ListFailedSubmissionsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Limit      int                   `json:"limit"`
}

type FailedSubmission struct {
	SubmissionID       pulid.ID `json:"submissionId"`
	Number             string   `json:"number"`
	FactoringCompanyID pulid.ID `json:"factoringCompanyId"`
	FactorName         string   `json:"factorName"`
	CurrencyCode       string   `json:"currencyCode"`
	InvoiceCount       int      `json:"invoiceCount"`
	FaceMinor          int64    `json:"faceMinor"`
	TransportTarget    string   `json:"transportTarget"`
	LastError          string   `json:"lastError"`
	FailedAt           int64    `json:"failedAt"`
}

type ListUnfundedSubmissionsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	At         int64                 `json:"at"`
	Limit      int                   `json:"limit"`
}

type UnfundedSubmission struct {
	SubmissionID         pulid.ID                             `json:"submissionId"`
	Number               string                               `json:"number"`
	Status               factoringsubmission.SubmissionStatus `json:"status"`
	FactoringCompanyID   pulid.ID                             `json:"factoringCompanyId"`
	FactorName           string                               `json:"factorName"`
	CurrencyCode         string                               `json:"currencyCode"`
	PendingCount         int                                  `json:"pendingCount"`
	ExpectedAdvanceMinor int64                                `json:"expectedAdvanceMinor"`
	SentAt               int64                                `json:"sentAt"`
	DueAt                int64                                `json:"dueAt"`
}

type ListChargebacksDueRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	Before       int64                 `json:"before"`
	UnwarnedOnly bool                  `json:"unwarnedOnly"`
	Limit        int                   `json:"limit"`
}

type ChargebackDue struct {
	ItemID             pulid.ID `json:"itemId"`
	SubmissionID       pulid.ID `json:"submissionId"`
	SubmissionNumber   string   `json:"submissionNumber"`
	InvoiceID          pulid.ID `json:"invoiceId"`
	InvoiceNumber      string   `json:"invoiceNumber"`
	CustomerID         pulid.ID `json:"customerId"`
	CustomerName       string   `json:"customerName"`
	FactoringCompanyID pulid.ID `json:"factoringCompanyId"`
	FactorName         string   `json:"factorName"`
	CurrencyCode       string   `json:"currencyCode"`
	FaceMinor          int64    `json:"faceMinor"`
	RepurchaseMinor    int64    `json:"repurchaseMinor"`
	FundedOn           int64    `json:"fundedOn"`
	DeadlineAt         int64    `json:"deadlineAt"`
}

type WatchtowerReader interface {
	ListUnfundedSubmissions(
		ctx context.Context,
		req *ListUnfundedSubmissionsRequest,
	) ([]*UnfundedSubmission, error)
	ListChargebacksDue(
		ctx context.Context,
		req *ListChargebacksDueRequest,
	) ([]*ChargebackDue, error)
	ListFailedSubmissions(
		ctx context.Context,
		req *ListFailedSubmissionsRequest,
	) ([]*FailedSubmission, error)
	ListInvoiceLabels(
		ctx context.Context,
		req *ListInvoiceLabelsRequest,
	) (map[pulid.ID]*InvoiceLabel, error)
	ListCustomersMissingNoa(
		ctx context.Context,
		req *ListCustomersMissingNoaRequest,
	) ([]*CustomerMissingNoa, error)
	ListNoasExpiringSoon(
		ctx context.Context,
		req *ListNoasExpiringSoonRequest,
	) ([]*ExpiringNoa, error)
}
