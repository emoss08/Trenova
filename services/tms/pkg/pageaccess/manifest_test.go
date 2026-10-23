package pageaccess_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/pageaccess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixture = `{
  "version": "sha256:test",
  "screens": [
    {"path": "/shipments", "name": "Shipments", "requires": [{"resource": "shipment", "operation": "read"}],
     "operations": ["ShipmentList", "ShipmentDetail"], "endpoints": [], "reads": ["shipment"],
     "references": ["worker"], "declared": ["customer"]},
    {"path": "/shipments/:id", "name": "", "requires": [], "operations": [], "endpoints": [],
     "reads": [], "references": [], "declared": []},
    {"path": "/shipments/new", "name": "", "requires": [], "operations": [],
     "endpoints": [{"method": "POST", "path": "/shipments/", "requires": [{"resource": "customer", "operation": "read"}]}],
     "reads": [], "references": [], "declared": []}
  ],
  "operations": [
    {"name": "ShipmentList", "type": "query", "reads": ["shipment"],
     "linked": [{"kind": "customer", "fields": ["name"]}, {"kind": "worker", "fields": ["firstName"]}], "declared": []},
    {"name": "ShipmentDetail", "type": "query", "reads": ["shipment"],
     "linked": [{"kind": "customer", "fields": ["creditLimit", "name"]}], "declared": ["customer"]}
  ]
}`

func load(t *testing.T) *pageaccess.Manifest {
	t.Helper()
	manifest, err := pageaccess.Load([]byte(fixture))
	require.NoError(t, err)
	return manifest
}

func TestScreen_FindsTheRouteALocationShows(t *testing.T) {
	t.Parallel()

	manifest := load(t)
	for location, want := range map[string]string{
		"/shipments":           "/shipments",
		"/shipments/":          "/shipments",
		"/shipments?tab=open":  "/shipments",
		"/shipments/shp_123":   "/shipments/:id",
		"/shipments/new":       "/shipments/new",
		"/shipments/new#notes": "/shipments/new",
	} {
		screen, ok := manifest.Screen(location)
		require.True(t, ok, location)
		assert.Equal(t, want, screen.Path, location)
	}
	_, ok := manifest.Screen("/nowhere/at/all")
	assert.False(t, ok)
}

func TestLinked_MergesEveryOperationsFieldsByKind(t *testing.T) {
	t.Parallel()

	manifest := load(t)
	screen, _ := manifest.Screen("/shipments")
	assert.Equal(t, []pageaccess.LinkedKind{
		{Kind: "customer", Fields: []string{"creditLimit", "name"}},
		{Kind: "worker", Fields: []string{"firstName"}},
	}, manifest.Linked(screen))
}

func TestScreensUsing_NamesTheStrongestUseOfAKind(t *testing.T) {
	t.Parallel()

	manifest := load(t)
	uses := func(kind string) map[string]pageaccess.Use {
		out := map[string]pageaccess.Use{}
		for _, use := range manifest.ScreensUsing(kind) {
			out[use.Screen.Path] = use.Use
		}
		return out
	}
	assert.Equal(t, map[string]pageaccess.Use{"/shipments": pageaccess.UseRequired}, uses("shipment"))
	assert.Equal(t, map[string]pageaccess.Use{
		"/shipments":     pageaccess.UseDeclared,
		"/shipments/new": pageaccess.UseRequired,
	}, uses("customer"))
	assert.Equal(t, map[string]pageaccess.Use{"/shipments": pageaccess.UseReference}, uses("worker"))
	assert.Empty(t, uses("invoice"))
}

func TestDefault_LoadsTheGeneratedManifest(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, pageaccess.Default.Screens)
	screen, ok := pageaccess.Default.Screen("/admin/roles")
	require.True(t, ok)
	assert.Equal(t, "/admin/roles", screen.Path)
}
