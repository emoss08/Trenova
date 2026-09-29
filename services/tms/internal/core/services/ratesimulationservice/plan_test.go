package ratesimulationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratesimulation"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPlanCreate_NamesTheAgreementAndSavesNothing(t *testing.T) {
	t.Parallel()

	agreement := &rateagreement.RateAgreement{
		ID:        pulid.MustNew("rag_"),
		Code:      "ACME-2027",
		Name:      "Acme renewal",
		PartyType: rateagreement.PartyTypeCustomer,
		Status:    rateagreement.StatusDraft,
	}
	agreements := mocks.NewMockRateAgreementRepository(t)
	agreements.EXPECT().GetByID(mock.Anything, mock.Anything).Return(agreement, nil).Once()
	agreements.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Rate agreement not found")).
		Once()

	svc := &Service{l: zap.NewNop(), agreementRepo: agreements}
	entity := &ratesimulation.RateSimulation{
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		RateAgreementID: agreement.ID,
		Name:            "Renewal against Q3",
		SampleFrom:      1_780_000_000,
		SampleTo:        1_787_000_000,
	}

	plan, err := svc.PlanCreate(t.Context(), entity)
	require.NoError(t, err)
	assert.Equal(t, ratesimulation.StatusPending, plan.Simulation.Status)
	assert.Equal(t, agreement.ID, plan.Agreement.ID)
	assert.Empty(t, entity.Status)

	_, err = svc.PlanCreate(t.Context(), entity)
	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)

	backwards := *entity
	backwards.SampleTo = backwards.SampleFrom
	_, err = svc.PlanCreate(t.Context(), &backwards)
	require.Error(t, err)
}
