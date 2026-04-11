package permission

import (
	"errors"
	"fmt"
	"sync"
)

type OperationDefinition struct {
	Operation   Operation `json:"operation"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
}

type ResourceDefinition struct {
	Resource           string                      `json:"resource"`
	DisplayName        string                      `json:"displayName"`
	Description        string                      `json:"description"`
	Category           string                      `json:"category"`
	Operations         []OperationDefinition       `json:"operations"`
	CompositeOps       map[string][]Operation      `json:"compositeOps,omitempty"`
	ParentResource     string                      `json:"parentResource,omitempty"`
	FieldSensitivities map[string]FieldSensitivity `json:"fieldSensitivities,omitempty"`
	DefaultSensitivity FieldSensitivity            `json:"defaultSensitivity"`
}

type Registry struct {
	mu        sync.RWMutex
	resources map[string]*ResourceDefinition
	children  map[string][]string
}

func NewRegistry() *Registry {
	r := &Registry{
		resources: make(map[string]*ResourceDefinition),
		children:  make(map[string][]string),
	}
	r.registerAll()
	return r
}

func NewEmptyRegistry() *Registry {
	return &Registry{
		resources: make(map[string]*ResourceDefinition),
		children:  make(map[string][]string),
	}
}

func (r *Registry) Register(def *ResourceDefinition) error {
	if def.Resource == "" {
		return errors.New("resource name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.resources[def.Resource]; exists {
		return fmt.Errorf("resource %s already registered", def.Resource)
	}

	if def.DefaultSensitivity == "" {
		def.DefaultSensitivity = SensitivityInternal
	}

	r.resources[def.Resource] = def

	if def.ParentResource != "" {
		r.children[def.ParentResource] = append(r.children[def.ParentResource], def.Resource)
	}

	return nil
}

func (r *Registry) Get(resource string) (*ResourceDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.resources[resource]
	return def, ok
}

func (r *Registry) GetChildren(parent string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.children[parent]
}

func (r *Registry) GetEffectiveResource(resource string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if def, ok := r.resources[resource]; ok && def.ParentResource != "" {
		return def.ParentResource
	}
	return resource
}

func (r *Registry) All() []*ResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ResourceDefinition, 0, len(r.resources))
	for _, def := range r.resources {
		result = append(result, def)
	}
	return result
}

func (r *Registry) GetFieldSensitivity(resource, field string) FieldSensitivity {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.resources[resource]
	if !ok {
		return SensitivityInternal
	}
	if sens, sensOK := def.FieldSensitivities[field]; sensOK {
		return sens
	}
	return def.DefaultSensitivity
}

func (r *Registry) GetOperationsForResource(resource string) []Operation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.resources[resource]
	if !ok {
		return nil
	}

	ops := make([]Operation, 0, len(def.Operations))
	for _, opDef := range def.Operations {
		ops = append(ops, opDef.Operation)
	}
	return ops
}

func (r *Registry) ExpandCompositeOperation(resource, compositeName string) []Operation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.resources[resource]
	if !ok {
		return nil
	}

	if ops, opsOK := def.CompositeOps[compositeName]; opsOK {
		return ops
	}
	return nil
}

func (r *Registry) HasResource(resource string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.resources[resource]
	return ok
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.resources)
}

func (r *Registry) GetByCategory(category string) []*ResourceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ResourceDefinition
	for _, def := range r.resources {
		if def.Category == category {
			result = append(result, def)
		}
	}
	return result
}

func (r *Registry) GetCategories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	categories := make(map[string]bool)
	for _, def := range r.resources {
		if def.Category != "" {
			categories[def.Category] = true
		}
	}

	result := make([]string, 0, len(categories))
	for cat := range categories {
		result = append(result, cat)
	}
	return result
}

func (r *Registry) registerAll() {
	r.registerAdministrationResources()
	r.registerEquipmentResources()
	r.registerWorkerResources()
	r.registerOperationsResources()
	r.registerBillingResources()
	r.registerCustomerResources()
	r.registerLocationResources()
	r.registerCommodityResources()
	r.registerAccountingResources()
	r.registerComplianceResources()
	r.registerReferenceDataResources()
	r.registerReportingResources()
}

var standardOps = []OperationDefinition{
	{Operation: OpRead, DisplayName: "Read", Description: "View records"},
	{Operation: OpCreate, DisplayName: "Create", Description: "Create new records"},
	{Operation: OpUpdate, DisplayName: "Update", Description: "Modify existing records"},
	{Operation: OpExport, DisplayName: "Export", Description: "Export records to file"},
	{Operation: OpImport, DisplayName: "Import", Description: "Import records from file"},
}

var readOnlyOps = []OperationDefinition{
	{Operation: OpRead, DisplayName: "Read", Description: "View records"},
}

func (r *Registry) registerAdministrationResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceOrganization.String(),
		DisplayName:        "Organization",
		Description:        "Organization settings and configuration",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceBusinessUnit.String(),
		DisplayName:        "Business Unit",
		Description:        "Business unit management",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceUser.String(),
		DisplayName: "User",
		Description: "User account management",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View user accounts"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create new users"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify user accounts"},
			{
				Operation:   OpAssign,
				DisplayName: "Assign Roles",
				Description: "Assign roles to users",
			},
			{
				Operation:   OpUnassign,
				DisplayName: "Unassign Roles",
				Description: "Remove roles from users",
			},
		},
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceRole.String(),
		DisplayName: "Role",
		Description: "Role and permission management",
		Category:    "Administration",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View roles"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create new roles"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify roles"},
		},
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceAuditLog.String(),
		DisplayName:        "Audit Log",
		Description:        "System audit logs",
		Category:           "Administration",
		Operations:         readOnlyOps,
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceTableConfiguration.String(),
		DisplayName:        "Table Configuration",
		Description:        "User table preferences and configurations",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityPublic,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceSequenceConfig.String(),
		DisplayName:        "Sequence Configuration",
		Description:        "Sequence generator configuration",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceIntegration.String(),
		DisplayName:        "Integration",
		Description:        "External integration configuration and synchronization",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceAPIKey.String(),
		DisplayName:        "API Key",
		Description:        "API key creation, rotation, revocation, and permission management",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})
}

func (r *Registry) registerEquipmentResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceEquipmentType.String(),
		DisplayName:        "Equipment Type",
		Description:        "Equipment type definitions",
		Category:           "Equipment",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceEquipmentManufacturer.String(),
		DisplayName:        "Equipment Manufacturer",
		Description:        "Equipment manufacturer records",
		Category:           "Equipment",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceTrailer.String(),
		DisplayName: "Trailer",
		Description: "Trailer fleet management",
		Category:    "Equipment",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View trailers"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add new trailers"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify trailer information"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export trailer data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import trailer data"},
			{Operation: OpArchive, DisplayName: "Archive", Description: "Archive trailers"},
			{
				Operation:   OpRestore,
				DisplayName: "Restore",
				Description: "Restore archived trailers",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceTractor.String(),
		DisplayName: "Tractor",
		Description: "Tractor fleet management",
		Category:    "Equipment",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View tractors"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add new tractors"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify tractor information"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export tractor data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import tractor data"},
			{Operation: OpArchive, DisplayName: "Archive", Description: "Archive tractors"},
			{
				Operation:   OpRestore,
				DisplayName: "Restore",
				Description: "Restore archived tractors",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceFleetCode.String(),
		DisplayName:        "Fleet Code",
		Description:        "Fleet code definitions",
		Category:           "Equipment",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})
}

func (r *Registry) registerWorkerResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceWorker.String(),
		DisplayName: "Worker",
		Description: "Driver and worker management",
		Category:    "Workers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View workers"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add new workers"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify worker information"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export worker data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import worker data"},
			{Operation: OpArchive, DisplayName: "Archive", Description: "Archive workers"},
			{Operation: OpRestore, DisplayName: "Restore", Description: "Restore archived workers"},
		},
		DefaultSensitivity: SensitivityRestricted,
	})
}

func (r *Registry) registerOperationsResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceShipment.String(),
		DisplayName: "Shipment",
		Description: "Shipment management",
		Category:    "Operations",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View shipments"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create new shipments"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify shipments"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export shipment data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import shipments"},
			{
				Operation:   OpSubmit,
				DisplayName: "Submit",
				Description: "Submit shipments for dispatch",
			},
			{Operation: OpCancel, DisplayName: "Cancel", Description: "Cancel shipments"},
			{Operation: OpDuplicate, DisplayName: "Duplicate", Description: "Duplicate shipments"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceShipmentMove.String(),
		DisplayName:    "Shipment Move",
		Description:    "Shipment move management",
		Category:       "Operations",
		ParentResource: ResourceShipment.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View moves"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add moves"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify moves"},
			{Operation: OpAssign, DisplayName: "Assign", Description: "Assign resources to moves"},
			{
				Operation:   OpUnassign,
				DisplayName: "Unassign",
				Description: "Remove assignments from moves",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceShipmentStop.String(),
		DisplayName:    "Shipment Stop",
		Description:    "Shipment stop management",
		Category:       "Operations",
		ParentResource: ResourceShipment.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View stops"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add stops"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify stops"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceShipmentHold.String(),
		DisplayName:    "Shipment Hold",
		Description:    "Shipment hold management",
		Category:       "Operations",
		ParentResource: ResourceShipment.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View shipment holds"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create shipment holds"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify and release shipment holds"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDispatchControl.String(),
		DisplayName:        "Dispatch Control",
		Description:        "Dispatch settings and configuration",
		Category:           "Operations",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDataEntryControl.String(),
		DisplayName:        "Data Entry Control",
		Description:        "Data entry case formatting settings",
		Category:           "Administration",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceShipmentControl.String(),
		DisplayName:        "Shipment Control",
		Description:        "Shipment control settings and configuration",
		Category:           "Operations",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})
}

func (r *Registry) registerBillingResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceInvoice.String(),
		DisplayName: "Invoice",
		Description: "Invoice management",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View invoices"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create invoices"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify invoices"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export invoices"},
			{Operation: OpApprove, DisplayName: "Approve", Description: "Approve invoices"},
			{Operation: OpReject, DisplayName: "Reject", Description: "Reject invoices"},
			{
				Operation:   OpSubmit,
				DisplayName: "Submit",
				Description: "Submit invoices for approval",
			},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceBillingQueue.String(),
		DisplayName: "Billing Queue",
		Description: "Billing queue review and approval",
		Category:    "Billing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View billing queue items"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Transfer shipments to billing queue"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Update billing queue item status"},
			{Operation: OpAssign, DisplayName: "Assign", Description: "Assign billers to queue items"},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceAccessorialCharge.String(),
		DisplayName:        "Accessorial Charge",
		Description:        "Accessorial charge definitions",
		Category:           "Billing",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceChargeType.String(),
		DisplayName:        "Charge Type",
		Description:        "Charge type definitions",
		Category:           "Billing",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceRevenueCode.String(),
		DisplayName:        "Revenue Code",
		Description:        "Revenue code definitions",
		Category:           "Billing",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceFormulaTemplate.String(),
		DisplayName:        "Formula Template",
		Description:        "Rate formula templates",
		Category:           "Billing",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})
}

func (r *Registry) registerCustomerResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceCustomer.String(),
		DisplayName: "Customer",
		Description: "Customer management",
		Category:    "Customers",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View customers"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create customers"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify customers"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export customer data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import customers"},
			{Operation: OpArchive, DisplayName: "Archive", Description: "Archive customers"},
			{
				Operation:   OpRestore,
				DisplayName: "Restore",
				Description: "Restore archived customers",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:       ResourceCustomerContact.String(),
		DisplayName:    "Customer Contact",
		Description:    "Customer contact management",
		Category:       "Customers",
		ParentResource: ResourceCustomer.String(),
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View contacts"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Add contacts"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify contacts"},
		},
		DefaultSensitivity: SensitivityInternal,
	})
}

func (r *Registry) registerLocationResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceLocation.String(),
		DisplayName: "Location",
		Description: "Location management",
		Category:    "Locations",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View locations"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create locations"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify locations"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export location data"},
			{Operation: OpImport, DisplayName: "Import", Description: "Import locations"},
			{Operation: OpArchive, DisplayName: "Archive", Description: "Archive locations"},
			{
				Operation:   OpRestore,
				DisplayName: "Restore",
				Description: "Restore archived locations",
			},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceLocationCategory.String(),
		DisplayName:        "Location Category",
		Description:        "Location category definitions",
		Category:           "Locations",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})
}

func (r *Registry) registerCommodityResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceCommodity.String(),
		DisplayName:        "Commodity",
		Description:        "Commodity definitions",
		Category:           "Commodities",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceHazardousMaterial.String(),
		DisplayName:        "Hazardous Material",
		Description:        "Hazardous material definitions",
		Category:           "Commodities",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceHazmatSegregationRule.String(),
		DisplayName:        "Hazmat Segregation Rule",
		Description:        "Hazmat segregation rule definitions",
		Category:           "Commodities",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})
}

func (r *Registry) registerAccountingResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceGeneralLedgerAccount.String(),
		DisplayName:        "General Ledger Account",
		Description:        "GL account management",
		Category:           "Accounting",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDivisionCode.String(),
		DisplayName:        "Division Code",
		Description:        "Division code definitions",
		Category:           "Accounting",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceFiscalYear.String(),
		DisplayName: "Fiscal Year",
		Description: "Fiscal year management",
		Category:    "Accounting",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View fiscal years"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create fiscal years"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify fiscal years"},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Delete fiscal years"},
			{Operation: OpClose, DisplayName: "Close", Description: "Close fiscal years"},
			{Operation: OpLock, DisplayName: "Lock", Description: "Lock fiscal years"},
			{Operation: OpUnlock, DisplayName: "Unlock", Description: "Unlock fiscal years"},
			{Operation: OpActivate, DisplayName: "Activate", Description: "Activate fiscal years"},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceFiscalPeriod.String(),
		DisplayName: "Fiscal Period",
		Description: "Fiscal period management",
		Category:    "Accounting",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View fiscal periods"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create fiscal periods"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify fiscal periods"},
			{Operation: OpDelete, DisplayName: "Delete", Description: "Delete fiscal periods"},
			{Operation: OpClose, DisplayName: "Close", Description: "Close fiscal periods"},
			{Operation: OpReopen, DisplayName: "Reopen", Description: "Reopen fiscal periods"},
			{Operation: OpLock, DisplayName: "Lock", Description: "Lock fiscal periods"},
			{Operation: OpUnlock, DisplayName: "Unlock", Description: "Unlock fiscal periods"},
		},
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceManualJournal.String(),
		DisplayName: "Manual Journal",
		Description: "Manual journal request workflow and approvals",
		Category:    "Accounting",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View manual journals"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create manual journal drafts"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Edit manual journal drafts"},
			{Operation: OpApprove, DisplayName: "Approve", Description: "Approve manual journals"},
			{Operation: OpReject, DisplayName: "Reject", Description: "Reject manual journals"},
			{Operation: OpSubmit, DisplayName: "Submit", Description: "Submit manual journals for approval"},
			{Operation: OpCancel, DisplayName: "Cancel", Description: "Cancel manual journals"},
		},
		DefaultSensitivity: SensitivityRestricted,
	})
}

func (r *Registry) registerComplianceResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceQualification.String(),
		DisplayName:        "Qualification",
		Description:        "Driver qualification management",
		Category:           "Compliance",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityRestricted,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDocumentClassification.String(),
		DisplayName:        "Document Classification",
		Description:        "Document classification definitions",
		Category:           "Compliance",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDocument.String(),
		DisplayName:        "Document",
		Description:        "Uploaded document records and intelligence output",
		Category:           "Compliance",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDocumentType.String(),
		DisplayName:        "Document Type",
		Description:        "Document type configuration and classification targets",
		Category:           "Compliance",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDocumentControl.String(),
		DisplayName:        "Document Control",
		Description:        "Document intelligence and OCR settings",
		Category:           "Compliance",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDocumentParsingRule.String(),
		DisplayName:        "Document Parsing Rule",
		Description:        "Tenant-managed parsing rules, versions, fixtures, and simulations",
		Category:           "Compliance",
		Operations:         GetAllOperations(),
		DefaultSensitivity: SensitivityInternal,
	})

}

func (r *Registry) registerReferenceDataResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceServiceType.String(),
		DisplayName:        "Service Type",
		Description:        "Service type definitions",
		Category:           "Reference Data",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDelayCode.String(),
		DisplayName:        "Delay Code",
		Description:        "Delay code definitions",
		Category:           "Reference Data",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceReasonCode.String(),
		DisplayName:        "Reason Code",
		Description:        "Reason code definitions",
		Category:           "Reference Data",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceCommentType.String(),
		DisplayName:        "Comment Type",
		Description:        "Comment type definitions",
		Category:           "Reference Data",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceTag.String(),
		DisplayName:        "Tag",
		Description:        "Tag management",
		Category:           "Reference Data",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityPublic,
	})
}

func (r *Registry) registerReportingResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceReport.String(),
		DisplayName: "Report",
		Description: "Report generation and management",
		Category:    "Reporting",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View reports"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create reports"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify reports"},
			{Operation: OpExport, DisplayName: "Export", Description: "Export reports"},
		},
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceDashboard.String(),
		DisplayName: "Dashboard",
		Description: "Dashboard configuration",
		Category:    "Reporting",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "View dashboards"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create dashboards"},
			{Operation: OpUpdate, DisplayName: "Update", Description: "Modify dashboards"},
		},
		DefaultSensitivity: SensitivityInternal,
	})
}

func GetAllOperations() []OperationDefinition {
	return []OperationDefinition{
		{Operation: OpRead, DisplayName: "Read", Description: "View records"},
		{Operation: OpCreate, DisplayName: "Create", Description: "Create new records"},
		{Operation: OpUpdate, DisplayName: "Update", Description: "Modify existing records"},
		{Operation: OpExport, DisplayName: "Export", Description: "Export records to file"},
		{Operation: OpImport, DisplayName: "Import", Description: "Import records from file"},
		{Operation: OpApprove, DisplayName: "Approve", Description: "Approve records"},
		{Operation: OpReject, DisplayName: "Reject", Description: "Reject records"},
		{Operation: OpAssign, DisplayName: "Assign", Description: "Assign to users"},
		{Operation: OpUnassign, DisplayName: "Unassign", Description: "Remove assignments"},
		{Operation: OpArchive, DisplayName: "Archive", Description: "Archive records"},
		{Operation: OpRestore, DisplayName: "Restore", Description: "Restore archived records"},
		{Operation: OpSubmit, DisplayName: "Submit", Description: "Submit for processing"},
		{Operation: OpCancel, DisplayName: "Cancel", Description: "Cancel records"},
		{Operation: OpDuplicate, DisplayName: "Duplicate", Description: "Create copies"},
		{Operation: OpClose, DisplayName: "Close", Description: "Close records"},
		{Operation: OpLock, DisplayName: "Lock", Description: "Lock records"},
		{Operation: OpUnlock, DisplayName: "Unlock", Description: "Unlock records"},
		{Operation: OpActivate, DisplayName: "Activate", Description: "Activate records"},
		{Operation: OpReopen, DisplayName: "Reopen", Description: "Reopen records"},
	}
}
