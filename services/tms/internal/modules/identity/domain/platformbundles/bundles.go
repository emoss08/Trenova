package platformbundles

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func All() []permissionspec.Bundle {
	return []permissionspec.Bundle{
		{
			ID:          "edi.exchange",
			Module:      "EDI",
			Label:       "EDI",
			Description: "Watch EDI traffic, retry and correct messages, and keep partners and mappings current.",
			Grants: []permissionspec.BundleGrant{
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
			Grants: []permissionspec.BundleGrant{
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
			Grants: []permissionspec.BundleGrant{
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
			Grants: []permissionspec.BundleGrant{
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
			Grants: []permissionspec.BundleGrant{
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
			Grants: []permissionspec.BundleGrant{
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
			Grants: []permissionspec.BundleGrant{
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
			Grants: []permissionspec.BundleGrant{
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
			Grants: []permissionspec.BundleGrant{
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
			Grants: []permissionspec.BundleGrant{
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
			Grants: []permissionspec.BundleGrant{
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
}
