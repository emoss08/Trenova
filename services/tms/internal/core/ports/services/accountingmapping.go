package services

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AccountingMappingGroup struct {
	TargetType accountingsync.MappingTargetType
	Unmatched  int
	Proposed   int
	Confirmed  int
}

type AccountingMappingSummary struct {
	IntegrationType   integration.Type
	ProviderName      string
	Connection        *accountingsync.AccountingConnection
	Groups            []AccountingMappingGroup
	RequiredTotal     int
	RequiredConfirmed int
	CanCompleteSetup  bool
}

type ListAccountingMappingsRequest struct {
	TenantInfo      pagination.TenantInfo
	IntegrationType integration.Type
	TargetTypes     []accountingsync.MappingTargetType
	States          []accountingsync.MappingState
	RequiredOnly    bool
	Search          string
	Cursor          pagination.CursorInfo
}

type SearchAccountingReferenceRequest struct {
	TenantInfo      pagination.TenantInfo
	IntegrationType integration.Type
	Kind            accountingsync.ReferenceKind
	Query           string
	UsableOnly      bool
	Limit           int
}

type GetAccountingReferenceObjectsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Kind         accountingsync.ReferenceKind
	ExternalIDs  []string
}

type AccountingMappingConfirmation struct {
	ID         pulid.ID
	ExternalID string
}

type ConfirmAccountingMappingsRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Items      []AccountingMappingConfirmation
	Source     accountingsync.MappingSource
}

type AccountingMappingActionRequest struct {
	TenantInfo         pagination.TenantInfo
	UserID             pulid.ID
	ID                 pulid.ID
	Source             accountingsync.MappingSource
	AcknowledgeHistory bool
}

type SetAccountingMappingRequest struct {
	TenantInfo         pagination.TenantInfo
	UserID             pulid.ID
	IntegrationType    integration.Type
	MappingID          pulid.ID
	TargetType         accountingsync.MappingTargetType
	TrenovaObjectID    pulid.ID
	TrenovaKey         string
	ExternalID         string
	Source             accountingsync.MappingSource
	Reason             string
	AcknowledgeHistory bool
}

type EnsureAccountingMappingRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	TargetType   accountingsync.MappingTargetType
	ObjectID     pulid.ID
	Key          string
}

func (r *SetAccountingMappingRequest) ValidateTarget() error {
	if !r.MappingID.IsNil() {
		return nil
	}
	if !r.TargetType.IsValid() {
		return errortypes.NewValidationError(
			"targetType",
			errortypes.ErrInvalid,
			"Name the mapping by its id, or by its target type with a record or key",
		)
	}
	if r.TargetType.KeyedByObject() {
		if r.TrenovaObjectID.IsNil() || r.TrenovaKey != "" {
			return errortypes.NewValidationError(
				"recordId",
				errortypes.ErrRequired,
				"A {0} mapping is named by the Trenova record's id",
				string(r.TargetType),
			)
		}
		return nil
	}
	if !r.TrenovaObjectID.IsNil() || !r.TargetType.AcceptsKey(r.TrenovaKey) {
		return errortypes.NewValidationError(
			"key",
			errortypes.ErrInvalid,
			"{0} is not a {1} key; use one of {2}",
			r.TrenovaKey,
			string(r.TargetType),
			strings.Join(r.TargetType.Keys(), ", "),
		)
	}
	return nil
}

type CreateAccountingReferenceRecordRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	MappingID  pulid.ID
	Name       string
	Source     accountingsync.MappingSource
}

type AccountingSetupRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
}

type AccountingReferencePull struct {
	Kind    accountingsync.ReferenceKind
	Fetched int
	Removed int64
}

type AccountingRescoreResult struct {
	Targets    int
	Created    int64
	Updated    int64
	Proposed   int
	NeedsModel []pulid.ID
}

type AccountingMappingService interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*AccountingMappingSummary, error)
	ListMappings(
		ctx context.Context,
		req *ListAccountingMappingsRequest,
	) (*pagination.CursorListResult[*accountingsync.AccountingMapping], error)
	GetMapping(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*accountingsync.AccountingMapping, error)
	GetMappingsByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*accountingsync.AccountingMapping, error)
	FindMapping(
		ctx context.Context,
		req *SetAccountingMappingRequest,
	) (*accountingsync.AccountingMapping, error)
	SearchReference(
		ctx context.Context,
		req *SearchAccountingReferenceRequest,
	) ([]*accountingsync.AccountingReferenceObject, error)
	GetReferenceObjects(
		ctx context.Context,
		req *GetAccountingReferenceObjectsRequest,
	) ([]*accountingsync.AccountingReferenceObject, error)
	Confirm(
		ctx context.Context,
		req *ConfirmAccountingMappingsRequest,
	) ([]*accountingsync.AccountingMapping, error)
	Reject(
		ctx context.Context,
		req *AccountingMappingActionRequest,
	) (*accountingsync.AccountingMapping, error)
	Set(
		ctx context.Context,
		req *SetAccountingMappingRequest,
	) (*accountingsync.AccountingMapping, error)
	Clear(
		ctx context.Context,
		req *AccountingMappingActionRequest,
	) (*accountingsync.AccountingMapping, error)
	GuardHistory(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		row *accountingsync.AccountingMapping,
		acknowledged bool,
	) (int, error)
	CreateReferenceRecord(
		ctx context.Context,
		req *CreateAccountingReferenceRecordRequest,
	) (*accountingsync.AccountingMapping, error)
	RequestRefresh(
		ctx context.Context,
		req *AccountingSetupRequest,
	) (*accountingsync.AccountingConnection, error)
	CompleteSetup(
		ctx context.Context,
		req *AccountingSetupRequest,
	) (*accountingsync.AccountingConnection, error)
	PullReference(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
		kind accountingsync.ReferenceKind,
	) (*AccountingReferencePull, error)
	Rescore(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
	) (*AccountingRescoreResult, error)
	ModelPass(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
		mappingIDs []pulid.ID,
	) (int, error)
	MarkRefreshStarted(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
	) error
	MarkRefreshFinished(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
		failure string,
	) error
	EnsureMapping(
		ctx context.Context,
		req *EnsureAccountingMappingRequest,
	) (*accountingsync.AccountingMapping, error)
	CustomerParty(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
	) (*AccountingPartyDraft, error)
	VendorParty(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		targetType accountingsync.MappingTargetType,
		objectID pulid.ID,
	) (*AccountingPartyDraft, error)
}
