package accountguard_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/accountguard"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type systemActors struct {
	id  pulid.ID
	err error
}

func (s systemActors) SystemActorID(context.Context, pulid.ID) (pulid.ID, error) {
	return s.id, s.err
}

func TestTheSystemActorIsAnsweredForAsAnUnknownUser(t *testing.T) {
	t.Parallel()

	system := pulid.MustNew("usr_")
	person := pulid.MustNew("usr_")
	unit := pulid.MustNew("bu_")
	actors := systemActors{id: system}

	err := accountguard.RequirePeople(t.Context(), actors, unit, person, system)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	require.NoError(t, accountguard.RequirePeople(t.Context(), actors, unit, person))
	require.NoError(t, accountguard.RequirePeople(t.Context(), actors, unit, pulid.Nil),
		"a request about the session's own person names nobody else")

	kept, err := accountguard.WithoutSystemActor(t.Context(), actors, unit,
		[]pulid.ID{person, system})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{person}, kept)
}

func TestATenantWithoutASystemActorHasNobodyToHide(t *testing.T) {
	t.Parallel()

	actors := systemActors{err: errortypes.NewNotFoundError("System actor not found")}
	person := pulid.MustNew("usr_")
	require.NoError(t, accountguard.RequirePeople(t.Context(), actors, pulid.MustNew("bu_"), person))

	failing := systemActors{err: errors.New("the database is unreachable")}
	require.Error(t, accountguard.RequirePeople(t.Context(), failing, pulid.MustNew("bu_"), person),
		"a failed read never lets the request through unchecked")
}
