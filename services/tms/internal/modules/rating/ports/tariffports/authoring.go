package tariffports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type LibrariesRequest struct {
	TenantInfo pagination.TenantInfo
	Sources    []string
}

type LibraryResolver interface {
	ResolveLibraries(ctx context.Context, req *LibrariesRequest) ([]rules.TariffLibrary, error)
}

type RatedByTemplateRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
	Limit      int
}

type RatedShipment struct {
	ShipmentID    pulid.ID
	ProNumber     string
	Record        rules.RawJSON
	Inputs        tarifftypes.Inputs
	Currency      string
	Exchange      *rules.TariffExchange
	VersionNumber int64
	Billed        decimal.NullDecimal
	RatedAt       int64
}

type RatedShipments interface {
	RatedByTemplate(ctx context.Context, req *RatedByTemplateRequest) ([]*RatedShipment, error)
}
