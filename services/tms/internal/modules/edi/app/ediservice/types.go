package ediservice

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/editransport"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type SubmitLoadTenderRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	SourceShipmentID pulid.ID              `json:"sourceShipmentId"`
	EDIPartnerID     pulid.ID              `json:"ediPartnerId"`
}

type CreateInternalPartnerPairRequest struct {
	TenantInfo            pagination.TenantInfo `json:"-"`
	TargetOrganizationID  pulid.ID              `json:"targetOrganizationId"`
	SourceCode            string                `json:"sourceCode"`
	SourceName            string                `json:"sourceName"`
	SourceDescription     string                `json:"sourceDescription"`
	SourceContactName     string                `json:"sourceContactName"`
	SourceContactEmail    string                `json:"sourceContactEmail"`
	SourceContactPhone    string                `json:"sourceContactPhone"`
	SourceEnabledInbound  bool                  `json:"sourceEnabledForInbound"`
	SourceEnabledOutbound bool                  `json:"sourceEnabledForOutbound"`
	SourceSettings        map[string]any        `json:"sourceSettings"`
	TargetCode            string                `json:"targetCode"`
	TargetName            string                `json:"targetName"`
	TargetDescription     string                `json:"targetDescription"`
	TargetContactName     string                `json:"targetContactName"`
	TargetContactEmail    string                `json:"targetContactEmail"`
	TargetContactPhone    string                `json:"targetContactPhone"`
	TargetEnabledInbound  bool                  `json:"targetEnabledForInbound"`
	TargetEnabledOutbound bool                  `json:"targetEnabledForOutbound"`
	TargetSettings        map[string]any        `json:"targetSettings"`
}

type CreateEDIConnectionRequest struct {
	TenantInfo           pagination.TenantInfo       `json:"-"`
	TargetOrganizationID pulid.ID                    `json:"targetOrganizationId"`
	Method               edi.ConnectionMethod        `json:"method"`
	Capabilities         edi.ConnectionCapabilities  `json:"capabilities"`
	SourcePartnerConfig  edi.ConnectionPartnerConfig `json:"sourcePartnerConfig"`
	TargetPartnerConfig  edi.ConnectionPartnerConfig `json:"targetPartnerConfig"`
}

type EDIConnectionActionRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	ConnectionID pulid.ID              `json:"-"`
	Reason       string                `json:"reason"`
}

type UpsertEDICommunicationProfileRequest struct {
	TenantInfo      pagination.TenantInfo `json:"-"`
	ProfileID       pulid.ID              `json:"-"`
	EDIConnectionID pulid.ID              `json:"ediConnectionId"`
	EDIPartnerID    pulid.ID              `json:"ediPartnerId"`
	Method          edi.ConnectionMethod  `json:"method"`
	Status          string                `json:"status"`
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	Config          map[string]any        `json:"config"`
	Secrets         map[string]string     `json:"secrets"`
	Version         int64                 `json:"version"`
}

type ApproveTransferRequest struct {
	TenantInfo pagination.TenantInfo        `json:"-"`
	TransferID pulid.ID                     `json:"-"`
	Mappings   []*edi.EDIMappingProfileItem `json:"mappings"`
}

type RejectTransferRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TransferID pulid.ID              `json:"-"`
	Reason     string                `json:"reason"`
}

type CancelTransferRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TransferID pulid.ID              `json:"-"`
}

type ExpireTransferRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TransferID pulid.ID              `json:"-"`
}

type TransferChangeActionRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ChangeID   pulid.ID              `json:"-"`
	Reason     string                `json:"reason"`
}

type TenderChangeActionRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ChangeID   pulid.ID              `json:"-"`
	Reason     string                `json:"reason"`
}

type PreviewEDIDocumentRequest = ediports.PreviewEDIDocumentRequest

type GenerateEDIDocumentRequest = ediports.GenerateEDIDocumentRequest

type InspectX12Request struct {
	TenantInfo     pagination.TenantInfo            `json:"-"`
	RawX12         string                           `json:"rawX12"`
	TransactionSet edidocument.TransactionSet       `json:"transactionSet"`
	X12Version     string                           `json:"x12Version"`
	Envelope       *edidocument.X12EnvelopeSettings `json:"envelope"`
	Diagnostics    []edidocument.Diagnostic         `json:"diagnostics"`
}

type EDIMessageInspection struct {
	Message    *edimessage.EDIMessage         `json:"message"`
	Inspection edix12inspect.InspectX12Result `json:"inspection"`
	Provenance EDIInspectionProvenance        `json:"provenance"`
}

type EDIInspectionProvenance struct {
	MessageID       pulid.ID `json:"messageId"`
	LadingVersionID pulid.ID `json:"ladingVersionId"`
	FlowInstanceID  pulid.ID `json:"flowInstanceId"`
	GeneratedAt     int64    `json:"generatedAt"`
	GeneratedByID   pulid.ID `json:"generatedById"`
}

type EDIDocumentPreview = ediports.EDIDocumentPreview

type MappingPreview struct {
	Resolved   []editransfer.MappingResolution `json:"resolved"`
	Unresolved []editransfer.MappingResolution `json:"unresolved"`
	All        []editransfer.MappingResolution `json:"all"`
}

type ApproveLoadTenderTransferWorkflowPayload struct {
	TransferID pulid.ID                      `json:"transferId"`
	TenantInfo pagination.TenantInfo         `json:"tenantInfo"`
	Actor      *permissionports.RequestActor `json:"actor"`
}

type ApproveLoadTenderTransferWorkflowResult struct {
	TransferID       pulid.ID `json:"transferId"`
	TargetShipmentID pulid.ID `json:"targetShipmentId"`
	ProcessedAt      int64    `json:"processedAt"`
}

type DeliverEDIMessageWorkflowPayload struct {
	MessageID   pulid.ID                          `json:"messageId"`
	TenantInfo  pagination.TenantInfo             `json:"tenantInfo"`
	RetryPolicy *editransport.DeliveryRetryPolicy `json:"retryPolicy,omitempty"`
}

type DeliverEDIMessageWorkflowResult struct {
	MessageID      pulid.ID                         `json:"messageId"`
	DeliveryStatus edimessage.MessageDeliveryStatus `json:"deliveryStatus"`
	RemotePath     string                           `json:"remotePath"`
}

type MarkEDIMessageDeadLetteredPayload struct {
	MessageID  pulid.ID              `json:"messageId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Reason     string                `json:"reason"`
}

type RetryMessageDeliveryRequest struct {
	MessageID  pulid.ID              `json:"-"`
	TenantInfo pagination.TenantInfo `json:"-"`
}
