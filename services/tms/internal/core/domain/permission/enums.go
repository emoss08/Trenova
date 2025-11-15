package permission

type Operation uint32

const (
	OpCreate Operation = 1 << iota // 1
	OpRead
	OpUpdate
	OpDelete
	OpExport
	OpImport
	OpApprove
	OpReject
	OpShare
	OpArchive
	OpRestore
	OpManage
	OpSubmit
	OpCopy
	OpAssign
	OpDuplicate
	OpClose
	OpLock
	OpUnlock
	OpActivate
)

func (o Operation) String() string {
	switch o {
	case OpCreate:
		return "create"
	case OpRead:
		return "read"
	case OpUpdate:
		return "update"
	case OpDelete:
		return "delete"
	case OpExport:
		return "export"
	case OpImport:
		return "import"
	case OpApprove:
		return "approve"
	case OpReject:
		return "reject"
	case OpShare:
		return "share"
	case OpArchive:
		return "archive"
	case OpRestore:
		return "restore"
	case OpManage:
		return "manage"
	case OpSubmit:
		return "submit"
	case OpCopy:
		return "copy"
	case OpAssign:
		return "assign"
	case OpDuplicate:
		return "duplicate"
	case OpClose:
		return "close"
	case OpLock:
		return "lock"
	case OpUnlock:
		return "unlock"
	case OpActivate:
		return "activate"
	default:
		return "unknown"
	}
}

func (o Operation) ToUint32() uint32 {
	return uint32(o)
}

type SubjectType string

const (
	SubjectTypeUser = SubjectType("user")
	SubjectTypeRole = SubjectType("role")
)

type MaskType string

const (
	MaskTypePartial = MaskType("partial") // Show partial data (e.g., ***-**-1234)
	MaskTypeFull    = MaskType("full")    // Hide completely (***********)
	MaskTypeHash    = MaskType("hash")    // Show hash (e.g., SHA256:abc123...)
)

type PolicyConditionType string

const (
	PolicyConditionTypeField     = PolicyConditionType("field")
	PolicyConditionTypeTime      = PolicyConditionType("time")
	PolicyConditionTypeIP        = PolicyConditionType("ip")
	PolicyConditionTypeAttribute = PolicyConditionType("attribute")
)

type DataScope string

const (
	DataScopeOwn          = DataScope("own")           // Only their records
	DataScopeOrganization = DataScope("organization")  // Current org only
	DataScopeBusinessUnit = DataScope("business_unit") // Across all orgs in BU
	DataScopeAll          = DataScope("all")           // Everything
)

type Effect string

const (
	EffectAllow = Effect("allow")
	EffectDeny  = Effect("deny")
)

type RoleLevel string

const (
	RoleLevelSystem = RoleLevel("system")
	RoleLevelBU     = RoleLevel("bu")
	RoleLevelOrg    = RoleLevel("org")
	RoleLevelCustom = RoleLevel("custom")
)

type ScopeType string

const (
	ScopeTypeBusinessUnit ScopeType = "business_unit"
	ScopeTypeOrganization ScopeType = "organization"
)

// TODO(Wolfred): All of the domains in this file should be moved to generated code at some point.
const (
	ResourceBusinessUnit = Resource(
		"business_unit",
	) // Represents resources related to business units.
	ResourceDocumentQualityConfig = Resource(
		"document_quality_config",
	) // Represents resources related to document quality config.
	ResourceConsolidationSettings = Resource(
		"consolidation_settings",
	) // Represents resources related to consolidation settings.
	ResourceDocument = Resource(
		"document",
	) // Represents resources related to documents.
	ResourceRole = Resource(
		"role",
	) // Represents resources related to roles.
	ResourcePageFavorite = Resource(
		"page_favorite",
	) // Represents resources related to page favorites.
	ResourceWorkerPTO = Resource(
		"worker_pto",
	) // Represents resources related to worker PTOs.
	ResourceShipmentHold = Resource(
		"shipment_hold",
	) // Represents resources for managing shipment holds.
	ResourceBillingClient = Resource(
		"billing_client",
	)
	ResourceAIClassification = Resource(
		"ai_classification",
	) // Represents resources for managing AI classifications.
	ResourceConsolidation = Resource(
		"consolidation",
	) // Represents resources for managing consolidation groups.
	ResourceBillingQueue = Resource(
		"billing_queue",
	) // Represents resources for managing billing queue.
	ResourceAssignment = Resource(
		"assignment",
	) // Represents resources for managing assignments.
	ResourceShipmentMove = Resource(
		"shipment_move",
	) // Represents resources for managing movements.
	ResourceStop = Resource(
		"stop",
	) // Represents resources for managing stops.

	ResourceFormat = Resource(
		"format",
	) // Represents resources for managing formats.
	ResourceInvoice = Resource(
		"invoice",
	) // Represents resources related to invoices.
	ResourceFormulaTemplate = Resource(
		"formula_template",
	) // Represents resources related to formula templates.
	ResourceDispatch  = Resource("dispatch") // Represents resources for dispatch management.
	ResourceReport    = Resource("report")   // Represents resources for managing reports.
	ResourceDashboard = Resource(
		"dashboard",
	) // Represents resources for managing dashboards.
	ResourceTableConfiguration = Resource(
		"table_configuration",
	) // Represents resources for managing table configurations.
	ResourceIntegration = Resource(
		"integration",
	) // Represents resources for integrations with external systems.
	ResourceSetting = Resource(
		"setting",
	) // Represents configuration or setting resources.
	ResourceTemplate = Resource(
		"template",
	) // Represents resources for managing templates.
	ResourceBackup = Resource(
		"backup",
	) // Represents resources for managing backups.
	ResourcePermission = Resource(
		"permission",
	) // Represents resources for managing permissions.
	ResourceAILog = Resource(
		"ai_log",
	) // Represents resources for managing ai logs.
	ResourceWorkflowExecution = Resource("workflow_execution")
	ResourceWorkflowTemplate  = Resource("workflow_template")
)
