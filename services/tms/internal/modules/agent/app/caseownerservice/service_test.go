package caseownerservice

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/caseowner"
	"github.com/emoss08/trenova/internal/modules/agent/ports/caseownerports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordshareports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/approvaldelegation"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/orgstructureports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type grants struct {
	permissionports.PermissionEngine

	held map[pulid.ID][]permission.Operation
}

func (g grants) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	for _, operation := range g.held[req.UserID] {
		if operation == req.Operation {
			return &permissionports.PermissionCheckResult{Allowed: true}, nil
		}
	}

	return &permissionports.PermissionCheckResult{}, nil
}

type handovers struct {
	active []*approvaldelegation.ApprovalDelegation
	asked  *orgstructureports.ListApprovalDelegationsRequest
}

func (h *handovers) ListDelegations(
	_ context.Context,
	req *orgstructureports.ListApprovalDelegationsRequest,
) ([]*approvaldelegation.ApprovalDelegation, error) {
	h.asked = req
	out := make([]*approvaldelegation.ApprovalDelegation, 0, len(h.active))
	for _, delegation := range h.active {
		if delegation.DelegateID == req.DelegateID {
			out = append(out, delegation)
		}
	}

	return out, nil
}

// sharedRecords is which records were shared with whom, by recipient.
type sharedRecords struct {
	with  map[pulid.ID][]pulid.ID
	asked int
}

func (s *sharedRecords) SharedWith(
	_ context.Context,
	req *recordshareports.SharedWithRequest,
) ([]pulid.ID, error) {
	s.asked++
	out := make([]pulid.ID, 0, len(req.RecordIDs))
	for _, id := range s.with[req.UserID] {
		if slices.Contains(req.RecordIDs, id) {
			out = append(out, id)
		}
	}

	return out, nil
}

type owners struct {
	caseownerports.Repository

	held       map[pulid.ID]*caseowner.Owner
	listed     int
	candidates []pulid.ID
	lost       bool
	// away is who is away right now, each with the person covering, or nil.
	away map[pulid.ID]pulid.ID
}

func (o *owners) AwayFor(
	_ context.Context,
	req caseownerports.AwayRequest,
) ([]pulid.ID, error) {
	out := make([]pulid.ID, 0, len(o.away))
	for person, cover := range o.away {
		if cover == req.CoverID {
			out = append(out, person)
		}
	}

	return out, nil
}

func (o *owners) Uncovered(
	_ context.Context,
	req caseownerports.AwayRequest,
) ([]pulid.ID, error) {
	out := make([]pulid.ID, 0, len(req.UserIDs))
	for _, person := range req.UserIDs {
		if cover, away := o.away[person]; away && cover.IsNil() {
			out = append(out, person)
		}
	}

	return out, nil
}

func (o *owners) Get(
	_ context.Context,
	req caseownerports.RecordRequest,
) (*caseowner.Owner, error) {
	return o.held[req.RecordID], nil
}

func (o *owners) List(
	_ context.Context,
	req caseownerports.RecordsRequest,
) ([]*caseowner.Owner, error) {
	o.listed++
	out := make([]*caseowner.Owner, 0, len(req.RecordIDs))
	for _, id := range req.RecordIDs {
		if owner, ok := o.held[id]; ok {
			out = append(out, owner)
		}
	}

	return out, nil
}

func (o *owners) Set(
	_ context.Context,
	req *caseownerports.SetOwnerRequest,
) (*caseowner.Owner, bool, error) {
	if o.lost {
		return nil, false, nil
	}
	owner := &caseowner.Owner{
		RecordKind: req.Kind,
		RecordID:   req.RecordID,
		OwnerID:    req.OwnerID,
		Source:     req.Source,
	}
	o.held[req.RecordID] = owner

	return owner, true, nil
}

func (o *owners) SetIfUnowned(_ context.Context, req *caseownerports.SetOwnerRequest) error {
	if _, owned := o.held[req.RecordID]; !owned {
		o.held[req.RecordID] = &caseowner.Owner{
			RecordID: req.RecordID,
			OwnerID:  req.OwnerID,
			Source:   req.Source,
		}
	}

	return nil
}

func (o *owners) Release(
	_ context.Context,
	req caseownerports.RecordRequest,
	_ *caseowner.Owner,
) (bool, error) {
	delete(o.held, req.RecordID)

	return true, nil
}

func (o *owners) Candidates(
	context.Context,
	caseownerports.CandidatesRequest,
) ([]pulid.ID, error) {
	return o.candidates, nil
}

type quietAudit struct {
	changes []*permissionports.SecurityChange
}

func (a *quietAudit) RecordChange(_ context.Context, change *permissionports.SecurityChange) {
	a.changes = append(a.changes, change)
}

type fixture struct {
	svc       *Service
	repo      *owners
	grants    grants
	handovers *handovers
	shares    *sharedRecords
	audit     *quietAudit
	org, bu   pulid.ID
}

var works = []permission.Operation{permission.OpRead, permission.OpUpdate}

func newFixture() *fixture {
	f := &fixture{
		repo:      &owners{held: map[pulid.ID]*caseowner.Owner{}},
		grants:    grants{held: map[pulid.ID][]permission.Operation{}},
		handovers: &handovers{},
		shares:    &sharedRecords{with: map[pulid.ID][]pulid.ID{}},
		audit:     &quietAudit{},
		org:       pulid.MustNew("org_"),
		bu:        pulid.MustNew("bu_"),
	}
	f.svc = &Service{
		l:           zap.NewNop(),
		repo:        f.repo,
		permissions: f.grants,
		delegations: f.handovers,
		auditor:     f.audit,
		realtime:    &mocks.NoopRealtimeService{},
		shares:      f.shares,
		now:         func() int64 { return 1_000 },
	}

	return f
}

func (f *fixture) person(operations ...permission.Operation) *permissionports.RequestActor {
	id := pulid.MustNew("usr_")
	f.grants.held[id] = operations

	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    id,
		UserID:         id,
		OrganizationID: f.org,
		BusinessUnitID: f.bu,
	}
}

func (f *fixture) record() caseownerports.RecordRequest {
	return caseownerports.RecordRequest{RecordID: pulid.MustNew("shp_")}
}

/*
A case someone else owns is neither shown to nor worked by a person who does
not own it. Its owner works it, an unowned case is anyone's, and a person who
sees every case reads it without being able to decide it.
*/
func TestStanding_KeepsACaseToItsOwner(t *testing.T) {
	t.Parallel()

	f := newFixture()
	owner, other, lead := f.person(works...), f.person(works...), f.person(
		permission.OpRead, permission.OpUpdate, permission.OpManage,
	)
	record := f.record()

	unowned, err := f.svc.Standing(t.Context(), record, other)
	require.NoError(t, err)
	assert.True(t, unowned.CanSee)
	assert.True(t, unowned.CanWork)
	assert.True(t, unowned.CanClaim)

	f.repo.held[record.RecordID] = &caseowner.Owner{RecordID: record.RecordID, OwnerID: owner.UserID}

	mine, err := f.svc.Standing(t.Context(), record, owner)
	require.NoError(t, err)
	assert.True(t, mine.CanWork)
	assert.False(t, mine.Covering)

	theirs, err := f.svc.Standing(t.Context(), record, other)
	require.NoError(t, err)
	assert.False(t, theirs.CanSee)
	assert.False(t, theirs.CanWork)
	assert.False(t, theirs.CanClaim)

	seen, err := f.svc.Standing(t.Context(), record, lead)
	require.NoError(t, err)
	assert.True(t, seen.CanSee, "someone who sees every case reads it")
	assert.False(t, seen.CanWork, "and still does not decide it")

	err = f.svc.AssertMayWork(t.Context(), record, other)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	require.NoError(t, f.svc.AssertMayWork(t.Context(), record, owner))
}

// A page of cases reads its owners once and gives every record asked for the
// same standing one read of it would, owned or not.
func TestStandings_AnswersAPageOfCasesInOneRead(t *testing.T) {
	t.Parallel()

	f := newFixture()
	owner, other := f.person(works...), f.person(works...)
	mine, theirs, unowned := f.record().RecordID, f.record().RecordID, f.record().RecordID
	f.repo.held[mine] = &caseowner.Owner{RecordID: mine, OwnerID: other.UserID}
	f.repo.held[theirs] = &caseowner.Owner{RecordID: theirs, OwnerID: owner.UserID}

	standings, err := f.svc.Standings(t.Context(), caseownerports.RecordsRequest{
		RecordIDs: []pulid.ID{mine, theirs, unowned},
	}, other)

	require.NoError(t, err)
	assert.Equal(t, 1, f.repo.listed)
	require.Len(t, standings, 3)
	assert.True(t, standings[mine].CanWork)
	assert.Equal(t, other.UserID, standings[mine].Owner.OwnerID)
	assert.False(t, standings[theirs].CanWork)
	assert.False(t, standings[theirs].CanSee)
	assert.Nil(t, standings[unowned].Owner)
	assert.True(t, standings[unowned].CanClaim)
}

/*
Sharing a record with someone brings its case into their sight: they read a
case another person owns, and still decide nothing on it. Nobody else gains
anything from the share, and it is read once for a page of cases.
*/
func TestStanding_ASharedRecordsCaseIsReadByWhoItWasSharedWith(t *testing.T) {
	t.Parallel()

	f := newFixture()
	owner, invited, stranger := f.person(works...), f.person(works...), f.person(works...)
	record := f.record()
	f.repo.held[record.RecordID] = &caseowner.Owner{RecordID: record.RecordID, OwnerID: owner.UserID}
	f.shares.with[invited.UserID] = []pulid.ID{record.RecordID}

	shared, err := f.svc.Standing(t.Context(), record, invited)
	require.NoError(t, err)
	assert.True(t, shared.CanSee, "the case was shared with them")
	assert.False(t, shared.CanWork, "and its owner still decides it")
	assert.False(t, shared.CanClaim)
	require.Error(t, f.svc.AssertMayWork(t.Context(), record, invited))

	other, err := f.svc.Standing(t.Context(), record, stranger)
	require.NoError(t, err)
	assert.False(t, other.CanSee)

	sight, err := f.svc.Sight(t.Context(), invited)
	require.NoError(t, err)
	assert.Equal(t, invited.UserID, sight.Viewer, "lists bring shared cases in by the reader")

	f.shares.asked = 0
	page, err := f.svc.Standings(t.Context(), caseownerports.RecordsRequest{
		RecordIDs: []pulid.ID{record.RecordID, f.record().RecordID},
	}, invited)
	require.NoError(t, err)
	assert.True(t, page[record.RecordID].CanSee)
	assert.Equal(t, 1, f.shares.asked)
}

/*
While an owner's delegation of their cases runs, the person covering works the
owner's cases as their own. A delegation of something else hands over no case.
*/
func TestSight_CoversForAnOwnerWhoHandedTheirCasesOver(t *testing.T) {
	t.Parallel()

	f := newFixture()
	owner, cover, approver := f.person(works...), f.person(works...), f.person(works...)
	f.handovers.active = []*approvaldelegation.ApprovalDelegation{
		{
			DelegatorID: owner.UserID,
			DelegateID:  cover.UserID,
			Scope:       approvaldelegation.ApprovalScopeCases,
		},
		{
			DelegatorID: owner.UserID,
			DelegateID:  approver.UserID,
			Scope:       approvaldelegation.ApprovalScopeTimeOff,
		},
	}
	record := f.record()
	f.repo.held[record.RecordID] = &caseowner.Owner{RecordID: record.RecordID, OwnerID: owner.UserID}

	sight, err := f.svc.Sight(t.Context(), cover)
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{cover.UserID, owner.UserID}, sight.As)
	assert.Equal(t, int64(1_000), f.handovers.asked.ActiveAt, "only delegations running now count")

	covering, err := f.svc.Standing(t.Context(), record, cover)
	require.NoError(t, err)
	assert.True(t, covering.CanWork)
	assert.True(t, covering.Covering)

	timeOff, err := f.svc.Standing(t.Context(), record, approver)
	require.NoError(t, err)
	assert.False(t, timeOff.CanSee, "covering time-off approvals hands over no case")
}

// What is not a person acting for themselves, an agent run or a job, is not
// what ownership keeps apart.
func TestSight_IsEveryCaseForWhatIsNotAPerson(t *testing.T) {
	t.Parallel()

	f := newFixture()
	system := &permissionports.RequestActor{PrincipalType: permissionports.PrincipalTypeSystem}

	sight, err := f.svc.Sight(t.Context(), system)
	require.NoError(t, err)
	assert.True(t, sight.All)
	require.NoError(t, f.svc.AssertMayWork(t.Context(), f.record(), system))
}

/*
An unowned case is claimed by anyone who works cases. One that has an owner is
taken over only by someone who may assign cases, and a claim that lost a race
is told so rather than overwriting the winner.
*/
func TestClaim_TakesAnUnownedCaseAndRefusesAnotherPersons(t *testing.T) {
	t.Parallel()

	f := newFixture()
	first, second, lead := f.person(works...), f.person(works...), f.person(
		permission.OpRead, permission.OpUpdate, permission.OpAssign, permission.OpManage,
	)
	record := f.record()
	req := &caseownerports.AssignRequest{RecordID: record.RecordID}

	owner, err := f.svc.Claim(t.Context(), req, first)
	require.NoError(t, err)
	assert.Equal(t, first.UserID, owner.OwnerID)
	assert.Equal(t, caseowner.SourceClaimed, owner.Source)
	require.Len(t, f.audit.changes, 1, "a change of owner is audited")
	assert.Equal(t, permission.OpAssign, f.audit.changes[0].Operation)

	_, err = f.svc.Claim(t.Context(), req, second)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))

	owner, err = f.svc.Claim(t.Context(), req, lead)
	require.NoError(t, err)
	assert.Equal(t, lead.UserID, owner.OwnerID)

	f.repo.lost = true
	delete(f.repo.held, record.RecordID)
	_, err = f.svc.Claim(t.Context(), req, first)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
}

/*
A record names the people its case may go to, most specific first. The case
goes to the first of them who may work cases, a named person who may not is
passed over, and a case that already has an owner is left alone.
*/
func TestRoute_GivesAnUnownedCaseToTheFirstNamedPersonWhoWorksCases(t *testing.T) {
	t.Parallel()

	f := newFixture()
	driverManager, biller := f.person(permission.OpRead), f.person(works...)
	f.repo.candidates = []pulid.ID{biller.UserID}
	record := f.record()
	req := caseownerports.CandidatesRequest{RecordID: record.RecordID}

	f.svc.Route(t.Context(), req, pulid.Nil, driverManager.UserID)

	routed := f.repo.held[record.RecordID]
	require.NotNil(t, routed)
	assert.Equal(t, biller.UserID, routed.OwnerID)
	assert.Equal(t, caseowner.SourceAuto, routed.Source)

	f.repo.candidates = []pulid.ID{f.person(works...).UserID}
	f.svc.Route(t.Context(), req)
	assert.Equal(t, biller.UserID, f.repo.held[record.RecordID].OwnerID)
}

func TestRoute_LeavesACaseUnownedWhenNobodyNamedWorksCases(t *testing.T) {
	t.Parallel()

	f := newFixture()
	record := f.record()

	f.svc.Route(t.Context(), caseownerports.CandidatesRequest{RecordID: record.RecordID},
		f.person(permission.OpRead).UserID)

	assert.Nil(t, f.repo.held[record.RecordID])
}

/*
Someone who marks themselves away and names a cover hands that person their
cases until they are back. A case is never routed to someone away with nobody
covering, where it would wait unseen: it goes to the next person named.
*/
func TestAway_TheNamedCoverWorksTheCasesAndRoutingPassesOverAnUncoveredOwner(t *testing.T) {
	t.Parallel()

	f := newFixture()
	owner, cover, onLeave, next := f.person(works...), f.person(works...),
		f.person(works...), f.person(works...)
	f.repo.away = map[pulid.ID]pulid.ID{owner.UserID: cover.UserID, onLeave.UserID: pulid.Nil}
	record := f.record()
	f.repo.held[record.RecordID] = &caseowner.Owner{RecordID: record.RecordID, OwnerID: owner.UserID}

	covering, err := f.svc.Standing(t.Context(), record, cover)
	require.NoError(t, err)
	assert.True(t, covering.CanWork)
	assert.True(t, covering.Covering)

	unrouted := f.record()
	f.repo.candidates = []pulid.ID{next.UserID}
	f.svc.Route(t.Context(), caseownerports.CandidatesRequest{RecordID: unrouted.RecordID},
		onLeave.UserID)

	require.NotNil(t, f.repo.held[unrouted.RecordID])
	assert.Equal(t, next.UserID, f.repo.held[unrouted.RecordID].OwnerID)
}
