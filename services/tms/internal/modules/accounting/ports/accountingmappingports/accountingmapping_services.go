package accountingmappingports

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
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
	Connection        *accountingconnection.AccountingConnection
	Groups            []AccountingMappingGroup
	RequiredTotal     int
	RequiredConfirmed int
	CanCompleteSetup  bool
}

type ListAccountingMappingsRequest struct {
	TenantInfo      pagination.TenantInfo
	IntegrationType integration.Type
	TargetTypes     []accountingsync.MappingTargetType
	States          []accountingmapping.MappingState
	RequiredOnly    bool
	Search          string
	Cursor          pagination.CursorInfo
}

type SearchAccountingReferenceRequest struct {
	TenantInfo      pagination.TenantInfo
	IntegrationType integration.Type
	Kind            accountingprovider.ReferenceKind
	Query           string
	UsableOnly      bool
	Limit           int
}

type AccountingMappingConfirmation struct {
	ID         pulid.ID
	ExternalID string
}

type AccountingMappingChange struct {
	Before *accountingmapping.AccountingMapping
	After  *accountingmapping.AccountingMapping
}

type AccountingMappingConfirmPlan struct {
	Changes   []*AccountingMappingChange
	Confirmed []*accountingmapping.AccountingMapping
}

type ConfirmAccountingMappingsRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Items      []AccountingMappingConfirmation
	Source     accountingmapping.MappingSource
}

type AccountingMappingActionRequest struct {
	TenantInfo         pagination.TenantInfo
	UserID             pulid.ID
	ID                 pulid.ID
	Source             accountingmapping.MappingSource
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
	Source             accountingmapping.MappingSource
	Reason             string
	AcknowledgeHistory bool
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

type EnsureAccountingMappingRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	TargetType   accountingsync.MappingTargetType
	ObjectID     pulid.ID
	Key          string
}

type CreateAccountingReferenceRecordRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	MappingID  pulid.ID
	Name       string
	Source     accountingmapping.MappingSource
}

type AccountingSetupRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
}

type AccountingReferencePull struct {
	Kind    accountingprovider.ReferenceKind
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

type AccountingMappingReader interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*AccountingMappingSummary, error)
	ListMappings(
		ctx context.Context,
		req *ListAccountingMappingsRequest,
	) (*pagination.CursorListResult[*accountingmapping.AccountingMapping], error)
	GetMapping(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*accountingmapping.AccountingMapping, error)
	GetMappingsByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*accountingmapping.AccountingMapping, error)
	FindMapping(
		ctx context.Context,
		req *SetAccountingMappingRequest,
	) (*accountingmapping.AccountingMapping, error)
	SearchReference(
		ctx context.Context,
		req *SearchAccountingReferenceRequest,
	) ([]*accountingmapping.AccountingReferenceObject, error)
	GetReferenceObjects(
		ctx context.Context,
		req *GetAccountingReferenceObjectsRequest,
	) ([]*accountingmapping.AccountingReferenceObject, error)
}

type AccountingMappingDecisionService interface {
	PlanConfirm(
		ctx context.Context,
		req *ConfirmAccountingMappingsRequest,
	) (*AccountingMappingConfirmPlan, error)
	Confirm(
		ctx context.Context,
		req *ConfirmAccountingMappingsRequest,
	) ([]*accountingmapping.AccountingMapping, error)
	PlanReject(
		ctx context.Context,
		req *AccountingMappingActionRequest,
	) (*AccountingMappingChange, error)
	Reject(
		ctx context.Context,
		req *AccountingMappingActionRequest,
	) (*accountingmapping.AccountingMapping, error)
	Set(
		ctx context.Context,
		req *SetAccountingMappingRequest,
	) (*accountingmapping.AccountingMapping, error)
	Clear(
		ctx context.Context,
		req *AccountingMappingActionRequest,
	) (*accountingmapping.AccountingMapping, error)
	RequestRefresh(
		ctx context.Context,
		req *AccountingSetupRequest,
	) (*accountingconnection.AccountingConnection, error)
	CompleteSetup(
		ctx context.Context,
		req *AccountingSetupRequest,
	) (*accountingconnection.AccountingConnection, error)
	EnsureMapping(
		ctx context.Context,
		req *EnsureAccountingMappingRequest,
	) (*accountingmapping.AccountingMapping, error)
}

type AccountingMappingScorer interface {
	PullReference(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
		kind accountingprovider.ReferenceKind,
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
}

type AccountingReferenceCreator interface {
	CreateReferenceRecord(
		ctx context.Context,
		req *CreateAccountingReferenceRecordRequest,
	) (*accountingmapping.AccountingMapping, error)
	CustomerParty(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
	) (*accountingconnectorports.AccountingPartyDraft, error)
	VendorParty(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		targetType accountingsync.MappingTargetType,
		objectID pulid.ID,
	) (*accountingconnectorports.AccountingPartyDraft, error)
}

type AccountingMappingHistoryGuard interface {
	GuardHistory(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		row *accountingmapping.AccountingMapping,
		acknowledged bool,
	) (int, error)
}
