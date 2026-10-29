package workforcebundles

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func All() []permissionspec.Bundle {
	return []permissionspec.Bundle{
		safetyDriversBundle(),
		safetyEquipmentBundle(),
		complianceFuelBundle(),
		hrWorkersBundle(),
		hrBenefitsBundle(),
		hrPerformanceBundle(),
		hrLeaveBundle(),
		hrTeamBundle(),
	}
}

func safetyDriversBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "safety.drivers",
		Module:      "Safety",
		Label:       "Driver safety and qualification",
		Description: "Keep qualifications, credentials, DOT tests, safety events and injuries.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func safetyEquipmentBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "safety.equipment",
		Module:      "Safety",
		Label:       "Equipment",
		Description: "Keep tractors and trailers and their types and fleets.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func complianceFuelBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "compliance.fuel",
		Module:      "Safety",
		Label:       "Fuel and IFTA",
		Description: "Record fuel purchases and cards and prepare IFTA returns.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func hrWorkersBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "hr.workers",
		Module:      "Workforce",
		Label:       "Worker records",
		Description: "Keep worker records, employment events, positions and checklists, and invite drivers to the portal or revoke their access.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func hrBenefitsBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "hr.benefits",
		Module:      "Workforce",
		Label:       "Benefits",
		Description: "Keep benefit plans and enroll workers on them.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func hrPerformanceBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "hr.performance",
		Module:      "Workforce",
		Label:       "Reviews and discipline",
		Description: "Write performance reviews and issue disciplinary actions.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func hrLeaveBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "hr.leave",
		Module:      "Workforce",
		Label:       "Leave and FMLA",
		Description: "Open and decide leave cases, record leave days and keep medical certification.",
		Grants: []permissionspec.BundleGrant{
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
	}
}

func hrTeamBundle() permissionspec.Bundle {
	return permissionspec.Bundle{
		ID:          "hr.team",
		Module:      "Workforce",
		Label:       "Manage a team",
		Description: "Approve time off, timesheets, swaps and profile changes, schedule a team, and invite its drivers to the portal or revoke their access.",
		Grants: []permissionspec.BundleGrant{
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
	}
}
