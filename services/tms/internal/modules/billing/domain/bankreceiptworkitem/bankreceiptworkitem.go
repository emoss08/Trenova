package bankreceiptworkitem

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type WorkItem struct {
	bun.BaseModel `bun:"table:bank_receipt_work_items,alias:brwi" json:"-"`
	keel.Table    `keel:"bank_receipt_work_items"`

	ID               pulid.ID       `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:brwi_"`
	OrganizationID   pulid.ID       `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID       `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	BankReceiptID    pulid.ID       `json:"bankReceiptId"    bun:"bank_receipt_id,type:VARCHAR(100),notnull" keel:"bank_receipt_id"`
	Status           Status         `json:"status"           bun:"status,type:VARCHAR(50),notnull" keel:"status"`
	AssignedToUserID pulid.ID       `json:"assignedToUserId" bun:"assigned_to_user_id,type:VARCHAR(100),nullzero" keel:"assigned_to_user_id,nullzero"`
	AssignedAt       *int64         `json:"assignedAt"       bun:"assigned_at,type:BIGINT,nullzero" keel:"assigned_at"`
	ResolutionType   ResolutionType `json:"resolutionType"   bun:"resolution_type,type:VARCHAR(50),nullzero" keel:"resolution_type,nullzero"`
	ResolutionNote   string         `json:"resolutionNote"   bun:"resolution_note,type:TEXT,nullzero" keel:"resolution_note,nullzero"`
	ResolvedByUserID pulid.ID       `json:"resolvedByUserId" bun:"resolved_by_user_id,type:VARCHAR(100),nullzero" keel:"resolved_by_user_id,nullzero"`
	ResolvedAt       *int64         `json:"resolvedAt"       bun:"resolved_at,type:BIGINT,nullzero" keel:"resolved_at"`
	CreatedByID      pulid.ID       `json:"createdById"      bun:"created_by_id,type:VARCHAR(100),notnull" keel:"created_by_id"`
	UpdatedByID      pulid.ID       `json:"updatedById"      bun:"updated_by_id,type:VARCHAR(100),nullzero" keel:"updated_by_id,nullzero"`
	Version          int64          `json:"version"          bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt        int64          `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt        int64          `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (w *WorkItem) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(w,
		validation.Field(&w.OrganizationID, validation.Required),
		validation.Field(&w.BusinessUnitID, validation.Required),
		validation.Field(&w.BankReceiptID, validation.Required),
	))
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
