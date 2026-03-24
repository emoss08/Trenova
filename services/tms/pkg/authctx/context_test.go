package authctx_test

import (
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c
}

func TestSetAndGetUserID(t *testing.T) {
	t.Parallel()

	t.Run("set and get user ID", func(t *testing.T) {
		t.Parallel()
		c := newTestContext()
		userID := pulid.MustNew("usr_")

		authctx.SetUserID(c, userID)
		got, exists := authctx.GetUserID(c)

		assert.True(t, exists)
		assert.Equal(t, userID, got)
	})

	t.Run("get user ID when not set", func(t *testing.T) {
		t.Parallel()
		c := newTestContext()

		got, exists := authctx.GetUserID(c)

		assert.False(t, exists)
		assert.True(t, got.IsNil())
	})

	t.Run("get user ID with wrong type in context", func(t *testing.T) {
		t.Parallel()
		c := newTestContext()
		c.Set(string(authctx.UserIDKey), "not-a-pulid")

		got, exists := authctx.GetUserID(c)

		assert.False(t, exists)
		assert.True(t, got.IsNil())
	})
}

func TestSetAndGetBusinessUnitID(t *testing.T) {
	t.Parallel()

	t.Run("set and get business unit ID", func(t *testing.T) {
		t.Parallel()
		c := newTestContext()
		buID := pulid.MustNew("bu_")

		authctx.SetBusinessUnitID(c, buID)
		got, exists := authctx.GetBusinessUnitID(c)

		assert.True(t, exists)
		assert.Equal(t, buID, got)
	})

	t.Run("get business unit ID when not set", func(t *testing.T) {
		t.Parallel()
		c := newTestContext()

		got, exists := authctx.GetBusinessUnitID(c)

		assert.False(t, exists)
		assert.True(t, got.IsNil())
	})
}

func TestSetAndGetOrganizationID(t *testing.T) {
	t.Parallel()

	t.Run("set and get organization ID", func(t *testing.T) {
		t.Parallel()
		c := newTestContext()
		orgID := pulid.MustNew("org_")

		authctx.SetOrganizationID(c, orgID)
		got, exists := authctx.GetOrganizationID(c)

		assert.True(t, exists)
		assert.Equal(t, orgID, got)
	})

	t.Run("get organization ID when not set", func(t *testing.T) {
		t.Parallel()
		c := newTestContext()

		got, exists := authctx.GetOrganizationID(c)

		assert.False(t, exists)
		assert.True(t, got.IsNil())
	})
}

func TestSetAuthContext(t *testing.T) {
	t.Parallel()

	t.Run("sets all three IDs at once", func(t *testing.T) {
		t.Parallel()
		c := newTestContext()
		userID := pulid.MustNew("usr_")
		buID := pulid.MustNew("bu_")
		orgID := pulid.MustNew("org_")

		authctx.SetAuthContext(c, userID, buID, orgID)

		gotUser, userExists := authctx.GetUserID(c)
		gotBu, buExists := authctx.GetBusinessUnitID(c)
		gotOrg, orgExists := authctx.GetOrganizationID(c)

		assert.True(t, userExists)
		assert.True(t, buExists)
		assert.True(t, orgExists)
		assert.Equal(t, userID, gotUser)
		assert.Equal(t, buID, gotBu)
		assert.Equal(t, orgID, gotOrg)
	})
}

func TestGetAuthContext(t *testing.T) {
	t.Parallel()

	t.Run("returns populated auth context", func(t *testing.T) {
		t.Parallel()
		c := newTestContext()
		userID := pulid.MustNew("usr_")
		buID := pulid.MustNew("bu_")
		orgID := pulid.MustNew("org_")

		authctx.SetAuthContext(c, userID, buID, orgID)
		ac := authctx.GetAuthContext(c)

		require.NotNil(t, ac)
		assert.Equal(t, userID, ac.UserID)
		assert.Equal(t, buID, ac.BusinessUnitID)
		assert.Equal(t, orgID, ac.OrganizationID)
	})

	t.Run("returns empty auth context when nothing set", func(t *testing.T) {
		t.Parallel()
		c := newTestContext()

		ac := authctx.GetAuthContext(c)

		require.NotNil(t, ac)
		assert.True(t, ac.UserID.IsNil())
		assert.True(t, ac.BusinessUnitID.IsNil())
		assert.True(t, ac.OrganizationID.IsNil())
	})

	t.Run("returns partial auth context", func(t *testing.T) {
		t.Parallel()
		c := newTestContext()
		orgID := pulid.MustNew("org_")
		authctx.SetOrganizationID(c, orgID)

		ac := authctx.GetAuthContext(c)

		require.NotNil(t, ac)
		assert.True(t, ac.UserID.IsNil())
		assert.True(t, ac.BusinessUnitID.IsNil())
		assert.Equal(t, orgID, ac.OrganizationID)
	})
}

func TestAddContextToRequest(t *testing.T) {
	t.Parallel()

	type EntityWithOrgAndBu struct {
		OrganizationID pulid.ID
		BusinessUnitID pulid.ID
		UserID         pulid.ID
	}

	type EntityWithAltFields struct {
		OrgID  pulid.ID
		BuID   pulid.ID
		UserID pulid.ID
	}

	type EntityWithNoMatchingFields struct {
		Name string
	}

	t.Run("sets OrganizationID, BusinessUnitID, UserID fields", func(t *testing.T) {
		t.Parallel()
		userID := pulid.MustNew("usr_")
		buID := pulid.MustNew("bu_")
		orgID := pulid.MustNew("org_")
		ac := &authctx.AuthContext{
			UserID:         userID,
			BusinessUnitID: buID,
			OrganizationID: orgID,
		}

		entity := &EntityWithOrgAndBu{}
		authctx.AddContextToRequest(ac, entity)

		assert.Equal(t, orgID, entity.OrganizationID)
		assert.Equal(t, buID, entity.BusinessUnitID)
		assert.Equal(t, userID, entity.UserID)
	})

	t.Run("sets OrgID and BuID alternate field names", func(t *testing.T) {
		t.Parallel()
		userID := pulid.MustNew("usr_")
		buID := pulid.MustNew("bu_")
		orgID := pulid.MustNew("org_")
		ac := &authctx.AuthContext{
			UserID:         userID,
			BusinessUnitID: buID,
			OrganizationID: orgID,
		}

		entity := &EntityWithAltFields{}
		authctx.AddContextToRequest(ac, entity)

		assert.Equal(t, orgID, entity.OrgID)
		assert.Equal(t, buID, entity.BuID)
		assert.Equal(t, userID, entity.UserID)
	})

	t.Run("does nothing for non-pointer value", func(t *testing.T) {
		t.Parallel()
		ac := &authctx.AuthContext{
			UserID:         pulid.MustNew("usr_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			OrganizationID: pulid.MustNew("org_"),
		}

		entity := EntityWithOrgAndBu{}
		authctx.AddContextToRequest(ac, entity)

		assert.True(t, entity.OrganizationID.IsNil())
		assert.True(t, entity.BusinessUnitID.IsNil())
		assert.True(t, entity.UserID.IsNil())
	})

	t.Run("does nothing for struct with no matching fields", func(t *testing.T) {
		t.Parallel()
		ac := &authctx.AuthContext{
			UserID:         pulid.MustNew("usr_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			OrganizationID: pulid.MustNew("org_"),
		}

		entity := &EntityWithNoMatchingFields{Name: "test"}
		authctx.AddContextToRequest(ac, entity)

		assert.Equal(t, "test", entity.Name)
	})

	t.Run("does nothing for nil pointer", func(t *testing.T) {
		t.Parallel()
		ac := &authctx.AuthContext{
			UserID:         pulid.MustNew("usr_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			OrganizationID: pulid.MustNew("org_"),
		}

		var entity *EntityWithOrgAndBu
		authctx.AddContextToRequest(ac, entity)
	})
}
