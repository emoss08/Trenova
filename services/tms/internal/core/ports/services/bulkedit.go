package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type BulkEditFieldKind string

const (
	BulkEditFieldRecord = BulkEditFieldKind("record")
	BulkEditFieldSelect = BulkEditFieldKind("select")
)

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
	Actor      *RequestActor
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
