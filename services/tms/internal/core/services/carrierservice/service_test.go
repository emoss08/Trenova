package carrierservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testDeps struct {
	repo  *mocks.MockCarrierRepository
	audit *mocks.MockAuditService
	svc   *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := mocks.NewMockCarrierRepository(t)
	auditSvc := mocks.NewMockAuditService(t)
	svc := &Service{
		l:            zap.NewNop(),
		repo:         repo,
		validator:    NewTestValidator(),
		auditService: auditSvc,
		realtime:     &mocks.NoopRealtimeService{},
	}
	return &testDeps{repo: repo, audit: auditSvc, svc: svc}
}

func newTestEntity() *carrier.Carrier {
	return &carrier.Carrier{
		ID:               pulid.MustNew("car_"),
		BusinessUnitID:   pulid.MustNew("bu_"),
		OrganizationID:   pulid.MustNew("org_"),
		Status:           carrier.StatusActive,
		Code:             "CARR001",
		Name:             "Test Carrier",
		CarrierType:      carrier.TypeCommon,
		ComplianceStatus: carrier.ComplianceStatusQualified,
		SafetyRating:     carrier.SafetyRatingSatisfactory,
		PaymentMethod:    carrier.PaymentMethodCheck,
		PaymentTermDays:  30,
		Version:          1,
	}
}

func newCreateEntity() *carrier.Carrier {
	entity := newTestEntity()
	entity.ID = ""
	entity.Version = 0
	return entity
}

func TestServiceCreate(t *testing.T) {
	t.Run("returns validation error for missing required fields", func(t *testing.T) {
		deps := setupTest(t)

		entity := newCreateEntity()
		entity.Code = ""
		entity.Name = ""

		created, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(pulid.MustNew("usr_"), entity.OrganizationID, entity.BusinessUnitID),
		)
		require.Error(t, err)
		assert.Nil(t, created)
	})

	t.Run("creates carrier and logs audit entry", func(t *testing.T) {
		deps := setupTest(t)

		entity := newCreateEntity()
		deps.repo.EXPECT().
			Create(mock.Anything, entity).
			RunAndReturn(func(_ context.Context, e *carrier.Carrier) (*carrier.Carrier, error) {
				e.ID = pulid.MustNew("car_")
				return e, nil
			})
		deps.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil)

		created, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(pulid.MustNew("usr_"), entity.OrganizationID, entity.BusinessUnitID),
		)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.False(t, created.ID.IsNil())
	})

	t.Run("propagates repository error", func(t *testing.T) {
		deps := setupTest(t)

		entity := newCreateEntity()
		repoErr := errors.New("insert failed")
		deps.repo.EXPECT().Create(mock.Anything, entity).Return(nil, repoErr)

		created, err := deps.svc.Create(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(pulid.MustNew("usr_"), entity.OrganizationID, entity.BusinessUnitID),
		)
		require.ErrorIs(t, err, repoErr)
		assert.Nil(t, created)
	})
}

func TestServiceUpdate(t *testing.T) {
	t.Run("updates carrier and logs audit entry with diff", func(t *testing.T) {
		deps := setupTest(t)

		entity := newTestEntity()
		original := *entity
		original.Name = "Old Name"

		deps.repo.EXPECT().
			GetByID(mock.Anything, mock.MatchedBy(func(req repositories.GetCarrierByIDRequest) bool {
				return req.ID == entity.ID
			})).
			Return(&original, nil)
		deps.repo.EXPECT().Update(mock.Anything, entity).Return(entity, nil)
		deps.audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		updated, err := deps.svc.Update(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(pulid.MustNew("usr_"), entity.OrganizationID, entity.BusinessUnitID),
		)
		require.NoError(t, err)
		assert.Equal(t, entity.Name, updated.Name)
	})

	t.Run("fails when original cannot be loaded", func(t *testing.T) {
		deps := setupTest(t)

		entity := newTestEntity()
		repoErr := errors.New("not found")
		deps.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, repoErr)

		updated, err := deps.svc.Update(
			t.Context(),
			entity,
			internaltestutil.NewSessionActor(pulid.MustNew("usr_"), entity.OrganizationID, entity.BusinessUnitID),
		)
		require.ErrorIs(t, err, repoErr)
		assert.Nil(t, updated)
	})
}

func TestServiceBulkUpdateStatus(t *testing.T) {
	deps := setupTest(t)

	entity := newTestEntity()
	req := &repositories.BulkUpdateCarrierStatusRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  entity.OrganizationID,
			BuID:   entity.BusinessUnitID,
			UserID: pulid.MustNew("usr_"),
		},
		CarrierIDs: []pulid.ID{entity.ID},
		Status:     carrier.StatusInactive,
	}

	deps.repo.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return(
		[]*carrier.Carrier{entity}, nil,
	)
	updatedEntity := *entity
	updatedEntity.Status = carrier.StatusInactive
	deps.repo.EXPECT().BulkUpdateStatus(mock.Anything, req).Return(
		[]*carrier.Carrier{&updatedEntity}, nil,
	)
	deps.audit.EXPECT().LogActions(mock.Anything).Return(nil)

	entities, err := deps.svc.BulkUpdateStatus(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, entities, 1)
	assert.Equal(t, carrier.StatusInactive, entities[0].Status)
}

func TestServiceGet(t *testing.T) {
	deps := setupTest(t)

	entity := newTestEntity()
	req := repositories.GetCarrierByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}
	deps.repo.EXPECT().GetByID(mock.Anything, req).Return(entity, nil)

	got, err := deps.svc.Get(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, entity.ID, got.ID)
}
