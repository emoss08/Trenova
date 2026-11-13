package pagetourservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/app/pagetourservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagetour"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/pagetourports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const now = int64(1_800_000_000)

var tenant = pagination.TenantInfo{
	OrgID:  pulid.ID("org_a"),
	BuID:   pulid.ID("bu_a"),
	UserID: pulid.ID("usr_ana"),
}

type fakeRepo struct {
	views     []*pagetour.View
	exists    bool
	count     int
	err       error
	upserted  []*pagetour.View
	listedFor []pagination.TenantInfo
	counted   int
}

func (f *fakeRepo) List(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*pagetour.View, error) {
	f.listedFor = append(f.listedFor, tenantInfo)

	return f.views, f.err
}

func (f *fakeRepo) Exists(context.Context, *pagetourports.ViewRequest) (bool, error) {
	return f.exists, f.err
}

func (f *fakeRepo) CountForUser(context.Context, pagination.TenantInfo) (int, error) {
	f.counted++

	return f.count, f.err
}

func (f *fakeRepo) Upsert(_ context.Context, entity *pagetour.View) (*pagetour.View, error) {
	f.upserted = append(f.upserted, entity)

	return entity, f.err
}

func newService(repo *fakeRepo) *pagetourservice.Service {
	return pagetourservice.New(pagetourservice.Params{
		Logger: zap.NewNop(),
		Repo:   repo,
		Clock:  func() int64 { return now },
	})
}

func record(outcome pagetour.Outcome) *pagetourservice.RecordRequest {
	return &pagetourservice.RecordRequest{
		TenantInfo:  tenant,
		Tour:        "freight-outlook",
		TourVersion: 2,
		Outcome:     outcome,
	}
}

func TestRecord_RemembersTheCallersOwnViewAtTheMomentItHappened(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	view, err := newService(repo).Record(t.Context(), record(pagetour.OutcomeSkipped))
	require.NoError(t, err)

	require.Len(t, repo.upserted, 1)
	assert.Same(t, repo.upserted[0], view)
	assert.Equal(t, &pagetour.View{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		UserID:         tenant.UserID,
		Tour:           "freight-outlook",
		TourVersion:    2,
		Outcome:        pagetour.OutcomeSkipped,
		SeenAt:         now,
	}, view)
}

func TestRecord_RefusesWhatCannotBeATourBeforeTouchingTheStore(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	req := record(pagetour.OutcomeFinished)
	req.Tour = "../../etc/passwd"

	_, err := newService(repo).Record(t.Context(), req)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Empty(t, repo.upserted)
	assert.Zero(t, repo.counted)
}

func TestRecord_SeesATourAgainWithoutCountingAndStopsANewOneAtTheCeiling(t *testing.T) {
	t.Parallel()

	seenBefore := &fakeRepo{exists: true, count: pagetour.MaxViewsPerUser}
	_, err := newService(seenBefore).Record(t.Context(), record(pagetour.OutcomeFinished))
	require.NoError(t, err, "replaying a tour replaces its own row, so the ceiling does not apply")
	assert.Zero(t, seenBefore.counted)
	assert.Len(t, seenBefore.upserted, 1)

	full := &fakeRepo{count: pagetour.MaxViewsPerUser}
	_, err = newService(full).Record(t.Context(), record(pagetour.OutcomeFinished))
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Empty(t, full.upserted)

	room := &fakeRepo{count: pagetour.MaxViewsPerUser - 1}
	_, err = newService(room).Record(t.Context(), record(pagetour.OutcomeFinished))
	require.NoError(t, err)
	assert.Len(t, room.upserted, 1)
}

func TestRecord_ReturnsTheStoresError(t *testing.T) {
	t.Parallel()

	down := errors.New("store down")
	_, err := newService(&fakeRepo{err: down}).Record(t.Context(), record(pagetour.OutcomeFinished))

	require.ErrorIs(t, err, down)
}

func TestList_ReadsTheCallersViews(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{views: []*pagetour.View{{Tour: "freight-outlook", TourVersion: 1}}}
	views, err := newService(repo).List(t.Context(), tenant)
	require.NoError(t, err)

	assert.Equal(t, repo.views, views)
	assert.Equal(t, []pagination.TenantInfo{tenant}, repo.listedFor)
}
