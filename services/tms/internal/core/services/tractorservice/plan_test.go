package tractorservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/equipmentcontinuityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/locationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tractorportsmock"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPlanLocate_NamesWhereTheTractorIsAndGoesWithoutMovingIt(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	tractorID := pulid.MustNew("trac_")
	fromID := pulid.MustNew("loc_")
	toID := pulid.MustNew("loc_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}

	repo := tractorportsmock.NewMockTractorRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tractor.Tractor{ID: tractorID, Code: "T-100"}, nil).Once()
	locationRepo := locationportsmock.NewMockLocationRepository(t)
	locationRepo.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&location.Location{ID: toID, Name: "Dallas yard"}, nil).Once()
	assignmentRepo := shipmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, tractorID, pulid.Nil).
		Return(nil, nil).Once()
	continuityRepo := equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().GetEffectiveCurrent(mock.Anything, mock.Anything).
		Return(&equipmentcontinuity.EquipmentContinuity{CurrentLocationID: fromID}, nil).Once()

	svc := newLocateTestService(repo, assignmentRepo, continuityRepo, locationRepo)
	plan, err := svc.PlanLocate(t.Context(), &tractorports.LocateTractorRequest{
		TenantInfo:    tenantInfo,
		TractorID:     tractorID,
		NewLocationID: toID,
	})
	require.NoError(t, err)
	assert.Equal(t, "T-100", plan.Tractor.Code)
	assert.Equal(t, "Dallas yard", plan.Location.Name)
	assert.Equal(t, fromID, plan.Current.CurrentLocationID)
	continuityRepo.AssertNotCalled(t, "Advance", mock.Anything, mock.Anything)
}
