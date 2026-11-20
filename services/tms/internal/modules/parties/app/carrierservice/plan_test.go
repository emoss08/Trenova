package carrierservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/testutil/mocks/dataentrycontrolportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPlanCreate_RefusesWhatCreateRefusesAndWritesNothing(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newCreateEntity()
	entity.Name = ""

	_, err := deps.svc.PlanCreate(t.Context(), entity)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	deps.repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestPlanCreate_FormatsTheCarrierWithTheTenantsSettings(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	formatter := dataentrycontrolportsmock.NewMockTextFormatter(t)
	deps.svc.formatter = formatter
	entity := newCreateEntity()
	formatter.EXPECT().
		Format(mock.Anything, entity).
		Run(func(_ context.Context, target dataentrycontrol.TextFormattable) {
			target.ApplyTextFormat(dataentrycontrol.TextFormat{
				CodeCase: dataentrycontrol.CaseFormatUpper,
				NameCase: dataentrycontrol.CaseFormatUpper,
			})
		}).
		Return(nil).
		Once()
	entity.Code = "swft 01"
	entity.Name = "swift transport"

	planned, err := deps.svc.PlanCreate(t.Context(), entity)

	require.NoError(t, err)
	assert.Equal(t, "SWFT01", planned.Code)
	assert.Equal(t, "SWIFT TRANSPORT", planned.Name)
}

func TestPlanCreate_StopsWhenFormattingFails(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	formatter := dataentrycontrolportsmock.NewMockTextFormatter(t)
	deps.svc.formatter = formatter
	entity := newCreateEntity()
	formatErr := errors.New("format failed")
	formatter.EXPECT().Format(mock.Anything, entity).Return(formatErr).Once()

	planned, err := deps.svc.PlanCreate(t.Context(), entity)

	require.ErrorIs(t, err, formatErr)
	assert.Nil(t, planned)
}

func TestPlanUpdate_ReturnsTheStoredCarrierBesideTheChange(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newTestEntity()
	stored := *entity
	stored.Name = "Old Name"
	deps.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&stored, nil)

	change, err := deps.svc.PlanUpdate(t.Context(), entity)
	require.NoError(t, err)
	assert.Equal(t, "Old Name", change.Before.Name)
	assert.Equal(t, entity.Name, change.After.Name)
	deps.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestPlanBulkUpdateStatus_RefusesACarrierThatIsNotHere(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	entity := newTestEntity()
	deps.repo.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return(
		[]*carrier.Carrier{entity}, nil,
	)

	_, err := deps.svc.PlanBulkUpdateStatus(
		t.Context(),
		&carrierports.BulkUpdateCarrierStatusRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			CarrierIDs: []pulid.ID{entity.ID, pulid.MustNew("car_")},
			Status:     carrier.StatusInactive,
		},
	)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "carrierIds", multiErr.Errors[0].Field())
	deps.repo.AssertNotCalled(t, "BulkUpdateStatus", mock.Anything, mock.Anything)
}
