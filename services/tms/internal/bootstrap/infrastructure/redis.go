package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/ratelimit"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/internal/infrastructure/redis/repositories"
	"go.uber.org/fx"
)

var RedisRepositoriesModule = fx.Module("redis-repositories",
	fx.Provide(
		repositories.NewOrganizationRepository,
		repositories.NewSessionRepository,
		repositories.NewSSOLoginStateRepository,
		repositories.NewMFAChallengeRepository,
		repositories.NewAccountingOAuthStateRepository,
		repositories.NewAuditBufferRepository,
		repositories.NewPermissionCacheRepository,
		repositories.NewScopeVerdictCacheRepository,
		repositories.NewModelPriceCacheRepository,
		repositories.NewProviderBreakerRepository,
		repositories.NewProviderSlotRepository,
		repositories.NewAccessPolicyCacheRepository,
		repositories.NewModeProfileCacheRepository,
		repositories.NewCustomerCacheRepository,
		repositories.NewDocumentCacheRepository,
		repositories.NewShipmentImportChatCacheRepository,
		repositories.NewShipmentCacheRepository,
		repositories.NewIFTAJurisdictionCacheRepository,
		repositories.NewWorkerCacheRepository,
		repositories.NewStoredMileageBufferRepository,
		repositories.NewIdempotencyStore,
		repositories.NewLoginThrottleStore,
		repositories.NewShipmentBoardCache,
		repositories.NewShipmentBoardEpochBumper,
		repositories.NewAIControlSummaryCache,
		repositories.NewAITuneUpFreshness,
	),
)

var RedisModule = fx.Module("redis",
	fx.Provide(redis.NewConnection),
)

var RateLimitModule = fx.Module("ratelimit",
	fx.Provide(ratelimit.NewStore),
)
