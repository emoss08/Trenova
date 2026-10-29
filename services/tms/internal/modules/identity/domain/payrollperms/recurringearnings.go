package payrollperms

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

const (
	fieldGLAccountID = "glAccountId"
)

func recurringEarningResources() []*permissionspec.ResourceDefinition {
	return []*permissionspec.ResourceDefinition{
		{
			Resource:           permission.ResourceRecurringEarning.String(),
			DisplayName:        "Recurring Earning",
			Description:        "Recurring driver earning schedules such as per diem and bonuses",
			Category:           "Payroll",
			Operations:         permissionspec.StandardOpsWithDelete,
			DefaultSensitivity: permission.SensitivityRestricted,
			FieldSensitivities: map[string]permission.FieldSensitivity{
				"businessUnitId":  permission.SensitivityInternal,
				"createdAt":       permission.SensitivityInternal,
				"createdById":     permission.SensitivityInternal,
				"currencyCode":    permission.SensitivityInternal,
				"description":     permission.SensitivityInternal,
				"endDate":         permission.SensitivityInternal,
				"frequency":       permission.SensitivityInternal,
				"id":              permission.SensitivityInternal,
				"kind":            permission.SensitivityInternal,
				"organizationId":  permission.SensitivityInternal,
				"payCodeId":       permission.SensitivityInternal,
				"startDate":       permission.SensitivityInternal,
				"status":          permission.SensitivityInternal,
				"updatedAt":       permission.SensitivityInternal,
				"version":         permission.SensitivityInternal,
				"workerId":        permission.SensitivityInternal,
				"amountMinor":     permission.SensitivityRestricted,
				"paidToDateMinor": permission.SensitivityRestricted,
				"totalCapMinor":   permission.SensitivityRestricted,
				"payCode":         permission.SensitivityInternal,
				"worker":          permission.SensitivityInternal,
			},
		},

		{
			Resource:           permission.ResourcePayCode.String(),
			DisplayName:        "Pay Code",
			Description:        "Carrier-defined earning and deduction codes with GL account mappings",
			Category:           "Payroll",
			ReferenceView:      []string{"code", "name", "description"},
			Operations:         permissionspec.StandardOpsWithDelete,
			DefaultSensitivity: permission.SensitivityRestricted,
			FieldSensitivities: map[string]permission.FieldSensitivity{
				"businessUnitId":        permission.SensitivityInternal,
				"code":                  permission.SensitivityInternal,
				"countsTowardGuarantee": permission.SensitivityInternal,
				"createdAt":             permission.SensitivityInternal,
				"description":           permission.SensitivityInternal,
				"direction":             permission.SensitivityInternal,
				fieldGLAccountID:        permission.SensitivityInternal,
				"id":                    permission.SensitivityInternal,
				"isSystem":              permission.SensitivityInternal,
				"name":                  permission.SensitivityInternal,
				"organizationId":        permission.SensitivityInternal,
				"status":                permission.SensitivityInternal,
				"taxable":               permission.SensitivityInternal,
				"updatedAt":             permission.SensitivityInternal,
				"version":               permission.SensitivityInternal,
				"defaultAmountMinor":    permission.SensitivityRestricted,
			},
		},
	}
}
