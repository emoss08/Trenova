package seedhelpers

import (
	"fmt"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type OrganizationOptions struct {
	BusinessUnitID pulid.ID
	Name           string
	ScacCode       string
	AddressLine1   string
	AddressLine2   string
	City           string
	StateID        pulid.ID
	PostalCode     string
	Timezone       string
	TaxID          string
	DOTNumber      string
	BucketName     string
	LogoURL        string
}

func (opts *OrganizationOptions) Validate() error {
	if opts == nil {
		return fmt.Errorf("options: %w", ErrNilValue)
	}
	if opts.BusinessUnitID == "" {
		return fmt.Errorf("business unit ID: %w", ErrEmptyKey)
	}
	if opts.Name == "" {
		return fmt.Errorf("name: %w", ErrEmptyKey)
	}
	if opts.ScacCode == "" {
		return fmt.Errorf("SCAC code: %w", ErrEmptyKey)
	}
	if opts.City == "" {
		return fmt.Errorf("city: %w", ErrEmptyKey)
	}
	if opts.StateID == "" {
		return fmt.Errorf("state ID: %w", ErrEmptyKey)
	}
	if opts.Timezone == "" {
		return fmt.Errorf("timezone: %w", ErrEmptyKey)
	}
	if opts.DOTNumber == "" {
		return fmt.Errorf("DOT number: %w", ErrEmptyKey)
	}
	if opts.BucketName == "" {
		return fmt.Errorf("bucket name: %w", ErrEmptyKey)
	}
	return nil
}

type UserOptions struct {
	OrganizationID     pulid.ID
	BusinessUnitID     pulid.ID
	Name               string
	Username           string
	Email              string
	Password           string
	Status             domaintypes.Status
	Timezone           string
	IsAdmin            bool
	IsPlatformAdmin    bool
	MustChangePassword bool
}

func (opts *UserOptions) Validate() error {
	if opts == nil {
		return fmt.Errorf("options: %w", ErrNilValue)
	}
	if opts.OrganizationID == "" {
		return fmt.Errorf("organization ID: %w", ErrEmptyKey)
	}
	if opts.BusinessUnitID == "" {
		return fmt.Errorf("business unit ID: %w", ErrEmptyKey)
	}
	if opts.Name == "" {
		return fmt.Errorf("name: %w", ErrEmptyKey)
	}
	if opts.Username == "" {
		return fmt.Errorf("username: %w", ErrEmptyKey)
	}
	if opts.Email == "" {
		return fmt.Errorf("email: %w", ErrEmptyKey)
	}
	if opts.Timezone == "" {
		return fmt.Errorf("timezone: %w", ErrEmptyKey)
	}
	return nil
}
