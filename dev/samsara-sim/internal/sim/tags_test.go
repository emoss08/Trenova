package sim

import (
	"net/http"
	"slices"
	"sort"
	"testing"
)

func TestTagIndexParentFilterIncludesDescendants(t *testing.T) {
	t.Parallel()

	index := newTagIndex([]Record{
		{"id": "1", "name": "Root"},
		{"id": "2", "name": "Child", fieldParentTagID: "1", "drivers": []any{"d2"}},
		{"id": "3", "name": "Grandchild", fieldParentTagID: "2", "vehicles": []any{"v3"}},
		{"id": "4", "name": "Other", "drivers": []any{"d4"}, "vehicles": []any{"v3"}},
	})

	tests := []struct {
		name     string
		tagIDs   []string
		parents  []string
		kind     tagMemberKind
		entityID string
		want     bool
	}{
		{name: "no filter matches everything", kind: tagMembersDrivers, entityID: "d9", want: true},
		{name: "tagIds is exact", tagIDs: []string{"1"}, kind: tagMembersDrivers, entityID: "d2"},
		{
			name:     "tagIds direct member",
			tagIDs:   []string{"2"},
			kind:     tagMembersDrivers,
			entityID: "d2",
			want:     true,
		},
		{
			name:     "parent covers child",
			parents:  []string{"1"},
			kind:     tagMembersDrivers,
			entityID: "d2",
			want:     true,
		},
		{
			name:     "parent covers grandchild",
			parents:  []string{"1"},
			kind:     tagMembersVehicles,
			entityID: "v3",
			want:     true,
		},
		{
			name:     "parent excludes unrelated",
			parents:  []string{"1"},
			kind:     tagMembersDrivers,
			entityID: "d4",
		},
		{
			name:     "union of filters",
			tagIDs:   []string{"4"},
			parents:  []string{"2"},
			kind:     tagMembersDrivers,
			entityID: "d4",
			want:     true,
		},
		{name: "kind must match", tagIDs: []string{"3"}, kind: tagMembersDrivers, entityID: "v3"},
		{
			name:     "unknown tag matches nothing",
			tagIDs:   []string{"999"},
			kind:     tagMembersDrivers,
			entityID: "d2",
		},
		{
			name:     "unknown parent matches nothing",
			parents:  []string{"999"},
			kind:     tagMembersDrivers,
			entityID: "d2",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			filter := index.filter(testCase.tagIDs, testCase.parents)
			if got := filter.matches(
				index,
				testCase.kind,
				testCase.entityID,
			); got != testCase.want {
				t.Fatalf("expected %v, got %v", testCase.want, got)
			}
		})
	}
	if !index.isAncestor("1", "3") || index.isAncestor("3", "1") {
		t.Fatal("expected ancestry to follow parentTagId links")
	}
}

func TestTagListAndGetExposeHierarchyAndMembers(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	list := requireStatus(t, srv, http.MethodGet, "/tags?limit=512", nil, http.StatusOK).list(t)
	if len(list) < 20 {
		t.Fatalf("expected the fixture tag tree, got %d tags", len(list))
	}
	austin := requireRecord(t, list, tagAustinTerminal)
	if stringValue(austin, fieldParentTagID) != tagCentralTexas ||
		nestedString(austin, "parentTag", "name") != "Central Texas" {
		t.Fatalf("expected Austin Terminal under Central Texas, got %v", austin)
	}
	for _, kind := range tagMemberKinds {
		if _, ok := austin[string(kind)].([]any); !ok {
			t.Fatalf("expected %s array on tag, got %T", kind, austin[string(kind)])
		}
	}
	vehicles := austin["vehicles"].([]any)
	first, _ := anyAsMap(vehicles[0])
	if stringValue(Record(first), "id") == "" || stringValue(Record(first), "name") == "" {
		t.Fatalf("expected tagged objects with id and name, got %v", first)
	}

	byExternal := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/tags/tmsTagId:terminal-aus",
		nil,
		http.StatusOK,
	).data(t)
	if recordID(byExternal) != tagAustinTerminal {
		t.Fatalf(
			"expected external ID lookup to resolve Austin Terminal, got %s",
			recordID(byExternal),
		)
	}
	callAPI(t, srv, http.MethodGet, "/tags/9999999", nil).expectError(t, http.StatusNotFound, "tag")
	callAPI(
		t,
		srv,
		http.MethodGet,
		"/tags?limit=0",
		nil,
	).expectError(t, http.StatusBadRequest, "limit")
}

func TestTagCreateValidatesAndAssignsServerID(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	tests := []struct {
		name     string
		body     map[string]any
		fragment string
	}{
		{name: "name required", body: map[string]any{}, fragment: "name"},
		{
			name:     "name length",
			body:     map[string]any{"name": string(make([]byte, 192))},
			fragment: "191",
		},
		{
			name:     "unknown parent",
			body:     map[string]any{"name": "X", "parentTagId": "1"},
			fragment: "parent tag",
		},
		{
			name:     "duplicate sibling",
			body:     map[string]any{"name": "austin terminal", "parentTagId": tagCentralTexas},
			fragment: "already exists",
		},
		{
			name:     "unknown driver",
			body:     map[string]any{"name": "X", "drivers": []any{"42"}},
			fragment: "driver",
		},
		{
			name:     "vehicle listed as asset",
			body:     map[string]any{"name": "X", "assets": []any{fixtureTruck1001}},
			fragment: "asset",
		},
		{
			name:     "trailer listed as vehicle",
			body:     map[string]any{"name": "X", "vehicles": []any{fixtureTrailer2042}},
			fragment: "vehicle",
		},
		{
			name:     "unknown machine",
			body:     map[string]any{"name": "X", "machines": []any{"5"}},
			fragment: "machine",
		},
		{
			name: "external id in use",
			body: map[string]any{
				"name":        "X",
				"externalIds": map[string]any{"tmsTagId": "region-tx"},
			},
			fragment: "already assigned",
		},
		{
			name: "reserved external id prefix",
			body: map[string]any{
				"name":        "X",
				"externalIds": map[string]any{"samsara.vin": "1"},
			},
			fragment: "reserved",
		},
	}
	for _, testCase := range tests {
		callAPI(t, srv, http.MethodPost, "/tags", testCase.body).
			expectError(t, http.StatusBadRequest, testCase.fragment)
	}

	created := requireStatus(t, srv, http.MethodPost, "/tags", map[string]any{
		"id":          "client-id",
		"name":        "Night Shift",
		"parentTagId": tagDriverPrograms,
		"drivers":     []any{fixtureDriverAlex},
		"vehicles":    []any{fixtureTruck1001},
		"assets":      []any{fixtureTrailer2042},
		"addresses":   []any{fixtureAustinYard},
		"externalIds": map[string]any{"tmsTagId": "night-shift"},
	}, http.StatusOK).data(t)
	tagID := recordID(created)
	if tagID == "client-id" || mustNumericID(t, "tag", tagID) <= 342654 {
		t.Fatalf("expected a server-assigned tag id above the fixture, got %s", tagID)
	}
	driver := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/drivers/"+fixtureDriverAlex,
		nil,
		http.StatusOK,
	).data(t)
	if !slices.Contains(tagIDsOf(t, driver), tagID) {
		t.Fatalf("expected the driver to report the new tag, got %v", tagIDsOf(t, driver))
	}
	vehicle := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/vehicles/"+fixtureTruck1001,
		nil,
		http.StatusOK,
	).data(t)
	if !slices.Contains(tagIDsOf(t, vehicle), tagID) {
		t.Fatalf("expected the vehicle to report the new tag, got %v", tagIDsOf(t, vehicle))
	}
}

func TestTagPatchPutAndDelete(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{})
	callAPI(
		t,
		srv,
		http.MethodPatch,
		"/tags/"+tagTexasOperations,
		map[string]any{"parentTagId": tagAustinTerminal},
	).
		expectError(t, http.StatusBadRequest, "descendants")
	callAPI(
		t,
		srv,
		http.MethodPatch,
		"/tags/"+tagAustinTerminal,
		map[string]any{"parentTagId": tagAustinTerminal},
	).
		expectError(t, http.StatusBadRequest, "itself")

	patched := requireStatus(
		t,
		srv,
		http.MethodPatch,
		"/tags/tmsTagId:terminal-aus",
		map[string]any{
			"name": "Austin North Terminal",
		},
		http.StatusOK,
	).data(t)
	if stringValue(patched, "name") != "Austin North Terminal" ||
		len(patched["vehicles"].([]any)) == 0 {
		t.Fatalf("expected PATCH to rename and keep members, got %v", patched)
	}
	moved := requireStatus(t, srv, http.MethodPatch, "/tags/"+tagAustinTerminal, map[string]any{
		"parentTagId": nil,
	}, http.StatusOK).data(t)
	if _, has := moved[fieldParentTagID]; has {
		t.Fatalf("expected null parentTagId to move the tag to the root, got %v", moved)
	}

	replaced := requireStatus(t, srv, http.MethodPut, "/tags/"+tagAustinTerminal, map[string]any{
		"name":        "Austin Terminal",
		"parentTagId": tagCentralTexas,
		"drivers":     []any{fixtureDriverAlex},
	}, http.StatusOK).data(t)
	if len(replaced["vehicles"].([]any)) != 0 || len(replaced["drivers"].([]any)) != 1 {
		t.Fatalf("expected PUT to replace membership, got %v", replaced)
	}
	if nestedString(replaced, fieldExternalIDs, "tmsTagId") != "terminal-aus" {
		t.Fatalf(
			"expected PUT to keep externalIds it cannot set, got %v",
			replaced[fieldExternalIDs],
		)
	}
	ignored := requireStatus(t, srv, http.MethodPut, "/tags/"+tagAustinTerminal, map[string]any{
		"parentTagId": tagCentralTexas,
		"externalIds": map[string]any{"a": "b"},
	}, http.StatusOK).data(t)
	if nestedString(ignored, fieldExternalIDs, "a") != "" ||
		stringValue(ignored, "name") != "Austin Terminal" {
		t.Fatalf("expected PUT to ignore externalIds and keep the name, got %v", ignored)
	}

	requireStatus(t, srv, http.MethodDelete, "/tags/"+tagCentralTexas, nil, http.StatusNoContent)
	for _, id := range []string{tagCentralTexas, tagAustinTerminal, "342434"} {
		callAPI(t, srv, http.MethodGet, "/tags/"+id, nil).expectError(t, http.StatusNotFound, "tag")
	}
	driver := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/fleet/drivers/"+fixtureDriverAlex,
		nil,
		http.StatusOK,
	).data(t)
	if _, has := driver["trailerGroupTag"]; has {
		t.Fatalf(
			"expected deleted group tags to clear from drivers, got %v",
			driver["trailerGroupTag"],
		)
	}
	callAPI(
		t,
		srv,
		http.MethodDelete,
		"/tags/"+tagCentralTexas,
		nil,
	).expectError(t, http.StatusNotFound, "tag")
}

func TestTagFiltersApplyAcrossFleetEndpoints(t *testing.T) {
	t.Parallel()

	srv := newFleetTestServer(t, fleetServerOptions{Dataset: true})
	tests := []struct {
		target string
		want   []string
	}{
		{
			target: "/fleet/vehicles?parentTagIds=" + tagNorthTexas,
			want:   []string{fixtureTruck1002, fixtureTruck1005, "281474977075973"},
		},
		{
			target: "/fleet/vehicles?tagIds=" + tagAustinTerminal,
			want:   []string{fixtureTruck1001, "281474977075860"},
		},
		{
			target: "/fleet/vehicles/stats?types=gps&parentTagIds=" + tagNorthTexas,
			want:   []string{fixtureTruck1002, fixtureTruck1005, "281474977075973"},
		},
		{
			target: "/fleet/trailers?tagIds=" + tagReeferTrailers,
			want: []string{
				fixtureTrailer2042,
				"281474979348663",
				fixtureTrailer2049,
				fixtureTrailer2053,
			},
		},
		{
			target: "/fleet/drivers?tagIds=" + tagHazmatCertified,
			want:   []string{fixtureDriverAlex, "1655012", "1655134", "1655230"},
		},
		{
			target: "/fleet/drivers?parentTagIds=" + tagNorthTexas,
			want:   []string{fixtureDriverJordan, "1655069", "1655192"},
		},
		{
			target: "/fleet/equipment?parentTagIds=" + tagAustinTerminal,
			want:   []string{"281474980315252", fixtureYardSpotter},
		},
	}
	for _, testCase := range tests {
		records := requireStatus(
			t,
			srv,
			http.MethodGet,
			testCase.target,
			nil,
			http.StatusOK,
		).list(t)
		got := listIDs(records)
		sort.Strings(got)
		want := append([]string(nil), testCase.want...)
		sort.Strings(want)
		if !slices.Equal(got, want) {
			t.Fatalf("%s: expected %v, got %v", testCase.target, want, got)
		}
	}
	assets := requireStatus(
		t,
		srv,
		http.MethodGet,
		"/assets?parentTagIds="+tagAustinTerminal+"&includeTags=true",
		nil,
		http.StatusOK,
	).list(t)
	for _, asset := range assets {
		tags := tagIDsOf(t, asset)
		if !slices.Contains(tags, tagAustinTerminal) {
			t.Fatalf("expected only Austin Terminal assets, got %s with %v", recordID(asset), tags)
		}
	}
	if len(assets) != 7 {
		t.Fatalf(
			"expected the 2 trucks, 3 trailers and 2 equipment units at Austin, got %d",
			len(assets),
		)
	}
}
