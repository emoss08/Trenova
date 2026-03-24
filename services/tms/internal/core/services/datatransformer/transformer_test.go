package datatransformer

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type mockDataEntryControlRepo struct {
	mock.Mock
}

func (m *mockDataEntryControlRepo) GetByOrgID(
	ctx context.Context,
	req repositories.GetDataEntryControlRequest,
) (*dataentrycontrol.DataEntryControl, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dataentrycontrol.DataEntryControl), args.Error(1)
}

func (m *mockDataEntryControlRepo) Create(
	ctx context.Context,
	entity *dataentrycontrol.DataEntryControl,
) (*dataentrycontrol.DataEntryControl, error) {
	args := m.Called(ctx, entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dataentrycontrol.DataEntryControl), args.Error(1)
}

func (m *mockDataEntryControlRepo) Update(
	ctx context.Context,
	entity *dataentrycontrol.DataEntryControl,
) (*dataentrycontrol.DataEntryControl, error) {
	args := m.Called(ctx, entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dataentrycontrol.DataEntryControl), args.Error(1)
}

func (m *mockDataEntryControlRepo) GetOrCreate(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*dataentrycontrol.DataEntryControl, error) {
	args := m.Called(ctx, orgID, buID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dataentrycontrol.DataEntryControl), args.Error(1)
}

func TestTransformFleetCode_Success(t *testing.T) {
	t.Parallel()

	mockRepo := new(mockDataEntryControlRepo)
	svc := &Service{
		l:    zap.NewNop(),
		repo: mockRepo,
	}

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	ctrl := &dataentrycontrol.DataEntryControl{
		CodeCase: dataentrycontrol.CaseFormatUpper,
	}

	mockRepo.On("GetOrCreate", mock.Anything, orgID, buID).Return(ctrl, nil)

	entity := &fleetcode.FleetCode{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           "  hello   world  ",
		Description:    "  some   description  ",
	}

	err := svc.TransformFleetCode(t.Context(), entity)

	assert.NoError(t, err)
	assert.Equal(t, "HELLOWORLD", entity.Code)
	assert.Equal(t, "some description", entity.Description)
	mockRepo.AssertExpectations(t)
}

func TestTransformFleetCode_ControlError_ReturnsNil(t *testing.T) {
	t.Parallel()

	mockRepo := new(mockDataEntryControlRepo)
	svc := &Service{
		l:    zap.NewNop(),
		repo: mockRepo,
	}

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mockRepo.On("GetOrCreate", mock.Anything, orgID, buID).Return(nil, errors.New("db error"))

	entity := &fleetcode.FleetCode{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           "test",
	}

	err := svc.TransformFleetCode(t.Context(), entity)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTransformEquipmentType_Success(t *testing.T) {
	t.Parallel()

	mockRepo := new(mockDataEntryControlRepo)
	svc := &Service{
		l:    zap.NewNop(),
		repo: mockRepo,
	}

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	ctrl := &dataentrycontrol.DataEntryControl{
		CodeCase: dataentrycontrol.CaseFormatUpper,
	}

	mockRepo.On("GetOrCreate", mock.Anything, orgID, buID).Return(ctrl, nil)

	entity := &equipmenttype.EquipmentType{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           "flatbed",
	}

	err := svc.TransformEquipmentType(t.Context(), entity)

	assert.NoError(t, err)
	assert.Equal(t, "FLATBED", entity.Code)
	mockRepo.AssertExpectations(t)
}

func TestTransformEquipmentType_ControlError_ReturnsNil(t *testing.T) {
	t.Parallel()

	mockRepo := new(mockDataEntryControlRepo)
	svc := &Service{
		l:    zap.NewNop(),
		repo: mockRepo,
	}

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mockRepo.On("GetOrCreate", mock.Anything, orgID, buID).Return(nil, errors.New("db error"))

	entity := &equipmenttype.EquipmentType{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           "flatbed",
	}

	err := svc.TransformEquipmentType(t.Context(), entity)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTransformAccessorialCharge_Success(t *testing.T) {
	t.Parallel()

	mockRepo := new(mockDataEntryControlRepo)
	svc := &Service{
		l:    zap.NewNop(),
		repo: mockRepo,
	}

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	ctrl := &dataentrycontrol.DataEntryControl{
		CodeCase: dataentrycontrol.CaseFormatUpper,
	}

	mockRepo.On("GetOrCreate", mock.Anything, orgID, buID).Return(ctrl, nil)

	entity := &accessorialcharge.AccessorialCharge{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           "detention",
	}

	err := svc.TransformAccessorialCharge(t.Context(), entity)

	assert.NoError(t, err)
	assert.Equal(t, "DETENTION", entity.Code)
	mockRepo.AssertExpectations(t)
}

func TestTransformAccessorialCharge_ControlError_ReturnsNil(t *testing.T) {
	t.Parallel()

	mockRepo := new(mockDataEntryControlRepo)
	svc := &Service{
		l:    zap.NewNop(),
		repo: mockRepo,
	}

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mockRepo.On("GetOrCreate", mock.Anything, orgID, buID).Return(nil, errors.New("db error"))

	entity := &accessorialcharge.AccessorialCharge{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           "detention",
	}

	err := svc.TransformAccessorialCharge(t.Context(), entity)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
