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

package server

import (
	"context"

	"github.com/emoss08/trenova-bg-jobs/internal/config"
	"github.com/emoss08/trenova-bg-jobs/internal/db"
	"github.com/emoss08/trenova-bg-jobs/internal/task"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Server struct {
	Config   config.Server
	DB       *pgx.Conn
	Logger   *zerolog.Logger
	Redis    *redis.Client
	Queue    *asynq.Client
	Enqueuer *task.TaskEnqueuer
}

func NewServer(config config.Server) *Server {
	return &Server{
		Config: config,
	}
}

func (s *Server) InitDB(ctx context.Context) error {
	db, err := db.InitDB(ctx, s.Config.DB.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database connection")
		return err
	}

	s.DB = db

	return nil
}

func (s *Server) InitalizeLogger() error {
	logger := zerolog.New(log.Logger).With().Timestamp().Logger()

	s.Logger = &logger

	return nil
}

func (s *Server) InitRedisClient(ctx context.Context) error {
	client := redis.NewClient(&redis.Options{
		Addr: s.Config.Redis.Addr,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
		return err
	}

	s.Redis = client

	return nil
}

func (s *Server) InitAsynqClient() error {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: s.Config.Redis.Addr})
	s.Queue = client
	s.Enqueuer = task.NewTaskEnqueuer(client, s.Logger)
	return nil
}

func (s *Server) Start() error {
	srv := asynq.NewServer(asynq.RedisClientOpt{Addr: s.Config.Redis.Addr},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(task.TypeSendReport, task.HandleSendReportTask)
	mux.HandleFunc(task.TypeNormalTask, task.HandleNormalTask)
	mux.HandleFunc(task.TypeCleanup, task.HandleCleanupTask)

	if err := srv.Run(mux); err != nil {
		log.Fatal().Err(err).Msg("Failed to start Asynq server")
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Debug().Msg("Shutting down server")
	defer func() {
		// Close the database connection
		if err := s.DB.Close(ctx); err != nil {
			log.Error().Err(err).Msg("Failed to close database connection")
		}

		// Close the Redis connection
		if err := s.Redis.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close Redis connection")
		}
	}()

	return nil
}
