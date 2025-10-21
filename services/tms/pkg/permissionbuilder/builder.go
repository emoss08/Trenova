package permissionbuilder

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/emoss08/trenova/pkg/permissionregistry"
	"github.com/emoss08/trenova/pkg/pulid"
)

type PolicyBuilder struct {
	name        string
	description string
	effect      permission.Effect
	priority    int
	resources   []PolicyResource
	subjects    []permission.Subject
	scopeType   permission.ScopeType
	buID        *pulid.ID
	orgIDs      []pulid.ID
	registry    *permissionregistry.Registry
}

type PolicyResource struct {
	ResourceType permission.Resource
	Actions      permission.ActionSet
	DataScope    permission.DataScope
}

func NewPolicyBuilder(name string, registry *permissionregistry.Registry) *PolicyBuilder {
	return &PolicyBuilder{
		name:      name,
		effect:    permission.EffectAllow,
		priority:  100,
		resources: []PolicyResource{},
		subjects:  []permission.Subject{},
		scopeType: permission.ScopeTypeOrganization,
		registry:  registry,
	}
}

func (pb *PolicyBuilder) WithDescription(desc string) *PolicyBuilder {
	pb.description = desc
	return pb
}

func (pb *PolicyBuilder) WithEffect(effect permission.Effect) *PolicyBuilder {
	pb.effect = effect
	return pb
}

func (pb *PolicyBuilder) WithPriority(priority int) *PolicyBuilder {
	pb.priority = priority
	return pb
}

func (pb *PolicyBuilder) WithBusinessUnitScope(buID pulid.ID) *PolicyBuilder {
	pb.scopeType = permission.ScopeTypeBusinessUnit
	pb.buID = &buID
	return pb
}

func (pb *PolicyBuilder) WithOrganizationScope(orgIDs ...pulid.ID) *PolicyBuilder {
	pb.scopeType = permission.ScopeTypeOrganization
	pb.orgIDs = orgIDs
	return pb
}

func (pb *PolicyBuilder) AddResource(
	resourceType permission.Resource,
	standardOps permission.Operation,
	extendedOps []string,
	dataScope permission.DataScope,
) *PolicyBuilder {
	pb.resources = append(pb.resources, PolicyResource{
		ResourceType: resourceType,
		Actions: permission.ActionSet{
			StandardOps: standardOps,
			ExtendedOps: extendedOps,
		},
		DataScope: dataScope,
	})
	return pb
}

func (pb *PolicyBuilder) AddResourceWithFieldRules(
	resourceType permission.Resource,
	standardOps permission.Operation,
	extendedOps []string,
	dataScope permission.DataScope,
) *PolicyBuilder {
	pb.resources = append(pb.resources, PolicyResource{
		ResourceType: resourceType,
		Actions: permission.ActionSet{
			StandardOps: standardOps,
			ExtendedOps: extendedOps,
		},
		DataScope: dataScope,
	})
	return pb
}

func (pb *PolicyBuilder) AddFullAccessResource(
	resourceType permission.Resource,
	dataScope permission.DataScope,
) *PolicyBuilder {
	standardOps := permission.OpCreate | permission.OpRead | permission.OpUpdate |
		permission.OpDelete | permission.OpExport | permission.OpImport

	// Query registry for extended operations (if registry is available)
	extendedOps := []string{}
	if pb.registry != nil {
		if res, exists := pb.registry.GetResource(string(resourceType)); exists {
			for _, op := range res.GetSupportedOperations() {
				// Check if this is an extended operation (beyond standard CRUD + export + import)
				// Standard ops are: 1, 2, 4, 8, 16, 32
				// Extended ops include: approve(64), reject(128), assign(16384), duplicate(32768)
				if op.Code > 32 && op.Code != permission.OpApprove &&
					op.Code != permission.OpReject {
					extendedOps = append(extendedOps, op.Name)
				}
			}
		}
	}

	pb.resources = append(pb.resources, PolicyResource{
		ResourceType: resourceType,
		Actions: permission.ActionSet{
			StandardOps: standardOps,
			ExtendedOps: extendedOps,
		},
		DataScope: dataScope,
	})
	return pb
}

func (pb *PolicyBuilder) AddReadOnlyResource(
	resourceType permission.Resource,
	dataScope permission.DataScope,
) *PolicyBuilder {
	pb.resources = append(pb.resources, PolicyResource{
		ResourceType: resourceType,
		Actions: permission.ActionSet{
			StandardOps: permission.OpRead,
			ExtendedOps: []string{},
		},
		DataScope: dataScope,
	})
	return pb
}

func (pb *PolicyBuilder) AddSubjectUser(userID pulid.ID) *PolicyBuilder {
	pb.subjects = append(pb.subjects, permission.Subject{
		Type: permission.SubjectTypeUser,
		ID:   userID,
	})
	return pb
}

func (pb *PolicyBuilder) AddSubjectRole(roleID pulid.ID) *PolicyBuilder {
	pb.subjects = append(pb.subjects, permission.Subject{
		Type: permission.SubjectTypeRole,
		ID:   roleID,
	})
	return pb
}

func (pb *PolicyBuilder) Build(buID pulid.ID) *permission.Policy {
	if pb.description == "" {
		pb.description = fmt.Sprintf("Policy: %s", pb.name)
	}

	policy := &permission.Policy{
		ID:          pulid.MustNew("pol_"),
		Name:        pb.name,
		Description: pb.description,
		Effect:      pb.effect,
		Priority:    pb.priority,
		Subjects:    pb.subjects,
		Resources:   make([]permission.ResourceRule, len(pb.resources)),
		Scope: permission.PolicyScope{
			BusinessUnitID:  buID,
			OrganizationIDs: pb.orgIDs,
			Inheritable:     false,
		},
	}

	for i, res := range pb.resources {
		policy.Resources[i] = permission.ResourceRule{
			ResourceType: string(res.ResourceType),
			Actions:      res.Actions,
			DataScope:    res.DataScope,
			Conditions:   []permission.PolicyCondition{},
		}
	}

	return policy
}

type RoleBuilder struct {
	name        string
	description string
	level       permission.RoleLevel
	isAdmin     bool
	policies    []pulid.ID
}

func NewRoleBuilder(name string, level permission.RoleLevel) *RoleBuilder {
	return &RoleBuilder{
		name:     name,
		level:    level,
		policies: []pulid.ID{},
	}
}

func (rb *RoleBuilder) WithDescription(desc string) *RoleBuilder {
	rb.description = desc
	return rb
}

func (rb *RoleBuilder) WithIsAdmin(isAdmin bool) *RoleBuilder {
	rb.isAdmin = isAdmin
	return rb
}

func (rb *RoleBuilder) AddPolicy(policyID pulid.ID) *RoleBuilder {
	rb.policies = append(rb.policies, policyID)
	return rb
}

func (rb *RoleBuilder) Build(buID pulid.ID) *permission.Role {
	if rb.description == "" {
		rb.description = fmt.Sprintf("Role: %s", rb.name)
	}

	return &permission.Role{
		ID:             pulid.MustNew("rol_"),
		BusinessUnitID: buID,
		Name:           rb.name,
		Description:    rb.description,
		Level:          rb.level,
		PolicyIDs:      rb.policies,
		ParentRoles:    []pulid.ID{},
		IsSystem:       rb.level == permission.RoleLevelSystem,
		IsAdmin:        rb.isAdmin,
		Scope: permission.RoleScope{
			Type:          permission.ScopeTypeOrganization,
			Organizations: []pulid.ID{},
			Inheritable:   false,
		},
	}
}

func CreateAdminPolicy(
	name string,
	buID pulid.ID,
	orgIDs []pulid.ID,
	registry *permissionregistry.Registry,
) *permission.Policy {
	builder := NewPolicyBuilder(name, registry).
		WithDescription("Full administrative access to all resources").
		WithPriority(1000).
		WithOrganizationScope(orgIDs...)

	allResources := GetAllResources()

	for _, resource := range allResources {
		builder.AddFullAccessResource(resource, permission.DataScopeAll)
	}

	return builder.Build(buID)
}

func CreateAdminRole(buID, policyID pulid.ID) *permission.Role {
	return NewRoleBuilder("Administrator", permission.RoleLevelSystem).
		WithDescription("System administrator with full access").
		WithIsAdmin(true).
		AddPolicy(policyID).
		Build(buID)
}

func CreatePermissionRegistry() *permissionregistry.Registry {
	registry := permissionregistry.NewRegistryManual()

	entities := domainregistry.RegisterPermissionAwareEntities()
	for _, entity := range entities {
		registry.Register(entity)
	}

	return registry
}
