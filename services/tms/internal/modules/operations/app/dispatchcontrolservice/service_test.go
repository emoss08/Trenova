package dispatchcontrolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchcontrolportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	org   = pulid.ID("org_a")
	bu    = pulid.ID("bu_a")
	actor = &permissionports.RequestActor{
		UserID:         pulid.ID("usr_a"),
		OrganizationID: org,
		BusinessUnitID: bu,
	}
)

type allFields struct{}

func (allFields) Visible(context.Context, *permissionports.RequestActor) protectedfield.GroupSet {
	return protectedfield.AllGroups()
}

type fakeLearning struct {
	started []pagination.TenantInfo
}

func (f *fakeLearning) StartLearning(_ context.Context, tenantInfo pagination.TenantInfo) error {
	f.started = append(f.started, tenantInfo)

	return nil
}

type fakeFitStarter struct {
	err     error
	started []pagination.TenantInfo
}

func (f *fakeFitStarter) StartFitting(_ context.Context, tenantInfo pagination.TenantInfo) error {
	f.started = append(f.started, tenantInfo)

	return f.err
}

func control(anticipate bool) *dispatchcontrol.DispatchControl {
	entity := dispatchcontrol.NewDefaultDispatchControl(org, bu)
	entity.ID = pulid.ID("dc_a")
	entity.AnticipateFreight = anticipate

	return entity
}

func newService(
	t *testing.T,
	stored *dispatchcontrol.DispatchControl,
	fits *fakeFitStarter,
) (*Service, *fakeLearning) {
	t.Helper()
	repo := dispatchcontrolportsmock.NewMockDispatchControlRepository(t)
	repo.EXPECT().
		GetByOrgID(mock.Anything, dispatchcontrolports.GetDispatchControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: org, BuID: bu, UserID: actor.UserID},
		}).
		Return(stored, nil).
		Once()
	repo.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			entity *dispatchcontrol.DispatchControl,
		) (*dispatchcontrol.DispatchControl, error) {
			return entity, nil
		}).
		Once()
	audit := auditportsmock.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	learning := &fakeLearning{}

	return New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Validator:    NewValidator(ValidatorParams{}),
		AuditService: audit,
		Fields:       allFields{},
		Learning:     learning,
		Forecasts:    fits,
	}), learning
}

func TestUpdate_FitsTheForecastNowOnlyWhenAnticipationIsTurnedOn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before bool
		after  bool
		starts []pagination.TenantInfo
	}{
		{
			name:   "turned on",
			after:  true,
			starts: []pagination.TenantInfo{{OrgID: org, BuID: bu}},
		},
		{name: "left on", before: true, after: true},
		{name: "turned off", before: true},
		{name: "left off"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fits := &fakeFitStarter{}
			svc, learning := newService(t, control(tt.before), fits)

			updated, err := svc.Update(t.Context(), control(tt.after), actor)
			require.NoError(t, err)

			assert.Equal(t, tt.after, updated.AnticipateFreight)
			assert.Equal(t, tt.starts, fits.started)
			assert.Empty(t, learning.started)
		})
	}
}

// Temporal being unreachable when an administrator turns anticipation on must
// not lose the setting: the nightly fit picks the tenant up instead.
func TestUpdate_SavesTheSettingWhenTheFitCannotStartNow(t *testing.T) {
	t.Parallel()

	fits := &fakeFitStarter{err: errors.New("temporal down")}
	svc, _ := newService(t, control(false), fits)

	updated, err := svc.Update(t.Context(), control(true), actor)

	require.NoError(t, err)
	assert.True(t, updated.AnticipateFreight)
	assert.Len(t, fits.started, 1)
}
