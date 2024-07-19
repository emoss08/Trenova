// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
