package customfield

import "sort"

var supportedResourceTypes = map[string]bool{
	"trailer":  true,
	"worker":   true,
	"shipment": true,
	"customer": true,
	"location": true,
	"tractor":  true,
}

func IsResourceTypeSupported(resourceType string) bool {
	return supportedResourceTypes[resourceType]
}

func GetSupportedResourceTypes() []string {
	types := make([]string, 0, len(supportedResourceTypes))
	for t := range supportedResourceTypes {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func RegisterResourceType(resourceType string) {
	supportedResourceTypes[resourceType] = true
}
