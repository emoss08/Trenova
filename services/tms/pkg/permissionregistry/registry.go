package permissionregistry

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"go.uber.org/fx"
)

type Registry struct {
	resources map[string]PermissionAware
}

type RegistryParams struct {
	fx.In

	Entities []PermissionAware `group:"permission_entities"`
}

func NewRegistry(p RegistryParams) *Registry {
	r := &Registry{
		resources: make(map[string]PermissionAware),
	}

	for _, entity := range p.Entities {
		r.Register(entity)
	}

	return r
}

func NewRegistryManual() *Registry {
	return &Registry{
		resources: make(map[string]PermissionAware),
	}
}

func (r *Registry) Register(entity PermissionAware) {
	r.resources[entity.GetResourceName()] = entity
}

func (r *Registry) GetResource(name string) (PermissionAware, bool) {
	res, exists := r.resources[name]
	return res, exists
}

func (r *Registry) GetAllResources() map[string]PermissionAware {
	return r.resources
}

func (r *Registry) GetResourceNames() []string {
	names := make([]string, 0, len(r.resources))
	for name := range r.resources {
		names = append(names, name)
	}
	return names
}

func (r *Registry) GetOperationsForResource(resourceName string) ([]OperationDefinition, bool) {
	res, exists := r.resources[resourceName]
	if !exists {
		return nil, false
	}
	return res.GetSupportedOperations(), true
}

func (r *Registry) GetCompositeOperationsForResource(
	resourceName string,
) (map[string]permission.Operation, bool) {
	res, exists := r.resources[resourceName]
	if !exists {
		return nil, false
	}
	return res.GetCompositeOperations(), true
}

func (r *Registry) ExpandCompositeOperation(
	resourceName, operationName string,
) (permission.Operation, bool) {
	res, exists := r.resources[resourceName]
	if !exists {
		return 0, false
	}

	composites := res.GetCompositeOperations()
	value, found := composites[operationName]
	return value, found
}
