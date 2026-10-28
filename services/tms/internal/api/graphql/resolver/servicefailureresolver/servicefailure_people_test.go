package servicefailureresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type usersByID struct {
	tenantports.UserRepository

	users    map[pulid.ID]*tenant.User
	requests [][]pulid.ID
}

func (u *usersByID) GetByIDs(
	_ context.Context,
	req tenantports.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	u.requests = append(u.requests, req.UserIDs)
	found := make([]*tenant.User, 0, len(req.UserIDs))
	for _, id := range req.UserIDs {
		if user, ok := u.users[id]; ok {
			found = append(found, user)
		}
	}
	return found, nil
}

func withUserLoader(ctx context.Context, users *usersByID) context.Context {
	factory := loaders.NewUserByIDLoaderFactory(loaders.UserByIDLoaderFactoryParams{Repo: users})
	return factory.Attach(ctx, pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	})
}

func peopleResolver(granted map[string]bool) *Deps {
	return &Deps{Core: &resolvercore.Core{
		PermissionEngine: &resolvertest.GrantingPermissionEngine{Granted: granted},
	}}
}

func TestServiceFailurePeople_TakeTheLoadedUserWithoutAskingAgain(t *testing.T) {
	t.Parallel()

	reviewer := &tenant.User{ID: pulid.MustNew("usr_"), Name: "Omar Haddad"}
	users := &usersByID{}
	failure := &servicefailure.ServiceFailure{ReviewedByID: &reviewer.ID, ReviewedBy: reviewer}

	got, err := (&ServiceFailureResolver{peopleResolver(nil)}).ReviewedBy(
		withUserLoader(t.Context(), users),
		failure,
	)
	require.NoError(t, err)
	assert.Same(t, reviewer, got)
	assert.Empty(t, users.requests)
}

func TestServiceFailurePeople_ReadEachUnloadedPersonThroughTheLoader(t *testing.T) {
	t.Parallel()

	resolverID := pulid.MustNew("usr_")
	voiderID := pulid.MustNew("usr_")
	users := &usersByID{users: map[pulid.ID]*tenant.User{
		resolverID: {ID: resolverID, Name: "Mia Chen"},
		voiderID:   {ID: voiderID, Name: "Ray Gutierrez"},
	}}
	ctx := withUserLoader(t.Context(), users)
	failure := &servicefailure.ServiceFailure{ResolvedByID: &resolverID, VoidedByID: &voiderID}
	resolver := &ServiceFailureResolver{peopleResolver(nil)}

	type result struct {
		user *tenant.User
		err  error
	}
	resolved := make(chan result, 1)
	voided := make(chan result, 1)
	go func() {
		user, err := resolver.ResolvedBy(ctx, failure)
		resolved <- result{user, err}
	}()
	go func() {
		user, err := resolver.VoidedBy(ctx, failure)
		voided <- result{user, err}
	}()

	first, second := <-resolved, <-voided
	require.NoError(t, first.err)
	require.NoError(t, second.err)
	require.NotNil(t, first.user)
	require.NotNil(t, second.user)
	assert.Equal(t, "Mia Chen", first.user.Name)
	assert.Equal(t, "Ray Gutierrez", second.user.Name)
	asked := make([]pulid.ID, 0, 2)
	for _, ids := range users.requests {
		asked = append(asked, ids...)
	}
	assert.ElementsMatch(t, []pulid.ID{resolverID, voiderID}, asked)
}

func TestServiceFailurePeople_NoOneForAStepNotTaken(t *testing.T) {
	t.Parallel()

	users := &usersByID{}
	resolver := &ServiceFailureResolver{peopleResolver(nil)}
	ctx := withUserLoader(t.Context(), users)
	failure := &servicefailure.ServiceFailure{}

	for name, read := range map[string]func(context.Context, *servicefailure.ServiceFailure) (*tenant.User, error){
		"createdBy":  resolver.CreatedBy,
		"reviewedBy": resolver.ReviewedBy,
		"resolvedBy": resolver.ResolvedBy,
		"voidedBy":   resolver.VoidedBy,
	} {
		user, err := read(ctx, failure)
		require.NoError(t, err, name)
		assert.Nil(t, user, name)
	}
	assert.Empty(t, users.requests)
}

func TestServiceFailureShipmentCustomer_OnlyForSomeoneWhoMayReadCustomers(t *testing.T) {
	t.Parallel()

	acme := &customer.Customer{ID: pulid.MustNew("cus_"), Name: "Acme Foods"}
	load := &shipment.Shipment{CustomerID: acme.ID, Customer: acme}

	denied, err := (&ServiceFailureShipmentResolver{peopleResolver(nil)}).Customer(
		listContext(t),
		load,
	)
	require.NoError(t, err)
	assert.Nil(t, denied)

	granted, err := (&ServiceFailureShipmentResolver{peopleResolver(map[string]bool{
		permission.ResourceCustomer.String() + "|" + string(permission.OpRead): true,
	})}).Customer(listContext(t), load)
	require.NoError(t, err)
	assert.Same(t, acme, granted)
}

func TestServiceFailureShipmentCustomer_NoneForALoadWithoutOne(t *testing.T) {
	t.Parallel()

	got, err := (&ServiceFailureShipmentResolver{peopleResolver(map[string]bool{
		permission.ResourceCustomer.String() + "|" + string(permission.OpRead): true,
	})}).Customer(listContext(t), &shipment.Shipment{})
	require.NoError(t, err)
	assert.Nil(t, got)
}
