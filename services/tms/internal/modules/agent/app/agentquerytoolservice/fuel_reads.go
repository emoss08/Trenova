package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
)

const (
	toolListFuelCards            = "fuel_list_cards"
	toolListFuelPurchases        = "fuel_list_purchases"
	toolListFuelPurchaseImports  = "fuel_list_purchase_imports"
	toolListIFTAReturns          = "fuel_list_ifta_returns"
	toolListIFTAMileageEntries   = "fuel_list_ifta_mileage_entries"
	toolListIFTAJurisdictions    = "fuel_list_ifta_jurisdictions"
	fieldAssignedTractorID       = "assignedTractorId"
	fieldAssignedWorkerID        = "assignedWorkerId"
	fieldFuelCardID              = "fuelCardId"
	fieldImportBatchID           = "importBatchId"
	fieldWorkerID                = "workerId"
	fieldDiscoveredAt            = "discoveredAt"
	fieldProblems                = "problems"
	fieldProblemCode             = "code"
	rowUnassigned                = "unassigned"
	rowDiscoveredByFeed          = "discoveredByFeed"
	rowBlockingProblems          = "blockingProblems"
	paramActiveOnly              = "activeOnly"
	noteParsedImports            = "Parsed imports are waiting to be committed."
	noteActiveJurisdictionsByDef = "Defaults to true."

	fuelCardsSummary = "List the organization's fuel cards with the provider, the last four " +
		"digits, the status and the tractor or driver each is assigned to. A card a feed " +
		"discovered is Suspended and unassigned until someone assigns it with " +
		"fuel_assign_card, and until then its purchases wait in their import."
	fuelPurchasesSummary = "List fuel bought for tractors, newest first, with where it was " +
		"bought for IFTA, the fuel type, the gallons and the amount. Each also says whether " +
		"tax was paid at the pump. Use it to find a purchase to correct with " +
		"fuel_correct_purchase or fuel_delete_purchase, or to check a receipt was " +
		"already recorded before fuel_record_purchase."
	fuelPurchaseImportsSummary = "List fuel card statements and feed runs staged for import, " +
		"newest first, with how many rows are ready to commit, already on file, duplicated " +
		"or in error. A Parsed import waits for fuel_commit_purchase_import or " +
		"fuel_discard_purchase_import; rows held because a card or tractor was unknown " +
		"are worked out again with fuel_resolve_purchase_import_rows."
	iftaReturnsSummary = "List the organization's quarterly IFTA fuel tax returns with their " +
		"status, miles, gallons, the net tax due and the problems each carries. A Draft is " +
		"recomputed with fuel_recalculate_ifta_return or deleted with " +
		"fuel_delete_ifta_return; a Filed return is corrected with fuel_amend_ifta_return."
	iftaMileageEntriesSummary = "List the jurisdiction miles entered by hand or from " +
		"telematics rather than routed from moves. Each has the tractor, the state or " +
		"province, the day and the quarter it counts in. fuel_correct_ifta_mileage_entry " +
		"and fuel_delete_ifta_mileage_entry take their ids."
	iftaJurisdictionsSummary = "List the active states and provinces fuel tax is reported " +
		"by, with each one's two-letter code and whether it is an IFTA member. Its id is " +
		"what fuel_record_purchase and fuel_record_ifta_mileage_entry take. Narrow by " +
		"country code, US, CA or MX."
)

var (
	tractorReference = idReference{
		resource: permission.ResourceTractor,
		what:     "The tractor",
		source:   toolListTractors,
	}
	jurisdictionReference = idReference{
		resource: permission.ResourceFuelPurchase,
		kinds:    []permission.RecordKind{permission.KindIFTAJurisdiction},
		what:     "The state or province",
		source:   toolListIFTAJurisdictions,
	}
)

// fuelReadSpecs are the fuel and fuel tax lists, read through the operations
// the Desk sends for them: the cards, the purchases and the statements they
// were imported from, the quarterly returns, the miles entered by hand and
// the jurisdictions every one of them names.
func fuelReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		{
			name:         toolListFuelCards,
			entityPlural: "fuel cards",
			summary:      fuelCardsSummary,
			resource:     permission.ResourceFuelCard,
			references: map[string]idReference{
				fieldAssignedTractorID: tractorReference,
				fieldAssignedWorkerID:  workerReference,
			},
			shape: shapeFuelCard,
		},
		{
			name:         toolListFuelPurchases,
			entityPlural: "fuel purchases",
			summary:      fuelPurchasesSummary,
			resource:     permission.ResourceFuelPurchase,
			references: map[string]idReference{
				fieldTractorID:      tractorReference,
				fieldWorkerID:       workerReference,
				fieldJurisdictionID: jurisdictionReference,
				fieldFuelCardID: {
					resource: permission.ResourceFuelCard,
					what:     "The fuel card",
					source:   toolListFuelCards,
				},
				fieldImportBatchID: {
					resource: permission.ResourceFuelPurchaseImport,
					what:     "The import it came in",
					source:   toolListFuelPurchaseImports,
				},
			},
		},
		{
			name:         toolListFuelPurchaseImports,
			entityPlural: "fuel purchase imports",
			summary:      fuelPurchaseImportsSummary,
			resource:     permission.ResourceFuelPurchaseImport,
			notes:        map[string]string{paramStatus: noteParsedImports},
		},
		{
			name:         toolListIFTAReturns,
			entityPlural: "IFTA returns",
			summary:      iftaReturnsSummary,
			resource:     permission.ResourceIFTAReturn,
			shape:        shapeIFTAReturn,
		},
		{
			name:         toolListIFTAMileageEntries,
			entityPlural: "IFTA mileage entries",
			summary:      iftaMileageEntriesSummary,
			resource:     permission.ResourceIFTAJurisdictionMileage,
			references: map[string]idReference{
				fieldTractorID:      tractorReference,
				fieldJurisdictionID: jurisdictionReference,
			},
		},
		{
			name:         toolListIFTAJurisdictions,
			entityPlural: "IFTA jurisdictions",
			summary:      iftaJurisdictionsSummary,
			resource:     permission.ResourceFuelPurchase,
			notes:        map[string]string{paramActiveOnly: noteActiveJurisdictionsByDef},
			defaults:     map[string]any{paramActiveOnly: true},
		},
	}
}

// shapeFuelCard says, as the repository row did, whether a card is carried
// by no tractor and no driver, and whether a feed created it: a model reads
// a missing assignment as unknown, not as none.
func shapeFuelCard(row map[string]any, withheld map[string]struct{}, _ clock) {
	_, tractorHidden := withheld[fieldAssignedTractorID]
	_, workerHidden := withheld[fieldAssignedWorkerID]
	if !tractorHidden && !workerHidden {
		_, tractor := row[fieldAssignedTractorID]
		_, worker := row[fieldAssignedWorkerID]
		row[rowUnassigned] = !tractor && !worker
	}
	if _, hidden := withheld[fieldDiscoveredAt]; !hidden {
		_, discovered := row[fieldDiscoveredAt]
		row[rowDiscoveredByFeed] = discovered
	}
}

// shapeIFTAReturn counts the problems that block finalizing a return, as
// the repository row did, by the domain's own rule.
func shapeIFTAReturn(row map[string]any, withheld map[string]struct{}, _ clock) {
	if _, hidden := withheld[fieldProblems]; hidden {
		return
	}
	problems, _ := row[fieldProblems].([]any)
	blocking := 0
	for _, raw := range problems {
		problem, _ := raw.(map[string]any)
		code, _ := problem[fieldProblemCode].(string)
		if ifta.ProblemCode(code).Blocks() {
			blocking++
		}
	}
	row[rowBlockingProblems] = blocking
}
