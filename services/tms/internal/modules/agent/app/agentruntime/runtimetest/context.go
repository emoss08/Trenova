package runtimetest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
)

type StubOrganizations struct {
	tenantports.OrganizationRepository

	Org   *tenant.Organization
	Calls int
}

func (s *StubOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	s.Calls++

	return s.Org, nil
}

type StubUsers struct {
	tenantports.UserRepository

	User *tenant.User
}

func (s *StubUsers) GetByID(
	context.Context,
	tenantports.GetUserByIDRequest,
) (*tenant.User, error) {
	return s.User, nil
}
