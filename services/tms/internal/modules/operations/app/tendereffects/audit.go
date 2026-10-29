package tendereffects

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AuditParams struct {
	TenantInfo pagination.TenantInfo
	TenderID   pulid.ID
	Operation  permission.Operation
	UserID     pulid.ID
	Comment    string
	Current    *tender.Tender
	Previous   *tender.Tender
}
