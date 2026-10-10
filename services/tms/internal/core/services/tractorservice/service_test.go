package tractorservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/equipmentcontinuityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/locationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tractorportsmock"
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
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
	repo tractorports.TractorRepository,
	assignmentRepo shipmentports.AssignmentRepository,
	continuityRepo equipmentcontinuityports.EquipmentContinuityRepository,
	locationRepo locationports.LocationRepository,
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

	repo := tractorportsmock.NewMockTractorRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, tractorports.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: tenantInfo,
		}).
		Return(&tractor.Tractor{ID: tractorID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	locationRepo := locationportsmock.NewMockLocationRepository(t)
	locationRepo.EXPECT().
		GetByID(mock.Anything, locationports.GetLocationByIDRequest{
			ID:         locationID,
			TenantInfo: tenantInfo,
		}).
		Return(&location.Location{ID: locationID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, tractorID, pulid.Nil).
		Return(nil, nil).
		Once()

	continuityRepo := equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		GetEffectiveCurrent(mock.Anything, equipmentcontinuityports.GetCurrentEquipmentContinuityRequest{
			TenantInfo:    tenantInfo,
			EquipmentType: equipmentcontinuity.EquipmentTypeTractor,
			EquipmentID:   tractorID,
		}).
		Return(nil, nil).
		Once()
	continuityRepo.EXPECT().
		Advance(mock.Anything, mock.MatchedBy(func(req equipmentcontinuityports.CreateEquipmentContinuityRequest) bool {
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
	result, err := svc.Locate(t.Context(), &tractorports.LocateTractorRequest{
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

	repo := tractorportsmock.NewMockTractorRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, tractorports.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: tenantInfo,
		}).
		Return(&tractor.Tractor{ID: tractorID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	locationRepo := locationportsmock.NewMockLocationRepository(t)
	locationRepo.EXPECT().
		GetByID(mock.Anything, locationports.GetLocationByIDRequest{
			ID:         newLocationID,
			TenantInfo: tenantInfo,
		}).
		Return(&location.Location{ID: newLocationID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, tractorID, pulid.Nil).
		Return(nil, nil).
		Once()

	continuityRepo := equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		GetEffectiveCurrent(mock.Anything, equipmentcontinuityports.GetCurrentEquipmentContinuityRequest{
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
		Advance(mock.Anything, mock.MatchedBy(func(req equipmentcontinuityports.CreateEquipmentContinuityRequest) bool {
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
	result, err := svc.Locate(t.Context(), &tractorports.LocateTractorRequest{
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

	repo := tractorportsmock.NewMockTractorRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, tractorports.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: tenantInfo,
		}).
		Return(&tractor.Tractor{ID: tractorID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	locationRepo := locationportsmock.NewMockLocationRepository(t)
	locationRepo.EXPECT().
		GetByID(mock.Anything, locationports.GetLocationByIDRequest{
			ID:         locationID,
			TenantInfo: tenantInfo,
		}).
		Return(&location.Location{ID: locationID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, tractorID, pulid.Nil).
		Return(nil, nil).
		Once()

	continuityRepo := equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		GetEffectiveCurrent(mock.Anything, equipmentcontinuityports.GetCurrentEquipmentContinuityRequest{
			TenantInfo:    tenantInfo,
			EquipmentType: equipmentcontinuity.EquipmentTypeTractor,
			EquipmentID:   tractorID,
		}).
		Return(&equipmentcontinuity.EquipmentContinuity{CurrentLocationID: locationID}, nil).
		Once()

	svc := newLocateTestService(repo, assignmentRepo, continuityRepo, locationRepo)
	result, err := svc.Locate(t.Context(), &tractorports.LocateTractorRequest{
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

	repo := tractorportsmock.NewMockTractorRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, tractorports.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: tenantInfo,
		}).
		Return(&tractor.Tractor{ID: tractorID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	locationRepo := locationportsmock.NewMockLocationRepository(t)
	locationRepo.EXPECT().
		GetByID(mock.Anything, locationports.GetLocationByIDRequest{
			ID:         locationID,
			TenantInfo: tenantInfo,
		}).
		Return(&location.Location{ID: locationID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	moveID := pulid.MustNew("sm_")
	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, tractorID, pulid.Nil).
		Return(&shipment.Assignment{
			ID:             pulid.MustNew("asn_"),
			ShipmentMoveID: moveID,
			TractorID:      &tractorID,
		}, nil).
		Once()

	continuityRepo := equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t)
	svc := newLocateTestService(repo, assignmentRepo, continuityRepo, locationRepo)
	result, err := svc.Locate(t.Context(), &tractorports.LocateTractorRequest{
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

	repo := tractorportsmock.NewMockTractorRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, tractorports.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: tenantInfo,
		}).
		Return(nil, tractorErr).
		Once()

	locationRepo := locationportsmock.NewMockLocationRepository(t)
	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	continuityRepo := equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t)
	svc := newLocateTestService(repo, assignmentRepo, continuityRepo, locationRepo)
	result, err := svc.Locate(t.Context(), &tractorports.LocateTractorRequest{
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

	repo := tractorportsmock.NewMockTractorRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, tractorports.GetTractorByIDRequest{
			ID:         tractorID,
			TenantInfo: tenantInfo,
		}).
		Return(&tractor.Tractor{ID: tractorID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	locationRepo := locationportsmock.NewMockLocationRepository(t)
	locationRepo.EXPECT().
		GetByID(mock.Anything, locationports.GetLocationByIDRequest{
			ID:         locationID,
			TenantInfo: tenantInfo,
		}).
		Return(nil, locationErr).
		Once()

	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	continuityRepo := equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t)
	svc := newLocateTestService(repo, assignmentRepo, continuityRepo, locationRepo)
	result, err := svc.Locate(t.Context(), &tractorports.LocateTractorRequest{
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
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}
