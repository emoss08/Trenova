package audit

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rateAgreementEntry() *Entry {
	return &Entry{
		Resource:      permission.ResourceRateAgreement,
		PreviousState: map[string]any{"name": "Acme", "marginFloorPercent": "10"},
		CurrentState:  map[string]any{"name": "Acme", "marginFloorPercent": "12"},
		Changes: map[string]any{
			"marginFloorPercent": map[string]any{
				"from": "10", "to": "12", "type": "updated", "path": "marginFloorPercent",
			},
			"name": map[string]any{"from": "Acme Co", "to": "Acme", "type": "updated"},
		},
	}
}

func TestAnAuditEntryWithholdsAMarginFromAReaderWithoutTheGroup(t *testing.T) {
	t.Parallel()

	entry := rateAgreementEntry()
	withheld := entry.WithheldCopy(protectedfield.GroupSet(0).With(protectedfield.Personal))

	assert.Nil(t, withheld.CurrentState["marginFloorPercent"])
	assert.Nil(t, withheld.PreviousState["marginFloorPercent"])
	change := withheld.Changes["marginFloorPercent"].(map[string]any)
	assert.Nil(t, change["from"])
	assert.Nil(t, change["to"])
	assert.Equal(t, "updated", change["type"], "the reader still sees that the field changed")
	assert.Equal(t, "Acme", withheld.Changes["name"].(map[string]any)["to"])

	assert.Equal(t, "12", entry.CurrentState["marginFloorPercent"],
		"an administrator with the group still reads the whole diff")
	assert.Same(t, entry, entry.WithheldCopy(protectedfield.AllGroups()))
}

func TestAnAuditEntryWithholdsPersonalDetailsInsideAChangedObject(t *testing.T) {
	t.Parallel()

	entry := &Entry{
		Resource: permission.ResourceWorker,
		Changes: map[string]any{
			"profile": map[string]any{
				"from": nil,
				"to":   map[string]any{"dob": 315532800, "endorsement": "H"},
				"type": "created",
			},
			"profile.licenseNumber": map[string]any{"from": "D1", "to": "D2", "type": "updated"},
		},
		CurrentState: map[string]any{"profile": map[string]any{"twicCardNumber": "T-9"}},
	}

	withheld := entry.WithheldCopy(protectedfield.GroupSet(0).With(protectedfield.Margins))

	created := withheld.Changes["profile"].(map[string]any)["to"].(map[string]any)
	assert.Nil(t, created["dob"])
	assert.Equal(t, "H", created["endorsement"])
	license := withheld.Changes["profile.licenseNumber"].(map[string]any)
	assert.Nil(t, license["from"])
	assert.Nil(t, license["to"])
	require.NotNil(t, withheld.CurrentState["profile"])
	assert.Nil(t, withheld.CurrentState["profile"].(map[string]any)["twicCardNumber"])
}
