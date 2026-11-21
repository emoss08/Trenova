package reporting

import (
	"context"
	"errors"
	"io"
	"maps"
	"testing"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	accountingAndCash = "Accounting & Cash"
	safetyCompliance  = "Safety & Compliance"
)

type idleRuns struct {
	reportports.ReportRunRepository
}

func (idleRuns) CountActive(
	context.Context,
	*reportports.CountActiveReportRunsRequest,
) (*reportports.ActiveReportRunCounts, error) {
	return &reportports.ActiveReportRunCounts{}, nil
}

func baseOnlyService(t *testing.T) *Service {
	t.Helper()
	svc, _ := engineService(t, engineOptions{plan: plantest.BaseOnly()})
	return svc
}

func factoringSchedule() *ir.Definition {
	return &ir.Definition{
		Version: ir.Version,
		Entity:  "submission",
		Columns: []ir.Column{{ID: "c_number", Dimension: &ir.Dimension{Field: "number"}}},
	}
}

func addOnOf(feature platformcatalog.FeatureKey) (string, bool) {
	known, ok := platformcatalog.Lookup(feature)
	if !ok || known.IsBase() {
		return "", false
	}
	addOn, ok := platformcatalog.AddOnFor(feature)
	return addOn.Name, ok
}

func requirePlanRestriction(t *testing.T, err error, addOn string) *errortypes.PlanRestrictionError {
	t.Helper()
	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.Truef(t, ok, "want PLAN_RESTRICTED, got %v", err)
	assert.Equal(t, errortypes.ErrPlanRestricted, restriction.Code)
	assert.Equal(t, addOn, restriction.AddOn)
	return restriction
}

func TestCatalog_ABaseOnlyPlanOffersNoAddOnTableAndNoOffPlanField(t *testing.T) {
	t.Parallel()

	svc := baseOnlyService(t)
	registry := permissioncatalog.NewRegistry()
	request := planRequest(testUserID())
	catalog, err := svc.Catalog(t.Context(), &request)
	require.NoError(t, err)

	analyticsCatalog := svc.engine.Catalog()
	offered := make(map[string]bool, len(catalog.Entities))
	for i := range catalog.Entities {
		decided := &catalog.Entities[i]
		offered[decided.View.Key] = true
		feature, _ := registry.FeatureOf(decided.Resource.String())
		_, isAddOn := addOnOf(feature)
		assert.Falsef(t, isAddOn, "%s belongs to an add-on and is offered", decided.View.Key)
		require.Len(t, decided.Fields, len(decided.View.Fields))
		require.Len(t, decided.Edges, len(decided.View.Edges))

		entity, ok := analyticsCatalog.Entity(decided.View.Key)
		require.True(t, ok)
		shown := make(map[string]bool, len(decided.View.Fields))
		for _, field := range decided.View.Fields {
			shown[field.Key] = true
		}
		for _, field := range entity.Fields {
			key := reportaccess.APIKey(field.Entity.Table.Columns[field.Column].Name)
			fieldFeature, declared := registry.FieldFeatureOf(decided.Resource.String(), key)
			if _, off := addOnOf(fieldFeature); declared && off {
				assert.Falsef(t, shown[field.Key], "%s.%s is off the plan and offered",
					decided.View.Key, field.Key)
			}
		}
		for j := range decided.View.Edges {
			target, found := analyticsCatalog.Entity(decided.View.Edges[j].Target)
			require.True(t, found)
			targetFeature, _ := registry.FeatureOf(svc.access.Resource(target).String())
			_, off := addOnOf(targetFeature)
			assert.Falsef(t, off, "%s.%s leads to an add-on table",
				decided.View.Key, decided.View.Edges[j].Name)
		}
	}

	withheldTables := 0
	for _, withheld := range catalog.WithheldByPlan {
		assert.NotEmpty(t, withheld.Denial.AddOnName)
		if withheld.Name == "" {
			withheldTables++
			assert.Falsef(t, offered[withheld.Entity], "%s is withheld and offered", withheld.Entity)
		}
	}
	assert.Equal(t, len(analyticsCatalog.Entities()), len(catalog.Entities)+withheldTables,
		"every table is either offered or named as withheld by the plan")
	submission, ok := catalog.WithheldEntity("submission")
	require.True(t, ok, "factoring schedules are an add-on table")
	assert.Equal(t, accountingAndCash, submission.Denial.AddOnName)
}

func TestCatalog_EveryFeatureOnOffersEveryTable(t *testing.T) {
	t.Parallel()

	svc, engine := engineService(t, engineOptions{})
	request := planRequest(testUserID())
	catalog, err := svc.Catalog(t.Context(), &request)
	require.NoError(t, err)

	assert.Len(t, catalog.Entities, len(engine.Catalog().Entities()))
	assert.Empty(t, catalog.WithheldByPlan)
}

func TestRuns_EveryKindOfRunOnAnAddOnTableIsPlanRestricted(t *testing.T) {
	t.Parallel()

	svc := baseOnlyService(t)
	request := planRequest(testUserID())

	_, err := svc.Preview(t.Context(), &PreviewRequest{
		Request:    request,
		Definition: factoringSchedule(),
	})
	restriction := requirePlanRestriction(t, err, accountingAndCash)
	assert.Equal(t, string(platformcatalog.FeatureFactoring), restriction.Feature)
	assert.Contains(t, restriction.Error(),
		"Factoring Schedules are part of the Accounting & Cash add-on, which is not on your plan")

	_, err = svc.DrillThrough(t.Context(), &DrillThroughRequest{
		Request:    request,
		Definition: factoringSchedule(),
		Dimensions: []DrillValue{{ColumnID: "c_number", Value: "S-1"}},
	})
	requirePlanRestriction(t, err, accountingAndCash)

	_, err = svc.StreamCSV(t.Context(), &StreamCSVRequest{
		Request:    request,
		Title:      "Factoring",
		Definition: factoringSchedule(),
	}, io.Discard)
	requirePlanRestriction(t, err, accountingAndCash)

	err = svc.Check(t.Context(), &PrepareParams{
		Tenant:     request.TenantInfo,
		Definition: factoringSchedule(),
		Mode:       reportaccess.ModeExport,
	})
	requirePlanRestriction(t, err, accountingAndCash)
}

func TestRuns_APlanRefusalWinsOverAPermissionRefusal(t *testing.T) {
	t.Parallel()

	svc, _ := engineService(t, engineOptions{
		plan:   plantest.BaseOnly(),
		denied: []permission.Resource{permission.ResourceFactoringSubmission},
	})

	_, err := svc.Preview(t.Context(), &PreviewRequest{
		Request:    planRequest(testUserID()),
		Definition: factoringSchedule(),
	})
	requirePlanRestriction(t, err, accountingAndCash)
}

func registryWithFieldFeatures(
	t *testing.T,
	resource permission.Resource,
	features map[string]platformcatalog.FeatureKey,
) *permissioncatalog.Registry {
	t.Helper()
	registry := permissioncatalog.NewEmptyRegistry()
	for _, def := range permissioncatalog.NewRegistry().All() {
		copied := *def
		if def.Resource == resource.String() {
			copied.FieldFeatures = make(map[string]platformcatalog.FeatureKey, len(features))
			maps.Copy(copied.FieldFeatures, def.FieldFeatures)
			maps.Copy(copied.FieldFeatures, features)
		}
		require.NoError(t, registry.Register(&copied))
	}
	return registry
}

func workerProfileService(t *testing.T, gate *plantest.Gate) *Service {
	t.Helper()
	svc, _ := engineService(t, engineOptions{
		plan: gate,
		registry: registryWithFieldFeatures(t, permission.ResourceWorker,
			map[string]platformcatalog.FeatureKey{
				"complianceStatus": platformcatalog.FeatureWorkforceCompliance,
				"isQualified":      platformcatalog.FeatureWorkforceCompliance,
				"safetyRating":     platformcatalog.FeatureWorkforceSafety,
				"safetyScore":      platformcatalog.FeatureWorkforceSafety,
			}),
	})
	return svc
}

func workerDefinition(fields ...string) *ir.Definition {
	columns := make([]ir.Column, 0, len(fields))
	for i, field := range fields {
		columns = append(columns, ir.Column{
			ID:        "c" + string(rune('a'+i)),
			Dimension: &ir.Dimension{Field: field},
		})
	}
	return &ir.Definition{Version: ir.Version, Entity: "worker", Columns: columns}
}

func TestWorkerProfile_ABaseOnlyPlanLeavesOutTheComputedComplianceColumns(t *testing.T) {
	t.Parallel()

	svc := workerProfileService(t, plantest.BaseOnly())
	request := planRequest(testUserID())
	catalog, err := svc.Catalog(t.Context(), &request)
	require.NoError(t, err)

	var profile *CatalogEntity
	for i := range catalog.Entities {
		if catalog.Entities[i].View.Key == "worker_profile" {
			profile = &catalog.Entities[i]
		}
	}
	require.NotNil(t, profile, "the worker profile is a base table")
	shown := make(map[string]bool, len(profile.View.Fields))
	for _, field := range profile.View.Fields {
		shown[field.Key] = true
	}
	assert.True(t, shown["licenseExpiry"], "the entered licence dates stay base")
	assert.True(t, shown["medicalCardExpiry"])
	assert.False(t, shown["complianceStatus"])
	assert.False(t, shown["safetyScore"])

	withheld := make(map[string]string)
	for _, item := range catalog.WithheldByPlan {
		if item.Entity == "worker_profile" {
			withheld[item.Name] = item.Denial.AddOnName
		}
	}
	assert.Equal(t, safetyCompliance, withheld["complianceStatus"])
	assert.Equal(t, safetyCompliance, withheld["safetyScore"])
	assert.NotContains(t, withheld, "licenseExpiry")
}

func TestWorkerProfile_ADefinitionNamingAnOffPlanColumnIsRefusedWithThePlanReason(t *testing.T) {
	t.Parallel()

	svc := workerProfileService(t, plantest.BaseOnly())
	request := planRequest(testUserID())

	err := svc.Check(t.Context(), &PrepareParams{
		Tenant:     request.TenantInfo,
		Definition: workerDefinition("wholeName", "profile.licenseExpiry"),
		Mode:       reportaccess.ModeRead,
	})
	require.NoError(t, err, "a report on the entered licence dates runs on the base plan")

	err = svc.Check(t.Context(), &PrepareParams{
		Tenant:     request.TenantInfo,
		Definition: workerDefinition("wholeName", "profile.complianceStatus"),
		Mode:       reportaccess.ModeRead,
	})
	restriction := requirePlanRestriction(t, err, safetyCompliance)
	assert.Equal(t, string(platformcatalog.FeatureWorkforceCompliance), restriction.Feature)
	assert.Equal(t,
		"This report reads data your plan does not include: \"Compliance Status\" on Worker Profile "+
			"is part of the Safety & Compliance add-on, which is not on your plan",
		restriction.Error())
}

func TestWorkerProfile_EveryFeatureOnKeepsTheComplianceColumns(t *testing.T) {
	t.Parallel()

	svc := workerProfileService(t, plantest.AllFeatures())
	request := planRequest(testUserID())

	err := svc.Check(t.Context(), &PrepareParams{
		Tenant:     request.TenantInfo,
		Definition: workerDefinition("profile.complianceStatus", "profile.safetyScore"),
		Mode:       reportaccess.ModeExport,
	})
	require.NoError(t, err)
}

func cannedService(t *testing.T, gate *plantest.Gate) *Service {
	t.Helper()
	svc, _ := engineService(t, engineOptions{plan: gate})
	library, err := canned.Default()
	require.NoError(t, err)
	svc.canned = library
	svc.runRepo = idleRuns{}
	svc.cfg = &reportingconfig.Config{PreviewRowLimit: 40}
	return svc
}

func addOnsReadBy(t *testing.T, svc *Service, def *ir.Definition) map[string]bool {
	t.Helper()
	registry := permissioncatalog.NewRegistry()
	catalog := svc.engine.Catalog()
	addOns := make(map[string]bool)
	note := func(resource permission.Resource, field string) {
		feature, _ := registry.FeatureOf(resource.String())
		if name, off := addOnOf(feature); off {
			addOns[name] = true
		}
		if field == "" {
			return
		}
		if fieldFeature, declared := registry.FieldFeatureOf(resource.String(), field); declared {
			if name, off := addOnOf(fieldFeature); off {
				addOns[name] = true
			}
		}
	}

	resolved, err := ir.Resolve(def, catalog, ir.Limits{})
	require.NoError(t, err)
	root, ok := catalog.Entity(def.Entity)
	require.True(t, ok)
	note(svc.access.Resource(root), "")
	for path := range resolved.Fields {
		via, field, found := catalog.Resolve(def.Entity, path)
		require.True(t, found)
		for _, edge := range via {
			note(svc.access.Resource(edge.From), edge.Name)
			note(svc.access.Resource(edge.Target), "")
			for _, picked := range svc.access.PickVia(edge) {
				note(svc.access.Resource(picked), "")
			}
		}
		note(svc.access.Resource(field.Entity),
			reportaccess.APIKey(field.Entity.Table.Columns[field.Column].Name))
	}
	return addOns
}

func TestCannedLibrary_ABaseOnlyPlanLocksEveryReportThatReadsAnAddOn(t *testing.T) {
	t.Parallel()

	svc := cannedService(t, plantest.BaseOnly())
	request := planRequest(testUserID())
	library, err := svc.CannedLibrary(t.Context(), &request)
	require.NoError(t, err)
	require.Len(t, library, len(svc.canned.All()), "every canned report is listed, locked or not")

	locked, wantLocked := 0, 0
	for i := range library {
		entry := &library[i]
		needs := addOnsReadBy(t, svc, entry.Entry.Definition)
		if len(needs) == 0 {
			assert.Truef(t, entry.Available, "%s reads only base data", entry.Entry.Key)
			assert.Nil(t, entry.Plan)
			continue
		}
		wantLocked++
		if assert.Falsef(t, entry.Available, "%s reads an add-on", entry.Entry.Key) &&
			assert.NotNil(t, entry.Plan) {
			locked++
			assert.Containsf(t, needs, entry.Plan.AddOnName,
				"%s names an add-on it reads", entry.Entry.Key)
		}
	}
	assert.Equal(t, wantLocked, locked)
	t.Logf("%d of %d canned reports are locked on a base-only plan", locked, len(library))

	byKey := make(map[string]*CannedReport, len(library))
	for i := range library {
		byKey[library[i].Entry.Key] = &library[i]
	}
	for key, addOn := range map[string]string{
		"accounting-open-drift-by-customer": accountingAndCash,
		"ar-aging-by-document":              accountingAndCash,
		"credential-expiry-forecast":        safetyCompliance,
		"dqf-credential-detail":             safetyCompliance,
		"factoring-days-to-fund":            accountingAndCash,
		"factoring-fees-by-customer":        accountingAndCash,
		"factoring-schedules":               accountingAndCash,
		"invoices-ready-to-factor":          accountingAndCash,
		"pto-liability":                     "HR Suite",
		"revenue-by-gl-account":             accountingAndCash,
		"settlement-register":               "Driver Pay",
		"trial-balance":                     accountingAndCash,
	} {
		entry, ok := byKey[key]
		require.Truef(t, ok, "canned report %s is missing", key)
		assert.Falsef(t, entry.Available, "%s is built on an add-on table", key)
		if assert.NotNil(t, entry.Plan, key) {
			assert.Equalf(t, addOn, entry.Plan.AddOnName, "%s", key)
		}
	}
	syncExceptions, ok := byKey["accounting-sync-exceptions-by-week"]
	require.True(t, ok, "canned report accounting-sync-exceptions-by-week is missing")
	assert.True(t, syncExceptions.Available,
		"the sync exceptions read accounting integrations, a base feature")
	assert.Nil(t, syncExceptions.Plan)
}

func TestCannedLibrary_EveryFeatureOnLeavesEveryReportAvailable(t *testing.T) {
	t.Parallel()

	svc := cannedService(t, plantest.AllFeatures())
	request := planRequest(testUserID())
	library, err := svc.CannedLibrary(t.Context(), &request)
	require.NoError(t, err)

	for i := range library {
		assert.Truef(t, library[i].Available, "%s", library[i].Entry.Key)
		assert.Nil(t, library[i].Plan)
	}
}

func TestCannedReport_RunningOrCustomizingALockedReportIsPlanRestricted(t *testing.T) {
	t.Parallel()

	svc := cannedService(t, plantest.BaseOnly())
	request := planRequest(testUserID())

	_, err := svc.RunReport(t.Context(), &RunReportRequest{
		Request:   request,
		CannedKey: "factoring-schedules",
		Format:    report.FormatCSV,
	})
	requirePlanRestriction(t, err, accountingAndCash)

	_, err = svc.ForkCanned(t.Context(), &ForkCannedRequest{
		Request:   request,
		CannedKey: "factoring-schedules",
	})
	restriction := requirePlanRestriction(t, err, accountingAndCash)
	assert.Equal(t,
		"\"Factoring Schedules\" is part of the Accounting & Cash add-on, which is not on your plan",
		restriction.Error())
}
