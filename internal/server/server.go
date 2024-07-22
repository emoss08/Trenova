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
	"database/sql"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/pkg/audit"
	tCasbin "github.com/emoss08/trenova/pkg/casbin"
	"github.com/emoss08/trenova/pkg/file"
	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/kfk"
	"github.com/emoss08/trenova/pkg/minio"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/redis"
	"github.com/fatih/color"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/golang-jwt/jwt/v5"
	mio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

type appCtxKey struct{}

// ContextWithApp returns a new context with the given app instance.
func ContextWithApp(ctx context.Context, app *Server) context.Context {
	return context.WithValue(ctx, appCtxKey{}, app)
}

type Server struct {
	ctx context.Context

	// Fiber stores the application instance.
	Fiber *fiber.App

	// Config stores the application configuration.
	Config config.Server

	// Logger stores the application logger.
	Logger *zerolog.Logger

	// Minio stores the Minio client.
	Minio *minio.Client

	// FileHandler stores the file handler.
	FileHandler file.FileHandler

	// Kafka stores the Kafka client.
	Kafka *kfk.KafkaClient

	// Enforcer stores the Casbin enforcer.
	Enforcer *casbin.Enforcer

	// AuditService stores the audit logging service.
	AuditService *audit.Service

	// Code Generate
	CodeGenerator   *gen.CodeGenerator
	CounterManager  *gen.CounterManager
	CodeChecker     *gen.CodeChecker
	CodeInitializer *gen.CodeInitializer

	// Hooks after stop
	onStop      appHooks
	onAfterstop appHooks

	// lazy init
	dbOnce sync.Once
	// DB stores the connection to the database.
	DB *bun.DB

	// Lazy init
	cacheOnce sync.Once
	// Cache stores the cache client.
	Cache *redis.Client
}

// NewServer creates a new server instance.
func NewServer(ctx context.Context, cfg config.Server) *Server {
	server := &Server{Config: cfg}
	server.ctx = ContextWithApp(ctx, server)

	return server
}

func (s *Server) Ready() bool {
	return s.DB != nil && s.Logger != nil && s.Minio != nil && s.Kafka != nil
}

// OnStop registers a function to be called when the server starts.
func (s *Server) OnStop(name string, fn HookFunc) {
	s.Logger.Debug().Msgf("Registering onStop hook: %s", name)
	s.onStop.add(newHook(name, fn))
}

// OnAfterStop registers a function to be called after the server is stopped.
func (s *Server) OnAfterStop(name string, fn HookFunc) {
	s.Logger.Debug().Msgf("Registering onAfterStop hook: %s", name)

	s.onAfterstop.add(newHook(name, fn))
}

// InitDB initializes the database connection.
func (s *Server) InitDB() *bun.DB {
	s.dbOnce.Do(func() {
		maxOpenConns := 4 * runtime.GOMAXPROCS(0)

		pgconn := pgdriver.NewConnector(
			pgdriver.WithDSN(s.Config.DB.DSN()),
			pgdriver.WithTimeout(30*time.Second),
			pgdriver.WithWriteTimeout(30*time.Second),
		)

		sqldb := sql.OpenDB(pgconn)
		sqldb.SetMaxOpenConns(maxOpenConns)
		sqldb.SetMaxIdleConns(maxOpenConns)

		db := bun.NewDB(sqldb, pgdialect.New(), bun.WithDiscardUnknownColumns())
		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(s.Config.DB.VerboseLogging),
			bundebug.WithEnabled(s.Config.DB.Debug),
		))

		// Register many-to-many model so bun can better recognize m2m relation.
		// This should be done before you use the model for the first time.
		db.RegisterModel(
			(*models.GeneralLedgerAccountTag)(nil),
		)

		s.OnStop("db.Close", func(_ context.Context, _ *Server) error {
			return db.Close()
		})

		s.DB = db
	})

	return s.DB
}

func (s *Server) InitCache() *redis.Client {
	s.cacheOnce.Do(func() {
		client := redis.NewClient(s.Config.Cache.Addr, s.Logger)

		s.OnStop("cache.Close", func(_ context.Context, _ *Server) error {
			return client.Close()
		})

		s.Cache = client
	})

	return s.Cache
}

func (s *Server) InitAuditService() error {
	s.AuditService = audit.NewAuditService(s.DB, s.Logger, s.Config.Audit.QueueSize, s.Config.Audit.WorkerCount)

	// Register shutdown hook
	s.OnStop("audit.Shutdown", func(ctx context.Context, _ *Server) error {
		return s.AuditService.Shutdown(ctx)
	})

	return nil
}

func (s *Server) InitCodeGenerationSystem(ctx context.Context) error {
	s.CounterManager = gen.NewCounterManager()
	s.CodeChecker = &gen.CodeChecker{DB: s.DB}
	s.CodeGenerator = gen.NewCodeGenerator(s.CounterManager, s.CodeChecker)
	s.CodeInitializer = &gen.CodeInitializer{DB: s.DB}

	mods := []gen.CodeGeneratable{
		&models.Worker{},
		&models.Location{},
		&models.Customer{},
	}

	// Initialize the counter manager with existing codes
	err := s.CodeInitializer.Initialize(ctx, s.CounterManager, mods...)
	if err != nil {
		return fmt.Errorf("failed to initialize code generator: %w", err)
	}

	return nil
}

// InitLogger initializes the logger.
func (s *Server) InitLogger() {
	logger := zerolog.New(log.Logger).With().Timestamp().Logger()

	if s.Config.Logger.PrettyPrintConsole {
		logger = logger.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
			FormatLevel: func(i any) string {
				return colorizeLevel(utils.ToUpper(i.(string)))
			},
			FormatMessage: func(i any) string {
				return fmt.Sprintf("%s", i)
			},
			FormatFieldName: func(i any) string {
				return fmt.Sprintf("%s=", i)
			},
			FormatFieldValue: func(i any) string {
				return fmt.Sprintf("%s", i)
			},
		})
	}

	s.Logger = &logger
}

func (s *Server) InitCasbin() error {
	adapter, err := tCasbin.NewBunAdapter(s.DB)
	if err != nil {
		return err
	}

	s.Enforcer, err = casbin.NewEnforcer(s.Config.Casbin.ModelPath, adapter)
	if err != nil {
		return err
	}

	// Load the policy rules from the database.
	if err = s.Enforcer.LoadPolicy(); err != nil {
		return err
	}

	return nil
}

func colorizeLevel(level string) string {
	switch level {
	case "INFO":
		return color.New(color.BgGreen, color.FgHiBlack).Sprint(level)
	case "DEBUG":
		return color.New(color.BgBlue, color.FgHiBlack).Sprint(level)
	case "WARN":
		return color.New(color.BgYellow, color.FgHiBlack).Sprint(level)
	case "ERROR":
		return color.New(color.FgRed).Sprint(level)
	default:
		return level
	}
}

// InitMinioClient initializes the Minio client.
func (s *Server) InitMinioClient() error {
	mc := minio.NewClient(s.Config.Minio.Endpoint, s.Logger, &mio.Options{
		Creds:  credentials.NewStaticV4(s.Config.Minio.AccessKey, s.Config.Minio.SecretKey, ""),
		Secure: s.Config.Minio.UseSSL,
	})

	s.Minio = mc

	return nil
}

func (s *Server) InitKafkaClient() error {
	client, err := kfk.NewKafkaClient(s.Config.Kafka.Seeds, s.Logger)
	if err != nil {
		return err
	}

	s.Kafka = client

	return nil
}

// LoadRSAKeys loads the RSA keys from the filesystem.
func (s *Server) LoadRSAKeys() error {
	privateKeyData, err := os.ReadFile("private_key.pem")
	if err != nil {
		return err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		return err
	}

	publicKeyData, err := os.ReadFile("public_key.pem")
	if err != nil {
		return err
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyData)
	if err != nil {
		return err
	}

	s.Config.Auth.PrivateKey = privateKey
	s.Config.Auth.PublicKey = publicKey

	return nil
}

func (s *Server) Start() error {
	if !s.Ready() {
		return errors.New("server is not ready")
	}

	if err := onStart.Run(s.ctx, s); err != nil {
		return err
	}

	return s.Fiber.Listen(s.Config.Fiber.ListenAddress)
}

func (s *Server) Shutdown() error {
	_ = s.onStop.Run(s.ctx, s)
	_ = s.onAfterstop.Run(s.ctx, s)

	return nil
}
