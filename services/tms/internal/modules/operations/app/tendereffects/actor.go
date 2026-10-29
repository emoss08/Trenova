package tendereffects

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func UserIDPtr(tenantInfo pagination.TenantInfo) *pulid.ID {
	if tenantInfo.UserID.IsNil() {
		return nil
	}
	userID := tenantInfo.UserID
	return &userID
}
