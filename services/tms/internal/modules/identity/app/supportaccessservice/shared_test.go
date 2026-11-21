package supportaccessservice

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tenantAInstance = "5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	tenantBInstance = "6e2b4d2f-9d2a-4b4a-8b54-6c1e8d5f3a21"
	tenantCInstance = "7f3c5e3a-ae3b-4c5b-9c65-7d2f9e6a4b32"
)

type sharedServer struct {
	*rig
	a, b, c pagination.TenantInfo
}

func newSharedServer(t *testing.T) *sharedServer {
	t.Helper()
	r := newSharedRig(t)
	require.NoError(t, r.svc.SetKeys(t.Context(), &supportaccess.KeySet{
		StaffDomain: testDomain,
		Keys: []supportaccess.SigningKey{{
			KeyID: testKeyID, KeyType: "OKP", Curve: "Ed25519", Algorithm: "EdDSA", Use: "sig",
			X: base64.RawURLEncoding.EncodeToString(r.public),
		}},
	}))
	s := &sharedServer{
		rig: r,
		a:   r.organization(newOrgID(), pulid.MustNew("bu_")),
		b:   r.organization(newOrgID(), pulid.MustNew("bu_")),
		c:   r.organization(newOrgID(), pulid.MustNew("bu_")),
	}
	r.tenants.held[tenantAInstance] = supportaccessports.SupportTenant{
		OrganizationID: s.a.OrgID, BusinessUnitID: s.a.BuID, Active: true,
	}
	r.tenants.held[tenantBInstance] = supportaccessports.SupportTenant{
		OrganizationID: s.b.OrgID, BusinessUnitID: s.b.BuID, Active: true,
	}
	r.tenants.held[tenantCInstance] = supportaccessports.SupportTenant{
		OrganizationID: s.c.OrgID, BusinessUnitID: s.c.BuID, Active: false,
	}
	return s
}

func TestSharedSupport_StaffEnterATenantUnderItsOwnGrant(t *testing.T) {
	t.Parallel()
	s := newSharedServer(t)
	grant := grantFor(t, s.rig, s.a, supportaccess.ScopeReadOnly, 1)

	result, err := s.svc.Enter(t.Context(), &EnterRequest{
		Assertion: s.assertion(t, assertionOptions{org: s.a.OrgID, audience: tenantAInstance}),
	})
	require.NoError(t, err)
	assert.Equal(t, grant.ID, result.Grant.ID)
	sessions := s.sessions.live()
	require.Len(t, sessions, 1)
	assert.Equal(
		t,
		s.a.OrgID,
		sessions[0].OrganizationID,
		"the session is in that organization alone",
	)
	assert.Equal(t, s.a.BuID, sessions[0].BusinessUnitID)
	assert.Equal(t, grant.PrincipalUserID, sessions[0].UserID)
	require.NotNil(t, sessions[0].Support)
	assert.Equal(t, string(supportaccess.ScopeReadOnly), sessions[0].Support.Scope)
}

func TestSharedSupport_AGrantOfOneCompanyNeverOpensAnother(t *testing.T) {
	t.Parallel()
	s := newSharedServer(t)
	grantFor(t, s.rig, s.a, supportaccess.ScopeFull, 1)
	grantFor(t, s.rig, s.c, supportaccess.ScopeFull, 1)

	cases := []struct {
		name     string
		audience string
		org      pulid.ID
		want     supportaccess.RefusalReason
		audited  pulid.ID
	}{
		{"A's instance naming B's organization", tenantAInstance, s.b.OrgID,
			supportaccess.RefusalOrganization, s.b.OrgID},
		{"B's instance naming A's organization", tenantBInstance, s.a.OrgID,
			supportaccess.RefusalOrganization, s.a.OrgID},
		{"B's instance and organization with a grant only in A", tenantBInstance, s.b.OrgID,
			supportaccess.RefusalNoGrant, s.b.OrgID},
		{"a suspended tenant", tenantCInstance, s.c.OrgID,
			supportaccess.RefusalTenant, s.c.OrgID},
		{"an instance this server does not hold", "8a4d6f4b-bf4c-4d6c-8d76-8e3a0f7b5c43", s.a.OrgID,
			supportaccess.RefusalTenant, s.a.OrgID},
		{"an audience that is not an instance id", "inst_another", s.a.OrgID,
			supportaccess.RefusalTenant, s.a.OrgID},
	}
	for _, tc := range cases {
		before := len(s.audit.Changes())
		_, err := s.svc.Enter(t.Context(), &EnterRequest{
			Assertion: s.assertion(t, assertionOptions{org: tc.org, audience: tc.audience}),
		})
		requireRefusal(t, err, tc.want)
		changes := s.audit.Changes()
		require.Len(t, changes, before+1, tc.name)
		refused := changes[before]
		assert.Equal(t, ActionEntryRefused, refused.Action, tc.name)
		assert.Equal(t, tc.audited, refused.OrganizationID, tc.name)
	}
	assert.Empty(t, s.sessions.live())
}

func TestSharedSupport_EachCheckRefusesInTheContractsOrder(t *testing.T) {
	t.Parallel()
	s := newSharedServer(t)
	grantFor(t, s.rig, s.a, supportaccess.ScopeFull, 1)

	replayed := s.assertion(t, assertionOptions{org: s.a.OrgID, audience: tenantAInstance})
	_, err := s.svc.Enter(t.Context(), &EnterRequest{Assertion: replayed})
	require.NoError(t, err)
	_, err = s.svc.Enter(t.Context(), &EnterRequest{Assertion: replayed})
	requireRefusal(t, err, supportaccess.RefusalReplayed)

	foreign := s.assertion(
		t,
		assertionOptions{org: s.b.OrgID, audience: tenantAInstance, id: "as_reused_for_test_0001"},
	)
	_, err = s.svc.Enter(t.Context(), &EnterRequest{Assertion: foreign})
	requireRefusal(t, err, supportaccess.RefusalOrganization)
	_, err = s.svc.Enter(t.Context(), &EnterRequest{Assertion: foreign})
	requireRefusal(t, err, supportaccess.RefusalReplayed)

	stale := s.assertion(t, assertionOptions{org: s.a.OrgID, audience: tenantAInstance})
	s.clock.Advance(2 * time.Minute)
	_, err = s.svc.Enter(t.Context(), &EnterRequest{Assertion: stale})
	requireRefusal(t, err, supportaccess.RefusalExpired)
}

func TestSharedSupport_KeysNameNoInstanceAndSupportCanBeTurnedOff(t *testing.T) {
	t.Parallel()
	s := newSharedServer(t)

	err := s.svc.SetKeys(t.Context(), &supportaccess.KeySet{
		InstanceID: tenantAInstance, StaffDomain: testDomain,
		Keys: []supportaccess.SigningKey{{
			KeyID: testKeyID, KeyType: "OKP", Curve: "Ed25519", Algorithm: "EdDSA", Use: "sig",
			X: base64.RawURLEncoding.EncodeToString(s.public),
		}},
	})
	require.Error(t, err, "a shared server's keys are the server's, never one tenant's")

	grantFor(t, s.rig, s.a, supportaccess.ScopeFull, 1)
	require.NoError(t, s.svc.ClearKeys(t.Context()))
	_, err = s.svc.Enter(t.Context(), &EnterRequest{
		Assertion: s.assertion(t, assertionOptions{org: s.a.OrgID, audience: tenantAInstance}),
	})
	requireRefusal(t, err, supportaccess.RefusalNotConfigured)
}

func TestDedicatedSupport_AnAssertionWithoutTheInstanceAudienceIsStillRefused(t *testing.T) {
	t.Parallel()
	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grantFor(t, r, admin, supportaccess.ScopeFull, 1)

	_, err := r.svc.Enter(t.Context(), &EnterRequest{
		Assertion: r.assertion(t, assertionOptions{org: admin.OrgID, audience: tenantAInstance}),
	})
	requireRefusal(t, err, supportaccess.RefusalAudience)
	_, err = r.svc.Enter(t.Context(), &EnterRequest{
		Assertion: r.assertion(t, assertionOptions{org: admin.OrgID}),
	})
	require.NoError(t, err)
}
