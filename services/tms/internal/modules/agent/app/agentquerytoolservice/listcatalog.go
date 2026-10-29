package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/accountingquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/billingquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/equipmentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/formulaquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/masterdataquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/operationsquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/partyquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/shipmentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/workerquerytools"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

// The catalog is a curated set, not blanket coverage. There are well over a
// hundred permission resources; publishing a tool per resource would expose
// columns nobody vetted and give the model a menu it cannot reason about. These
// are the entities the assistant is actually asked about.

// listCatalogSpecs mirrors the registered tools so a test can hold every entry
// against the entity it claims to query. A curated field that stops mapping to a
// column is dropped silently by the query builder, which is the one failure the
// catalog cannot afford.
func listCatalogSpecs() []listtool.Spec {
	return []listtool.Spec{
		listtool.SpecOf(workerquerytools.NewListWorkersTool(nil, nil)),
		shipmentquerytools.ListSpec(nil),
		listtool.SpecOf(equipmentquerytools.NewListTractorsTool(nil, nil)),
		listtool.SpecOf(equipmentquerytools.NewListTrailersTool(nil)),
		listtool.SpecOf(partyquerytools.NewListCustomersTool(nil)),
		listtool.SpecOf(partyquerytools.NewListLocationsTool(nil)),
		listtool.SpecOf(billingquerytools.NewListInvoicesTool(nil, nil)),
		listtool.SpecOf(billingquerytools.NewListBillingQueueItemsTool(nil, nil)),
		listtool.SpecOf(partyquerytools.NewListCarriersTool(nil)),
		listtool.SpecOf(masterdataquerytools.NewListEquipmentTypesTool(nil)),
		listtool.SpecOf(masterdataquerytools.NewListFleetCodesTool(nil)),
		listtool.SpecOf(masterdataquerytools.NewListServiceTypesTool(nil)),
		listtool.SpecOf(masterdataquerytools.NewListShipmentTypesTool(nil)),
		listtool.SpecOf(masterdataquerytools.NewListCommoditiesTool(nil)),
		listtool.SpecOf(partyquerytools.NewListCarrierCapacityPostingsTool(nil)),
		listtool.SpecOf(masterdataquerytools.NewListHazardousMaterialsTool(nil)),
		listtool.SpecOf(masterdataquerytools.NewListAccessorialChargesTool(nil)),
		listtool.SpecOf(masterdataquerytools.NewListDocumentTypesTool(nil)),
		listtool.SpecOf(masterdataquerytools.NewListLocationCategoriesTool(nil)),
		listtool.SpecOf(masterdataquerytools.NewListHoldReasonsTool(nil)),
		listtool.SpecOf(accountingquerytools.NewListGLAccountsTool(nil, nil)),
		listtool.SpecOf(accountingquerytools.NewListFiscalPeriodsTool(nil, nil)),
		listtool.SpecOf(operationsquerytools.NewListOrdersTool(nil, nil)),
		listtool.SpecOf(operationsquerytools.NewListRecurringShipmentsTool(nil)),
		listtool.SpecOf(formulaquerytools.NewListFormulaTemplatesTool(nil)),
		listtool.SpecOf(masterdataquerytools.NewListEquipmentManufacturersTool(nil)),
	}
}

// FilterCatalog is the vocabulary every catalogued resource answers to.
//
// It is built from the same specs the list tools are, with nil repositories,
// because what a caller may narrow an entity by is a property of the entity
// and not of the connection. That is what lets the Ask input on a data table
// compile "shipments in transit" into the filters shipment_list would have
// built from the same words.
func FilterCatalog() *filtercatalog.Catalog {
	specs := listCatalogSpecs()
	resources := make([]filtercatalog.Resource, 0, len(specs))
	for _, spec := range specs {
		resources = append(resources, listtool.CatalogResource(spec))
	}

	return filtercatalog.New(resources...)
}
