package kernelports

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
)

type EnvelopeRewrapper interface {
	IsEnvelope(value string) bool
	Rewrap(ctx context.Context, req *EnvelopeRewrapRequest) (string, bool, error)
}

type EnvelopeRewrapRequest struct {
	Sealed string
	Scopes []SecretScope
	Rebind bool
}

type EnvelopeRow struct {
	ID     string
	Tenant pagination.TenantInfo
	Sealed string
	Extra  map[string]string
}

type EnvelopeScopes func(row *EnvelopeRow, field string) []SecretScope

type EnvelopeColumn struct {
	Name           string
	Table          buncolgen.TableInfo
	ID             buncolgen.Column
	OrganizationID buncolgen.Column
	BusinessUnitID buncolgen.Column
	Value          buncolgen.Column
	Extra          []buncolgen.Column
	JSONMap        bool
	Rebind         bool
	Scopes         EnvelopeScopes
}

type EnvelopeObject struct {
	Column  EnvelopeColumn
	Version *buncolgen.Column
}

type EnvelopeSources struct {
	Columns []EnvelopeColumn
	Objects []EnvelopeObject
}

type ListEnvelopeRowsRequest struct {
	Column *EnvelopeColumn
	Tenant pagination.TenantInfo
	After  string
	Limit  int
}

type SwapEnvelopeRequest struct {
	Column      *EnvelopeColumn
	Row         *EnvelopeRow
	Replacement string
}

type SwapEnvelopeVersionRequest struct {
	Object      *EnvelopeObject
	Row         *EnvelopeRow
	Expected    string
	Replacement string
}

type EnvelopeRepository interface {
	ListTenants(ctx context.Context) ([]pagination.TenantInfo, error)
	ListRows(ctx context.Context, req *ListEnvelopeRowsRequest) ([]*EnvelopeRow, error)
	Present(ctx context.Context, column *EnvelopeColumn, row *EnvelopeRow) (bool, error)
	Swap(ctx context.Context, req *SwapEnvelopeRequest) (bool, error)
	SwapVersion(ctx context.Context, req *SwapEnvelopeVersionRequest) error
}
