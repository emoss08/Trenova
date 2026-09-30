package permission

func (r *Registry) registerOperationsResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceOrder.String(),
		DisplayName: "Order",
		Description: "Commercial order management (groups shipments/legs under one customer order)",
		Category:    "Operations",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View orders"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create new orders"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify orders"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export order data"},
			{Operation: OpCancel, DisplayName: "Cancel", Description: "Cancel orders"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceShipment.String(),
		DisplayName: "Shipment",
		Description: "Shipment management",
		Category:    "Operations",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View shipments"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create new shipments"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify shipments"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export shipment data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import shipments"},
			{
				Operation:   OpSubmit,
				DisplayName: "Submit",
				Description: "Submit shipments for dispatch",
			},
			{Operation: OpCancel, DisplayName: "Cancel", Description: "Cancel shipments"},
			{Operation: OpDuplicate, DisplayName: "Duplicate", Description: "Duplicate shipments"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceRecurringShipment.String(),
		DisplayName:    "Recurring Shipment",
		Description:    "Recurring shipment series management and scheduled generation",
		Category:       "Operations",
		ParentResource: ResourceShipment.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View recurring shipments"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Create recurring shipments",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Modify recurring shipments, pause and resume series",
			},
			{
				Operation:   OpDuplicate,
				DisplayName: "Generate",
				Description: "Generate shipments from a recurring series",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceShipmentComment.String(),
		DisplayName:    "Shipment Comment",
		Description:    "Shipment comment management",
		Category:       "Operations",
		ParentResource: ResourceShipment.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View shipment comments"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add shipment comments"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify shipment comments"},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Delete shipment comments"},
			{
				Operation:   OpPin,
				DisplayName: "Pin",
				Description: "Pin comments to the top of a shipment",
			},
			{
				Operation:   OpUnpin,
				DisplayName: "Unpin",
				Description: "Remove pinned comments from the top of a shipment",
			},
			{
				Operation:   OpResolve,
				DisplayName: "Resolve",
				Description: "Resolve and reopen shipment comments",
			},
			{
				Operation:   OpManage,
				DisplayName: "Manage",
				Description: "Edit or delete any user's shipment comments",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceShipmentMove.String(),
		DisplayName:    "Shipment Move",
		Description:    "Shipment move management",
		Category:       "Operations",
		ParentResource: ResourceShipment.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View moves"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add moves"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify moves"},
			{Operation: OpAssign, DisplayName: "Assign", Description: "Assign resources to moves"},
			{
				Operation:   OpUnassign,
				DisplayName: "Unassign",
				Description: "Remove assignments from moves",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export shipment move data"},
		},
		DefaultSensitivity: SensitivityInternal,
		FieldSensitivities: map[string]FieldSensitivity{
			"externalDriverName":  SensitivityInternal,
			"externalDriverPhone": SensitivityRestricted,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceShipmentStop.String(),
		DisplayName:    "Shipment Stop",
		Description:    "Shipment stop management",
		Category:       "Operations",
		ParentResource: ResourceShipment.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View stops"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add stops"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify stops"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export shipment stop data"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceWorkerDispatchHold.String(),
		DisplayName: "Driver Dispatch Hold",
		Description: "Whether a driver can be given new freight. Held apart from the " +
			"worker record on purpose: taking somebody off the board until a paper is " +
			"renewed is a dispatch decision, and it is not the same permission as " +
			"editing their employment, their pay or their profile.",
		Category:       "Operations",
		ParentResource: ResourceWorker.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "See who is held off dispatch"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Take a driver off dispatch, and put them back",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceShipmentHold.String(),
		DisplayName:    "Shipment Hold",
		Description:    "Shipment hold management",
		Category:       "Operations",
		ParentResource: ResourceShipment.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View shipment holds"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create shipment holds"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Modify and release shipment holds",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceJurisdictionRuleOverride.String(),
		DisplayName: "Carrier Override",
		Description: "This organization's stricter posture on a jurisdiction. " +
			"Unlike a jurisdiction rule these are yours alone, and an override can " +
			"only tighten a state limit, never loosen one.",
		Category:       "Operations",
		ParentResource: ResourceJurisdictionRule.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View carrier overrides"},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Hold this fleet to a stricter limit than a state requires",
			},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify an override"},
			{
				Operation:   OpDelete,
				DisplayName: "Delete",
				Description: "Remove an override, returning the jurisdiction to its statutory limits",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceJurisdictionRule.String(),
		DisplayName: "Jurisdiction Rule",
		Description: "Shared oversize and overweight limits per state. " +
			"This data is global: it has no organization column and every tenant " +
			"reads the same rows, so a change here alters what the whole platform " +
			"is told is legal. Carrier-specific posture belongs in an override.",
		Category: "Operations",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View jurisdiction limits and their verification state",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Add a jurisdiction rule, visible to every organization",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Change limits every organization is held to",
			},
			{
				Operation:   OpApprove,
				DisplayName: "Verify",
				Description: "Confirm a rule against the issuing state, or mark it disputed",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourcePermit.String(),
		DisplayName:    "Oversize Permit",
		Description:    "Oversize and overweight permits and their derived requirements",
		Category:       "Operations",
		ParentResource: ResourceShipment.String(),
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View permits and permit requirements",
			},
			{Operation: OpCreate, DisplayName: "Create", Description: "Record permits"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify recorded permits"},
			{
				Operation:   OpApprove,
				DisplayName: "Waive",
				Description: "Waive a permit requirement, accepting the compliance risk",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceServiceFailure.String(),
		DisplayName: "Service Failure",
		Description: "Service failure investigation and approval records",
		Category:    "Operations",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View service failures"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create service failures"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify service failures"},
			{Operation: OpApprove, DisplayName: "Approve", Description: "Approve service failures"},
			{Operation: OpArchive, DisplayName: "Archive", Description: "Archive service failures"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export service failures"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceServiceFailureReasonCode.String(),
		DisplayName: "Service Failure Reason Code",
		Description: "Service failure reason code reference data",
		Category:    "Operations",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View service failure reason codes",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Create",
				Description: "Create service failure reason codes",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Modify service failure reason codes",
			},
			{
				Operation:   OpApprove,
				DisplayName: "Approve",
				Description: "Approve service failure reason codes",
			},
			{
				Operation:   OpArchive,
				DisplayName: "Archive",
				Description: "Archive service failure reason codes",
			},
			{
				Operation:   OpExport,
				DisplayName: "Export",
				Description: "Export service failure reason codes",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceShipmentType.String(),
		DisplayName:        "Shipment Type",
		Description:        "Shipment type definitions",
		Category:           "Operations",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"id":             SensitivityInternal,
			"businessUnitId": SensitivityInternal,
			"organizationId": SensitivityInternal,
			"status":         SensitivityInternal,
			"code":           SensitivityInternal,
			"description":    SensitivityInternal,
			"color":          SensitivityInternal,
			"version":        SensitivityInternal,
			"createdAt":      SensitivityInternal,
			"updatedAt":      SensitivityInternal,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDispatchControl.String(),
		DisplayName:        "Dispatch Control",
		Description:        "Dispatch settings and configuration",
		Category:           "Operations",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDistanceProfile.String(),
		DisplayName:        "Distance Profile",
		Description:        "Distance calculation routing profiles",
		Category:           "Operations",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDistanceOverride.String(),
		DisplayName:        "Distance Override",
		Description:        "Lane-specific distance overrides",
		Category:           "Operations",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDistanceControl.String(),
		DisplayName:        "Distance Control",
		Description:        "Distance calculation settings and profile assignments",
		Category:           "Operations",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceStoredMileage.String(),
		DisplayName:        "Stored Mileage",
		Description:        "Reusable stored mileage lane records",
		Category:           "Operations",
		Operations:         standardOpsWithDelete,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDataEntryControl.String(),
		DisplayName:        "Data Entry Control",
		Description:        "Data entry case formatting settings",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceShipmentControl.String(),
		DisplayName:        "Shipment Control",
		Description:        "Shipment control settings and configuration",
		Category:           "Operations",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceHoldReason.String(),
		DisplayName: "Hold Reason",
		Description: "Hold reason management",
		Category:    "Operations",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View hold reasons"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create new hold reasons"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify hold reasons"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export hold reason data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import hold reasons"},
		},
		DefaultSensitivity: SensitivityInternal,
	})
}
