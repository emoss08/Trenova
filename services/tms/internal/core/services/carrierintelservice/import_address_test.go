package carrierintelservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeUsStateRepo struct {
	repositories.UsStateRepository
	states map[string]*usstate.UsState
}

func (f *fakeUsStateRepo) GetByAbbreviation(
	_ context.Context,
	abbreviation string,
) (*usstate.UsState, error) {
	if state, ok := f.states[abbreviation]; ok {
		return state, nil
	}
	return nil, errortypes.NewNotFoundError("UsState not found within your organization")
}

func newStateService() (*Service, map[string]pulid.ID) {
	ids := map[string]pulid.ID{
		"TN": pulid.MustNew("us_"),
		"PA": pulid.MustNew("us_"),
	}
	states := make(map[string]*usstate.UsState, len(ids))
	for abbreviation, id := range ids {
		states[abbreviation] = &usstate.UsState{ID: id, Abbreviation: abbreviation}
	}
	return &Service{l: zap.NewNop(), usStateRepo: &fakeUsStateRepo{states: states}}, ids
}

func TestApplyProspectAddresses(t *testing.T) {
	t.Parallel()

	t.Run("resolves physical and mailing states to state ids", func(t *testing.T) {
		t.Parallel()
		svc, ids := newStateService()
		entity := &carrier.Carrier{Name: "FEDEX GROUND PACKAGE SYSTEM INC", DOTNumber: "265752"}

		err := svc.applyProspectAddresses(t.Context(), entity, &carrierintel.Identity{
			PhysicalAddress: &carrierintel.Address{
				Line1:      "3660 HACKS CROSS RD",
				City:       "MEMPHIS",
				State:      "tn",
				PostalCode: "38125",
			},
			MailingAddress: &carrierintel.Address{
				Line1:      "1000 FEDEX DR",
				City:       "MOON TOWNSHIP",
				State:      "PA",
				PostalCode: "15108",
			},
		})

		require.NoError(t, err)
		require.NotNil(t, entity.StateID)
		require.NotNil(t, entity.RemitStateID)
		assert.Equal(t, ids["TN"], *entity.StateID)
		assert.Equal(t, ids["PA"], *entity.RemitStateID)
		assert.Equal(t, "MEMPHIS", entity.City)
		assert.Equal(t, "MOON TOWNSHIP", entity.RemitCity)
		assert.Equal(t, entity.Name, entity.RemitToName)
	})

	t.Run("uses the physical address for remittance without a mailing address", func(t *testing.T) {
		t.Parallel()
		svc, ids := newStateService()
		entity := &carrier.Carrier{Name: "ACME", DOTNumber: "1"}

		err := svc.applyProspectAddresses(t.Context(), entity, &carrierintel.Identity{
			PhysicalAddress: &carrierintel.Address{
				Line1: "1 MAIN ST",
				City:  "MEMPHIS",
				State: "TN",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, ids["TN"], *entity.StateID)
		assert.Equal(t, ids["TN"], *entity.RemitStateID)
		assert.Equal(t, "1 MAIN ST", entity.RemitAddressLine1)
	})

	t.Run(
		"falls back to the mailing address when the physical state is unknown",
		func(t *testing.T) {
			t.Parallel()
			svc, ids := newStateService()
			entity := &carrier.Carrier{Name: "ACME", DOTNumber: "1"}

			err := svc.applyProspectAddresses(t.Context(), entity, &carrierintel.Identity{
				PhysicalAddress: &carrierintel.Address{City: "TORONTO", State: "ON"},
				MailingAddress:  &carrierintel.Address{City: "ERIE", State: "PA"},
			})

			require.NoError(t, err)
			assert.Equal(t, ids["PA"], *entity.StateID)
			assert.Equal(t, "ERIE", entity.City)
		},
	)

	t.Run("refuses a carrier with no U.S. state", func(t *testing.T) {
		t.Parallel()
		svc, _ := newStateService()
		entity := &carrier.Carrier{Name: "ACME", DOTNumber: "1"}

		err := svc.applyProspectAddresses(t.Context(), entity, &carrierintel.Identity{
			PhysicalAddress: &carrierintel.Address{City: "TORONTO", State: "ON"},
		})

		var business *errortypes.BusinessError
		require.ErrorAs(t, err, &business)
		assert.Nil(t, entity.StateID)
	})
}
