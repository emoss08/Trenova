package seedtest

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
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
	tx bun.Tx,
) *tenant.BusinessUnit {
	t.Helper()

	_, err := tx.NewInsert().Model(b.bu).Exec(ctx)
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
			BusinessUnitID: buID,
			Name:           "Test Organization",
			ScacCode:       "TEST",
			DOTNumber:      "1234567",
			AddressLine1:   "123 Test St",
			City:           "Test City",
			StateID:        stateID,
			PostalCode:     "12345",
			Timezone:       "America/Los_Angeles",
			BucketName:     "test-bucket",
			CreatedAt:      now,
			UpdatedAt:      now,
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
	tx bun.Tx,
) *tenant.Organization {
	t.Helper()

	_, err := tx.NewInsert().Model(b.org).Exec(ctx)
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

func (b *UserBuilder) AsPlatformAdmin() *UserBuilder {
	b.user.IsPlatformAdmin = true
	return b
}

func (b *UserBuilder) Build(t *testing.T, ctx context.Context, tx bun.Tx) *tenant.User {
	t.Helper()

	_, err := tx.NewInsert().Model(b.user).Exec(ctx)
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

func (b *StateBuilder) Build(t *testing.T, ctx context.Context, tx bun.Tx) *usstate.UsState {
	t.Helper()

	_, err := tx.NewInsert().Model(b.state).Exec(ctx)
	require.NoError(t, err, "failed to insert test state")

	return b.state
}

type TestData struct {
	BusinessUnit *tenant.BusinessUnit
	Organization *tenant.Organization
	User         *tenant.User
	State        *usstate.UsState
}

func CreateFullTestData(t *testing.T, ctx context.Context, tx bun.Tx) *TestData {
	t.Helper()

	state := NewState().Build(t, ctx, tx)
	bu := NewBusinessUnit().Build(t, ctx, tx)
	org := NewOrganization(bu.ID, state.ID).Build(t, ctx, tx)
	user := NewUser(org.ID, bu.ID).WithPassword("password123").Build(t, ctx, tx)

	return &TestData{
		BusinessUnit: bu,
		Organization: org,
		User:         user,
		State:        state,
	}
}

func SeedFullTestData(t *testing.T, ctx context.Context, db *bun.DB) *TestData {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err, "failed to begin transaction for test data")

	data := CreateFullTestData(t, ctx, tx)

	require.NoError(t, tx.Commit(), "failed to commit test data")
	return data
}
