package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emoss08/routing/internal/api"
	"github.com/emoss08/routing/internal/database"
	"github.com/emoss08/routing/internal/kafka"
	"github.com/emoss08/routing/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/template/html/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	// _ Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {
	// _ Load configuration
	if err := loadConfig(); err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// _ Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// _ Initialize PostgreSQL
	dbDSN := viper.GetString("database.dsn")
	if dbDSN == "" {
		dbDSN = "postgres://postgres:password@localhost:5432/routing?sslmode=disable"
	}

	// _ Run migrations
	if viper.GetBool("database.auto_migrate") {
		migrator, err := database.NewMigrator(dbDSN, log.Logger)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create migrator")
		}
		defer migrator.Close()

		if err := migrator.Migrate(ctx); err != nil {
			log.Fatal().Err(err).Msg("Failed to run migrations")
		}
	}

	storage, err := storage.NewPostgresStorage(ctx, dbDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer storage.Close()

	// _ Initialize Redis
	redisAddr := viper.GetString("redis.addr")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	cache := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     viper.GetString("redis.password"),
		DB:           viper.GetInt("redis.db"),
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
	})

	if err := cache.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer cache.Close()

	// _ Initialize Kafka producer if configured
	var kafkaProducer *kafka.Producer
	if len(viper.GetStringSlice("kafka.brokers")) > 0 {
		kafkaConfig := kafka.ProducerConfig{
			Brokers:      viper.GetStringSlice("kafka.brokers"),
			Topic:        viper.GetString("kafka.topics.route_events"),
			BatchSize:    viper.GetInt("kafka.producer.batch_size"),
			BatchTimeout: viper.GetDuration("kafka.producer.batch_timeout"),
			Async:        viper.GetBool("kafka.producer.async"),
			Compression:  viper.GetString("kafka.producer.compression"),
		}

		kafkaProducer = kafka.NewProducer(kafkaConfig, log.Logger)
		defer kafkaProducer.Close()

		log.Info().
			Strs("brokers", kafkaConfig.Brokers).
			Str("topic", kafkaConfig.Topic).
			Msg("Kafka producer initialized")
	} else {
		log.Warn().Msg("Kafka not configured, events will not be published")
	}

	// _ Create API handler
	handler := api.NewHandler(storage, cache, log.Logger, kafkaProducer)

	// _ Initialize template engine
	engine := html.New("./views", ".html")
	engine.Reload(true) // Enable template reload in development

	// _ Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "Routing Service",
		ServerHeader:          "Routing",
		DisableStartupMessage: true,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           120 * time.Second,
		BodyLimit:             4 * 1024 * 1024, // 4MB
		CompressedFileSuffix:  ".gz",
		Prefork:               false,
		Views:                 engine,
	})

	// _ Middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "UTC",
	}))
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// _ Add metrics middleware
	app.Use(handler.Metrics().RecordHTTPMetrics())

	// _ Routes
	setupRoutes(app, handler)

	// _ Start server
	port := viper.GetString("server.port")
	if port == "" {
		port = "8080"
	}

	go func() {
		log.Info().Str("port", port).Msg("Starting routing service")
		if err := app.Listen(":" + port); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// _ Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// _ Graceful shutdown
	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/routing/")

	// _ Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("database.auto_migrate", true)
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("redis.db", 0)

	// _ Read environment variables
	viper.SetEnvPrefix("ROUTING")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("reading config: %w", err)
		}
		log.Warn().Msg("No config file found, using defaults and environment variables")
	}

	return nil
}

func setupRoutes(app *fiber.App, handler *api.Handler) {
	// _ Root route - visualization page
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("visualize", fiber.Map{
			"Title":       "Route Visualization",
			"APIEndpoint": "/api/v1/route/distance",
		})
	})

	// _ Health check
	app.Get("/health", handler.HealthCheck)

	// _ Metrics endpoint
	app.Get("/metrics", api.PrometheusHandler())

	// _ API v1 routes
	v1 := app.Group("/api/v1")
	v1.Get("/route/distance", handler.GetRouteDistance)
}
