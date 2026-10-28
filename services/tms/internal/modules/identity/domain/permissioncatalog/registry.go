package permissioncatalog

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingperms"
	"github.com/emoss08/trenova/internal/modules/identity/domain/adminperms"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentperms"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingperms"
	"github.com/emoss08/trenova/internal/modules/identity/domain/carrierperms"
	"github.com/emoss08/trenova/internal/modules/identity/domain/complianceperms"
	"github.com/emoss08/trenova/internal/modules/identity/domain/fleetperms"
	"github.com/emoss08/trenova/internal/modules/identity/domain/payrollperms"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/reportperms"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentperms"
	"github.com/emoss08/trenova/internal/modules/identity/domain/workerperms"
)

type Registry struct {
	mu        sync.RWMutex
	resources map[string]*permissionspec.ResourceDefinition
	children  map[string][]string
}

func NewRegistry() *Registry {
	r := &Registry{
		resources: make(map[string]*permissionspec.ResourceDefinition),
		children:  make(map[string][]string),
	}
	r.registerAll()
	return r
}

func NewEmptyRegistry() *Registry {
	return &Registry{
		resources: make(map[string]*permissionspec.ResourceDefinition),
		children:  make(map[string][]string),
	}
}

func (r *Registry) Register(def *permissionspec.ResourceDefinition) error {
	if def.Resource == "" {
		return errors.New("resource name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.resources[def.Resource]; exists {
		return fmt.Errorf("resource %s already registered", def.Resource)
	}

	if def.DefaultSensitivity == "" {
		def.DefaultSensitivity = permission.SensitivityInternal
	}
	def.Operations = withImplications(def.Operations)

	r.resources[def.Resource] = def

	if def.ParentResource != "" {
		r.children[def.ParentResource] = append(r.children[def.ParentResource], def.Resource)
	}

	return nil
}

func (r *Registry) Get(resource string) (*permissionspec.ResourceDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.resources[resource]
	return def, ok
}

func (r *Registry) GetChildren(parent string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.children[parent]
}

func (r *Registry) GetEffectiveResource(resource string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if def, ok := r.resources[resource]; ok && def.ParentResource != "" {
		return def.ParentResource
	}
	return resource
}

func (r *Registry) All() []*permissionspec.ResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*permissionspec.ResourceDefinition, 0, len(r.resources))
	for _, def := range r.resources {
		result = append(result, def)
	}
	return result
}

func (r *Registry) GetFieldSensitivity(resource, field string) permission.FieldSensitivity {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.resources[resource]
	if !ok {
		return permission.SensitivityInternal
	}
	if sens, sensOK := def.FieldSensitivities[field]; sensOK {
		return sens
	}
	if inReferenceView(def, field) {
		return permission.SensitivityInternal
	}
	return def.DefaultSensitivity
}

func (r *Registry) GetOperationsForResource(resource string) []permission.Operation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.resources[resource]
	if !ok {
		return nil
	}

	ops := make([]permission.Operation, 0, len(def.Operations))
	for _, opDef := range def.Operations {
		ops = append(ops, opDef.Operation)
	}
	return ops
}

func (r *Registry) ExpandCompositeOperation(resource, compositeName string) []permission.Operation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.resources[resource]
	if !ok {
		return nil
	}

	if ops, opsOK := def.CompositeOps[compositeName]; opsOK {
		return ops
	}
	return nil
}

func (r *Registry) Declares(resource string, operation permission.Operation) bool {
	if r.declaresOn(resource, operation) {
		return true
	}
	parent := r.GetEffectiveResource(resource)
	return parent != resource && r.declaresOn(parent, operation)
}

func (r *Registry) declaresOn(resource string, operation permission.Operation) bool {
	if slices.Contains(r.GetOperationsForResource(resource), operation) {
		return true
	}
	return len(r.ExpandCompositeOperation(resource, string(operation))) > 0
}

func (r *Registry) HasResource(resource string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.resources[resource]
	return ok
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.resources)
}

func (r *Registry) GetByCategory(category string) []*permissionspec.ResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*permissionspec.ResourceDefinition
	for _, def := range r.resources {
		if def.Category == category {
			result = append(result, def)
		}
	}
	return result
}

type categoryLabel struct {
	Label string
}

var categoryOrder = []categoryLabel{
	{Label: "Desk"},
	{Label: "Triage"},
	{Label: "Intake"},
	{Label: "Shipments"},
	{Label: "Dispatch"},
	{Label: "Carriers"},
	{Label: "Equipment"},
	{Label: "Fuel & tax"},
	{Label: "EDI"},
	{Label: "Human resources"},
	{Label: "Payroll"},
	{Label: "Billing"},
	{Label: "Detention"},
	{Label: "Carrier settlements"},
	{Label: "Accounting"},
	{Label: "Reports"},
	{Label: "Documents"},
	{Label: "AI & agents"},
	{Label: "Organization settings"},
	{Label: "Users & security"},
	{Label: "Protected fields"},
}

func CategoryOrder() []string {
	order := make([]string, len(categoryOrder))
	for i, category := range categoryOrder {
		order[i] = category.Label
	}
	return order
}

func (r *Registry) GetCategories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	categories := make(map[string]bool)
	for _, def := range r.resources {
		if def.Category != "" {
			categories[def.Category] = true
		}
	}

	result := make([]string, 0, len(categories))
	for _, category := range categoryOrder {
		if categories[category.Label] {
			result = append(result, category.Label)
			delete(categories, category.Label)
		}
	}
	rest := make([]string, 0, len(categories))
	for cat := range categories {
		rest = append(rest, cat)
	}
	sort.Strings(rest)
	return append(result, rest...)
}

func (r *Registry) registerAll() {
	for _, def := range slices.Concat(
		adminperms.Resources(),
		fleetperms.Resources(),
		workerperms.Resources(),
		shipmentperms.Resources(),
		billingperms.Resources(),
		agentperms.Resources(),
		carrierperms.Resources(),
		accountingperms.Resources(),
		payrollperms.Resources(),
		complianceperms.Resources(),
		reportperms.Resources(),
	) {
		_ = r.Register(def)
	}
}
