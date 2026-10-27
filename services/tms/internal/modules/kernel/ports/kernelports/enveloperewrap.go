package kernelports

import (
	"context"
	"io"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/pagination"
)

type EnvelopeRewrapper interface {
	IsEnvelope(value string) bool
	Rewrap(ctx context.Context, req *EnvelopeRewrapRequest) (string, bool, error)
	IsBlob(prefix []byte) bool
	BlobPrefixSize() int
	RewrapBlob(ctx context.Context, sealed io.Reader, scopes []SecretScope) (*BlobRewrap, error)
}

type BlobRewrap struct {
	Header   []byte
	Consumed int64
	Changed  bool
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
	Name    string
	Value   keel.Expr
	Extra   []keel.Expr
	JSONMap bool
	Rebind  bool
	Scopes  EnvelopeScopes
}

type EnvelopeObject struct {
	Column  EnvelopeColumn
	Version keel.Expr
}

func EnvelopeColumnName(column keel.Expr) string {
	table, index, ok := keel.ColumnOf(column)
	if !ok {
		return ""
	}

	return table.Columns[index].Name
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
	ListRows(ctx context.Context, req *ListEnvelopeRowsRequest) ([]*EnvelopeRow, error)
	Present(ctx context.Context, column *EnvelopeColumn, row *EnvelopeRow) (bool, error)
	Swap(ctx context.Context, req *SwapEnvelopeRequest) (bool, error)
	SwapVersion(ctx context.Context, req *SwapEnvelopeVersionRequest) error
}
