package permissioncatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

var capabilityBundles = []Bundle{
	{
		ID:          "dispatch.board",
		Module:      "Dispatch",
		Label:       "Dispatch board",
		Description: "Work the board: move loads, assign drivers and equipment, and keep stops and holds current.",
		Grants: []BundleGrant{
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
		Grants: []BundleGrant{
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
		Grants: []BundleGrant{
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
		Grants: []BundleGrant{
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
	{
		ID:          "billing.queue",
		Module:      "Billing",
		Label:       "Billing queue",
		Description: "Work the billing queue and prepare invoices for review.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceBillingQueue,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpUpdate,
					permission.OpAssign,
				},
			},
			{
				Resource:   permission.ResourceShipment,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource: permission.ResourceInvoice,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpSubmit,
				},
			},
			{
				Resource:   permission.ResourceAccessorialCharge,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceChargeType,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceDocument,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceCustomer,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "billing.post_invoices",
		Module:      "Billing",
		Label:       "Post invoices",
		Description: "Post, approve, reject and cancel invoices, run invoice batches and settle disputes.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceInvoice,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpPost,
					permission.OpApprove,
					permission.OpReject,
					permission.OpCancel,
					permission.OpExport,
				},
			},
			{
				Resource: permission.ResourceInvoiceRun,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpApprove,
					permission.OpCancel,
				},
			},
			{
				Resource: permission.ResourceInvoiceDispute,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpApprove,
					permission.OpCancel,
				},
			},
		},
	},
	{
		ID:          "billing.detention",
		Module:      "Billing",
		Label:       "Detention",
		Description: "Work the detention desk: approve, waive and dispute detention, and send notices.",
		Grants: []BundleGrant{
			{
				Resource:   permission.ResourceDetention,
				Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			},
			{
				Resource:   permission.ResourceDetentionPolicy,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceShipment,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceCustomer,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceLocation,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "billing.rates",
		Module:      "Billing",
		Label:       "Rates and agreements",
		Description: "Maintain rate agreements, matrices, zones, formulas and fuel surcharges.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceRateAgreement,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpSubmit,
					permission.OpDuplicate,
				},
			},
			{
				Resource: permission.ResourceRateMatrix,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpSubmit,
				},
			},
			{
				Resource: permission.ResourceRateZone,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource: permission.ResourceFormulaTemplate,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpDuplicate,
					permission.OpSubmit,
				},
			},
			{
				Resource: permission.ResourceFormulaLibrary,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpDuplicate,
					permission.OpSubmit,
				},
			},
			{
				Resource: permission.ResourceAccessorialCharge,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource: permission.ResourceFuelSurchargeProgram,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceRateSimulation,
				Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
			},
			{
				Resource:   permission.ResourceDetentionPolicy,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceProtectedMargins,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "billing.receivables",
		Module:      "Billing",
		Label:       "Receivables",
		Description: "Apply customer payments and bank receipts and work receivables.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceAccountsReceivable,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpUpdate,
					permission.OpExport,
				},
			},
			{
				Resource: permission.ResourceCustomerPayment,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource: permission.ResourceBankReceipt,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceBankReceiptWorkItem,
				Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			},
			{
				Resource:   permission.ResourceCustomer,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceInvoice,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceInvoiceDispute,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "accounting.ledger",
		Module:      "Accounting",
		Label:       "General ledger",
		Description: "Keep the chart of accounts and enter manual journals.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceGeneralLedgerAccount,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceJournalEntry,
				Operations: []permission.Operation{permission.OpRead, permission.OpExport},
			},
			{
				Resource: permission.ResourceManualJournal,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpSubmit,
				},
			},
			{
				Resource:   permission.ResourceJournalReversal,
				Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
			},
			{
				Resource:   permission.ResourceAccountType,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceDivisionCode,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceAccountingReport,
				Operations: []permission.Operation{permission.OpRead, permission.OpExport},
			},
		},
	},
	{
		ID:          "accounting.close",
		Module:      "Accounting",
		Label:       "Period close",
		Description: "Approve journals and close, reopen and lock fiscal periods.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceFiscalPeriod,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpClose,
					permission.OpReopen,
					permission.OpLock,
					permission.OpUnlock,
				},
			},
			{
				Resource: permission.ResourceFiscalYear,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpClose,
					permission.OpReopen,
				},
			},
			{
				Resource:   permission.ResourceJournalEntry,
				Operations: []permission.Operation{permission.OpRead, permission.OpApprove},
			},
			{
				Resource: permission.ResourceManualJournal,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpApprove,
					permission.OpReject,
				},
			},
			{
				Resource: permission.ResourceJournalReversal,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpApprove,
					permission.OpReject,
				},
			},
		},
	},
	{
		ID:          "settlement.drivers",
		Module:      "Settlements",
		Label:       "Driver settlements",
		Description: "Build driver settlements with their expenses, advances and disputes.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceDriverSettlement,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpSubmit,
				},
			},
			{
				Resource: permission.ResourceDriverExpense,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource: permission.ResourcePayAdvance,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceRecurringDeduction,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceRecurringEarning,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource: permission.ResourceSettlementDispute,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceEscrowAccount,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "settlement.carriers",
		Module:      "Settlements",
		Label:       "Carrier settlements",
		Description: "Match carrier invoices and build carrier settlements.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceCarrierSettlement,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpSubmit,
				},
			},
			{
				Resource: permission.ResourceCarrierInvoiceMatch,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceCarrier,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "settlement.approve",
		Module:      "Settlements",
		Label:       "Approve settlements",
		Description: "Approve, reject and cancel driver and carrier settlements.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceDriverSettlement,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpApprove,
					permission.OpReject,
					permission.OpCancel,
				},
			},
			{
				Resource:   permission.ResourceDriverExpense,
				Operations: []permission.Operation{permission.OpRead, permission.OpApprove},
			},
			{
				Resource:   permission.ResourceSettlementDispute,
				Operations: []permission.Operation{permission.OpRead, permission.OpApprove},
			},
			{
				Resource: permission.ResourceCarrierSettlement,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpApprove,
					permission.OpReject,
					permission.OpCancel,
				},
			},
			{
				Resource: permission.ResourceCarrierInvoiceMatch,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpApprove,
					permission.OpReject,
				},
			},
			{
				Resource:   permission.ResourceCarrier,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourcePayAdvance,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceEscrowAccount,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "safety.drivers",
		Module:      "Safety",
		Label:       "Driver safety and qualification",
		Description: "Keep qualifications, credentials, DOT tests, safety events and injuries.",
		Grants: []BundleGrant{
			{
				Resource:   permission.ResourceWorker,
				Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			},
			{
				Resource: permission.ResourceQualification,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource: permission.ResourceWorkerCredential,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpApprove,
				},
			},
			{
				Resource: permission.ResourceWorkerDOTTest,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource: permission.ResourceWorkerSafetyEvent,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpClose,
				},
			},
			{
				Resource: permission.ResourceWorkerInjury,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceDOTRandomPool,
				Operations: []permission.Operation{permission.OpRead, permission.OpManage},
			},
			{
				Resource: permission.ResourceDocument,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpApprove,
					permission.OpReject,
				},
			},
			{
				Resource:   permission.ResourceProtectedPersonal,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceWorkerCredentialType,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "safety.equipment",
		Module:      "Safety",
		Label:       "Equipment",
		Description: "Keep tractors and trailers and their types and fleets.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceTractor,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpArchive,
					permission.OpRestore,
				},
			},
			{
				Resource: permission.ResourceTrailer,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpArchive,
					permission.OpRestore,
				},
			},
			{
				Resource:   permission.ResourceEquipmentType,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceEquipmentManufacturer,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceFleetCode,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "compliance.fuel",
		Module:      "Safety",
		Label:       "Fuel and IFTA",
		Description: "Record fuel purchases and cards and prepare IFTA returns.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceFuelPurchase,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpImport,
				},
			},
			{
				Resource: permission.ResourceFuelCard,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource: permission.ResourceIFTAReturn,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpSubmit,
				},
			},
			{
				Resource: permission.ResourceIFTAJurisdictionMileage,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceIFTATaxRate,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "hr.workers",
		Module:      "Workforce",
		Label:       "Worker records",
		Description: "Keep worker records, employment events, positions and checklists, and invite drivers to the portal or revoke their access.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceWorker,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpArchive,
					permission.OpRestore,
				},
			},
			{
				Resource: permission.ResourceWorkerEmploymentEvent,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceJobPosition,
				Operations: []permission.Operation{permission.OpRead, permission.OpAssign},
			},
			{
				Resource: permission.ResourceWorkerChecklist,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceWorkerPolicy,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceProtectedPersonal,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource: permission.ResourceDriverPortal,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpCancel,
				},
			},
		},
	},
	{
		ID:          "hr.benefits",
		Module:      "Workforce",
		Label:       "Benefits",
		Description: "Keep benefit plans and enroll workers on them.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceBenefitPlan,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpAssign,
				},
			},
		},
	},
	{
		ID:          "hr.performance",
		Module:      "Workforce",
		Label:       "Reviews and discipline",
		Description: "Write performance reviews and issue disciplinary actions.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourcePerformanceReview,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceWorkerDisciplinaryAction,
				Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
			},
		},
	},
	{
		ID:          "hr.leave",
		Module:      "Workforce",
		Label:       "Leave and FMLA",
		Description: "Open and decide leave cases, record leave days and keep medical certification.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceWorkerLeave,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpApprove,
				},
			},
		},
	},
	{
		ID:          "hr.team",
		Module:      "Workforce",
		Label:       "Manage a team",
		Description: "Approve time off, timesheets, swaps and profile changes, schedule a team, and invite its drivers to the portal or revoke their access.",
		Grants: []BundleGrant{
			{
				Resource:   permission.ResourceWorker,
				Operations: []permission.Operation{permission.OpRead},
				Scope:      permission.DataScopeTeam,
			},
			{
				Resource: permission.ResourceWorkerPTO,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpApprove,
					permission.OpReject,
				},
			},
			{
				Resource:   permission.ResourceWorkerLeave,
				Operations: []permission.Operation{permission.OpRead, permission.OpApprove},
				Scope:      permission.DataScopeTeam,
			},
			{
				Resource: permission.ResourceTimesheet,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpApprove,
					permission.OpReject,
				},
				Scope: permission.DataScopeTeam,
			},
			{
				Resource: permission.ResourceShiftSwap,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpApprove,
					permission.OpReject,
				},
			},
			{
				Resource: permission.ResourceProfileChangeRequest,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpApprove,
					permission.OpReject,
				},
				Scope: permission.DataScopeTeam,
			},
			{
				Resource: permission.ResourceWorkerSchedule,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpAssign,
					permission.OpUpdate,
				},
				Scope: permission.DataScopeTeam,
			},
			{
				Resource:   permission.ResourceShiftTemplate,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource: permission.ResourceDriverPortal,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpCancel,
				},
				Scope: permission.DataScopeTeam,
			},
		},
	},
	{
		ID:          "edi.exchange",
		Module:      "EDI",
		Label:       "EDI",
		Description: "Watch EDI traffic, retry and correct messages, and keep partners and mappings current.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceEDI,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpImport,
					permission.OpExport,
				},
			},
		},
	},
	{
		ID:          "customers.records",
		Module:      "Customers",
		Label:       "Customers and locations",
		Description: "Keep customers, locations and commodities.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceCustomer,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpArchive,
					permission.OpRestore,
				},
			},
			{
				Resource: permission.ResourceLocation,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpArchive,
					permission.OpRestore,
				},
			},
			{
				Resource:   permission.ResourceLocationCategory,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource: permission.ResourceCommodity,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceHazardousMaterial,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "reporting.reports",
		Module:      "Reporting",
		Label:       "Reports and dashboards",
		Description: "Build, run and export reports and dashboards, and share table views with the organization.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceReport,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpExport,
				},
			},
			{
				Resource: permission.ResourceDashboard,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource: permission.ResourceTableConfiguration,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
		},
	},
	{
		ID:          "platform.assistant",
		Module:      "Desk",
		Label:       "Desk",
		Description: "Talk with the agents on the Desk, and read briefings and insights.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceAssistant,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpDelete,
				},
			},
			{
				Resource:   permission.ResourceBriefing,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceInsight,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceWebResearch,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourcePortalAutomation,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceCodeMode,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "desk.decisions",
		Module:      "Desk",
		Label:       "Decisions",
		Description: "Approve or decline what the agents propose, and work the watchtower's exceptions.",
		Grants: []BundleGrant{
			{
				Resource:   permission.ResourceAgentProposal,
				Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			},
			{
				Resource:   permission.ResourceWatchtower,
				Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			},
		},
	},
	{
		ID:          "desk.triage",
		Module:      "Desk",
		Label:       "Triage",
		Description: "Work the messages that arrive from connected sources: link them to records, reply, hand them off and mark them handled.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceInboundMessage,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpUpdate,
					permission.OpAssign,
				},
			},
			{
				Resource:   permission.ResourceDocument,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "desk.intake",
		Module:      "Desk",
		Label:       "Intake",
		Description: "Scan and print into Trenova, and file what arrives onto its records.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceCaptureBatch,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceCaptureDevice,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceDocument,
				Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
			},
		},
	},
	{
		ID:          "admin.users",
		Module:      "Administration",
		Label:       "Users",
		Description: "Invite, change and deactivate people and give them roles.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceUser,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpAssign,
					permission.OpUnassign,
				},
			},
			{
				Resource:   permission.ResourceRole,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "admin.roles",
		Module:      "Administration",
		Label:       "Roles",
		Description: "Create, change and delete roles.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceRole,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpDelete,
				},
			},
			{
				Resource:   permission.ResourceUser,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceAgentDefinition,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
	{
		ID:          "admin.organization",
		Module:      "Administration",
		Label:       "Organization settings",
		Description: "Change the organization's profile and operating controls.",
		Grants: []BundleGrant{
			{
				Resource:   permission.ResourceOrganization,
				Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			},
			{
				Resource:   permission.ResourceBusinessUnit,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceSequenceConfig,
				Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			},
			{
				Resource:   permission.ResourceDataEntryControl,
				Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			},
			{
				Resource:   permission.ResourceShipmentControl,
				Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			},
			{
				Resource:   permission.ResourceDispatchControl,
				Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			},
			{
				Resource:   permission.ResourceBillingControl,
				Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			},
		},
	},
	{
		ID:          "admin.security",
		Module:      "Administration",
		Label:       "Security and sign-in",
		Description: "Configure single sign-on, SCIM, access policies and API keys, and review sign-in activity.",
		Grants: []BundleGrant{
			{
				Resource: permission.ResourceIdentityProvider,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpDelete,
				},
			},
			{
				Resource: permission.ResourceSCIMDirectory,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpDelete,
				},
			},
			{
				Resource: permission.ResourceAccessPolicy,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
					permission.OpDelete,
				},
			},
			{
				Resource:   permission.ResourceAuthEvent,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceRiskDecision,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceExternalIdentity,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceMFAAuthenticator,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceProvisioningAudit,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceAuditLog,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource: permission.ResourceAPIKey,
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
			},
			{
				Resource:   permission.ResourceUser,
				Operations: []permission.Operation{permission.OpRead},
			},
			{
				Resource:   permission.ResourceRole,
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	},
}

var jobTemplates = []JobTemplate{
	{
		ID:          "dispatcher",
		Label:       "Dispatcher",
		Description: "Books shipments and runs the board.",
		Bundles: []string{
			"dispatch.board",
			"dispatch.shipments",
			"dispatch.exceptions",
			"desk.triage",
			"platform.assistant",
		},
		MaxSensitivity: permission.SensitivityInternal,
	},
	{
		ID:          "dispatch_supervisor",
		Label:       "Dispatch supervisor",
		Description: "Everything a dispatcher does, plus carrier coverage and customer records.",
		Bundles: []string{
			"dispatch.board",
			"dispatch.shipments",
			"dispatch.exceptions",
			"dispatch.carriers",
			"customers.records",
			"reporting.reports",
			"desk.triage",
			"desk.decisions",
			"platform.assistant",
		},
		MaxSensitivity: permission.SensitivityRestricted,
	},
	{
		ID:             "carrier_rep",
		Label:          "Carrier sales",
		Description:    "Covers brokered loads with carriers.",
		Bundles:        []string{"dispatch.carriers", "dispatch.board", "platform.assistant"},
		MaxSensitivity: permission.SensitivityInternal,
	},
	{
		ID:          "billing_clerk",
		Label:       "Billing clerk",
		Description: "Prepares invoices and applies payments.",
		Bundles: []string{
			"billing.queue",
			"billing.receivables",
			"desk.intake",
			"platform.assistant",
		},
		MaxSensitivity: permission.SensitivityInternal,
	},
	{
		ID:          "billing_manager",
		Label:       "Billing manager",
		Description: "Posts invoices and keeps rates, on top of the clerk's work.",
		Bundles: []string{
			"billing.queue",
			"billing.post_invoices",
			"billing.rates",
			"billing.receivables",
			"billing.detention",
			"desk.intake",
			"desk.decisions",
			"reporting.reports",
			"platform.assistant",
		},
		MaxSensitivity: permission.SensitivityRestricted,
	},
	{
		ID:          "accountant",
		Label:       "Accountant",
		Description: "Keeps the ledger and closes periods.",
		Bundles: []string{
			"accounting.ledger",
			"accounting.close",
			"billing.receivables",
			"reporting.reports",
			"platform.assistant",
		},
		MaxSensitivity: permission.SensitivityConfidential,
	},
	{
		ID:          "payroll",
		Label:       "Settlements",
		Description: "Builds and approves driver and carrier settlements.",
		Bundles: []string{
			"settlement.drivers",
			"settlement.carriers",
			"settlement.approve",
			"platform.assistant",
		},
		MaxSensitivity: permission.SensitivityConfidential,
	},
	{
		ID:          "safety",
		Label:       "Safety and compliance",
		Description: "Keeps drivers qualified, equipment current and fuel tax filed.",
		Bundles: []string{
			"safety.drivers",
			"safety.equipment",
			"compliance.fuel",
			"platform.assistant",
		},
		MaxSensitivity: permission.SensitivityRestricted,
	},
	{
		ID:             "driver_manager",
		Label:          "Driver manager",
		Description:    "Looks after the drivers on their team.",
		Bundles:        []string{"hr.team", "platform.assistant"},
		MaxSensitivity: permission.SensitivityRestricted,
	},
	{
		ID:          "hr",
		Label:       "Human resources",
		Description: "Keeps worker records and approves time off for everyone.",
		Bundles: []string{
			"hr.workers",
			"hr.benefits",
			"hr.performance",
			"hr.leave",
			"hr.team",
			"platform.assistant",
		},
		MaxSensitivity: permission.SensitivityConfidential,
	},
	{
		ID:             "read_only",
		Label:          "Read-only",
		Description:    "Can read every record except HR, medical, drug testing and pay records, and change nothing.",
		MaxSensitivity: permission.SensitivityInternal,
		ReadEverything: true,
		ReadExcept:     ownGrantKinds,
	},
}

var ownGrantKinds = []permission.Resource{
	permission.ResourceWorkerLeave,
	permission.ResourceWorkerInjury,
	permission.ResourceWorkerDOTTest,
	permission.ResourceBenefitPlan,
	permission.ResourceWorkerEmploymentEvent,
	permission.ResourceWorkerDisciplinaryAction,
	permission.ResourcePerformanceReview,
	permission.ResourceTimesheet,
	permission.ResourceProfileChangeRequest,
}
