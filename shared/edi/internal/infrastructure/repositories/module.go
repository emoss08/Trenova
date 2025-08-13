package repositories

import "go.uber.org/fx"

var Module = fx.Module("repositories",
	fx.Provide(
		NewEDIDocumentRepository,
		NewEDITransactionRepository,
		NewEDIShipmentRepository,
		NewEDIPartnerProfileRepository,
	),
)