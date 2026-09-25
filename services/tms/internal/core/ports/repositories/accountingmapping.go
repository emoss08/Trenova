package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type UpsertAccountingReferenceObjectsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Objects      []*accountingsync.AccountingReferenceObject
	SeenAt       int64
}

type MarkAccountingReferenceRemovedRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Kind         accountingsync.ReferenceKind
	SeenBefore   int64
	At           int64
}

type ListAccountingReferenceObjectsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Kind         accountingsync.ReferenceKind
	UsableOnly   bool
}

type SearchAccountingReferenceObjectsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Kind         accountingsync.ReferenceKind
	Query        string
	UsableOnly   bool
	Limit        int
}

type GetAccountingReferenceObjectsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Kind         accountingsync.ReferenceKind
	ExternalIDs  []string
}

type AccountingReferenceObjectRepository interface {
	Upsert(ctx context.Context, req *UpsertAccountingReferenceObjectsRequest) error
	MarkRemovedUnseen(
		ctx context.Context,
		req *MarkAccountingReferenceRemovedRequest,
	) (int64, error)
	ListByKind(
		ctx context.Context,
		req *ListAccountingReferenceObjectsRequest,
	) ([]*accountingsync.AccountingReferenceObject, error)
	Search(
		ctx context.Context,
		req *SearchAccountingReferenceObjectsRequest,
	) ([]*accountingsync.AccountingReferenceObject, error)
	GetByExternalIDs(
		ctx context.Context,
		req *GetAccountingReferenceObjectsRequest,
	) ([]*accountingsync.AccountingReferenceObject, error)
}

type ListAccountingMappingsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	TargetTypes  []accountingsync.MappingTargetType
}

type ListAccountingMappingsConnectionRequest struct {
	Filter       *pagination.QueryOptions
	Cursor       pagination.CursorInfo
	ConnectionID pulid.ID
	TargetTypes  []accountingsync.MappingTargetType
	States       []accountingsync.MappingState
	RequiredOnly bool
	Search       string
}

type GetAccountingMappingRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type GetAccountingMappingsByIDsRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

type GetAccountingMappingByTargetRequest struct {
	TenantInfo      pagination.TenantInfo
	ConnectionID    pulid.ID
	TargetType      accountingsync.MappingTargetType
	TrenovaObjectID pulid.ID
	TrenovaKey      string
}

type AccountingMappingCount struct {
	TargetType accountingsync.MappingTargetType
	State      accountingsync.MappingState
	Count      int
}

type AccountingMappingRepository interface {
	ListByConnection(
		ctx context.Context,
		req *ListAccountingMappingsRequest,
	) ([]*accountingsync.AccountingMapping, error)
	ListConnection(
		ctx context.Context,
		req *ListAccountingMappingsConnectionRequest,
	) (*pagination.CursorListResult[*accountingsync.AccountingMapping], error)
	GetByID(
		ctx context.Context,
		req GetAccountingMappingRequest,
	) (*accountingsync.AccountingMapping, error)
	GetByIDs(
		ctx context.Context,
		req GetAccountingMappingsByIDsRequest,
	) ([]*accountingsync.AccountingMapping, error)
	GetByTarget(
		ctx context.Context,
		req *GetAccountingMappingByTargetRequest,
	) (*accountingsync.AccountingMapping, error)
	CreateMissing(ctx context.Context, entities []*accountingsync.AccountingMapping) (int64, error)
	ApplyScoring(ctx context.Context, entities []*accountingsync.AccountingMapping) (int64, error)
	Update(
		ctx context.Context,
		entity *accountingsync.AccountingMapping,
	) (*accountingsync.AccountingMapping, error)
	CountByState(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
	) ([]AccountingMappingCount, error)
}
