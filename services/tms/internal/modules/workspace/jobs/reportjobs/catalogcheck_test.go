package reportjobs

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/reportengine"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

const (
	shipmentReportStart   = `{"version":2,"entity":"shipment","columns":[{"id":"pro","dimension":`
	validShipmentReport   = shipmentReportStart + `{"field":"proNumber"}}]}`
	brokenShipmentReport  = shipmentReportStart + `{"field":"proNumbr"}}]}`
	staleCatalogVersion   = "catalog-before-the-deploy"
	catalogCheckTestOrgID = pulid.ID("org_test")
)

type catalogCheckRepo struct {
	reportports.ReportDefinitionRepository

	mu          sync.Mutex
	definitions []*report.ReportDefinition
}

func (r *catalogCheckRepo) ListCatalogStale(
	_ context.Context,
	req *reportports.ListCatalogStaleReportDefinitionsRequest,
) ([]*report.ReportDefinition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*report.ReportDefinition, 0, req.Limit)
	for _, def := range r.definitions {
		if def.CatalogVersion == req.CatalogVersion {
			continue
		}
		copied := *def
		out = append(out, &copied)
		if len(out) == req.Limit {
			break
		}
	}
	return out, nil
}

func (r *catalogCheckRepo) RecordCatalogCheck(
	_ context.Context,
	req *reportports.RecordReportCatalogCheckRequest,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, def := range r.definitions {
		if def.ID != req.DefinitionID {
			continue
		}
		if def.Version != req.Version || def.CatalogVersion == req.CatalogVersion {
			return false, nil
		}
		def.Status = req.Status
		def.Diagnostics = req.Diagnostics
		def.CatalogVersion = req.CatalogVersion
		return true, nil
	}
	return false, nil
}

func (r *catalogCheckRepo) get(id pulid.ID) *report.ReportDefinition {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, def := range r.definitions {
		if def.ID == id {
			copied := *def
			return &copied
		}
	}
	return nil
}

type catalogCheckEngine struct {
	reportports.ReportEngine

	catalog *analytics.Catalog
}

func (e *catalogCheckEngine) Catalog() *analytics.Catalog { return e.catalog }

func (e *catalogCheckEngine) Validate(def *ir.Definition) error {
	_, err := ir.Resolve(def, e.catalog, ir.Limits{})
	return err
}

type recordingAudit struct {
	auditports.AuditService

	mu      sync.Mutex
	entries []*auditports.LogActionParams
}

func (a *recordingAudit) LogAction(
	params *auditports.LogActionParams,
	_ ...auditports.LogOption,
) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, params)
	return nil
}

func savedReport(
	t *testing.T,
	id string,
	status report.DefinitionStatus,
	definition string,
) *report.ReportDefinition {
	t.Helper()
	decoded, err := ir.Decode([]byte(definition))
	require.NoError(t, err)
	return &report.ReportDefinition{
		ID:             pulid.ID(id),
		OrganizationID: catalogCheckTestOrgID,
		BusinessUnitID: pulid.ID("bu_test"),
		Name:           id,
		Status:         status,
		CatalogVersion: staleCatalogVersion,
		Definition:     decoded,
		Version:        3,
	}
}

func runCatalogCheck(t *testing.T, a *Activities) *CheckCatalogResult {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(a.CheckDefinitionsAgainstCatalogActivity)

	encoded, err := env.ExecuteActivity(a.CheckDefinitionsAgainstCatalogActivity)
	require.NoError(t, err)
	var result CheckCatalogResult
	require.NoError(t, encoded.Get(&result))
	return &result
}

func TestCheckDefinitionsAgainstCatalog_MarksWhatNoLongerValidatesAndRestoresWhatDoes(
	t *testing.T,
) {
	t.Parallel()

	catalog, err := reportengine.NewCatalog()
	require.NoError(t, err)

	current := savedReport(t, "rd_current", report.DefinitionStatusActive, brokenShipmentReport)
	current.CatalogVersion = catalog.Version()

	repo := &catalogCheckRepo{definitions: []*report.ReportDefinition{
		savedReport(t, "rd_valid", report.DefinitionStatusActive, validShipmentReport),
		savedReport(t, "rd_broken", report.DefinitionStatusActive, brokenShipmentReport),
		savedReport(t, "rd_repaired", report.DefinitionStatusNeedsAttention, validShipmentReport),
		savedReport(t, "rd_draft", report.DefinitionStatusDraft, brokenShipmentReport),
		current,
	}}
	audit := &recordingAudit{}
	a := &Activities{
		defRepo: repo,
		engine:  &catalogCheckEngine{catalog: catalog},
		audit:   audit,
		l:       zap.NewNop(),
	}

	result := runCatalogCheck(t, a)
	assert.Equal(t, CheckCatalogResult{Checked: 4, NeedsAttention: 1, Restored: 1}, *result)

	valid := repo.get("rd_valid")
	assert.Equal(t, report.DefinitionStatusActive, valid.Status)
	assert.Empty(t, valid.Diagnostics)
	assert.Equal(t, catalog.Version(), valid.CatalogVersion)

	broken := repo.get("rd_broken")
	assert.Equal(t, report.DefinitionStatusNeedsAttention, broken.Status)
	require.Len(t, broken.Diagnostics, 1)
	assert.Contains(t, broken.Diagnostics[0], "definition.columns[0].dimension.field")
	assert.Contains(t, broken.Diagnostics[0], "proNumbr")

	repaired := repo.get("rd_repaired")
	assert.Equal(t, report.DefinitionStatusActive, repaired.Status)
	assert.Empty(t, repaired.Diagnostics)

	draft := repo.get("rd_draft")
	assert.Equal(t, report.DefinitionStatusDraft, draft.Status,
		"a draft is not run, so it keeps its status and only learns what is wrong")
	assert.NotEmpty(t, draft.Diagnostics)

	untouched := repo.get("rd_current")
	assert.Equal(t, report.DefinitionStatusActive, untouched.Status,
		"a definition already checked against this catalog is not checked again")
	assert.Empty(t, untouched.Diagnostics)

	require.Len(t, audit.entries, 2, "only a change of status is audited")
	for _, entry := range audit.entries {
		assert.Equal(t, permission.ResourceReport, entry.Resource)
		assert.Equal(t, permission.OpUpdate, entry.Operation)
		assert.Equal(t, permissionports.PrincipalTypeSystem, entry.PrincipalType)
		assert.Equal(t, catalogCheckTestOrgID, entry.OrganizationID)
	}

	again := runCatalogCheck(t, a)
	assert.Equal(t, CheckCatalogResult{}, *again, "a second pass finds nothing to check")
	assert.Len(t, audit.entries, 2)
}

func TestCheckDefinitionsAgainstCatalog_LeavesADefinitionSavedMeanwhileAlone(t *testing.T) {
	t.Parallel()

	catalog, err := reportengine.NewCatalog()
	require.NoError(t, err)

	saved := savedReport(t, "rd_saved", report.DefinitionStatusActive, brokenShipmentReport)
	repo := &catalogCheckRepo{definitions: []*report.ReportDefinition{saved}}
	audit := &recordingAudit{}
	a := &Activities{
		defRepo: &racingRepo{catalogCheckRepo: repo},
		engine:  &catalogCheckEngine{catalog: catalog},
		audit:   audit,
		l:       zap.NewNop(),
	}

	result := runCatalogCheck(t, a)
	assert.Equal(t, 1, result.Checked)
	assert.Zero(t, result.NeedsAttention)
	assert.Equal(t, report.DefinitionStatusActive, repo.get("rd_saved").Status)
	assert.Empty(t, audit.entries)
}

type racingRepo struct {
	*catalogCheckRepo
}

func (r *racingRepo) RecordCatalogCheck(
	ctx context.Context,
	req *reportports.RecordReportCatalogCheckRequest,
) (bool, error) {
	r.mu.Lock()
	for _, def := range r.definitions {
		if def.ID == req.DefinitionID {
			def.Version++
			def.CatalogVersion = req.CatalogVersion
		}
	}
	r.mu.Unlock()
	return r.catalogCheckRepo.RecordCatalogCheck(ctx, req)
}

func TestCatalogCheckStatus(t *testing.T) {
	t.Parallel()

	broken := []string{"definition.columns[0].dimension.field: unknown field"}
	cases := []struct {
		current     report.DefinitionStatus
		diagnostics []string
		want        report.DefinitionStatus
	}{
		{report.DefinitionStatusActive, nil, report.DefinitionStatusActive},
		{report.DefinitionStatusActive, broken, report.DefinitionStatusNeedsAttention},
		{report.DefinitionStatusNeedsAttention, nil, report.DefinitionStatusActive},
		{report.DefinitionStatusNeedsAttention, broken, report.DefinitionStatusNeedsAttention},
		{report.DefinitionStatusDraft, broken, report.DefinitionStatusDraft},
		{report.DefinitionStatusArchived, broken, report.DefinitionStatusArchived},
		{report.DefinitionStatusArchived, nil, report.DefinitionStatusArchived},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, catalogCheckStatus(tc.current, tc.diagnostics),
			"%s with %d diagnostics", tc.current, len(tc.diagnostics))
	}
}
