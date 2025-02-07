package permission

import (
	"github.com/bytedance/sonic"
)

type Status string

const (
	StatusActive    = Status("Active")    // Active permissions.
	StatusInactive  = Status("Inactive")  // Inactive permissions.
	StatusSuspended = Status("Suspended") // Temporarily suspended permissions.
	StatusArchived  = Status("Archived")  // Archived permissions, no longer active.
)

type Resource string

const (
	// Core resources
	ResourceUser                  = Resource("user")                    // Represents user management resources.
	ResourceBusinessUnit          = Resource("business_unit")           // Represents resources related to business units.
	ResourceOrganization          = Resource("organization")            // Represents resources related to organizations.
	ResourceDocumentQualityConfig = Resource("document_quality_config") // Represents resources related to document quality config.

	// Operations resources
	ResourceWorker                = Resource("worker")                 // Represents resources related to workers.
	ResourceTractor               = Resource("tractor")                // Represents resources for managing tractors.
	ResourceTrailer               = Resource("trailer")                // Represents resources for managing trailers.
	ResourceShipment              = Resource("shipment")               // Represents resources for managing shipments.
	ResourceAssignment            = Resource("assignment")             // Represents resources for managing assignments.
	ResourceShipmentMove          = Resource("shipment_move")          // Represents resources for managing movements.
	ResourceFleetCode             = Resource("fleet_code")             // Represents resources for managing fleet codes.
	ResourceEquipmentType         = Resource("equipment_type")         // Represents resources for managing equipment types.
	ResourceEquipmentManufacturer = Resource("equipment_manufacturer") // Represents resources for managing equipment manfacturers.
	ResourceShipmentType          = Resource("shipment_type")          // Represents resources for managing shipment type.
	ResourceServiceType           = Resource("service_type")           // Represents resources for managing service types.
	ResourceHazardousMaterial     = Resource("hazardous_material")     // Represents resources for managing hazardous materials.
	ResourceCommodity             = Resource("commodity")              // Represents resources for managing commodities.
	ResourceLocationCategory      = Resource("location_category")      // Represents resources for managing location categories.
	ResourceLocation              = Resource("location")               // Represents resources for managing locations.
	ResourceCustomer              = Resource("customer")               // Represents resources for managing customers.

	// Financial resources
	ResourceInvoice = Resource("invoice") // Represents resources related to invoices.

	// Management resources
	ResourceDispatch = Resource("dispatch")  // Represents resources for dispatch management.
	ResourceReport   = Resource("report")    // Represents resources for managing reports.
	ResourceAuditLog = Resource("audit_log") // Represents resources for tracking and auditing logs.

	// System resources
	ResourceTableConfiguration = Resource("table_configuration") // Represents resources for managing table configurations.
	ResourceIntegration        = Resource("integration")         // Represents resources for integrations with external systems.
	ResourceSetting            = Resource("setting")             // Represents configuration or setting resources.
	ResourceTemplate           = Resource("template")            // Represents resources for managing templates.
)

func (r Resource) MarshalJSON() ([]byte, error) {
	return sonic.Marshal(string(r))
}

type Action string

const (
	// Standard CRUD
	ActionCreate = Action("create") // Create a new resource.
	ActionRead   = Action("read")   // Read or view a resource.
	ActionUpdate = Action("update") // Update an existing resource.
	ActionDelete = Action("delete") // Delete an existing resource.

	// Field-level actions
	ActionModifyField = Action("modify_field") // Modify specific fields in a resource.
	ActionViewField   = Action("view_field")   // View specific fields in a resource.

	// Workflow actions
	ActionApprove  = Action("approve")  // Approve an action or resource.
	ActionReject   = Action("reject")   // Reject an action or resource.
	ActionSubmit   = Action("submit")   // Submit an action or resource for approval.
	ActionCancel   = Action("cancel")   // Cancel an action or resource.
	ActionAssign   = Action("assign")   // Assign a resource to a user or group.
	ActionReassign = Action("reassign") // Reassign a resource to a different user or group.
	ActionComplete = Action("complete") // Mark a resource or action as completed.

	// Configuration actions
	ActionManageDefaults = Action("manage_defaults") // Manage default table configurations.
	ActionShare          = Action("share")           // Share a table configuration with others.

	// Data actions
	ActionExport  = Action("export")  // Export data from the system.
	ActionImport  = Action("import")  // Import data into the system.
	ActionArchive = Action("archive") // Archive a resource.
	ActionRestore = Action("restore") // Restore an archived resource.

	// Administrative actions
	ActionManage    = Action("manage")    // Perform administrative actions, including full access.
	ActionAudit     = Action("audit")     // Audit actions for compliance and review.
	ActionDelegate  = Action("delegate")  // Delegate permissions or responsibilities to others.
	ActionConfigure = Action("configure") // Configure system settings or resources.
)

type Scope string

const (
	ScopeGlobal   = Scope("global")        // Permissions apply globally across all scopes.
	ScopeBU       = Scope("business_unit") // Permissions are limited to a specific business unit.
	ScopeOrg      = Scope("organization")  // Permissions are limited to a specific organization.
	ScopePersonal = Scope("personal")      // Permissions are limited to the individual user or resource.
)

func (s Scope) MarshalJSON() ([]byte, error) {
	return sonic.Marshal(string(s))
}

// Operator types for conditions
type Operator string

const (
	OpEquals      = Operator("eq")           // Checks if a value equals another.
	OpNotEquals   = Operator("neq")          // Checks if a value does not equal another.
	OpGreaterThan = Operator("gt")           // Checks if a value is greater than another.
	OpLessThan    = Operator("lt")           // Checks if a value is less than another.
	OpIn          = Operator("in")           // Checks if a value exists within a set of values.
	OpNotIn       = Operator("not_in")       // Checks if a value does not exist within a set of values.
	OpContains    = Operator("contains")     // Checks if a value contains another value (e.g., substring match).
	OpNotContains = Operator("not_contains") // Checks if a value does not contain another value.
)

// AuditLevel defines how changes to a field should be tracked
type AuditLevel string

const (
	AuditNone    = AuditLevel("none")    // No auditing for the field.
	AuditChanges = AuditLevel("changes") // Track only changes to the field.
	AuditAccess  = AuditLevel("access")  // Track all access events for the field.
	AuditFull    = AuditLevel("full")    // Track all actions, including changes and views.
)

func (a Action) MarshalJSON() ([]byte, error) {
	return sonic.Marshal(string(a))
}

type FieldPermission struct {
	Field           string         `json:"field"`                     // The field name
	Actions         []Action       `json:"actions"`                   // Actions that can be performed on the field
	Conditions      []*Condition   `json:"conditions,omitempty"`      // Conditions for the field
	ValidationRules map[string]any `json:"validationRules,omitempty"` // Custom validation rules for the field
	Mask            string         `json:"mask,omitempty"`            // Data masking pattern
	AuditLevel      AuditLevel     `json:"auditLevel,omitempty"`      // Level of auditing for this field
}

type ConditionType string

const (
	ConditionTypeField     = ConditionType("field")     // Field-based condition checks.
	ConditionTypeTime      = ConditionType("time")      // Time-based condition checks.
	ConditionTypeRole      = ConditionType("role")      // Role-based condition checks.
	ConditionTypeOwnership = ConditionType("ownership") // Ownership-based condition checks.
	ConditionTypeCustom    = ConditionType("custom")    // Custom condition checks defined by the user.
)

type Condition struct {
	Type         ConditionType  `json:"type"`
	Field        string         `json:"field"`
	Operator     string         `json:"operator"`
	Value        any            `json:"value"`
	Values       []any          `json:"values,omitempty"`
	Description  string         `json:"description,omitempty"`  // Human-readable description
	ErrorMessage string         `json:"errorMessage,omitempty"` // Custom error message
	Priority     int            `json:"priority"`               // Evaluation priority
	Metadata     map[string]any `json:"metadata,omitempty"`     // Additional condition metadata
}

type RoleType string

const (
	RoleTypeSystem       = RoleType("System")       // Predefined system-level roles.
	RoleTypeOrganization = RoleType("Organization") // Organization-specific roles.
	RoleTypeCustom       = RoleType("Custom")       // User-defined roles.
	RoleTypeTemporary    = RoleType("Temporary")    // Temporary roles for specific use cases.
)

var (
	// Base actions that most resources have
	BaseActions = []Action{
		ActionCreate,
		ActionRead,
		ActionUpdate,
		ActionDelete,
		ActionManage,
	}

	// Actions for resources that can be archived
	ArchivableActions = []Action{
		ActionArchive,
		ActionRestore,
	}

	// Actions for workflow-based resources
	WorkflowActions = []Action{
		ActionApprove,
		ActionReject,
		ActionSubmit,
		ActionCancel,
	}

	// Actions for assignable resources
	AssignableActions = []Action{
		ActionAssign,
		ActionReassign,
	}

	// Actions for resources that support import/export
	DataActions = []Action{
		ActionExport,
		ActionImport,
	}

	// Actions for table configuration resources
	TableConfigurationActions = []Action{
		ActionManageDefaults,
	}

	// Field-level actions
	FieldActions = []Action{
		ActionModifyField,
		ActionViewField,
	}

	// Resource-specific action mappings
	ResourceActionMap = map[Resource][]Action{
		// Core resources
		ResourceUser: append(
			BaseActions,
			ActionDelegate,
		),
		ResourceBusinessUnit: append(
			BaseActions,
			ActionConfigure,
			ActionAudit,
		),
		ResourceOrganization: append(
			BaseActions,
			ActionConfigure,
			ActionAudit,
			ActionModifyField,
		),

		// Operations resources
		ResourceWorker: append(
			BaseActions,
			append(AssignableActions, FieldActions...)...,
		),
		ResourceTractor: append(
			BaseActions,
			append(AssignableActions, FieldActions...)...,
		),
		ResourceTrailer: append(
			BaseActions,
			append(AssignableActions, FieldActions...)...,
		),
		ResourceShipment: append(
			append(BaseActions, WorkflowActions...),
			append(AssignableActions,
				ActionComplete,
				ActionModifyField,
				ActionViewField,
			)...,
		),
		ResourceAssignment: {
			ActionAssign, // can the user assign the move to the worker?
			ActionRead,   // can the user view the assignment?
			ActionCancel, // can the user cancel the assignment?
			ActionAudit,  // can the user view the audit logs for the assignment?
			ActionManage, // does the user have permission to manage the assignment? (Full access)
		},
		ResourceShipmentMove: append(
			append(BaseActions, WorkflowActions...),
			append(DataActions, FieldActions...)...,
		),
		ResourceFleetCode: append(
			BaseActions,
			append(DataActions, FieldActions...)...,
		),
		ResourceDocumentQualityConfig: append(
			BaseActions,
			append(DataActions, FieldActions...)...,
		),
		ResourceEquipmentType: append(
			BaseActions,
			append(DataActions, FieldActions...)...,
		),
		ResourceEquipmentManufacturer: append(
			BaseActions,
			append(DataActions, FieldActions...)...,
		),
		ResourceShipmentType: append(
			BaseActions,
			append(DataActions, FieldActions...)...,
		),
		ResourceServiceType: append(
			BaseActions,
			append(DataActions, FieldActions...)...,
		),
		ResourceHazardousMaterial: append(
			BaseActions,
			append(DataActions, FieldActions...)...,
		),
		ResourceCommodity: append(
			BaseActions,
			append(DataActions, FieldActions...)...,
		),
		ResourceLocationCategory: append(
			BaseActions,
			append(DataActions, FieldActions...)...,
		),
		ResourceLocation: append(
			BaseActions,
			append(DataActions, FieldActions...)...,
		),
		ResourceCustomer: append(
			BaseActions,
			append(DataActions, FieldActions...)...,
		),

		// Financial resources
		ResourceInvoice: append(
			append(BaseActions, WorkflowActions...),
			append(DataActions, FieldActions...)...,
		),
		// Management resources
		ResourceDispatch: append(
			BaseActions,
			append(AssignableActions,
				ActionComplete,
				ActionModifyField,
				ActionViewField,
				ActionCancel,
			)...,
		),
		ResourceReport: append(
			BaseActions,
			ActionExport,
		),
		ResourceAuditLog: {
			ActionRead,
			ActionExport,
			ActionManage,
		},
		ResourceTableConfiguration: append(
			BaseActions,
			TableConfigurationActions...,
		),

		// System resources
		ResourceSetting: append(
			BaseActions,
			ActionConfigure,
			ActionAudit,
		),
		ResourceIntegration: append(
			BaseActions,
			ActionConfigure,
			ActionDelegate,
		),
		ResourceTemplate: append(
			BaseActions,
			append(DataActions, ArchivableActions...)...,
		),
	}
)

type RolesAndPermissions struct {
	Roles       []*string
	Permissions []*Permission
}
