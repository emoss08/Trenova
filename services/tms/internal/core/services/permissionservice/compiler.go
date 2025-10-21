package permissionservice

import (
	"hash/fnv"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/permissionregistry"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"go.uber.org/zap"
)

type policyCompiler struct {
	registry *permissionregistry.Registry
	logger   *zap.Logger
}

func NewPolicyCompiler(
	registry *permissionregistry.Registry,
	logger *zap.Logger,
) ports.PolicyCompiler {
	return &policyCompiler{
		registry: registry,
		logger:   logger.Named("permission.policy-compiler"),
	}
}

func (pc *policyCompiler) Compile(
	policies []*permission.Policy,
) (*ports.CompiledPermissions, error) {
	log := pc.logger.With(
		zap.String("operation", "Compile"),
		zap.Int("policyCount", len(policies)),
	)

	resources := make(ports.ResourcePermissionMap)
	dataScopes := make(map[string]permission.DataScope)
	var globalFlags uint64

	for _, policy := range policies {
		if policy.Effect == permission.EffectDeny {
			pc.applyDenyPolicy(policy, resources)
		} else {
			pc.applyAllowPolicy(policy, resources, dataScopes)
		}
	}

	for resourceType, perms := range resources {
		perms.QuickCheck = pc.computeQuickCheck(perms.StandardOps, len(perms.ExtendedOps))

		resources[resourceType] = perms
	}

	compiled := &ports.CompiledPermissions{
		Resources:   resources,
		GlobalFlags: globalFlags,
		DataScopes:  dataScopes,
	}

	log.Debug("compiled permissions",
		zap.Int("resourceCount", len(resources)),
		zap.Int("dataScopeCount", len(dataScopes)),
	)

	return compiled, nil
}

func (pc *policyCompiler) CompileForUser(
	userID, organizationID pulid.ID,
	policies []*permission.Policy,
) (*ports.CompiledPermissions, error) {
	log := pc.logger.With(
		zap.String("operation", "CompileForUser"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
		zap.Int("policyCount", len(policies)),
	)

	filteredPolicies := pc.filterPoliciesForUser(policies, userID, organizationID)

	log.Debug("filtered policies for user",
		zap.Int("originalCount", len(policies)),
		zap.Int("filteredCount", len(filteredPolicies)),
	)

	return pc.Compile(filteredPolicies)
}

func (pc *policyCompiler) OptimizeBitfields(actions []string) uint32 {
	var bitfield uint32
	for _, action := range actions {
		bitfield = ports.AddAction(bitfield, action)
	}

	return bitfield
}

func (pc *policyCompiler) BuildBloomFilter(
	permissions *ports.CompiledPermissions,
) ([]byte, error) {
	bloomSize := 10240                    // 10KB
	bloom := newBloomFilter(bloomSize, 7) // 7 hash functions
	for resourceType, perms := range permissions.Resources {
		for action := range ports.ActionBits {
			if ports.HasAction(perms.StandardOps, action) {
				key := resourceType + ":" + action
				bloom.Add(key)
			}
		}

		for _, action := range perms.ExtendedOps {
			key := resourceType + ":" + action
			bloom.Add(key)
		}
	}

	return bloom.Bytes(), nil
}

func (pc *policyCompiler) applyDenyPolicy(
	policy *permission.Policy,
	resources ports.ResourcePermissionMap,
) {
	for _, resource := range policy.Resources {
		resourceKey := resource.ResourceType
		existing, ok := resources[resourceKey]

		if !ok {
			continue
		}

		for action := range ports.ActionBits {
			if hasActionInActionSet(&resource.Actions, action) {
				existing.StandardOps = ports.RemoveAction(existing.StandardOps, action)
			}
		}

		for _, deniedAction := range resource.Actions.ExtendedOps {
			existing.ExtendedOps = utils.RemoveString(existing.ExtendedOps, deniedAction)
		}

		resources[resourceKey] = existing
	}
}

func (pc *policyCompiler) applyAllowPolicy(
	policy *permission.Policy,
	resources ports.ResourcePermissionMap,
	dataScopes map[string]permission.DataScope,
) {
	for _, resource := range policy.Resources {
		resourceKey := resource.ResourceType

		registryResource, exists := pc.registry.GetResource(resourceKey)
		if !exists {
			pc.logger.Warn("resource not found in registry, skipping",
				zap.String("resource", resourceKey),
				zap.String("policyID", policy.ID.String()),
			)
			continue
		}

		existing, ok := resources[resourceKey]

		if !ok {
			existing = &ports.ResourcePermission{
				StandardOps: 0,
				ExtendedOps: []string{},
				DataScope:   resource.DataScope,
			}
		}

		expandedOps := pc.expandOperations(
			registryResource,
			resource.Actions.StandardOps.ToUint32(),
		)
		existing.StandardOps |= expandedOps

		for _, action := range resource.Actions.ExtendedOps {
			if pc.validateOperation(registryResource, action) {
				if !slices.Contains(existing.ExtendedOps, action) {
					existing.ExtendedOps = append(existing.ExtendedOps, action)
				}
			} else {
				pc.logger.Warn("extended operation not supported by resource",
					zap.String("resource", resourceKey),
					zap.String("operation", action),
				)
			}
		}

		if isMorePermissive(resource.DataScope, existing.DataScope) {
			existing.DataScope = resource.DataScope
		}

		resources[resourceKey] = existing
		dataScopes[resourceKey] = existing.DataScope
	}
}

func (pc *policyCompiler) filterPoliciesForUser(
	policies []*permission.Policy,
	userID, organizationID pulid.ID,
) []*permission.Policy {
	filtered := make([]*permission.Policy, 0, len(policies))

	for _, policy := range policies {
		if !pc.policyAppliesToOrg(policy, organizationID) {
			continue
		}
		if len(policy.Subjects) > 0 && !pc.policyAppliesToUser(policy, userID) {
			continue
		}

		filtered = append(filtered, policy)
	}

	return filtered
}

func (pc *policyCompiler) policyAppliesToOrg(
	policy *permission.Policy,

	organizationID pulid.ID,
) bool {
	if len(policy.Scope.OrganizationIDs) == 0 {
		return true
	}

	for _, orgID := range policy.Scope.OrganizationIDs {
		if orgID == organizationID {
			return true
		}
	}

	return false
}

func (pc *policyCompiler) policyAppliesToUser(
	policy *permission.Policy,
	userID pulid.ID,
) bool {
	if len(policy.Subjects) == 0 {
		return true
	}

	for _, subject := range policy.Subjects {
		if subject.Type == permission.SubjectTypeUser && subject.ID == userID {
			return true
		}
	}

	return false
}

func (pc *policyCompiler) computeQuickCheck(standardOps uint32, extendedCount int) uint64 {
	h := fnv.New64a()

	h.Write([]byte{
		byte(standardOps),
		byte(standardOps >> 8),
		byte(standardOps >> 16),
		byte(standardOps >> 24),
		byte(extendedCount),
	})

	return h.Sum64()
}

func (pc *policyCompiler) expandOperations(
	resource permissionregistry.PermissionAware,
	bitfield uint32,
) uint32 {
	// ! The bitfield is already expanded, just validate it contains valid operations
	supportedOps := resource.GetSupportedOperations()
	validated := uint32(0)

	for _, op := range supportedOps {
		if (bitfield & op.Code.ToUint32()) == op.Code.ToUint32() {
			validated |= uint32(op.Code)
		}
	}

	if validated != bitfield {
		pc.logger.Warn("policy contains unsupported operations",
			zap.String("resource", resource.GetResourceName()),
			zap.Uint32("requested", bitfield),
			zap.Uint32("validated", validated),
		)
	}

	return validated
}

func (pc *policyCompiler) validateOperation(
	resource permissionregistry.PermissionAware,
	operation string,
) bool {
	supportedOps := resource.GetSupportedOperations()
	for _, op := range supportedOps {
		if op.Name == operation {
			return true
		}
	}
	return false
}

func hasActionInActionSet(actions *permission.ActionSet, action string) bool {
	if bit, ok := ports.ActionBits[action]; ok {
		return (actions.StandardOps.ToUint32() & bit) != 0
	}

	return slices.Contains(actions.ExtendedOps, action)
}

func isMorePermissive(a, b permission.DataScope) bool {
	scopeOrder := map[permission.DataScope]int{
		permission.DataScopeOwn:          1,
		permission.DataScopeOrganization: 2,
		permission.DataScopeBusinessUnit: 3,
		permission.DataScopeAll:          4,
	}

	return scopeOrder[a] > scopeOrder[b]
}
