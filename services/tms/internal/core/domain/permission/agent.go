package permission

var agentAllowedPermissions = map[Resource]map[Operation]struct{}{
	ResourceBillingQueue: {
		OpRead: {},
	},
	ResourceShipment: {
		OpRead: {},
	},
	ResourceDocument: {
		OpRead:   {},
		OpCreate: {},
	},
	ResourceAgentRun: {
		OpRead:   {},
		OpCreate: {},
	},
	ResourceAgentProposal: {
		OpRead:   {},
		OpCreate: {},
	},
	ResourceAgentException: {
		OpRead:   {},
		OpCreate: {},
	},
}

func IsAgentAllowed(resource Resource, operation Operation) bool {
	operations, ok := agentAllowedPermissions[resource]
	if !ok {
		return false
	}

	_, ok = operations[operation]
	return ok
}
