package sim

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestApplyRouteDatasetReplacesManagedAssets(t *testing.T) {
	t.Parallel()

	store := NewStore(&Fixture{
		Assets: []Record{
			{"id": "veh-1001", "type": "vehicle"},
			{"id": "veh-1002", "type": "vehicle"},
		},
		AssetLocation: []Record{
			{
				"asset": map[string]any{"id": "veh-1001"},
				"location": map[string]any{
					"latitude":  30.0,
					"longitude": -97.0,
				},
			},
			{
				"asset": map[string]any{"id": "veh-1002"},
				"location": map[string]any{
					"latitude":  31.0,
					"longitude": -96.0,
				},
			},
		},
	})

	dataset := `{
		"type": "FeatureCollection",
		"features": [
			{
				"type": "Feature",
				"properties": {"assetId": "veh-1001", "speedMps": 20},
				"geometry": {
					"type": "LineString",
					"coordinates": [[-97.7431,30.2672],[-97.7004,30.3001],[-97.6102,30.3499]]
				}
			}
		]
	}`

	path := filepath.Join(t.TempDir(), "routes.geojson")
	if err := os.WriteFile(path, []byte(dataset), 0o600); err != nil {
		t.Fatalf("write dataset: %v", err)
	}

	if err := ApplyRouteDataset(store, path); err != nil {
		t.Fatalf("apply route dataset: %v", err)
	}

	records, err := store.List(ResourceAssetLocation)
	if err != nil {
		t.Fatalf("list asset locations: %v", err)
	}

	vehOne := 0
	vehTwo := 0
	for _, record := range records {
		switch nestedString(record, "asset", "id") {
		case "veh-1001":
			vehOne++
		case "veh-1002":
			vehTwo++
		}
	}

	if vehOne != 3 {
		t.Fatalf("expected 3 generated records for veh-1001, got %d", vehOne)
	}
	if vehTwo != 1 {
		t.Fatalf("expected preserved record for veh-1002, got %d", vehTwo)
	}
}

func TestApplyRouteDatasetInvalidReturnsError(t *testing.T) {
	t.Parallel()

	store := NewStore(&Fixture{
		Assets: []Record{
			{"id": "veh-1001", "type": "vehicle"},
		},
	})

	path := filepath.Join(t.TempDir(), "bad.geojson")
	if err := os.WriteFile(path, []byte(`{"type":"FeatureCollection","features":[]}`), 0o600); err != nil {
		t.Fatalf("write bad dataset: %v", err)
	}

	err := ApplyRouteDataset(store, path)
	if !errors.Is(err, ErrRouteDatasetInvalid) {
		t.Fatalf("expected route dataset invalid error, got %v", err)
	}
}
