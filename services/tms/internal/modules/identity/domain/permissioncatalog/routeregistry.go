package permissioncatalog

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

type RouteMatchType int

const (
	RouteMatchExact RouteMatchType = iota
	RouteMatchPrefix
	RouteMatchPattern
)

type RouteRequirement struct {
	Resource   permission.Resource
	Operation  permission.Operation
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
	resource permission.Resource,
	op permission.Operation,
) bool {
	bitmask, ok := permissions[string(resource)]
	if !ok {
		return false
	}
	return (bitmask & permission.OperationToBit[op]) != 0
}

func (rr *RouteRegistry) registerAll() {
	rr.registerAdministrationRoutes()
	rr.registerEDIRoutes()
	rr.registerEquipmentRoutes()
	rr.registerWorkerRoutes()
	rr.registerOperationsRoutes()
	rr.registerBillingRoutes()
	rr.registerDetentionRoutes()
	rr.registerCustomerRoutes()
	rr.registerCarrierRoutes()
	rr.registerCarrierSettlementRoutes()
	rr.registerLocationRoutes()
	rr.registerCommodityRoutes()
	rr.registerHoldReasonRoutes()
	rr.registerAccountingRoutes()
	rr.registerReportingRoutes()
	rr.registerDeskRoutes()
}

func (rr *RouteRegistry) registerEDIRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/edi/partners",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceEDI, Operation: permission.OpRead},
		},
		DisplayName: "EDI Partners",
		Category:    "EDI",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/edi/transfers/inbound",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceEDI, Operation: permission.OpRead},
		},
		DisplayName: "EDI Inbound Transfers",
		Category:    "EDI",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/edi/transfers/outbound",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceEDI, Operation: permission.OpRead},
		},
		DisplayName: "EDI Outbound Transfers",
		Category:    "EDI",
	})
}

//nolint:funlen // this is for the purpose of registering the routes
func (rr *RouteRegistry) registerAdministrationRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/users",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceUser, Operation: permission.OpRead},
		},
		DisplayName: "Users List",
		Category:    "Administration",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/users/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceUser, Operation: permission.OpCreate},
		},
		DisplayName: "Create User",
		Category:    "Administration",
		ParentRoute: "/admin/users",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/users/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceUser, Operation: permission.OpRead},
		},
		DisplayName: "User Details",
		Category:    "Administration",
		ParentRoute: "/admin/users",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/users/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceUser, Operation: permission.OpUpdate},
		},
		DisplayName: "Edit User",
		Category:    "Administration",
		ParentRoute: "/admin/users",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/roles",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceRole, Operation: permission.OpRead},
		},
		DisplayName: "Roles List",
		Category:    "Administration",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/roles/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceRole, Operation: permission.OpCreate},
		},
		DisplayName: "Create Role",
		Category:    "Administration",
		ParentRoute: "/admin/roles",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/roles/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceRole, Operation: permission.OpRead},
		},
		DisplayName: "Role Details",
		Category:    "Administration",
		ParentRoute: "/admin/roles",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/roles/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceRole, Operation: permission.OpUpdate},
		},
		DisplayName: "Edit Role",
		Category:    "Administration",
		ParentRoute: "/admin/roles",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/organizations",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceOrganization, Operation: permission.OpRead},
		},
		DisplayName: "Organizations",
		Category:    "Administration",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/audit-logs",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceAuditLog, Operation: permission.OpRead},
		},
		DisplayName: "Audit Logs",
		Category:    "Administration",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/sequence-configs",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceSequenceConfig, Operation: permission.OpRead},
		},
		DisplayName: "Sequence Configuration",
		Category:    "Administration",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/platform-catalog",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourcePlatformCatalog, Operation: permission.OpRead},
		},
		DisplayName: "Platform Catalog",
		Category:    "Administration",
	})
}

func (rr *RouteRegistry) registerEquipmentRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/equipment/tractors",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceTractor, Operation: permission.OpRead},
		},
		DisplayName: "Tractors List",
		Category:    "Equipment",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/equipment/tractors/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceTractor, Operation: permission.OpRead},
		},
		DisplayName: "Tractor Details",
		Category:    "Equipment",
		ParentRoute: "/equipment/tractors",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/equipment/trailers",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceTrailer, Operation: permission.OpRead},
		},
		DisplayName: "Trailers List",
		Category:    "Equipment",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/equipment/configuration-files/equipment-types",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceEquipmentType, Operation: permission.OpRead},
		},
		DisplayName: "Equipment Types",
		Category:    "Equipment",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/equipment/configuration-files/equipment-manufacturers",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceEquipmentManufacturer, Operation: permission.OpRead},
		},
		DisplayName: "Equipment Manufacturers",
		Category:    "Equipment",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/equipment/configuration-files/fleet-codes",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceFleetCode, Operation: permission.OpRead},
		},
		DisplayName: "Fleet Codes",
		Category:    "Equipment",
	})
}

func (rr *RouteRegistry) registerWorkerRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/checklist-templates",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceWorkerChecklistTemplate, Operation: permission.OpRead},
		},
		DisplayName: "Checklist Templates",
		Category:    "Workers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/holidays",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceOrgHoliday, Operation: permission.OpRead},
		},
		DisplayName: "Holiday Calendar",
		Category:    "Workers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/review-templates",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourcePerformanceReviewTemplate, Operation: permission.OpRead},
		},
		DisplayName: "Review Templates",
		Category:    "Workers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/training-courses",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceTrainingCourse, Operation: permission.OpRead},
		},
		DisplayName: "Training Courses",
		Category:    "Workers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/credential-types",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceWorkerCredentialType, Operation: permission.OpRead},
		},
		DisplayName: "Credential Types",
		Category:    "Workers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/pto-policies",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourcePTOPolicy, Operation: permission.OpRead},
		},
		DisplayName: "PTO Policies",
		Category:    "Workers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/workers",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceWorker, Operation: permission.OpRead},
		},
		DisplayName: "Workers List",
		Category:    "Workers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/workers/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceWorker, Operation: permission.OpCreate},
		},
		DisplayName: "Add Worker",
		Category:    "Workers",
		ParentRoute: "/workers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/workers/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceWorker, Operation: permission.OpRead},
		},
		DisplayName: "Worker Details",
		Category:    "Workers",
		ParentRoute: "/workers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/workers/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceWorker, Operation: permission.OpUpdate},
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
			{Resource: permission.ResourceShipment, Operation: permission.OpRead},
		},
		DisplayName: "Shipments",
		Category:    "Operations",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/shipments/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceShipment, Operation: permission.OpCreate},
		},
		DisplayName: "Create Shipment",
		Category:    "Operations",
		ParentRoute: "/shipments",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/shipments/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceShipment, Operation: permission.OpRead},
		},
		DisplayName: "Shipment Details",
		Category:    "Operations",
		ParentRoute: "/shipments",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/shipments/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceShipment, Operation: permission.OpUpdate},
		},
		DisplayName: "Edit Shipment",
		Category:    "Operations",
		ParentRoute: "/shipments",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/shipment-management/recurring-shipments",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceRecurringShipment, Operation: permission.OpRead},
		},
		DisplayName: "Recurring Shipments",
		Category:    "Operations",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/shipment-management/service-failures",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceServiceFailure, Operation: permission.OpRead},
		},
		DisplayName: "Service Failures",
		Category:    "Operations",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/dispatch/console",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceShipmentMove, Operation: permission.OpRead},
		},
		DisplayName: "Dispatch Console",
		Category:    "Operations",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/dispatch/control",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceDispatchControl, Operation: permission.OpRead},
		},
		DisplayName: "Dispatch Control",
		Category:    "Operations",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/data-entry-controls",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceDataEntryControl, Operation: permission.OpRead},
		},
		DisplayName: "Data Entry Controls",
		Category:    "Administration",
	})
}

func (rr *RouteRegistry) registerBillingRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/invoices",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceInvoice, Operation: permission.OpRead},
		},
		DisplayName: "Invoices",
		Category:    "Billing",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/invoice-runs",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceInvoiceRun, Operation: permission.OpRead},
		},
		DisplayName: "Invoice Runs",
		Category:    "Billing",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/invoice-runs/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceInvoiceRun, Operation: permission.OpCreate},
		},
		DisplayName: "New Invoice Run",
		Category:    "Billing",
		ParentRoute: "/billing/invoice-runs",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/invoices/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceInvoice, Operation: permission.OpCreate},
		},
		DisplayName: "Create Invoice",
		Category:    "Billing",
		ParentRoute: "/billing/invoices",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/invoices/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceInvoice, Operation: permission.OpRead},
		},
		DisplayName: "Invoice Details",
		Category:    "Billing",
		ParentRoute: "/billing/invoices",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/charge-types",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceChargeType, Operation: permission.OpRead},
		},
		DisplayName: "Charge Types",
		Category:    "Billing",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/configuration-files/accessorial-charges",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceAccessorialCharge, Operation: permission.OpRead},
		},
		DisplayName: "Accessorial Charges",
		Category:    "Billing",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/revenue-codes",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceRevenueCode, Operation: permission.OpRead},
		},
		DisplayName: "Revenue Codes",
		Category:    "Billing",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/configuration-files/formula-libraries",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceFormulaLibrary, Operation: permission.OpRead},
		},
		DisplayName: "Formula Libraries",
		Category:    "Billing",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/configuration-files/formula-templates",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceFormulaTemplate, Operation: permission.OpRead},
		},
		DisplayName: "Formula Templates",
		Category:    "Billing",
	})

	rr.registerRateRoutes()
}

func (rr *RouteRegistry) registerRateRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/rate-agreements",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceRateAgreement, Operation: permission.OpRead},
		},
		DisplayName: "Rate Agreements",
		Category:    "Billing",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/rate-agreements/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceRateAgreement, Operation: permission.OpRead},
		},
		DisplayName: "Rate Agreement Details",
		Category:    "Billing",
		ParentRoute: "/billing/rate-agreements",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/configuration-files/rate-zones",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceRateZone, Operation: permission.OpRead},
		},
		DisplayName: "Rate Zones",
		Category:    "Billing",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/configuration-files/rate-matrices",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceRateMatrix, Operation: permission.OpRead},
		},
		DisplayName: "Rate Matrices",
		Category:    "Billing",
	})
}

func (rr *RouteRegistry) registerDetentionRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/detention/desk",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceDetention, Operation: permission.OpRead},
		},
		DisplayName: "Detention Desk",
		Category:    "Detention",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/detention/intelligence",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceDetention, Operation: permission.OpRead},
		},
		DisplayName: "Detention Intelligence",
		Category:    "Detention",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/detention/configuration-files/detention-policies",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceDetentionPolicy, Operation: permission.OpRead},
		},
		DisplayName: "Detention Policies",
		Category:    "Detention",
	})
}

func (rr *RouteRegistry) registerCarrierRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/dispatch/carriers",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceCarrier, Operation: permission.OpRead},
		},
		DisplayName: "Carriers",
		Category:    "Carriers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/dispatch/carriers/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceCarrier, Operation: permission.OpRead},
		},
		DisplayName: "Carrier Details",
		Category:    "Carriers",
		ParentRoute: "/dispatch/carriers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/dispatch/routing-guides",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceRoutingGuide, Operation: permission.OpRead},
		},
		DisplayName: "Routing Guides",
		Category:    "Carriers",
	})
}

func (rr *RouteRegistry) registerCarrierSettlementRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/carrier-settlements/workspace",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceCarrierSettlement, Operation: permission.OpRead},
		},
		DisplayName: "Settlement Workspace",
		Category:    "Carrier Settlements",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/carrier-settlements/settlements",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceCarrierSettlement, Operation: permission.OpRead},
		},
		DisplayName: "Carrier Settlements",
		Category:    "Carrier Settlements",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/carrier-settlements/batches",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceCarrierSettlement, Operation: permission.OpRead},
		},
		DisplayName: "Settlement Batches",
		Category:    "Carrier Settlements",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/carrier-settlements/cost-events",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceCarrierSettlement, Operation: permission.OpRead},
		},
		DisplayName: "Carrier Cost Events",
		Category:    "Carrier Settlements",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/carrier-settlements/invoice-matching",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceCarrierInvoiceMatch, Operation: permission.OpRead},
		},
		DisplayName: "Invoice Matching",
		Category:    "Carrier Settlements",
	})
}

func (rr *RouteRegistry) registerCustomerRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/customers",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceCustomer, Operation: permission.OpRead},
		},
		DisplayName: "Customers",
		Category:    "Customers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/customers/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceCustomer, Operation: permission.OpCreate},
		},
		DisplayName: "Add Customer",
		Category:    "Customers",
		ParentRoute: "/customers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/customers/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceCustomer, Operation: permission.OpRead},
		},
		DisplayName: "Customer Details",
		Category:    "Customers",
		ParentRoute: "/customers",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/customers/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceCustomer, Operation: permission.OpUpdate},
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
			{Resource: permission.ResourceLocation, Operation: permission.OpRead},
		},
		DisplayName: "Locations",
		Category:    "Locations",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/locations/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceLocation, Operation: permission.OpCreate},
		},
		DisplayName: "Add Location",
		Category:    "Locations",
		ParentRoute: "/locations",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/locations/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceLocation, Operation: permission.OpRead},
		},
		DisplayName: "Location Details",
		Category:    "Locations",
		ParentRoute: "/locations",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/locations/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceLocation, Operation: permission.OpUpdate},
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
			{Resource: permission.ResourceCommodity, Operation: permission.OpRead},
		},
		DisplayName: "Commodities",
		Category:    "Commodities",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/hazmat",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceHazardousMaterial, Operation: permission.OpRead},
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
			{Resource: permission.ResourceGeneralLedgerAccount, Operation: permission.OpRead},
		},
		DisplayName: "GL Accounts",
		Category:    "Accounting",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/accounting/divisions",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceDivisionCode, Operation: permission.OpRead},
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
			{Resource: permission.ResourceHoldReason, Operation: permission.OpRead},
		},
		DisplayName: "Hold Reasons",
		Category:    "Organization",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/organization/hazmat-segregation-rules",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceHazmatSegregationRule, Operation: permission.OpRead},
		},
		DisplayName: "Hazmat Segregation Rules",
		Category:    "Organization",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/shipment-controls",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceShipmentControl, Operation: permission.OpRead},
		},
		DisplayName: "Shipment Controls",
		Category:    "Organization",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/service-failure-reason-codes",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceServiceFailureReasonCode, Operation: permission.OpRead},
		},
		DisplayName: "Service Failure Reason Codes",
		Category:    "Organization",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/document-controls",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceDocumentControl, Operation: permission.OpRead},
		},
		DisplayName: "Document Controls",
		Category:    "Organization",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/document-templates",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceDocumentTemplate, Operation: permission.OpRead},
		},
		DisplayName: "Document Templates",
		Category:    "Organization",
	})

	// The prefix covers the detail, version, and assignment surfaces, which are
	// reachable directly by link and must not be readable by someone who cannot
	// read the list they hang off.
	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/document-templates/",
		MatchType: RouteMatchPrefix,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceDocumentTemplate, Operation: permission.OpRead},
		},
		DisplayName: "Document Template Detail",
		Category:    "Organization",
		ParentRoute: "/admin/document-templates",
	})
}

// The Desk is where a person works with the assistant. Its conversations
// need the assistant; its decisions queue needs to read proposals, which
// is what AI Control asks for the same rows.
func (rr *RouteRegistry) registerDeskRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/desk",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceAssistant, Operation: permission.OpRead},
		},
		DisplayName: "Desk",
		Category:    "Desk",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/desk/t/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceAssistant, Operation: permission.OpRead},
		},
		DisplayName: "Desk conversation",
		Category:    "Desk",
		ParentRoute: "/desk",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/desk/watchtower",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceWatchtower, Operation: permission.OpRead},
		},
		DisplayName: "Watchtower",
		Category:    "Desk",
		ParentRoute: "/desk",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/desk/decisions",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceAgentProposal, Operation: permission.OpRead},
		},
		DisplayName: "Decisions",
		Category:    "Desk",
		ParentRoute: "/desk",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/desk/watchtower",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceWatchtower, Operation: permission.OpRead},
		},
		DisplayName: "Watchtower",
		Category:    "Desk",
		ParentRoute: "/desk",
	})
}

func (rr *RouteRegistry) registerReportingRoutes() {
	_ = rr.Register(&RouteDefinition{
		Path:      "/reports",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceReport, Operation: permission.OpRead},
		},
		DisplayName: "Reports",
		Category:    "Reporting",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/dashboard",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: permission.ResourceDashboard, Operation: permission.OpRead},
		},
		DisplayName: "Dashboard",
		Category:    "Reporting",
	})
}
