package seedhelpers

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/fatih/color"
	"github.com/uptrace/bun"
)

type BaseSeed struct {
	name         string
	version      string
	description  string
	environments []common.Environment
	dependsOn    []string
}

func NewBaseSeed(name, version, description string, environments []common.Environment) *BaseSeed {
	return &BaseSeed{
		name:         name,
		version:      version,
		description:  description,
		environments: environments,
		dependsOn:    []string{},
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

func (b *BaseSeed) SetDependencies(deps ...string) {
	b.dependsOn = deps
}

func (b *BaseSeed) DependsOn() []string {
	return b.dependsOn
}

func RunInTransaction(
	ctx context.Context,
	db *bun.DB,
	seedName string,
	fn func(context.Context, bun.Tx, *SeedContext) error,
) error {
	seedCtx := NewSeedContext(ctx, db)

	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := fn(ctx, tx, seedCtx); err != nil {
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
