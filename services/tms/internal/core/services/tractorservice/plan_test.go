package tractorservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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

	repo := mocks.NewMockTractorRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tractor.Tractor{ID: tractorID, Code: "T-100"}, nil).Once()
	locationRepo := mocks.NewMockLocationRepository(t)
	locationRepo.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&location.Location{ID: toID, Name: "Dallas yard"}, nil).Once()
	assignmentRepo := mocks.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, tractorID, pulid.Nil).
		Return(nil, nil).Once()
	continuityRepo := mocks.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().GetEffectiveCurrent(mock.Anything, mock.Anything).
		Return(&equipmentcontinuity.EquipmentContinuity{CurrentLocationID: fromID}, nil).Once()

	svc := newLocateTestService(repo, assignmentRepo, continuityRepo, locationRepo)
	plan, err := svc.PlanLocate(t.Context(), &repositories.LocateTractorRequest{
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
