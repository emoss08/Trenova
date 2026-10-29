package editenderservice

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type SubmitLoadTenderRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	SourceShipmentID pulid.ID              `json:"sourceShipmentId"`
	EDIPartnerID     pulid.ID              `json:"ediPartnerId"`
}
