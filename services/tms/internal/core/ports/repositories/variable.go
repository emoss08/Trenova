package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/variable"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListVariableRequest struct {
	Filter        *pagination.QueryOptions
	IncludeFormat bool
}

type GetVariableByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type GetVariablesByContextRequest struct {
	OrgID   pulid.ID
	Context variable.Context
	Active  bool
}

type GetVariablesByKeysRequest struct {
	OrgID pulid.ID
	Keys  []string
}

type ResolveVariableRequest struct {
	Variable *variable.Variable
	Params   map[string]any
}

type ListVariableFormatRequest struct {
	Filter *pagination.QueryOptions
}

type GetVariableFormatByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type ExecuteFormatSQLRequest struct {
	FormatSQL string
	Value     string
}

type VariableRepository interface {
	List(
		ctx context.Context,
		req *ListVariableRequest,
	) (*pagination.ListResult[*variable.Variable], error)
	GetByID(ctx context.Context, req GetVariableByIDRequest) (*variable.Variable, error)
	Create(ctx context.Context, v *variable.Variable) (*variable.Variable, error)
	Update(ctx context.Context, v *variable.Variable) (*variable.Variable, error)
	Delete(ctx context.Context, req GetVariableByIDRequest) error
	GetVariablesByContext(
		ctx context.Context,
		req GetVariablesByContextRequest,
	) ([]*variable.Variable, error)
	GetActiveVariablesByKeys(
		ctx context.Context,
		req GetVariablesByKeysRequest,
	) ([]*variable.Variable, error)
	GetByKey(ctx context.Context, orgID pulid.ID, key string) (*variable.Variable, error)
	ResolveVariable(ctx context.Context, req ResolveVariableRequest) (string, error)
	ListFormats(
		ctx context.Context,
		req *ListVariableFormatRequest,
	) (*pagination.ListResult[*variable.VariableFormat], error)
	GetFormatByID(
		ctx context.Context,
		req GetVariableFormatByIDRequest,
	) (*variable.VariableFormat, error)
	CreateFormat(ctx context.Context, f *variable.VariableFormat) (*variable.VariableFormat, error)
	UpdateFormat(ctx context.Context, f *variable.VariableFormat) (*variable.VariableFormat, error)
	DeleteFormat(ctx context.Context, req GetVariableFormatByIDRequest) error
	ExecuteFormatSQL(ctx context.Context, req ExecuteFormatSQLRequest) (string, error)
}
