package reportcatalogjobs

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/reportengine"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

type seenVersion struct {
	first, last int64
}

type catalogVersions struct {
	mu   sync.Mutex
	seen map[string]seenVersion
}

func newCatalogVersions() *catalogVersions {
	return &catalogVersions{seen: map[string]seenVersion{}}
}

func (v *catalogVersions) Heartbeat(
	_ context.Context,
	req *reportports.ReportCatalogHeartbeatRequest,
) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	seen, ok := v.seen[req.Version]
	if !ok {
		seen.first = req.Now
	}
	seen.last = req.Now
	v.seen[req.Version] = seen
	for version, other := range v.seen {
		if other.last < req.ForgetBefore {
			delete(v.seen, version)
		}
	}
	return nil
}

func (v *catalogVersions) Newest(_ context.Context, aliveSince int64) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	newest, first := "", int64(-1)
	for version, seen := range v.seen {
		if seen.last >= aliveSince && seen.first > first {
			newest, first = version, seen.first
		}
	}
	return newest, nil
}

func (v *catalogVersions) record(version string, first, last int64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.seen[version] = seenVersion{first: first, last: last}
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
		defRepo:  repo,
		engine:   &catalogCheckEngine{catalog: catalog},
		versions: newCatalogVersions(),
		audit:    audit,
		l:        zap.NewNop(),
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
		defRepo:  &racingRepo{catalogCheckRepo: repo},
		engine:   &catalogCheckEngine{catalog: catalog},
		versions: newCatalogVersions(),
		audit:    audit,
		l:        zap.NewNop(),
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

func TestCheckDefinitionsAgainstCatalog_LeavesTheCheckToTheNewestRunningVersion(t *testing.T) {
	t.Parallel()

	catalog, err := reportengine.NewCatalog()
	require.NoError(t, err)

	broken := savedReport(t, "rd_broken", report.DefinitionStatusActive, brokenShipmentReport)
	repo := &catalogCheckRepo{definitions: []*report.ReportDefinition{broken}}
	versions := newCatalogVersions()
	now := timeutils.NowUnix()
	versions.record("catalog-of-the-next-deploy", now+60, now)
	audit := &recordingAudit{}
	a := &Activities{
		defRepo:  repo,
		engine:   &catalogCheckEngine{catalog: catalog},
		versions: versions,
		audit:    audit,
		l:        zap.NewNop(),
	}

	result := runCatalogCheck(t, a)
	assert.True(t, result.Superseded)
	assert.Zero(t, result.Checked)
	assert.Equal(t, report.DefinitionStatusActive, repo.get("rd_broken").Status,
		"an older worker of a rolling deploy leaves the verdict to the newer one")
	assert.Equal(t, staleCatalogVersion, repo.get("rd_broken").CatalogVersion)

	versions.record("catalog-of-the-next-deploy", now+60, now-int64(time.Hour.Seconds()))
	result = runCatalogCheck(t, a)
	assert.False(t, result.Superseded,
		"once the newer version stops running, as after a rollback, this one checks again")
	assert.Equal(t, 1, result.NeedsAttention)
	assert.Equal(t, report.DefinitionStatusNeedsAttention, repo.get("rd_broken").Status)
}

func TestAttentionNoticesGroupEachOwnersReports(t *testing.T) {
	t.Parallel()

	owner := pulid.ID("usr_owner")
	other := pulid.ID("usr_other")
	notices := newAttentionNotices()
	for i := range maxNoticeDefinitionIDs + 5 {
		def := savedReport(t, fmt.Sprintf("rd_%02d", i), report.DefinitionStatusActive,
			validShipmentReport)
		def.OwnerID = owner
		notices.add(def)
	}
	single := savedReport(t, "rd_single", report.DefinitionStatusActive, validShipmentReport)
	single.OwnerID = other
	notices.add(single)

	list := notices.list()
	require.Len(t, list, 2, "one notice per owner, however many of their reports broke")

	byOwner := map[pulid.ID]*attentionNotice{}
	for _, notice := range list {
		byOwner[notice.Owner.OwnerID] = notice
	}
	many := byOwner[owner]
	require.NotNil(t, many)
	assert.Equal(t, maxNoticeDefinitionIDs+5, many.Count)
	assert.Len(t, many.DefinitionIDs, maxNoticeDefinitionIDs)
	assert.Equal(t, "rd_00", many.FirstReport)

	one := byOwner[other]
	require.NotNil(t, one)
	assert.Equal(t, 1, one.Count)
	assert.Equal(t, []pulid.ID{"rd_single"}, one.DefinitionIDs)
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
