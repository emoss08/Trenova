package sim

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

type geoJSONDataset struct {
	Type     string           `json:"type"`
	Features []geoJSONFeature `json:"features"`
	Geometry *geoJSONGeometry `json:"geometry,omitempty"`
	Props    map[string]any   `json:"properties,omitempty"`
}

type geoJSONFeature struct {
	Type       string           `json:"type"`
	Properties map[string]any   `json:"properties"`
	Geometry   *geoJSONGeometry `json:"geometry"`
}

type geoJSONGeometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

func ApplyRouteDataset(store *Store, datasetPath string) error {
	if store == nil {
		return ErrRouteDatasetInvalid
	}

	cleanPath := strings.TrimSpace(datasetPath)
	if cleanPath == "" {
		return ErrRouteDatasetPathRequired
	}

	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("read route dataset: %w", err)
	}
	if len(raw) == 0 {
		return ErrRouteDatasetInvalid
	}

	dataset := geoJSONDataset{}
	if err = sonic.Unmarshal(raw, &dataset); err != nil {
		return fmt.Errorf("parse route dataset: %w", err)
	}

	vehicles, err := store.List(ResourceAssets)
	if err != nil {
		return fmt.Errorf("list assets for route dataset: %w", err)
	}
	vehicleIDs := extractVehicleIDs(vehicles)

	records, affectedAssetIDs, err := routeDatasetToAssetLocations(dataset, vehicleIDs)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return fmt.Errorf("%w: no usable line strings", ErrRouteDatasetInvalid)
	}
	if len(affectedAssetIDs) == 0 {
		return fmt.Errorf("%w: no asset mapping", ErrRouteDatasetInvalid)
	}

	existing, err := store.List(ResourceAssetLocation)
	if err != nil {
		return fmt.Errorf("list existing asset locations: %w", err)
	}

	preserved := make([]Record, 0, len(existing))
	for _, record := range existing {
		assetID := nestedString(record, "asset", "id")
		if _, managed := affectedAssetIDs[assetID]; managed {
			continue
		}
		preserved = append(preserved, record)
	}

	combined := append([]Record{}, preserved...)
	combined = append(combined, records...)
	return store.Replace(ResourceAssetLocation, combined)
}

func routeDatasetToAssetLocations(
	dataset geoJSONDataset,
	vehicleIDs []string,
) ([]Record, map[string]struct{}, error) {
	features := dataset.Features
	if strings.EqualFold(strings.TrimSpace(dataset.Type), "Feature") && dataset.Geometry != nil {
		features = []geoJSONFeature{
			{
				Type:       "Feature",
				Geometry:   dataset.Geometry,
				Properties: dataset.Props,
			},
		}
	}
	if len(features) == 0 {
		return nil, nil, ErrRouteDatasetInvalid
	}

	autoAssign := append([]string{}, vehicleIDs...)
	autoIndex := 0
	used := map[string]struct{}{}
	output := make([]Record, 0, 256)
	affected := map[string]struct{}{}
	baseTime := time.Date(2026, time.March, 1, 6, 0, 0, 0, time.UTC)

	for featureIndex, feature := range features {
		coords := parseLineStringCoordinates(feature.Geometry)
		if len(coords) < 2 {
			continue
		}

		assetID := strings.TrimSpace(stringFromAny(feature.Properties["assetId"]))
		if assetID == "" && len(autoAssign) > 0 {
			for loops := 0; loops < len(autoAssign); loops++ {
				candidate := autoAssign[autoIndex%len(autoAssign)]
				autoIndex++
				if _, seen := used[candidate]; seen {
					continue
				}
				assetID = candidate
				break
			}
		}
		if assetID == "" {
			continue
		}
		used[assetID] = struct{}{}
		affected[assetID] = struct{}{}

		speedRaw := floatFromAny(feature.Properties["speedMps"])
		speed := clampFloat64(speedRaw, minRouteSpeedMPS, maxRouteSpeedMPS)
		if speedRaw <= 0 {
			speed = clampFloat64(defaultAssetSpeedMPS, minRouteSpeedMPS, maxRouteSpeedMPS)
		}

		featureBase := baseTime.Add(time.Duration(featureIndex) * 3 * time.Hour)
		for coordinateIndex := range coords {
			current := coords[coordinateIndex]
			next := coords[(coordinateIndex+1)%len(coords)]
			heading := bearingDegrees(current[1], current[0], next[1], next[0])

			record := Record{
				"asset": map[string]any{
					"id": assetID,
				},
				"happenedAtTime": featureBase.
					Add(time.Duration(coordinateIndex) * time.Minute).
					Format(time.RFC3339),
				"location": map[string]any{
					"latitude":       current[1],
					"longitude":      current[0],
					"headingDegrees": int64(heading),
				},
				"speed": map[string]any{
					"gpsSpeedMetersPerSecond": speed,
					"ecuSpeedMetersPerSecond": clampFloat64(
						speed*0.985,
						minRouteSpeedMPS,
						maxRouteSpeedMPS,
					),
				},
			}
			output = append(output, record)
		}
	}

	sort.Slice(output, func(i, j int) bool {
		ti := stringValue(output[i], "happenedAtTime")
		tj := stringValue(output[j], "happenedAtTime")
		if ti == tj {
			return nestedString(output[i], "asset", "id") < nestedString(output[j], "asset", "id")
		}
		return ti < tj
	})

	return output, affected, nil
}

func extractVehicleIDs(assets []Record) []string {
	ids := make([]string, 0, len(assets))
	for _, asset := range assets {
		if !strings.EqualFold(stringValue(asset, "type"), "vehicle") {
			continue
		}
		id := recordID(asset)
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func parseLineStringCoordinates(geometry *geoJSONGeometry) [][2]float64 {
	if geometry == nil || !strings.EqualFold(strings.TrimSpace(geometry.Type), "LineString") {
		return [][2]float64{}
	}

	raw, ok := geometry.Coordinates.([]any)
	if !ok {
		return [][2]float64{}
	}

	coords := make([][2]float64, 0, len(raw))
	for _, item := range raw {
		entries, entriesOK := item.([]any)
		if !entriesOK || len(entries) < 2 {
			continue
		}

		lon := floatFromAny(entries[0])
		lat := floatFromAny(entries[1])
		if !isReasonableCoordinate(lat, lon) {
			continue
		}
		coords = append(coords, [2]float64{lon, lat})
	}
	return coords
}

func isReasonableCoordinate(latitude, longitude float64) bool {
	return latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

func stringFromAny(value any) string {
	candidate, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(candidate)
}
