package planfields_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/app/planfields"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const kind = permission.Resource("customer")

type billing struct {
	CreditLimit int    `json:"creditLimit"`
	Terms       string `json:"terms"`
}

type record struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	TenderStatus *string  `json:"tenderStatus"`
	Billing      *billing `json:"billingProfile"`
	Notes        []string `json:"notes"`
}

func registry(t *testing.T) *permissioncatalog.Registry {
	t.Helper()

	reg := permissioncatalog.NewEmptyRegistry()
	require.NoError(t, reg.Register(&permissionspec.ResourceDefinition{
		Resource: kind.String(),
		Feature:  platformcatalog.FeatureCoreTMS,
		FieldFeatures: map[string]platformcatalog.FeatureKey{
			"tenderStatus": platformcatalog.FeatureCarrierBrokerage,
			"creditLimit":  platformcatalog.FeatureAccounting,
		},
	}))

	return reg
}

func keeper(t *testing.T, gate platformcatalogports.FeatureGate) *planfields.Keeper {
	t.Helper()

	reg := registry(t)
	return planfields.New(reg, onplan.New(reg, gate))
}

func stored() *record {
	return &record{
		ID:           "cus_1",
		Name:         "Acme",
		TenderStatus: new("Accepted"),
		Billing:      &billing{CreditLimit: 5000, Terms: "Net30"},
	}
}

func sent() *record {
	return &record{
		ID:      "cus_1",
		Name:    "Acme Freight",
		Billing: &billing{CreditLimit: 0, Terms: "Net15"},
	}
}

func TestKeep_UpdateKeepsWhatThePlanWithholdsAndTakesTheRest(t *testing.T) {
	t.Parallel()

	incoming := sent()
	err := keeper(t, plantest.BaseOnly()).Keep(t.Context(), kind, incoming, stored())

	require.NoError(t, err)
	require.NotNil(t, incoming.TenderStatus)
	assert.Equal(t, "Accepted", *incoming.TenderStatus, "Carrier & Brokerage is off")
	assert.Equal(t, 5000, incoming.Billing.CreditLimit,
		"a plain struct's field is kept by its own name")
	assert.Equal(t, "Acme Freight", incoming.Name, "a base field takes the sent value")
	assert.Equal(t, "Net15", incoming.Billing.Terms, "a base field of a plain struct as well")
}

func TestKeep_KeepsOnlyTheFeaturesThatAreOff(t *testing.T) {
	t.Parallel()

	incoming := sent()
	incoming.TenderStatus = new("Tendered")
	err := keeper(t, plantest.Without(platformcatalog.FeatureAccounting)).
		Keep(t.Context(), kind, incoming, stored())

	require.NoError(t, err)
	assert.Equal(t, "Tendered", *incoming.TenderStatus, "Carrier & Brokerage is on")
	assert.Equal(t, 5000, incoming.Billing.CreditLimit, "Accounting & Cash is off")
}

func TestKeep_CreateZeroesWhatThePlanWithholds(t *testing.T) {
	t.Parallel()

	incoming := sent()
	incoming.TenderStatus = new("Tendered")
	incoming.Billing.CreditLimit = 900
	err := keeper(t, plantest.BaseOnly()).Keep(t.Context(), kind, incoming, nil)

	require.NoError(t, err)
	assert.Nil(t, incoming.TenderStatus)
	assert.Zero(t, incoming.Billing.CreditLimit)
	assert.Equal(t, "Net15", incoming.Billing.Terms)
}

func TestKeep_CreateTakesTheDefaultsGiven(t *testing.T) {
	t.Parallel()

	incoming := sent()
	incoming.Billing.CreditLimit = 900
	defaults := &record{Billing: &billing{CreditLimit: 100}}
	err := keeper(t, plantest.BaseOnly()).Keep(t.Context(), kind, incoming, defaults)

	require.NoError(t, err)
	assert.Equal(t, 100, incoming.Billing.CreditLimit)
}

func TestKeep_ATypedNilStoredRecordIsACreate(t *testing.T) {
	t.Parallel()

	incoming := sent()
	incoming.Billing.CreditLimit = 900
	var none *record
	err := keeper(t, plantest.BaseOnly()).Keep(t.Context(), kind, incoming, none)

	require.NoError(t, err)
	assert.Zero(t, incoming.Billing.CreditLimit)
}

func TestKeep_LeavesANestedStructThatWasNotSent(t *testing.T) {
	t.Parallel()

	incoming := sent()
	incoming.Billing = nil
	err := keeper(t, plantest.BaseOnly()).Keep(t.Context(), kind, incoming, stored())

	require.NoError(t, err)
	assert.Nil(t, incoming.Billing, "nothing is written under a struct the write leaves alone")
}

func TestKeep_ZeroesANestedFieldTheStoredRecordDoesNotHold(t *testing.T) {
	t.Parallel()

	incoming := sent()
	incoming.Billing.CreditLimit = 900
	before := stored()
	before.Billing = nil
	err := keeper(t, plantest.BaseOnly()).Keep(t.Context(), kind, incoming, before)

	require.NoError(t, err)
	assert.Zero(t, incoming.Billing.CreditLimit)
}

func TestKeep_ChangesNothingWithEveryFeatureOn(t *testing.T) {
	t.Parallel()

	incoming := sent()
	incoming.TenderStatus = new("Tendered")
	want := *incoming
	wantBilling := *incoming.Billing
	err := keeper(t, plantest.AllFeatures()).Keep(t.Context(), kind, incoming, stored())

	require.NoError(t, err)
	assert.Equal(t, want.Name, incoming.Name)
	assert.Equal(t, "Tendered", *incoming.TenderStatus)
	assert.Equal(t, wantBilling, *incoming.Billing)
}

type failingGate struct{ err error }

func (g failingGate) Enabled(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, g.err
}

func (g failingGate) EnabledAnywhere(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	return g.Enabled(ctx, feature)
}

func TestKeep_FailsWhenThePlanCannotBeRead(t *testing.T) {
	t.Parallel()

	unread := errors.New("plan store unavailable")
	incoming := sent()
	err := keeper(t, failingGate{err: unread}).Keep(t.Context(), kind, incoming, stored())

	require.ErrorIs(t, err, unread)
	assert.Nil(t, incoming.TenderStatus, "a failed read writes nothing")
}

func TestKeep_RefusesAStoredRecordOfAnotherType(t *testing.T) {
	t.Parallel()

	err := keeper(t, plantest.BaseOnly()).Keep(t.Context(), kind, sent(), &billing{})

	require.Error(t, err)
}

func TestKeep_RefusesARecordThatIsNotAPointerToAStruct(t *testing.T) {
	t.Parallel()

	err := keeper(t, plantest.BaseOnly()).Keep(t.Context(), kind, *sent(), stored())

	require.Error(t, err)
}

func TestKeep_IgnoresAKindWithNoFieldFeatures(t *testing.T) {
	t.Parallel()

	incoming := sent()
	err := keeper(t, failingGate{err: errors.New("never asked")}).
		Keep(t.Context(), permission.Resource("unknown"), incoming, stored())

	require.NoError(t, err, "a kind whose fields belong to no other feature asks nothing")
	assert.Nil(t, incoming.TenderStatus)
}
