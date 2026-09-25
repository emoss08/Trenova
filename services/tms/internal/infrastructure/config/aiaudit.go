package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAIAuditExportSyncMaxRows   = 5000
	defaultAIAuditExportMaxRows       = 1_000_000
	defaultAIAuditExportTTL           = 7 * 24 * time.Hour
	defaultAIAuditProjectorInterval   = time.Minute
	defaultAIAuditProjectorBatchSize  = 1000
	minAIAuditProjectorInterval       = 10 * time.Second
	maxAIAuditProjectorBatchSize      = 10_000
	MinAIAuditChainSecretLength       = 32
	maxAIAuditChainKeyIDLength        = 40
	aiAuditChainKeysEnvSuffix         = "_AI_AUDIT_CHAIN_KEYS"
	aiAuditChainActiveKeyIDEnvSuffix  = "_AI_AUDIT_CHAIN_ACTIVE_KEY_ID"
	aiAuditChainKeysEnvEntrySeparator = ","
	aiAuditChainKeysEnvPairSeparator  = ":"
	redactedValue                     = "[redacted]"
)

var (
	ErrAIAuditChainKeyIDInvalid = errors.New(
		"aiAudit.chain.keys ids must be 1 to 40 letters, digits, '.', '_' or '-'",
	)
	ErrAIAuditChainKeyIDDuplicate = errors.New("aiAudit.chain.keys ids must be unique")
	ErrAIAuditChainSecretTooShort = fmt.Errorf(
		"aiAudit.chain.keys secrets must be at least %d characters",
		MinAIAuditChainSecretLength,
	)
	ErrAIAuditChainActiveKeyRequired = errors.New(
		"aiAudit.chain.activeKeyId is required when aiAudit.chain.keys are configured",
	)
	ErrAIAuditChainActiveKeyUnknown = errors.New(
		"aiAudit.chain.activeKeyId must name one of aiAudit.chain.keys",
	)
	ErrAIAuditChainActiveKeyWithoutKeys = errors.New(
		"aiAudit.chain.activeKeyId is set but aiAudit.chain.keys is empty",
	)
	ErrAIAuditChainSecretInsecure = errors.New(
		"production and staging require aiAudit.chain.keys secrets that are not placeholders",
	)
	ErrAIAuditChainKeysEnvMalformed = errors.New(
		"AI audit chain keys from the environment must be id:secret pairs separated by commas",
	)
	ErrAIAuditExportRowsInvalid = errors.New(
		"aiAudit.export.syncMaxRows must not exceed aiAudit.export.maxRows",
	)
	ErrAIAuditExportTTLInvalid  = errors.New("aiAudit.export.ttl cannot be negative")
	ErrAIAuditProjectorInterval = fmt.Errorf(
		"aiAudit.projector.interval must be at least %s",
		minAIAuditProjectorInterval,
	)
)

type AIAuditConfig struct {
	Chain     AIAuditChainConfig     `mapstructure:"chain"`
	Export    AIAuditExportConfig    `mapstructure:"export"`
	Projector AIAuditProjectorConfig `mapstructure:"projector"`
}

type AIAuditChainKey struct {
	ID     string `mapstructure:"id"`
	Secret string `mapstructure:"secret"`
}

func (k AIAuditChainKey) String() string {
	return k.ID + aiAuditChainKeysEnvPairSeparator + redactedValue
}

func (k AIAuditChainKey) GoString() string {
	return "config.AIAuditChainKey{ID:" + strconv.Quote(k.ID) + ", Secret:" + redactedValue + "}"
}

type AIAuditChainConfig struct {
	Keys        []AIAuditChainKey `mapstructure:"keys"`
	ActiveKeyID string            `mapstructure:"activeKeyId"`
}

type AIAuditExportConfig struct {
	SyncMaxRows int           `mapstructure:"syncMaxRows" validate:"omitempty,min=1"`
	MaxRows     int           `mapstructure:"maxRows"     validate:"omitempty,min=1"`
	TTL         time.Duration `mapstructure:"ttl"`
}

type AIAuditProjectorConfig struct {
	Interval  time.Duration `mapstructure:"interval"`
	BatchSize int           `mapstructure:"batchSize" validate:"omitempty,min=1,max=10000"`
}

func (c *AIAuditChainConfig) Configured() bool {
	return len(c.Keys) > 0
}

func (c *AIAuditChainConfig) Key(id string) (AIAuditChainKey, bool) {
	for _, key := range c.Keys {
		if key.ID == id {
			return key, true
		}
	}

	return AIAuditChainKey{}, false
}

func (c *AIAuditChainConfig) ActiveKey() (AIAuditChainKey, bool) {
	if c.ActiveKeyID == "" {
		return AIAuditChainKey{}, false
	}

	return c.Key(c.ActiveKeyID)
}

func (c *AIAuditExportConfig) GetSyncMaxRows() int {
	if c.SyncMaxRows <= 0 {
		return defaultAIAuditExportSyncMaxRows
	}

	return c.SyncMaxRows
}

func (c *AIAuditExportConfig) GetMaxRows() int {
	if c.MaxRows <= 0 {
		return defaultAIAuditExportMaxRows
	}

	return c.MaxRows
}

func (c *AIAuditExportConfig) GetTTL() time.Duration {
	if c.TTL <= 0 {
		return defaultAIAuditExportTTL
	}

	return c.TTL
}

func (c *AIAuditProjectorConfig) GetInterval() time.Duration {
	if c.Interval <= 0 {
		return defaultAIAuditProjectorInterval
	}

	return c.Interval
}

func (c *AIAuditProjectorConfig) GetBatchSize() int {
	if c.BatchSize <= 0 {
		return defaultAIAuditProjectorBatchSize
	}

	return c.BatchSize
}

func validateAIAuditConfig(config *Config) error {
	if err := validateAIAuditChain(&config.AIAudit.Chain); err != nil {
		return err
	}

	export := &config.AIAudit.Export
	if export.GetSyncMaxRows() > export.GetMaxRows() {
		return ErrAIAuditExportRowsInvalid
	}
	if export.TTL < 0 {
		return ErrAIAuditExportTTLInvalid
	}

	if interval := config.AIAudit.Projector.Interval; interval != 0 &&
		interval < minAIAuditProjectorInterval {
		return ErrAIAuditProjectorInterval
	}

	return nil
}

func validateAIAuditChain(chain *AIAuditChainConfig) error {
	if !chain.Configured() {
		if chain.ActiveKeyID != "" {
			return ErrAIAuditChainActiveKeyWithoutKeys
		}

		return nil
	}

	seen := make(map[string]struct{}, len(chain.Keys))
	for i, key := range chain.Keys {
		if !validAIAuditChainKeyID(key.ID) {
			return fmt.Errorf("%w: entry %d", ErrAIAuditChainKeyIDInvalid, i)
		}
		if _, duplicate := seen[key.ID]; duplicate {
			return fmt.Errorf("%w: %q", ErrAIAuditChainKeyIDDuplicate, key.ID)
		}
		seen[key.ID] = struct{}{}
		if len(key.Secret) < MinAIAuditChainSecretLength {
			return fmt.Errorf("%w: key %q", ErrAIAuditChainSecretTooShort, key.ID)
		}
	}

	if chain.ActiveKeyID == "" {
		return ErrAIAuditChainActiveKeyRequired
	}
	if _, ok := seen[chain.ActiveKeyID]; !ok {
		return fmt.Errorf("%w: %q", ErrAIAuditChainActiveKeyUnknown, chain.ActiveKeyID)
	}

	return nil
}

func validateAIAuditChainSecrets(config *Config) error {
	for _, key := range config.AIAudit.Chain.Keys {
		secret := strings.ToLower(key.Secret)
		for _, placeholder := range InsecureDefaultValues {
			if strings.Contains(secret, placeholder) {
				return fmt.Errorf("%w: key %q", ErrAIAuditChainSecretInsecure, key.ID)
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

func parseAIAuditChainKeys(raw string) ([]AIAuditChainKey, error) {
	entries := strings.Split(raw, aiAuditChainKeysEnvEntrySeparator)
	keys := make([]AIAuditChainKey, 0, len(entries))
	for i, entry := range entries {
		id, secret, found := strings.Cut(strings.TrimSpace(entry), aiAuditChainKeysEnvPairSeparator)
		if !found || id == "" || secret == "" {
			return nil, fmt.Errorf("%w: entry %d", ErrAIAuditChainKeysEnvMalformed, i)
		}
		keys = append(keys, AIAuditChainKey{ID: id, Secret: secret})
	}

	return keys, nil
}
