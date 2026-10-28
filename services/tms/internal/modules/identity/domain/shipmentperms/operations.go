package shipmentperms

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func operationsResources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		{
			Resource:    permission.ResourceOrder.String(),
			DisplayName: "Order",
			Description: "Commercial order management (groups shipments/legs under one customer order)",
			Category:    "Shipments",
			Operations: []permissionspec.OperationDefinition{
				{Operation: permission.OpRead, DisplayName: "View", Description: "View orders"},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Create new orders",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify, cancel and close orders",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export order data",
				},
				{
					Operation:   permission.OpCancel,
					DisplayName: "Cancel",
					Description: "Cancel orders",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
			Scopes: []permissionspec.ScopeSupport{
				{
					Scope:      permission.DataScopeOwn,
					EnforcedBy: permissionspec.ScopeEnforcedByDatabase,
				},
			},
		},

		{
			Resource:    permission.ResourceShipment.String(),
			DisplayName: "Shipment",
			Description: "Shipment management",
			Category:    "Shipments",
			ReferenceView: []string{
				"proNumber",
				"bol",
				"status",
				"customerId",
				"poNumber",
				"originCity",
				"originState",
				"destinationCity",
				"destinationState",
			},
			Operations: []permissionspec.OperationDefinition{
				{Operation: permission.OpRead, DisplayName: "View", Description: "View shipments"},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Create new shipments",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify shipments and cancel billing transfer runs",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export shipment data",
				},
				{
					Operation:   permission.OpImport,
					DisplayName: "Import",
					Description: "Import shipments",
				},
				{
					Operation:   permission.OpSubmit,
					DisplayName: "Submit",
					Description: "Submit shipments for dispatch",
				},
				{
					Operation:   permission.OpCancel,
					DisplayName: "Cancel",
					Description: "Cancel shipments",
				},
				{
					Operation:   permission.OpDuplicate,
					DisplayName: "Duplicate",
					Description: "Duplicate shipments",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
			FieldSensitivities: map[string]permission.FieldSensitivity{
				"profitabilityEstimate": permission.SensitivityConfidential,
			},
			Scopes: []permissionspec.ScopeSupport{
				{
					Scope:      permission.DataScopeOwn,
					EnforcedBy: permissionspec.ScopeEnforcedByDatabase,
				},
			},
		},

		{
			Resource:       permission.ResourceRecurringShipment.String(),
			DisplayName:    "Recurring Shipment",
			Description:    "Recurring shipment series management and scheduled generation",
			Category:       "Shipments",
			ParentResource: permission.ResourceShipment.String(),
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View recurring shipments",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Create recurring shipments",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify recurring shipments, pause and resume series",
				},
				{
					Operation:   permission.OpDuplicate,
					DisplayName: "Generate",
					Description: "Generate shipments from a recurring series",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:       permission.ResourceShipmentComment.String(),
			DisplayName:    "Shipment Comment",
			Description:    "Shipment comment management",
			Category:       "Shipments",
			ParentResource: permission.ResourceShipment.String(),
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View shipment comments",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Add shipment comments",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify shipment comments",
				},
				{
					Operation:   permission.OpDelete,
					DisplayName: "Delete",
					Description: "Delete shipment comments",
				},
				{
					Operation:   permission.OpPin,
					DisplayName: "Pin",
					Description: "Pin comments to the top of a shipment",
				},
				{
					Operation:   permission.OpUnpin,
					DisplayName: "Unpin",
					Description: "Remove pinned comments from the top of a shipment",
				},
				{
					Operation:   permission.OpResolve,
					DisplayName: "Resolve",
					Description: "Resolve and reopen shipment comments",
				},
				{
					Operation:   permission.OpManage,
					DisplayName: "Manage",
					Description: "Edit or delete any user's shipment comments",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:       permission.ResourceShipmentMove.String(),
			DisplayName:    "Shipment Move",
			Description:    "Shipment move management",
			Category:       "Dispatch",
			ParentResource: permission.ResourceShipment.String(),
			Operations: []permissionspec.OperationDefinition{
				{Operation: permission.OpRead, DisplayName: "View", Description: "View moves"},
				{Operation: permission.OpCreate, DisplayName: "Create", Description: "Add moves"},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify moves",
				},
				{
					Operation:   permission.OpAssign,
					DisplayName: "Assign",
					Description: "Assign resources to moves",
				},
				{
					Operation:   permission.OpUnassign,
					DisplayName: "Unassign",
					Description: "Remove assignments from moves",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export shipment move data",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
			FieldSensitivities: map[string]permission.FieldSensitivity{
				"externalDriverName":  permission.SensitivityInternal,
				"externalDriverPhone": permission.SensitivityRestricted,
			},
		},

		{
			Resource:       permission.ResourceShipmentStop.String(),
			DisplayName:    "Shipment Stop",
			Description:    "Shipment stop management",
			Category:       "Shipments",
			ParentResource: permission.ResourceShipment.String(),
			Operations: []permissionspec.OperationDefinition{
				{Operation: permission.OpRead, DisplayName: "View", Description: "View stops"},
				{Operation: permission.OpCreate, DisplayName: "Create", Description: "Add stops"},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify stops",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export shipment stop data",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:    permission.ResourceWorkerDispatchHold.String(),
			DisplayName: "Driver Dispatch Hold",
			Description: "Whether a driver can be given new freight. Held apart from the " +
				"worker record on purpose: taking somebody off the board until a paper is " +
				"renewed is a dispatch decision, and it is not the same permission as " +
				"editing their employment, their pay or their profile.",
			Category:       "Dispatch",
			ParentResource: permission.ResourceWorker.String(),
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "See who is held off dispatch",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Take a driver off dispatch, and put them back",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:       permission.ResourceShipmentHold.String(),
			DisplayName:    "Shipment Hold",
			Description:    "Shipment hold management",
			Category:       "Shipments",
			ParentResource: permission.ResourceShipment.String(),
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View shipment holds",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Create shipment holds",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify and release shipment holds",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:    permission.ResourceJurisdictionRuleOverride.String(),
			DisplayName: "Carrier Override",
			Description: "This organization's stricter posture on a jurisdiction. " +
				"Unlike a jurisdiction rule these are yours alone, and an override can " +
				"only tighten a state limit, never loosen one.",
			Category:       "Dispatch",
			ParentResource: permission.ResourceJurisdictionRule.String(),
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View carrier overrides",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Hold this fleet to a stricter limit than a state requires",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify an override",
				},
				{
					Operation:   permission.OpDelete,
					DisplayName: "Delete",
					Description: "Remove an override, returning the jurisdiction to its statutory limits",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:    permission.ResourceJurisdictionRule.String(),
			DisplayName: "Jurisdiction Rule",
			Description: "Shared oversize and overweight limits per state. " +
				"This data is global: it has no organization column and every tenant " +
				"reads the same rows, so a change here alters what the whole platform " +
				"is told is legal. Carrier-specific posture belongs in an override.",
			Category: "Dispatch",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View jurisdiction limits and their verification state",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Add a jurisdiction rule, visible to every organization",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Change limits every organization is held to",
				},
				{
					Operation:   permission.OpApprove,
					DisplayName: "Verify",
					Description: "Confirm a rule against the issuing state, or mark it disputed",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:       permission.ResourcePermit.String(),
			DisplayName:    "Oversize Permit",
			Description:    "Oversize and overweight permits and their derived requirements",
			Category:       "Shipments",
			ParentResource: permission.ResourceShipment.String(),
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View permits and permit requirements",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Record permits",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify recorded permits",
				},
				{
					Operation:   permission.OpApprove,
					DisplayName: "Waive",
					Description: "Waive a permit requirement, accepting the compliance risk",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:    permission.ResourceServiceFailure.String(),
			DisplayName: "Service Failure",
			Description: "Service failure investigation and approval records",
			Category:    "Shipments",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View service failures",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Create service failures",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify and resolve service failures",
				},
				{
					Operation:   permission.OpApprove,
					DisplayName: "Approve",
					Description: "Approve service failures",
				},
				{
					Operation:   permission.OpArchive,
					DisplayName: "Void",
					Description: "Void service failures",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export service failures",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:      permission.ResourceServiceFailureReasonCode.String(),
			DisplayName:   "Service Failure Reason Code",
			Description:   "Service failure reason code reference data",
			Category:      "Shipments",
			ReferenceView: []string{"code", "label", "category"},
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View service failure reason codes",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Create service failure reason codes",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify service failure reason codes",
				},
				{
					Operation:   permission.OpApprove,
					DisplayName: "Approve",
					Description: "Approve service failure reason codes",
				},
				{
					Operation:   permission.OpArchive,
					DisplayName: "Archive",
					Description: "Archive and reactivate reason codes",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export service failure reason codes",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:           permission.ResourceShipmentType.String(),
			DisplayName:        "Shipment Type",
			Description:        "Shipment type definitions",
			Category:           "Shipments",
			Operations:         permissionspec.StandardOps,
			DefaultSensitivity: permission.SensitivityRestricted,
			FieldSensitivities: map[string]permission.FieldSensitivity{
				"id":             permission.SensitivityInternal,
				"businessUnitId": permission.SensitivityInternal,
				"organizationId": permission.SensitivityInternal,
				"status":         permission.SensitivityInternal,
				"code":           permission.SensitivityInternal,
				"description":    permission.SensitivityInternal,
				"color":          permission.SensitivityInternal,
				"version":        permission.SensitivityInternal,
				"createdAt":      permission.SensitivityInternal,
				"updatedAt":      permission.SensitivityInternal,
			},
		},

		{
			Resource:           permission.ResourceDispatchControl.String(),
			DisplayName:        "Dispatch Control",
			Description:        "Dispatch settings and configuration",
			Category:           "Dispatch",
			Operations:         permissionspec.StandardOps,
			DefaultSensitivity: permission.SensitivityInternal,
			FieldSensitivities: map[string]permission.FieldSensitivity{
				"tenderCounterMarginFloorPercent": permission.SensitivityConfidential,
			},
		},

		{
			Resource:           permission.ResourceDistanceProfile.String(),
			DisplayName:        "Distance Profile",
			Description:        "Distance calculation routing profiles",
			Category:           "Dispatch",
			Operations:         permissionspec.StandardOpsWithDelete,
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:           permission.ResourceDistanceOverride.String(),
			DisplayName:        "Distance Override",
			Description:        "Lane-specific distance overrides",
			Category:           "Dispatch",
			Operations:         permissionspec.StandardOpsWithDelete,
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:           permission.ResourceDistanceControl.String(),
			DisplayName:        "Distance Control",
			Description:        "Distance calculation settings and profile assignments",
			Category:           "Dispatch",
			Operations:         permissionspec.StandardOps,
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:           permission.ResourceStoredMileage.String(),
			DisplayName:        "Stored Mileage",
			Description:        "Reusable stored mileage lane records",
			Category:           "Dispatch",
			Operations:         permissionspec.StandardOpsWithDelete,
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:           permission.ResourceDataEntryControl.String(),
			DisplayName:        "Data Entry Control",
			Description:        "Data entry case formatting settings",
			Category:           "Organization settings",
			Operations:         permissionspec.StandardOps,
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:           permission.ResourceShipmentControl.String(),
			DisplayName:        "Shipment Control",
			Description:        "Shipment control settings and configuration",
			Category:           "Shipments",
			Operations:         permissionspec.StandardOps,
			DefaultSensitivity: permission.SensitivityRestricted,
		},

		{
			Resource:    permission.ResourceHoldReason.String(),
			DisplayName: "Hold Reason",
			Description: "Hold reason management",
			Category:    "Shipments",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View hold reasons",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Create new hold reasons",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Modify hold reasons",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export hold reason data",
				},
				{
					Operation:   permission.OpImport,
					DisplayName: "Import",
					Description: "Import hold reasons",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},
	}
}
