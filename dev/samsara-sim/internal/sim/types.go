package sim

import (
	"strings"
)

type Record map[string]any

type Resource string

const (
	ResourceAddresses                Resource = "addresses"
	ResourceAssets                   Resource = "assets"
	ResourceAssetLocation            Resource = "assetLocationStream"
	ResourceDrivers                  Resource = "drivers"
	ResourceRoutes                   Resource = "routes"
	ResourceFormTemplates            Resource = "formTemplates"
	ResourceFormSubmissions          Resource = "formSubmissions"
	ResourceLiveShares               Resource = "liveShares"
	ResourceMessages                 Resource = "messages"
	ResourceWebhooks                 Resource = "webhooks"
	ResourceVehicleStats             Resource = "vehicleStats"
	ResourceHOSClocks                Resource = "hosClocks"
	ResourceHOSLogs                  Resource = "hosLogs"
	ResourceDriverTachograph         Resource = "driverTachograph"
	ResourceVehicleTachograph        Resource = "vehicleTachograph"
	ResourceTags                     Resource = "tags"
	ResourceDriverVehicleAssignments Resource = "driverVehicleAssignments"
	ResourceDriverSignOuts           Resource = "driverSignOuts"
	ResourceDriverWorkflows          Resource = "driverWorkflows"
	ResourceContacts                 Resource = "contacts"
	ResourceDocumentTypes            Resource = "documentTypes"
	ResourceDocuments                Resource = "documents"
	ResourceDocumentPDFs             Resource = "documentPdfs"
	ResourceFormPDFExports           Resource = "formSubmissionPdfExports"
	ResourceUsers                    Resource = "users"
	ResourceDvirs                    Resource = "dvirs"
	ResourceDvirResolutions          Resource = "dvirResolutions"
)

var allResources = []Resource{
	ResourceAddresses,
	ResourceAssets,
	ResourceAssetLocation,
	ResourceDrivers,
	ResourceRoutes,
	ResourceFormTemplates,
	ResourceFormSubmissions,
	ResourceLiveShares,
	ResourceMessages,
	ResourceWebhooks,
	ResourceVehicleStats,
	ResourceHOSClocks,
	ResourceHOSLogs,
	ResourceDriverTachograph,
	ResourceVehicleTachograph,
	ResourceTags,
	ResourceDriverVehicleAssignments,
	ResourceDriverSignOuts,
	ResourceDriverWorkflows,
	ResourceContacts,
	ResourceDocumentTypes,
	ResourceDocuments,
	ResourceDocumentPDFs,
	ResourceFormPDFExports,
	ResourceUsers,
	ResourceDvirs,
	ResourceDvirResolutions,
}

type Fixture struct {
	Addresses                []Record `json:"addresses"`
	Assets                   []Record `json:"assets"`
	AssetLocation            []Record `json:"assetLocationStream"`
	Drivers                  []Record `json:"drivers"`
	Routes                   []Record `json:"routes"`
	FormTemplates            []Record `json:"formTemplates"`
	FormSubmissions          []Record `json:"formSubmissions"`
	LiveShares               []Record `json:"liveShares"`
	Messages                 []Record `json:"messages"`
	Webhooks                 []Record `json:"webhooks"`
	VehicleStats             []Record `json:"vehicleStats"`
	HOSClocks                []Record `json:"hosClocks"`
	HOSLogs                  []Record `json:"hosLogs"`
	DriverTachograph         []Record `json:"driverTachograph"`
	VehicleTachograph        []Record `json:"vehicleTachograph"`
	Tags                     []Record `json:"tags"`
	DriverVehicleAssignments []Record `json:"driverVehicleAssignments"`
	DriverSignOuts           []Record `json:"driverSignOuts"`
	DriverWorkflows          []Record `json:"driverWorkflows"`
	Contacts                 []Record `json:"contacts"`
	DocumentTypes            []Record `json:"documentTypes"`
	Documents                []Record `json:"documents"`
	DocumentPDFs             []Record `json:"documentPdfs"`
	FormPDFExports           []Record `json:"formSubmissionPdfExports"`
	Users                    []Record `json:"users"`
	Dvirs                    []Record `json:"dvirs"`
	DvirResolutions          []Record `json:"dvirResolutions"`
}

var fixtureCollections = map[Resource]func(f *Fixture) *[]Record{
	ResourceAddresses:                func(f *Fixture) *[]Record { return &f.Addresses },
	ResourceAssets:                   func(f *Fixture) *[]Record { return &f.Assets },
	ResourceAssetLocation:            func(f *Fixture) *[]Record { return &f.AssetLocation },
	ResourceDrivers:                  func(f *Fixture) *[]Record { return &f.Drivers },
	ResourceRoutes:                   func(f *Fixture) *[]Record { return &f.Routes },
	ResourceFormTemplates:            func(f *Fixture) *[]Record { return &f.FormTemplates },
	ResourceFormSubmissions:          func(f *Fixture) *[]Record { return &f.FormSubmissions },
	ResourceLiveShares:               func(f *Fixture) *[]Record { return &f.LiveShares },
	ResourceMessages:                 func(f *Fixture) *[]Record { return &f.Messages },
	ResourceWebhooks:                 func(f *Fixture) *[]Record { return &f.Webhooks },
	ResourceVehicleStats:             func(f *Fixture) *[]Record { return &f.VehicleStats },
	ResourceHOSClocks:                func(f *Fixture) *[]Record { return &f.HOSClocks },
	ResourceHOSLogs:                  func(f *Fixture) *[]Record { return &f.HOSLogs },
	ResourceDriverTachograph:         func(f *Fixture) *[]Record { return &f.DriverTachograph },
	ResourceVehicleTachograph:        func(f *Fixture) *[]Record { return &f.VehicleTachograph },
	ResourceTags:                     func(f *Fixture) *[]Record { return &f.Tags },
	ResourceDriverVehicleAssignments: func(f *Fixture) *[]Record { return &f.DriverVehicleAssignments },
	ResourceDriverSignOuts:           func(f *Fixture) *[]Record { return &f.DriverSignOuts },
	ResourceDriverWorkflows:          func(f *Fixture) *[]Record { return &f.DriverWorkflows },
	ResourceContacts:                 func(f *Fixture) *[]Record { return &f.Contacts },
	ResourceDocumentTypes:            func(f *Fixture) *[]Record { return &f.DocumentTypes },
	ResourceDocuments:                func(f *Fixture) *[]Record { return &f.Documents },
	ResourceDocumentPDFs:             func(f *Fixture) *[]Record { return &f.DocumentPDFs },
	ResourceFormPDFExports:           func(f *Fixture) *[]Record { return &f.FormPDFExports },
	ResourceUsers:                    func(f *Fixture) *[]Record { return &f.Users },
	ResourceDvirs:                    func(f *Fixture) *[]Record { return &f.Dvirs },
	ResourceDvirResolutions:          func(f *Fixture) *[]Record { return &f.DvirResolutions },
}

func (f *Fixture) collection(resource Resource) (*[]Record, bool) {
	accessor, ok := fixtureCollections[resource]
	if !ok {
		return nil, false
	}
	return accessor(f), true
}

func (f *Fixture) normalize() {
	for _, resource := range allResources {
		if slot, ok := f.collection(resource); ok {
			*slot = ensureRecordsSlice(*slot)
		}
	}
}

func (f *Fixture) clone() Fixture {
	out := Fixture{}
	for _, resource := range allResources {
		source, sourceOK := f.collection(resource)
		target, targetOK := out.collection(resource)
		if sourceOK && targetOK {
			*target = cloneRecords(*source)
		}
	}
	return out
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
