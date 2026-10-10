package bulkeditports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type BulkEditFieldKind string

const BulkEditFieldRecord = BulkEditFieldKind("record")

const BulkEditFieldSelect = BulkEditFieldKind("select")

type BulkEditOption struct {
	Value string
	Label string
}

type BulkEditField struct {
	Name    string
	Label   string
	Kind    BulkEditFieldKind
	Record  string
	Options []BulkEditOption
}

type BulkEditApplyRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      *permissionports.RequestActor
	IDs        []pulid.ID
	Field      string
	Value      string
}

type BulkEditOutcome struct {
	ID       pulid.ID
	Previous string
	Changed  bool
	Err      error
}

type BulkEditor interface {
	Resource() permission.Resource
	Fields() []BulkEditField
	Apply(ctx context.Context, req *BulkEditApplyRequest) ([]BulkEditOutcome, error)
}
