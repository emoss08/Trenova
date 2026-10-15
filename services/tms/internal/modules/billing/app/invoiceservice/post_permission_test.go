package invoiceservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingEngine struct {
	mocks.DenyResourcesPermissionEngine

	asked []*permissionports.PermissionCheckRequest
}

func (e *recordingEngine) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	e.asked = append(e.asked, req)
	return e.DenyResourcesPermissionEngine.Check(ctx, req)
}

func poster() *permissionports.RequestActor {
	userID := pulid.MustNew("usr_")
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func denyingInvoices() *recordingEngine {
	return &recordingEngine{DenyResourcesPermissionEngine: mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceInvoice.String()},
	}}
}

func TestGuardPosterRefusesAPersonWithoutThePostPermission(t *testing.T) {
	t.Parallel()

	engine := denyingInvoices()
	svc := &Service{permissions: engine}
	actor := poster()

	err := svc.guardPoster(t.Context(), &invoiceports.PostInvoiceRequest{}, actor)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	require.Len(t, engine.asked, 1)
	assert.Equal(t, permission.ResourceInvoice.String(), engine.asked[0].Resource)
	assert.Equal(t, permission.OpPost, engine.asked[0].Operation)
	assert.Equal(t, actor.UserID, engine.asked[0].UserID)
}

func TestGuardPosterLetsAPersonWithThePostPermissionPost(t *testing.T) {
	t.Parallel()

	svc := &Service{permissions: &mocks.AllowAllPermissionEngine{}}

	require.NoError(t, svc.guardPoster(t.Context(), &invoiceports.PostInvoiceRequest{}, poster()))
}

func TestGuardPosterLeavesAutomaticPostingToTheOrganizationsSettings(t *testing.T) {
	t.Parallel()

	engine := denyingInvoices()
	svc := &Service{permissions: engine}

	err := svc.guardPoster(
		t.Context(),
		&invoiceports.PostInvoiceRequest{Automatic: true},
		poster(),
	)

	require.NoError(t, err)
	assert.Empty(t, engine.asked)
}

func TestGuardPosterLetsTheSystemPost(t *testing.T) {
	t.Parallel()

	engine := denyingInvoices()
	svc := &Service{permissions: engine}

	require.NoError(t, svc.guardPoster(t.Context(), &invoiceports.PostInvoiceRequest{},
		&permissionports.RequestActor{PrincipalType: permissionports.PrincipalTypeSystem}))
	require.NoError(t, svc.guardPoster(t.Context(), &invoiceports.PostInvoiceRequest{}, nil))
	assert.Empty(t, engine.asked)
}

func TestPostingNeedsOnlyReadBesidesItself(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []permission.Operation{permission.OpRead}, permission.Dependencies[permission.OpPost])
	assert.Contains(
		t,
		permission.NewRegistry().GetOperationsForResource(permission.ResourceInvoice.String()),
		permission.OpPost,
	)
}
