package seedhelpers

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
)

const (
	defaultTimezone = "America/Los_Angeles"
)

type UserOptions struct {
	Name           string
	Username       string
	Email          string
	Password       string // defaults to "password123!"
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Timezone       string // defaults to "America/Los_Angeles"
	Status         domaintypes.Status
}

func (sc *SeedContext) CreateUser(
	tx bun.Tx,
	opts *UserOptions,
) (*tenant.User, error) {
	if opts.Password == "" {
		opts.Password = "password123!"
	}
	if opts.Timezone == "" {
		opts.Timezone = defaultTimezone
	}
	if opts.Status == "" {
		opts.Status = domaintypes.StatusActive
	}

	if opts.OrganizationID == "" {
		org, err := sc.GetDefaultOrganization()
		if err != nil {
			return nil, err
		}
		opts.OrganizationID = org.ID
	}
	if opts.BusinessUnitID == "" {
		bu, err := sc.GetDefaultBusinessUnit()
		if err != nil {
			return nil, err
		}
		opts.BusinessUnitID = bu.ID
	}

	user := &tenant.User{
		CurrentOrganizationID: opts.OrganizationID,
		BusinessUnitID:        opts.BusinessUnitID,
		Name:                  opts.Name,
		Username:              opts.Username,
		EmailAddress:          opts.Email,
		Status:                opts.Status,
		Timezone:              opts.Timezone,
		CreatedAt:             utils.NowUnix(),
		UpdatedAt:             utils.NowUnix(),
	}

	hashedPassword, err := user.GeneratePassword(opts.Password)
	if err != nil {
		return nil, fmt.Errorf("generate password for %s: %w", opts.Username, err)
	}
	user.Password = hashedPassword

	if _, err = tx.NewInsert().Model(user).Exec(sc.ctx); err != nil {
		return nil, fmt.Errorf("create user %s: %w", opts.Username, err)
	}

	return user, nil
}

func (sc *SeedContext) AssignRoleToUser(tx bun.Tx, user *tenant.User, roleName string) error {
	role, err := sc.GetRole(roleName)
	if err != nil {
		return err
	}

	userRole := &tenant.OrganizationMembership{
		OrganizationID: user.CurrentOrganizationID,
		UserID:         user.ID,
		RoleIDs:        []pulid.ID{role.ID},
	}

	if _, err = tx.NewInsert().Model(userRole).Exec(sc.ctx); err != nil {
		return fmt.Errorf("assign role %s to user %s: %w", roleName, user.Username, err)
	}

	return nil
}

type OrgOptions struct {
	Name           string
	ScacCode       string
	DOTNumber      string
	BusinessUnitID pulid.ID
	StateID        pulid.ID
	City           string
	PostalCode     string
	AddressLine1   string
	OrgType        tenant.Type
	Timezone       string // defaults to "America/Los_Angeles"
	BucketName     string
}

func (sc *SeedContext) CreateOrganization(
	tx bun.Tx,
	opts *OrgOptions,
) (*tenant.Organization, error) {
	// Apply defaults
	if opts.Timezone == "" {
		opts.Timezone = defaultTimezone
	}
	if opts.OrgType == "" {
		opts.OrgType = tenant.TypeCarrier
	}
	if opts.BusinessUnitID == "" {
		bu, err := sc.GetDefaultBusinessUnit()
		if err != nil {
			return nil, err
		}
		opts.BusinessUnitID = bu.ID
	}

	org := &tenant.Organization{
		BusinessUnitID: opts.BusinessUnitID,
		Name:           opts.Name,
		ScacCode:       opts.ScacCode,
		DOTNumber:      opts.DOTNumber,
		StateID:        opts.StateID,
		City:           opts.City,
		PostalCode:     opts.PostalCode,
		AddressLine1:   opts.AddressLine1,
		OrgType:        opts.OrgType,
		Timezone:       opts.Timezone,
		BucketName:     opts.BucketName,
		CreatedAt:      utils.NowUnix(),
		UpdatedAt:      utils.NowUnix(),
	}

	if _, err := tx.NewInsert().Model(org).Exec(sc.ctx); err != nil {
		return nil, fmt.Errorf("create organization %s: %w", opts.Name, err)
	}

	if err := sc.CreateOrganizationSettings(tx, org); err != nil {
		return nil, err
	}

	return org, nil
}

func (sc *SeedContext) CreateOrganizationSettings(tx bun.Tx, org *tenant.Organization) error {
	shipmentControl := &tenant.ShipmentControl{
		ID:             pulid.MustNew("sc_"),
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		CreatedAt:      utils.NowUnix(),
		UpdatedAt:      utils.NowUnix(),
	}

	if _, err := tx.NewInsert().Model(shipmentControl).Exec(sc.ctx); err != nil {
		return fmt.Errorf("create shipment control: %w", err)
	}

	billingControl := &tenant.BillingControl{
		ID:             pulid.MustNew("bc_"),
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		CreatedAt:      utils.NowUnix(),
		UpdatedAt:      utils.NowUnix(),
	}

	if _, err := tx.NewInsert().Model(billingControl).Exec(sc.ctx); err != nil {
		return fmt.Errorf("create billing control: %w", err)
	}

	dataRetention := &tenant.DataRetention{
		ID:             pulid.MustNew("dr_"),
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		CreatedAt:      utils.NowUnix(),
		UpdatedAt:      utils.NowUnix(),
	}

	if _, err := tx.NewInsert().Model(dataRetention).Exec(sc.ctx); err != nil {
		return fmt.Errorf("create data retention: %w", err)
	}

	return nil
}

type BusinessUnitOptions struct {
	Name           string
	Code           string
	PrimaryContact string
	PrimaryEmail   string
	PrimaryPhone   string
	AddressLine1   string
	City           string
	StateID        pulid.ID
	PostalCode     string
	TaxID          string
	Description    string
	Timezone       string // defaults to "America/Los_Angeles"
	Locale         string // defaults to "en-US"
}

func (sc *SeedContext) CreateBusinessUnit(
	tx bun.Tx,
	opts *BusinessUnitOptions,
) (*tenant.BusinessUnit, error) {
	if opts.Timezone == "" {
		opts.Timezone = defaultTimezone
	}
	if opts.Locale == "" {
		opts.Locale = "en-US"
	}

	bu := &tenant.BusinessUnit{
		Name:           opts.Name,
		Code:           opts.Code,
		PrimaryContact: opts.PrimaryContact,
		PrimaryEmail:   opts.PrimaryEmail,
		PrimaryPhone:   opts.PrimaryPhone,
		AddressLine1:   opts.AddressLine1,
		City:           opts.City,
		StateID:        opts.StateID,
		PostalCode:     opts.PostalCode,
		TaxID:          opts.TaxID,
		Description:    opts.Description,
		Timezone:       opts.Timezone,
		Locale:         opts.Locale,
	}

	if _, err := tx.NewInsert().Model(bu).Exec(sc.ctx); err != nil {
		return nil, fmt.Errorf("create business unit %s: %w", opts.Name, err)
	}

	return bu, nil
}

func (sc *SeedContext) GetRoleByName(name string) (*permission.Role, error) {
	return sc.GetRole(name)
}

func (sc *SeedContext) GetStateByAbbreviation(abbr string) (pulid.ID, error) {
	state, err := sc.GetState(abbr)
	if err != nil {
		return "", err
	}
	return state.ID, nil
}
