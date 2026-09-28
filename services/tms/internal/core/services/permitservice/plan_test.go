package permitservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPlanPermit_ChecksWithoutWriting(t *testing.T) {
	stored := activePermit()
	svc := &service{
		permitRepo: &permitRepoStub{
			getByID: func(*repositories.GetPermitByIDRequest) (*permit.Permit, error) {
				copied := *stored
				return &copied, nil
			},
			create: func(*permit.Permit) (*permit.Permit, error) {
				t.Fatal("a plan must not record the permit")
				return nil, nil
			},
			update: func(*permit.Permit) (*permit.Permit, error) {
				t.Fatal("a plan must not update the permit")
				return nil, nil
			},
		},
		l: zap.NewNop(),
	}

	fresh := activePermit()
	fresh.ExpiresAt = nil
	_, err := svc.PlanCreatePermit(t.Context(), fresh)
	require.Error(t, err, "an active permit records when it expires")

	planned, err := svc.PlanCreatePermit(t.Context(), activePermit())
	require.NoError(t, err)
	assert.Equal(t, "GA-1234", planned.PermitNumber)

	edit := *stored
	edit.PermitNumber = "GA-9999"
	change, err := svc.PlanUpdatePermit(t.Context(), &edit)
	require.NoError(t, err)
	assert.Equal(t, "GA-1234", change.Before.PermitNumber)
	assert.Equal(t, "GA-9999", change.After.PermitNumber)

	elsewhere := edit
	elsewhere.ShipmentID = pulid.MustNew("shp_")
	_, err = svc.PlanUpdatePermit(t.Context(), &elsewhere)
	require.Error(t, err, "a permit is updated on the shipment it belongs to")
}
