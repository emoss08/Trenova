package ediservice

import (
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/services"
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
