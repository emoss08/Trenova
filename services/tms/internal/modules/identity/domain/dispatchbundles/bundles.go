package dispatchbundles

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func All() []permissionspec.Bundle {
	return []permissionspec.Bundle{
		{
			ID:          "dispatch.board",
			Module:      "Dispatch",
			Label:       "Dispatch board",
			Description: "Work the board: move loads, assign drivers and equipment, and keep stops and holds current.",
			Grants: []permissionspec.BundleGrant{
				{
					Resource:   permission.ResourceShipment,
					Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
				},
				{
					Resource: permission.ResourceShipmentMove,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpUpdate,
						permission.OpAssign,
						permission.OpUnassign,
					},
				},
				{
					Resource:   permission.ResourceShipmentStop,
					Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
				},
				{
					Resource: permission.ResourceShipmentComment,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
					},
				},
				{
					Resource: permission.ResourceShipmentHold,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
					},
				},
				{
					Resource:   permission.ResourceWorker,
					Operations: []permission.Operation{permission.OpRead},
				},
				{
					Resource:   permission.ResourceTractor,
					Operations: []permission.Operation{permission.OpRead},
				},
				{
					Resource:   permission.ResourceTrailer,
					Operations: []permission.Operation{permission.OpRead},
				},
				{
					Resource:   permission.ResourceLocation,
					Operations: []permission.Operation{permission.OpRead},
				},
				{
					Resource:   permission.ResourceCustomer,
					Operations: []permission.Operation{permission.OpRead},
				},
				{
					Resource:   permission.ResourceWatchtower,
					Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
				},
				{
					Resource:   permission.ResourceDriverMessage,
					Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
				},
				{
					Resource:   permission.ResourceWorkerDispatchHold,
					Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
				},
				{
					Resource:   permission.ResourceLocationCategory,
					Operations: []permission.Operation{permission.OpRead},
				},
			},
		},
		{
			ID:          "dispatch.shipments",
			Module:      "Dispatch",
			Label:       "Book shipments",
			Description: "Enter, change, duplicate and cancel shipments and orders.",
			Grants: []permissionspec.BundleGrant{
				{
					Resource: permission.ResourceShipment,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
						permission.OpSubmit,
						permission.OpCancel,
						permission.OpDuplicate,
						permission.OpExport,
					},
				},
				{
					Resource: permission.ResourceShipmentMove,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
					},
				},
				{
					Resource: permission.ResourceShipmentStop,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
					},
				},
				{
					Resource: permission.ResourceOrder,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
						permission.OpCancel,
					},
				},
				{
					Resource: permission.ResourceRecurringShipment,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
						permission.OpDuplicate,
					},
				},
				{
					Resource:   permission.ResourceCommodity,
					Operations: []permission.Operation{permission.OpRead},
				},
				{
					Resource:   permission.ResourceHazardousMaterial,
					Operations: []permission.Operation{permission.OpRead},
				},
				{
					Resource:   permission.ResourceCustomer,
					Operations: []permission.Operation{permission.OpRead},
				},
				{
					Resource:   permission.ResourceLocation,
					Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
				},
				{
					Resource:   permission.ResourceRateQuote,
					Operations: []permission.Operation{permission.OpRead},
				},
			},
		},
		{
			ID:          "dispatch.carriers",
			Module:      "Dispatch",
			Label:       "Carrier coverage",
			Description: "Source, tender and confirm carriers for brokered loads.",
			Grants: []permissionspec.BundleGrant{
				{
					Resource: permission.ResourceCarrier,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
					},
				},
				{
					Resource: permission.ResourceTender,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
						permission.OpCancel,
						permission.OpApprove,
					},
				},
				{
					Resource: permission.ResourceRateConfirmation,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
					},
				},
				{
					Resource:   permission.ResourceRoutingGuide,
					Operations: []permission.Operation{permission.OpRead},
				},
				{
					Resource: permission.ResourceCarrierCapacityPosting,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
					},
				},
				{
					Resource:   permission.ResourceCarrierSourcing,
					Operations: []permission.Operation{permission.OpRead},
				},
				{
					Resource:   permission.ResourceCarrierIntelligence,
					Operations: []permission.Operation{permission.OpRead},
				},
				{
					Resource:   permission.ResourceEquipmentVerification,
					Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
				},
			},
		},
		{
			ID:          "dispatch.exceptions",
			Module:      "Dispatch",
			Label:       "Service failures and holds",
			Description: "Record service failures and place and release holds.",
			Grants: []permissionspec.BundleGrant{
				{
					Resource: permission.ResourceServiceFailure,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
					},
				},
				{
					Resource: permission.ResourceShipmentHold,
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
					},
				},
				{
					Resource:   permission.ResourceHoldReason,
					Operations: []permission.Operation{permission.OpRead},
				},
				{
					Resource:   permission.ResourceServiceFailureReasonCode,
					Operations: []permission.Operation{permission.OpRead},
				},
			},
		},
	}
}
