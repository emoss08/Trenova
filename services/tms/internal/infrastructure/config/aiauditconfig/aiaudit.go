package aiauditconfig

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
)

const (
	defaultAIAuditExportSyncMaxRows   = 5000
	defaultAIAuditExportMaxRows       = 1_000_000
	defaultAIAuditExportTTL           = 7 * 24 * time.Hour
	defaultAIAuditProjectorInterval   = time.Minute
	defaultAIAuditProjectorBatchSize  = 1000
	minAIAuditProjectorInterval       = 10 * time.Second
	maxAIAuditProjectorBatchSize      = 10_000
	MinChainSecretLength              = 32
	maxAIAuditChainKeyIDLength        = 40
	aiAuditChainKeysEnvEntrySeparator = ","
	aiAuditChainKeysEnvPairSeparator  = ":"
	redactedValue                     = "[redacted]"
)

var (
	ErrChainKeyIDInvalid = errors.New(
		"aiAudit.chain.keys ids must be 1 to 40 letters, digits, '.', '_' or '-'",
	)
	ErrChainKeyIDDuplicate = errors.New("aiAudit.chain.keys ids must be unique")
	ErrChainSecretTooShort = fmt.Errorf(
		"aiAudit.chain.keys secrets must be at least %d characters",
		MinChainSecretLength,
	)
	ErrChainActiveKeyRequired = errors.New(
		"aiAudit.chain.activeKeyId is required when aiAudit.chain.keys are configured",
	)
	ErrChainActiveKeyUnknown = errors.New(
		"aiAudit.chain.activeKeyId must name one of aiAudit.chain.keys",
	)
	ErrChainActiveKeyWithoutKeys = errors.New(
		"aiAudit.chain.activeKeyId is set but aiAudit.chain.keys is empty",
	)
	ErrChainSecretInsecure = errors.New(
		"production and staging require aiAudit.chain.keys secrets that are not placeholders",
	)
	ErrChainKeysEnvMalformed = errors.New(
		"AI audit chain keys from the environment must be id:secret pairs separated by commas",
	)
	ErrExportRowsInvalid = errors.New(
		"aiAudit.export.syncMaxRows must not exceed aiAudit.export.maxRows",
	)
	ErrExportTTLInvalid  = errors.New("aiAudit.export.ttl cannot be negative")
	ErrProjectorInterval = fmt.Errorf(
		"aiAudit.projector.interval must be at least %s",
		minAIAuditProjectorInterval,
	)
)

type Config struct {
	Chain     ChainConfig     `mapstructure:"chain"`
	Export    ExportConfig    `mapstructure:"export"`
	Projector ProjectorConfig `mapstructure:"projector"`
}

type ChainKey struct {
	ID     string `mapstructure:"id"`
	Secret string `mapstructure:"secret"`
}

func (k ChainKey) String() string {
	return k.ID + aiAuditChainKeysEnvPairSeparator + redactedValue
}

func (k ChainKey) GoString() string {
	return "aiauditconfig.ChainKey{ID:" + strconv.Quote(k.ID) + ", Secret:" + redactedValue + "}"
}

type ChainConfig struct {
	Keys        []ChainKey `mapstructure:"keys"`
	ActiveKeyID string     `mapstructure:"activeKeyId"`
}

type ExportConfig struct {
	SyncMaxRows int           `mapstructure:"syncMaxRows" validate:"omitempty,min=1"`
	MaxRows     int           `mapstructure:"maxRows"     validate:"omitempty,min=1"`
	TTL         time.Duration `mapstructure:"ttl"`
}

type ProjectorConfig struct {
	Interval  time.Duration `mapstructure:"interval"`
	BatchSize int           `mapstructure:"batchSize" validate:"omitempty,min=1,max=10000"`
}

func (c *ChainConfig) Configured() bool {
	return len(c.Keys) > 0
}

func (c *ChainConfig) Key(id string) (ChainKey, bool) {
	for _, key := range c.Keys {
		if key.ID == id {
			return key, true
		}
	}

	return ChainKey{}, false
}

func (c *ChainConfig) ActiveKey() (ChainKey, bool) {
	if c.ActiveKeyID == "" {
		return ChainKey{}, false
	}

	return c.Key(c.ActiveKeyID)
}

func (c *ExportConfig) GetSyncMaxRows() int {
	if c.SyncMaxRows <= 0 {
		return defaultAIAuditExportSyncMaxRows
	}

	return c.SyncMaxRows
}

func (c *ExportConfig) GetMaxRows() int {
	if c.MaxRows <= 0 {
		return defaultAIAuditExportMaxRows
	}

	return c.MaxRows
}

func (c *ExportConfig) GetTTL() time.Duration {
	if c.TTL <= 0 {
		return defaultAIAuditExportTTL
	}

	return c.TTL
}

func (c *ProjectorConfig) GetInterval() time.Duration {
	if c.Interval <= 0 {
		return defaultAIAuditProjectorInterval
	}

	return c.Interval
}

func (c *ProjectorConfig) GetBatchSize() int {
	if c.BatchSize <= 0 {
		return defaultAIAuditProjectorBatchSize
	}

	return c.BatchSize
}

func Defaults() map[string]any {
	return map[string]any{
		"chain.activeKeyId":   "",
		"export.syncMaxRows":  defaultAIAuditExportSyncMaxRows,
		"export.maxRows":      defaultAIAuditExportMaxRows,
		"export.ttl":          defaultAIAuditExportTTL.String(),
		"projector.interval":  defaultAIAuditProjectorInterval.String(),
		"projector.batchSize": defaultAIAuditProjectorBatchSize,
	}
}

func (c *Config) Validate() error {
	if err := validateAIAuditChain(&c.Chain); err != nil {
		return err
	}

	export := &c.Export
	if export.GetSyncMaxRows() > export.GetMaxRows() {
		return ErrExportRowsInvalid
	}
	if export.TTL < 0 {
		return ErrExportTTLInvalid
	}

	if interval := c.Projector.Interval; interval != 0 &&
		interval < minAIAuditProjectorInterval {
		return ErrProjectorInterval
	}

	return nil
}

func validateAIAuditChain(chain *ChainConfig) error {
	if !chain.Configured() {
		if chain.ActiveKeyID != "" {
			return ErrChainActiveKeyWithoutKeys
		}

		return nil
	}

	seen := make(map[string]struct{}, len(chain.Keys))
	for i, key := range chain.Keys {
		if !validAIAuditChainKeyID(key.ID) {
			return fmt.Errorf("%w: entry %d", ErrChainKeyIDInvalid, i)
		}
		if _, duplicate := seen[key.ID]; duplicate {
			return fmt.Errorf("%w: %q", ErrChainKeyIDDuplicate, key.ID)
		}
		seen[key.ID] = struct{}{}
		if len(key.Secret) < MinChainSecretLength {
			return fmt.Errorf("%w: key %q", ErrChainSecretTooShort, key.ID)
		}
	}

	if chain.ActiveKeyID == "" {
		return ErrChainActiveKeyRequired
	}
	if _, ok := seen[chain.ActiveKeyID]; !ok {
		return fmt.Errorf("%w: %q", ErrChainActiveKeyUnknown, chain.ActiveKeyID)
	}

	return nil
}

func (c *Config) ValidateProduction() error {
	for _, key := range c.Chain.Keys {
		secret := strings.ToLower(key.Secret)
		for _, placeholder := range securityconfig.InsecureDefaultValues {
			if strings.Contains(secret, placeholder) {
				return fmt.Errorf("%w: key %q", ErrChainSecretInsecure, key.ID)
			}
		}
	}

	return nil
}

func validAIAuditChainKeyID(id string) bool {
	if id == "" || len(id) > maxAIAuditChainKeyIDLength {
		return false
	}

	for i := range len(id) {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '.', c == '_', c == '-':
		default:
			return false
		}
	}

	return true
}

func ParseChainKeys(raw string) ([]ChainKey, error) {
	entries := strings.Split(raw, aiAuditChainKeysEnvEntrySeparator)
	keys := make([]ChainKey, 0, len(entries))
	for i, entry := range entries {
		id, secret, found := strings.Cut(strings.TrimSpace(entry), aiAuditChainKeysEnvPairSeparator)
		if !found || id == "" || secret == "" {
			return nil, fmt.Errorf("%w: entry %d", ErrChainKeysEnvMalformed, i)
		}
		keys = append(keys, ChainKey{ID: id, Secret: secret})
	}

	return keys, nil
}
