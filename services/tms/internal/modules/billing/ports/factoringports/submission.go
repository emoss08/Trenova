package factoringports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetSubmissionRequest struct {
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"-"`
	IncludeItems bool                  `json:"includeItems"`
}

type GetSubmissionByKeyRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	IdempotencyKey string                `json:"idempotencyKey"`
}

type ListSubmissionConnectionRequest struct {
	Filter             *pagination.QueryOptions     `json:"filter"`
	Cursor             pagination.CursorInfo        `json:"-"`
	FactoringCompanyID pulid.ID                     `json:"factoringCompanyId"`
	Statuses           []factoring.SubmissionStatus `json:"statuses"`
}

type ListLiveItemsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InvoiceIDs []pulid.ID            `json:"invoiceIds"`
}

type LiveItem struct {
	InvoiceID        pulid.ID                   `json:"invoiceId"        bun:"invoice_id"`
	SubmissionID     pulid.ID                   `json:"submissionId"     bun:"submission_id"`
	SubmissionNumber string                     `json:"submissionNumber" bun:"submission_number"`
	SubmissionStatus factoring.SubmissionStatus `json:"submissionStatus" bun:"submission_status"`
}

type ListItemsBySubmissionIDsRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	SubmissionIDs []pulid.ID            `json:"submissionIds"`
}

type SetSubmissionDocumentsRequest struct {
	TenantInfo            pagination.TenantInfo `json:"-"`
	SubmissionID          pulid.ID              `json:"submissionId"`
	SchedulePDFDocumentID pulid.ID              `json:"schedulePdfDocumentId"`
	ScheduleCSVDocumentID pulid.ID              `json:"scheduleCsvDocumentId"`
	Packets               map[pulid.ID]pulid.ID `json:"packets"`
}

type SubmissionRepository interface {
	Create(ctx context.Context, entity *factoring.Submission) (*factoring.Submission, error)
	Update(ctx context.Context, entity *factoring.Submission) (*factoring.Submission, error)
	SetDocuments(ctx context.Context, req *SetSubmissionDocumentsRequest) error
	GetByID(ctx context.Context, req GetSubmissionRequest) (*factoring.Submission, error)
	GetByIdempotencyKey(
		ctx context.Context,
		req GetSubmissionByKeyRequest,
	) (*factoring.Submission, error)
	ListConnection(
		ctx context.Context,
		req *ListSubmissionConnectionRequest,
	) (*pagination.CursorListResult[*factoring.Submission], error)
	ListLiveItems(ctx context.Context, req *ListLiveItemsRequest) (map[pulid.ID]*LiveItem, error)
	ListItemsBySubmissionIDs(
		ctx context.Context,
		req *ListItemsBySubmissionIDsRequest,
	) (map[pulid.ID][]*factoring.SubmissionItem, error)
}

type GetTransportRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
}

type TransportRepository interface {
	Get(ctx context.Context, req GetTransportRequest) (*factoring.Transport, error)
	Create(ctx context.Context, entity *factoring.Transport) (*factoring.Transport, error)
	Update(ctx context.Context, entity *factoring.Transport) (*factoring.Transport, error)
	RecordTest(ctx context.Context, entity *factoring.Transport) error
}
