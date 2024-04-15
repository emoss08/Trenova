package api

import (
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gofiber/storage/redis/v2"

	"entgo.io/ent/dialect"
	"github.com/gofiber/fiber/v2/middleware/session"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/internal/config"
	"github.com/emoss08/trenova/internal/ent"
	kfk "github.com/emoss08/trenova/internal/util/kafka"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Server struct {
	Config  config.Server
	Client  *ent.Client
	Fiber   *fiber.App
	Logger  *zerolog.Logger
	Session *session.Store
	Kafka   *kfk.Client
	Router  fiber.Router
}

func NewServer(config config.Server) *Server {
	s := &Server{
		Config:  config,
		Client:  nil,
		Fiber:   nil,
		Logger:  nil,
		Session: nil,
		Kafka:   nil,
		Router:  nil,
	}

	return s
}

func (s *Server) Ready() bool {
	return s.Client != nil && s.Fiber != nil && s.Session != nil && s.Logger != nil
}

// RegisterGob registers the UUID type with gob, so it can be used in sessions.
func (s *Server) RegisterGob() {
	gob.Register(uuid.UUID{})
}

func (s *Server) InitClient(ctx context.Context) error {
	db, err := sql.Open("pgx", "postgresql://postgres:postgres@localhost:5432/trenova_go_db?sslmode=disable")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database connection")
	}
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(time.Minute * 30)

	// Create an ent.Driver from `db`.
	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv))

	if err = db.PingContext(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to ping database")
	}

	s.Client = client

	return nil
}

func (s *Server) InitKafkaClient() error {
	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers": s.Config.Kafka.Broker,
	}

	client := kfk.NewKafkaClient(kafkaConfig)

	s.Kafka = client

	return nil
}

func (s *Server) InitSessionStore() error {
	store := redis.New(redis.Config{
		Host:     s.Config.Redis.Host,
		Port:     s.Config.Redis.Port,
		Username: s.Config.Redis.Username,
		Password: s.Config.Redis.Password,
		Database: s.Config.Redis.Database,
	})

	s.Session = session.New(session.Config{
		KeyLookup: "cookie:trenova_session_id",
		Storage:   store,
	})

	return nil
}

func (s *Server) Start() error {
	if !s.Ready() {
		return errors.New("server not ready")
	}

	// Register gob types
	s.RegisterGob()

	return s.Fiber.Listen(s.Config.Fiber.ListenAddress)
}

func (s *Server) Shutdown() error {
	log.Warn().Msg("Shutting down server")

	if s.Client != nil {
		log.Debug().Msg("Closing database connection")

		if err := s.Client.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close database connection")
		}
	}

	log.Debug().Msg("Shutting down fiber server")

	return s.Fiber.Shutdown()
}

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

func (s *Server) Cleanup() error {
	if err := s.Client.Close(); err != nil {
		return err
	}

	log.Info().Msg("Cleanup complete")

	return nil
}
