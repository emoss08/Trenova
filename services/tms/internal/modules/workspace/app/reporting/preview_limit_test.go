package reporting

import (
	"context"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type emptyDataset struct{}

func (emptyDataset) Schema() []reportports.ReportResultColumn { return nil }

func (emptyDataset) Next(context.Context) (reportports.ReportRow, error) { return nil, io.EOF }

func (emptyDataset) Totals(context.Context) (reportports.ReportRow, error) { return nil, nil }

func (emptyDataset) RowCount() int64 { return 0 }

func (emptyDataset) Truncated() bool { return false }

func (emptyDataset) Close() error { return nil }

type recordingExecutor struct {
	opened []*reportports.OpenReportDatasetRequest
}

func (e *recordingExecutor) Open(
	_ context.Context,
	req *reportports.OpenReportDatasetRequest,
) (reportports.ReportDatasetReader, error) {
	e.opened = append(e.opened, req)

	return emptyDataset{}, nil
}

func previewLimitService(compiler *recordingCompiler, executor *recordingExecutor) *Service {
	return &Service{
		l:        zap.NewNop(),
		orgRepo:  previewOrganizations{},
		compiler: compiler,
		executor: executor,
		cfg:      &config.ReportingConfig{PreviewRowLimit: 40},
		previews: newPreviewLimiter(),
	}
}

func TestPreview_CompilesForTheRowsItKeeps(t *testing.T) {
	t.Parallel()

	compiler := &recordingCompiler{}
	executor := &recordingExecutor{}
	svc := previewLimitService(compiler, executor)

	_, err := svc.Preview(t.Context(), &PreviewRequest{
		Request:    planRequest(testUserID()),
		Definition: storedRunDefinition(),
	})
	require.NoError(t, err)

	require.Len(t, compiler.previewRequests, 1)
	assert.Equal(t, int64(40), compiler.previewRequests[0].MaxRows)
	require.Len(t, executor.opened, 1)
	assert.Equal(t, int64(40), executor.opened[0].MaxRows)
}

func TestDrillThrough_LeavesTheRowCapToThePreview(t *testing.T) {
	t.Parallel()

	for _, requested := range []int{0, 40, 5000} {
		compiler := &recordingCompiler{}
		svc := previewLimitService(compiler, &recordingExecutor{})
		svc.access = drillAccess()

		_, err := svc.DrillThrough(t.Context(), &DrillThroughRequest{
			Request:    planRequest(testUserID()),
			Definition: drillSource(),
			Limit:      requested,
		})
		require.NoError(t, err, requested)

		require.Len(t, compiler.previewRequests, 1, requested)
		assert.Equal(t, int64(40), compiler.previewRequests[0].MaxRows, requested)
		assert.Zero(t, compiler.previewRequests[0].Definition.Limit,
			"a drill of %d rows asks for no fewer rows than the preview keeps", requested)
	}

	compiler := &recordingCompiler{}
	svc := previewLimitService(compiler, &recordingExecutor{})
	svc.access = drillAccess()
	_, err := svc.DrillThrough(t.Context(), &DrillThroughRequest{
		Request:    planRequest(testUserID()),
		Definition: drillSource(),
		Limit:      10,
	})
	require.NoError(t, err)
	assert.Equal(t, 10, compiler.previewRequests[0].Definition.Limit)
}

func drillSource() *report.Definition {
	return &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{{
			ID:   "c_status",
			Ref:  report.FieldRef{Field: "status"},
			Kind: report.ColumnKindDimension,
		}},
	}
}

func drillAccess() *reportaccess.Resolver {
	return reportaccess.New(reportaccess.Params{
		Catalog: &reportcatalog.Default,
		Engine:  &confidentialReader{},
		Fields:  groupAccess{visible: permission.AllFieldGroups()},
	})
}

func testUserID() pulid.ID {
	return pulid.MustNew("usr_")
}
