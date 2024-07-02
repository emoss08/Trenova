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
	log.Warn().Msg("Shutting down server")
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
