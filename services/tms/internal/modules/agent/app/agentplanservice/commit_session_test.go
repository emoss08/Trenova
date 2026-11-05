package agentplanservice

import (
	"testing"

	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecideOwn_HandsTheCommitTheSessionThatApproved(t *testing.T) {
	t.Parallel()

	f, window := gatedOwnFixture(t)
	f.svc.activity = &announcedPlans{}
	f.req.PreviewDigest = "plan-digest"
	f.actor.Session = &authctx.SessionAttributes{
		UserID:           f.actor.UserID,
		AuthenticatorAAL: 2,
	}

	_, err := f.svc.DecideOwn(t.Context(), f.req, f.actor)

	require.NoError(t, err)
	require.Len(t, window.started, 1)
	assert.Equal(t, f.actor.Session, window.started[0].Session)
}
