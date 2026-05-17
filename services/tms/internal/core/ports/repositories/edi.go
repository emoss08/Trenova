package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDIPartnersRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetEDIPartnerByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type EDIPartnerSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	Kind               edi.PartnerKind                `json:"kind"`
	EnabledForOutbound bool                           `json:"enabledForOutbound"`
}

type GetReciprocalInternalPartnerRequest struct {
	SourceOrganizationID pulid.ID `json:"sourceOrganizationId"`
	TargetOrganizationID pulid.ID `json:"targetOrganizationId"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"`
}

type GetMappingProfileRequest struct {
	PartnerID  pulid.ID              `json:"partnerId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListEDIMappingProfilesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetMappingProfileByIDRequest struct {
	ProfileID  pulid.ID              `json:"profileId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type SaveMappingItemsRequest struct {
	PartnerID  pulid.ID                     `json:"partnerId"`
	TenantInfo pagination.TenantInfo        `json:"tenantInfo"`
	ActorID    pulid.ID                     `json:"actorId"`
	Items      []*edi.EDIMappingProfileItem `json:"items"`
}

type SaveMappingProfileItemsRequest struct {
	ProfileID  pulid.ID                     `json:"profileId"`
	TenantInfo pagination.TenantInfo        `json:"tenantInfo"`
	ActorID    pulid.ID                     `json:"actorId"`
	Items      []*edi.EDIMappingProfileItem `json:"items"`
}

type GetMappingItemsRequest struct {
	PartnerID   pulid.ID                `json:"partnerId"`
	TenantInfo  pagination.TenantInfo   `json:"tenantInfo"`
	EntityTypes []edi.MappingEntityType `json:"entityTypes"`
	SourceIDs   []pulid.ID              `json:"sourceIds"`
}

type DeleteMappingItemRequest struct {
	PartnerID     pulid.ID              `json:"partnerId"`
	MappingItemID pulid.ID              `json:"mappingItemId"`
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
}

type DeleteMappingProfileItemRequest struct {
	ProfileID     pulid.ID              `json:"profileId"`
	MappingItemID pulid.ID              `json:"mappingItemId"`
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
}

type ListEDIConnectionsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetEDIConnectionByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetEDIConnectionForUpdateRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetActiveEDIConnectionForPartnerRequest struct {
	PartnerID  pulid.ID              `json:"partnerId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Method     edi.ConnectionMethod  `json:"method"`
}

type CreateInternalEDIConnectionAcceptanceRequest struct {
	Connection    *edi.EDIConnection           `json:"connection"`
	SourcePartner *edi.EDIPartner              `json:"sourcePartner"`
	TargetPartner *edi.EDIPartner              `json:"targetPartner"`
	SourceProfile *edi.EDICommunicationProfile `json:"sourceProfile"`
	TargetProfile *edi.EDICommunicationProfile `json:"targetProfile"`
	TenantInfo    pagination.TenantInfo        `json:"tenantInfo"`
}

type ListEDICommunicationProfilesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetEDICommunicationProfileByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetActiveEDICommunicationProfileByPartnerRequest struct {
	PartnerID  pulid.ID              `json:"partnerId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Method     edi.ConnectionMethod  `json:"method"`
}

type ListEDITransfersRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetEDITransferByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Direction  string                `json:"direction"`
}

type GetEDITransferForUpdateRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Direction  string                `json:"direction"`
}

type SetEDITransferApprovalWorkflowRunIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	RunID      string                `json:"runId"`
}

type ListEDIShipmentLinksRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetEDIShipmentLinkByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetEDIShipmentLinksByShipmentIDRequest struct {
	ShipmentID pulid.ID              `json:"shipmentId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListEDITransferChangesRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	ShipmentLinkID pulid.ID                 `json:"shipmentLinkId"`
}

type GetEDITransferChangeByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListEDIDocumentTypesRequest struct {
	Standard       edi.EDIStandard       `json:"standard"`
	TransactionSet edi.TransactionSet    `json:"transactionSet"`
	Direction      edi.DocumentDirection `json:"direction"`
}

type ListEDITemplatesRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	TransactionSet edi.TransactionSet       `json:"transactionSet"`
	Direction      edi.DocumentDirection    `json:"direction"`
}

type GetEDITemplateByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CreateEDITemplateRequest struct {
	Template *edi.EDITemplate          `json:"template"`
	Version  *edi.EDITemplateVersion   `json:"version"`
	Segments []*edi.EDITemplateSegment `json:"segments"`
}

type GetActiveEDITemplateVersionRequest struct {
	TemplateID pulid.ID              `json:"templateId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	VersionID  pulid.ID              `json:"versionId"`
}

type ListEDITemplateVersionsRequest struct {
	TemplateID pulid.ID              `json:"templateId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetEDITemplateVersionByIDRequest struct {
	TemplateID pulid.ID              `json:"templateId"`
	VersionID  pulid.ID              `json:"versionId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ReplaceEDITemplateVersionSegmentsRequest struct {
	Version  *edi.EDITemplateVersion   `json:"version"`
	Segments []*edi.EDITemplateSegment `json:"segments"`
}

type ActivateEDITemplateVersionRequest struct {
	VersionID  pulid.ID              `json:"versionId"`
	TemplateID pulid.ID              `json:"templateId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ActorID    pulid.ID              `json:"actorId"`
	Notes      string                `json:"notes"`
	IsRollback bool                  `json:"isRollback"`
}

type ArchiveEDITemplateVersionRequest struct {
	VersionID  pulid.ID              `json:"versionId"`
	TemplateID pulid.ID              `json:"templateId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ActorID    pulid.ID              `json:"actorId"`
	Notes      string                `json:"notes"`
}

type ListEDIPartnerDocumentProfilesRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	TransactionSet edi.TransactionSet       `json:"transactionSet"`
	Direction      edi.DocumentDirection    `json:"direction"`
}

type GetEDIPartnerDocumentProfileByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetActiveEDIPartnerDocumentProfileRequest struct {
	PartnerID      pulid.ID              `json:"partnerId"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	TransactionSet edi.TransactionSet    `json:"transactionSet"`
	Direction      edi.DocumentDirection `json:"direction"`
}

type AllocateEDIControlNumbersRequest struct {
	TenantInfo     pagination.TenantInfo   `json:"tenantInfo"`
	PartnerID      pulid.ID                `json:"partnerId"`
	DocumentTypeID pulid.ID                `json:"documentTypeId"`
	Kinds          []edi.ControlNumberKind `json:"kinds"`
}

type ListEDIMessagesRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	TransactionSet edi.TransactionSet       `json:"transactionSet"`
	Direction      edi.DocumentDirection    `json:"direction"`
}

type GetEDIMessageByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CreateEDIMessageWithDiagnosticsRequest struct {
	Message     *edi.EDIMessage                  `json:"message"`
	Diagnostics []*edi.EDIMessageValidationError `json:"diagnostics"`
}

type ListEDITestCasesRequest struct {
	Filter                   *pagination.QueryOptions `json:"filter"`
	PartnerDocumentProfileID pulid.ID                 `json:"partnerDocumentProfileId"`
}

type GetEDITestCaseByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type EDIShipmentLinkRepository interface {
	ListShipmentLinks(
		ctx context.Context,
		req *ListEDIShipmentLinksRequest,
	) (*pagination.ListResult[*edi.ShipmentLink], error)
	GetShipmentLinkByID(
		ctx context.Context,
		req GetEDIShipmentLinkByIDRequest,
	) (*edi.ShipmentLink, error)
	GetShipmentLinksByShipmentID(
		ctx context.Context,
		req GetEDIShipmentLinksByShipmentIDRequest,
	) ([]*edi.ShipmentLink, error)
	CreateShipmentLink(ctx context.Context, entity *edi.ShipmentLink) (*edi.ShipmentLink, error)
}

type EDITransferChangeRepository interface {
	ListTransferChanges(
		ctx context.Context,
		req *ListEDITransferChangesRequest,
	) (*pagination.ListResult[*edi.TransferChange], error)
	GetTransferChangeByID(
		ctx context.Context,
		req GetEDITransferChangeByIDRequest,
	) (*edi.TransferChange, error)
	CreateTransferChange(
		ctx context.Context,
		entity *edi.TransferChange,
	) (*edi.TransferChange, error)
	UpdateTransferChange(
		ctx context.Context,
		entity *edi.TransferChange,
	) (*edi.TransferChange, error)
}

type EDILoadTenderTransferRepository interface {
	ListInbound(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.ListResult[*edi.EDITransfer], error)
	ListOutbound(
		ctx context.Context,
		req *ListEDITransfersRequest,
	) (*pagination.ListResult[*edi.EDITransfer], error)
	GetTransferByID(
		ctx context.Context,
		req GetEDITransferByIDRequest,
	) (*edi.EDITransfer, error)
	GetTransferForUpdate(
		ctx context.Context,
		req GetEDITransferForUpdateRequest,
	) (*edi.EDITransfer, error)
	CreateTransfer(
		ctx context.Context,
		entity *edi.EDITransfer,
	) (*edi.EDITransfer, error)
	UpdateTransfer(
		ctx context.Context,
		entity *edi.EDITransfer,
	) (*edi.EDITransfer, error)
	SetApprovalWorkflowRunID(
		ctx context.Context,
		req SetEDITransferApprovalWorkflowRunIDRequest,
	) (*edi.EDITransfer, error)
}

type EDIPartnerRepository interface {
	List(
		ctx context.Context,
		req *ListEDIPartnersRequest,
	) (*pagination.ListResult[*edi.EDIPartner], error)
	SelectOptions(
		ctx context.Context,
		req *EDIPartnerSelectOptionsRequest,
	) (*pagination.ListResult[*edi.EDIPartner], error)
	GetByID(
		ctx context.Context,
		req GetEDIPartnerByIDRequest,
	) (*edi.EDIPartner, error)
	Create(
		ctx context.Context,
		entity *edi.EDIPartner,
	) (*edi.EDIPartner, error)
	Update(
		ctx context.Context,
		entity *edi.EDIPartner,
	) (*edi.EDIPartner, error)
	GetReciprocalInternalPartner(
		ctx context.Context,
		req GetReciprocalInternalPartnerRequest,
	) (*edi.EDIPartner, error)
	GetMappingProfile(
		ctx context.Context,
		req GetMappingProfileRequest,
	) (*edi.EDIMappingProfile, error)
	ListMappingProfiles(
		ctx context.Context,
		req *ListEDIMappingProfilesRequest,
	) (*pagination.ListResult[*edi.EDIMappingProfile], error)
	GetMappingProfileByID(
		ctx context.Context,
		req GetMappingProfileByIDRequest,
	) (*edi.EDIMappingProfile, error)
	SaveMappingItems(
		ctx context.Context,
		req *SaveMappingItemsRequest,
	) ([]*edi.EDIMappingProfileItem, error)
	SaveMappingProfileItems(
		ctx context.Context,
		req *SaveMappingProfileItemsRequest,
	) ([]*edi.EDIMappingProfileItem, error)
	GetMappingItems(
		ctx context.Context,
		req GetMappingItemsRequest,
	) ([]*edi.EDIMappingProfileItem, error)
	DeleteMappingItem(ctx context.Context, req DeleteMappingItemRequest) error
	DeleteMappingProfileItem(ctx context.Context, req DeleteMappingProfileItemRequest) error
}

type EDIConnectionRepository interface {
	ListConnections(
		ctx context.Context,
		req *ListEDIConnectionsRequest,
	) (*pagination.ListResult[*edi.EDIConnection], error)
	GetConnectionByID(
		ctx context.Context,
		req GetEDIConnectionByIDRequest,
	) (*edi.EDIConnection, error)
	GetConnectionForUpdate(
		ctx context.Context,
		req GetEDIConnectionForUpdateRequest,
	) (*edi.EDIConnection, error)
	GetActiveConnectionForPartner(
		ctx context.Context,
		req GetActiveEDIConnectionForPartnerRequest,
	) (*edi.EDIConnection, error)
	CreateConnection(
		ctx context.Context,
		entity *edi.EDIConnection,
	) (*edi.EDIConnection, error)
	UpdateConnection(
		ctx context.Context,
		entity *edi.EDIConnection,
	) (*edi.EDIConnection, error)
	AcceptInternalConnection(
		ctx context.Context,
		req *CreateInternalEDIConnectionAcceptanceRequest,
	) (*edi.EDIConnection, error)
}

type EDICommunicationProfileRepository interface {
	ListProfiles(
		ctx context.Context,
		req *ListEDICommunicationProfilesRequest,
	) (*pagination.ListResult[*edi.EDICommunicationProfile], error)
	GetProfileByID(
		ctx context.Context,
		req GetEDICommunicationProfileByIDRequest,
	) (*edi.EDICommunicationProfile, error)
	GetActiveProfileByPartner(
		ctx context.Context,
		req GetActiveEDICommunicationProfileByPartnerRequest,
	) (*edi.EDICommunicationProfile, error)
	CreateProfile(
		ctx context.Context,
		entity *edi.EDICommunicationProfile,
	) (*edi.EDICommunicationProfile, error)
	UpdateProfile(
		ctx context.Context,
		entity *edi.EDICommunicationProfile,
	) (*edi.EDICommunicationProfile, error)
}

type EDIDocumentTypeRepository interface {
	ListDocumentTypes(
		ctx context.Context,
		req ListEDIDocumentTypesRequest,
	) ([]*edi.EDIDocumentType, error)
}

type EDITemplateRepository interface {
	ListTemplates(
		ctx context.Context,
		req *ListEDITemplatesRequest,
	) (*pagination.ListResult[*edi.EDITemplate], error)
	GetTemplateByID(ctx context.Context, req GetEDITemplateByIDRequest) (*edi.EDITemplate, error)
	CreateTemplate(
		ctx context.Context,
		req *CreateEDITemplateRequest,
	) (*edi.EDITemplate, *edi.EDITemplateVersion, error)
	UpdateTemplate(ctx context.Context, entity *edi.EDITemplate) (*edi.EDITemplate, error)
	ListTemplateVersions(
		ctx context.Context,
		req ListEDITemplateVersionsRequest,
	) ([]*edi.EDITemplateVersion, error)
	GetTemplateVersionByID(
		ctx context.Context,
		req GetEDITemplateVersionByIDRequest,
	) (*edi.EDITemplateVersion, error)
	GetActiveTemplateVersion(
		ctx context.Context,
		req GetActiveEDITemplateVersionRequest,
	) (*edi.EDITemplateVersion, error)
	CreateTemplateVersion(
		ctx context.Context,
		version *edi.EDITemplateVersion,
		segments []*edi.EDITemplateSegment,
	) (*edi.EDITemplateVersion, error)
	UpdateTemplateVersionMetadata(
		ctx context.Context,
		version *edi.EDITemplateVersion,
	) (*edi.EDITemplateVersion, error)
	ReplaceTemplateVersionSegments(
		ctx context.Context,
		req ReplaceEDITemplateVersionSegmentsRequest,
	) (*edi.EDITemplateVersion, error)
	ActivateTemplateVersion(
		ctx context.Context,
		req ActivateEDITemplateVersionRequest,
	) (*edi.EDITemplateVersion, error)
	ArchiveTemplateVersion(
		ctx context.Context,
		req ArchiveEDITemplateVersionRequest,
	) (*edi.EDITemplateVersion, error)
	EnsureBase204Template(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*edi.EDITemplate, *edi.EDITemplateVersion, error)
}

type EDIPartnerDocumentProfileRepository interface {
	ListPartnerDocumentProfiles(
		ctx context.Context,
		req *ListEDIPartnerDocumentProfilesRequest,
	) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error)
	GetPartnerDocumentProfileByID(
		ctx context.Context,
		req GetEDIPartnerDocumentProfileByIDRequest,
	) (*edi.EDIPartnerDocumentProfile, error)
	GetActivePartnerDocumentProfile(
		ctx context.Context,
		req GetActiveEDIPartnerDocumentProfileRequest,
	) (*edi.EDIPartnerDocumentProfile, error)
	CreatePartnerDocumentProfile(
		ctx context.Context,
		entity *edi.EDIPartnerDocumentProfile,
	) (*edi.EDIPartnerDocumentProfile, error)
	UpdatePartnerDocumentProfile(
		ctx context.Context,
		entity *edi.EDIPartnerDocumentProfile,
	) (*edi.EDIPartnerDocumentProfile, error)
}

type EDIControlNumberRepository interface {
	AllocateControlNumbers(
		ctx context.Context,
		req AllocateEDIControlNumbersRequest,
	) (map[edi.ControlNumberKind]int64, error)
}

type EDIMessageRepository interface {
	ListMessages(
		ctx context.Context,
		req *ListEDIMessagesRequest,
	) (*pagination.ListResult[*edi.EDIMessage], error)
	GetMessageByID(ctx context.Context, req GetEDIMessageByIDRequest) (*edi.EDIMessage, error)
	CreateMessageWithDiagnostics(
		ctx context.Context,
		req CreateEDIMessageWithDiagnosticsRequest,
	) (*edi.EDIMessage, error)
}

type EDITestCaseRepository interface {
	ListTestCases(
		ctx context.Context,
		req *ListEDITestCasesRequest,
	) (*pagination.ListResult[*edi.EDITestCase], error)
	GetTestCaseByID(ctx context.Context, req GetEDITestCaseByIDRequest) (*edi.EDITestCase, error)
	CreateTestCase(ctx context.Context, entity *edi.EDITestCase) (*edi.EDITestCase, error)
}

type EDIDocumentRepository interface {
	EDIDocumentTypeRepository
	EDITemplateRepository
	EDIPartnerDocumentProfileRepository
	EDIControlNumberRepository
	EDIMessageRepository
	EDITestCaseRepository
}
