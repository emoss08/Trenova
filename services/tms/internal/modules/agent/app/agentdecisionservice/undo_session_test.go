package agentdecisionservice

import (
	"testing"

	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApproveDeferred_HandsTheCommitTheSessionThatApproved(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 1)
	h.service.previews = &stalePreviews{}
	h.actor.Session = &authctx.SessionAttributes{
		UserID:             h.actor.UserID,
		AuthenticatorAAL:   2,
		MFAAuthenticatedAt: 1_800_000_000,
	}

	h.approve(t)

	require.Len(t, h.workflows.payloads, 1)
	assert.Equal(t, h.actor.Session, h.workflows.payloads[0].Session)
}
