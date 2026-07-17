package sidebarpreference

import (
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/permission"
)

const (
	DocumentSchemaVersion   = 1
	MaxQuickActions         = 6
	DefaultActivityPageSize = 5
)

const (
	SectionAttention    = "attention"
	SectionQuickActions = "quickActions"
	SectionFavorites    = "favorites"
	SectionActivity     = "activity"
	SectionBrowse       = "browse"
)

type SectionDefinition struct {
	Key      string
	Label    string
	Hideable bool
}

type AttentionMetricDefinition struct {
	Key      string
	Label    string
	Resource permission.Resource
}

type QuickActionDefinition struct {
	ID        string
	Label     string
	Resource  permission.Resource
	Operation permission.Operation
}

func SectionCatalog() []SectionDefinition {
	return []SectionDefinition{
		{Key: SectionAttention, Label: "Needs Attention", Hideable: true},
		{Key: SectionQuickActions, Label: "Quick Actions", Hideable: true},
		{Key: SectionFavorites, Label: "Favorites", Hideable: true},
		{Key: SectionActivity, Label: "Recent Activity", Hideable: true},
		{Key: SectionBrowse, Label: "Browse", Hideable: false},
	}
}

func AttentionMetricCatalog() []AttentionMetricDefinition {
	return []AttentionMetricDefinition{
		{Key: "billingQueue", Label: "Billing Queue", Resource: permission.ResourceBillingQueue},
		{Key: "pendingApprovals", Label: "Pending Approvals", Resource: permission.ResourceInvoice},
		{
			Key:      "reconciliationExceptions",
			Label:    "Reconciliation Exceptions",
			Resource: permission.ResourceInvoice,
		},
		{
			Key:      "serviceFailures",
			Label:    "Service Failures",
			Resource: permission.ResourceServiceFailure,
		},
		{Key: "ediAttention", Label: "EDI Needs Attention", Resource: permission.ResourceEDI},
	}
}

func createQuickAction(id, label string, resource permission.Resource) QuickActionDefinition {
	return QuickActionDefinition{
		ID:        id,
		Label:     label,
		Resource:  resource,
		Operation: permission.OpCreate,
	}
}

func QuickActionCatalog() []QuickActionDefinition {
	return slices.Concat(
		shipmentQuickActions(),
		billingQuickActions(),
		equipmentQuickActions(),
		dispatchQuickActions(),
		adminQuickActions(),
	)
}

func shipmentQuickActions() []QuickActionDefinition {
	return []QuickActionDefinition{
		createQuickAction("create-shipment", "Create Shipment", permission.ResourceShipment),
		createQuickAction(
			"create-shipment-type",
			"Create Shipment Type",
			permission.ResourceShipmentType,
		),
		createQuickAction(
			"create-service-type",
			"Create Service Type",
			permission.ResourceServiceType,
		),
		createQuickAction(
			"create-hazardous-material",
			"Create Hazardous Material",
			permission.ResourceHazardousMaterial,
		),
		createQuickAction("create-commodity", "Create Commodity", permission.ResourceCommodity),
	}
}

func billingQuickActions() []QuickActionDefinition {
	return []QuickActionDefinition{
		createQuickAction(
			"create-accessorial-charge",
			"Create Accessorial Charge",
			permission.ResourceAccessorialCharge,
		),
		createQuickAction("create-customer", "Create Customer", permission.ResourceCustomer),
		createQuickAction(
			"create-document-type",
			"Create Document Type",
			permission.ResourceDocumentType,
		),
		createQuickAction(
			"create-formula-template",
			"Create Formula Template",
			permission.ResourceFormulaTemplate,
		),
		createQuickAction("create-rate-table", "Create Rate Table", permission.ResourceRateTable),
		createQuickAction(
			"create-account-type",
			"Create Account Type",
			permission.ResourceAccountType,
		),
		createQuickAction(
			"create-fiscal-year",
			"Create Fiscal Year",
			permission.ResourceFiscalYear,
		),
	}
}

func equipmentQuickActions() []QuickActionDefinition {
	return []QuickActionDefinition{
		createQuickAction("create-tractor", "Create Tractor", permission.ResourceTractor),
		createQuickAction("create-trailer", "Create Trailer", permission.ResourceTrailer),
		createQuickAction(
			"create-equipment-type",
			"Create Equipment Type",
			permission.ResourceEquipmentType,
		),
		createQuickAction(
			"create-equipment-manufacturer",
			"Create Equipment Manufacturer",
			permission.ResourceEquipmentManufacturer,
		),
	}
}

func dispatchQuickActions() []QuickActionDefinition {
	return []QuickActionDefinition{
		createQuickAction("create-location", "Create Location", permission.ResourceLocation),
		createQuickAction("create-worker", "Create Worker", permission.ResourceWorker),
		createQuickAction(
			"create-location-category",
			"Create Location Category",
			permission.ResourceLocationCategory,
		),
		createQuickAction("create-fleet-code", "Create Fleet Code", permission.ResourceFleetCode),
	}
}

func adminQuickActions() []QuickActionDefinition {
	return []QuickActionDefinition{
		createQuickAction(
			"create-hold-reason",
			"Create Hold Reason",
			permission.ResourceHoldReason,
		),
		createQuickAction(
			"create-service-failure-reason-code",
			"Create Service Failure Reason",
			permission.ResourceServiceFailureReasonCode,
		),
		createQuickAction(
			"create-hazmat-segregation-rule",
			"Create Hazmat Segregation Rule",
			permission.ResourceHazmatSegregationRule,
		),
		createQuickAction(
			"create-distance-override",
			"Create Distance Override",
			permission.ResourceDistanceOverride,
		),
		createQuickAction(
			"create-distance-profile",
			"Create Distance Profile",
			permission.ResourceDistanceProfile,
		),
		createQuickAction("create-user", "Create User", permission.ResourceUser),
		createQuickAction("create-role", "Create Role", permission.ResourceRole),
		createQuickAction(
			"create-custom-field",
			"Create Custom Field",
			permission.ResourceCustomFieldDefinition,
		),
	}
}

func ActivityPageSizes() []int {
	return []int{5, 10, 15}
}

func DefaultDocument() *Document {
	sections := SectionCatalog()
	sectionPrefs := make([]SectionPreference, 0, len(sections))
	for _, section := range sections {
		sectionPrefs = append(sectionPrefs, SectionPreference{Key: section.Key})
	}

	metrics := AttentionMetricCatalog()
	metricKeys := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		metricKeys = append(metricKeys, metric.Key)
	}

	return &Document{
		SchemaVersion:    DocumentSchemaVersion,
		Sections:         sectionPrefs,
		AttentionMetrics: metricKeys,
		QuickActionIDs: []string{
			"create-shipment",
			"create-worker",
			"create-location",
			"create-customer",
		},
		Activity: ActivityPreference{
			PageSize:    DefaultActivityPageSize,
			DefaultOpen: true,
		},
	}
}

func sectionDefinition(key string) (SectionDefinition, bool) {
	catalog := SectionCatalog()
	idx := slices.IndexFunc(catalog, func(section SectionDefinition) bool {
		return section.Key == key
	})
	if idx < 0 {
		return SectionDefinition{}, false
	}
	return catalog[idx], true
}

func attentionMetricExists(key string) bool {
	return slices.ContainsFunc(
		AttentionMetricCatalog(),
		func(metric AttentionMetricDefinition) bool { return metric.Key == key },
	)
}

func quickActionExists(id string) bool {
	return slices.ContainsFunc(
		QuickActionCatalog(),
		func(action QuickActionDefinition) bool { return action.ID == id },
	)
}

func activityPageSizeAllowed(size int) bool {
	return slices.Contains(ActivityPageSizes(), size)
}
