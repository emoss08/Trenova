package agentapprovalports

import (
	"testing"

	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestCommitApprovalRequest_ActorKeepsTheSessionThatApproved(t *testing.T) {
	t.Parallel()

	person := pulid.MustNew("usr_")
	session := &authctx.SessionAttributes{UserID: person, AuthenticatorAAL: 2}
	req := &CommitApprovalRequest{
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		DecidedByUserID: person,
		Session:         session,
	}

	actor := req.Actor()

	assert.Equal(t, person, actor.UserID)
	assert.Equal(t, req.OrganizationID, actor.OrganizationID)
	assert.Same(t, session, actor.Session)
}
