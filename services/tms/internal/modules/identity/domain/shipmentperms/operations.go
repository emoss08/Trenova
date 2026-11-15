package shipmentperms

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

func operationsResources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		orderResource(),
		shipmentResource(),
		recurringShipmentResource(),
		shipmentCommentResource(),
		shipmentMoveResource(),
		shipmentStopResource(),
		workerDispatchHoldResource(),
		driverPositioningResource(),
		shipmentHoldResource(),
		jurisdictionRuleOverrideResource(),
		jurisdictionRuleResource(),
		permitResource(),
		serviceFailureResource(),
		serviceFailureReasonCodeResource(),
		shipmentTypeResource(),
		dispatchControlResource(),
		distanceProfileResource(),
		distanceOverrideResource(),
		distanceControlResource(),
		storedMileageResource(),
		dataEntryControlResource(),
		shipmentControlResource(),
		holdReasonResource(),
	}
}

func orderResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceOrder.String(),
		Feature:     platformcatalog.FeatureCoreTMS,
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
	}
}

func shipmentResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceShipment.String(),
		Feature:     platformcatalog.FeatureCoreTMS,
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
		FieldFeatures: shipmentFieldFeatures(),
		Scopes: []permissionspec.ScopeSupport{
			{
				Scope:      permission.DataScopeOwn,
				EnforcedBy: permissionspec.ScopeEnforcedByDatabase,
			},
		},
	}
}

func recurringShipmentResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceRecurringShipment.String(),
		Feature:        platformcatalog.FeatureCoreTMS,
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
	}
}

func shipmentCommentResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceShipmentComment.String(),
		Feature:        platformcatalog.FeatureCoreTMS,
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
	}
}

func shipmentMoveResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceShipmentMove.String(),
		Feature:        platformcatalog.FeatureDispatch,
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
		FieldFeatures: map[string]platformcatalog.FeatureKey{
			"ackStatus": platformcatalog.FeatureDriverPortal,
			"ackAt":     platformcatalog.FeatureDriverPortal,
			"ackReason": platformcatalog.FeatureDriverPortal,
		},
	}
}

func shipmentStopResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceShipmentStop.String(),
		Feature:        platformcatalog.FeatureCoreTMS,
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
		FieldFeatures: map[string]platformcatalog.FeatureKey{
			"countDetentionOverride": platformcatalog.FeatureDetention,
		},
	}
}

func workerDispatchHoldResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceWorkerDispatchHold.String(),
		Feature:     platformcatalog.FeatureDispatch,
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
	}
}

func driverPositioningResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceDriverPositioning.String(),
		Feature:     platformcatalog.FeatureDispatchIntelligence,
		DisplayName: "Driver Positioning",
		Description: "Sending an empty driver toward an area where freight is coming, from " +
			"the freight outlook. The driver is told in Dash, and the forecast and the " +
			"planner count them at the target until they arrive or it is called off.",
		Category:       "Dispatch",
		ParentResource: permission.ResourceWorker.String(),
		Operations: []permissionspec.OperationDefinition{
			{
				Operation:   permission.OpRead,
				DisplayName: "View",
				Description: "See which drivers are on their way to an area",
			},
			{
				Operation:   permission.OpCreate,
				DisplayName: "Send",
				Description: "Send a driver toward an area and tell them in Dash",
			},
			{
				Operation:   permission.OpCancel,
				DisplayName: "Call off",
				Description: "Call off a positioning and tell the driver",
			},
		},
		DefaultSensitivity: permission.SensitivityInternal,
	}
}

func shipmentHoldResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourceShipmentHold.String(),
		Feature:        platformcatalog.FeatureCoreTMS,
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
	}
}

func jurisdictionRuleOverrideResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceJurisdictionRuleOverride.String(),
		Feature:     platformcatalog.FeatureDispatch,
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
	}
}

func jurisdictionRuleResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceJurisdictionRule.String(),
		Feature:     platformcatalog.FeatureDispatch,
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
	}
}

func permitResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:       permission.ResourcePermit.String(),
		Feature:        platformcatalog.FeatureCoreTMS,
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
	}
}

func serviceFailureResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceServiceFailure.String(),
		Feature:     platformcatalog.FeatureCoreTMS,
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
		FieldFeatures: map[string]platformcatalog.FeatureKey{
			"x12StatusCodeOverride": platformcatalog.FeatureEDIIntegration,
			"x12ReasonCodeOverride": platformcatalog.FeatureEDIIntegration,
			"x12ExceptionCode":      platformcatalog.FeatureEDIIntegration,
		},
	}
}

func serviceFailureReasonCodeResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:      permission.ResourceServiceFailureReasonCode.String(),
		Feature:       platformcatalog.FeatureCoreTMS,
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
	}
}

func shipmentTypeResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceShipmentType.String(),
		Feature:            platformcatalog.FeatureCoreTMS,
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
	}
}

func dispatchControlResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceDispatchControl.String(),
		Feature:            platformcatalog.FeatureDispatch,
		DisplayName:        "Dispatch Control",
		Description:        "Dispatch settings and configuration",
		Category:           "Dispatch",
		Operations:         permissionspec.StandardOps,
		DefaultSensitivity: permission.SensitivityInternal,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"tenderCounterMarginFloorPercent": permission.SensitivityConfidential,
		},
	}
}

func distanceProfileResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceDistanceProfile.String(),
		Feature:            platformcatalog.FeatureDispatch,
		DisplayName:        "Distance Profile",
		Description:        "Distance calculation routing profiles",
		Category:           "Dispatch",
		Operations:         permissionspec.StandardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityInternal,
	}
}

func distanceOverrideResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceDistanceOverride.String(),
		Feature:            platformcatalog.FeatureDispatch,
		DisplayName:        "Distance Override",
		Description:        "Lane-specific distance overrides",
		Category:           "Dispatch",
		Operations:         permissionspec.StandardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityInternal,
	}
}

func distanceControlResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceDistanceControl.String(),
		Feature:            platformcatalog.FeatureDispatch,
		DisplayName:        "Distance Control",
		Description:        "Distance calculation settings and profile assignments",
		Category:           "Dispatch",
		Operations:         permissionspec.StandardOps,
		DefaultSensitivity: permission.SensitivityInternal,
	}
}

func storedMileageResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceStoredMileage.String(),
		Feature:            platformcatalog.FeatureDispatch,
		DisplayName:        "Stored Mileage",
		Description:        "Reusable stored mileage lane records",
		Category:           "Dispatch",
		Operations:         permissionspec.StandardOpsWithDelete,
		DefaultSensitivity: permission.SensitivityInternal,
	}
}

func dataEntryControlResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceDataEntryControl.String(),
		Feature:            platformcatalog.FeatureCoreTMS,
		DisplayName:        "Data Entry Control",
		Description:        "Data entry case formatting settings",
		Category:           "Organization settings",
		Operations:         permissionspec.StandardOps,
		DefaultSensitivity: permission.SensitivityInternal,
	}
}

func shipmentControlResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:           permission.ResourceShipmentControl.String(),
		Feature:            platformcatalog.FeatureCoreTMS,
		DisplayName:        "Shipment Control",
		Description:        "Shipment control settings and configuration",
		Category:           "Shipments",
		Operations:         permissionspec.StandardOps,
		DefaultSensitivity: permission.SensitivityRestricted,
	}
}

func holdReasonResource() *permissionspec.ResourceDefinition {
	return &permissionspec.ResourceDefinition{
		Resource:    permission.ResourceHoldReason.String(),
		Feature:     platformcatalog.FeatureCoreTMS,
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
	}
}
