package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/shared/edi/internal/config"
	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/transactions"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// TransactionRegistry manages all transaction types and their builders
type TransactionRegistry struct {
	mu            sync.RWMutex
	builders      map[string]transactions.TransactionBuilder
	configs       map[string]*config.TransactionConfig
	configManager *config.ConfigManager
	segRegistry   *segments.SegmentRegistry
	delims        x12.Delimiters
}

// NewTransactionRegistry creates a new transaction registry
func NewTransactionRegistry(
	segRegistry *segments.SegmentRegistry,
	delims x12.Delimiters,
) *TransactionRegistry {
	return &TransactionRegistry{
		builders:      make(map[string]transactions.TransactionBuilder),
		configs:       make(map[string]*config.TransactionConfig),
		configManager: config.NewConfigManager(),
		segRegistry:   segRegistry,
		delims:        delims,
	}
}

// RegisterBuilder registers a transaction builder
func (r *TransactionRegistry) RegisterBuilder(
	txType, version string,
	builder transactions.TransactionBuilder,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.makeKey(txType, version)
	r.builders[key] = builder
	return nil
}

// RegisterConfig registers a transaction configuration
func (r *TransactionRegistry) RegisterConfig(cfg *config.TransactionConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.makeKey(cfg.TransactionType, cfg.Version)
	r.configs[key] = cfg

	return r.configManager.SaveConfig(cfg)
}

// GetBuilder retrieves a builder for a transaction type
func (r *TransactionRegistry) GetBuilder(
	txType, version string,
) (transactions.TransactionBuilder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := r.makeKey(txType, version)

	if builder, exists := r.builders[key]; exists {
		return builder, nil
	}

	if cfg, exists := r.configs[key]; exists {
		return r.createConfigurableBuilder(cfg, nil)
	}

	cfg, err := r.configManager.GetConfig(txType, version)
	if err == nil {
		return r.createConfigurableBuilder(cfg, nil)
	}

	return nil, fmt.Errorf("no builder found for %s version %s", txType, version)
}

// GetBuilderForCustomer retrieves a builder with customer-specific configuration
func (r *TransactionRegistry) GetBuilderForCustomer(
	txType, version, customerID string,
) (transactions.TransactionBuilder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cfg, err := r.configManager.GetConfig(txType, version)
	if err != nil {
		return nil, fmt.Errorf("no configuration found for %s version %s", txType, version)
	}

	customerCfg, err := r.configManager.GetCustomerConfig(txType, version, customerID)
	if err != nil {
		return r.createConfigurableBuilder(cfg, nil)
	}

	return r.createConfigurableBuilder(cfg, customerCfg)
}

// Build builds a transaction using the appropriate builder
func (r *TransactionRegistry) Build(
	ctx context.Context,
	txType, version string,
	data any,
) (string, error) {
	builder, err := r.GetBuilder(txType, version)
	if err != nil {
		return "", err
	}

	return builder.Build(ctx, data)
}

// BuildForCustomer builds a transaction with customer-specific configuration
func (r *TransactionRegistry) BuildForCustomer(
	ctx context.Context,
	txType, version, customerID string,
	data any,
) (string, error) {
	builder, err := r.GetBuilderForCustomer(txType, version, customerID)
	if err != nil {
		return "", err
	}

	return builder.Build(ctx, data)
}

// Parse parses EDI segments using the appropriate builder
func (r *TransactionRegistry) Parse(
	ctx context.Context,
	txType, version string,
	segments []x12.Segment,
) (any, error) {
	builder, err := r.GetBuilder(txType, version)
	if err != nil {
		return nil, err
	}

	return builder.Parse(ctx, segments)
}

// ParseForCustomer parses EDI segments with customer-specific configuration
func (r *TransactionRegistry) ParseForCustomer(
	ctx context.Context,
	txType, version, customerID string,
	segments []x12.Segment,
) (any, error) {
	builder, err := r.GetBuilderForCustomer(txType, version, customerID)
	if err != nil {
		return nil, err
	}

	return builder.Parse(ctx, segments)
}

// ListTransactionTypes returns all registered transaction types
func (r *TransactionRegistry) ListTransactionTypes() []TransactionTypeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	var types []TransactionTypeInfo

	for key, builder := range r.builders {
		if !seen[key] {
			types = append(types, TransactionTypeInfo{
				Type:       builder.GetTransactionType(),
				Version:    builder.GetVersion(),
				HasBuilder: true,
				HasConfig:  r.configs[key] != nil,
			})
			seen[key] = true
		}
	}

	for key, cfg := range r.configs {
		if !seen[key] {
			types = append(types, TransactionTypeInfo{
				Type:        cfg.TransactionType,
				Version:     cfg.Version,
				Name:        cfg.Name,
				Description: cfg.Description,
				HasBuilder:  false,
				HasConfig:   true,
			})
			seen[key] = true
		}
	}

	return types
}

// LoadDefaultTransactions loads default transaction configurations
func (r *TransactionRegistry) LoadDefaultTransactions() error {
	builder997 := transactions.NewAck997Builder(r.segRegistry, "004010", r.delims)
	if err := r.RegisterBuilder("997", "004010", builder997); err != nil {
		return err
	}

	if err := r.RegisterConfig(config.Example997Config()); err != nil {
		return err
	}

	builder999 := transactions.NewAck999Builder(r.segRegistry, "004010", r.delims)
	if err := r.RegisterBuilder("999", "004010", builder999); err != nil {
		return err
	}

	builder204 := transactions.NewTX204Builder(r.segRegistry, "004010", r.delims)
	if err := r.RegisterBuilder("204", "004010", builder204); err != nil {
		return err
	}

	if err := r.RegisterConfig(config.Example204Config()); err != nil {
		return err
	}

	return nil
}

func (r *TransactionRegistry) makeKey(txType, version string) string {
	return fmt.Sprintf("%s:%s", txType, version)
}

func (r *TransactionRegistry) createConfigurableBuilder(
	cfg *config.TransactionConfig,
	customerCfg *config.CustomerConfig,
) (transactions.TransactionBuilder, error) {
	return &ConfigurableTransactionBuilder{
		builder: config.NewConfigurableBuilder(cfg, customerCfg, r.segRegistry, r.delims),
		config:  cfg,
	}, nil
}

// TransactionTypeInfo provides information about a registered transaction type
type TransactionTypeInfo struct {
	Type        string `json:"type"`
	Version     string `json:"version"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	HasBuilder  bool   `json:"has_builder"`
	HasConfig   bool   `json:"has_config"`
}

// ConfigurableTransactionBuilder wraps a configurable builder to implement TransactionBuilder interface
type ConfigurableTransactionBuilder struct {
	builder *config.ConfigurableBuilder
	config  *config.TransactionConfig
}

// Build constructs the transaction from provided data
func (b *ConfigurableTransactionBuilder) Build(ctx context.Context, data any) (string, error) {
	return b.builder.BuildFromObject(ctx, data)
}

// Parse parses raw segments into transaction-specific structure
func (b *ConfigurableTransactionBuilder) Parse(
	ctx context.Context,
	segments []x12.Segment,
) (any, error) {
	return b.builder.ParseToObject(ctx, segments)
}

// GetTransactionType returns the transaction type
func (b *ConfigurableTransactionBuilder) GetTransactionType() string {
	return b.config.TransactionType
}

// GetVersion returns the X12 version
func (b *ConfigurableTransactionBuilder) GetVersion() string {
	return b.config.Version
}
