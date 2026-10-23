package tariff

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"go.uber.org/fx"
)

type CodesParams struct {
	fx.In

	Accessorials accessorialchargeports.AccessorialChargeRepository
}

type Codes struct {
	accessorials accessorialchargeports.AccessorialChargeRepository
}

func NewCodes(p CodesParams) *Codes {
	return &Codes{accessorials: p.Accessorials}
}

func (c *Codes) Catalog(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]*accessorialcharge.AccessorialCharge, error) {
	return c.accessorials.ListCatalog(ctx, tenant)
}

func ChargeCodes(catalog []*accessorialcharge.AccessorialCharge) []rules.TariffChargeCode {
	codes := make([]rules.TariffChargeCode, 0, len(catalog))
	for _, charge := range catalog {
		if charge == nil {
			continue
		}
		codes = append(codes, rules.TariffChargeCode{
			Code:        charge.Code,
			Description: charge.Description,
			Active:      charge.Status == domaintypes.StatusActive,
		})
	}
	return codes
}

func ChargeByCode(
	catalog []*accessorialcharge.AccessorialCharge,
) map[string]*accessorialcharge.AccessorialCharge {
	byCode := make(map[string]*accessorialcharge.AccessorialCharge, len(catalog))
	for _, charge := range catalog {
		if charge != nil {
			byCode[accessorialcharge.NormalizeCode(charge.Code)] = charge
		}
	}
	return byCode
}
