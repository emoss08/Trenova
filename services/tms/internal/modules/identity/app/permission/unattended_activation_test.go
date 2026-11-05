package permission

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubAccessSettings struct {
	required bool
	err      error
}

func (s *stubAccessSettings) Get(
	context.Context,
	pagination.TenantInfo,
) (*iam.AccessSettings, error) {
	return &iam.AccessSettings{RequireRoleActivation: s.required}, s.err
}

func (s *stubAccessSettings) Save(
	_ context.Context,
	settings *iam.AccessSettings,
) (*iam.AccessSettings, error) {
	return settings, nil
}

func TestWithoutActivatedRoles(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	unattended := func(ctx context.Context) context.Context {
		return authctx.WithSessionAttributes(ctx, authctx.SessionAttributes{
			UserID:           userID,
			AuthenticatorAAL: authctx.UnprovenAssurance,
		})
	}
	activated := func(e *engine, ctx context.Context) (authctx.SessionRoleActivation, bool) {
		return authctx.GetSessionRoleActivation(e.withoutActivatedRoles(
			ctx, permissionports.PrincipalTypeUser, userID, orgID, buID,
		))
	}

	t.Run("an organization that requires activation leaves a queued turn no roles", func(t *testing.T) {
		t.Parallel()
		e := &engine{settings: &stubAccessSettings{required: true}, l: zap.NewNop()}

		activation, bound := activated(e, unattended(t.Context()))
		require.True(t, bound)
		assert.True(t, activation.RequiresActivation)
		assert.Empty(t, activation.ActiveRoleIDs)
	})

	t.Run("an organization that does not require it changes nothing", func(t *testing.T) {
		t.Parallel()
		e := &engine{settings: &stubAccessSettings{}, l: zap.NewNop()}

		_, bound := activated(e, unattended(t.Context()))
		assert.False(t, bound)
	})

	t.Run("a setting that cannot be read requires activation", func(t *testing.T) {
		t.Parallel()
		e := &engine{settings: &stubAccessSettings{err: errors.New("down")}, l: zap.NewNop()}

		activation, bound := activated(e, unattended(t.Context()))
		require.True(t, bound)
		assert.True(t, activation.RequiresActivation)
	})

	t.Run("a person's own session keeps what it activated", func(t *testing.T) {
		t.Parallel()
		e := &engine{settings: &stubAccessSettings{required: true}, l: zap.NewNop()}
		ctx := authctx.WithSessionAttributes(t.Context(), authctx.SessionAttributes{
			UserID:           userID,
			AuthenticatorAAL: 2,
		})

		_, bound := activated(e, ctx)
		assert.False(t, bound)
	})
}
