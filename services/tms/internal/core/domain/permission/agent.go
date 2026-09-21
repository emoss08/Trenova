package permission

var agentAllowedPermissions = map[Resource]map[Operation]struct{}{
	ResourceShipment: {
		OpRead: {},
	},
	ResourceShipmentMove: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceWorker: {
		OpRead: {},
	},
	ResourceTractor: {
		OpRead: {},
	},
	ResourceTrailer: {
		OpRead: {},
	},
	ResourceCustomer: {
		OpRead: {},
	},
	ResourceLocation: {
		OpRead: {},
	},
	ResourceBillingQueue: {
		OpRead:   {},
		OpUpdate: {},
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
	ResourceAgentMemory: {
		OpRead:   {},
		OpCreate: {},
		OpUpdate: {},
	},
	ResourceInsight: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceWatchtower: {
		OpRead: {},
	},
	ResourceBriefing: {
		OpRead: {},
	},
	ResourceInvoice: {
		OpRead: {},
	},
	ResourceCustomerPayment: {
		OpRead:   {},
		OpCreate: {},
	},
	ResourceBankReceipt: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceBankReceiptWorkItem: {
		OpRead:   {},
		OpUpdate: {},
	},
	ResourceWorkerCredential: {
		OpRead: {},
	},
	ResourceDriverMessage: {
		OpRead:   {},
		OpCreate: {},
	},
	ResourceCustomerCommunication: {
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

func AgentAllowedPermissions() map[Resource][]Operation {
	out := make(map[Resource][]Operation, len(agentAllowedPermissions))
	for resource, operations := range agentAllowedPermissions {
		ops := make([]Operation, 0, len(operations))
		for operation := range operations {
			ops = append(ops, operation)
		}
		out[resource] = ops
	}

	return out
}
