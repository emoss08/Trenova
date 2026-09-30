package permission

import (
	"errors"
	"fmt"
	"sync"
)

const (
	fieldGLAccountID   = "glAccountId"
	fieldPostedBatchID = "postedBatchId"
	fieldUpdatedByID   = "updatedById"
)

type OperationDefinition struct {
	Operation   Operation `json:"operation"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
}

type ResourceDefinition struct {
	Resource           string                      `json:"resource"`
	DisplayName        string                      `json:"displayName"`
	Description        string                      `json:"description"`
	Category           string                      `json:"category"`
	Operations         []OperationDefinition       `json:"operations"`
	CompositeOps       map[string][]Operation      `json:"compositeOps,omitempty"`
	ParentResource     string                      `json:"parentResource,omitempty"`
	FieldSensitivities map[string]FieldSensitivity `json:"fieldSensitivities,omitempty"`
	DefaultSensitivity FieldSensitivity            `json:"defaultSensitivity"`
}

type Registry struct {
	mu        sync.RWMutex
	resources map[string]*ResourceDefinition
	children  map[string][]string
}

func NewRegistry() *Registry {
	r := &Registry{
		resources: make(map[string]*ResourceDefinition),
		children:  make(map[string][]string),
	}
	r.registerAll()
	return r
}

func NewEmptyRegistry() *Registry {
	return &Registry{
		resources: make(map[string]*ResourceDefinition),
		children:  make(map[string][]string),
	}
}

func (r *Registry) Register(def *ResourceDefinition) error {
	if def.Resource == "" {
		return errors.New("resource name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.resources[def.Resource]; exists {
		return fmt.Errorf("resource %s already registered", def.Resource)
	}

	if def.DefaultSensitivity == "" {
		def.DefaultSensitivity = SensitivityInternal
	}

	r.resources[def.Resource] = def

	if def.ParentResource != "" {
		r.children[def.ParentResource] = append(r.children[def.ParentResource], def.Resource)
	}

	return nil
}

func (r *Registry) Get(resource string) (*ResourceDefinition, bool) {
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

func (r *Registry) All() []*ResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ResourceDefinition, 0, len(r.resources))
	for _, def := range r.resources {
		result = append(result, def)
	}
	return result
}

func (r *Registry) GetFieldSensitivity(resource, field string) FieldSensitivity {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.resources[resource]
	if !ok {
		return SensitivityInternal
	}
	if sens, sensOK := def.FieldSensitivities[field]; sensOK {
		return sens
	}
	return def.DefaultSensitivity
}

func (r *Registry) GetOperationsForResource(resource string) []Operation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.resources[resource]
	if !ok {
		return nil
	}

	ops := make([]Operation, 0, len(def.Operations))
	for _, opDef := range def.Operations {
		ops = append(ops, opDef.Operation)
	}
	return ops
}

func (r *Registry) ExpandCompositeOperation(resource, compositeName string) []Operation {
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

func (r *Registry) GetByCategory(category string) []*ResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ResourceDefinition
	for _, def := range r.resources {
		if def.Category == category {
			result = append(result, def)
		}
	}
	return result
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
	for cat := range categories {
		result = append(result, cat)
	}
	return result
}

// registerAll loads every resource the system knows about. Each call is the
// whole of one category's table and lives in its own file beside it; two of
// them, the rate and recurring-earning tables, are reached through the billing
// and settlement categories they belong with rather than from here.
func (r *Registry) registerAll() {
	r.registerAdministrationResources()
	r.registerEquipmentResources()
	r.registerFuelAndIFTAResources()
	r.registerWorkerResources()
	r.registerOperationsResources()
	r.registerBillingResources()
	r.registerCustomerResources()
	r.registerCarrierResources()
	r.registerLocationResources()
	r.registerCommodityResources()
	r.registerAccountingResources()
	r.registerSettlementResources()
	r.registerComplianceResources()
	r.registerReferenceDataResources()
	r.registerReportingResources()
	r.registerCommunicationResources()
}
