package tariffports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type ResolveRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
	AsOf       int64
}

type Template struct {
	TemplateID    pulid.ID
	VersionNumber int64
	Name          string
	Source        string
	Inputs        []tarifftypes.InputDecl
	Imports       []tarifftypes.Import
	Libraries     []rules.TariffLibrary
}

type Resolver interface {
	ResolveBilling(ctx context.Context, req *ResolveRequest) (*Template, error)
}

type CheckRequest struct {
	TenantInfo pagination.TenantInfo
	Source     string
	Kind       rules.TariffKind
	Libraries  []rules.TariffLibrary
	Values     *rules.TariffInputValues
}

type Checker interface {
	Check(ctx context.Context, req *CheckRequest) (*rules.TariffCheck, error)
}

type ContextRequest struct {
	TenantInfo pagination.TenantInfo
	Libraries  []rules.TariffLibrary
}

type ContextSource interface {
	Context(ctx context.Context, req *ContextRequest) (*rules.TariffContext, error)
	LoadCells(
		ctx context.Context,
		tenant pagination.TenantInfo,
		tariffCtx *rules.TariffContext,
		needs []rules.TariffNeed,
	) error
}

type Condition struct {
	ID     string
	Source string
	Inputs tarifftypes.Inputs
}

type RateRequest struct {
	TenantInfo pagination.TenantInfo
	Template   *Template
	Inputs     tarifftypes.Inputs
	Conditions []Condition
	Record     rules.RawJSON
	Currency   string
	Exchange   *rules.TariffExchange
	Trace      bool
}

type RateResult struct {
	Bill       *tarifftypes.Bill
	Refusal    *rules.TariffRefusal
	Conditions map[string]bool
	Trace      rules.RawJSON
	Lookups    rules.RawJSON
}

type Rater interface {
	Rate(ctx context.Context, req *RateRequest) (*RateResult, error)
}
