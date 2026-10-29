package reporting

import (
	"context"
	"testing"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportingtest"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRevisions struct {
	reportports.ReportDefinitionRepository

	definition *ir.Definition
	asked      pulid.ID
}

func (s *stubRevisions) GetRevision(
	_ context.Context,
	req *reportports.GetReportRevisionRequest,
) (*report.ReportDefinitionRevision, error) {
	s.asked = req.RevisionID
	return &report.ReportDefinitionRevision{ID: req.RevisionID, Definition: s.definition}, nil
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

func withdrawShipments(t *testing.T, svc *Service) {
	t.Helper()
	withdrawn, _ := engineService(t, engineOptions{denied: []permission.Resource{
		permission.ResourceShipment,
	}})
	svc.access = withdrawn.access
	svc.perms = withdrawn.perms
}

func TestReadRunRows_RechecksTheReportAgainstTheRequestersCurrentGrants(t *testing.T) {
	t.Parallel()

	store := &stubRowsStorage{body: []byte(storedEnvelope)}
	run := storedRun()
	run.Params = map[string]any{"windowDays": 30}
	svc := rowsService(t, run, store)
	definition := storedRunDefinition()
	definition.Parameters = []ir.Parameter{{Name: "windowDays", Type: "int"}}
	revisions := &stubRevisions{definition: definition}
	svc.defRepo = revisions
	engine, ok := svc.engine.(*reportingtest.Engine)
	require.True(t, ok)
	req := rowsRequest()

	_, err := svc.ReadRunRows(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, run.RevisionID, revisions.asked, "the run's own revision is checked")
	compiles := engine.Recorded("compile")
	require.Len(t, compiles, 1, "a stored run is checked as the export it was")
	access, ok := compiles[0].Request.Policy.(*reportaccess.Access)
	require.True(t, ok)
	assert.Equal(t, reportaccess.ModeExport, access.Mode())
	assert.Equal(t, run.Params, compiles[0].Request.Params)
	assert.Equal(t, "shipment", compiles[0].Request.Definition.Entity)
}

func TestReadRunRows_RefusesWhenAccessWasWithdrawnSinceTheRun(t *testing.T) {
	t.Parallel()

	store := &stubRowsStorage{body: []byte(storedEnvelope)}
	svc := rowsService(t, storedRun(), store)
	withdrawShipments(t, svc)

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
	svc := rowsService(t, run, &store.stubRowsStorage)
	svc.storage = store

	download, err := svc.DownloadRun(t.Context(), rowsRequest())
	require.NoError(t, err)
	assert.Equal(t, run.ArtifactKey, store.presigned)
	assert.NotEmpty(t, download.URL)

	store.presigned = ""
	withdrawShipments(t, svc)

	_, err = svc.DownloadRun(t.Context(), rowsRequest())
	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
	assert.Empty(t, store.presigned, "no download link is minted for a withdrawn reader")
}

func TestDownloadRun_RechecksACannedRunAgainstItsReport(t *testing.T) {
	t.Parallel()

	registry, err := canned.Default()
	require.NoError(t, err)
	entries := registry.All()
	require.NotEmpty(t, entries)

	run := storedRun()
	run.RevisionID = ""
	run.CannedKey = entries[0].Key
	run.ArtifactKey = "reports/org/run/1/report.xlsx"
	store := &presigningStorage{}
	svc := rowsService(t, run, &store.stubRowsStorage)
	svc.storage = store
	svc.canned = registry
	engine, ok := svc.engine.(*reportingtest.Engine)
	require.True(t, ok)

	_, err = svc.DownloadRun(t.Context(), rowsRequest())
	require.NoError(t, err)
	compiles := engine.Recorded("compile")
	require.Len(t, compiles, 1)
	assert.Equal(t, entries[0].Definition, compiles[0].Request.Definition)
}

func TestDownloadRun_RefusesARunWhoseReportNoLongerCompiles(t *testing.T) {
	t.Parallel()

	run := storedRun()
	run.ArtifactKey = "reports/org/run/1/report.xlsx"
	store := &presigningStorage{}
	svc := rowsService(t, run, &store.stubRowsStorage)
	svc.storage = store
	svc.defRepo = &stubRevisions{definition: &ir.Definition{
		Version: ir.Version,
		Entity:  "shipment",
		Columns: []ir.Column{{ID: "gone", Dimension: &ir.Dimension{Field: "noSuchField"}}},
	}}

	_, err := svc.DownloadRun(t.Context(), rowsRequest())
	var businessErr *errortypes.BusinessError
	require.ErrorAs(t, err, &businessErr)
	assert.Empty(t, store.presigned)
}
