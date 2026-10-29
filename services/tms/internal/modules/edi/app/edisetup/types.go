package edisetup

import (
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

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
