package middleware

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/stretchr/testify/assert"
)

func supportSession(scope supportaccess.Scope) *authctx.Support {
	return &authctx.Support{StaffEmail: "jane.doe@trenova.app", Scope: scope.String()}
}

func TestSupportWriteRefusal_AReadOnlySessionChangesNothing(t *testing.T) {
	t.Parallel()

	readOnly := supportSession(supportaccess.ScopeReadOnly)
	for _, route := range []string{
		"/api/v1/shipments/:shipmentID/",
		"/api/v1/users/me/settings",
		"/api/v1/favorites/",
		"/api/v1/assistant/threads/:threadID/case/",
	} {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			refusal := SupportWriteRefusal(readOnly, method, route)
			if assert.NotNil(t, refusal, "%s %s", method, route) {
				assert.Contains(t, refusal.Message, "read-only")
			}
		}
	}
}

func TestSupportWriteRefusal_AReadOnlySessionStillReadsPreviewsAndAsksTheAssistant(t *testing.T) {
	t.Parallel()

	readOnly := supportSession(supportaccess.ScopeReadOnly)
	assert.Nil(t, SupportWriteRefusal(readOnly, http.MethodGet, "/api/v1/shipments/"))
	assert.Nil(t, SupportWriteRefusal(readOnly, http.MethodPost, "/api/v1/billing/invoice-adjustments/preview"),
		"a preview writes nothing")
	assert.Nil(t, SupportWriteRefusal(readOnly, http.MethodPost, "/graphql"),
		"GraphQL mutations are refused by the operation guard, not here")
	assert.Nil(t, SupportWriteRefusal(readOnly, http.MethodPost, "/api/v1/assistant/threads/"),
		"the assistant decides by the grant's assistant permission")
	assert.Nil(t, SupportWriteRefusal(readOnly, http.MethodPost, "/api/v1/realtime/presence/:resource"),
		"presence shows the customer who is looking")
}

func TestSupportWriteRefusal_AFullSessionKeepsItsReachButNothingThatOutlivesTheGrant(t *testing.T) {
	t.Parallel()

	full := supportSession(supportaccess.ScopeFull)
	assert.Nil(t, SupportWriteRefusal(full, http.MethodPatch, "/api/v1/shipments/:shipmentID/"))
	assert.Nil(t, SupportWriteRefusal(full, http.MethodPut, "/api/v1/users/me/settings"),
		"the principal's own preferences go with the grant")
	assert.NotNil(t, SupportWriteRefusal(full, http.MethodPost, "/api/v1/assistant/threads/:threadID/schedules/"))
	assert.NotNil(t, SupportWriteRefusal(full, http.MethodPost, "/api/v1/users/me/change-password"))
	assert.NotNil(t, SupportWriteRefusal(full, http.MethodPost, "/api/v1/users/me/switch-organization"))
}

func TestSupportWriteRefusal_LeavesEveryoneElseAlone(t *testing.T) {
	t.Parallel()

	assert.Nil(t, SupportWriteRefusal(nil, http.MethodPost, "/api/v1/users/me/change-password"))
	assert.Nil(t, SupportWriteRefusal(nil, http.MethodDelete, "/api/v1/shipments/:shipmentID/"))
}
