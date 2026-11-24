package boardregistry

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingboard"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediboard"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentboard"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionboard"
	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/parties/domain/partiesboard"
	"github.com/emoss08/trenova/internal/modules/places/domain/placesboard"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelboard"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/settlementboard"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workforceboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homeboard"
)

var catalog = boardcatalog.New(boardcatalog.Entries{
	Kinds: slices.Concat(
		operationsboard.Kinds,
		billingboard.Kinds,
		accountingboard.Kinds,
		partiesboard.Kinds,
		workforceboard.Kinds,
		equipmentboard.Kinds,
		placesboard.Kinds,
		detentionboard.Kinds,
		ediboard.Kinds,
		fuelboard.Kinds,
		settlementboard.Kinds,
	),
	Measures: slices.Concat(
		operationsboard.Measures,
		billingboard.Measures,
		accountingboard.Measures,
		workforceboard.Measures,
		detentionboard.Measures,
		ediboard.Measures,
		partiesboard.Measures,
		fuelboard.Measures,
		settlementboard.Measures,
	),
	Metrics: slices.Concat(
		homeboard.Metrics,
		billingboard.Metrics,
		workforceboard.Metrics,
		detentionboard.Metrics,
		operationsboard.Metrics,
		accountingboard.Metrics,
		ediboard.Metrics,
		fuelboard.Metrics,
		settlementboard.Metrics,
	),
})

func Catalog() *boardcatalog.Catalog {
	return catalog
}
