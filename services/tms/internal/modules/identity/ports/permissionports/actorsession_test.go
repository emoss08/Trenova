package permissionports

import (
	"testing"

	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindActorSession_APersonsTurnKeepsTheSessionItStartedIn(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	ctx := BindActorSession(t.Context(), &RequestActor{
		PrincipalType: PrincipalTypeUser,
		UserID:        userID,
		Session: &authctx.SessionAttributes{
			UserID:             userID,
			AuthenticatorAAL:   2,
			MFAAuthenticatedAt: 1_700_000_000,
		},
	})

	session, ok := authctx.GetSessionAttributes(ctx)
	require.True(t, ok)
	assert.Equal(t, 2, session.AuthenticatorAAL)
}

func TestBindActorSession_APersonWithNoSessionProvesNothing(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	ctx := BindActorSession(t.Context(), &RequestActor{
		PrincipalType: PrincipalTypeUser,
		UserID:        userID,
	})

	session, ok := authctx.GetSessionAttributes(ctx)
	require.True(t, ok)
	assert.Equal(t, authctx.UnprovenAssurance, session.AuthenticatorAAL)
	assert.Zero(t, session.MFAAuthenticatedAt)
}

func TestBindActorSession_TheRequestsOwnSessionIsNotDowngraded(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	ctx := authctx.WithSessionAttributes(t.Context(), authctx.SessionAttributes{
		UserID:           userID,
		AuthenticatorAAL: 2,
	})

	bound := BindActorSession(ctx, &RequestActor{
		PrincipalType: PrincipalTypeUser,
		UserID:        userID,
	})

	session, _ := authctx.GetSessionAttributes(bound)
	assert.Equal(t, 2, session.AuthenticatorAAL)
}

func TestBindActorSession_AnotherPersonsSessionIsNotBorrowed(t *testing.T) {
	t.Parallel()

	ctx := authctx.WithSessionAttributes(t.Context(), authctx.SessionAttributes{
		UserID:           pulid.MustNew("usr_"),
		AuthenticatorAAL: 2,
	})
	other := pulid.MustNew("usr_")

	bound := BindActorSession(ctx, &RequestActor{
		PrincipalType: PrincipalTypeUser,
		UserID:        other,
	})

	session, _ := authctx.GetSessionAttributes(bound)
	assert.Equal(t, other, session.UserID)
	assert.Equal(t, authctx.UnprovenAssurance, session.AuthenticatorAAL)
}

func TestBindActorSession_APrincipalThatIsNotAPersonHasNoSession(t *testing.T) {
	t.Parallel()

	for _, actor := range []*RequestActor{
		nil,
		{PrincipalType: PrincipalTypeAgent},
		{PrincipalType: PrincipalTypeAPIKey},
	} {
		_, ok := authctx.GetSessionAttributes(BindActorSession(t.Context(), actor))
		assert.False(t, ok)
	}
}
