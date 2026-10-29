package financebundles

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func All() []permissionspec.Bundle {
	return []permissionspec.Bundle{
		billingQueueBundle(),
		billingPostInvoicesBundle(),
		billingDetentionBundle(),
		billingRatesBundle(),
		billingReceivablesBundle(),
		accountingLedgerBundle(),
		accountingCloseBundle(),
		settlementDriversBundle(),
		settlementCarriersBundle(),
		settlementApproveBundle(),
	}
}

func billingQueueBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "billing.queue",
		Module:      "Billing",
		Label:       "Billing queue",
		Description: "Work the billing queue and prepare invoices for review.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func billingPostInvoicesBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "billing.post_invoices",
		Module:      "Billing",
		Label:       "Post invoices",
		Description: "Post, approve, reject and cancel invoices, run invoice batches and settle disputes.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func billingDetentionBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "billing.detention",
		Module:      "Billing",
		Label:       "Detention",
		Description: "Work the detention desk: approve, waive and dispute detention, and send notices.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func billingRatesBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "billing.rates",
		Module:      "Billing",
		Label:       "Rates and agreements",
		Description: "Maintain rate agreements, matrices, zones, formulas and fuel surcharges.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func billingReceivablesBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "billing.receivables",
		Module:      "Billing",
		Label:       "Receivables",
		Description: "Apply customer payments and bank receipts and work receivables.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func accountingLedgerBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "accounting.ledger",
		Module:      "Accounting",
		Label:       "General ledger",
		Description: "Keep the chart of accounts and enter manual journals.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func accountingCloseBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "accounting.close",
		Module:      "Accounting",
		Label:       "Period close",
		Description: "Approve journals and close, reopen and lock fiscal periods.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func settlementDriversBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "settlement.drivers",
		Module:      "Settlements",
		Label:       "Driver settlements",
		Description: "Build driver settlements with their expenses, advances and disputes.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func settlementCarriersBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "settlement.carriers",
		Module:      "Settlements",
		Label:       "Carrier settlements",
		Description: "Match carrier invoices and build carrier settlements.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func settlementApproveBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "settlement.approve",
		Module:      "Settlements",
		Label:       "Approve settlements",
		Description: "Approve, reject and cancel driver and carrier settlements.",
		Grants: []permissionspec.BundleGrant{
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
	}
}
