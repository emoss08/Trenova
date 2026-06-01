package ediservice

import (
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/internal/core/services/edix12inspect"
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

type PreviewEDIDocumentRequest struct {
	TenantInfo               pagination.TenantInfo `json:"-"`
	PartnerDocumentProfileID pulid.ID              `json:"partnerDocumentProfileId"`
	EDIPartnerID             pulid.ID              `json:"ediPartnerId"`
	ShipmentID               pulid.ID              `json:"shipmentId"`
	TransferID               pulid.ID              `json:"transferId"`
	InvoiceID                pulid.ID              `json:"invoiceId"`
	ShipmentEventID          pulid.ID              `json:"shipmentEventId"`
	ServiceFailureID         pulid.ID              `json:"serviceFailureId"`
	SourceMessageID          pulid.ID              `json:"sourceMessageId"`
	TransactionSet           edi.TransactionSet    `json:"transactionSet"`
	Direction                edi.DocumentDirection `json:"direction"`
	Payload                  *edi.DocumentPayload  `json:"payload"`
}

type GenerateEDIDocumentRequest struct {
	TenantInfo               pagination.TenantInfo `json:"-"`
	PartnerDocumentProfileID pulid.ID              `json:"partnerDocumentProfileId"`
	EDIPartnerID             pulid.ID              `json:"ediPartnerId"`
	ShipmentID               pulid.ID              `json:"shipmentId"`
	TransferID               pulid.ID              `json:"transferId"`
	InvoiceID                pulid.ID              `json:"invoiceId"`
	ShipmentEventID          pulid.ID              `json:"shipmentEventId"`
	ServiceFailureID         pulid.ID              `json:"serviceFailureId"`
	SourceMessageID          pulid.ID              `json:"sourceMessageId"`
	TransactionSet           edi.TransactionSet    `json:"transactionSet"`
	Direction                edi.DocumentDirection `json:"direction"`
	Payload                  *edi.DocumentPayload  `json:"payload"`
	GeneratedByID            pulid.ID              `json:"-"`
}

type InspectX12Request struct {
	TenantInfo     pagination.TenantInfo    `json:"-"`
	RawX12         string                   `json:"rawX12"`
	TransactionSet edi.TransactionSet       `json:"transactionSet"`
	X12Version     string                   `json:"x12Version"`
	Envelope       *edi.X12EnvelopeSettings `json:"envelope"`
	Diagnostics    []edix12.Diagnostic      `json:"diagnostics"`
}

type EDIMessageInspection struct {
	Message    *edi.EDIMessage                `json:"message"`
	Inspection edix12inspect.InspectX12Result `json:"inspection"`
	Provenance EDIInspectionProvenance        `json:"provenance"`
}

type EDIInspectionProvenance struct {
	MessageID         pulid.ID `json:"messageId"`
	ProfileID         pulid.ID `json:"profileId"`
	TemplateID        pulid.ID `json:"templateId"`
	TemplateVersionID pulid.ID `json:"templateVersionId"`
	GeneratedAt       int64    `json:"generatedAt"`
	GeneratedByID     pulid.ID `json:"generatedById"`
}

type UpsertEDIPartnerDocumentProfileRequest struct {
	TenantInfo                   pagination.TenantInfo    `json:"-"`
	ProfileID                    pulid.ID                 `json:"-"`
	EDIPartnerID                 pulid.ID                 `json:"ediPartnerId"`
	TemplateID                   pulid.ID                 `json:"templateId"`
	TemplateVersionID            pulid.ID                 `json:"templateVersionId"`
	Name                         string                   `json:"name"`
	Status                       edi.DocumentStatus       `json:"status"`
	X12VersionOverride           string                   `json:"x12VersionOverride"`
	FunctionalGroupID            string                   `json:"functionalGroupId"`
	Envelope                     edi.X12EnvelopeSettings  `json:"envelope"`
	Acknowledgment               edi.AcknowledgmentConfig `json:"acknowledgment"`
	ValidationMode               edi.ValidationMode       `json:"validationMode"`
	PartnerSettings              map[string]any           `json:"partnerSettings"`
	PartnerSettingsSchemaID      pulid.ID                 `json:"partnerSettingsSchemaId"`
	PartnerSettingsSchemaVersion int64                    `json:"partnerSettingsSchemaVersion"`
	Version                      int64                    `json:"version"`
}

type ValidatePartnerSettingsRequest struct {
	TenantInfo                   pagination.TenantInfo `json:"-"`
	PartnerDocumentProfileID     pulid.ID              `json:"partnerDocumentProfileId"`
	PartnerSettingsSchemaID      pulid.ID              `json:"partnerSettingsSchemaId"`
	PartnerSettingsSchemaVersion int64                 `json:"partnerSettingsSchemaVersion"`
	DocumentTypeID               pulid.ID              `json:"documentTypeId"`
	Standard                     edi.EDIStandard       `json:"standard"`
	TransactionSet               edi.TransactionSet    `json:"transactionSet"`
	Direction                    edi.DocumentDirection `json:"direction"`
	X12Version                   string                `json:"x12Version"`
	Settings                     map[string]any        `json:"settings"`
}

type CreateEDITemplateRequest struct {
	TenantInfo        pagination.TenantInfo           `json:"-"`
	DocumentTypeID    pulid.ID                        `json:"documentTypeId"`
	Name              string                          `json:"name"`
	Description       string                          `json:"description"`
	Direction         edi.DocumentDirection           `json:"direction"`
	Standard          edi.EDIStandard                 `json:"standard"`
	TransactionSet    edi.TransactionSet              `json:"transactionSet"`
	X12Version        string                          `json:"x12Version"`
	FunctionalGroupID string                          `json:"functionalGroupId"`
	Notes             string                          `json:"notes"`
	Segments          []*edi.EDITemplateSegment       `json:"segments"`
	ScriptLibraries   []*edi.EDITemplateScriptLibrary `json:"scriptLibraries"`
}

type UpdateEDITemplateRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	TemplateID  pulid.ID              `json:"-"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Status      edi.TemplateStatus    `json:"status"`
	Version     int64                 `json:"version"`
}

type CreateEDITemplateDraftRequest struct {
	TenantInfo      pagination.TenantInfo `json:"-"`
	TemplateID      pulid.ID              `json:"-"`
	SourceVersionID pulid.ID              `json:"sourceVersionId"`
	Notes           string                `json:"notes"`
}

type UpdateEDITemplateVersionRequest struct {
	TenantInfo        pagination.TenantInfo `json:"-"`
	TemplateID        pulid.ID              `json:"-"`
	VersionID         pulid.ID              `json:"-"`
	X12Version        string                `json:"x12Version"`
	FunctionalGroupID string                `json:"functionalGroupId"`
	Notes             string                `json:"notes"`
	Version           int64                 `json:"version"`
}

type ReplaceEDITemplateSegmentsRequest struct {
	TenantInfo pagination.TenantInfo     `json:"-"`
	TemplateID pulid.ID                  `json:"-"`
	VersionID  pulid.ID                  `json:"-"`
	Segments   []*edi.EDITemplateSegment `json:"segments"`
	Version    int64                     `json:"version"`
}

type ReplaceEDITemplateScriptLibrariesRequest struct {
	TenantInfo      pagination.TenantInfo           `json:"-"`
	TemplateID      pulid.ID                        `json:"-"`
	VersionID       pulid.ID                        `json:"-"`
	ScriptLibraries []*edi.EDITemplateScriptLibrary `json:"scriptLibraries"`
	Version         int64                           `json:"version"`
}

type EDIActionNotesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TemplateID pulid.ID              `json:"-"`
	VersionID  pulid.ID              `json:"-"`
	Notes      string                `json:"notes"`
}

type EDIDocumentPreview struct {
	RawX12                   string                         `json:"rawX12"`
	SegmentCount             int64                          `json:"segmentCount"`
	X12Version               string                         `json:"x12Version"`
	InterchangeControlNumber string                         `json:"interchangeControlNumber"`
	GroupControlNumber       string                         `json:"groupControlNumber"`
	TransactionControlNumber string                         `json:"transactionControlNumber"`
	Diagnostics              []edix12.Diagnostic            `json:"diagnostics"`
	Profile                  *edi.EDIPartnerDocumentProfile `json:"profile"`
	TemplateVersion          *edi.EDITemplateVersion        `json:"templateVersion"`
}

type MappingPreview struct {
	Resolved   []edi.MappingResolution `json:"resolved"`
	Unresolved []edi.MappingResolution `json:"unresolved"`
	All        []edi.MappingResolution `json:"all"`
}

type ApproveLoadTenderTransferWorkflowPayload struct {
	TransferID pulid.ID               `json:"transferId"`
	TenantInfo pagination.TenantInfo  `json:"tenantInfo"`
	Actor      *services.RequestActor `json:"actor"`
}

type ApproveLoadTenderTransferWorkflowResult struct {
	TransferID       pulid.ID `json:"transferId"`
	TargetShipmentID pulid.ID `json:"targetShipmentId"`
	ProcessedAt      int64    `json:"processedAt"`
}
