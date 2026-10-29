package factoringsubmissionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetSubmissionRequest struct {
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"-"`
	IncludeItems bool                  `json:"includeItems"`
}

type GetSubmissionsByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	IDs        []pulid.ID            `json:"ids"`
}

type GetSubmissionByKeyRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	IdempotencyKey string                `json:"idempotencyKey"`
}

type ListSubmissionConnectionRequest struct {
	Filter             *pagination.QueryOptions               `json:"filter"`
	Cursor             pagination.CursorInfo                  `json:"-"`
	FactoringCompanyID pulid.ID                               `json:"factoringCompanyId"`
	Statuses           []factoringsubmission.SubmissionStatus `json:"statuses"`
}

type ListLiveItemsRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	InvoiceIDs         []pulid.ID            `json:"invoiceIds"`
	IncludeChargedBack bool                  `json:"includeChargedBack"`
}

type LiveItem struct {
	InvoiceID        pulid.ID                                 `json:"invoiceId"`
	SubmissionID     pulid.ID                                 `json:"submissionId"`
	SubmissionNumber string                                   `json:"submissionNumber"`
	SubmissionStatus factoringsubmission.SubmissionStatus     `json:"submissionStatus"`
	ItemID           pulid.ID                                 `json:"itemId"`
	ItemStatus       factoringsubmission.SubmissionItemStatus `json:"itemStatus"`
}

func (l *LiveItem) IsLive() bool {
	return l != nil && l.ItemStatus.IsLive()
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

type ClaimChargebackWarningRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ItemID     pulid.ID              `json:"itemId"`
	At         int64                 `json:"at"`
}

type SubmissionRepository interface {
	ClaimChargebackWarning(ctx context.Context, req ClaimChargebackWarningRequest) (bool, error)
	Create(
		ctx context.Context,
		entity *factoringsubmission.Submission,
	) (*factoringsubmission.Submission, error)
	Update(
		ctx context.Context,
		entity *factoringsubmission.Submission,
	) (*factoringsubmission.Submission, error)
	SetDocuments(ctx context.Context, req *SetSubmissionDocumentsRequest) error
	GetByID(ctx context.Context, req GetSubmissionRequest) (*factoringsubmission.Submission, error)
	GetByIDs(
		ctx context.Context,
		req GetSubmissionsByIDsRequest,
	) ([]*factoringsubmission.Submission, error)
	GetByIdempotencyKey(
		ctx context.Context,
		req GetSubmissionByKeyRequest,
	) (*factoringsubmission.Submission, error)
	ListConnection(
		ctx context.Context,
		req *ListSubmissionConnectionRequest,
	) (*pagination.CursorListResult[*factoringsubmission.Submission], error)
	ListLiveItems(ctx context.Context, req *ListLiveItemsRequest) (map[pulid.ID]*LiveItem, error)
	ListItemsBySubmissionIDs(
		ctx context.Context,
		req *ListItemsBySubmissionIDsRequest,
	) (map[pulid.ID][]*factoringsubmission.SubmissionItem, error)
}

type GetTransportRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
}

type TransportRepository interface {
	Get(ctx context.Context, req GetTransportRequest) (*factoringsubmission.Transport, error)
	Create(
		ctx context.Context,
		entity *factoringsubmission.Transport,
	) (*factoringsubmission.Transport, error)
	Update(
		ctx context.Context,
		entity *factoringsubmission.Transport,
	) (*factoringsubmission.Transport, error)
	RecordTest(ctx context.Context, entity *factoringsubmission.Transport) error
}
