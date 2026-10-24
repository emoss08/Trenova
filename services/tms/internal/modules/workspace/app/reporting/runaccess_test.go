package reporting

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRevisions struct {
	reportports.ReportDefinitionRepository

	definition *report.Definition
	asked      pulid.ID
}

func (s *stubRevisions) GetRevision(
	_ context.Context,
	req *reportports.GetReportRevisionRequest,
) (*report.ReportDefinitionRevision, error) {
	s.asked = req.RevisionID
	return &report.ReportDefinitionRevision{ID: req.RevisionID, Definition: s.definition}, nil
}

type recordingCompiler struct {
	err             error
	exports         []*reportports.ReportCompileRequest
	previews        int
	previewRequests []*reportports.ReportCompileRequest
	validation      int
}

func (c *recordingCompiler) ValidateAndAuthorize(
	context.Context,
	*reportports.ReportCompileRequest,
) (*reportports.ReportValidationResult, error) {
	c.validation++
	return &reportports.ReportValidationResult{}, c.err
}

func (c *recordingCompiler) Compile(
	_ context.Context,
	req *reportports.ReportCompileRequest,
) (*reportports.CompiledReportQuery, error) {
	c.exports = append(c.exports, req)
	if c.err != nil {
		return nil, c.err
	}
	return &reportports.CompiledReportQuery{}, nil
}

func (c *recordingCompiler) CompileForPreview(
	_ context.Context,
	req *reportports.ReportCompileRequest,
) (*reportports.CompiledReportQuery, error) {
	c.previews++
	c.previewRequests = append(c.previewRequests, req)
	if c.err != nil {
		return nil, c.err
	}
	return &reportports.CompiledReportQuery{}, nil
}

type presigningStorage struct {
	stubRowsStorage

	presigned string
}

func (s *presigningStorage) GetPresignedURL(
	_ context.Context,
	params *kernelports.PresignedURLParams,
) (string, error) {
	s.presigned = params.Key
	return "https://storage.example/" + params.Key, nil
}

func storedRunDefinition() *report.Definition {
	return &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{{
			ID:   "c_customer",
			Ref:  report.FieldRef{Path: []string{"customer"}, Field: "name"},
			Kind: report.ColumnKindDimension,
		}},
	}
}

var errAccessWithdrawn = errortypes.NewAuthorizationError(
	"This report references data you do not have access to: {0}",
	"you do not have read access to Customers",
)

func TestReadRunRows_RechecksTheReportAgainstTheRequestersCurrentGrants(t *testing.T) {
	t.Parallel()

	store := &stubRowsStorage{body: []byte(storedEnvelope)}
	run := storedRun()
	run.Params = map[string]any{"windowDays": 30}
	svc := rowsService(run, store)
	compiler := &recordingCompiler{}
	revisions := &stubRevisions{definition: storedRunDefinition()}
	svc.compiler = compiler
	svc.defRepo = revisions
	req := rowsRequest()

	_, err := svc.ReadRunRows(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, run.RevisionID, revisions.asked, "the run's own revision is checked")
	require.Len(t, compiler.exports, 1, "a stored run is checked as the export it was")
	assert.Zero(t, compiler.previews)
	assert.Equal(t, req.TenantInfo.UserID, compiler.exports[0].Tenant.UserID)
	assert.Equal(t, run.Params, compiler.exports[0].Params)
	assert.Equal(t, "shipment", compiler.exports[0].Definition.Entity)
}

func TestReadRunRows_RefusesWhenAccessWasWithdrawnSinceTheRun(t *testing.T) {
	t.Parallel()

	store := &stubRowsStorage{body: []byte(storedEnvelope)}
	svc := rowsService(storedRun(), store)
	svc.compiler = &recordingCompiler{err: errAccessWithdrawn}

	_, err := svc.ReadRunRows(t.Context(), rowsRequest())

	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
	assert.Empty(t, store.downloaded, "no stored row is read for a withdrawn reader")
}

func TestDownloadRun_RefusesWhenAccessWasWithdrawnSinceTheRun(t *testing.T) {
	t.Parallel()

	run := storedRun()
	run.ArtifactKey = "reports/org/run/1/report.xlsx"
	store := &presigningStorage{}
	svc := rowsService(run, &store.stubRowsStorage)
	svc.storage = store

	download, err := svc.DownloadRun(t.Context(), rowsRequest())
	require.NoError(t, err)
	assert.Equal(t, run.ArtifactKey, store.presigned)
	assert.NotEmpty(t, download.URL)

	store.presigned = ""
	svc.compiler = &recordingCompiler{err: errAccessWithdrawn}

	_, err = svc.DownloadRun(t.Context(), rowsRequest())
	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
	assert.Empty(t, store.presigned, "no download link is minted for a withdrawn reader")
}

func TestDownloadRun_RechecksACannedRunAgainstItsReport(t *testing.T) {
	t.Parallel()

	registry := canned.Default()
	entries := registry.All()
	require.NotEmpty(t, entries)

	run := storedRun()
	run.RevisionID = ""
	run.CannedKey = entries[0].Key
	run.ArtifactKey = "reports/org/run/1/report.xlsx"
	store := &presigningStorage{}
	svc := rowsService(run, &store.stubRowsStorage)
	svc.storage = store
	svc.canned = registry
	compiler := &recordingCompiler{}
	svc.compiler = compiler

	_, err := svc.DownloadRun(t.Context(), rowsRequest())
	require.NoError(t, err)
	require.Len(t, compiler.exports, 1)
	assert.Equal(t, entries[0].Definition, compiler.exports[0].Definition)
}

func TestDownloadRun_RefusesARunWhoseReportNoLongerCompiles(t *testing.T) {
	t.Parallel()

	run := storedRun()
	run.ArtifactKey = "reports/org/run/1/report.xlsx"
	store := &presigningStorage{}
	svc := rowsService(run, &store.stubRowsStorage)
	svc.storage = store
	multiErr := errortypes.NewMultiError()
	multiErr.Add("definition.columns[0].ref", errortypes.ErrInvalid, "unknown field")
	svc.compiler = &recordingCompiler{err: multiErr}

	_, err := svc.DownloadRun(t.Context(), rowsRequest())
	var businessErr *errortypes.BusinessError
	require.ErrorAs(t, err, &businessErr)
	assert.Empty(t, store.presigned)
}

func TestExportDashboard_NeedsTheExportOperationOnEveryTile(t *testing.T) {
	t.Parallel()

	dashboard := dashboardWith([]report.DashboardTile{{
		ID:           "t1",
		Kind:         report.TileKindTable,
		DefinitionID: tileDefID,
	}}, nil)
	svc := tileTestService(dashboard, map[pulid.ID]*report.ReportDefinition{
		tileDefID: sharedDefinition("shipment", nil),
	})
	compiler := &recordingCompiler{err: errortypes.NewAuthorizationError(
		"This report references data you do not have access to: {0}",
		"you do not have export access to Shipments",
	)}
	svc.compiler = compiler
	svc.orgRepo = previewOrganizations{}

	_, err := svc.ExportDashboard(t.Context(), &ExportDashboardRequest{
		Request:     tileRequest(),
		DashboardID: pulid.MustNew("rdash_"),
	})

	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
	require.Len(t, compiler.exports, 1, "each tile compiles as an export")
	assert.Zero(t, compiler.previews, "an export never compiles as a preview")
}
