package tractorservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func newLocateTestService(
	repo repositories.TractorRepository,
	assignmentRepo repositories.AssignmentRepository,
	continuityRepo repositories.EquipmentContinuityRepository,
	locationRepo repositories.LocationRepository,
) *Service {
	return &Service{
		l:              zap.NewNop(),
		db:             tractorTestDBConnection{},
		repo:           repo,
		assignmentRepo: assignmentRepo,
		continuityRepo: continuityRepo,
		locationRepo:   locationRepo,
	}
}

func TestLocate_CreatesFirstContinuityRow(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	tractorID := pulid.MustNew("trac_")
	locationID := pulid.MustNew("loc_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	repo := mocks.NewMockTractorRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, repositories.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: tenantInfo,
		}).
		Return(&tractor.Tractor{ID: tractorID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	locationRepo := mocks.NewMockLocationRepository(t)
	locationRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetLocationByIDRequest{
			ID:         locationID,
			TenantInfo: tenantInfo,
		}).
		Return(&location.Location{ID: locationID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	assignmentRepo := mocks.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, tractorID, pulid.Nil).
		Return(nil, nil).
		Once()

	continuityRepo := mocks.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		GetEffectiveCurrent(mock.Anything, repositories.GetCurrentEquipmentContinuityRequest{
			TenantInfo:    tenantInfo,
			EquipmentType: equipmentcontinuity.EquipmentTypeTractor,
			EquipmentID:   tractorID,
		}).
		Return(nil, nil).
		Once()
	continuityRepo.EXPECT().
		Advance(mock.Anything, mock.MatchedBy(func(req repositories.CreateEquipmentContinuityRequest) bool {
			return req.TenantInfo == tenantInfo &&
				req.EquipmentType == equipmentcontinuity.EquipmentTypeTractor &&
				req.EquipmentID == tractorID &&
				req.CurrentLocationID == locationID &&
				req.SourceType == equipmentcontinuity.SourceTypeManualLocate &&
				req.SourceShipmentID.IsNil() &&
				req.SourceShipmentMoveID.IsNil() &&
				req.SourceAssignmentID.IsNil()
		})).
		Return(&equipmentcontinuity.EquipmentContinuity{
			ID:                pulid.MustNew("eqc_"),
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			EquipmentType:     equipmentcontinuity.EquipmentTypeTractor,
			EquipmentID:       tractorID,
			CurrentLocationID: locationID,
			SourceType:        equipmentcontinuity.SourceTypeManualLocate,
			IsCurrent:         true,
		}, nil).
		Once()

	svc := newLocateTestService(repo, assignmentRepo, continuityRepo, locationRepo)
	result, err := svc.Locate(t.Context(), &repositories.LocateTractorRequest{
		TenantInfo:    tenantInfo,
		TractorID:     tractorID,
		NewLocationID: locationID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, locationID, result.CurrentLocationID)
}

func TestLocate_ReplacesCurrentContinuity(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	tractorID := pulid.MustNew("trac_")
	currentLocationID := pulid.MustNew("loc_")
	newLocationID := pulid.MustNew("loc_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}

	repo := mocks.NewMockTractorRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, repositories.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: tenantInfo,
		}).
		Return(&tractor.Tractor{ID: tractorID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	locationRepo := mocks.NewMockLocationRepository(t)
	locationRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetLocationByIDRequest{
			ID:         newLocationID,
			TenantInfo: tenantInfo,
		}).
		Return(&location.Location{ID: newLocationID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	assignmentRepo := mocks.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, tractorID, pulid.Nil).
		Return(nil, nil).
		Once()

	continuityRepo := mocks.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		GetEffectiveCurrent(mock.Anything, repositories.GetCurrentEquipmentContinuityRequest{
			TenantInfo:    tenantInfo,
			EquipmentType: equipmentcontinuity.EquipmentTypeTractor,
			EquipmentID:   tractorID,
		}).
		Return(&equipmentcontinuity.EquipmentContinuity{
			ID:                pulid.MustNew("eqc_"),
			CurrentLocationID: currentLocationID,
		}, nil).
		Once()
	continuityRepo.EXPECT().
		Advance(mock.Anything, mock.MatchedBy(func(req repositories.CreateEquipmentContinuityRequest) bool {
			return req.EquipmentID == tractorID &&
				req.CurrentLocationID == newLocationID &&
				req.SourceType == equipmentcontinuity.SourceTypeManualLocate
		})).
		Return(&equipmentcontinuity.EquipmentContinuity{
			ID:                pulid.MustNew("eqc_"),
			CurrentLocationID: newLocationID,
		}, nil).
		Once()

	svc := newLocateTestService(repo, assignmentRepo, continuityRepo, locationRepo)
	result, err := svc.Locate(t.Context(), &repositories.LocateTractorRequest{
		TenantInfo:    tenantInfo,
		TractorID:     tractorID,
		NewLocationID: newLocationID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, newLocationID, result.CurrentLocationID)
}

func TestLocate_RejectsSameLocation(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	tractorID := pulid.MustNew("trac_")
	locationID := pulid.MustNew("loc_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}

	repo := mocks.NewMockTractorRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, repositories.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: tenantInfo,
		}).
		Return(&tractor.Tractor{ID: tractorID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	locationRepo := mocks.NewMockLocationRepository(t)
	locationRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetLocationByIDRequest{
			ID:         locationID,
			TenantInfo: tenantInfo,
		}).
		Return(&location.Location{ID: locationID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	assignmentRepo := mocks.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, tractorID, pulid.Nil).
		Return(nil, nil).
		Once()

	continuityRepo := mocks.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		GetEffectiveCurrent(mock.Anything, repositories.GetCurrentEquipmentContinuityRequest{
			TenantInfo:    tenantInfo,
			EquipmentType: equipmentcontinuity.EquipmentTypeTractor,
			EquipmentID:   tractorID,
		}).
		Return(&equipmentcontinuity.EquipmentContinuity{CurrentLocationID: locationID}, nil).
		Once()

	svc := newLocateTestService(repo, assignmentRepo, continuityRepo, locationRepo)
	result, err := svc.Locate(t.Context(), &repositories.LocateTractorRequest{
		TenantInfo:    tenantInfo,
		TractorID:     tractorID,
		NewLocationID: locationID,
	})

	require.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, "Tractor is already located at the requested location", err.Error())
	continuityRepo.AssertNotCalled(t, "Advance")
}

func TestLocate_RejectsTractorAlreadyInProgress(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	tractorID := pulid.MustNew("trac_")
	locationID := pulid.MustNew("loc_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}

	repo := mocks.NewMockTractorRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, repositories.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: tenantInfo,
		}).
		Return(&tractor.Tractor{ID: tractorID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	locationRepo := mocks.NewMockLocationRepository(t)
	locationRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetLocationByIDRequest{
			ID:         locationID,
			TenantInfo: tenantInfo,
		}).
		Return(&location.Location{ID: locationID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	moveID := pulid.MustNew("sm_")
	assignmentRepo := mocks.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, tractorID, pulid.Nil).
		Return(&shipment.Assignment{
			ID:             pulid.MustNew("asn_"),
			ShipmentMoveID: moveID,
			TractorID:      &tractorID,
		}, nil).
		Once()

	continuityRepo := mocks.NewMockEquipmentContinuityRepository(t)
	svc := newLocateTestService(repo, assignmentRepo, continuityRepo, locationRepo)
	result, err := svc.Locate(t.Context(), &repositories.LocateTractorRequest{
		TenantInfo:    tenantInfo,
		TractorID:     tractorID,
		NewLocationID: locationID,
	})

	require.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, "Tractor is currently in progress on another move", err.Error())
	continuityRepo.AssertNotCalled(t, "GetEffectiveCurrent")
	continuityRepo.AssertNotCalled(t, "Advance")
}

func TestLocate_PropagatesMissingTractor(t *testing.T) {
	t.Parallel()

	tractorID := pulid.MustNew("trac_")
	locationID := pulid.MustNew("loc_")
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	tractorErr := errors.New("tractor not found")

	repo := mocks.NewMockTractorRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, repositories.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: tenantInfo,
		}).
		Return(nil, tractorErr).
		Once()

	locationRepo := mocks.NewMockLocationRepository(t)
	assignmentRepo := mocks.NewMockAssignmentRepository(t)
	continuityRepo := mocks.NewMockEquipmentContinuityRepository(t)
	svc := newLocateTestService(repo, assignmentRepo, continuityRepo, locationRepo)
	result, err := svc.Locate(t.Context(), &repositories.LocateTractorRequest{
		TenantInfo:    tenantInfo,
		TractorID:     tractorID,
		NewLocationID: locationID,
	})

	require.Nil(t, result)
	assert.ErrorIs(t, err, tractorErr)
	locationRepo.AssertNotCalled(t, "GetByID")
	assignmentRepo.AssertNotCalled(t, "FindInProgressByTractorID")
	continuityRepo.AssertNotCalled(t, "Advance")
}

func TestLocate_PropagatesMissingLocation(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	tractorID := pulid.MustNew("trac_")
	locationID := pulid.MustNew("loc_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	locationErr := errors.New("location not found")

	repo := mocks.NewMockTractorRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, repositories.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: tenantInfo,
		}).
		Return(&tractor.Tractor{ID: tractorID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	locationRepo := mocks.NewMockLocationRepository(t)
	locationRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetLocationByIDRequest{
			ID:         locationID,
			TenantInfo: tenantInfo,
		}).
		Return(nil, locationErr).
		Once()

	assignmentRepo := mocks.NewMockAssignmentRepository(t)
	continuityRepo := mocks.NewMockEquipmentContinuityRepository(t)
	svc := newLocateTestService(repo, assignmentRepo, continuityRepo, locationRepo)
	result, err := svc.Locate(t.Context(), &repositories.LocateTractorRequest{
		TenantInfo:    tenantInfo,
		TractorID:     tractorID,
		NewLocationID: locationID,
	})

	require.Nil(t, result)
	assert.ErrorIs(t, err, locationErr)
	assignmentRepo.AssertNotCalled(t, "FindInProgressByTractorID")
	continuityRepo.AssertNotCalled(t, "Advance")
}

type tractorTestDBConnection struct{}

func (tractorTestDBConnection) DB() *bun.DB                          { return nil }
func (tractorTestDBConnection) DBForContext(context.Context) bun.IDB { return nil }
func (tractorTestDBConnection) HealthCheck(context.Context) error    { return nil }
func (tractorTestDBConnection) IsHealthy(context.Context) bool       { return true }
func (tractorTestDBConnection) Close() error                         { return nil }
func (tractorTestDBConnection) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}
