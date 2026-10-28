package permissioncatalog

import (
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
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

	def := &permissionspec.ResourceDefinition{
		Resource:    "test_resource",
		DisplayName: "Test Resource",
		Description: "A test resource",
		Category:    "Testing",
		Operations: []permissionspec.OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "Read", Description: "Read access"},
			{Operation: permission.OpCreate, DisplayName: "Create", Description: "Create access"},
		},
	}

	err := reg.Register(def)
	require.NoError(t, err)
	assert.Equal(t, 1, reg.Count())
}

func TestRegistry_Register_DuplicateFails(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &permissionspec.ResourceDefinition{
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

	def := &permissionspec.ResourceDefinition{
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

	parent := &permissionspec.ResourceDefinition{
		Resource: "shipment",
	}
	child1 := &permissionspec.ResourceDefinition{
		Resource:       "shipment_move",
		ParentResource: "shipment",
	}
	child2 := &permissionspec.ResourceDefinition{
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

	parent := &permissionspec.ResourceDefinition{
		Resource: "shipment",
	}
	child := &permissionspec.ResourceDefinition{
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

	def1 := &permissionspec.ResourceDefinition{Resource: "resource1"}
	def2 := &permissionspec.ResourceDefinition{Resource: "resource2"}

	require.NoError(t, reg.Register(def1))
	require.NoError(t, reg.Register(def2))

	all := reg.All()
	assert.Len(t, all, 2)
}

func TestRegistry_GetFieldSensitivity(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &permissionspec.ResourceDefinition{
		Resource:           "worker",
		DefaultSensitivity: permission.SensitivityInternal,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"ssn":      permission.SensitivityConfidential,
			"pay_rate": permission.SensitivityRestricted,
		},
	}
	require.NoError(t, reg.Register(def))

	assert.Equal(t, permission.SensitivityConfidential, reg.GetFieldSensitivity("worker", "ssn"))
	assert.Equal(t, permission.SensitivityRestricted, reg.GetFieldSensitivity("worker", "pay_rate"))
	assert.Equal(t, permission.SensitivityInternal, reg.GetFieldSensitivity("worker", "name"))
	assert.Equal(t, permission.SensitivityInternal, reg.GetFieldSensitivity("unknown", "field"))
}

func TestRegistry_GetOperationsForResource(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &permissionspec.ResourceDefinition{
		Resource: "shipment",
		Operations: []permissionspec.OperationDefinition{
			{Operation: permission.OpRead},
			{Operation: permission.OpCreate},
			{Operation: permission.OpUpdate},
		},
	}
	require.NoError(t, reg.Register(def))

	ops := reg.GetOperationsForResource("shipment")
	assert.Len(t, ops, 3)
	assert.Contains(t, ops, permission.OpRead)
	assert.Contains(t, ops, permission.OpCreate)
	assert.Contains(t, ops, permission.OpUpdate)

	noOps := reg.GetOperationsForResource("unknown")
	assert.Empty(t, noOps)
}

func TestRegistry_ExpandCompositeOperation(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &permissionspec.ResourceDefinition{
		Resource: "shipment",
		CompositeOps: map[string][]permission.Operation{
			"manage": {
				permission.OpRead,
				permission.OpCreate,
				permission.OpUpdate,
				permission.OpExport,
				permission.OpImport,
			},
			"dispatch": {
				permission.OpRead,
				permission.OpUpdate,
				permission.OpAssign,
				permission.OpSubmit,
			},
		},
	}
	require.NoError(t, reg.Register(def))

	manageOps := reg.ExpandCompositeOperation("shipment", "manage")
	assert.Len(t, manageOps, 5)
	assert.Contains(t, manageOps, permission.OpRead)
	assert.Contains(t, manageOps, permission.OpCreate)
	assert.Contains(t, manageOps, permission.OpUpdate)

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

	def := &permissionspec.ResourceDefinition{Resource: "shipment"}
	require.NoError(t, reg.Register(def))

	assert.True(t, reg.HasResource("shipment"))
	assert.False(t, reg.HasResource("unknown"))
}

func TestRegistry_GetByCategory(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def1 := &permissionspec.ResourceDefinition{Resource: "shipment", Category: "Operations"}
	def2 := &permissionspec.ResourceDefinition{Resource: "invoice", Category: "Billing"}
	def3 := &permissionspec.ResourceDefinition{Resource: "customer", Category: "Operations"}

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

	def1 := &permissionspec.ResourceDefinition{Resource: "shipment", Category: "Operations"}
	def2 := &permissionspec.ResourceDefinition{Resource: "invoice", Category: "Billing"}
	def3 := &permissionspec.ResourceDefinition{Resource: "customer", Category: "Operations"}

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
	err := reg.Register(&permissionspec.ResourceDefinition{
		Resource: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource name is required")
}

func TestRegistry_Register_DefaultSensitivity(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()
	def := &permissionspec.ResourceDefinition{
		Resource: "test_resource",
	}
	err := reg.Register(def)
	require.NoError(t, err)

	result, ok := reg.Get("test_resource")
	require.True(t, ok)
	assert.Equal(t, permission.SensitivityInternal, result.DefaultSensitivity)
}

func TestRegistry_Count(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()
	assert.Equal(t, 0, reg.Count())

	_ = reg.Register(&permissionspec.ResourceDefinition{Resource: "r1"})
	assert.Equal(t, 1, reg.Count())

	_ = reg.Register(&permissionspec.ResourceDefinition{Resource: "r2"})
	assert.Equal(t, 2, reg.Count())
}

func TestGetAllOperations(t *testing.T) {
	t.Parallel()

	ops := GetAllOperations()
	assert.Len(t, ops, 19)

	opNames := make(map[permission.Operation]bool)
	for _, op := range ops {
		opNames[op.Operation] = true
	}

	expectedOps := []permission.Operation{
		permission.OpRead, permission.OpCreate, permission.OpUpdate, permission.OpExport, permission.OpImport,
		permission.OpApprove, permission.OpReject, permission.OpAssign, permission.OpUnassign,
		permission.OpArchive, permission.OpRestore, permission.OpSubmit, permission.OpCancel, permission.OpDuplicate,
		permission.OpClose, permission.OpLock, permission.OpUnlock, permission.OpActivate, permission.OpReopen,
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
		resource permission.Resource
		expected string
	}{
		{permission.ResourceOrganization, "organization"},
		{permission.ResourceUser, "user"},
		{permission.ResourceShipment, "shipment"},
		{permission.ResourceInvoice, "invoice"},
		{permission.ResourceCustomer, "customer"},
		{permission.ResourceWorker, "worker"},
		{permission.ResourceWorkerPTO, "worker_pto"},
		{permission.ResourceWorkerCredential, "worker_credential"},
		{permission.ResourceWorkerCredentialType, "worker_credential_type"},
		{permission.ResourceWorkerEmploymentEvent, "worker_employment_event"},
		{permission.ResourceWorkerChecklist, "worker_checklist"},
		{permission.ResourceWorkerChecklistTemplate, "worker_checklist_template"},
		{permission.ResourceOrgHoliday, "org_holiday"},
		{permission.ResourceTrainingCourse, "training_course"},
		{permission.ResourceWorkerTraining, "worker_training"},
		{permission.ResourceWorkerSafetyEvent, "worker_safety_event"},
		{permission.ResourceWorkerDisciplinaryAction, "worker_disciplinary_action"},
		{permission.ResourceWorkerRecognition, "worker_recognition"},
		{permission.ResourcePerformanceReviewTemplate, "performance_review_template"},
		{permission.ResourcePerformanceReview, "performance_review"},
		{permission.ResourceTractor, "tractor"},
		{permission.ResourceTrailer, "trailer"},
		{permission.ResourceLocation, "location"},
		{permission.ResourceIntegration, "integration"},
		{permission.ResourceAgentExtension, "agent_extension"},
		{permission.ResourceWebResearch, "web_research"},
		{permission.ResourcePortalAutomation, "portal_automation"},
		{permission.ResourceCodeMode, "code_mode"},
		{permission.ResourceReport, "report"},
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
		"Desk", "Triage", "Intake", "Shipments", "Dispatch", "Carriers", "Equipment",
		"Fuel & tax", "EDI", "Human resources", "Payroll", "Billing", "Detention",
		"Carrier settlements", "Accounting", "Reports", "Documents", "AI & agents",
		"Organization settings", "Users & security", "Protected fields",
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
		permission.ResourceOrganization.String(),
		permission.ResourceBusinessUnit.String(),
		permission.ResourceUser.String(),
		permission.ResourceRole.String(),
		permission.ResourceIntegration.String(),
		permission.ResourceAgentExtension.String(),
		permission.ResourceWebResearch.String(),
		permission.ResourcePortalAutomation.String(),
		permission.ResourceCodeMode.String(),
		permission.ResourceShipment.String(),
		permission.ResourceServiceFailure.String(),
		permission.ResourceServiceFailureReasonCode.String(),
		permission.ResourceInvoice.String(),
		permission.ResourceCustomer.String(),
		permission.ResourceWorker.String(),
		permission.ResourceWorkerPTO.String(),
		permission.ResourceWorkerCredential.String(),
		permission.ResourceWorkerCredentialType.String(),
		permission.ResourceWorkerEmploymentEvent.String(),
		permission.ResourceWorkerChecklist.String(),
		permission.ResourceWorkerChecklistTemplate.String(),
		permission.ResourceOrgHoliday.String(),
		permission.ResourceTrainingCourse.String(),
		permission.ResourceWorkerTraining.String(),
		permission.ResourceWorkerSafetyEvent.String(),
		permission.ResourceWorkerDisciplinaryAction.String(),
		permission.ResourceWorkerRecognition.String(),
		permission.ResourcePerformanceReviewTemplate.String(),
		permission.ResourcePerformanceReview.String(),
		permission.ResourceDistanceOverride.String(),
		permission.ResourceTractor.String(),
		permission.ResourceTrailer.String(),
		permission.ResourceLocation.String(),
		permission.ResourceCommodity.String(),
		permission.ResourceShipmentControl.String(),
		permission.ResourceHazmatSegregationRule.String(),
	}

	for _, res := range knownResources {
		assert.True(t, reg.HasResource(res), "registry should have resource %s", res)
	}
}

func TestRegistry_ServiceFailureResourcesUseReadinessOperations(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	wantOps := []permission.Operation{
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpApprove,
		permission.OpArchive,
		permission.OpExport,
	}

	for _, resource := range []permission.Resource{
		permission.ResourceServiceFailure,
		permission.ResourceServiceFailureReasonCode,
	} {
		t.Run(resource.String(), func(t *testing.T) {
			t.Parallel()

			def, ok := reg.Get(resource.String())
			require.True(t, ok)
			assert.Equal(t, "Shipments", def.Category)
			assert.ElementsMatch(t, wantOps, reg.GetOperationsForResource(resource.String()))
		})
	}
}

func TestRegistry_RegisterWorkerPTOResource(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	def, ok := reg.Get(permission.ResourceWorkerPTO.String())
	require.True(t, ok)
	assert.Equal(t, "Worker PTO", def.DisplayName)
	assert.Equal(t, "Human resources", def.Category)
	assert.Empty(t, def.ParentResource)
	assert.Equal(t, permission.SensitivityRestricted, def.DefaultSensitivity)

	ops := reg.GetOperationsForResource(permission.ResourceWorkerPTO.String())
	assert.ElementsMatch(t, []permission.Operation{
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpExport,
		permission.OpApprove,
		permission.OpCancel,
		permission.OpManage,
		permission.OpReject,
	}, ops)
}

func TestRegistry_ShipmentMoveClassifiesTheExternalDriver(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	resource := permission.ResourceShipmentMove.String()

	def, ok := reg.Get(resource)
	require.True(t, ok)
	assert.Equal(t, permission.SensitivityInternal, def.DefaultSensitivity)
	assert.Equal(
		t,
		permission.SensitivityRestricted,
		reg.GetFieldSensitivity(resource, "externalDriverPhone"),
	)
	assert.Equal(
		t,
		permission.SensitivityInternal,
		reg.GetFieldSensitivity(resource, "externalDriverName"),
	)
	assert.Equal(t, permission.SensitivityInternal, reg.GetFieldSensitivity(resource, "carrierId"))
}

func TestRegistry_HasHazmatSegregationRuleResource(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	assert.True(t, reg.HasResource(permission.ResourceHazmatSegregationRule.String()))
}

func TestRegistry_HasShipmentControlResource(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	assert.True(t, reg.HasResource(permission.ResourceShipmentControl.String()))
}

func TestRegistry_HasDistanceOverrideResource(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	def, ok := reg.Get(permission.ResourceDistanceOverride.String())
	require.True(t, ok)
	assert.Equal(t, "Distance Override", def.DisplayName)
	assert.Equal(t, "Dispatch", def.Category)
	assert.Equal(t, permission.SensitivityInternal, def.DefaultSensitivity)
	assert.Contains(
		t,
		reg.GetOperationsForResource(permission.ResourceDistanceOverride.String()),
		permission.OpRead,
	)
}

func TestRegistry_ParentChildRelationships(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	children := reg.GetChildren(permission.ResourceShipment.String())
	assert.Contains(t, children, permission.ResourceShipmentMove.String())
	assert.Contains(t, children, permission.ResourceShipmentStop.String())
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
	assert.Equal(t, permission.SensitivityInternal, reg.GetFieldSensitivity("", "field"))
}

func TestRegistry_GetFieldSensitivity_EmptyField(t *testing.T) {
	t.Parallel()

	reg := NewEmptyRegistry()

	def := &permissionspec.ResourceDefinition{
		Resource:           "test",
		DefaultSensitivity: permission.SensitivityRestricted,
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"secret": permission.SensitivityConfidential,
		},
	}
	require.NoError(t, reg.Register(def))

	assert.Equal(t, permission.SensitivityRestricted, reg.GetFieldSensitivity("test", ""))
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
			_, _ = reg.Get(permission.ResourceShipment.String())
			_ = reg.HasResource(permission.ResourceUser.String())
			_ = reg.GetCategories()
			_ = reg.GetByCategory("Operations")
			_ = reg.GetChildren(permission.ResourceShipment.String())
			_ = reg.GetOperationsForResource(permission.ResourceShipment.String())
			_ = reg.GetFieldSensitivity(permission.ResourceWorker.String(), "name")
			_ = reg.GetEffectiveResource(permission.ResourceShipmentMove.String())
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

func TestRegistry_EveryResourceSitsInACategoryTheRoleEditorOrders(t *testing.T) {
	t.Parallel()

	order := CategoryOrder()
	known := make(map[string]struct{}, len(order))
	for _, category := range order {
		known[category] = struct{}{}
	}
	for _, def := range NewRegistry().All() {
		_, ok := known[def.Category]
		assert.Truef(t, ok, "%s is in category %q, which CategoryOrder does not list",
			def.Resource, def.Category)
	}
}

func TestRegistry_GetCategoriesFollowsTheModuleRail(t *testing.T) {
	t.Parallel()

	assert.Equal(t, CategoryOrder(), NewRegistry().GetCategories())

	reg := NewEmptyRegistry()
	require.NoError(
		t,
		reg.Register(&permissionspec.ResourceDefinition{Resource: "b", Category: "Zeta"}),
	)
	require.NoError(
		t,
		reg.Register(&permissionspec.ResourceDefinition{Resource: "a", Category: "Billing"}),
	)
	require.NoError(
		t,
		reg.Register(&permissionspec.ResourceDefinition{Resource: "c", Category: "Alpha"}),
	)
	require.NoError(
		t,
		reg.Register(&permissionspec.ResourceDefinition{Resource: "d", Category: "Desk"}),
	)
	assert.Equal(t, []string{"Desk", "Billing", "Alpha", "Zeta"}, reg.GetCategories())
}

func TestRegistry_ModulesAreNamedAsTheAppNamesThem(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	for resource, want := range map[permission.Resource]struct{ name, category string }{
		permission.ResourceAssistant:      {"Desk", "Desk"},
		permission.ResourceInboundMessage: {"Triage", "Triage"},
		permission.ResourceCaptureBatch:   {"Intake", "Intake"},
		permission.ResourceAgentProposal:  {"Decisions", "Desk"},
		permission.ResourceDetention:      {"Detention", "Detention"},
	} {
		def, ok := reg.Get(resource.String())
		require.True(t, ok, resource)
		assert.Equal(t, want.name, def.DisplayName, resource)
		assert.Equal(t, want.category, def.Category, resource)
	}
}

func TestRegistry_WorkingDetentionIsApartFromItsPolicies(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	assert.ElementsMatch(t, []permission.Operation{permission.OpRead, permission.OpUpdate},
		reg.GetOperationsForResource(permission.ResourceDetention.String()))
	assert.Contains(
		t,
		reg.GetOperationsForResource(permission.ResourceDetentionPolicy.String()),
		permission.OpDelete,
	)

	assert.Equal(t, permission.ResourceDetention, permission.KindDetentionOccurrence.Resource())
}
