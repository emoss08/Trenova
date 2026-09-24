package agentdefinition

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestUsableWith_UnderEachMode(t *testing.T) {
	t.Parallel()

	open := &Definition{ID: pulid.MustNew("agdef_"), AccessMode: AccessEveryone}
	restricted := &Definition{ID: pulid.MustNew("agdef_"), AccessMode: AccessRoles}

	assert.True(t, open.UsableWith(nil), "an open agent needs no grant")
	assert.False(t, restricted.UsableWith(nil), "a restricted agent with no grant is nobody's")
	assert.False(t, restricted.UsableWith([]pulid.ID{open.ID}))
	assert.True(t, restricted.UsableWith([]pulid.ID{open.ID, restricted.ID}))
}

// A system agent is open whatever its row says: the platform fires it for
// people, and restricting it would break the feature it serves.
func TestSystemAgent_IsAlwaysOpenAndCannotBeRestricted(t *testing.T) {
	t.Parallel()

	system := &Definition{
		ID:         pulid.MustNew("agdef_"),
		Name:       "Briefing",
		SystemKey:  "briefing",
		AccessMode: AccessRoles,
	}

	assert.Equal(t, AccessEveryone, system.EffectiveAccessMode())
	assert.True(t, system.OpenToEveryone())
	assert.True(t, system.UsableWith(nil))
	assert.Contains(t, system.AccessRefusal(AccessRoles), "system agent")
	assert.Empty(t, system.AccessRefusal(AccessEveryone))

	multiErr := errortypes.NewMultiError()
	system.validateAccess(multiErr)
	assert.True(t, multiErr.HasErrors(), "a system agent stored as restricted is invalid")
}

func TestAccessMode_DefaultsToEveryoneAndRefusesAnUnknownValue(t *testing.T) {
	t.Parallel()

	definition := &Definition{Name: "Dispatch"}
	assert.Equal(t, AccessEveryone, definition.EffectiveAccessMode())

	definition.ApplyDefaults()
	assert.Equal(t, AccessEveryone, definition.AccessMode, "a new agent is open to everyone")

	assert.NotEmpty(t, definition.AccessRefusal(AccessMode("Admins")))
	assert.Empty(t, definition.AccessRefusal(AccessRoles))
	assert.ElementsMatch(t, []AccessMode{AccessEveryone, AccessRoles}, AllAccessModes())

	definition.AccessMode = AccessMode("Admins")
	multiErr := errortypes.NewMultiError()
	definition.validateAccess(multiErr)
	assert.True(t, multiErr.HasErrors())
}
