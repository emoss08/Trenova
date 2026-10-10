package trailerservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/customfieldportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/equipmentcontinuityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/locationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/trailerportsmock"
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/workspace/app/customfieldservice"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*trailer.Trailer]().
			WithModelName("Trailer").
			Build(),
	}
}

type testDeps struct {
	repo      *trailerportsmock.MockTrailerRepository
	userRepo  *tenantportsmock.MockUserRepository
	audit     *auditportsmock.MockAuditService
	valueRepo *customfieldportsmock.MockCustomFieldValueRepository
	defRepo   *customfieldportsmock.MockCustomFieldDefinitionRepository
	svc       *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := trailerportsmock.NewMockTrailerRepository(t)
	userRepo := tenantportsmock.NewMockUserRepository(t)
	auditSvc := auditportsmock.NewMockAuditService(t)
	valueRepo := customfieldportsmock.NewMockCustomFieldValueRepository(t)
	defRepo := customfieldportsmock.NewMockCustomFieldDefinitionRepository(t)

	logger := zap.NewNop()

	valuesValidator := customfieldservice.NewValuesValidator(
		customfieldservice.ValuesValidatorParams{
			Logger: logger,
			Repo:   defRepo,
		},
	)

	cfService := customfieldservice.NewValuesService(customfieldservice.ValuesServiceParams{
		Logger:         logger,
		ValueRepo:      valueRepo,
		DefinitionRepo: defRepo,
		Validator:      valuesValidator,
	})

	svc := &Service{
		l:                         logger,
		repo:                      repo,
		userRepo:                  userRepo,
		validator:                 newTestValidator(),
		auditService:              auditSvc,
		realtime:                  &mocks.NoopRealtimeService{},
		customFieldsValuesService: cfService,
	}
	return &testDeps{
		repo:      repo,
		userRepo:  userRepo,
		audit:     auditSvc,
		valueRepo: valueRepo,
		defRepo:   defRepo,
		svc:       svc,
	}
}

func newTestEntity() *trailer.Trailer {
	year := 2020
	return &trailer.Trailer{
		ID:                      pulid.MustNew("tr_"),
		BusinessUnitID:          pulid.MustNew("bu_"),
		OrganizationID:          pulid.MustNew("org_"),
		EquipmentTypeID:         pulid.MustNew("et_"),
		EquipmentManufacturerID: pulid.MustNew("em_"),
		RegistrationStateID:     pulid.MustNew("st_"),
		Status:                  domaintypes.EquipmentStatusAvailable,
		Code:                    "TRL001",
		Year:                    &year,
		Vin:                     "1HGBH41JXMN109186",
		Version:                 1,
	}
}

func newCreateEntity() *trailer.Trailer {
	year := 2020
	return &trailer.Trailer{
		BusinessUnitID:          pulid.MustNew("bu_"),
		OrganizationID:          pulid.MustNew("org_"),
		EquipmentTypeID:         pulid.MustNew("et_"),
		EquipmentManufacturerID: pulid.MustNew("em_"),
		RegistrationStateID:     pulid.MustNew("st_"),
		Status:                  domaintypes.EquipmentStatusAvailable,
		Code:                    "TRL001",
		Year:                    &year,
		Vin:                     "1HGBH41JXMN109186",
	}
}

func TestList_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	entity := newTestEntity()
	expected := &pagination.CursorListResult[*trailer.Trailer]{
		Items:       []*trailer.Trailer{entity},
		HasNextPage: false,
	}
	req := &trailerports.ListTrailersRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
		TrailerRelationIncludes: trailerports.TrailerRelationIncludes{
			IncludeCustomFields: true,
		},
	}

	deps.repo.On("List", mock.Anything, req).Return(expected, nil)
	deps.valueRepo.On("GetByResources", mock.Anything, mock.Anything).
		Return(make(map[string][]*customfield.CustomFieldValue), nil)

	result, err := deps.svc.List(ctx, req)

	require.NoError(t, err)
	assert.Len(t, result.Items, 1)
	assert.False(t, result.HasNextPage)
	deps.repo.AssertExpectations(t)
	deps.valueRepo.AssertExpectations(t)
}

func TestList_EmptyResult(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.CursorListResult[*trailer.Trailer]{
		Items:       []*trailer.Trailer{},
		HasNextPage: false,
	}
	req := &trailerports.ListTrailersRequest{
		Filter: &pagination.QueryOptions{},
	}

	deps.repo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.List(ctx, req)

	require.NoError(t, err)
	assert.Empty(t, result.Items)
	assert.False(t, result.HasNextPage)
	deps.repo.AssertExpectations(t)
	deps.valueRepo.AssertNotCalled(t, "GetByResources")
}

func TestGet_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()

	req := trailerports.GetTrailerByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		TrailerRelationIncludes: trailerports.TrailerRelationIncludes{
			IncludeCustomFields: true,
		},
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)
	deps.valueRepo.On("GetByResource", mock.Anything, mock.Anything).
		Return([]*customfield.CustomFieldValue{}, nil)

	result, err := deps.svc.Get(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	assert.Equal(t, entity.Code, result.Code)
	deps.repo.AssertExpectations(t)
	deps.valueRepo.AssertExpectations(t)
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newCreateEntity()
	userID := pulid.MustNew("usr_")

	created := newTestEntity()
	created.BusinessUnitID = entity.BusinessUnitID
	created.OrganizationID = entity.OrganizationID

	deps.repo.On("Create", mock.Anything, mock.Anything).Return(created, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	assert.Equal(t, created.ID, result.ID)
	assert.Equal(t, created.Code, result.Code)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestCreate_ValidationError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")

	entity := &trailer.Trailer{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.EquipmentStatusAvailable,
	}

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestCreate_RepoError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newCreateEntity()
	userID := pulid.MustNew("usr_")
	repoErr := errors.New("database error")

	deps.repo.On("Create", mock.Anything, mock.Anything).Return(nil, repoErr)

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, repoErr, err)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertNotCalled(t, "LogAction")
}

func TestLocate_RejectsBrandNewTrailer(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	trailerID := pulid.MustNew("tr_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}

	continuityRepo := equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		GetEffectiveCurrent(mock.Anything, equipmentcontinuityports.GetCurrentEquipmentContinuityRequest{
			TenantInfo:    tenantInfo,
			EquipmentType: equipmentcontinuity.EquipmentTypeTrailer,
			EquipmentID:   trailerID,
		}).
		Return(nil, nil).
		Once()

	deps.repo.EXPECT().
		GetByID(mock.Anything, trailerports.GetTrailerByIDRequest{
			ID:         trailerID,
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&trailer.Trailer{ID: trailerID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTrailerID(mock.Anything, tenantInfo, trailerID, pulid.Nil).
		Return(nil, nil).
		Once()

	deps.svc.db = dbtest.NopConnection{}
	deps.svc.assignmentRepo = assignmentRepo
	deps.svc.continuityRepo = continuityRepo
	deps.svc.shipmentRepo = shipmentportsmock.NewMockShipmentRepository(t)
	deps.svc.shipmentCommentRepo = shipmentportsmock.NewMockShipmentCommentRepository(t)
	deps.svc.controlRepo = tenantportsmock.NewMockShipmentControlRepository(t)
	deps.svc.locationRepo = locationportsmock.NewMockLocationRepository(t)
	deps.svc.shipmentValidator = newTestShipmentValidator(t)
	deps.svc.coordinator = shipmentstate.NewCoordinator()
	deps.svc.commercial = shipmentcommercial.New(shipmentcommercial.Params{
		Logger:          zap.NewNop(),
		RateEngine:      ratingtest.NewFallbackEngine(t, formulatemplateportsmock.NewMockFormulaCalculator(t)),
		AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
	})

	result, err := deps.svc.Locate(t.Context(), &trailerports.LocateTrailerRequest{
		TenantInfo:    tenantInfo,
		TrailerID:     trailerID,
		NewLocationID: pulid.MustNew("loc_"),
	}, internaltestutil.NewSessionActor(pulid.MustNew("usr_"), orgID, buID))

	require.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestLocate_AppendsMoveAndAdvancesContinuity(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	trailerID := pulid.MustNew("tr_")
	shipmentID := pulid.MustNew("shp_")
	sourceMoveID := pulid.MustNew("sm_")
	newLocationID := pulid.MustNew("loc_new_")
	currentLocationID := pulid.MustNew("loc_current_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
	trailerEntity := newTestEntity()
	trailerEntity.ID = trailerID
	trailerEntity.OrganizationID = orgID
	trailerEntity.BusinessUnitID = buID
	trailerEntity.Code = "TRL-100"

	deps.repo.EXPECT().
		GetByID(mock.Anything, trailerports.GetTrailerByIDRequest{
			ID: trailerID,
			TenantInfo: pagination.TenantInfo{
				OrgID: orgID,
				BuID:  buID,
			},
		}).
		Return(trailerEntity, nil).
		Once()

	continuityRepo := equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		GetEffectiveCurrent(mock.Anything, equipmentcontinuityports.GetCurrentEquipmentContinuityRequest{
			TenantInfo:    tenantInfo,
			EquipmentType: equipmentcontinuity.EquipmentTypeTrailer,
			EquipmentID:   trailerID,
		}).
		Return(&equipmentcontinuity.EquipmentContinuity{
			ID:                   pulid.MustNew("eqc_"),
			OrganizationID:       orgID,
			BusinessUnitID:       buID,
			EquipmentType:        equipmentcontinuity.EquipmentTypeTrailer,
			EquipmentID:          trailerID,
			CurrentLocationID:    currentLocationID,
			SourceShipmentID:     shipmentID,
			SourceShipmentMoveID: sourceMoveID,
		}, nil).
		Once()
	continuityRepo.EXPECT().
		Advance(mock.Anything, mock.MatchedBy(func(req equipmentcontinuityports.CreateEquipmentContinuityRequest) bool {
			return req.EquipmentID == trailerID &&
				req.CurrentLocationID == newLocationID &&
				req.SourceType == equipmentcontinuity.SourceTypeManualLocate &&
				req.SourceShipmentID == shipmentID &&
				req.SourceAssignmentID != pulid.Nil
		})).
		Return(&equipmentcontinuity.EquipmentContinuity{ID: pulid.MustNew("eqc_")}, nil).
		Once()

	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		Return(&shipment.Shipment{
			ID:                shipmentID,
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			ServiceTypeID:     pulid.MustNew("svc_"),
			CustomerID:        pulid.MustNew("cust_"),
			FormulaTemplateID: pulid.MustNew("ft_"),
			BOL:               "BOL-1",
			Status:            shipment.StatusNew,
			Moves: []*shipment.ShipmentMove{
				{
					ID:             sourceMoveID,
					ShipmentID:     shipmentID,
					OrganizationID: orgID,
					BusinessUnitID: buID,
					Status:         shipment.MoveStatusAssigned,
					Stops: []*shipment.Stop{
						{
							ID:                   pulid.MustNew("stp_"),
							ShipmentMoveID:       sourceMoveID,
							OrganizationID:       orgID,
							BusinessUnitID:       buID,
							LocationID:           currentLocationID,
							Type:                 shipment.StopTypePickup,
							ScheduleType:         shipment.StopScheduleTypeOpen,
							Sequence:             0,
							ScheduledWindowStart: 1,
							ScheduledWindowEnd:   new(int64(2)),
							Status:               shipment.StopStatusNew,
						},
						{
							ID:                   pulid.MustNew("stp_"),
							ShipmentMoveID:       sourceMoveID,
							OrganizationID:       orgID,
							BusinessUnitID:       buID,
							LocationID:           currentLocationID,
							Type:                 shipment.StopTypeDelivery,
							ScheduleType:         shipment.StopScheduleTypeOpen,
							Sequence:             1,
							ScheduledWindowStart: 3,
							ScheduledWindowEnd:   new(int64(4)),
							Status:               shipment.StopStatusNew,
						},
					},
				},
			},
		}, nil).
		Once()
	shipmentRepo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			require.Len(t, entity.Moves, 2)
			lastMove := entity.Moves[1]
			lastMove.ID = pulid.MustNew("sm_")
			require.False(t, lastMove.Loaded)
			assert.Equal(t, shipment.MoveStatusCompleted, lastMove.Status)
			require.Len(t, lastMove.Stops, 2)
			assert.Equal(t, currentLocationID, lastMove.Stops[0].LocationID)
			assert.Equal(t, newLocationID, lastMove.Stops[1].LocationID)
			assert.Equal(t, shipment.StopStatusCompleted, lastMove.Stops[0].Status)
			assert.Equal(t, shipment.StopStatusCompleted, lastMove.Stops[1].Status)
			require.NotNil(t, lastMove.Stops[0].ActualArrival)
			require.NotNil(t, lastMove.Stops[0].ActualDeparture)
			require.NotNil(t, lastMove.Stops[1].ActualArrival)
			require.NotNil(t, lastMove.Stops[1].ActualDeparture)
			require.NotNil(t, lastMove.Stops[0].ScheduledWindowEnd)
			require.NotNil(t, lastMove.Stops[1].ScheduledWindowEnd)
			assert.Less(
				t,
				lastMove.Stops[0].ScheduledWindowStart,
				*lastMove.Stops[0].ScheduledWindowEnd,
			)
			assert.Less(
				t,
				*lastMove.Stops[0].ScheduledWindowEnd,
				lastMove.Stops[1].ScheduledWindowStart,
			)
			assert.LessOrEqual(
				t,
				lastMove.Stops[1].ScheduledWindowStart,
				*lastMove.Stops[1].ScheduledWindowEnd,
			)
			return entity, nil
		}).
		Once()

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{AutoDelayShipmentsThreshold: new(int16(30))}, nil).
		Once()

	formula := formulatemplateportsmock.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{Amount: decimal.Zero}, nil).
		Once()
	systemUserID := pulid.MustNew("usr_")
	deps.userRepo.EXPECT().
		GetSystemUser(mock.Anything, mock.Anything).
		Return(&tenant.User{ID: systemUserID}, nil).
		Once()

	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTrailerID(mock.Anything, tenantInfo, trailerID, pulid.Nil).
		Return(nil, nil).
		Once()
	assignmentRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *shipment.Assignment) bool {
			return entity.OrganizationID == orgID &&
				entity.BusinessUnitID == buID &&
				entity.ShipmentMoveID != pulid.Nil &&
				entity.Status == shipment.AssignmentStatusCompleted &&
				entity.PrimaryWorkerID == nil &&
				entity.TractorID == nil &&
				entity.TrailerID != nil &&
				*entity.TrailerID == trailerID
		})).
		RunAndReturn(func(_ context.Context, entity *shipment.Assignment) (*shipment.Assignment, error) {
			entity.ID = pulid.MustNew("asn_")
			return entity, nil
		}).
		Once()

	locationRepo := locationportsmock.NewMockLocationRepository(t)
	locationRepo.EXPECT().
		GetByID(mock.Anything, locationports.GetLocationByIDRequest{
			ID:         currentLocationID,
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&location.Location{ID: currentLocationID, OrganizationID: orgID, BusinessUnitID: buID, Name: "Madison Yard"}, nil).
		Once()
	locationRepo.EXPECT().
		GetByID(mock.Anything, locationports.GetLocationByIDRequest{
			ID:         newLocationID,
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&location.Location{ID: newLocationID, OrganizationID: orgID, BusinessUnitID: buID, Name: "Greensboro Yard"}, nil).
		Once()

	commentRepo := shipmentportsmock.NewMockShipmentCommentRepository(t)
	commentRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(comment *shipment.ShipmentComment) bool {
			return comment.ShipmentID == shipmentID &&
				comment.UserID == systemUserID &&
				comment.Comment == "System-generated empty reposition move created from trailer locate for trailer TRL-100: Madison Yard -> Greensboro Yard."
		})).
		Return(&shipment.ShipmentComment{ID: pulid.MustNew("shc_")}, nil).
		Once()

	deps.svc.db = dbtest.NopConnection{}
	deps.svc.assignmentRepo = assignmentRepo
	deps.svc.continuityRepo = continuityRepo
	deps.svc.shipmentRepo = shipmentRepo
	deps.svc.shipmentCommentRepo = commentRepo
	deps.svc.controlRepo = controlRepo
	deps.svc.locationRepo = locationRepo
	deps.svc.shipmentValidator = newTestShipmentValidator(t)
	deps.svc.coordinator = shipmentstate.NewCoordinator()
	deps.svc.commercial = shipmentcommercial.New(shipmentcommercial.Params{
		Logger:          zap.NewNop(),
		RateEngine:      ratingtest.NewFallbackEngine(t, formula),
		AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
	})

	result, err := deps.svc.Locate(t.Context(), &trailerports.LocateTrailerRequest{
		TenantInfo:    tenantInfo,
		TrailerID:     trailerID,
		NewLocationID: newLocationID,
	}, internaltestutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestLocate_RejectsTrailerAlreadyInProgress(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	trailerID := pulid.MustNew("tr_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}

	deps.repo.EXPECT().
		GetByID(mock.Anything, trailerports.GetTrailerByIDRequest{
			ID:         trailerID,
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&trailer.Trailer{ID: trailerID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Once()

	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTrailerID(mock.Anything, tenantInfo, trailerID, pulid.Nil).
		Return(&shipment.Assignment{
			ID:             pulid.MustNew("asn_"),
			ShipmentMoveID: pulid.MustNew("sm_"),
			TrailerID:      &trailerID,
		}, nil).
		Once()

	deps.svc.db = dbtest.NopConnection{}
	deps.svc.assignmentRepo = assignmentRepo
	deps.svc.continuityRepo = equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t)
	deps.svc.shipmentRepo = shipmentportsmock.NewMockShipmentRepository(t)
	deps.svc.shipmentCommentRepo = shipmentportsmock.NewMockShipmentCommentRepository(t)
	deps.svc.controlRepo = tenantportsmock.NewMockShipmentControlRepository(t)
	deps.svc.locationRepo = locationportsmock.NewMockLocationRepository(t)
	deps.svc.shipmentValidator = newTestShipmentValidator(t)
	deps.svc.coordinator = shipmentstate.NewCoordinator()
	deps.svc.commercial = shipmentcommercial.New(shipmentcommercial.Params{
		Logger:          zap.NewNop(),
		RateEngine:      ratingtest.NewFallbackEngine(t, formulatemplateportsmock.NewMockFormulaCalculator(t)),
		AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
	})

	result, err := deps.svc.Locate(t.Context(), &trailerports.LocateTrailerRequest{
		TenantInfo:    tenantInfo,
		TrailerID:     trailerID,
		NewLocationID: pulid.MustNew("loc_"),
	}, internaltestutil.NewSessionActor(pulid.MustNew("usr_"), orgID, buID))

	require.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, "Trailer is currently in progress on another move", err.Error())
}

//go:fix inline
func ptrInt16Trailer(v int16) *int16 {
	return new(v)
}

func TestResolveDelayThresholdMinutes_DisablesAutomaticDelayWhenToggleOff(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		shipmentstate.DisabledDelayThresholdMinutes,
		shipmentstate.ResolveControlDelayThreshold(nil),
	)
	assert.Equal(
		t,
		shipmentstate.DisabledDelayThresholdMinutes,
		shipmentstate.ResolveControlDelayThreshold(&tenant.ShipmentControl{
			AutoDelayShipments:          false,
			AutoDelayShipmentsThreshold: new(int16(30)),
		}),
	)
	assert.Equal(t, int16(30), shipmentstate.ResolveControlDelayThreshold(&tenant.ShipmentControl{
		AutoDelayShipments:          true,
		AutoDelayShipmentsThreshold: new(int16(30)),
	}))
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")

	original := newTestEntity()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID
	original.Code = "OLD"

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	assert.Equal(t, entity.Code, result.Code)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestUpdate_GetOriginalError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")
	getErr := errors.New("not found")

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, getErr)

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, getErr, err)
	deps.repo.AssertNotCalled(t, "Update")
	deps.repo.AssertExpectations(t)
}

func TestBulkUpdateStatus_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	entity1 := newTestEntity()
	entity2 := newTestEntity()

	original1 := newTestEntity()
	original1.ID = entity1.ID
	original1.Status = domaintypes.EquipmentStatusAvailable

	original2 := newTestEntity()
	original2.ID = entity2.ID
	original2.Status = domaintypes.EquipmentStatusAvailable

	entity1.Status = domaintypes.EquipmentStatusOOS
	entity2.Status = domaintypes.EquipmentStatusOOS

	req := &trailerports.BulkUpdateTrailerStatusRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  entity1.OrganizationID,
			BuID:   entity1.BusinessUnitID,
			UserID: pulid.MustNew("usr_"),
		},
		TrailerIDs: []pulid.ID{entity1.ID, entity2.ID},
		Status:     domaintypes.EquipmentStatusOOS,
	}

	deps.repo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*trailer.Trailer{original1, original2}, nil)
	deps.repo.On("BulkUpdateStatus", mock.Anything, req).
		Return([]*trailer.Trailer{entity1, entity2}, nil)
	deps.audit.On("LogActions", mock.Anything).Return(nil)

	result, err := deps.svc.BulkUpdateStatus(ctx, req)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, domaintypes.EquipmentStatusOOS, result[0].Status)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestBulkUpdateStatus_GetByIDsError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	getErr := errors.New("not found")

	req := &trailerports.BulkUpdateTrailerStatusRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		TrailerIDs: []pulid.ID{pulid.MustNew("tr_")},
		Status:     domaintypes.EquipmentStatusOOS,
	}

	deps.repo.On("GetByIDs", mock.Anything, mock.Anything).Return(nil, getErr)

	result, err := deps.svc.BulkUpdateStatus(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, getErr, err)
	deps.repo.AssertNotCalled(t, "BulkUpdateStatus")
	deps.repo.AssertExpectations(t)
}

func TestNew(t *testing.T) {
	t.Parallel()
	repo := trailerportsmock.NewMockTrailerRepository(t)
	auditSvc := auditportsmock.NewMockAuditService(t)
	logger := zap.NewNop()

	svc := New(Params{
		Logger:       logger,
		Repo:         repo,
		Validator:    newTestValidator(),
		AuditService: auditSvc,
		Realtime:     &mocks.NoopRealtimeService{},
	})

	require.NotNil(t, svc)
}

func TestNewTestValidator(t *testing.T) {
	t.Parallel()
	v := NewTestValidator()
	require.NotNil(t, v)
	require.NotNil(t, v.validator)
}

func TestValidator_ValidateCreate(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	ctx := t.Context()

	t.Run("valid entity passes", func(t *testing.T) {
		t.Parallel()
		entity := newCreateEntity()
		result := v.ValidateCreate(ctx, entity)
		assert.Nil(t, result)
	})

	t.Run("invalid entity fails", func(t *testing.T) {
		t.Parallel()
		entity := &trailer.Trailer{
			BusinessUnitID: pulid.MustNew("bu_"),
			OrganizationID: pulid.MustNew("org_"),
			Status:         domaintypes.EquipmentStatusAvailable,
		}
		result := v.ValidateCreate(ctx, entity)
		assert.NotNil(t, result)
	})
}

func TestList_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	dbErr := errors.New("db error")

	req := &trailerports.ListTrailersRequest{
		Filter: &pagination.QueryOptions{},
	}

	deps.repo.On("List", mock.Anything, req).Return(nil, dbErr)

	result, err := deps.svc.List(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
	deps.repo.AssertExpectations(t)
}

func TestList_CustomFieldsError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	entity := newTestEntity()
	expected := &pagination.CursorListResult[*trailer.Trailer]{
		Items:       []*trailer.Trailer{entity},
		HasNextPage: false,
	}
	req := &trailerports.ListTrailersRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
		TrailerRelationIncludes: trailerports.TrailerRelationIncludes{
			IncludeCustomFields: true,
		},
	}

	deps.repo.On("List", mock.Anything, req).Return(expected, nil)
	deps.valueRepo.On("GetByResources", mock.Anything, mock.Anything).
		Return(nil, errors.New("custom fields error"))

	result, err := deps.svc.List(ctx, req)

	require.NoError(t, err)
	assert.Len(t, result.Items, 1)
	assert.False(t, result.HasNextPage)
	deps.repo.AssertExpectations(t)
}

func TestGet_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	notFoundErr := errors.New("not found")

	req := trailerports.GetTrailerByIDRequest{
		ID: pulid.MustNew("tr_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(nil, notFoundErr)

	result, err := deps.svc.Get(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)
	deps.repo.AssertExpectations(t)
}

func TestGet_CustomFieldsError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()

	req := trailerports.GetTrailerByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		TrailerRelationIncludes: trailerports.TrailerRelationIncludes{
			IncludeCustomFields: true,
		},
	}

	deps.repo.On("GetByID", mock.Anything, req).Return(entity, nil)
	deps.valueRepo.On("GetByResource", mock.Anything, mock.Anything).
		Return(nil, errors.New("custom fields error"))

	result, err := deps.svc.Get(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, entity.ID, result.ID)
	deps.repo.AssertExpectations(t)
}

func TestCreate_AuditLogError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newCreateEntity()
	userID := pulid.MustNew("usr_")

	created := newTestEntity()
	created.BusinessUnitID = entity.BusinessUnitID
	created.OrganizationID = entity.OrganizationID

	deps.repo.On("Create", mock.Anything, mock.Anything).Return(created, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything).Return(errors.New("audit error"))

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestCreate_WithCustomFields(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newCreateEntity()
	fieldID := pulid.MustNew("cfd_")
	entity.CustomFields = map[string]any{
		fieldID.String(): "testValue",
	}
	userID := pulid.MustNew("usr_")

	created := newTestEntity()
	created.BusinessUnitID = entity.BusinessUnitID
	created.OrganizationID = entity.OrganizationID

	defs := []*customfield.CustomFieldDefinition{
		{
			ID:        fieldID,
			Name:      "testField",
			FieldType: customfield.FieldTypeText,
		},
	}

	deps.repo.On("Create", mock.Anything, mock.Anything).Return(created, nil)
	deps.defRepo.On("GetActiveByResourceType", mock.Anything, mock.Anything).
		Return(defs, nil)
	deps.valueRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	assert.NotNil(t, result)
	deps.repo.AssertExpectations(t)
}

func TestCreate_CustomFieldsError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newCreateEntity()
	entity.CustomFields = map[string]any{
		"testField": "testValue",
	}
	userID := pulid.MustNew("usr_")

	created := newTestEntity()
	created.BusinessUnitID = entity.BusinessUnitID
	created.OrganizationID = entity.OrganizationID

	deps.repo.On("Create", mock.Anything, mock.Anything).Return(created, nil)
	deps.defRepo.On("GetActiveByResourceType", mock.Anything, mock.Anything).
		Return(nil, errors.New("custom field error"))

	result, err := deps.svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_RepoError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")
	updateErr := errors.New("update failed")

	original := newTestEntity()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(nil, updateErr)

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, updateErr, err)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_ValidationError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")

	entity := &trailer.Trailer{
		ID:             pulid.MustNew("tr_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.EquipmentStatusAvailable,
	}

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "GetByID")
	deps.repo.AssertNotCalled(t, "Update")
}

func TestUpdate_WithCustomFields(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	fieldID := pulid.MustNew("cfd_")
	entity.CustomFields = map[string]any{
		fieldID.String(): "testValue",
	}
	userID := pulid.MustNew("usr_")

	original := newTestEntity()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID
	original.Code = "OLD"

	defs := []*customfield.CustomFieldDefinition{
		{
			ID:        fieldID,
			Name:      "testField",
			FieldType: customfield.FieldTypeText,
		},
	}

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(entity, nil)
	deps.defRepo.On("GetActiveByResourceType", mock.Anything, mock.Anything).
		Return(defs, nil)
	deps.valueRepo.On("Upsert", mock.Anything, mock.Anything).Return(nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	assert.NotNil(t, result)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_CustomFieldsError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	entity.CustomFields = map[string]any{
		"testField": "testValue",
	}
	userID := pulid.MustNew("usr_")

	original := newTestEntity()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(entity, nil)
	deps.defRepo.On("GetActiveByResourceType", mock.Anything, mock.Anything).
		Return(nil, errors.New("custom field error"))

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_AuditLogError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	entity := newTestEntity()
	userID := pulid.MustNew("usr_")

	original := newTestEntity()
	original.ID = entity.ID
	original.OrganizationID = entity.OrganizationID
	original.BusinessUnitID = entity.BusinessUnitID
	original.Code = "OLD"

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(entity, nil)
	deps.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("audit error"))

	result, err := deps.svc.Update(
		ctx,
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

func TestBulkUpdateStatus_RepoError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	bulkErr := errors.New("bulk update failed")

	req := &trailerports.BulkUpdateTrailerStatusRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		TrailerIDs: []pulid.ID{pulid.MustNew("tr_")},
		Status:     domaintypes.EquipmentStatusOOS,
	}

	originals := []*trailer.Trailer{newTestEntity()}
	deps.repo.On("GetByIDs", mock.Anything, mock.Anything).Return(originals, nil)
	deps.repo.On("BulkUpdateStatus", mock.Anything, req).Return(nil, bulkErr)

	result, err := deps.svc.BulkUpdateStatus(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, bulkErr, err)
	deps.repo.AssertExpectations(t)
}

func TestBulkUpdateStatus_AuditLogError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	entity := newTestEntity()
	entity.Status = domaintypes.EquipmentStatusOOS

	original := newTestEntity()
	original.ID = entity.ID

	req := &trailerports.BulkUpdateTrailerStatusRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  entity.OrganizationID,
			BuID:   entity.BusinessUnitID,
			UserID: pulid.MustNew("usr_"),
		},
		TrailerIDs: []pulid.ID{entity.ID},
		Status:     domaintypes.EquipmentStatusOOS,
	}

	deps.repo.On("GetByIDs", mock.Anything, mock.Anything).Return([]*trailer.Trailer{original}, nil)
	deps.repo.On("BulkUpdateStatus", mock.Anything, req).Return([]*trailer.Trailer{entity}, nil)
	deps.audit.On("LogActions", mock.Anything).Return(errors.New("audit error"))

	result, err := deps.svc.BulkUpdateStatus(ctx, req)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	deps.repo.AssertExpectations(t)
	deps.audit.AssertExpectations(t)
}

//go:fix inline
func int64PtrTrailer(v int64) *int64 {
	return new(v)
}
