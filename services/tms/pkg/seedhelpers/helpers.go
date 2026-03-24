package seedhelpers

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type BusinessUnitOptions struct {
	Name string
	Code string
}

func (opts *BusinessUnitOptions) Validate() error {
	if opts == nil {
		return fmt.Errorf("options: %w", ErrNilValue)
	}
	if opts.Name == "" {
		return fmt.Errorf("name: %w", ErrEmptyKey)
	}
	if opts.Code == "" {
		return fmt.Errorf("code: %w", ErrEmptyKey)
	}
	return nil
}

func (sc *SeedContext) CreateBusinessUnit(
	ctx context.Context,
	tx bun.Tx,
	opts *BusinessUnitOptions,
	seedName string,
) (*tenant.BusinessUnit, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %w", err)
	}

	bu := &tenant.BusinessUnit{
		Name: opts.Name,
		Code: opts.Code,
	}

	if _, err := tx.NewInsert().Model(bu).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create business unit %s: %w", opts.Name, err)
	}

	if err := sc.TrackCreated(ctx, "business_units", bu.ID, seedName); err != nil {
		return nil, fmt.Errorf("track business unit: %w", err)
	}

	sc.logger.EntityCreated("business_units", bu.ID, opts.Name)
	return bu, nil
}

func (sc *SeedContext) CreateOrganization(
	ctx context.Context,
	tx bun.Tx,
	opts *OrganizationOptions,
	seedName string,
) (*tenant.Organization, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %w", err)
	}

	now := timeutils.NowUnix()
	org := &tenant.Organization{
		BusinessUnitID: opts.BusinessUnitID,
		Name:           opts.Name,
		ScacCode:       opts.ScacCode,
		AddressLine1:   opts.AddressLine1,
		AddressLine2:   opts.AddressLine2,
		City:           opts.City,
		StateID:        opts.StateID,
		PostalCode:     opts.PostalCode,
		Timezone:       opts.Timezone,
		TaxID:          opts.TaxID,
		DOTNumber:      opts.DOTNumber,
		BucketName:     opts.BucketName,
		LogoURL:        opts.LogoURL,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if _, err := tx.NewInsert().Model(org).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create organization %s: %w", opts.Name, err)
	}

	if err := sc.TrackCreated(ctx, "organizations", org.ID, seedName); err != nil {
		return nil, fmt.Errorf("track organization: %w", err)
	}

	sc.logger.EntityCreated("organizations", org.ID, opts.Name)
	return org, nil
}

func (sc *SeedContext) CreateUser(
	ctx context.Context,
	tx bun.Tx,
	opts *UserOptions,
	seedName string,
) (*tenant.User, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %w", err)
	}

	now := timeutils.NowUnix()
	status := opts.Status
	if status == "" {
		status = domaintypes.StatusActive
	}

	user := &tenant.User{
		CurrentOrganizationID: opts.OrganizationID,
		BusinessUnitID:        opts.BusinessUnitID,
		Name:                  opts.Name,
		Username:              opts.Username,
		EmailAddress:          opts.Email,
		Status:                status,
		Timezone:              opts.Timezone,
		IsPlatformAdmin:       opts.IsPlatformAdmin,
		MustChangePassword:    opts.MustChangePassword,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if opts.Password != "" {
		hashedPassword, err := user.GeneratePassword(opts.Password)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		user.Password = hashedPassword
	}

	if _, err := tx.NewInsert().Model(user).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create user %s: %w", opts.Username, err)
	}

	if err := sc.TrackCreated(ctx, "users", user.ID, seedName); err != nil {
		return nil, fmt.Errorf("track user: %w", err)
	}

	sc.logger.EntityCreated("users", user.ID, opts.Name)
	return user, nil
}

func (sc *SeedContext) GetStateByAbbreviation(ctx context.Context, abbr string) (pulid.ID, error) {
	state, err := sc.GetState(ctx, abbr)
	if err != nil {
		return "", err
	}
	return state.ID, nil
}
