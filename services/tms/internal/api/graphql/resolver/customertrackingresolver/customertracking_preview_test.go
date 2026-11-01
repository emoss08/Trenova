package customertrackingresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentCustomerTrackingPreview_ChecksShipmentRead(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	_, err := (&QueryResolver{deps}).ShipmentCustomerTrackingPreview(
		ctx,
		pulid.MustNew("shp_").String(),
	)

	resolvertest.RequireRefused(t, err, engine, permission.ResourceShipment, permission.OpRead)
}

func TestCustomerTrackingViewStatus_IsNullWhenThePageHidesIt(t *testing.T) {
	t.Parallel()

	resolver := &CustomerTrackingViewResolver{}

	hidden, err := resolver.Status(t.Context(), &customertrackingports.PublicTrackingView{})
	require.NoError(t, err)
	assert.Nil(t, hidden)

	shown, err := resolver.Status(t.Context(), &customertrackingports.PublicTrackingView{
		Status: customertrackingports.PublicStatusDelayed,
	})
	require.NoError(t, err)
	require.NotNil(t, shown)
	assert.Equal(t, customertrackingports.PublicStatusDelayed, *shown)
}
