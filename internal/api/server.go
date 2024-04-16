// Package api manages the server setup and its dependencies like database, cache, and logging.
package api

import (
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/redis/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver

	"github.com/emoss08/trenova/internal/config"
	"github.com/emoss08/trenova/internal/ent"
	kfk "github.com/emoss08/trenova/internal/util/kafka"
	"github.com/emoss08/trenova/internal/util/minio"
	rop "github.com/emoss08/trenova/internal/util/redis"
	mio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	rdb "github.com/redis/go-redis/v9"
)

// Server holds all dependencies for the API server, including database, cache, and session clients.
type Server struct {
	Config  config.Server
	Client  *ent.Client
	Fiber   *fiber.App
	Logger  *zerolog.Logger
	Session *session.Store
	Kafka   *kfk.Client
	Redis   *rop.Client
	Minio   *minio.Client
}

// NewServer initializes a new Server instance with given configuration.
func NewServer(cfg config.Server) *Server {
	return &Server{Config: cfg}
}

// Ready checks if all critical components of the server are initialized.
func (s *Server) Ready() bool {
	return s.Client != nil &&
		s.Fiber != nil &&
		s.Session != nil &&
		s.Logger != nil &&
		s.Kafka != nil &&
		s.Redis != nil &&
		s.Minio != nil
}

// RegisterGobTypes registers necessary types with the gob package, needed for session management.
func (s *Server) RegisterGobTypes() {
	gob.Register(uuid.UUID{})
}

// InitClient sets up the database client using the configured options.
func (s *Server) InitClient(ctx context.Context) error {
	db, err := sql.Open("pgx", s.Config.DB.ConnectionString())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database connection")
		return err
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(time.Minute * 30)

	drv := entsql.OpenDB(dialect.Postgres, db)
	s.Client = ent.NewClient(ent.Driver(drv))

	if err = db.PingContext(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to ping database")
		return err
	}

	return nil
}

// InitKafkaClient initializes the Kafka client with the specified configuration.
func (s *Server) InitKafkaClient() error {
	config := kafka.ConfigMap{"bootstrap.servers": s.Config.Kafka.Broker}
	s.Kafka = kfk.NewClient(&config)
	return nil
}

// InitSessionStore configures the session storage using Redis.
func (s *Server) InitSessionStore() error {
	store := redis.New(redis.Config{
		Host:     s.Config.Redis.Host,
		Port:     s.Config.Redis.Port,
		Username: s.Config.Redis.Username,
		Password: s.Config.Redis.Password,
		Database: s.Config.Redis.Database,
	})
	s.Session = session.New(session.Config{KeyLookup: "cookie:trenova_session_id", Storage: store})
	return nil
}

// InitLogger initializes the logger based on the configuration.
func (s *Server) InitLogger() error {
	logger := zerolog.New(log.Logger)

	if s.Config.Logger.PrettyPrintConsole {
		logger = log.Output(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.TimeFormat = "15:04:05"
		}))
	}

	s.Logger = &logger

	return nil
}

// InitRedisClient creates and verifies a Redis client connection.
func (s *Server) InitRedisClient(ctx context.Context) error {
	client := rop.NewClient(&rdb.Options{
		Addr:     s.Config.Redis.Addr,
		Password: s.Config.Redis.Password,
		DB:       s.Config.Redis.Database,
	})
	if err := client.Ping(ctx); err != nil {
		return err
	}
	s.Redis = client
	return nil
}

// InitMinioClient initializes the Minio client with the specified configuration.
func (s *Server) InitMinioClient(ctx context.Context) error {
	mc := minio.NewClient(s.Config.Minio.Endpoint, &mio.Options{
		Creds:  credentials.NewStaticV4(s.Config.Minio.AccessKey, s.Config.Minio.SecretKey, ""),
		Secure: s.Config.Minio.UseSSL,
	})

	if err := mc.Ping(ctx); err != nil {
		return err
	}

	s.Minio = mc

	return nil
}

// Start runs the server if all components are ready and registers necessary types.
func (s *Server) Start() error {
	if !s.Ready() {
		return errors.New("server not ready")
	}
	s.RegisterGobTypes()
	return s.Fiber.Listen(s.Config.Fiber.ListenAddress)
}

// Shutdown performs a graceful shutdown of all server components.
func (s *Server) Shutdown() error {
	log.Warn().Msg("Shutting down server.")
	defer func() {
		if err := s.Client.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close database connection.")
		}
		if err := s.Fiber.Shutdown(); err != nil {
			log.Error().Err(err).Msg("Failed to shutdown Fiber server.")
		}
	}()
	return nil
}

// Cleanup closes all connections and cleans up resources.
func (s *Server) Cleanup() error {
	if err := s.Redis.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to close Redis connection.")
		return err
	}
	if s.Kafka != nil {
		s.Kafka.Close()
	}
	log.Info().Msg("Cleanup complete.")
	return nil
}
