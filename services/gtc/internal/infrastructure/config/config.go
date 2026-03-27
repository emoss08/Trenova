package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/gtc/internal/core/domain"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"gopkg.in/yaml.v3"
)

var identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type Config struct {
	LogLevel              string
	HTTPPort              int
	DatabaseURL           string
	RedisURL              string
	MeilisearchURL        string
	MeilisearchAPIKey     string
	ProjectionConfigFile  string
	SlotName              string
	PublicationName       string
	StandbyTimeout        time.Duration
	AutoCreateSlot        bool
	AutoCreatePublication bool
	InactiveSlotAction    string
	MaxLagBytes           int64
	SnapshotBatchSize     int
	SnapshotConcurrency   int
	ProcessTimeout        time.Duration
	WorkerCount           int
	WorkerQueueSize       int
	RetryMaxAttempts      int
	RetryBackoff          time.Duration
	DLQStream             string
	CheckpointTable       string
	CheckpointSchema      string
	HealthPollInterval    time.Duration
}

type ProjectionFile struct {
	Projections []ProjectionConfig `yaml:"projections"`
}

type ProjectionConfig struct {
	Name             string            `yaml:"name"`
	SourceTable      string            `yaml:"source_table"`
	PrimaryKeys      []string          `yaml:"primary_keys"`
	PrimaryKey       string            `yaml:"primary_key"`
	Fields           []string          `yaml:"fields"`
	SearchableFields []string          `yaml:"searchable_fields"`
	FilterableFields []string          `yaml:"filterable_fields"`
	IgnoredUpdates   []string          `yaml:"ignored_updates"`
	Destination      DestinationConfig `yaml:"destination"`
}

type DestinationConfig struct {
	Kind        domain.DestinationKind `yaml:"kind"`
	Index       string                 `yaml:"index"`
	KeyTemplate string                 `yaml:"key_template"`
	Stream      string                 `yaml:"stream"`
}

func Load() (*Config, error) {
	maxLagBytes, err := loadMaxLagBytes()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		LogLevel:              getEnv("LOG_LEVEL", "INFO"),
		HTTPPort:              getInt("HTTP_PORT", 8080),
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/trenova_go_db?replication=database"),
		RedisURL:              getEnv("REDIS_URL", "redis://localhost:6379/0"),
		MeilisearchURL:        getEnv("MEILISEARCH_URL", "http://localhost:7700"),
		MeilisearchAPIKey:     os.Getenv("MEILISEARCH_API_KEY"),
		ProjectionConfigFile:  getEnv("GTC_CONFIG_FILE", "config/gtc.yaml"),
		SlotName:              getEnv("CDC_SLOT_NAME", "trenova_gtc_slot"),
		PublicationName:       getEnv("CDC_PUBLICATION_NAME", "trenova_gtc_publication"),
		StandbyTimeout:        getDuration("CDC_STANDBY_TIMEOUT", 10*time.Second),
		AutoCreateSlot:        getBool("CDC_AUTO_CREATE_SLOT", false),
		AutoCreatePublication: getBool("CDC_AUTO_CREATE_PUBLICATION", true),
		InactiveSlotAction:    getEnv("CDC_INACTIVE_SLOT_ACTION", "fail"),
		MaxLagBytes:           maxLagBytes,
		SnapshotBatchSize:     getInt("CDC_SNAPSHOT_BATCH_SIZE", 500),
		SnapshotConcurrency:   getInt("CDC_SNAPSHOT_CONCURRENCY", 2),
		ProcessTimeout:        getDuration("CDC_PROCESS_TIMEOUT", 15*time.Second),
		WorkerCount:           getInt("CDC_WORKER_COUNT", 4),
		WorkerQueueSize:       getInt("CDC_WORKER_QUEUE_SIZE", 128),
		RetryMaxAttempts:      getInt("CDC_RETRY_MAX_ATTEMPTS", 3),
		RetryBackoff:          getDuration("CDC_RETRY_BACKOFF", 500*time.Millisecond),
		DLQStream:             getEnv("CDC_DLQ_STREAM", "gtc:dlq"),
		CheckpointTable:       getEnv("CDC_CHECKPOINT_TABLE", "gtc_checkpoints"),
		CheckpointSchema:      getEnv("CDC_CHECKPOINT_SCHEMA", "public"),
		HealthPollInterval:    getDuration("CDC_HEALTH_POLL_INTERVAL", 10*time.Second),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func LoadProjections(path string) ([]domain.Projection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read projection config: %w", err)
	}

	var file ProjectionFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse projection config: %w", err)
	}

	if len(file.Projections) == 0 {
		return nil, fmt.Errorf("projection config contains no projections")
	}

	projections := make([]domain.Projection, 0, len(file.Projections))
	names := make(map[string]struct{}, len(file.Projections))

	for _, projectionCfg := range file.Projections {
		if err := projectionCfg.Validate(); err != nil {
			return nil, err
		}

		if _, exists := names[projectionCfg.Name]; exists {
			return nil, fmt.Errorf("duplicate projection name %q", projectionCfg.Name)
		}
		names[projectionCfg.Name] = struct{}{}

		schema, table, err := domain.ParseFullTableName(projectionCfg.SourceTable)
		if err != nil {
			return nil, err
		}

		projections = append(projections, domain.Projection{
			Name:             projectionCfg.Name,
			SourceSchema:     schema,
			SourceTable:      table,
			PrimaryKeys:      projectionCfg.KeyFields(),
			Fields:           slices.Clone(projectionCfg.Fields),
			SearchableFields: slices.Clone(projectionCfg.SearchableFields),
			FilterableFields: slices.Clone(projectionCfg.FilterableFields),
			IgnoredUpdates:   slices.Clone(projectionCfg.IgnoredUpdates),
			Destination: domain.Destination{
				Kind:        projectionCfg.Destination.Kind,
				Index:       projectionCfg.Destination.Index,
				KeyTemplate: projectionCfg.Destination.KeyTemplate,
				Stream:      projectionCfg.Destination.Stream,
			},
		})
	}

	return projections, nil
}

func (c Config) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.LogLevel, validation.Required),
		validation.Field(&c.HTTPPort, validation.Required, validation.Min(1), validation.Max(65535)),
		validation.Field(&c.DatabaseURL, validation.Required, validation.By(validateURL)),
		validation.Field(&c.RedisURL, validation.Required, validation.By(validateURL)),
		validation.Field(&c.MeilisearchURL, validation.Required, validation.By(validateURL)),
		validation.Field(&c.ProjectionConfigFile, validation.Required),
		validation.Field(&c.SlotName, validation.Required, validation.By(validateIdentifier)),
		validation.Field(&c.PublicationName, validation.Required, validation.By(validateIdentifier)),
		validation.Field(&c.CheckpointTable, validation.Required, validation.By(validateIdentifier)),
		validation.Field(&c.CheckpointSchema, validation.Required, validation.By(validateIdentifier)),
		validation.Field(&c.InactiveSlotAction, validation.Required, validation.In("fail", "warn")),
		validation.Field(&c.MaxLagBytes, validation.Required, validation.Min(int64(1))),
		validation.Field(&c.StandbyTimeout, validation.Required, validation.Min(time.Second)),
		validation.Field(&c.ProcessTimeout, validation.Required, validation.Min(500*time.Millisecond)),
		validation.Field(&c.WorkerCount, validation.Required, validation.Min(1)),
		validation.Field(&c.WorkerQueueSize, validation.Required, validation.Min(1)),
		validation.Field(&c.RetryMaxAttempts, validation.Required, validation.Min(1)),
		validation.Field(&c.RetryBackoff, validation.Required, validation.Min(100*time.Millisecond)),
		validation.Field(&c.DLQStream, validation.Required),
		validation.Field(&c.SnapshotBatchSize, validation.Required, validation.Min(1)),
		validation.Field(&c.SnapshotConcurrency, validation.Required, validation.Min(1)),
		validation.Field(&c.HealthPollInterval, validation.Required, validation.Min(time.Second)),
	)
}

func (p ProjectionConfig) Validate() error {
	keyFields := p.KeyFields()
	if len(keyFields) == 0 {
		return fmt.Errorf("projection requires at least one primary key field")
	}
	for _, keyField := range keyFields {
		if err := validateIdentifier(keyField); err != nil {
			return err
		}
	}

	return validation.ValidateStruct(
		&p,
		validation.Field(&p.Name, validation.Required),
		validation.Field(&p.SourceTable, validation.Required),
		validation.Field(&p.Destination, validation.Required),
	)
}

func (p ProjectionConfig) KeyFields() []string {
	if len(p.PrimaryKeys) > 0 {
		return slices.Clone(p.PrimaryKeys)
	}
	if strings.TrimSpace(p.PrimaryKey) == "" {
		return nil
	}
	return []string{p.PrimaryKey}
}

func (d DestinationConfig) Validate() error {
	if err := validation.ValidateStruct(
		&d,
		validation.Field(
			&d.Kind,
			validation.Required,
			validation.In(
				domain.DestinationMeilisearch,
				domain.DestinationRedisJSON,
				domain.DestinationRedisStream,
				domain.DestinationTCAStream,
			),
		),
	); err != nil {
		return err
	}

	switch d.Kind {
	case domain.DestinationMeilisearch:
		if strings.TrimSpace(d.Index) == "" {
			return fmt.Errorf("meilisearch destination requires index")
		}
	case domain.DestinationRedisJSON:
		if strings.TrimSpace(d.KeyTemplate) == "" {
			return fmt.Errorf("redis_json destination requires key_template")
		}
	case domain.DestinationRedisStream, domain.DestinationTCAStream:
		if strings.TrimSpace(d.Stream) == "" {
			return fmt.Errorf("%s destination requires stream", d.Kind)
		}
	}

	return nil
}

func validateIdentifier(value any) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("identifier must be a string")
	}
	if !identifierPattern.MatchString(s) {
		return fmt.Errorf("invalid identifier %q", s)
	}
	return nil
}

func validateURL(value any) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("url must be a string")
	}

	parsed, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("must be a valid URL")
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("must be a valid URL")
	}

	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		parsed, err := strconv.ParseBool(val)
		if err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		parsed, err := time.ParseDuration(val)
		if err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		parsed, err := strconv.Atoi(val)
		if err == nil {
			return parsed
		}
	}
	return defaultVal
}

func loadMaxLagBytes() (int64, error) {
	const (
		key        = "CDC_MAX_LAG_BYTES"
		defaultVal = int64(5 * 1024 * 1024 * 1024)
	)

	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}

	parsed, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: expected integer bytes value, got %q", key, val)
	}

	return parsed, nil
}
