package editransferchangeservice

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TransferChangeActionRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ChangeID   pulid.ID              `json:"-"`
	Reason     string                `json:"reason"`
}
