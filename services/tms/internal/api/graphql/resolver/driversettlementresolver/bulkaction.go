package driversettlementresolver

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementworkspaceservice"
)

func bulkActionOperation(
	action driversettlementworkspaceservice.BulkActionType,
) permission.Operation {
	switch action {
	case driversettlementworkspaceservice.BulkActionSubmit:
		return permission.OpSubmit
	case driversettlementworkspaceservice.BulkActionApprove,
		driversettlementworkspaceservice.BulkActionPost:
		return permission.OpApprove
	default:
		return permission.OpUpdate
	}
}
