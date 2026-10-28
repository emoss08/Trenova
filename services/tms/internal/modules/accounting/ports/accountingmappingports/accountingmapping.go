package accountingmappingports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type UpsertAccountingReferenceObjectsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Objects      []*accountingmapping.AccountingReferenceObject
	SeenAt       int64
}

type MarkAccountingReferenceRemovedRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Kind         accountingprovider.ReferenceKind
	SeenBefore   int64
	At           int64
}

type MarkAccountingReferencesRemovedRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Kind         accountingprovider.ReferenceKind
	ExternalIDs  []string
	At           int64
}

type ListAccountingReferenceObjectsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Kind         accountingprovider.ReferenceKind
	UsableOnly   bool
}

type SearchAccountingReferenceObjectsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Kind         accountingprovider.ReferenceKind
	Query        string
	UsableOnly   bool
	Limit        int
}

type GetAccountingReferenceObjectsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Kind         accountingprovider.ReferenceKind
	ExternalIDs  []string
}

type AccountingReferenceObjectRepository interface {
	Upsert(ctx context.Context, req *UpsertAccountingReferenceObjectsRequest) error
	MarkRemoved(ctx context.Context, req *MarkAccountingReferencesRemovedRequest) (int64, error)
	MarkRemovedUnseen(
		ctx context.Context,
		req *MarkAccountingReferenceRemovedRequest,
	) (int64, error)
	ListByKind(
		ctx context.Context,
		req *ListAccountingReferenceObjectsRequest,
	) ([]*accountingmapping.AccountingReferenceObject, error)
	Search(
		ctx context.Context,
		req *SearchAccountingReferenceObjectsRequest,
	) ([]*accountingmapping.AccountingReferenceObject, error)
	GetByExternalIDs(
		ctx context.Context,
		req *GetAccountingReferenceObjectsRequest,
	) ([]*accountingmapping.AccountingReferenceObject, error)
}

type ListAccountingMappingsQuery struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	TargetTypes  []accountingsync.MappingTargetType
}

type ListAccountingMappingsConnectionRequest struct {
	Filter       *pagination.QueryOptions
	Cursor       pagination.CursorInfo
	ConnectionID pulid.ID
	TargetTypes  []accountingsync.MappingTargetType
	States       []accountingmapping.MappingState
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
	State      accountingmapping.MappingState
	Count      int
}

type AccountingMappingRepository interface {
	ListByConnection(
		ctx context.Context,
		req *ListAccountingMappingsQuery,
	) ([]*accountingmapping.AccountingMapping, error)
	ListConnection(
		ctx context.Context,
		req *ListAccountingMappingsConnectionRequest,
	) (*pagination.CursorListResult[*accountingmapping.AccountingMapping], error)
	GetByID(
		ctx context.Context,
		req GetAccountingMappingRequest,
	) (*accountingmapping.AccountingMapping, error)
	GetByIDs(
		ctx context.Context,
		req GetAccountingMappingsByIDsRequest,
	) ([]*accountingmapping.AccountingMapping, error)
	GetByTarget(
		ctx context.Context,
		req *GetAccountingMappingByTargetRequest,
	) (*accountingmapping.AccountingMapping, error)
	CreateMissing(
		ctx context.Context,
		entities []*accountingmapping.AccountingMapping,
	) (int64, error)
	ApplyScoring(
		ctx context.Context,
		entities []*accountingmapping.AccountingMapping,
	) (int64, error)
	UpdateLabels(ctx context.Context, entities []*accountingmapping.AccountingMapping) error
	Update(
		ctx context.Context,
		entity *accountingmapping.AccountingMapping,
	) (*accountingmapping.AccountingMapping, error)
	CountByState(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
	) ([]AccountingMappingCount, error)
	ListOwnerOperators(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*worker.Worker, error)
}
