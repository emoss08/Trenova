package base

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

var ErrKnownCredentialsOutsideDevelopment = errors.New(
	"refusing to create accounts with publicly known passwords outside development and test",
)

func defaultOrganization(
	ctx context.Context,
	sc *seedhelpers.SeedContext,
) (*tenant.Organization, error) {
	if org, err := sc.GetOrganization("default_org"); err == nil {
		return org, nil
	}

	org, err := sc.GetDefaultOrganization(ctx)
	switch {
	case err == nil:
		return org, nil
	case dberror.IsNotFoundError(err):
		return nil, nil
	default:
		return nil, fmt.Errorf("get default organization: %w", err)
	}
}

func listOrganizations(ctx context.Context, tx bun.Tx) ([]tenant.Organization, error) {
	var orgs []tenant.Organization
	if err := tx.NewSelect().Model(&orgs).Order("created_at ASC").Scan(ctx); err != nil {
		return nil, fmt.Errorf("get organizations: %w", err)
	}

	return orgs, nil
}

func requireDevelopmentFixtures(sc *seedhelpers.SeedContext) error {
	cfg := sc.Config()
	if cfg == nil {
		return nil
	}
	if cfg.App.IsProduction() || cfg.App.IsStaging() {
		return ErrKnownCredentialsOutsideDevelopment
	}

	return nil
}
