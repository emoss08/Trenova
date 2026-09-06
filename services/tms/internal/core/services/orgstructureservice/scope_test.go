package orgstructureservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scopeRepo answers the two questions the scope check asks: what has been
// delegated to this user, and do these managers cover this worker.
type scopeRepo struct {
	repositories.OrgStructureRepository
	delegations []*worker.ApprovalDelegation
	// manages maps a manager onto the workers they cover.
	manages map[pulid.ID]map[pulid.ID]bool
	// asked records every ManagesWorker call, so a test can prove the check
	// did not fan out over the whole roster.
	asked int
}

func (r *scopeRepo) ListDelegations(
	_ context.Context,
	req *repositories.ListApprovalDelegationsRequest,
) ([]*worker.ApprovalDelegation, error) {
	out := make([]*worker.ApprovalDelegation, 0, len(r.delegations))
	for _, row := range r.delegations {
		if !req.DelegateID.IsNil() && row.DelegateID != req.DelegateID {
			continue
		}
		if !req.DelegatorID.IsNil() && row.DelegatorID != req.DelegatorID {
			continue
		}
		if req.ActiveAt > 0 && !row.IsActive(req.ActiveAt) {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func (r *scopeRepo) ManagesWorker(
	_ context.Context,
	_ pagination.TenantInfo,
	managerIDs []pulid.ID,
	workerID pulid.ID,
) (bool, error) {
	r.asked++
	for _, managerID := range managerIDs {
		if r.manages[managerID][workerID] {
			return true, nil
		}
	}
	return false, nil
}

const (
	managerUser  = pulid.ID("usr_manager")
	coverUser    = pulid.ID("usr_cover")
	strangerUser = pulid.ID("usr_stranger")
	teamWorker   = pulid.ID("wrk_team")
	otherWorker  = pulid.ID("wrk_other")
	asOf         = int64(1_800_000_000)
)

func newScopeService(repo *scopeRepo) *orgstructureservice.Service {
	return orgstructureservice.NewWithDeps(orgstructureservice.Deps{Repo: repo})
}

func scopeTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestCanActFor_TheirOwnPeople(t *testing.T) {
	t.Parallel()

	repo := &scopeRepo{
		manages: map[pulid.ID]map[pulid.ID]bool{managerUser: {teamWorker: true}},
	}

	result, err := newScopeService(repo).CanActFor(t.Context(), &orgstructureservice.ScopeRequest{
		TenantInfo: scopeTenant(),
		UserID:     managerUser,
		WorkerID:   teamWorker,
		Scope:      worker.ApprovalScopeTimeOff,
		AsOf:       asOf,
	})
	require.NoError(t, err)

	assert.True(t, result.Allowed)
	// Nothing was borrowed, so nothing should be recorded as borrowed.
	assert.True(t, result.OnBehalfOf.IsNil())
}

func TestCanActFor_RefusesSomebodyElsesWorker(t *testing.T) {
	t.Parallel()

	repo := &scopeRepo{
		manages: map[pulid.ID]map[pulid.ID]bool{managerUser: {teamWorker: true}},
	}

	result, err := newScopeService(repo).CanActFor(t.Context(), &orgstructureservice.ScopeRequest{
		TenantInfo: scopeTenant(),
		UserID:     managerUser,
		WorkerID:   otherWorker,
		AsOf:       asOf,
	})
	require.NoError(t, err)

	assert.False(t, result.Allowed)
}

// The delegate does not manage the worker in their own right; they reach them
// only through the person who handed over. The answer has to say whose
// authority was used, so the approval can be explained afterwards.
func TestCanActFor_ThroughADelegation(t *testing.T) {
	t.Parallel()

	repo := &scopeRepo{
		delegations: []*worker.ApprovalDelegation{
			{
				DelegatorID: managerUser,
				DelegateID:  coverUser,
				Scope:       worker.ApprovalScopeTimeOff,
				StartsAt:    asOf - 86400,
			},
		},
		manages: map[pulid.ID]map[pulid.ID]bool{managerUser: {teamWorker: true}},
	}

	result, err := newScopeService(repo).CanActFor(t.Context(), &orgstructureservice.ScopeRequest{
		TenantInfo: scopeTenant(),
		UserID:     coverUser,
		WorkerID:   teamWorker,
		Scope:      worker.ApprovalScopeTimeOff,
		AsOf:       asOf,
	})
	require.NoError(t, err)

	assert.True(t, result.Allowed)
	assert.Equal(t, managerUser, result.OnBehalfOf)
}

// A delegation that quietly covered more than it said would be worse than no
// delegation at all.
func TestCanActFor_DelegationDoesNotCoverAnotherScope(t *testing.T) {
	t.Parallel()

	repo := &scopeRepo{
		delegations: []*worker.ApprovalDelegation{
			{
				DelegatorID: managerUser,
				DelegateID:  coverUser,
				Scope:       worker.ApprovalScopeExpenses,
				StartsAt:    asOf - 86400,
			},
		},
		manages: map[pulid.ID]map[pulid.ID]bool{managerUser: {teamWorker: true}},
	}

	result, err := newScopeService(repo).CanActFor(t.Context(), &orgstructureservice.ScopeRequest{
		TenantInfo: scopeTenant(),
		UserID:     coverUser,
		WorkerID:   teamWorker,
		Scope:      worker.ApprovalScopeTimeOff,
		AsOf:       asOf,
	})
	require.NoError(t, err)

	assert.False(t, result.Allowed)
}

func TestCanActFor_RevokedDelegationStopsAnswering(t *testing.T) {
	t.Parallel()

	revoked := asOf - 3600
	repo := &scopeRepo{
		delegations: []*worker.ApprovalDelegation{
			{
				DelegatorID: managerUser,
				DelegateID:  coverUser,
				Scope:       worker.ApprovalScopeAll,
				StartsAt:    asOf - 86400,
				RevokedAt:   &revoked,
			},
		},
		manages: map[pulid.ID]map[pulid.ID]bool{managerUser: {teamWorker: true}},
	}

	result, err := newScopeService(repo).CanActFor(t.Context(), &orgstructureservice.ScopeRequest{
		TenantInfo: scopeTenant(),
		UserID:     coverUser,
		WorkerID:   teamWorker,
		AsOf:       asOf,
	})
	require.NoError(t, err)

	assert.False(t, result.Allowed)
}

// A delegation hands over what the delegator has. Delegating from somebody who
// manages nobody hands over nothing, which is correct rather than a bug.
func TestCanActFor_DelegationFromSomebodyWhoManagesNobody(t *testing.T) {
	t.Parallel()

	repo := &scopeRepo{
		delegations: []*worker.ApprovalDelegation{
			{
				DelegatorID: strangerUser,
				DelegateID:  coverUser,
				Scope:       worker.ApprovalScopeAll,
				StartsAt:    asOf - 86400,
			},
		},
		manages: map[pulid.ID]map[pulid.ID]bool{managerUser: {teamWorker: true}},
	}

	result, err := newScopeService(repo).CanActFor(t.Context(), &orgstructureservice.ScopeRequest{
		TenantInfo: scopeTenant(),
		UserID:     coverUser,
		WorkerID:   teamWorker,
		AsOf:       asOf,
	})
	require.NoError(t, err)

	assert.False(t, result.Allowed)
}

// The authorization question is answered directly rather than by listing a
// team and searching it: a manager of four hundred drivers must not pull four
// hundred rows to approve one day of leave.
func TestCanActFor_AsksTheDatabaseRatherThanListingTheTeam(t *testing.T) {
	t.Parallel()

	repo := &scopeRepo{
		manages: map[pulid.ID]map[pulid.ID]bool{managerUser: {teamWorker: true}},
	}

	_, err := newScopeService(repo).CanActFor(t.Context(), &orgstructureservice.ScopeRequest{
		TenantInfo: scopeTenant(),
		UserID:     managerUser,
		WorkerID:   teamWorker,
		AsOf:       asOf,
	})
	require.NoError(t, err)

	assert.Equal(t, 1, repo.asked, "the direct answer needs exactly one check")
}

func TestManagerIDs_IncludesActiveDelegatorsOnly(t *testing.T) {
	t.Parallel()

	ended := asOf - 3600
	repo := &scopeRepo{
		delegations: []*worker.ApprovalDelegation{
			{
				DelegatorID: managerUser,
				DelegateID:  coverUser,
				Scope:       worker.ApprovalScopeAll,
				StartsAt:    asOf - 86400,
			},
			{
				DelegatorID: strangerUser,
				DelegateID:  coverUser,
				Scope:       worker.ApprovalScopeAll,
				StartsAt:    asOf - 200000,
				EndsAt:      &ended,
			},
		},
	}

	ids, via, err := newScopeService(repo).ManagerIDs(
		t.Context(),
		scopeTenant(),
		coverUser,
		worker.ApprovalScopeAll,
		asOf,
	)
	require.NoError(t, err)

	assert.Contains(t, ids, coverUser, "a user always answers for themselves")
	assert.Contains(t, ids, managerUser)
	assert.NotContains(t, ids, strangerUser, "an expired delegation hands over nothing")
	assert.Contains(t, via, managerUser)
}
