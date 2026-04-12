package bankreceiptworkitem

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type WorkItem struct {
	bun.BaseModel `bun:"table:bank_receipt_work_items,alias:brwi" json:"-"`

	ID               pulid.ID       `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID       `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID       `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	BankReceiptID    pulid.ID       `json:"bankReceiptId"    bun:"bank_receipt_id,type:VARCHAR(100),notnull"`
	Status           Status         `json:"status"           bun:"status,type:VARCHAR(50),notnull"`
	AssignedToUserID pulid.ID       `json:"assignedToUserId" bun:"assigned_to_user_id,type:VARCHAR(100),nullzero"`
	AssignedAt       *int64         `json:"assignedAt"       bun:"assigned_at,type:BIGINT,nullzero"`
	ResolutionType   ResolutionType `json:"resolutionType"   bun:"resolution_type,type:VARCHAR(50),nullzero"`
	ResolutionNote   string         `json:"resolutionNote"   bun:"resolution_note,type:TEXT,nullzero"`
	ResolvedByUserID pulid.ID       `json:"resolvedByUserId" bun:"resolved_by_user_id,type:VARCHAR(100),nullzero"`
	ResolvedAt       *int64         `json:"resolvedAt"       bun:"resolved_at,type:BIGINT,nullzero"`
	CreatedByID      pulid.ID       `json:"createdById"      bun:"created_by_id,type:VARCHAR(100),notnull"`
	UpdatedByID      pulid.ID       `json:"updatedById"      bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	Version          int64          `json:"version"          bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt        int64          `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64          `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (w *WorkItem) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(w,
		validation.Field(&w.OrganizationID, validation.Required),
		validation.Field(&w.BusinessUnitID, validation.Required),
		validation.Field(&w.BankReceiptID, validation.Required),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (w *WorkItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if w.ID.IsNil() {
			w.ID = pulid.MustNew("brwi_")
		}
		w.CreatedAt = now
	case *bun.UpdateQuery:
		w.UpdatedAt = now
	}
	return nil
}
