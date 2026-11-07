package driverpositioningservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/driverpositioningports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testNow = int64(1_793_577_600 + 9*3600)

var (
	tenant = pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")}
	userID = pulid.ID("usr_dispatcher")
	worker = pulid.ID("wrk_ana")
	dallas = dispatchmarket.Point{Latitude: 32.7767, Longitude: -96.7970}
	okc    = dispatchmarket.Point{Latitude: 35.4676, Longitude: -97.5164}
	actor  = &permissionports.RequestActor{
		PrincipalType: permissionports.PrincipalTypeUser,
		PrincipalID:   userID,
		UserID:        userID,
	}
	readyQ  = &demandforecast.Quality{Status: demandforecast.StatusReady}
	slotNow = dispatchmarket.SlotStart(testNow)
)

func cellOf(t *testing.T, p dispatchmarket.Point) int64 {
	t.Helper()
	cell, ok := dispatchmarket.CellOf(p)
	require.True(t, ok)
	return cell
}

type fakeRepo struct {
	replaced *driverpositioningports.ReplaceRequest
	previous *driverpositioning.Positioning
	current  *driverpositioning.Positioning
	canceled *driverpositioningports.CancelRequest
	err      error
}

func (f *fakeRepo) Get(
	context.Context,
	*driverpositioningports.GetRequest,
) (*driverpositioning.Positioning, error) {
	if f.current == nil {
		return nil, errortypes.NewNotFoundError("Positioning not found")
	}
	copied := *f.current
	return &copied, nil
}

func (f *fakeRepo) ListLive(
	context.Context,
	*driverpositioningports.LiveRequest,
) ([]*driverpositioning.Positioning, error) {
	return nil, nil
}

func (f *fakeRepo) Replace(
	_ context.Context,
	req *driverpositioningports.ReplaceRequest,
) (*driverpositioningports.Replaced, error) {
	f.replaced = req
	if f.err != nil {
		return nil, f.err
	}
	req.Positioning.ID = pulid.ID("dpo_new")
	return &driverpositioningports.Replaced{Positioning: req.Positioning, Previous: f.previous}, nil
}

func (f *fakeRepo) Cancel(
	_ context.Context,
	req *driverpositioningports.CancelRequest,
) (*driverpositioning.Positioning, error) {
	f.canceled = req
	canceled := *f.current
	canceled.Status = driverpositioning.StatusCanceled
	canceled.CanceledByID = req.CanceledByID
	canceled.CanceledAt = &req.CanceledAt
	return &canceled, nil
}

type fakeStarts struct {
	rows []demandforecastports.DriverStart
	req  *demandforecastports.DriverStartsRequest
}

func (f *fakeStarts) ListDriverStarts(
	_ context.Context,
	req *demandforecastports.DriverStartsRequest,
) ([]demandforecastports.DriverStart, error) {
	f.req = req
	return f.rows, nil
}

type fakeForecasts struct {
	areas   []*demandforecast.Area
	quality *demandforecast.Quality
}

func (f *fakeForecasts) Forecaster(
	_ context.Context,
	req *demandforecastports.ForecasterRequest,
) (*demandforecast.Forecaster, error) {
	return demandforecast.NewForecaster(&demandforecast.ForecasterParams{
		Now:           req.Now,
		Quality:       f.quality,
		Areas:         f.areas,
		MarginPercent: req.MarginPercent,
		CapCents:      req.CapCents,
	}), nil
}

func (f *fakeForecasts) Location(context.Context, pagination.TenantInfo) *time.Location {
	return time.UTC
}

func (f *fakeForecasts) Economics(
	context.Context,
	pagination.TenantInfo,
) (margin, capCents int64, err error) {
	return 80, 0, nil
}

type fakeNotifier struct {
	sent []*drivernotificationservice.DriverNotification
}

func (f *fakeNotifier) Notify(
	_ context.Context,
	req *drivernotificationservice.DriverNotification,
) {
	f.sent = append(f.sent, req)
}

type rig struct {
	svc      *Service
	repo     *fakeRepo
	starts   *fakeStarts
	notifier *fakeNotifier
	audit    *auditportsmock.MockAuditService
}

func newRig(t *testing.T) *rig {
	t.Helper()
	r := &rig{
		repo: &fakeRepo{},
		starts: &fakeStarts{rows: []demandforecastports.DriverStart{{
			WorkerID: worker, Name: "Ana Ruiz", Latitude: okc.Latitude, Longitude: okc.Longitude,
			FreeAt: testNow + 3600,
		}}},
		notifier: &fakeNotifier{},
		audit:    auditportsmock.NewMockAuditService(t),
	}
	r.svc = &Service{
		l:      zap.NewNop(),
		repo:   r.repo,
		starts: r.starts,
		forecasts: &fakeForecasts{quality: readyQ, areas: []*demandforecast.Area{
			{Cell: cellOf(t, dallas), Label: "Dallas, TX", MedianRevenueCents: 45_000, Loads: 400},
			{
				Cell:               cellOf(t, okc),
				Label:              "Oklahoma City, OK",
				MedianRevenueCents: 30_000,
				Loads:              200,
			},
		}},
		audit:  r.audit,
		notify: r.notifier,
		clock:  func() int64 { return testNow },
	}
	return r
}

func TestCreate_SendsTheDriverRecordsItAndTellsThem(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	target := slotNow + 4*dispatchmarket.SlotSeconds
	var logged *auditports.LogActionParams
	r.audit.EXPECT().
		LogAction(mock.Anything, mock.Anything).
		RunAndReturn(func(params *auditports.LogActionParams, _ ...auditports.LogOption) error {
			logged = params
			return nil
		}).
		Once()

	positioning, err := r.svc.Create(t.Context(), &driverpositioningports.CreateRequest{
		TenantInfo: tenant,
		Actor:      actor,
		WorkerID:   worker,
		ToCell:     cellOf(t, dallas),
		SlotStart:  target,
	})
	require.NoError(t, err)

	center := dispatchmarket.CenterOf(cellOf(t, dallas))
	miles := geoutils.PlanningRoadMiles(
		okc.Latitude,
		okc.Longitude,
		center.Latitude,
		center.Longitude,
	)
	assert.Equal(
		t,
		[]pulid.ID{worker},
		r.starts.req.WorkerIDs,
		"only the driver being sent is read",
	)
	assert.Equal(t, worker, positioning.WorkerID)
	assert.Equal(t, cellOf(t, okc), positioning.FromCell)
	assert.Equal(t, "Oklahoma City, OK", positioning.FromLabel)
	assert.Equal(t, "Dallas, TX", positioning.ToLabel)
	assert.InDelta(t, center.Latitude, positioning.ToLatitude, 1e-9)
	assert.Equal(t, target, positioning.SlotStart)
	assert.Equal(t, testNow+3600, positioning.FreeAt, "they leave when their last drop is done")
	assert.Equal(t, testNow+3600+geoutils.PlanningDriveSeconds(miles), positioning.ArriveAt)
	assert.Equal(t, driverpositioning.StatusActive, positioning.Status)
	assert.Equal(t, userID, positioning.CreatedByID)
	assert.True(t, positioning.OrganizationID.IsNil(), "keel fills tenant columns on insert")

	require.Len(t, r.notifier.sent, 1)
	sent := r.notifier.sent[0]
	assert.Equal(t, worker, sent.WorkerID)
	assert.Equal(t, "dash.positioning", sent.EventType)
	body, ok := sent.Context.(notificationtemplate.DriverNotificationContext)
	require.True(t, ok)
	assert.Equal(t, "Dallas, TX", body.AreaName)
	assert.False(t, body.Canceled)
	assert.Equal(t, time.Unix(target, 0).UTC().Format(timeLayout), body.FreightFrom)

	require.NotNil(t, logged)
	assert.Equal(t, permission.ResourceDriverPositioning, logged.Resource)
	assert.Equal(t, permission.OpCreate, logged.Operation)
	assert.Equal(t, "dpo_new", logged.ResourceID)
	assert.Equal(t, userID, logged.UserID)
	assert.Nil(t, logged.PreviousState, "nothing was replaced")
}

func TestCreate_AuditsTheReplacedPositioningAsItsOwnRecord(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	r.repo.previous = &driverpositioning.Positioning{
		ID: "dpo_old", WorkerID: worker, ToLabel: "Austin, TX",
		Status: driverpositioning.StatusReplaced,
	}
	r.audit.EXPECT().
		LogAction(mock.MatchedBy(func(params *auditports.LogActionParams) bool {
			return params.ResourceID == "dpo_old" &&
				params.Operation == permission.OpCancel &&
				params.CurrentState["status"] == string(driverpositioning.StatusReplaced) &&
				params.PreviousState == nil
		}), mock.Anything).
		Return(nil).
		Once()
	r.audit.EXPECT().
		LogAction(mock.MatchedBy(func(params *auditports.LogActionParams) bool {
			return params.ResourceID != "dpo_old" &&
				params.Operation == permission.OpCreate &&
				params.PreviousState == nil
		}), mock.Anything).
		Return(nil).
		Once()

	_, err := r.svc.Create(t.Context(), &driverpositioningports.CreateRequest{
		TenantInfo: tenant, Actor: actor, WorkerID: worker,
		ToCell: cellOf(t, dallas), SlotStart: slotNow + dispatchmarket.SlotSeconds,
	})
	require.NoError(t, err)
}

func TestCreate_RefusesWhatCannotBeSent(t *testing.T) {
	t.Parallel()

	denver := dispatchmarket.Point{Latitude: 39.7392, Longitude: -104.9903}
	tests := []struct {
		name   string
		mutate func(r *rig, req *driverpositioningports.CreateRequest)
		field  string
	}{
		{
			name: "a window past the outlook",
			mutate: func(_ *rig, req *driverpositioningports.CreateRequest) {
				req.SlotStart = slotNow + 12*dispatchmarket.SlotSeconds
			},
			field: "slotStart",
		},
		{
			name: "a window that is not a four-hour boundary",
			mutate: func(_ *rig, req *driverpositioningports.CreateRequest) {
				req.SlotStart = slotNow + 3600
			},
			field: "slotStart",
		},
		{
			name:   "no driver",
			mutate: func(_ *rig, req *driverpositioningports.CreateRequest) { req.WorkerID = pulid.Nil },
			field:  "workerId",
		},
		{
			name:   "a driver not available for dispatch",
			mutate: func(r *rig, _ *driverpositioningports.CreateRequest) { r.starts.rows = nil },
		},
		{
			name: "the area the driver is already in",
			mutate: func(_ *rig, req *driverpositioningports.CreateRequest) {
				req.ToCell = cellOf(t, okc)
			},
		},
		{
			name: "an area the forecast does not know",
			mutate: func(_ *rig, req *driverpositioningports.CreateRequest) {
				req.ToCell = cellOf(t, denver)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newRig(t)
			req := &driverpositioningports.CreateRequest{
				TenantInfo: tenant, Actor: actor, WorkerID: worker,
				ToCell: cellOf(t, dallas), SlotStart: slotNow + dispatchmarket.SlotSeconds,
			}
			tt.mutate(r, req)

			_, err := r.svc.Create(t.Context(), req)
			require.Error(t, err)
			if tt.field != "" {
				var multi *errortypes.MultiError
				require.ErrorAs(t, err, &multi)
				fields := []string{}
				for _, fieldErr := range multi.Errors {
					fields = append(fields, fieldErr.Field)
				}
				assert.Contains(t, fields, tt.field, "the message lands on its field")
			} else {
				var business *errortypes.BusinessError
				assert.ErrorAs(t, err, &business)
			}
			assert.Nil(t, r.repo.replaced, "nothing is written")
			assert.Empty(t, r.notifier.sent, "nobody is told")
		})
	}
}

func TestCreate_SaysNothingToTheDriverWhenTheWriteFails(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	r.repo.err = errors.New("database down")

	_, err := r.svc.Create(t.Context(), &driverpositioningports.CreateRequest{
		TenantInfo: tenant, Actor: actor, WorkerID: worker,
		ToCell: cellOf(t, dallas), SlotStart: slotNow + dispatchmarket.SlotSeconds,
	})

	require.ErrorIs(t, err, r.repo.err)
	assert.Empty(t, r.notifier.sent)
}

func TestCallOff_CancelsItTellsTheDriverAndAuditsTheChange(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	r.repo.current = &driverpositioning.Positioning{
		ID: "dpo_live", WorkerID: worker, ToLabel: "Dallas, TX",
		SlotStart: slotNow, ArriveAt: testNow + 4*3600, Status: driverpositioning.StatusActive,
	}
	r.audit.EXPECT().
		LogAction(mock.MatchedBy(func(params *auditports.LogActionParams) bool {
			return params.Operation == permission.OpCancel &&
				params.Resource == permission.ResourceDriverPositioning &&
				params.PreviousState["status"] == string(driverpositioning.StatusActive) &&
				params.CurrentState["status"] == string(driverpositioning.StatusCanceled)
		}), mock.Anything).
		Return(nil).
		Once()

	positioning, err := r.svc.CallOff(t.Context(), &driverpositioningports.CallOffRequest{
		TenantInfo: tenant, Actor: actor, ID: "dpo_live",
	})
	require.NoError(t, err)

	assert.Equal(t, driverpositioning.StatusCanceled, positioning.Status)
	require.NotNil(t, r.repo.canceled)
	assert.Equal(t, userID, r.repo.canceled.CanceledByID)
	assert.Equal(t, testNow, r.repo.canceled.CanceledAt)
	require.Len(t, r.notifier.sent, 1)
	body, ok := r.notifier.sent[0].Context.(notificationtemplate.DriverNotificationContext)
	require.True(t, ok)
	assert.True(t, body.Canceled, "the driver is told it is off")
}
