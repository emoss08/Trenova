package pageaccessservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/pageaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/pageaccess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func guides(
	t *testing.T,
	maxSensitivity permission.FieldSensitivity,
	resources ...string,
) map[string]pageaccessservice.ResourceGuide {
	t.Helper()
	svc := newService(t, &engine{}, permissionportsmock.NewMockRoleRepository(t))
	out, err := svc.Guide(t.Context(), &pageaccessservice.GuideRequest{
		Resources:      resources,
		MaxSensitivity: maxSensitivity,
	})
	require.NoError(t, err)
	byResource := make(map[string]pageaccessservice.ResourceGuide, len(out))
	for _, guide := range out {
		byResource[guide.Resource] = guide
	}
	return byResource
}

func TestGuide_WordsAViewAndTheKindsThatLinkToIt(t *testing.T) {
	t.Parallel()

	customer := guides(t, permission.SensitivityInternal, "customer")["customer"]

	assert.Equal(t, "Customer", customer.Name)
	assert.Equal(t,
		"Can see every field of Customer records, on their own screens and wherever another "+
			"record shows one.",
		customer.Granted)
	assert.Equal(t,
		"Customer records show only their code, name, status, address, external ID, allow "+
			"consolidation, exclusive consolidation, consolidation priority and EDI partner "+
			"wherever a record they can read links to one (Invoice, Shipment, "+
			"Customer Payment and 7 others); the rest needs View on Customer.",
		customer.Withheld)
	assert.Empty(t, customer.Ceiling, "an Internal role reaches every view field")
}

func TestGuide_NamesFewLinkingKindsInFull(t *testing.T) {
	t.Parallel()

	commodity := guides(t, permission.SensitivityInternal, "fleet_code")["fleet_code"]

	assert.Contains(t, commodity.Withheld,
		"wherever a record they can read links to one (Tractor, Trailer and Worker)")
}

func TestGuide_NamesTheViewFieldsAboveTheRolesCeiling(t *testing.T) {
	t.Parallel()

	worker := guides(t, permission.SensitivityPublic, "worker")["worker"]

	assert.Equal(t,
		"Without View on Worker, its first name, last name, whole name, profile pic URL, "+
			"status, type and fleet code stay withheld there too: they rank above this role's "+
			"sensitivity ceiling.",
		worker.Ceiling)
}

func TestGuide_AKindNoRecordLinksToAppearsNowhereWithoutItsRead(t *testing.T) {
	t.Parallel()

	role := guides(t, permission.SensitivityInternal, "api_key")["api_key"]

	assert.Equal(t,
		"No other record shows API Key records, so without View on API Key they see none "+
			"of them.",
		role.Withheld)
}

func TestGuide_ListsTheScreensEachGrantChanges(t *testing.T) {
	t.Parallel()

	byResource := guides(t, permission.SensitivityInternal, "customer", "worker", "invoice")

	assert.Equal(t, []pageaccessservice.ScreenUse{
		{Path: "/shipments", Name: "Shipments", Use: pageaccess.UseDeclared},
	}, byResource["customer"].Screens)
	assert.Equal(t, []pageaccessservice.ScreenUse{
		{Path: "/shipments", Name: "Shipments", Use: pageaccess.UseReference},
	}, byResource["worker"].Screens)
	assert.Equal(t, []pageaccessservice.ScreenUse{
		{Path: "/invoices/:id", Name: "", Use: pageaccess.UseRead},
	}, byResource["invoice"].Screens)
}

func TestGuide_RefusesAnUnknownResourceOrTooMany(t *testing.T) {
	t.Parallel()

	svc := newService(t, &engine{}, permissionportsmock.NewMockRoleRepository(t))
	_, err := svc.Guide(t.Context(), &pageaccessservice.GuideRequest{
		Resources: []string{"nope"}, MaxSensitivity: permission.SensitivityInternal,
	})
	require.Error(t, err)

	many := make([]string, 65)
	for i := range many {
		many[i] = "customer"
	}
	_, err = svc.Guide(t.Context(), &pageaccessservice.GuideRequest{
		Resources: many, MaxSensitivity: permission.SensitivityInternal,
	})
	require.Error(t, err)

	_, err = svc.Guide(t.Context(), &pageaccessservice.GuideRequest{
		Resources: []string{"customer"}, MaxSensitivity: "secret",
	})
	require.Error(t, err)
}
