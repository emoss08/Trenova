package scimservice

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/scimproto"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testPrefix = "scim_AbCdEfGhIjKl"
	testToken  = testPrefix + ".the-secret-half-of-the-token"
)

func issuedCredential(h *harness) {
	h.repo.prefix = testPrefix
	h.repo.credential = &scimports.Credential{
		TokenID:          h.dir.TokenID,
		DirectoryID:      h.dir.ID,
		OrganizationID:   h.dir.OrganizationID,
		BusinessUnitID:   h.dir.BusinessUnitID,
		TokenHash:        scim.HashToken(testToken),
		Status:           iam.SCIMTokenStatusActive,
		DirectoryEnabled: true,
		TenantSlug:       h.dir.TenantSlug,
	}
}

func TestAuthenticate_BindsTheTokenToItsDirectoryAndTenant(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	issuedCredential(h)
	h.svc.plans = plantest.Allowing(t, platformplan.CapabilitySSO)

	dir, err := h.svc.Authenticate(t.Context(), scimports.AuthenticateRequest{
		Token:      testToken,
		TenantSlug: "ACME",
	})

	require.NoError(t, err)
	assert.Equal(t, h.dir.ID, dir.ID)
	assert.Equal(t, h.dir.OrganizationID, dir.OrganizationID)
	assert.Equal(t, h.dir.BusinessUnitID, dir.BusinessUnitID)
	assert.Equal(t, h.dir.TokenID, dir.TokenID)
	assert.Equal(t, []pulid.ID{h.dir.TokenID}, h.repo.touched)
}

func TestAuthenticate_RefusesTokensThatDoNotAdmitTheRequest(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		token  string
		slug   string
		change func(*scimports.Credential)
	}{
		"no token":               {token: "", slug: "acme"},
		"not a SCIM token":       {token: "trv_abc.def", slug: "acme"},
		"no secret half":         {token: testPrefix, slug: "acme"},
		"unknown prefix":         {token: "scim_ZZZZZZZZZZZZ.secret", slug: "acme"},
		"wrong secret":           {token: testPrefix + ".another-secret", slug: "acme"},
		"another directory slug": {token: testToken, slug: "globex"},
		"revoked token":          {token: testToken, slug: "acme", change: revoke},
		"expired token":          {token: testToken, slug: "acme", change: expire},
		"disabled directory":     {token: testToken, slug: "acme", change: disable},
		"oversized token": {
			token: testPrefix + "." + string(make([]byte, 300)),
			slug:  "acme",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			issuedCredential(h)
			if tc.change != nil {
				tc.change(h.repo.credential)
			}
			h.svc.plans = platformplanportsmock.NewMockPlanService(t)

			dir, err := h.svc.Authenticate(t.Context(), scimports.AuthenticateRequest{
				Token:      tc.token,
				TenantSlug: tc.slug,
			})

			require.Error(t, err)
			assert.Nil(t, dir)
			scimErr, ok := scimproto.AsError(err)
			require.True(t, ok)
			assert.Equal(t, http.StatusUnauthorized, scimErr.Status)
			assert.Empty(t, h.repo.touched)
		})
	}
}

func revoke(credential *scimports.Credential) {
	credential.Status = iam.SCIMTokenStatusRevoked
}

func expire(credential *scimports.Credential) {
	credential.ExpiresAt = timeutils.NowUnix() - 1
}

func disable(credential *scimports.Credential) {
	credential.DirectoryEnabled = false
}

func TestAuthenticate_RefusesWhenThePlanDoesNotIncludeSSO(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	issuedCredential(h)
	h.svc.plans = plantest.Restricting(t, platformplan.CapabilitySSO)

	_, err := h.svc.Authenticate(t.Context(), scimports.AuthenticateRequest{
		Token:      testToken,
		TenantSlug: "acme",
	})

	scimErr, ok := scimproto.AsError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusForbidden, scimErr.Status)
}
