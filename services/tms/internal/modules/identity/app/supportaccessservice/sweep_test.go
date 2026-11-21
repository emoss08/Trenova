package supportaccessservice

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupportAccess_OneGrantThatCannotBeRecordedDoesNotHoldBackTheOthers(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	first := r.organization(newOrgID(), pulid.MustNew("bu_"))
	second := r.organization(newOrgID(), pulid.MustNew("bu_"))
	stuck := grantFor(t, r, first, supportaccess.ScopeFull, 1)
	other := grantFor(t, r, second, supportaccess.ScopeFull, 1)
	r.grants.mu.Lock()
	r.grants.markFail = map[pulid.ID]error{stuck.ID: errors.New("the database refused")}
	r.grants.mu.Unlock()

	r.clock.Advance(time.Hour + time.Second)
	result, err := r.svc.RecordExpiries(t.Context(), r.clock.Now())
	require.NoError(t, err, "a run that recorded some grants succeeds")
	assert.Equal(t, 1, result.Recorded)
	assert.Equal(t, 1, result.Failed)
	r.grants.mu.Lock()
	assert.NotNil(t, r.grants.grants[other.ID].ExpiryRecordedAt)
	assert.Nil(t, r.grants.grants[stuck.ID].ExpiryRecordedAt, "it is tried again on the next run")
	r.grants.markFail = nil
	r.grants.mu.Unlock()

	again, err := r.svc.RecordExpiries(t.Context(), r.clock.Now())
	require.NoError(t, err)
	assert.Equal(t, 1, again.Recorded)
}

func TestSupportAccess_ARunWhereEveryGrantFailsIsAnError(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeFull, 1)
	r.grants.mu.Lock()
	r.grants.markFail = map[pulid.ID]error{grant.ID: errors.New("the database refused")}
	r.grants.mu.Unlock()

	r.clock.Advance(time.Hour + time.Second)
	result, err := r.svc.RecordExpiries(t.Context(), r.clock.Now())
	require.Error(t, err, "the schedule's retry policy runs it again")
	assert.Equal(t, 1, result.Failed)
	assert.Zero(t, result.Recorded)
}

func TestSupportAccess_RemovingTheSupportKeysEndsALiveSessionAtItsNextRequest(t *testing.T) {
	t.Parallel()

	r := newRig(t, true)
	r.deliverKeys(t)
	admin := r.organization(newOrgID(), pulid.MustNew("bu_"))
	grant := grantFor(t, r, admin, supportaccess.ScopeFull, 4)
	_, err := r.svc.Enter(t.Context(), &EnterRequest{
		Assertion: r.assertion(t, assertionOptions{org: admin.OrgID}),
	})
	require.NoError(t, err)
	live := &supportaccessports.RequireLiveRequest{
		OrganizationID:  admin.OrgID,
		BusinessUnitID:  admin.BuID,
		GrantID:         grant.ID,
		PrincipalUserID: grant.PrincipalUserID,
		Now:             r.clock.Now(),
	}
	require.NoError(t, r.principals.RequireLive(t.Context(), live), "the session is live")

	require.NoError(t, r.svc.ClearKeys(t.Context()))
	require.ErrorIs(t, r.principals.RequireLive(t.Context(), live), supportaccess.ErrGrantNotFound,
		"support access turned off ends the session at its next request, on any process")

	r.deliverKeys(t)
	require.NoError(t, r.principals.RequireLive(t.Context(), live),
		"the grant is still the organization's while it lasts")
}
