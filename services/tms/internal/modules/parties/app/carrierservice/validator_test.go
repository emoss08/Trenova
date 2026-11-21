package carrierservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestValidateCreate_ACarrierWithoutAStateIsValid(t *testing.T) {
	t.Parallel()

	v := NewValidator(ValidatorParams{Integrity: validationtest.NewIntegrity()})

	assert.True(
		t,
		validationtest.Outcome(t, v.ValidateCreate(t.Context(), newCreateEntity())).Valid(),
	)
}

func TestValidateCreate_RefusesAStateThatDoesNotExist(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("us_")
	remitStateID := pulid.MustNew("us_")
	integrity := validationtest.NewIntegrity().
		Missing("us_states", stateID).
		Missing("us_states", remitStateID)
	v := NewValidator(ValidatorParams{Integrity: integrity})
	entity := newCreateEntity()
	entity.StateID = &stateID
	entity.RemitStateID = &remitStateID

	refused := validationtest.Refused(t, v.ValidateCreate(t.Context(), entity))
	refused.Refuses("stateId", errortypes.ErrInvalidReference)
	refused.Refuses("remitStateId", errortypes.ErrInvalidReference)
}
