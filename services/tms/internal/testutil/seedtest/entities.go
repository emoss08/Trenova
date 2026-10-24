package seedtest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/require"
)

type BusinessUnitBuilder struct {
	bu *tenant.BusinessUnit
}

func NewBusinessUnit() *BusinessUnitBuilder {
	return &BusinessUnitBuilder{
		bu: &tenant.BusinessUnit{
			Name: "Test Business Unit",
			Code: "TEST",
		},
	}
}

func (b *BusinessUnitBuilder) WithName(name string) *BusinessUnitBuilder {
	b.bu.Name = name
	return b
}

func (b *BusinessUnitBuilder) WithCode(code string) *BusinessUnitBuilder {
	b.bu.Code = code
	return b
}

func (b *BusinessUnitBuilder) Build(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
) *tenant.BusinessUnit {
	t.Helper()

	err := tenantdb.BusinessUnits.Create(systemContext(ctx), db, b.bu)
	require.NoError(t, err, "failed to insert test business unit")

	return b.bu
}

type OrganizationBuilder struct {
	org *tenant.Organization
}

func NewOrganization(buID pulid.ID, stateID pulid.ID) *OrganizationBuilder {
	now := timeutils.NowUnix()
	return &OrganizationBuilder{
		org: &tenant.Organization{
			BusinessUnitID:         buID,
			BrokerageEnabled:       true,
			AssetOperationsEnabled: true,
			Name:                   "Test Organization",
			ScacCode:               "TEST",
			DOTNumber:              "1234567",
			AddressLine1:           "123 Test St",
			City:                   "Test City",
			StateID:                stateID,
			PostalCode:             "12345",
			Timezone:               "America/Los_Angeles",
			BucketName:             "test-bucket",
			CreatedAt:              now,
			UpdatedAt:              now,
		},
	}
}

func (b *OrganizationBuilder) WithName(name string) *OrganizationBuilder {
	b.org.Name = name
	return b
}

func (b *OrganizationBuilder) WithScacCode(code string) *OrganizationBuilder {
	b.org.ScacCode = code
	return b
}

func (b *OrganizationBuilder) WithDOTNumber(dot string) *OrganizationBuilder {
	b.org.DOTNumber = dot
	return b
}

func (b *OrganizationBuilder) WithBucketName(bucket string) *OrganizationBuilder {
	b.org.BucketName = bucket
	return b
}

func (b *OrganizationBuilder) Build(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
) *tenant.Organization {
	t.Helper()

	err := tenantdb.Organizations.Create(systemContext(ctx), db, b.org)
	require.NoError(t, err, "failed to insert test organization")

	return b.org
}

type UserBuilder struct {
	user *tenant.User
}

func NewUser(orgID pulid.ID, buID pulid.ID) *UserBuilder {
	now := timeutils.NowUnix()
	return &UserBuilder{
		user: &tenant.User{
			CurrentOrganizationID: orgID,
			BusinessUnitID:        buID,
			Name:                  "Test User",
			Username:              fmt.Sprintf("testuser_%d", now),
			EmailAddress:          fmt.Sprintf("test_%d@example.com", now),
			Status:                domaintypes.StatusActive,
			Timezone:              "America/Los_Angeles",
			CreatedAt:             now,
			UpdatedAt:             now,
		},
	}
}

func (b *UserBuilder) WithName(name string) *UserBuilder {
	b.user.Name = name
	return b
}

func (b *UserBuilder) WithUsername(username string) *UserBuilder {
	b.user.Username = username
	return b
}

func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.user.EmailAddress = email
	return b
}

func (b *UserBuilder) WithPassword(password string) *UserBuilder {
	hashedPassword, err := b.user.GeneratePassword(password)
	if err != nil {
		panic(fmt.Sprintf("failed to hash password: %v", err))
	}
	b.user.Password = hashedPassword
	return b
}

func (b *UserBuilder) Build(t *testing.T, ctx context.Context, db *keel.DB) *tenant.User {
	t.Helper()

	err := tenantdb.Users.Create(systemContext(ctx), db, b.user)
	require.NoError(t, err, "failed to insert test user")

	return b.user
}

type StateBuilder struct {
	state *usstate.UsState
}

func NewState() *StateBuilder {
	return &StateBuilder{
		state: &usstate.UsState{
			Name:         "California",
			Abbreviation: "CA",
			CountryName:  "United States",
			CountryIso3:  "USA",
		},
	}
}

func (b *StateBuilder) WithName(name string) *StateBuilder {
	b.state.Name = name
	return b
}

func (b *StateBuilder) WithAbbreviation(abbr string) *StateBuilder {
	b.state.Abbreviation = abbr
	return b
}

func (b *StateBuilder) Build(t *testing.T, ctx context.Context, db *keel.DB) *usstate.UsState {
	t.Helper()

	s := usstatedb.UsStates
	_, err := s.Upsert(ctx, db, []*usstate.UsState{b.state}, keel.OnConflict(s.Abbreviation).Update(s.Name))
	require.NoError(t, err, "failed to insert test state")

	state, err := s.Where(s.Abbreviation.Eq(b.state.Abbreviation)).One(ctx, db)
	require.NoError(t, err, "failed to read test state")

	return state
}

type TestData struct {
	BusinessUnit *tenant.BusinessUnit
	Organization *tenant.Organization
	User         *tenant.User
	State        *usstate.UsState
}

func (d *TestData) Info() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  d.Organization.ID,
		BuID:   d.BusinessUnit.ID,
		UserID: d.User.ID,
	}
}

func (d *TestData) Ctx(ctx context.Context) context.Context {
	return dbscope.WithTenant(ctx, d.Info().DBTenant())
}

func SeedFullTestData(t *testing.T, ctx context.Context, db *keel.DB) *TestData {
	t.Helper()

	state := NewState().Build(t, ctx, db)
	bu := NewBusinessUnit().Build(t, ctx, db)
	org := NewOrganization(bu.ID, state.ID).Build(t, ctx, db)
	user := NewUser(org.ID, bu.ID).WithPassword("password123").Build(t, ctx, db)

	return &TestData{
		BusinessUnit: bu,
		Organization: org,
		User:         user,
		State:        state,
	}
}

func SeedAdditionalTenant(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	suffix string,
) *TestData {
	t.Helper()

	state := NewState().
		WithName("Test State "+suffix).
		WithAbbreviation(suffix).
		Build(t, ctx, db)
	bu := NewBusinessUnit().
		WithName("Test Business Unit "+suffix).
		WithCode(suffix).
		Build(t, ctx, db)
	org := NewOrganization(bu.ID, state.ID).
		WithName("Test Organization "+suffix).
		WithScacCode(suffix).
		WithDOTNumber("999"+suffix).
		WithBucketName("test-bucket-"+strings.ToLower(suffix)).
		Build(t, ctx, db)
	user := NewUser(org.ID, bu.ID).
		WithUsername("testuser"+strings.ToLower(suffix)).
		WithEmail("test-"+strings.ToLower(suffix)+"@example.com").
		WithPassword("password123").
		Build(t, ctx, db)

	return &TestData{
		BusinessUnit: bu,
		Organization: org,
		User:         user,
		State:        state,
	}
}

func systemContext(ctx context.Context) context.Context {
	return dbscope.WithSystem(ctx, "seedtest")
}

func ShipmentsWithoutReceivables() keel.Pred {
	s := shipmentdb.Shipments
	inv := invoicedb.Invoices
	cle := customerledgerdb.CustomerLedgerEntries

	return keel.And(
		keel.Not(keel.Exists(inv.Where(inv.CustomerID.EqCol(s.CustomerID)))),
		keel.Not(keel.Exists(cle.Where(cle.CustomerID.EqCol(s.CustomerID)))),
	)
}
