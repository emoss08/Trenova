package api

import (
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/gofiber/storage/redis/v2"
	"time"

	"entgo.io/ent/dialect"
	"github.com/gofiber/fiber/v2/middleware/session"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Server struct {
	Config  config.Server
	Client  *ent.Client
	Fiber   *fiber.App
	Logger  *zerolog.Logger
	Session *session.Store
}

func NewServer(config config.Server) *Server {
	s := &Server{
		Config: config,
		Client: nil,
		Fiber:  nil,
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
		fmt.Printf("Failed to open database connection: %v\n", err)
		return err // Return immediately if connection fails
	}

	// Ping the database to check if the connection is working.
	if err := db.PingContext(ctx); err != nil {
		fmt.Printf("Failed to ping database: %v\n", err)
		return err
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Hour)

	// Create an ent.Driver from `db`.
	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv))

	s.Client = client

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
