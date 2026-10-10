package sim

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"
)

func TestExternalIDValuesAreUniqueAcrossObjectClasses(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverAlex, map[string]any{
		"externalIds": map[string]any{"maintenance": "1234"},
	}).expect(t, http.StatusOK)

	callAPI(t, srv, http.MethodPatch, "/fleet/vehicles/"+fixtureTruck1001, map[string]any{
		"externalIds": map[string]any{"maintenance": "1234"},
	}).expectError(t, http.StatusBadRequest, "unique across all objects")

	callAPI(t, srv, http.MethodPost, "/tags", map[string]any{
		"name":        "External ID Clash",
		"externalIds": map[string]any{"maintenance": "1234"},
	}).expectError(t, http.StatusBadRequest, "already assigned")

	callAPI(t, srv, http.MethodPatch, "/fleet/vehicles/"+fixtureTruck1001, map[string]any{
		"externalIds": map[string]any{"payroll": "1234"},
	}).expect(t, http.StatusOK)

	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverAlex, map[string]any{
		"externalIds": map[string]any{"maintenance": "1234", "payroll": "9"},
	}).expect(t, http.StatusOK)
}

func TestExternalIDKeyAndValueRules(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	target := "/fleet/drivers/" + fixtureDriverAlex

	cases := []struct {
		name     string
		ids      map[string]any
		fragment string
	}{
		{name: "underscore key", ids: map[string]any{"payroll_id": "1"}, fragment: "letters and digits"},
		{name: "dash key", ids: map[string]any{"payroll-id": "1"}, fragment: "letters and digits"},
		{name: "space key", ids: map[string]any{"pay roll": "1"}, fragment: "letters and digits"},
		{
			name:     "long key",
			ids:      map[string]any{"k123456789012345678901234567890123": "1"},
			fragment: "1-32",
		},
		{name: "reserved key", ids: map[string]any{"samsara.vin": "1"}, fragment: "reserved"},
		{name: "slash value", ids: map[string]any{"payroll": "a/b"}, fragment: "@ . _ % + -"},
		{name: "space value", ids: map[string]any{"payroll": "a b"}, fragment: "@ . _ % + -"},
		{name: "object value", ids: map[string]any{"payroll": map[string]any{}}, fragment: "string"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			callAPI(t, srv, http.MethodPatch, target, map[string]any{
				"externalIds": testCase.ids,
			}).expectError(t, http.StatusBadRequest, testCase.fragment)
		})
	}

	updated := callAPI(t, srv, http.MethodPatch, target, map[string]any{
		"externalIds": map[string]any{
			"payrollSys1": "Ty_Frec-%99+88+77%",
			"badge":       float64(4417),
			"active":      true,
			"retired":     "",
		},
	}).expect(t, http.StatusOK).data(t)
	ids := mapOf(updated["externalIds"])
	if ids["payrollSys1"] != "Ty_Frec-%99+88+77%" || ids["badge"] != "4417" ||
		ids["active"] != "true" {
		t.Fatalf("expected converted external IDs, got %v", ids)
	}
	if _, kept := ids["retired"]; kept {
		t.Fatalf("expected empty external ID value to remove the key, got %v", ids)
	}

	ref := "/fleet/drivers/" + url.PathEscape("payrollSys1:Ty_Frec-%99+88+77%")
	fetched := callAPI(t, srv, http.MethodGet, ref, nil).expect(t, http.StatusOK).data(t)
	if recordID(fetched) != fixtureDriverAlex {
		t.Fatalf("expected external ID path lookup to find the driver, got %v", fetched)
	}
}

func TestExternalIDKeysAreLimitedPerObjectType(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	drivers := callAPI(t, srv, http.MethodGet, "/fleet/drivers", nil).expect(t, http.StatusOK).list(t)
	existing := map[string]struct{}{}
	for _, driver := range drivers {
		for key := range mapOf(driver["externalIds"]) {
			existing[key] = struct{}{}
		}
	}
	ids := map[string]any{}
	for idx := 0; len(existing)+len(ids) < maxExternalIDKeysPerType; idx++ {
		key := "limit" + strconv.Itoa(idx)
		if _, taken := existing[key]; !taken {
			ids[key] = "v" + strconv.Itoa(idx)
		}
	}
	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverJordan, map[string]any{
		"externalIds": ids,
	}).expect(t, http.StatusOK)

	callAPI(t, srv, http.MethodPatch, "/fleet/drivers/"+fixtureDriverCameron, map[string]any{
		"externalIds": map[string]any{"oneTooMany": "1"},
	}).expectError(t, http.StatusBadRequest, "at most 30 unique external ID keys")

	callAPI(t, srv, http.MethodPatch, "/fleet/vehicles/"+fixtureTruck1001, map[string]any{
		"externalIds": map[string]any{"oneTooMany": "1"},
	}).expect(t, http.StatusOK)

	vehicle := callAPI(t, srv, http.MethodGet, "/fleet/vehicles/"+fixtureTruck1001, nil).
		expect(t, http.StatusOK).data(t)
	if _, ok := mapOf(vehicle["externalIds"])[externalIDVinKey]; !ok {
		t.Fatalf("expected automatic samsara.vin external ID, got %v", vehicle["externalIds"])
	}
}

func TestTagResolvesAutomaticNameExternalID(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	ref := "/tags/" + url.PathEscape("samsara.name:Austin Terminal")
	tag := callAPI(t, srv, http.MethodGet, ref, nil).expect(t, http.StatusOK).data(t)
	if recordID(tag) != tagAustinTerminal {
		t.Fatalf("expected samsara.name lookup to resolve Austin Terminal, got %v", tag)
	}
}
