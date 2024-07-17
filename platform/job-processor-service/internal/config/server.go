// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
