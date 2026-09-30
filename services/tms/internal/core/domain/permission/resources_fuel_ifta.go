package permission

func (r *Registry) registerFuelAndIFTAResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceFuelCard.String(),
		DisplayName: "Fuel Card",
		Description: "Fuel cards issued by a card provider and assigned to a driver or tractor",
		Category:    "Fuel & IFTA",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View fuel cards"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Register a fuel card"},
			{
				Operation:   OpUpdate,
				DisplayName: "Update",
				Description: "Change a card's label, assignment or suspension",
			},
			{
				Operation:   OpCancel,
				DisplayName: "Cancel",
				Description: "Cancel a card permanently, with a reason",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export fuel card data"},
		},
		DefaultSensitivity: SensitivityInternal,
		FieldSensitivities: map[string]FieldSensitivity{
			"externalCardId": SensitivityConfidential,
		},
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceFuelPurchase.String(),
		DisplayName: "Fuel Purchase",
		Description: "Fuel bought for a tractor, recorded by hand or imported from a card statement",
		Category:    "Fuel & IFTA",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View fuel purchases"},
			{Operation: OpCreate, DisplayName: "Record", Description: "Record a fuel purchase"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Correct a fuel purchase"},
			{
				Operation:   OpDelete,
				DisplayName: "Delete",
				Description: "Remove a fuel purchase from the tax record",
			},
			{
				Operation:   OpImport,
				DisplayName: "Import",
				Description: "Stage and commit a fuel card statement",
			},
			{
				Operation:   OpCancel,
				DisplayName: "Discard import",
				Description: "Discard a staged statement before it is committed",
			},
			{Operation: OpExport, DisplayName: "Export", Description: "Export fuel purchase data"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceFuelPurchaseImport.String(),
		DisplayName:    "Fuel Purchase Import",
		Description:    "A fuel card statement staged for review before its rows become purchases",
		Category:       "Fuel & IFTA",
		ParentResource: ResourceFuelPurchase.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View staged statements"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Open a statement import"},
			{
				Operation:   OpImport,
				DisplayName: "Import",
				Description: "Parse a statement and commit its rows as purchases",
			},
			{
				Operation:   OpCancel,
				DisplayName: "Discard",
				Description: "Discard a statement that will not be committed",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceIFTAJurisdictionMileage.String(),
		DisplayName: "Jurisdiction Mileage",
		Description: "Miles a tractor ran in a jurisdiction that routing did not capture, " +
			"such as deadhead between shipments",
		Category: "Fuel & IFTA",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View mileage entries"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Record a mileage entry"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Correct a mileage entry"},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Remove a mileage entry"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export mileage entries"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceIFTAReturn.String(),
		DisplayName: "IFTA Return",
		Description: "Quarterly fuel tax return built from jurisdiction miles and tax-paid gallons",
		Category:    "Fuel & IFTA",
		Operations: []OperationDefinition{
			{
				Operation:   OpRead,
				DisplayName: "Read",
				Description: "View returns, their lines and diagnostics",
			},
			{
				Operation:   OpCreate,
				DisplayName: "Generate",
				Description: "Generate a draft return for a quarter or amend a filed one",
			},
			{
				Operation:   OpUpdate,
				DisplayName: "Recompute",
				Description: "Recompute a draft from current miles, fuel and rates",
			},
			{
				Operation:   OpApprove,
				DisplayName: "Finalize",
				Description: "Lock a draft so its figures no longer move",
			},
			{
				Operation:   OpReopen,
				DisplayName: "Reopen",
				Description: "Return a finalized return to draft, with a reason",
			},
			{
				Operation:   OpSubmit,
				DisplayName: "Mark filed",
				Description: "Record that a finalized return was filed with the base jurisdiction",
			},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Delete a draft return"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export a return worksheet"},
			{
				Operation:   OpManage,
				DisplayName: "Backfill mileage",
				Description: "Re-route completed moves to attribute miles by jurisdiction; " +
					"may be billed per move",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceIFTATaxRate.String(),
		DisplayName: "IFTA Tax Rate",
		Description: "Published per-gallon fuel tax rates by jurisdiction, quarter and fuel type. " +
			"This data is global: it has no organization column and every tenant " +
			"reads the same rows, so a change here alters what every organization's " +
			"return owes.",
		Category: "Fuel & IFTA",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View tax rates"},
			{
				Operation:   OpManage,
				DisplayName: "Manage",
				Description: "Add, correct or remove rates every organization is taxed at",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})
}
