package config

import (
	"time"

	"github.com/emoss08/trenova-bg-jobs/internal/util"
)

type Redis struct {
	Addr string
}

type Server struct {
	Redis Redis    `json:"redis"`
	DB    Database `json:"database"`
}

func DefaultServiceConfigFromEnv() Server {
	return Server{
		Redis: Redis{
			Addr: util.GetEnv("REDIS_ADDR", "localhost:6379"),
		},
		DB: Database{
			Host:            util.GetEnv("DB_HOST", "localhost"),
			Port:            util.GetEnvAsInt("DB_PORT", 5432),
			Username:        util.GetEnv("DB_USER", "postgres"),
			Password:        util.GetEnv("DB_PASSWORD", "postgres"),
			Database:        util.GetEnv("DB_NAME", "trenova_go_db"),
			MaxOpenConns:    util.GetEnvAsInt("DB_MAX_OPEN_CONNS", 100),
			MaxIdleConns:    util.GetEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: time.Second * time.Duration(util.GetEnvAsInt("DB_CONN_MAX_LIFETIME_SECONDS", 300)),
		},
	}
}
