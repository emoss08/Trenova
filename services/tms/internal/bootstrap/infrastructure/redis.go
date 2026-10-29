package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/ratelimit"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/internal/infrastructure/redis/accesscache"
	"github.com/emoss08/trenova/internal/infrastructure/redis/aicache"
	"github.com/emoss08/trenova/internal/infrastructure/redis/authstore"
	"github.com/emoss08/trenova/internal/infrastructure/redis/entitycache"
	"github.com/emoss08/trenova/internal/infrastructure/redis/feedstate"
	"github.com/emoss08/trenova/internal/infrastructure/redis/idempotencystore"
	"github.com/emoss08/trenova/internal/infrastructure/redis/providerstate"
	"github.com/emoss08/trenova/internal/infrastructure/redis/writebuffer"
	"go.uber.org/fx"
)

var RedisRepositoriesModule = fx.Module("redis-repositories",
	fx.Provide(
		entitycache.NewOrganizationRepository,
		authstore.NewSessionRepository,
		authstore.NewSSOLoginStateRepository,
		authstore.NewMFAChallengeRepository,
		authstore.NewAccountingOAuthStateRepository,
		writebuffer.NewAuditBufferRepository,
		accesscache.NewPermissionCacheRepository,
		accesscache.NewScopeVerdictCacheRepository,
		providerstate.NewModelPriceCacheRepository,
		aicache.NewQueryVectorCache,
		providerstate.NewProviderBreakerRepository,
		providerstate.NewProviderSlotRepository,
		entitycache.NewCustomerCacheRepository,
		entitycache.NewDocumentCacheRepository,
		entitycache.NewShipmentImportChatCacheRepository,
		entitycache.NewShipmentCacheRepository,
		entitycache.NewWorkerCacheRepository,
		writebuffer.NewStoredMileageBufferRepository,
		idempotencystore.NewIdempotencyStore,
		authstore.NewLoginThrottleStore,
		entitycache.NewShipmentBoardCache,
		entitycache.NewShipmentBoardEpochBumper,
		aicache.NewAIControlSummaryCache,
		aicache.NewAIAnalyticsCache,
		aicache.NewAITuneUpFreshness,
		feedstate.NewWeatherAlertFeedStateStore,
	),
)

var RedisModule = fx.Module("redis",
	fx.Provide(redis.NewConnection),
)

var RateLimitModule = fx.Module("ratelimit",
	fx.Provide(ratelimit.NewStore),
)
