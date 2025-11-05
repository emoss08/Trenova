package repositories

import (
	"github.com/emoss08/trenova/internal/infrastructure/redis/repositories/permissioncache"
	"go.uber.org/fx"
)

var Module = fx.Module("redis-repositories",
	fx.Provide(
		NewSessionRepository,
		permissioncache.NewPermissionRepository,
		NewAPITokenRepository,
		NewOrganizationRepository,
		NewUsStateRepository,
		NewShipmentControlRepository,
		NewBillingControlRepository,
		NewDataRetentionRepository,
		NewHazmatExpirationRepository,
		NewDispatchControlRepository,
		NewAccountingControlRepository,
	),
)
