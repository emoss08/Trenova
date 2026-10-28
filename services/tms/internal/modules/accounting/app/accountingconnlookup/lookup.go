package accountingconnlookup

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func ByType(
	ctx context.Context,
	connections accountingconnectionports.AccountingConnectionRepository,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingsync.AccountingConnection, error) {
	if !accountingsync.SupportsAccountingSync(typ) {
		return nil, errortypes.NewValidationError(
			"integrationType",
			errortypes.ErrInvalid,
			"{0} is not an accounting system Trenova can sync with",
			string(typ),
		)
	}
	conn, err := connections.GetByType(
		ctx,
		accountingconnectionports.GetAccountingConnectionRequest{
			TenantInfo:      tenantInfo,
			IntegrationType: typ,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewNotFoundError(
				"{0} has not been connected yet",
				accountingsync.ProviderName(typ),
			)
		}
		return nil, err
	}
	return conn, nil
}
