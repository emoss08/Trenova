package seedhelpers

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/fatih/color"
	"github.com/uptrace/bun"
)

type configContextKey struct{}

func WithConfig(ctx context.Context, cfg *config.Config) context.Context {
	return context.WithValue(ctx, configContextKey{}, cfg)
}

func ConfigFromContext(ctx context.Context) *config.Config {
	cfg, _ := ctx.Value(configContextKey{}).(*config.Config)
	return cfg
}

var ErrRollbackNotSupported = errors.New("rollback not supported for this seed")

type BaseSeed struct {
	name         string
	version      string
	description  string
	environments []common.Environment
	dependsOn    []SeedID
}

func NewBaseSeed(name, version, description string, environments []common.Environment) *BaseSeed {
	return &BaseSeed{
		name:         name,
		version:      version,
		description:  description,
		environments: environments,
		dependsOn:    []SeedID{},
	}
}

func (b *BaseSeed) Name() string {
	return b.name
}

func (b *BaseSeed) Version() string {
	return b.version
}

func (b *BaseSeed) Description() string {
	return b.description
}

func (b *BaseSeed) Environment() []common.Environment {
	return b.environments
}

func (b *BaseSeed) Environments() []common.Environment {
	return b.environments
}

func (b *BaseSeed) SetDependencies(deps ...SeedID) {
	b.dependsOn = deps
}

func (b *BaseSeed) DependsOn() []string {
	result := make([]string, len(b.dependsOn))
	for i, dep := range b.dependsOn {
		result[i] = string(dep)
	}
	return result
}

func (b *BaseSeed) Dependencies() []string {
	return b.DependsOn()
}

func (b *BaseSeed) Down(ctx context.Context, tx bun.Tx) error {
	return ErrRollbackNotSupported
}

func (b *BaseSeed) CanRollback() bool {
	return false
}

func RunInTransaction(
	ctx context.Context,
	db bun.IDB,
	seedName string,
	logger SeedLogger,
	fn func(context.Context, bun.Tx, *SeedContext) error,
) error {
	if logger == nil {
		logger = NewNoOpLogger()
	}

	seedCtx := NewSeedContext(db, logger, ConfigFromContext(ctx))

	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := fn(ctx, tx, seedCtx); err != nil {
			logger.Error("Seed %s failed: %v", seedName, err)
			return fmt.Errorf("%s seed failed: %w", seedName, err)
		}
		return nil
	})
}

func LogSuccess(message string, details ...string) {
	color.Green("✓ %s", message)
	for _, detail := range details {
		fmt.Printf("  %s\n", detail)
	}
}
