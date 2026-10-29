package reportingconfig

import (
	"strings"
	"time"
)

type Config struct {
	PoolMaxOpenConns              int               `mapstructure:"poolMaxOpenConns"              validate:"min=0,max=64"`
	StatementTimeout              time.Duration     `mapstructure:"statementTimeout"`
	PreviewStatementTimeout       time.Duration     `mapstructure:"previewStatementTimeout"`
	MaxRunDuration                time.Duration     `mapstructure:"maxRunDuration"`
	MaxRows                       int64             `mapstructure:"maxRows"                       validate:"min=0"`
	MaxArtifactBytes              int64             `mapstructure:"maxArtifactBytes"              validate:"min=0"`
	PreviewRowLimit               int               `mapstructure:"previewRowLimit"               validate:"min=0,max=1000"`
	PDFMaxRows                    int64             `mapstructure:"pdfMaxRows"                    validate:"min=0"`
	MaxToOneJoins                 int               `mapstructure:"maxToOneJoins"                 validate:"min=0,max=16"`
	MaxToManySubqueries           int               `mapstructure:"maxToManySubqueries"           validate:"min=0,max=8"`
	MaxDimensions                 int               `mapstructure:"maxDimensions"                 validate:"min=0,max=16"`
	MaxPivotColumns               int               `mapstructure:"maxPivotColumns"               validate:"min=0,max=200"`
	MaxPathDepth                  int               `mapstructure:"maxPathDepth"                  validate:"min=0,max=6"`
	MaxDefinitionLimit            int               `mapstructure:"maxDefinitionLimit"            validate:"min=0"`
	MaxConcurrentRunsPerOrg       int               `mapstructure:"maxConcurrentRunsPerOrg"       validate:"min=0,max=32"`
	MaxQueuedRunsPerOrg           int               `mapstructure:"maxQueuedRunsPerOrg"           validate:"min=0,max=256"`
	MaxConcurrentStatementsPerOrg int               `mapstructure:"maxConcurrentStatementsPerOrg" validate:"min=0,max=64"`
	ArtifactRetention             time.Duration     `mapstructure:"artifactRetention"`
	ResultCache                   ResultCacheConfig `mapstructure:"resultCache"`
	ArtifactPrefix                string            `mapstructure:"artifactPrefix"`
	CSVIncludeBOM                 bool              `mapstructure:"csvIncludeBom"`
	ExplainCostLimit              float64           `mapstructure:"explainCostLimit"              validate:"min=0"`
	ExplainRowLimit               float64           `mapstructure:"explainRowLimit"               validate:"min=0"`
	DeliveryLinkBaseURL           string            `mapstructure:"deliveryLinkBaseUrl"           validate:"omitempty,url"`
	EmailMaxAttachmentBytes       int64             `mapstructure:"emailMaxAttachmentBytes"       validate:"min=0"`
}

func (c *Config) GetPoolMaxOpenConns() int {
	if c.PoolMaxOpenConns == 0 {
		return 4
	}
	return c.PoolMaxOpenConns
}

func (c *Config) GetStatementTimeout() time.Duration {
	if c.StatementTimeout == 0 {
		return 5 * time.Minute
	}
	return c.StatementTimeout
}

func (c *Config) GetPreviewStatementTimeout() time.Duration {
	if c.PreviewStatementTimeout == 0 {
		return 10 * time.Second
	}
	return c.PreviewStatementTimeout
}

func (c *Config) GetMaxRunDuration() time.Duration {
	if c.MaxRunDuration == 0 {
		return 30 * time.Minute
	}
	return c.MaxRunDuration
}

func (c *Config) GetMaxRows() int64 {
	if c.MaxRows == 0 {
		return 5_000_000
	}
	return c.MaxRows
}

func (c *Config) GetMaxArtifactBytes() int64 {
	if c.MaxArtifactBytes == 0 {
		return 1 << 30
	}
	return c.MaxArtifactBytes
}

func (c *Config) GetPreviewRowLimit() int {
	if c.PreviewRowLimit == 0 {
		return 100
	}
	return c.PreviewRowLimit
}

func (c *Config) GetPDFMaxRows() int64 {
	if c.PDFMaxRows == 0 {
		return 5000
	}
	return c.PDFMaxRows
}

func (c *Config) GetMaxToOneJoins() int {
	if c.MaxToOneJoins == 0 {
		return 6
	}
	return c.MaxToOneJoins
}

func (c *Config) GetMaxToManySubqueries() int {
	if c.MaxToManySubqueries == 0 {
		return 3
	}
	return c.MaxToManySubqueries
}

func (c *Config) GetMaxDimensions() int {
	if c.MaxDimensions == 0 {
		return 6
	}
	return c.MaxDimensions
}

func (c *Config) GetMaxPivotColumns() int {
	if c.MaxPivotColumns == 0 {
		return 50
	}
	return c.MaxPivotColumns
}

func (c *Config) GetMaxPathDepth() int {
	if c.MaxPathDepth == 0 {
		return 3
	}
	return c.MaxPathDepth
}

func (c *Config) GetMaxDefinitionLimit() int {
	if c.MaxDefinitionLimit == 0 {
		return 100_000
	}
	return c.MaxDefinitionLimit
}

func (c *Config) GetMaxConcurrentRunsPerOrg() int {
	if c.MaxConcurrentRunsPerOrg == 0 {
		return 2
	}
	return c.MaxConcurrentRunsPerOrg
}

func (c *Config) GetMaxConcurrentStatementsPerOrg() int {
	if c.MaxConcurrentStatementsPerOrg == 0 {
		return 4
	}
	return c.MaxConcurrentStatementsPerOrg
}

func (c *Config) GetMaxQueuedRunsPerOrg() int {
	if c.MaxQueuedRunsPerOrg == 0 {
		return 10
	}
	return c.MaxQueuedRunsPerOrg
}

func (c *Config) GetArtifactRetention() time.Duration {
	if c.ArtifactRetention == 0 {
		return 7 * 24 * time.Hour
	}
	return c.ArtifactRetention
}

const DefaultResultCacheLocalBytes int64 = 128 << 20

type ResultCacheConfig struct {
	Enabled    bool                    `mapstructure:"enabled"`
	LocalBytes int64                   `mapstructure:"localBytes" validate:"min=0"`
	TTL        time.Duration           `mapstructure:"ttl"`
	Shared     SharedResultCacheConfig `mapstructure:"shared"`
}

type SharedResultCacheConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

func (c *ResultCacheConfig) GetLocalBytes() int64 {
	if c.LocalBytes <= 0 {
		return DefaultResultCacheLocalBytes
	}
	return c.LocalBytes
}

func (c *ResultCacheConfig) GetTTL() time.Duration {
	if c.TTL <= 0 {
		return 15 * time.Minute
	}
	return c.TTL
}

func (c *ResultCacheConfig) SharedEnabled() bool {
	return c.Enabled && c.Shared.Enabled
}

func (c *Config) GetArtifactPrefix() string {
	if c.ArtifactPrefix == "" {
		return "reports"
	}
	return c.ArtifactPrefix
}

func (c *Config) GetExplainCostLimit() float64 {
	if c.ExplainCostLimit == 0 {
		return 5_000_000
	}
	return c.ExplainCostLimit
}

func (c *Config) GetExplainRowLimit() float64 {
	if c.ExplainRowLimit == 0 {
		return 10_000_000
	}
	return c.ExplainRowLimit
}

func (c *Config) GetDeliveryLinkBaseURL() string {
	return strings.TrimSuffix(c.DeliveryLinkBaseURL, "/")
}

func (c *Config) GetEmailMaxAttachmentBytes() int64 {
	if c.EmailMaxAttachmentBytes == 0 {
		return 10 << 20
	}
	return c.EmailMaxAttachmentBytes
}

func Defaults() map[string]any {
	return map[string]any{
		"resultCache.enabled":        true,
		"resultCache.localBytes":     DefaultResultCacheLocalBytes,
		"resultCache.ttl":            "15m",
		"resultCache.shared.enabled": false,
	}
}
