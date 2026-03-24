package permission

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	require.NotNil(t, reg)
	assert.True(t, reg.Count() > 0, "NewRegistry should have pre-registered resources")
}

func TestNewEmptyRegistry(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()
	require.NotNil(t, reg)
	assert.Equal(t, 0, reg.Count())
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &ResourceDefinition{
		Resource:    "test_resource",
		DisplayName: "Test Resource",
		Description: "A test resource",
		Category:    "Testing",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "Read", Description: "Read access"},
			{Operation: OpCreate, DisplayName: "Create", Description: "Create access"},
		},
	}

	err := reg.Register(def)
	require.NoError(t, err)
	assert.Equal(t, 1, reg.Count())
}

func TestRegistry_Register_DuplicateFails(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &ResourceDefinition{
		Resource:    "test_resource",
		DisplayName: "Test Resource",
	}

	err := reg.Register(def)
	require.NoError(t, err)

	err = reg.Register(def)
	require.Error(t, err)
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &ResourceDefinition{
		Resource:    "test_resource",
		DisplayName: "Test Resource",
	}
	err := reg.Register(def)
	require.NoError(t, err)

	result, ok := reg.Get("test_resource")
	assert.True(t, ok)
	assert.Equal(t, "test_resource", result.Resource)

	_, ok = reg.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_GetChildren(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	parent := &ResourceDefinition{
		Resource: "shipment",
	}
	child1 := &ResourceDefinition{
		Resource:       "shipment_move",
		ParentResource: "shipment",
	}
	child2 := &ResourceDefinition{
		Resource:       "shipment_stop",
		ParentResource: "shipment",
	}

	require.NoError(t, reg.Register(parent))
	require.NoError(t, reg.Register(child1))
	require.NoError(t, reg.Register(child2))

	children := reg.GetChildren("shipment")
	assert.Len(t, children, 2)
	assert.Contains(t, children, "shipment_move")
	assert.Contains(t, children, "shipment_stop")

	noChildren := reg.GetChildren("nonexistent")
	assert.Empty(t, noChildren)
}

func TestRegistry_GetEffectiveResource(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	parent := &ResourceDefinition{
		Resource: "shipment",
	}
	child := &ResourceDefinition{
		Resource:       "shipment_move",
		ParentResource: "shipment",
	}

	require.NoError(t, reg.Register(parent))
	require.NoError(t, reg.Register(child))

	assert.Equal(t, "shipment", reg.GetEffectiveResource("shipment"))
	assert.Equal(t, "shipment", reg.GetEffectiveResource("shipment_move"))
	assert.Equal(t, "unknown_resource", reg.GetEffectiveResource("unknown_resource"))
}

func TestRegistry_All(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def1 := &ResourceDefinition{Resource: "resource1"}
	def2 := &ResourceDefinition{Resource: "resource2"}

	require.NoError(t, reg.Register(def1))
	require.NoError(t, reg.Register(def2))

	all := reg.All()
	assert.Len(t, all, 2)
}

func TestRegistry_GetFieldSensitivity(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &ResourceDefinition{
		Resource:           "worker",
		DefaultSensitivity: SensitivityInternal,
		FieldSensitivities: map[string]FieldSensitivity{
			"ssn":      SensitivityConfidential,
			"pay_rate": SensitivityRestricted,
		},
	}
	require.NoError(t, reg.Register(def))

	assert.Equal(t, SensitivityConfidential, reg.GetFieldSensitivity("worker", "ssn"))
	assert.Equal(t, SensitivityRestricted, reg.GetFieldSensitivity("worker", "pay_rate"))
	assert.Equal(t, SensitivityInternal, reg.GetFieldSensitivity("worker", "name"))
	assert.Equal(t, SensitivityInternal, reg.GetFieldSensitivity("unknown", "field"))
}

func TestRegistry_GetOperationsForResource(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &ResourceDefinition{
		Resource: "shipment",
		Operations: []OperationDefinition{
			{Operation: OpRead},
			{Operation: OpCreate},
			{Operation: OpUpdate},
		},
	}
	require.NoError(t, reg.Register(def))

	ops := reg.GetOperationsForResource("shipment")
	assert.Len(t, ops, 3)
	assert.Contains(t, ops, OpRead)
	assert.Contains(t, ops, OpCreate)
	assert.Contains(t, ops, OpUpdate)

	noOps := reg.GetOperationsForResource("unknown")
	assert.Empty(t, noOps)
}

func TestRegistry_ExpandCompositeOperation(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &ResourceDefinition{
		Resource: "shipment",
		CompositeOps: map[string][]Operation{
			"manage":   {OpRead, OpCreate, OpUpdate, OpExport, OpImport},
			"dispatch": {OpRead, OpUpdate, OpAssign, OpSubmit},
		},
	}
	require.NoError(t, reg.Register(def))

	manageOps := reg.ExpandCompositeOperation("shipment", "manage")
	assert.Len(t, manageOps, 5)
	assert.Contains(t, manageOps, OpRead)
	assert.Contains(t, manageOps, OpCreate)
	assert.Contains(t, manageOps, OpUpdate)

	dispatchOps := reg.ExpandCompositeOperation("shipment", "dispatch")
	assert.Len(t, dispatchOps, 4)

	noOps := reg.ExpandCompositeOperation("shipment", "unknown")
	assert.Empty(t, noOps)

	noResource := reg.ExpandCompositeOperation("unknown", "manage")
	assert.Empty(t, noResource)
}

func TestRegistry_HasResource(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &ResourceDefinition{Resource: "shipment"}
	require.NoError(t, reg.Register(def))

	assert.True(t, reg.HasResource("shipment"))
	assert.False(t, reg.HasResource("unknown"))
}

func TestRegistry_GetByCategory(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def1 := &ResourceDefinition{Resource: "shipment", Category: "Operations"}
	def2 := &ResourceDefinition{Resource: "invoice", Category: "Billing"}
	def3 := &ResourceDefinition{Resource: "customer", Category: "Operations"}

	require.NoError(t, reg.Register(def1))
	require.NoError(t, reg.Register(def2))
	require.NoError(t, reg.Register(def3))

	opsResources := reg.GetByCategory("Operations")
	assert.Len(t, opsResources, 2)

	billingResources := reg.GetByCategory("Billing")
	assert.Len(t, billingResources, 1)

	unknownCategory := reg.GetByCategory("Unknown")
	assert.Empty(t, unknownCategory)
}

func TestRegistry_GetCategories(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def1 := &ResourceDefinition{Resource: "shipment", Category: "Operations"}
	def2 := &ResourceDefinition{Resource: "invoice", Category: "Billing"}
	def3 := &ResourceDefinition{Resource: "customer", Category: "Operations"}

	require.NoError(t, reg.Register(def1))
	require.NoError(t, reg.Register(def2))
	require.NoError(t, reg.Register(def3))

	categories := reg.GetCategories()
	assert.Len(t, categories, 2)
	assert.Contains(t, categories, "Operations")
	assert.Contains(t, categories, "Billing")
}

func TestRegistry_HasResources(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	assert.True(t, reg.Count() > 0, "Registry should have resources registered")
	assert.True(t, reg.HasResource("shipment"), "Registry should have shipment resource")
}

func TestRegistry_Register_EmptyResourceName(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()
	err := reg.Register(&ResourceDefinition{
		Resource: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource name is required")
}

func TestRegistry_Register_DefaultSensitivity(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()
	def := &ResourceDefinition{
		Resource: "test_resource",
	}
	err := reg.Register(def)
	require.NoError(t, err)

	result, ok := reg.Get("test_resource")
	require.True(t, ok)
	assert.Equal(t, SensitivityInternal, result.DefaultSensitivity)
}

func TestRegistry_Count(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()
	assert.Equal(t, 0, reg.Count())

	_ = reg.Register(&ResourceDefinition{Resource: "r1"})
	assert.Equal(t, 1, reg.Count())

	_ = reg.Register(&ResourceDefinition{Resource: "r2"})
	assert.Equal(t, 2, reg.Count())
}

func TestGetAllOperations(t *testing.T) {
	t.Parallel()

	ops := GetAllOperations()
	assert.Len(t, ops, 19)

	opNames := make(map[Operation]bool)
	for _, op := range ops {
		opNames[op.Operation] = true
	}

	expectedOps := []Operation{
		OpRead, OpCreate, OpUpdate, OpExport, OpImport,
		OpApprove, OpReject, OpAssign, OpUnassign,
		OpArchive, OpRestore, OpSubmit, OpCancel, OpDuplicate,
		OpClose, OpLock, OpUnlock, OpActivate, OpReopen,
	}

	for _, expected := range expectedOps {
		assert.True(t, opNames[expected], "expected operation %s in GetAllOperations", expected)
	}
}

func TestGetAllOperations_HasDisplayNames(t *testing.T) {
	t.Parallel()

	ops := GetAllOperations()
	for _, op := range ops {
		assert.NotEmpty(t, op.DisplayName, "operation %s should have a display name", op.Operation)
		assert.NotEmpty(t, op.Description, "operation %s should have a description", op.Operation)
	}
}

func TestResource_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		resource Resource
		expected string
	}{
		{ResourceOrganization, "organization"},
		{ResourceUser, "user"},
		{ResourceShipment, "shipment"},
		{ResourceInvoice, "invoice"},
		{ResourceCustomer, "customer"},
		{ResourceWorker, "worker"},
		{ResourceTractor, "tractor"},
		{ResourceTrailer, "trailer"},
		{ResourceLocation, "location"},
		{ResourceIntegration, "integration"},
		{ResourceReport, "report"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.resource.String())
		})
	}
}

func TestRegistry_RegisterAll_CategoriesExist(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	expectedCategories := []string{
		"Administration", "Equipment", "Workers", "Operations",
		"Billing", "Customers", "Locations", "Commodities",
		"Accounting", "Compliance", "Reference Data", "Reporting",
	}

	categories := reg.GetCategories()
	for _, cat := range expectedCategories {
		assert.Contains(t, categories, cat, "registry should have category %s", cat)
	}
}

func TestRegistry_RegisterAll_KnownResources(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	knownResources := []string{
		ResourceOrganization.String(),
		ResourceBusinessUnit.String(),
		ResourceUser.String(),
		ResourceRole.String(),
		ResourceIntegration.String(),
		ResourceShipment.String(),
		ResourceInvoice.String(),
		ResourceCustomer.String(),
		ResourceWorker.String(),
		ResourceTractor.String(),
		ResourceTrailer.String(),
		ResourceLocation.String(),
		ResourceCommodity.String(),
		ResourceShipmentControl.String(),
		ResourceHazmatSegregationRule.String(),
	}

	for _, res := range knownResources {
		assert.True(t, reg.HasResource(res), "registry should have resource %s", res)
	}
}

func TestRegistry_HasHazmatSegregationRuleResource(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	assert.True(t, reg.HasResource(ResourceHazmatSegregationRule.String()))
}

func TestRegistry_HasShipmentControlResource(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	assert.True(t, reg.HasResource(ResourceShipmentControl.String()))
}

func TestRegistry_ParentChildRelationships(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	children := reg.GetChildren(ResourceShipment.String())
	assert.Contains(t, children, ResourceShipmentMove.String())
	assert.Contains(t, children, ResourceShipmentStop.String())

	children = reg.GetChildren(ResourceCustomer.String())
	assert.Contains(t, children, ResourceCustomerContact.String())
}

func TestRegistry_Get_EmptyString(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	_, ok := reg.Get("")
	assert.False(t, ok)
}

func TestRegistry_HasResource_EmptyString(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	assert.False(t, reg.HasResource(""))
}

func TestRegistry_GetEffectiveResource_EmptyString(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	assert.Equal(t, "", reg.GetEffectiveResource(""))
}

func TestRegistry_GetFieldSensitivity_EmptyResource(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	assert.Equal(t, SensitivityInternal, reg.GetFieldSensitivity("", "field"))
}

func TestRegistry_GetFieldSensitivity_EmptyField(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &ResourceDefinition{
		Resource:           "test",
		DefaultSensitivity: SensitivityRestricted,
		FieldSensitivities: map[string]FieldSensitivity{
			"secret": SensitivityConfidential,
		},
	}
	require.NoError(t, reg.Register(def))

	assert.Equal(t, SensitivityRestricted, reg.GetFieldSensitivity("test", ""))
}

func TestRegistry_GetOperationsForResource_EmptyString(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ops := reg.GetOperationsForResource("")
	assert.Empty(t, ops)
}

func TestRegistry_ExpandCompositeOperation_EmptyStrings(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	ops := reg.ExpandCompositeOperation("", "")
	assert.Empty(t, ops)
}

func TestRegistry_GetChildren_EmptyString(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	children := reg.GetChildren("")
	assert.Empty(t, children)
}

func TestRegistry_GetByCategory_EmptyString(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	resources := reg.GetByCategory("")
	assert.Empty(t, resources)
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			_ = reg.All()
			_ = reg.Count()
			_, _ = reg.Get(ResourceShipment.String())
			_ = reg.HasResource(ResourceUser.String())
			_ = reg.GetCategories()
			_ = reg.GetByCategory("Operations")
			_ = reg.GetChildren(ResourceShipment.String())
			_ = reg.GetOperationsForResource(ResourceShipment.String())
			_ = reg.GetFieldSensitivity(ResourceWorker.String(), "name")
			_ = reg.GetEffectiveResource(ResourceShipmentMove.String())
		})
	}
	wg.Wait()
}

func TestRegistry_RegisterAll_ResourcesHaveOperations(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	all := reg.All()
	for _, def := range all {
		assert.NotEmpty(t, def.Operations, "resource %s should have operations", def.Resource)
	}
}

func TestRegistry_RegisterAll_ResourcesHaveDisplayName(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	all := reg.All()
	for _, def := range all {
		assert.NotEmpty(t, def.DisplayName, "resource %s should have a display name", def.Resource)
		assert.NotEmpty(t, def.Description, "resource %s should have a description", def.Resource)
		assert.NotEmpty(t, def.Category, "resource %s should have a category", def.Resource)
	}
}

func TestRegistry_RegisterAll_AllResourcesHaveValidSensitivity(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	all := reg.All()
	for _, def := range all {
		assert.True(
			t,
			def.DefaultSensitivity.IsValid(),
			"resource %s has invalid default sensitivity: %s",
			def.Resource,
			def.DefaultSensitivity,
		)
	}
}

func TestRouteRegistry_Match_EmptyPath(t *testing.T) {
	t.Parallel()

	rr := NewRouteRegistry()
	_, ok := rr.Match("")
	assert.False(t, ok)
}

func TestRouteRegistry_Get_EmptyPath(t *testing.T) {
	t.Parallel()

	rr := NewRouteRegistry()
	_, ok := rr.Get("")
	assert.False(t, ok)
}

func TestRouteRegistry_GetByCategory_EmptyCategory(t *testing.T) {
	t.Parallel()

	rr := NewRouteRegistry()
	routes := rr.GetByCategory("")
	assert.Empty(t, routes)
}

func TestRouteRegistry_ComputeAccess_NilPermissions(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()
	_ = rr.Register(&RouteDefinition{
		Path:      "/test",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
	})

	access := rr.ComputeAccess(nil)
	assert.False(t, access["/test"])
}

func TestRouteRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	rr := NewRouteRegistry()

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			_ = rr.All()
			_ = rr.Count()
			_, _ = rr.Match("/admin/users")
			_, _ = rr.Get("/admin/users")
			_ = rr.GetByCategory("Administration")
			_ = rr.ComputeAccess(map[string]uint32{
				string(ResourceUser): ClientOpRead,
			})
		})
	}
	wg.Wait()
}

func TestRouteRegistry_RegisterAll_AllRoutesHaveRequirements(t *testing.T) {
	t.Parallel()

	rr := NewRouteRegistry()

	all := rr.All()
	for _, def := range all {
		assert.NotEmpty(t, def.Requirements, "route %s should have requirements", def.Path)
		assert.NotEmpty(t, def.DisplayName, "route %s should have a display name", def.Path)
		assert.NotEmpty(t, def.Category, "route %s should have a category", def.Path)
	}
}

func TestRouteRegistry_RegisterAll_CategoriesExist(t *testing.T) {
	t.Parallel()

	rr := NewRouteRegistry()

	expectedCategories := []string{
		"Administration", "Equipment", "Workers", "Operations",
		"Billing", "Customers", "Locations", "Commodities",
		"Accounting", "Reporting",
	}

	allRoutes := rr.All()
	categorySet := make(map[string]bool)
	for _, def := range allRoutes {
		categorySet[def.Category] = true
	}

	for _, cat := range expectedCategories {
		assert.True(t, categorySet[cat], "route registry should have category %s", cat)
	}
}
