package permission

// RoleGrants is what a set of roles grants together, as the permission
// engine evaluates it: every operation each grant implies, the widest
// sensitivity ceiling among them, and a resource's grant reaching the
// resources whose parent it is. It holds no person or tenant, so it answers
// what a role would allow before anyone holds it.
type RoleGrants struct {
	operations     map[string]OperationSet
	MaxSensitivity FieldSensitivity
}

// GrantsOf is the union of the roles' grants. Pass a role and every role it
// inherits from.
func GrantsOf(roles ...*Role) RoleGrants {
	grants := RoleGrants{
		operations:     make(map[string]OperationSet),
		MaxSensitivity: SensitivityPublic,
	}
	for _, role := range roles {
		if role == nil {
			continue
		}
		if role.MaxSensitivity.Level() > grants.MaxSensitivity.Level() {
			grants.MaxSensitivity = role.MaxSensitivity
		}
		for _, rp := range role.Permissions {
			if rp == nil {
				continue
			}
			set, ok := grants.operations[rp.Resource]
			if !ok {
				set = NewOperationSet()
				grants.operations[rp.Resource] = set
			}
			set.Add(Implied(rp.Operations)...)
		}
	}

	return grants
}

// Allows reports whether the grants allow an operation on a resource,
// directly or through the resource's parent.
func (g RoleGrants) Allows(registry *Registry, resource string, op Operation) bool {
	if set, ok := g.operations[resource]; ok && set.Has(op) {
		return true
	}
	if registry == nil {
		return false
	}
	parent := registry.GetEffectiveResource(resource)
	if parent == resource {
		return false
	}
	set, ok := g.operations[parent]

	return ok && set.Has(op)
}
