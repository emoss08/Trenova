package bankreceiptworkitem

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type WorkItem struct {
	keel.Table `keel:"bank_receipt_work_items"`

	ID               pulid.ID       `json:"id" keel:"id,pk,autoid:brwi_"`
	OrganizationID   pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	BankReceiptID    pulid.ID       `json:"bankReceiptId" keel:"bank_receipt_id"`
	Status           Status         `json:"status" keel:"status"`
	AssignedToUserID pulid.ID       `json:"assignedToUserId" keel:"assigned_to_user_id,nullzero"`
	AssignedAt       *int64         `json:"assignedAt" keel:"assigned_at"`
	ResolutionType   ResolutionType `json:"resolutionType" keel:"resolution_type,nullzero"`
	ResolutionNote   string         `json:"resolutionNote" keel:"resolution_note,nullzero"`
	ResolvedByUserID pulid.ID       `json:"resolvedByUserId" keel:"resolved_by_user_id,nullzero"`
	ResolvedAt       *int64         `json:"resolvedAt" keel:"resolved_at"`
	CreatedByID      pulid.ID       `json:"createdById" keel:"created_by_id"`
	UpdatedByID      pulid.ID       `json:"updatedById" keel:"updated_by_id,nullzero"`
	Version          int64          `json:"version" keel:"version,version"`
	CreatedAt        int64          `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt        int64          `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (w *WorkItem) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(w,
		validation.Field(&w.OrganizationID, validation.Required),
		validation.Field(&w.BusinessUnitID, validation.Required),
		validation.Field(&w.BankReceiptID, validation.Required),
	))
}
