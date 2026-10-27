//go:build integration

package homelayoutrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout/homelayoutdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/homelayoutports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type presetFixture struct {
	admin *keel.DB
	repo  homelayoutports.HomeLayoutRepository
	ctx   context.Context
	info  pagination.TenantInfo
}

func newPresetFixture(t *testing.T) *presetFixture {
	t.Helper()

	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, admin)

	return &presetFixture{
		admin: admin,
		repo:  New(Params{DB: dbtest.NewRLSConnection(t, admin, dsn), Logger: zap.NewNop()}),
		ctx:   data.Ctx(t.Context()),
		info:  data.Info(),
	}
}

func (f *presetFixture) preset(name string, orgDefault bool) *homelayout.Preset {
	return &homelayout.Preset{
		OrganizationID: f.info.OrgID,
		BusinessUnitID: f.info.BuID,
		Name:           name,
		Description:    name + " layout",
		Layout: &homelayout.Layout{Widgets: []homelayout.Widget{{
			ID: "widget_1", Key: homelayout.WidgetAttention, W: 4, H: 4,
		}}},
		IsOrgDefault: orgDefault,
		Locked:       true,
		Priority:     5,
	}
}

func (f *presetFixture) stored(t *testing.T, id pulid.ID) *homelayout.Preset {
	t.Helper()

	entity, err := homelayoutdb.Presets.Get(f.ctx, f.admin, id)
	require.NoError(t, err)

	return entity
}

func TestUpdatePresetWritesZeroValues(t *testing.T) {
	f := newPresetFixture(t)

	created, err := f.repo.CreatePreset(f.ctx, f.preset("Dispatch Home", true))
	require.NoError(t, err)

	version := created.Version
	created.Name = "Dispatch Board"
	created.Description = ""
	created.IsOrgDefault = false
	created.Locked = false
	created.Priority = 0

	updated, err := f.repo.UpdatePreset(f.ctx, created)
	require.NoError(t, err)
	assert.Equal(t, version+1, updated.Version)

	stored := f.stored(t, created.ID)
	assert.Equal(t, updated.Version, stored.Version)
	assert.Equal(t, "Dispatch Board", stored.Name)
	assert.Empty(t, stored.Description)
	assert.False(t, stored.IsOrgDefault)
	assert.False(t, stored.Locked)
	assert.Zero(t, stored.Priority)
	require.NotNil(t, stored.Layout)
	assert.Len(t, stored.Layout.Widgets, 1)
}

func TestPromotingAPresetDemotesAndInvalidatesTheIncumbent(t *testing.T) {
	f := newPresetFixture(t)

	incumbent, err := f.repo.CreatePreset(f.ctx, f.preset("Incumbent", true))
	require.NoError(t, err)
	before := f.stored(t, incumbent.ID)

	challenger, err := f.repo.CreatePreset(f.ctx, f.preset("Challenger", true))
	require.NoError(t, err)

	demoted := f.stored(t, incumbent.ID)
	assert.False(t, demoted.IsOrgDefault)
	assert.Greater(t, demoted.Version, before.Version)
	assert.True(t, f.stored(t, challenger.ID).IsOrgDefault)

	_, err = f.repo.UpdatePreset(f.ctx, incumbent)
	require.Error(t, err, "an editor holding the demoted preset fails the optimistic lock")
	assert.True(t, errortypes.IsVersionMismatchError(err), err.Error())

	challenger.Priority = 9
	_, err = f.repo.UpdatePreset(f.ctx, challenger)
	require.NoError(t, err, "re-saving the default does not demote itself")
	assert.True(t, f.stored(t, challenger.ID).IsOrgDefault)
}

func TestCreatePresetKeepsTheIncumbentWhenTheInsertFails(t *testing.T) {
	f := newPresetFixture(t)

	incumbent, err := f.repo.CreatePreset(f.ctx, f.preset("Dispatch Home", true))
	require.NoError(t, err)

	_, err = f.repo.CreatePreset(f.ctx, f.preset("dispatch home", true))
	require.Error(t, err)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "name", validation.Field)
	assert.True(t, f.stored(t, incumbent.ID).IsOrgDefault)
}

func TestDeletePresetMapsMissingRowsToNotFound(t *testing.T) {
	f := newPresetFixture(t)

	created, err := f.repo.CreatePreset(f.ctx, f.preset("Doomed", false))
	require.NoError(t, err)

	req := &homelayoutports.DeleteHomeLayoutPresetRequest{TenantInfo: f.info, PresetID: created.ID}
	require.NoError(t, f.repo.DeletePreset(f.ctx, req))

	err = f.repo.DeletePreset(f.ctx, req)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}
