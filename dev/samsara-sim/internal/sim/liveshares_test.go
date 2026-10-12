package sim

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

const fixtureLiveShareTexas = "kq3m8vz2pt7xw4nb6rd"

func TestLiveShareListFiltersAndShape(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})

	all := requireStatus(t, srv, http.MethodGet, "/live-shares", nil, http.StatusOK).list(t)
	if len(all) != 2 {
		t.Fatalf("expected 2 fixture live shares, got %d", len(all))
	}
	texas := requireRecord(t, all, fixtureLiveShareTexas)
	config := mapOf(texas["assetsLocationLinkConfig"])
	tags := listOf(config["tags"])
	if len(tags) != 1 || mapOf(tags[0])["id"] != tagTexasOperations ||
		mapOf(tags[0])["name"] != "Texas Operations" {
		t.Fatalf("expected the by-tag config to render tag mini-objects, got %v", config)
	}
	if _, raw := config["tagIds"]; raw {
		t.Fatalf("expected tagIds to stay internal, got %v", config)
	}
	if !strings.HasPrefix(
		stringValue(texas, "liveSharingUrl"),
		"https://cloud.samsara.com/o/20936/fleet/viewer/",
	) {
		t.Fatalf("expected a Samsara viewer URL, got %v", texas["liveSharingUrl"])
	}

	near := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/live-shares?type=assetsNearLocation",
		nil,
		http.StatusOK,
	).
		list(t)
	if len(near) != 1 ||
		mapOf(near[0]["assetsNearLocationLinkConfig"])["addressId"] != fixtureAustinYard {
		t.Fatalf("expected the by-location fixture share, got %v", near)
	}
	byID := requireStatus(t, srv, http.MethodGet, "/live-shares?ids="+fixtureLiveShareTexas, nil,
		http.StatusOK).list(t)
	if len(byID) != 1 {
		t.Fatalf("expected the ids filter to select one share, got %d", len(byID))
	}
	callAPI(t, srv, http.MethodGet, "/live-shares?type=everything", nil).
		expectError(t, http.StatusBadRequest, "type")
	callAPI(t, srv, http.MethodGet, "/live-shares?limit=101", nil).
		expectError(t, http.StatusBadRequest, "limit")
	page := requireStatus(t, srv, http.MethodGet, "/live-shares?limit=1", nil, http.StatusOK)
	if page.pagination(t)["hasNextPage"] != true {
		t.Fatalf("expected a second page, got %s", page.Body)
	}
}

func TestLiveShareCreateValidation(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	future := fleetTestTime.Add(48 * time.Hour).Format(time.RFC3339)
	past := fleetTestTime.Add(-time.Hour).Format(time.RFC3339)

	cases := []struct {
		name     string
		body     map[string]any
		fragment string
	}{
		{"missing name", map[string]any{"type": "assetsLocation"}, "name is required"},
		{"missing type", map[string]any{"name": "x"}, "type is required"},
		{"bad type", map[string]any{"name": "x", "type": "everything"}, "type"},
		{
			"missing config",
			map[string]any{"name": "x", "type": "assetsLocation"},
			"is required for type",
		},
		{
			"wrong config",
			map[string]any{
				"name": "x", "type": "assetsLocation",
				"assetsLocationLinkConfig":     map[string]any{"assetId": fixtureTruck1001},
				"assetsNearLocationLinkConfig": map[string]any{"addressId": fixtureAustinYard},
			},
			"does not apply",
		},
		{
			"asset and tags",
			map[string]any{
				"name": "x",
				"type": "assetsLocation",
				"assetsLocationLinkConfig": map[string]any{
					"assetId": fixtureTruck1001, "tagIds": []any{tagAustinTerminal},
				},
			},
			"exactly one of assetId or tagIds",
		},
		{
			"unknown asset",
			map[string]any{
				"name": "x",
				"type": "assetsLocation",
				"assetsLocationLinkConfig": map[string]any{
					"assetId": "1",
				},
			},
			"asset",
		},
		{
			"unknown tag",
			map[string]any{
				"name": "x",
				"type": "assetsLocation",
				"assetsLocationLinkConfig": map[string]any{
					"tagIds": []any{"1"},
				},
			},
			"tag",
		},
		{
			"incomplete location",
			map[string]any{
				"name": "x",
				"type": "assetsLocation",
				"assetsLocationLinkConfig": map[string]any{
					"assetId": fixtureTruck1001,
					"location": map[string]any{
						"name":      "Dock",
						"latitude":  30.1,
						"longitude": -97.7,
					},
				},
			},
			"formattedAddress is required",
		},
		{
			"unknown address",
			map[string]any{
				"name": "x",
				"type": "assetsNearLocation",
				"assetsNearLocationLinkConfig": map[string]any{
					"addressId": "nope:1",
				},
			},
			"address",
		},
		{
			"unknown route",
			map[string]any{
				"name": "x",
				"type": "assetsOnRoute",
				"assetsOnRouteLinkConfig": map[string]any{
					"recurringRouteId": "1",
				},
			},
			"recurring route",
		},
		{
			"route description",
			map[string]any{
				"name": "x", "type": "assetsOnRoute", "description": "d",
				"assetsOnRouteLinkConfig": map[string]any{"recurringRouteId": "4129806431"},
			},
			"does not apply to assetsOnRoute",
		},
		{
			"past expiry",
			map[string]any{
				"name": "x", "type": "assetsLocation", "expiresAtTime": past,
				"assetsLocationLinkConfig": map[string]any{"assetId": fixtureTruck1001},
			},
			"in the past",
		},
		{
			"long description",
			map[string]any{
				"name": "x", "type": "assetsLocation", "description": strings.Repeat("d", 256),
				"assetsLocationLinkConfig": map[string]any{"assetId": fixtureTruck1001},
				"expiresAtTime":            future,
			},
			"at most 255",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			callAPI(t, srv, http.MethodPost, "/live-shares", testCase.body).
				expectError(t, http.StatusBadRequest, testCase.fragment)
		})
	}
}

func TestLiveShareLifecycle(t *testing.T) {
	srv := newFleetTestServer(t, fleetServerOptions{})
	requireStatus(t, srv, http.MethodPatch, "/addresses/"+fixtureAustinYard, map[string]any{
		"externalIds": map[string]any{"siteId": "AUSYARD"},
	}, http.StatusOK)

	expires := fleetTestTime.Add(2 * time.Hour).Format(time.RFC3339)
	created := requireStatus(t, srv, http.MethodPost, "/live-shares", map[string]any{
		"name":          "Truck 1001 to Alamo Grocers",
		"type":          "assetsLocation",
		"description":   "ETA for the San Antonio receiver",
		"expiresAtTime": expires,
		"assetsLocationLinkConfig": map[string]any{
			"assetId": fixtureTruck1001,
			"location": map[string]any{
				"name":             "San Antonio Customer Drop",
				"formattedAddress": "455 Commerce St, San Antonio, TX 78205",
				"latitude":         29.4241,
				"longitude":        -98.4936,
			},
		},
	}, http.StatusOK).data(t)
	shareID := recordID(created)
	if len(shareID) != 19 || created["expiresAtTime"] != expires {
		t.Fatalf("expected a 19-character token ID and the expiry, got %v", created)
	}
	if !strings.HasPrefix(stringValue(created, "liveSharingUrl"),
		"https://cloud.samsara.com/o/20936/fleet/viewer/asset/") {
		t.Fatalf("expected an asset viewer URL, got %v", created["liveSharingUrl"])
	}
	location := mapOf(mapOf(created["assetsLocationLinkConfig"])["location"])
	if location["name"] != "San Antonio Customer Drop" || location["latitude"] != 29.4241 {
		t.Fatalf("expected the destination location echoed, got %v", created)
	}

	near := requireStatus(t, srv, http.MethodPost, "/live-shares", map[string]any{
		"name": "Yard watch",
		"type": "assetsNearLocation",
		"assetsNearLocationLinkConfig": map[string]any{
			"addressId": "siteId:AUSYARD",
		},
	}, http.StatusOK).data(t)
	if mapOf(near["assetsNearLocationLinkConfig"])["addressId"] != fixtureAustinYard ||
		!strings.Contains(stringValue(near, "liveSharingUrl"), "/fleet/viewer/address/") {
		t.Fatalf("expected the external address ID resolved, got %v", near)
	}
	route := requireStatus(t, srv, http.MethodPost, "/live-shares", map[string]any{
		"name":                    "Texas Route 1 riders",
		"type":                    "assetsOnRoute",
		"assetsOnRouteLinkConfig": map[string]any{"recurringRouteId": "4129806431"},
	}, http.StatusOK).data(t)
	if _, has := route["expiresAtTime"]; has {
		t.Fatalf("expected a link without expiry to omit expiresAtTime, got %v", route)
	}

	target := "/live-shares?id=" + url.QueryEscape(shareID)
	callAPI(t, srv, http.MethodPatch, target, map[string]any{"description": "x"}).
		expectError(t, http.StatusBadRequest, "name is required")
	patched := requireStatus(t, srv, http.MethodPatch, target, map[string]any{
		"name":          "Truck 1001 ETA",
		"expiresAtTime": nil,
		"type":          "assetsOnRoute",
	}, http.StatusOK).data(t)
	if patched["name"] != "Truck 1001 ETA" || patched["type"] != "assetsLocation" ||
		patched["description"] != "ETA for the San Antonio receiver" {
		t.Fatalf("expected only name and expiry to change, got %v", patched)
	}
	if _, has := patched["expiresAtTime"]; has {
		t.Fatalf("expected the expiry cleared, got %v", patched)
	}
	callAPI(t, srv, http.MethodPatch, "/live-shares?id="+recordID(route), map[string]any{
		"name": "x", "description": "nope",
	}).expectError(t, http.StatusBadRequest, "assetsOnRoute")
	callAPI(t, srv, http.MethodPatch, "/live-shares", map[string]any{"name": "x"}).
		expectError(t, http.StatusBadRequest, "id")

	soon := requireStatus(t, srv, http.MethodPost, "/live-shares", map[string]any{
		"name":                     "Short lived",
		"type":                     "assetsLocation",
		"expiresAtTime":            fleetTestTime.Add(time.Minute).Format(time.RFC3339),
		"assetsLocationLinkConfig": map[string]any{"tagIds": []any{tagAustinTerminal}},
	}, http.StatusOK).data(t)
	srv.clock.SetTime(fleetTestTime.Add(2 * time.Minute))
	listed := requireStatus(t, srv, http.MethodGet, "/live-shares", nil, http.StatusOK).list(t)
	if _, visible := idSet(listed)[recordID(soon)]; visible {
		t.Fatal("expected expired links to drop out of the listing")
	}
	requireStatus(t, srv, http.MethodPatch, "/live-shares?id="+recordID(soon),
		map[string]any{"name": "late"}, http.StatusNotFound)
	requireStatus(
		t,
		srv,
		http.MethodDelete,
		"/live-shares?id="+recordID(soon),
		nil,
		http.StatusNotFound,
	)

	requireStatus(t, srv, http.MethodDelete, target, nil, http.StatusNoContent)
	requireStatus(t, srv, http.MethodDelete, target, nil, http.StatusNotFound)
}
