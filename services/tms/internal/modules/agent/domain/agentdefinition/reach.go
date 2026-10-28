package agentdefinition

func (d *Definition) ReachableByNobody(grantedRoles int) bool {
	return d.EffectiveAccessMode() == AccessRoles && grantedRoles == 0
}
