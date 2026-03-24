package permission

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

type RouteMatchType int

const (
	RouteMatchExact RouteMatchType = iota
	RouteMatchPrefix
	RouteMatchPattern
)

type RouteRequirement struct {
	Resource   Resource
	Operation  Operation
	Combinator string
}

type RouteDefinition struct {
	Path         string
	MatchType    RouteMatchType
	Requirements []RouteRequirement
	DisplayName  string
	Category     string
	ParentRoute  string
}

type routeWithPattern struct {
	def     *RouteDefinition
	pattern *regexp.Regexp
}

type RouteRegistry struct {
	mu            sync.RWMutex
	routes        map[string]*RouteDefinition
	prefixRoutes  []*routeWithPattern
	patternRoutes []*routeWithPattern
}

func NewRouteRegistry() *RouteRegistry {
	rr := &RouteRegistry{
		routes:        make(map[string]*RouteDefinition),
		prefixRoutes:  make([]*routeWithPattern, 0),
		patternRoutes: make([]*routeWithPattern, 0),
	}
	rr.registerAll()
	return rr
}

func NewEmptyRouteRegistry() *RouteRegistry {
	return &RouteRegistry{
		routes:        make(map[string]*RouteDefinition),
		prefixRoutes:  make([]*routeWithPattern, 0),
		patternRoutes: make([]*routeWithPattern, 0),
	}
}

func (rr *RouteRegistry) Register(def *RouteDefinition) error {
	if def.Path == "" {
		return errors.New("route path is required")
	}

	if len(def.Requirements) == 0 {
		return errors.New("at least one route requirement is required")
	}

	rr.mu.Lock()
	defer rr.mu.Unlock()

	switch def.MatchType {
	case RouteMatchExact:
		if _, exists := rr.routes[def.Path]; exists {
			return fmt.Errorf("route %s already registered", def.Path)
		}
		rr.routes[def.Path] = def

	case RouteMatchPrefix:
		pattern := rr.pathToRegex(def.Path)
		rr.prefixRoutes = append(rr.prefixRoutes, &routeWithPattern{
			def:     def,
			pattern: pattern,
		})

	case RouteMatchPattern:
		pattern := rr.pathToRegex(def.Path)
		rr.patternRoutes = append(rr.patternRoutes, &routeWithPattern{
			def:     def,
			pattern: pattern,
		})
	}

	return nil
}

func (rr *RouteRegistry) pathToRegex(path string) *regexp.Regexp {
	escaped := regexp.QuoteMeta(path)
	pattern := regexp.MustCompile(`:([a-zA-Z][a-zA-Z0-9]*)`).ReplaceAllString(escaped, `[^/]+`)
	pattern = strings.ReplaceAll(pattern, `\*`, `.*`)
	return regexp.MustCompile("^" + pattern + "$")
}

func (rr *RouteRegistry) Match(path string) (*RouteDefinition, bool) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	if def, ok := rr.routes[path]; ok {
		return def, true
	}

	for _, rp := range rr.patternRoutes {
		if rp.pattern.MatchString(path) {
			return rp.def, true
		}
	}

	for _, rp := range rr.prefixRoutes {
		if rp.pattern.MatchString(path) {
			return rp.def, true
		}
	}

	return nil, false
}

func (rr *RouteRegistry) Get(path string) (*RouteDefinition, bool) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	def, ok := rr.routes[path]
	return def, ok
}

func (rr *RouteRegistry) All() []*RouteDefinition {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	result := make([]*RouteDefinition, 0, len(rr.routes)+len(rr.patternRoutes)+len(rr.prefixRoutes))

	for _, def := range rr.routes {
		result = append(result, def)
	}

	for _, rp := range rr.patternRoutes {
		result = append(result, rp.def)
	}

	for _, rp := range rr.prefixRoutes {
		result = append(result, rp.def)
	}

	return result
}

func (rr *RouteRegistry) GetByCategory(category string) []*RouteDefinition {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	var result []*RouteDefinition

	for _, def := range rr.routes {
		if def.Category == category {
			result = append(result, def)
		}
	}

	for _, rp := range rr.patternRoutes {
		if rp.def.Category == category {
			result = append(result, rp.def)
		}
	}

	for _, rp := range rr.prefixRoutes {
		if rp.def.Category == category {
			result = append(result, rp.def)
		}
	}

	return result
}

func (rr *RouteRegistry) Count() int {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	return len(rr.routes) + len(rr.patternRoutes) + len(rr.prefixRoutes)
}

func (rr *RouteRegistry) ComputeAccess(permissions map[string]uint32) map[string]bool {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	routeAccess := make(map[string]bool)

	for path, def := range rr.routes {
		routeAccess[path] = rr.checkRequirements(def.Requirements, permissions)
	}

	for _, rp := range rr.patternRoutes {
		routeAccess[rp.def.Path] = rr.checkRequirements(rp.def.Requirements, permissions)
	}

	for _, rp := range rr.prefixRoutes {
		routeAccess[rp.def.Path] = rr.checkRequirements(rp.def.Requirements, permissions)
	}

	return routeAccess
}

func (rr *RouteRegistry) checkRequirements(
	reqs []RouteRequirement,
	permissions map[string]uint32,
) bool {
	if len(reqs) == 0 {
		return true
	}

	combinator := "and"
	if len(reqs) > 0 && reqs[0].Combinator != "" {
		combinator = reqs[0].Combinator
	}

	if combinator == "or" {
		for _, req := range reqs {
			if rr.hasPermission(permissions, req.Resource, req.Operation) {
				return true
			}
		}
		return false
	}

	for _, req := range reqs {
		if !rr.hasPermission(permissions, req.Resource, req.Operation) {
			return false
		}
	}
	return true
}

func (rr *RouteRegistry) hasPermission(
	permissions map[string]uint32,
	resource Resource,
	op Operation,
) bool {
	bitmask, ok := permissions[string(resource)]
	if !ok {
		return false
	}
	return (bitmask & OperationToBit[op]) != 0
}

func (rr *RouteRegistry) registerAll() {
	rr.registerAdministrationRoutes()
	rr.registerEquipmentRoutes()
	rr.registerWorkerRoutes()
	rr.registerOperationsRoutes()
	rr.registerBillingRoutes()
	rr.registerCustomerRoutes()
	rr.registerLocationRoutes()
	rr.registerCommodityRoutes()
	rr.registerHoldReasonRoutes()
	rr.registerAccountingRoutes()
	rr.registerReportingRoutes()
}

//nolint:funlen // this is for the purpose of registering the routes
func (rr *RouteRegistry) registerAdministrationRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/users",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
		DisplayName: "Users List",
		Category:    "Administration",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/users/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpCreate},
		},
		DisplayName: "Create User",
		Category:    "Administration",
		ParentRoute: "/admin/users",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/users/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
		DisplayName: "User Details",
		Category:    "Administration",
		ParentRoute: "/admin/users",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/users/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpUpdate},
		},
		DisplayName: "Edit User",
		Category:    "Administration",
		ParentRoute: "/admin/users",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/roles",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceRole, Operation: OpRead},
		},
		DisplayName: "Roles List",
		Category:    "Administration",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/roles/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceRole, Operation: OpCreate},
		},
		DisplayName: "Create Role",
		Category:    "Administration",
		ParentRoute: "/admin/roles",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/roles/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceRole, Operation: OpRead},
		},
		DisplayName: "Role Details",
		Category:    "Administration",
		ParentRoute: "/admin/roles",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/roles/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceRole, Operation: OpUpdate},
		},
		DisplayName: "Edit Role",
		Category:    "Administration",
		ParentRoute: "/admin/roles",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/organizations",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceOrganization, Operation: OpRead},
		},
		DisplayName: "Organizations",
		Category:    "Administration",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/audit-logs",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceAuditLog, Operation: OpRead},
		},
		DisplayName: "Audit Logs",
		Category:    "Administration",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/sequence-configs",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceSequenceConfig, Operation: OpRead},
		},
		DisplayName: "Sequence Configuration",
		Category:    "Administration",
	})
}

func (rr *RouteRegistry) registerEquipmentRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/equipment/tractors",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceTractor, Operation: OpRead},
		},
		DisplayName: "Tractors List",
		Category:    "Equipment",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/equipment/tractors/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceTractor, Operation: OpRead},
		},
		DisplayName: "Tractor Details",
		Category:    "Equipment",
		ParentRoute: "/equipment/tractors",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/equipment/trailers",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceTrailer, Operation: OpRead},
		},
		DisplayName: "Trailers List",
		Category:    "Equipment",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/equipment/configuration-files/equipment-types",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceEquipmentType, Operation: OpRead},
		},
		DisplayName: "Equipment Types",
		Category:    "Equipment",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/equipment/configuration-files/equipment-manufacturers",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceEquipmentManufacturer, Operation: OpRead},
		},
		DisplayName: "Equipment Manufacturers",
		Category:    "Equipment",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/equipment/configuration-files/fleet-codes",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceFleetCode, Operation: OpRead},
		},
		DisplayName: "Fleet Codes",
		Category:    "Equipment",
	})
}

func (rr *RouteRegistry) registerWorkerRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/workers",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceWorker, Operation: OpRead},
		},
		DisplayName: "Workers List",
		Category:    "Workers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/workers/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceWorker, Operation: OpCreate},
		},
		DisplayName: "Add Worker",
		Category:    "Workers",
		ParentRoute: "/workers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/workers/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceWorker, Operation: OpRead},
		},
		DisplayName: "Worker Details",
		Category:    "Workers",
		ParentRoute: "/workers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/workers/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceWorker, Operation: OpUpdate},
		},
		DisplayName: "Edit Worker",
		Category:    "Workers",
		ParentRoute: "/workers",
	})
}

func (rr *RouteRegistry) registerOperationsRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/shipments",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceShipment, Operation: OpRead},
		},
		DisplayName: "Shipments",
		Category:    "Operations",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/shipments/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceShipment, Operation: OpCreate},
		},
		DisplayName: "Create Shipment",
		Category:    "Operations",
		ParentRoute: "/shipments",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/shipments/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceShipment, Operation: OpRead},
		},
		DisplayName: "Shipment Details",
		Category:    "Operations",
		ParentRoute: "/shipments",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/shipments/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceShipment, Operation: OpUpdate},
		},
		DisplayName: "Edit Shipment",
		Category:    "Operations",
		ParentRoute: "/shipments",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/dispatch/control",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceDispatchControl, Operation: OpRead},
		},
		DisplayName: "Dispatch Control",
		Category:    "Operations",
	})
}

func (rr *RouteRegistry) registerBillingRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/invoices",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceInvoice, Operation: OpRead},
		},
		DisplayName: "Invoices",
		Category:    "Billing",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/invoices/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceInvoice, Operation: OpCreate},
		},
		DisplayName: "Create Invoice",
		Category:    "Billing",
		ParentRoute: "/billing/invoices",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/invoices/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceInvoice, Operation: OpRead},
		},
		DisplayName: "Invoice Details",
		Category:    "Billing",
		ParentRoute: "/billing/invoices",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/charge-types",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceChargeType, Operation: OpRead},
		},
		DisplayName: "Charge Types",
		Category:    "Billing",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/configuration-files/accessorial-charges",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceAccessorialCharge, Operation: OpRead},
		},
		DisplayName: "Accessorial Charges",
		Category:    "Billing",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/revenue-codes",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceRevenueCode, Operation: OpRead},
		},
		DisplayName: "Revenue Codes",
		Category:    "Billing",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/configuration-files/formula-templates",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceFormulaTemplate, Operation: OpRead},
		},
		DisplayName: "Formula Templates",
		Category:    "Billing",
	})
}

func (rr *RouteRegistry) registerCustomerRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/customers",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceCustomer, Operation: OpRead},
		},
		DisplayName: "Customers",
		Category:    "Customers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/customers/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceCustomer, Operation: OpCreate},
		},
		DisplayName: "Add Customer",
		Category:    "Customers",
		ParentRoute: "/customers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/customers/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceCustomer, Operation: OpRead},
		},
		DisplayName: "Customer Details",
		Category:    "Customers",
		ParentRoute: "/customers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/customers/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceCustomer, Operation: OpUpdate},
		},
		DisplayName: "Edit Customer",
		Category:    "Customers",
		ParentRoute: "/customers",
	})
}

func (rr *RouteRegistry) registerLocationRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/locations",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceLocation, Operation: OpRead},
		},
		DisplayName: "Locations",
		Category:    "Locations",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/locations/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceLocation, Operation: OpCreate},
		},
		DisplayName: "Add Location",
		Category:    "Locations",
		ParentRoute: "/locations",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/locations/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceLocation, Operation: OpRead},
		},
		DisplayName: "Location Details",
		Category:    "Locations",
		ParentRoute: "/locations",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/locations/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceLocation, Operation: OpUpdate},
		},
		DisplayName: "Edit Location",
		Category:    "Locations",
		ParentRoute: "/locations",
	})
}

func (rr *RouteRegistry) registerCommodityRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/commodities",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceCommodity, Operation: OpRead},
		},
		DisplayName: "Commodities",
		Category:    "Commodities",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/hazmat",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceHazardousMaterial, Operation: OpRead},
		},
		DisplayName: "Hazardous Materials",
		Category:    "Commodities",
	})
}

func (rr *RouteRegistry) registerAccountingRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/accounting/gl-accounts",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceGeneralLedgerAccount, Operation: OpRead},
		},
		DisplayName: "GL Accounts",
		Category:    "Accounting",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/accounting/divisions",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceDivisionCode, Operation: OpRead},
		},
		DisplayName: "Division Codes",
		Category:    "Accounting",
	})
}

func (rr *RouteRegistry) registerHoldReasonRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/organization/hold-reasons",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceHoldReason, Operation: OpRead},
		},
		DisplayName: "Hold Reasons",
		Category:    "Organization",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/organization/hazmat-segregation-rules",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceHazmatSegregationRule, Operation: OpRead},
		},
		DisplayName: "Hazmat Segregation Rules",
		Category:    "Organization",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/shipment-controls",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceShipmentControl, Operation: OpRead},
		},
		DisplayName: "Shipment Controls",
		Category:    "Organization",
	})
}

func (rr *RouteRegistry) registerReportingRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/reports",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceReport, Operation: OpRead},
		},
		DisplayName: "Reports",
		Category:    "Reporting",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/dashboard",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceDashboard, Operation: OpRead},
		},
		DisplayName: "Dashboard",
		Category:    "Reporting",
	})
}
