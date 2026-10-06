package seedaccountport

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	SeedName       = "AdminAccount"
	SeededPassword = "admin123!"
	ReferenceCap   = 1000
)

type KnownAccount struct {
	Username     string
	EmailAddress string
}

type KnownOrganization struct {
	Name     string
	ScacCode string
}

func KnownAccounts() []KnownAccount {
	return []KnownAccount{
		{Username: "admin", EmailAddress: "admin@trenova.app"},
		{Username: "admin-logistics", EmailAddress: "admin.logistics@trenova.app"},
		{Username: "admin-transport", EmailAddress: "admin.transport@trenova.app"},
	}
}

func KnownOrganizations() []KnownOrganization {
	return []KnownOrganization{
		{Name: "Trenova Logistics", ScacCode: "TRNV"},
		{Name: "Trenova Transportation", ScacCode: "TTNV"},
	}
}

type Reference struct {
	Table  string `json:"table"`
	Column string `json:"column"`
	Rows   int64  `json:"rows"`
}

type Membership struct {
	ID             pulid.ID `json:"id"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

type RoleAssignment struct {
	ID             pulid.ID `json:"id"`
	OrganizationID pulid.ID `json:"organizationId"`
	RoleID         pulid.ID `json:"roleId"`
}

type APIKey struct {
	ID             pulid.ID `json:"id"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Name           string   `json:"name"`
	KeyPrefix      string   `json:"keyPrefix"`
}

type MFAAuthenticator struct {
	ID             pulid.ID `json:"id"`
	OrganizationID pulid.ID `json:"organizationId"`
}

type UserFootprint struct {
	ID                    pulid.ID           `json:"id"`
	BusinessUnitID        pulid.ID           `json:"businessUnitId"`
	CurrentOrganizationID pulid.ID           `json:"currentOrganizationId"`
	Username              string             `json:"username"`
	EmailAddress          string             `json:"emailAddress"`
	Name                  string             `json:"name"`
	Status                domaintypes.Status `json:"status"`
	IsLocked              bool               `json:"isLocked"`
	PasswordHash          string             `json:"-"`
	SeedTracked           bool               `json:"seedTracked"`
	Memberships           []Membership       `json:"memberships"`
	RoleAssignments       []RoleAssignment   `json:"roleAssignments"`
	ActiveAPIKeys         []APIKey           `json:"activeApiKeys"`
	MFAAuthenticators     []MFAAuthenticator `json:"mfaAuthenticators"`
	OpenResetTokens       int64              `json:"openResetTokens"`
	References            []Reference        `json:"references"`
}

type OrganizationFootprint struct {
	ID             pulid.ID    `json:"id"`
	BusinessUnitID pulid.ID    `json:"businessUnitId"`
	Name           string      `json:"name"`
	ScacCode       string      `json:"scacCode"`
	SeedTracked    bool        `json:"seedTracked"`
	Users          []string    `json:"users"`
	History        []Reference `json:"history"`
	Data           []Reference `json:"data"`
}

type SystemUser struct {
	ID             pulid.ID `json:"id"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

type FindUsersRequest struct {
	Accounts  []KnownAccount
	SeedName  string
	ForUpdate bool
}

type InspectOrganizationsRequest struct {
	Organizations []KnownOrganization
	SeedName      string
	ExcludeUsers  []pulid.ID
	ExcludeOwners []pulid.ID
	ForUpdate     bool
}

type CountReferencesRequest struct {
	UserID        pulid.ID
	ExcludeOwners []pulid.ID
}

type TransactionOptions struct {
	ReadOnly bool
}

type StripUserRequest struct {
	UserID      pulid.ID
	RevokedByID pulid.ID
	Now         int64
}

type StripUserResult struct {
	Memberships       []Membership       `json:"memberships"`
	RoleAssignments   []RoleAssignment   `json:"roleAssignments"`
	APIKeys           []APIKey           `json:"apiKeys"`
	MFAAuthenticators []MFAAuthenticator `json:"mfaAuthenticators"`
	ResetTokens       int64              `json:"resetTokens"`
}

type RemoveUserRequest struct {
	UserID       pulid.ID
	PasswordHash string
	Now          int64
}

type RemoveUserResult struct {
	Deleted        bool        `json:"deleted"`
	Disabled       bool        `json:"disabled"`
	References     []Reference `json:"references"`
	RetainedReason string      `json:"retainedReason,omitempty"`
}

type RemoveOrganizationResult struct {
	Deleted        bool   `json:"deleted"`
	RetainedReason string `json:"retainedReason,omitempty"`
}

type Repository interface {
	InTransaction(
		ctx context.Context,
		opts TransactionOptions,
		fn func(ctx context.Context) error,
	) error
	FindUsers(ctx context.Context, req *FindUsersRequest) ([]*UserFootprint, error)
	CountReferences(ctx context.Context, req *CountReferencesRequest) ([]Reference, error)
	FindSystemUser(ctx context.Context) (*SystemUser, error)
	InspectOrganizations(
		ctx context.Context,
		req *InspectOrganizationsRequest,
	) ([]*OrganizationFootprint, error)
	StripUser(ctx context.Context, req *StripUserRequest) (*StripUserResult, error)
	RemoveUser(ctx context.Context, req *RemoveUserRequest) (*RemoveUserResult, error)
	RemoveOrganization(
		ctx context.Context,
		org *OrganizationFootprint,
	) (*RemoveOrganizationResult, error)
}
