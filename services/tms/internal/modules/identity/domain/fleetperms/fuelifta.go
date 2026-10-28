package fleetperms

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func fuelAndIFTAResources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		{
			Resource:      permission.ResourceFuelCard.String(),
			DisplayName:   "Fuel Card",
			Description:   "Fuel cards issued by a card provider and assigned to a driver or tractor",
			Category:      "Fuel & tax",
			ReferenceView: []string{"label", "lastFour", "provider", "status"},
			Operations: []permissionspec.OperationDefinition{
				{Operation: permission.OpRead, DisplayName: "View", Description: "View fuel cards"},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Register a fuel card",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Change a card's label, assignment or suspension",
				},
				{
					Operation:   permission.OpCancel,
					DisplayName: "Cancel",
					Description: "Cancel a card permanently, with a reason",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export fuel card data",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
			FieldSensitivities: map[string]permission.FieldSensitivity{
				"externalCardId": permission.SensitivityConfidential,
			},
		},

		{
			Resource:    permission.ResourceFuelPurchase.String(),
			DisplayName: "Fuel Purchase",
			Description: "Fuel bought for a tractor, recorded by hand or imported from a card statement",
			Category:    "Fuel & tax",
			ReferenceView: []string{
				"purchasedAt",
				"vendor",
				"vendorCity",
				"fuelType",
				"tractorId",
				"workerId",
			},
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View fuel purchases",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Record",
					Description: "Record a fuel purchase",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Correct a fuel purchase",
				},
				{
					Operation:   permission.OpDelete,
					DisplayName: "Delete",
					Description: "Remove a fuel purchase from the tax record",
				},
				{
					Operation:   permission.OpImport,
					DisplayName: "Import",
					Description: "Stage and commit a fuel card statement",
				},
				{
					Operation:   permission.OpCancel,
					DisplayName: "Discard import",
					Description: "Discard a staged statement before it is committed",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export fuel purchase data",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:       permission.ResourceFuelPurchaseImport.String(),
			DisplayName:    "Fuel Purchase Import",
			Description:    "A fuel card statement staged for review before its rows become purchases",
			Category:       "Fuel & tax",
			ParentResource: permission.ResourceFuelPurchase.String(),
			ReferenceView:  []string{"fileName", "origin", "provider", "rowCount"},
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View staged statements",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Open a statement import",
				},
				{
					Operation:   permission.OpImport,
					DisplayName: "Import",
					Description: "Parse a statement, resolve rows that need review and commit them as purchases",
				},
				{
					Operation:   permission.OpCancel,
					DisplayName: "Discard",
					Description: "Discard a statement that will not be committed",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:    permission.ResourceIFTAJurisdictionMileage.String(),
			DisplayName: "Jurisdiction Mileage",
			Description: "Miles a tractor ran in a jurisdiction that routing did not capture, " +
				"such as deadhead between shipments",
			Category: "Fuel & tax",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View mileage entries",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Create",
					Description: "Record a mileage entry",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Update",
					Description: "Correct a mileage entry",
				},
				{
					Operation:   permission.OpDelete,
					DisplayName: "Delete",
					Description: "Remove a mileage entry",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export mileage entries",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:    permission.ResourceIFTAReturn.String(),
			DisplayName: "IFTA Return",
			Description: "Quarterly fuel tax return built from jurisdiction miles and tax-paid gallons",
			Category:    "Fuel & tax",
			Operations: []permissionspec.OperationDefinition{
				{
					Operation:   permission.OpRead,
					DisplayName: "View",
					Description: "View returns, their lines and diagnostics",
				},
				{
					Operation:   permission.OpCreate,
					DisplayName: "Generate",
					Description: "Generate a draft return for a quarter or amend a filed one",
				},
				{
					Operation:   permission.OpUpdate,
					DisplayName: "Recompute",
					Description: "Recompute a draft from current miles, fuel and rates",
				},
				{
					Operation:   permission.OpApprove,
					DisplayName: "Finalize",
					Description: "Lock a draft so its figures no longer move",
				},
				{
					Operation:   permission.OpReopen,
					DisplayName: "Reopen",
					Description: "Return a finalized return to draft, with a reason",
				},
				{
					Operation:   permission.OpSubmit,
					DisplayName: "Mark filed",
					Description: "Record that a finalized return was filed with the base jurisdiction",
				},
				{
					Operation:   permission.OpDelete,
					DisplayName: "Delete",
					Description: "Delete a draft return",
				},
				{
					Operation:   permission.OpExport,
					DisplayName: "Export",
					Description: "Export a return worksheet",
				},
				{
					Operation:   permission.OpManage,
					DisplayName: "Backfill mileage",
					Description: "Re-route completed moves to attribute miles by jurisdiction; " +
						"may be billed per move",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},

		{
			Resource:    permission.ResourceIFTATaxRate.String(),
			DisplayName: "IFTA Tax Rate",
			Description: "Published per-gallon fuel tax rates by jurisdiction, quarter and fuel type. " +
				"This data is global: it has no organization column and every tenant " +
				"reads the same rows, so a change here alters what every organization's " +
				"return owes.",
			Category: "Fuel & tax",
			Operations: []permissionspec.OperationDefinition{
				{Operation: permission.OpRead, DisplayName: "View", Description: "View tax rates"},
				{
					Operation:   permission.OpManage,
					DisplayName: "Manage",
					Description: "Add, correct or remove rates every organization is taxed at",
				},
			},
			DefaultSensitivity: permission.SensitivityInternal,
		},
	}
}
