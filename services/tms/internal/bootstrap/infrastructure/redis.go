package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/internal/infrastructure/redis/repositories"
	"go.uber.org/fx"
)

var RedisRepositoriesModule = fx.Module("redis-repositories",
	fx.Provide(
		repositories.NewOrganizationRepository,
		repositories.NewSessionRepository,
		repositories.NewSSOLoginStateRepository,
		repositories.NewAuditBufferRepository,
		repositories.NewPermissionCacheRepository,
		repositories.NewCustomerCacheRepository,
		repositories.NewDocumentCacheRepository,
		repositories.NewShipmentImportChatCacheRepository,
		repositories.NewShipmentCacheRepository,
		repositories.NewUsStateCacheRepository,
		repositories.NewWorkerCacheRepository,
	),
)

var RedisModule = fx.Module("redis",
	fx.Provide(redis.NewConnection),
)
