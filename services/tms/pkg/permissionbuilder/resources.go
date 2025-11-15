package permissionbuilder

import (
	"reflect"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/domainregistry"
)

type ResourceRegistry struct {
	resources map[string]permission.Resource
	entities  map[permission.Resource]reflect.Type
}

var defaultRegistry *ResourceRegistry

func init() {
	defaultRegistry = NewResourceRegistry()
	defaultRegistry.RegisterFromDomainRegistry()
}

func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		resources: make(map[string]permission.Resource),
		entities:  make(map[permission.Resource]reflect.Type),
	}
}

func (rr *ResourceRegistry) RegisterFromDomainRegistry() {
	entities := domainregistry.RegisterEntities()

	for _, entity := range entities {
		t := reflect.TypeOf(entity)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		// Convert struct name to snake_case for resource name
		tableName := toSnakeCase(t.Name())
		resourceName := permission.Resource(tableName)

		rr.resources[tableName] = resourceName
		rr.entities[resourceName] = t
	}

	// Add manual overrides for resources without domain entities
	rr.RegisterManual("dashboard", permission.ResourceDashboard)
	rr.RegisterManual("report", permission.ResourceReport)
	rr.RegisterManual("setting", permission.ResourceSetting)
	rr.RegisterManual("audit_entry", permission.ResourceAuditEntry)
	rr.RegisterManual("distance_override", permission.ResourceDistanceOverride)
	rr.RegisterManual("dedicated_lane_suggestion", permission.ResourceDedicatedLaneSuggestion)
	rr.RegisterManual("fiscal_year", permission.ResourceFiscalYear)
	rr.RegisterManual("variable_format", permission.ResourceVariableFormat)
	rr.RegisterManual("variable", permission.ResourceVariable)
	rr.RegisterManual("docker", permission.ResourceDocker)
}

func (rr *ResourceRegistry) RegisterManual(name string, resource permission.Resource) {
	rr.resources[name] = resource
}

func (rr *ResourceRegistry) GetAllResources() []permission.Resource {
	resources := make([]permission.Resource, 0, len(rr.resources))
	for _, res := range rr.resources {
		resources = append(resources, res)
	}
	return resources
}

func (rr *ResourceRegistry) GetResource(name string) (permission.Resource, bool) {
	res, ok := rr.resources[name]
	return res, ok
}

func (rr *ResourceRegistry) GetEntityType(resource permission.Resource) (reflect.Type, bool) {
	t, ok := rr.entities[resource]
	return t, ok
}

func GetAllResources() []permission.Resource {
	return defaultRegistry.GetAllResources()
}

func GetResource(name string) (permission.Resource, bool) {
	return defaultRegistry.GetResource(name)
}

// toSnakeCase converts PascalCase to snake_case
// Handles acronyms properly: APIToken -> api_token, WorkerPTO -> worker_pto
func toSnakeCase(s string) string {
	var result strings.Builder
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Check if current character is uppercase
		if r >= 'A' && r <= 'Z' {
			// Add underscore before uppercase if:
			// 1. Not at the start (i > 0)
			// 2. Previous char is lowercase OR next char is lowercase (not part of acronym)
			if i > 0 {
				prevLower := runes[i-1] >= 'a' && runes[i-1] <= 'z'
				nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'

				// Add underscore if:
				// - Previous is lowercase (start of new word: userId -> user_id)
				// - OR we're at end of acronym (APIToken: after I before T)
				if prevLower || nextLower {
					result.WriteRune('_')
				}
			}
		}

		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}
