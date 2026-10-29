package permissioncatalog

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

var jobTemplates = []permissionspec.JobTemplate{
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
