package infrastructure

import (
	postgresRepositories "github.com/emoss08/trenova/internal/infrastructure/postgres/repositories"
	redisRepositories "github.com/emoss08/trenova/internal/infrastructure/redis/repositories"
	"go.uber.org/fx"
)

var RedisRepositoriesModule = fx.Module("redis-repositories",
	redisRepositories.Module,
)

var PostgresRepositoriesModule = fx.Module("postgres-repositories",
	postgresRepositories.Module,
)
