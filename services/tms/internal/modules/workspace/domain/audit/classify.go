package audit

import (
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
)

var financialCategories = map[string]struct{}{
	"Billing":             {},
	"Accounting":          {},
	"Payroll":             {},
	"Fuel & tax":          {},
	"Carrier Settlements": {},
	"Carrier settlements": {},
	"Detention":           {},
}

var (
	financialOnce      sync.Once
	financialResources map[permission.Resource]struct{}
)

func loadFinancialResources() {
	financialResources = make(map[permission.Resource]struct{})
	for _, def := range permissioncatalog.Default().All() {
		if _, ok := financialCategories[def.Category]; ok {
			financialResources[permission.Resource(def.Resource)] = struct{}{}
		}
	}
	financialResources[permission.ResourceRateConfirmation] = struct{}{}
}

func IsFinancialResource(resource permission.Resource) bool {
	financialOnce.Do(loadFinancialResources)
	_, ok := financialResources[resource]
	return ok
}

func ClassifyRetention(
	resource permission.Resource,
	operation permission.Operation,
	category Category,
) RetentionClass {
	switch {
	case category == CategorySecurity:
		return RetentionSecurity
	case operation == permission.OpRead:
		return RetentionAccess
	case IsFinancialResource(resource):
		return RetentionFinancial
	default:
		return RetentionOperational
	}
}
