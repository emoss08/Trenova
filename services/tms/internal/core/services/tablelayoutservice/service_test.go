package tablelayoutservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tablelayout"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestService(t *testing.T) (*Service, *mocks.MockTableLayoutRepository) {
	t.Helper()

	repo := mocks.NewMockTableLayoutRepository(t)
	return New(Params{Logger: zap.NewNop(), Repo: repo}), repo
}

func testRequest() Request {
	return Request{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		Resource: "shipment",
	}
}

func TestService_Get_ReturnsNilWhenThePersonHasNoLayout(t *testing.T) {
	svc, repo := newTestService(t)
	req := testRequest()

	repo.EXPECT().Get(mock.Anything, &repositories.TableLayoutRequest{
		TenantInfo: req.TenantInfo,
		Resource:   "shipment",
	}).Return(nil, false, nil)

	got, err := svc.Get(t.Context(), &req)

	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestService_Get_RefusesACallerWithNoUser(t *testing.T) {
	svc, _ := newTestService(t)
	req := testRequest()
	req.TenantInfo.UserID = pulid.Nil

	_, err := svc.Get(t.Context(), &req)

	var authErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authErr)
}

func TestService_Save_UpsertsTheCallersLayout(t *testing.T) {
	svc, repo := newTestService(t)
	req := testRequest()
	layout := &tablelayout.Layout{
		ColumnOrder:  []string{"status", "proNumber"},
		ColumnSizing: map[string]float64{"status": 120},
		Density:      tablelayout.DensityCompact,
	}

	repo.EXPECT().Get(mock.Anything, mock.Anything).Return(&tablelayout.TableLayout{}, true, nil)
	repo.EXPECT().
		Upsert(mock.Anything, mock.MatchedBy(func(entity *tablelayout.TableLayout) bool {
			return entity.UserID == req.TenantInfo.UserID &&
				entity.OrganizationID == req.TenantInfo.OrgID &&
				entity.BusinessUnitID == req.TenantInfo.BuID &&
				entity.Resource == "shipment" &&
				entity.Layout == layout
		})).
		RunAndReturn(func(_ context.Context, entity *tablelayout.TableLayout) (*tablelayout.TableLayout, error) {
			entity.Version = 3
			return entity, nil
		})

	saved, err := svc.Save(t.Context(), &SaveRequest{Request: req, Layout: layout})

	require.NoError(t, err)
	assert.Equal(t, int64(3), saved.Version)
}

func TestService_Save_RejectsAnInvalidLayoutBeforeTouchingTheStore(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Save(t.Context(), &SaveRequest{
		Request: testRequest(),
		Layout:  &tablelayout.Layout{ColumnSizing: map[string]float64{"status": 1}},
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestService_Save_RefusesANewTableOnceThePersonIsAtTheCap(t *testing.T) {
	svc, repo := newTestService(t)
	req := testRequest()

	repo.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, false, nil)
	repo.EXPECT().CountForUser(mock.Anything, req.TenantInfo).Return(tablelayout.MaxLayoutsPerUser, nil)

	_, err := svc.Save(t.Context(), &SaveRequest{Request: req, Layout: &tablelayout.Layout{}})

	var businessErr *errortypes.BusinessError
	require.ErrorAs(t, err, &businessErr)
}

func TestService_Save_LetsAPersonAtTheCapOverwriteATableTheyAlreadyHave(t *testing.T) {
	svc, repo := newTestService(t)
	req := testRequest()

	repo.EXPECT().Get(mock.Anything, mock.Anything).Return(&tablelayout.TableLayout{}, true, nil)
	repo.EXPECT().
		Upsert(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *tablelayout.TableLayout) (*tablelayout.TableLayout, error) {
			return entity, nil
		})

	_, err := svc.Save(t.Context(), &SaveRequest{Request: req, Layout: &tablelayout.Layout{}})

	require.NoError(t, err)
	repo.AssertNotCalled(t, "CountForUser", mock.Anything, mock.Anything)
}

func TestService_Save_PassesAStoreFailureThrough(t *testing.T) {
	svc, repo := newTestService(t)
	boom := errors.New("boom")

	repo.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, false, boom)

	_, err := svc.Save(t.Context(), &SaveRequest{Request: testRequest(), Layout: &tablelayout.Layout{}})

	require.ErrorIs(t, err, boom)
}

func TestService_Reset_DeletesOnlyTheCallersLayoutForTheTable(t *testing.T) {
	svc, repo := newTestService(t)
	req := testRequest()

	repo.EXPECT().Delete(mock.Anything, &repositories.TableLayoutRequest{
		TenantInfo: req.TenantInfo,
		Resource:   "shipment",
	}).Return(nil)

	require.NoError(t, svc.Reset(t.Context(), &req))
}

func TestService_Save_KeepsALayoutLeftAtEveryDefault(t *testing.T) {
	svc, repo := newTestService(t)
	req := testRequest()

	repo.EXPECT().Get(mock.Anything, mock.Anything).Return(&tablelayout.TableLayout{}, true, nil)
	repo.EXPECT().
		Upsert(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *tablelayout.TableLayout) (*tablelayout.TableLayout, error) {
			return entity, nil
		})

	_, err := svc.Save(t.Context(), &SaveRequest{Request: req, Layout: &tablelayout.Layout{}})

	require.NoError(t, err)
}

func TestService_Save_RefusesAMissingLayout(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Save(t.Context(), &SaveRequest{Request: testRequest(), Layout: nil})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Contains(t, multiErr.Error(), "Layout is required")
}
