package factoringports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListInvoiceLabelsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceIDs []pulid.ID            `json:"invoiceIds"`
}

type InvoiceLabel struct {
	InvoiceID    pulid.ID `json:"invoiceId"    bun:"invoice_id"`
	Number       string   `json:"number"       bun:"number"`
	CurrencyCode string   `json:"currencyCode" bun:"currency_code"`
	CustomerName string   `json:"customerName" bun:"customer_name"`
	FactorName   string   `json:"factorName"   bun:"factor_name"`
}

type ListCustomersMissingNoaRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	At         int64                 `json:"at"`
	Limit      int                   `json:"limit"`
}

type CustomerMissingNoa struct {
	CustomerID         pulid.ID            `json:"customerId"         bun:"customer_id"`
	CustomerCode       string              `json:"customerCode"       bun:"customer_code"`
	CustomerName       string              `json:"customerName"       bun:"customer_name"`
	FactoringCompanyID pulid.ID            `json:"factoringCompanyId" bun:"factoring_company_id"`
	FactorName         string              `json:"factorName"         bun:"factor_name"`
	ProfileUpdatedAt   int64               `json:"profileUpdatedAt"   bun:"profile_updated_at"`
	NoaID              pulid.ID            `json:"noaId"              bun:"noa_id"`
	NoaStatus          factoring.NoaStatus `json:"noaStatus"          bun:"noa_status"`
	NoaSentAt          *int64              `json:"noaSentAt"          bun:"noa_sent_at"`
	NoaExpiresAt       *int64              `json:"noaExpiresAt"       bun:"noa_expires_at"`
	NoaCreatedAt       *int64              `json:"noaCreatedAt"       bun:"noa_created_at"`
}

type ListNoasExpiringSoonRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	At         int64                 `json:"at"`
	Before     int64                 `json:"before"`
	Limit      int                   `json:"limit"`
}

type ExpiringNoa struct {
	NoaID              pulid.ID `json:"noaId"              bun:"noa_id"`
	CustomerID         pulid.ID `json:"customerId"         bun:"customer_id"`
	CustomerName       string   `json:"customerName"       bun:"customer_name"`
	FactoringCompanyID pulid.ID `json:"factoringCompanyId" bun:"factoring_company_id"`
	FactorName         string   `json:"factorName"         bun:"factor_name"`
	AcknowledgedAt     *int64   `json:"acknowledgedAt"     bun:"acknowledged_at"`
	ExpiresAt          int64    `json:"expiresAt"          bun:"expires_at"`
}

type ListFailedSubmissionsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Limit      int                   `json:"limit"`
}

type FailedSubmission struct {
	SubmissionID       pulid.ID `json:"submissionId"       bun:"submission_id"`
	Number             string   `json:"number"             bun:"number"`
	FactoringCompanyID pulid.ID `json:"factoringCompanyId" bun:"factoring_company_id"`
	FactorName         string   `json:"factorName"         bun:"factor_name"`
	CurrencyCode       string   `json:"currencyCode"       bun:"currency_code"`
	InvoiceCount       int      `json:"invoiceCount"       bun:"invoice_count"`
	FaceMinor          int64    `json:"faceMinor"          bun:"face_minor"`
	TransportTarget    string   `json:"transportTarget"    bun:"transport_target"`
	LastError          string   `json:"lastError"          bun:"last_error"`
	FailedAt           int64    `json:"failedAt"           bun:"failed_at"`
}

type ListUnfundedSubmissionsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	At         int64                 `json:"at"`
	Limit      int                   `json:"limit"`
}

type UnfundedSubmission struct {
	SubmissionID         pulid.ID                   `json:"submissionId"         bun:"submission_id"`
	Number               string                     `json:"number"               bun:"number"`
	Status               factoring.SubmissionStatus `json:"status"               bun:"status"`
	FactoringCompanyID   pulid.ID                   `json:"factoringCompanyId"   bun:"factoring_company_id"`
	FactorName           string                     `json:"factorName"           bun:"factor_name"`
	CurrencyCode         string                     `json:"currencyCode"         bun:"currency_code"`
	PendingCount         int                        `json:"pendingCount"         bun:"pending_count"`
	ExpectedAdvanceMinor int64                      `json:"expectedAdvanceMinor" bun:"expected_advance_minor"`
	SentAt               int64                      `json:"sentAt"               bun:"sent_at"`
	DueAt                int64                      `json:"dueAt"                bun:"due_at"`
}

type ListChargebacksDueRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	Before       int64                 `json:"before"`
	UnwarnedOnly bool                  `json:"unwarnedOnly"`
	Limit        int                   `json:"limit"`
}

type ChargebackDue struct {
	ItemID             pulid.ID `json:"itemId"             bun:"item_id"`
	SubmissionID       pulid.ID `json:"submissionId"       bun:"submission_id"`
	SubmissionNumber   string   `json:"submissionNumber"   bun:"submission_number"`
	InvoiceID          pulid.ID `json:"invoiceId"          bun:"invoice_id"`
	InvoiceNumber      string   `json:"invoiceNumber"      bun:"invoice_number"`
	CustomerID         pulid.ID `json:"customerId"         bun:"customer_id"`
	CustomerName       string   `json:"customerName"       bun:"customer_name"`
	FactoringCompanyID pulid.ID `json:"factoringCompanyId" bun:"factoring_company_id"`
	FactorName         string   `json:"factorName"         bun:"factor_name"`
	CurrencyCode       string   `json:"currencyCode"       bun:"currency_code"`
	FaceMinor          int64    `json:"faceMinor"          bun:"face_minor"`
	RepurchaseMinor    int64    `json:"repurchaseMinor"    bun:"repurchase_minor"`
	FundedOn           int64    `json:"fundedOn"           bun:"funded_on"`
	DeadlineAt         int64    `json:"deadlineAt"         bun:"deadline_at"`
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
