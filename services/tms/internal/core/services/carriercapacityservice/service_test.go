package carriercapacityservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type memoryRepo struct {
	repositories.CarrierCapacityPostingRepository
	stored  map[pulid.ID]*carriercapacity.Posting
	deleted []pulid.ID
}

func (m *memoryRepo) GetByID(
	_ context.Context,
	req *repositories.GetCarrierCapacityPostingRequest,
) (*carriercapacity.Posting, error) {
	entity, ok := m.stored[req.ID]
	if !ok {
		return nil, assert.AnError
	}
	clone := *entity
	return &clone, nil
}

func (m *memoryRepo) Create(
	_ context.Context,
	entity *carriercapacity.Posting,
) (*carriercapacity.Posting, error) {
	entity.ID = pulid.MustNew("ccp_")
	entity.CreatedAt = 10
	m.stored[entity.ID] = entity
	return entity, nil
}

func (m *memoryRepo) Update(
	_ context.Context,
	entity *carriercapacity.Posting,
) (*carriercapacity.Posting, error) {
	entity.Version++
	m.stored[entity.ID] = entity
	return entity, nil
}

func (m *memoryRepo) Delete(
	_ context.Context,
	req *repositories.DeleteCarrierCapacityPostingRequest,
) error {
	m.deleted = append(m.deleted, req.ID)
	delete(m.stored, req.ID)
	return nil
}

type recordingInvalidator struct {
	calls []*services.ShipmentInvalidation
}

func (r *recordingInvalidator) InvalidateShipments(
	_ context.Context,
	req *services.ShipmentInvalidation,
) {
	r.calls = append(r.calls, req)
}

func newTestService(t *testing.T) (*Service, *memoryRepo, *recordingInvalidator) {
	t.Helper()

	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Maybe()
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	repo := &memoryRepo{stored: map[pulid.ID]*carriercapacity.Posting{}}
	invalidator := &recordingInvalidator{}
	svc := &Service{
		l:            zap.NewNop(),
		repo:         repo,
		validator:    &Validator{validator: nil},
		auditService: audit,
		invalidator:  invalidator,
	}
	return svc, repo, invalidator
}

func TestNormalizeDefaultsAndDetaches(t *testing.T) {
	t.Parallel()

	posting := &carriercapacity.Posting{EquipmentType: nil}
	normalize(posting)
	assert.Equal(t, carriercapacity.SourceManual, posting.Source)
	assert.Equal(t, carriercapacity.RateMethodPerMile, posting.RateMethod)
	assert.Equal(t, 1, posting.TruckCount)
}

func TestDeleteAuditsAndInvalidates(t *testing.T) {
	t.Parallel()

	svc, repo, invalidator := newTestService(t)
	id := pulid.MustNew("ccp_")
	org := pulid.MustNew("org_")
	bu := pulid.MustNew("bu_")
	repo.stored[id] = &carriercapacity.Posting{ID: id, OrganizationID: org, BusinessUnitID: bu}

	err := svc.Delete(t.Context(), &repositories.DeleteCarrierCapacityPostingRequest{
		ID:         id,
		Version:    0,
		TenantInfo: pagination.TenantInfo{OrgID: org, BuID: bu},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{id}, repo.deleted)
	require.Len(t, invalidator.calls, 1)
	assert.Equal(t, org, invalidator.calls[0].OrganizationID)
	assert.Equal(t, invalidationAction, invalidator.calls[0].Action)

	err = svc.Delete(t.Context(), &repositories.DeleteCarrierCapacityPostingRequest{ID: id}, nil)
	require.Error(t, err)
}
