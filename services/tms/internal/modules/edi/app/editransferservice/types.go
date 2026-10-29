package editransferservice

import (
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

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
