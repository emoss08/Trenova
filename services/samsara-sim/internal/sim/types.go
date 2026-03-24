package sim

import (
	"strconv"
	"strings"
)

type Record map[string]any

type Resource string

const (
	ResourceAddresses         Resource = "addresses"
	ResourceAssets            Resource = "assets"
	ResourceAssetLocation     Resource = "assetLocationStream"
	ResourceDrivers           Resource = "drivers"
	ResourceRoutes            Resource = "routes"
	ResourceFormTemplates     Resource = "formTemplates"
	ResourceFormSubmissions   Resource = "formSubmissions"
	ResourceLiveShares        Resource = "liveShares"
	ResourceMessages          Resource = "messages"
	ResourceWebhooks          Resource = "webhooks"
	ResourceVehicleStats      Resource = "vehicleStats"
	ResourceHOSClocks         Resource = "hosClocks"
	ResourceHOSLogs           Resource = "hosLogs"
	ResourceDriverTachograph  Resource = "driverTachograph"
	ResourceVehicleTachograph Resource = "vehicleTachograph"
)

type Fixture struct {
	Addresses         []Record `json:"addresses"`
	Assets            []Record `json:"assets"`
	AssetLocation     []Record `json:"assetLocationStream"`
	Drivers           []Record `json:"drivers"`
	Routes            []Record `json:"routes"`
	FormTemplates     []Record `json:"formTemplates"`
	FormSubmissions   []Record `json:"formSubmissions"`
	LiveShares        []Record `json:"liveShares"`
	Messages          []Record `json:"messages"`
	Webhooks          []Record `json:"webhooks"`
	VehicleStats      []Record `json:"vehicleStats"`
	HOSClocks         []Record `json:"hosClocks"`
	HOSLogs           []Record `json:"hosLogs"`
	DriverTachograph  []Record `json:"driverTachograph"`
	VehicleTachograph []Record `json:"vehicleTachograph"`
}

func (f *Fixture) normalize() {
	f.Addresses = ensureRecordsSlice(f.Addresses)
	f.Assets = ensureRecordsSlice(f.Assets)
	f.AssetLocation = ensureRecordsSlice(f.AssetLocation)
	f.Drivers = ensureRecordsSlice(f.Drivers)
	f.Routes = ensureRecordsSlice(f.Routes)
	f.FormTemplates = ensureRecordsSlice(f.FormTemplates)
	f.FormSubmissions = ensureRecordsSlice(f.FormSubmissions)
	f.LiveShares = ensureRecordsSlice(f.LiveShares)
	f.Messages = ensureRecordsSlice(f.Messages)
	f.Webhooks = ensureRecordsSlice(f.Webhooks)
	f.VehicleStats = ensureRecordsSlice(f.VehicleStats)
	f.HOSClocks = ensureRecordsSlice(f.HOSClocks)
	f.HOSLogs = ensureRecordsSlice(f.HOSLogs)
	f.DriverTachograph = ensureRecordsSlice(f.DriverTachograph)
	f.VehicleTachograph = ensureRecordsSlice(f.VehicleTachograph)
}

func (f *Fixture) clone() Fixture {
	return Fixture{
		Addresses:         cloneRecords(f.Addresses),
		Assets:            cloneRecords(f.Assets),
		AssetLocation:     cloneRecords(f.AssetLocation),
		Drivers:           cloneRecords(f.Drivers),
		Routes:            cloneRecords(f.Routes),
		FormTemplates:     cloneRecords(f.FormTemplates),
		FormSubmissions:   cloneRecords(f.FormSubmissions),
		LiveShares:        cloneRecords(f.LiveShares),
		Messages:          cloneRecords(f.Messages),
		Webhooks:          cloneRecords(f.Webhooks),
		VehicleStats:      cloneRecords(f.VehicleStats),
		HOSClocks:         cloneRecords(f.HOSClocks),
		HOSLogs:           cloneRecords(f.HOSLogs),
		DriverTachograph:  cloneRecords(f.DriverTachograph),
		VehicleTachograph: cloneRecords(f.VehicleTachograph),
	}
}

func ensureRecordsSlice(in []Record) []Record {
	if in == nil {
		return []Record{}
	}
	return in
}

func cloneRecord(in Record) Record {
	out := make(Record, len(in))
	for key, value := range in {
		out[key] = cloneAny(value)
	}
	return out
}

func cloneRecords(in []Record) []Record {
	if len(in) == 0 {
		return []Record{}
	}

	out := make([]Record, 0, len(in))
	for _, record := range in {
		out = append(out, cloneRecord(record))
	}
	return out
}

func cloneAny(in any) any {
	switch typed := in.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = cloneAny(value)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, value := range typed {
			out = append(out, cloneAny(value))
		}
		return out
	default:
		return in
	}
}

func recordID(record Record) string {
	rawID, ok := record["id"]
	if !ok {
		return ""
	}
	id, ok := rawID.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(id)
}

func parseTrailingInt(prefix, value string) int {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return 0
	}
	if !strings.HasPrefix(clean, prefix+"-") {
		return 0
	}

	number := strings.TrimPrefix(clean, prefix+"-")
	parsed, err := strconv.Atoi(number)
	if err != nil {
		return 0
	}
	return parsed
}

func mustResourcePrefix(resource Resource) string {
	switch resource {
	case ResourceAddresses:
		return "addr"
	case ResourceAssets:
		return "asset"
	case ResourceAssetLocation:
		return "asset-loc"
	case ResourceDrivers:
		return "drv"
	case ResourceRoutes:
		return "route"
	case ResourceFormTemplates:
		return "form-template"
	case ResourceFormSubmissions:
		return "form-sub"
	case ResourceLiveShares:
		return "ls"
	case ResourceWebhooks:
		return "wh"
	case ResourceVehicleStats:
		return "veh"
	case ResourceHOSClocks:
		return "hos-clock"
	case ResourceHOSLogs:
		return "hos-log"
	case ResourceDriverTachograph:
		return "driver-tacho"
	case ResourceVehicleTachograph:
		return "vehicle-tacho"
	case ResourceMessages:
		return "msg"
	default:
		return "obj"
	}
}

func mergePatch(target, patch Record) {
	for key, value := range patch {
		if key == "id" {
			continue
		}
		if value == nil {
			delete(target, key)
			continue
		}

		existing, ok := target[key]
		if !ok {
			target[key] = cloneAny(value)
			continue
		}

		targetMap, targetIsMap := existing.(map[string]any)
		patchMap, patchIsMap := value.(map[string]any)
		if targetIsMap && patchIsMap {
			target[key] = mergePatchMap(targetMap, patchMap)
			continue
		}
		target[key] = cloneAny(value)
	}
}

func mergePatchMap(target, patch map[string]any) map[string]any {
	cloned := map[string]any{}
	for key, value := range target {
		cloned[key] = cloneAny(value)
	}
	for key, value := range patch {
		if value == nil {
			delete(cloned, key)
			continue
		}

		existing, ok := cloned[key]
		if !ok {
			cloned[key] = cloneAny(value)
			continue
		}

		existingMap, existingIsMap := existing.(map[string]any)
		patchMap, patchIsMap := value.(map[string]any)
		if existingIsMap && patchIsMap {
			cloned[key] = mergePatchMap(existingMap, patchMap)
			continue
		}
		cloned[key] = cloneAny(value)
	}
	return cloned
}
