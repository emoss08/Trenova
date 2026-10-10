package documentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanReview_AnAgentCarryingTheSystemUserIsNotAReviewer(t *testing.T) {
	t.Parallel()

	_, err := (&Service{}).PlanReview(t.Context(), &ReviewRequest{
		DocumentID: pulid.MustNew("doc_"),
		Decision:   document.ReviewDecisionApprove,
		Actor: permissionports.RequestActor{
			PrincipalType: permissionports.PrincipalTypeAgent,
			PrincipalID:   pulid.MustNew("agdef_"),
			UserID:        pulid.MustNew("usr_"),
		},
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err), "only a person reviews a document")
}
