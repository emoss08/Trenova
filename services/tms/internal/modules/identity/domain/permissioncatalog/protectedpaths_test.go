package permissioncatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

func onlyGroup(group permission.FieldGroup) permission.FieldGroupSet {
	return permission.FieldGroupSet(0).With(group)
}

func TestWithholdProtectedJSONAnswersAProtectedFieldNullWithoutItsGroup(t *testing.T) {
	t.Parallel()

	worker := map[string]any{
		"firstName": "Rosa",
		"profile":   map[string]any{"dob": 315532800, "licenseNumber": "D123", "endorsement": "H"},
	}

	withheld := permission.WithholdProtectedJSON(
		permission.ResourceWorker,
		worker,
		onlyGroup(permission.FieldGroupMargins),
	)
	assert.Equal(t, map[string]any{
		"firstName": "Rosa",
		"profile":   map[string]any{"dob": nil, "licenseNumber": nil, "endorsement": "H"},
	}, withheld)
	assert.Equal(t, 315532800, worker["profile"].(map[string]any)["dob"],
		"the stored entry is never changed in place")

	assert.Equal(
		t,
		worker,
		permission.WithholdProtectedJSON(
			permission.ResourceWorker,
			worker,
			permission.AllFieldGroups(),
		),
	)
}

func TestWithholdProtectedJSONFindsAProtectedFieldAtAnyDepthAndInLists(t *testing.T) {
	t.Parallel()

	tender := map[string]any{
		"status": "Live",
		"offers": []any{
			map[string]any{"rate": "1500", "counterMargin": "7.5"},
			map[string]any{"rate": "1400", "counterMargin": nil},
		},
	}

	withheld := permission.WithholdProtectedJSON(
		permission.ResourceTender,
		tender,
		onlyGroup(permission.FieldGroupPersonal),
	)
	offers := withheld["offers"].([]any)
	assert.Nil(t, offers[0].(map[string]any)["counterMargin"])
	assert.Equal(
		t,
		"1500",
		offers[0].(map[string]any)["rate"],
		"the carrier's rate is not a margin",
	)
}

func TestWithholdProtectedJSONWithholdsOnlyTheDriverPayStepOfATrace(t *testing.T) {
	t.Parallel()

	occurrence := map[string]any{
		"billableAmount": "225",
		"netMargin":      "150",
		"calculationTrace": map[string]any{"steps": []any{
			map[string]any{"kind": "DriverPay", "amount": "75"},
			map[string]any{"kind": "Final", "amount": "225"},
		}},
	}

	withheld := permission.WithholdProtectedJSON(permission.ResourceDetention, occurrence, 0)
	steps := withheld["calculationTrace"].(map[string]any)["steps"].([]any)
	assert.Nil(t, steps[0].(map[string]any)["amount"])
	assert.Equal(t, "225", steps[1].(map[string]any)["amount"])
	assert.Nil(t, withheld["netMargin"])
	assert.Equal(t, "225", withheld["billableAmount"])
}

func TestARootedProtectedPathHoldsOnlyAtTheRootOfItsResource(t *testing.T) {
	t.Parallel()

	credential := map[string]any{"number": "CDL-1", "status": "Active"}
	assert.Nil(
		t,
		permission.WithholdProtectedJSON(permission.ResourceWorkerCredential, credential, 0)["number"],
	)

	invoice := map[string]any{"number": "INV-1"}
	assert.Equal(
		t,
		"INV-1",
		permission.WithholdProtectedJSON(permission.ResourceInvoice, invoice, 0)["number"],
		"a credential's number is protected, not every number",
	)

	quote := map[string]any{"costAmount": "800", "totalAmount": "1000"}
	assert.Nil(
		t,
		permission.WithholdProtectedJSON(permission.ResourceRateQuote, quote, 0)["costAmount"],
	)
	safety := map[string]any{"costAmount": "120"}
	assert.Equal(
		t,
		"120",
		permission.WithholdProtectedJSON(permission.ResourceWorker, safety, 0)["costAmount"],
	)
}

func TestEveryProtectedPathNamesAKnownGroupAndResource(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	for _, path := range permission.ProtectedPaths() {
		assert.Truef(t, path.Group.IsValid(), "%s names no known group", path.Path)
		if path.Resource != "" {
			assert.Truef(t, registry.HasResource(path.Resource.String()),
				"%s is rooted at unknown resource %s", path.Path, path.Resource)
		}
	}
}
