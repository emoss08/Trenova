package userhandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRespondWithSelf_ReportsTheSessionTenantNotTheLastOneChosen(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	authctx.SetSessionAuthContext(c, authctx.SessionAuthContextParams{
		UserID:         pulid.ID("usr_01JQ0000000000000000000001"),
		OrganizationID: pulid.ID("org_01JQ0000000000000000000001"),
		BusinessUnitID: pulid.ID("bu_01JQ0000000000000000000001"),
	})

	respondWithSelf(c, &tenant.User{
		ID:                    pulid.ID("usr_01JQ0000000000000000000001"),
		CurrentOrganizationID: pulid.ID("org_01JQ0000000000000000000002"),
		BusinessUnitID:        pulid.ID("bu_01JQ0000000000000000000002"),
	})

	require.Equal(t, http.StatusOK, recorder.Code)
	var body struct {
		CurrentOrganizationID string `json:"currentOrganizationId"`
		BusinessUnitID        string `json:"businessUnitId"`
	}
	require.NoError(t, sonic.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "org_01JQ0000000000000000000001", body.CurrentOrganizationID)
	assert.Equal(t, "bu_01JQ0000000000000000000001", body.BusinessUnitID)
}
