package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDIInboundFilesRequest struct {
	Filter    *pagination.QueryOptions `json:"filter"`
	Cursor    pagination.CursorInfo    `json:"-"`
	Status    edi.InboundFileStatus    `json:"status"`
	PartnerID pulid.ID                 `json:"partnerId"`
}

type GetEDIInboundFileByIDRequest struct {
	ID              pulid.ID              `json:"id"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	IncludeMessages bool                  `json:"includeMessages"`
}

type ExistsEDIInboundFileByChecksumRequest struct {
	TenantInfo             pagination.TenantInfo `json:"tenantInfo"`
	CommunicationProfileID pulid.ID              `json:"communicationProfileId"`
	Checksum               string                `json:"checksum"`
}

type GetEDIInboundFileStatusCountsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Since      int64                 `json:"since"`
}

type ListRecentQuarantinedEDIInboundFilesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Limit      int                   `json:"limit"`
}

type EDIInboundFileRepository interface {
	ListInboundFiles(
		ctx context.Context,
		req *ListEDIInboundFilesRequest,
	) (*pagination.ListResult[*edi.EDIInboundFile], error)
	ListInboundFilesCursor(
		ctx context.Context,
		req *ListEDIInboundFilesRequest,
	) (*pagination.CursorListResult[*edi.EDIInboundFile], error)
	GetInboundFileStatusCounts(
		ctx context.Context,
		req GetEDIInboundFileStatusCountsRequest,
	) (map[edi.InboundFileStatus]int, error)
	ListRecentQuarantined(
		ctx context.Context,
		req ListRecentQuarantinedEDIInboundFilesRequest,
	) ([]*edi.EDIInboundFile, error)
	GetInboundFileByID(
		ctx context.Context,
		req GetEDIInboundFileByIDRequest,
	) (*edi.EDIInboundFile, error)
	CreateInboundFile(
		ctx context.Context,
		entity *edi.EDIInboundFile,
	) (*edi.EDIInboundFile, error)
	UpdateInboundFile(
		ctx context.Context,
		entity *edi.EDIInboundFile,
	) (*edi.EDIInboundFile, error)
	ExistsByChecksum(
		ctx context.Context,
		req ExistsEDIInboundFileByChecksumRequest,
	) (bool, error)
	CountQuarantinedSince(ctx context.Context, since int64) (int64, error)
	PurgeRawContentBefore(ctx context.Context, req PurgeEDIRawPayloadsRequest) (int64, error)
}

type PurgeEDIRawPayloadsRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
	PurgedAt   int64
	Limit      int
}
