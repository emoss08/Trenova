package authservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type supportGrants struct {
	err      error
	requests []supportaccessports.RequireLiveRequest
}

func (s *supportGrants) Permissions(
	context.Context,
	pulid.ID,
	pulid.ID,
) (*permissionports.CachedPermissions, error) {
	return &permissionports.CachedPermissions{}, nil
}

func (s *supportGrants) RequireLive(_ context.Context, req *supportaccessports.RequireLiveRequest) error {
	s.requests = append(s.requests, *req)
	return s.err
}

func TestAuthenticateSession_ChecksASupportSessionsGrantOnEveryRequest(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	grants := &supportGrants{}
	deps.svc.support = grants

	principal := pulid.MustNew(supportaccess.PrincipalIDPrefix)
	sessionID := pulid.MustNew("ses_")
	issued, err := testTokens.Issue(sessionID, principal)
	require.NoError(t, err)
	stored := storedSession(sessionID, issued.SecretHash)
	stored.UserID = principal
	stored.Support = &authctx.Support{GrantID: pulid.MustNew("sag_"), StaffEmail: "jane@trenova.app"}
	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(stored, nil)

	for range 3 {
		sess, authErr := deps.svc.AuthenticateSession(t.Context(), issued.Token)
		require.NoError(t, authErr)
		assert.False(t, sess.RoleActivationRequired, "a support session's reach comes from its grant, never roles")
	}
	require.Len(t, grants.requests, 3)
	assert.Equal(t, stored.Support.GrantID, grants.requests[0].GrantID)
	assert.Equal(t, principal, grants.requests[0].PrincipalUserID)
}

func TestAuthenticateSession_EndsASupportSessionWhoseGrantEnded(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.svc.support = &supportGrants{err: supportaccess.ErrGrantNotFound}

	principal := pulid.MustNew(supportaccess.PrincipalIDPrefix)
	sessionID := pulid.MustNew("ses_")
	issued, err := testTokens.Issue(sessionID, principal)
	require.NoError(t, err)
	stored := storedSession(sessionID, issued.SecretHash)
	stored.UserID = principal
	stored.Support = &authctx.Support{GrantID: pulid.MustNew("sag_")}
	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(stored, nil)
	deps.sessionRepo.On("DeleteAllForUser", mock.Anything, principal).Return(nil)

	_, err = deps.svc.AuthenticateSession(t.Context(), issued.Token)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	deps.sessionRepo.AssertCalled(t, "DeleteAllForUser", mock.Anything, principal)
}

func TestAuthenticateSession_KeepsASupportSessionWhoseGrantCannotBeRead(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.svc.support = &supportGrants{err: errors.New("database unavailable")}

	principal := pulid.MustNew(supportaccess.PrincipalIDPrefix)
	sessionID := pulid.MustNew("ses_")
	issued, err := testTokens.Issue(sessionID, principal)
	require.NoError(t, err)
	stored := storedSession(sessionID, issued.SecretHash)
	stored.UserID = principal
	stored.Support = &authctx.Support{GrantID: pulid.MustNew("sag_")}
	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(stored, nil)

	_, err = deps.svc.AuthenticateSession(t.Context(), issued.Token)

	require.ErrorIs(t, err, sessionports.ErrSessionAccessUnconfirmed, "refused, but the cookie is kept")
	deps.sessionRepo.AssertNotCalled(t, "DeleteAllForUser", mock.Anything, mock.Anything)
}

func TestCreateSession_NeverSignsInASupportPrincipal(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	usr := newTestUser(t)
	usr.ID = pulid.MustNew(supportaccess.PrincipalIDPrefix)

	_, _, err := deps.svc.createSession(t.Context(), usr, loginSessionContext{AuthenticatorAAL: 1})

	require.Error(t, err)
	assert.Equal(t, errInvalidCredentials, err)
}
