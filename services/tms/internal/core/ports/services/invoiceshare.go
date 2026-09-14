package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ShareInvoiceRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceID  pulid.ID              `json:"-"`
	UserIDs    []pulid.ID            `json:"userIds"`
	Note       string                `json:"note"`
	Tab        invoice.ShareTab      `json:"tab"`
}

type ShareCandidate struct {
	ID            pulid.ID `json:"id"`
	Name          string   `json:"name"`
	Username      string   `json:"username"`
	EmailAddress  string   `json:"emailAddress"`
	ProfilePicURL string   `json:"profilePicUrl"`
	ThumbnailURL  string   `json:"thumbnailUrl"`
}

type ListShareCandidatesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceID  pulid.ID              `json:"-"`
	Query      string                `json:"query"`
	Pagination pagination.Info       `json:"pagination"`
}

type GetShareCandidateRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceID  pulid.ID              `json:"-"`
	UserID     pulid.ID              `json:"-"`
}

type InvoiceShareEmailStatus string

const (
	InvoiceShareEmailQueued        = InvoiceShareEmailStatus("Queued")
	InvoiceShareEmailPartial       = InvoiceShareEmailStatus("Partial")
	InvoiceShareEmailFailed        = InvoiceShareEmailStatus("Failed")
	InvoiceShareEmailNotConfigured = InvoiceShareEmailStatus("NotConfigured")
)

type ShareInvoiceResult struct {
	Shares         []*invoice.InvoiceShare `json:"shares"`
	RecipientCount int                     `json:"recipientCount"`
	EmailsQueued   int                     `json:"emailsQueued"`
	EmailStatus    InvoiceShareEmailStatus `json:"emailStatus"`
}

type InvoiceShareService interface {
	List(
		ctx context.Context,
		req *repositories.ListInvoiceSharesRequest,
	) ([]*invoice.InvoiceShare, error)
	Share(
		ctx context.Context,
		req *ShareInvoiceRequest,
		actor *RequestActor,
	) (*ShareInvoiceResult, error)
	ListCandidates(
		ctx context.Context,
		req *ListShareCandidatesRequest,
	) (*pagination.ListResult[*ShareCandidate], error)
	GetCandidate(ctx context.Context, req *GetShareCandidateRequest) (*ShareCandidate, error)
}
