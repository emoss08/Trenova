package reporting

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The stubs embed their interfaces rather than implementing every method: what
// this file exercises is one read path, and a stub that answers the rest would
// be forty methods of noise around the four that matter.

type stubRunRepo struct {
	repositories.ReportRunRepository

	run *report.ReportRun
}

func (s *stubRunRepo) GetByID(
	_ context.Context, _ *repositories.GetReportRunRequest,
) (*report.ReportRun, error) {
	return s.run, nil
}

type stubRowsStorage struct {
	storage.Client

	body        []byte
	size        int64
	statErr     error
	downloadErr error
	downloaded  string
}

func (s *stubRowsStorage) GetFileInfo(_ context.Context, key string) (*storage.FileInfo, error) {
	if s.statErr != nil {
		return nil, s.statErr
	}
	size := s.size
	if size == 0 {
		size = int64(len(s.body))
	}

	return &storage.FileInfo{Key: key, Size: size}, nil
}

func (s *stubRowsStorage) Download(
	_ context.Context, key string,
) (*storage.DownloadResult, error) {
	if s.downloadErr != nil {
		return nil, s.downloadErr
	}
	s.downloaded = key

	return &storage.DownloadResult{
		Body: io.NopCloser(bytes.NewReader(s.body)),
		Size: int64(len(s.body)),
	}, nil
}

func rowsService(run *report.ReportRun, store *stubRowsStorage) *Service {
	return &Service{
		l:       zap.NewNop(),
		runRepo: &stubRunRepo{run: run},
		storage: store,
	}
}

func storedRun() *report.ReportRun {
	return &report.ReportRun{
		ID:                pulid.MustNew("rrun_"),
		Status:            report.RunStatusSucceeded,
		Format:            report.FormatXLSX,
		RowsKey:           "reports/org/run/1/report.xlsx.rows.json",
		ArtifactExpiresAt: timeutils.NowUnix() + 3600,
	}
}

func rowsRequest() *GetRunRequest {
	return &GetRunRequest{
		Request: Request{TenantInfo: pagination.TenantInfo{UserID: pulid.MustNew("usr_")}},
		RunID:   pulid.MustNew("rrun_"),
	}
}

const storedEnvelope = `{"meta":{"title":"Revenue","generatedAt":1784131200},` +
	`"schema":[{"id":"customer","label":"Customer","type":"string","display":{}},` +
	`{"id":"revenue","label":"Revenue","type":"decimal","display":{}}],` +
	`"rows":[["ACME","1000.00"]],"summary":{"rowCount":1,"truncated":false}}`

func TestReadRunRows_ReturnsTheStoredRows(t *testing.T) {
	t.Parallel()

	store := &stubRowsStorage{body: []byte(storedEnvelope)}
	run := storedRun()

	envelope, err := rowsService(run, store).ReadRunRows(t.Context(), rowsRequest())
	require.NoError(t, err)

	assert.Equal(t, run.RowsKey, store.downloaded)
	assert.Equal(t, "Revenue", envelope.Meta.Title)
	require.Len(t, envelope.Rows, 1)
	assert.Equal(t, "1000.00", envelope.Rows[0][1])
}

// Each refusal has its own reason because a caller deciding whether two runs
// can be compared needs to know which of them is the problem and why. "No
// rows" covers a failed run, a run made before the sidecar existed and an
// expired one, and only one of those is worth running the report again for.
func TestReadRunRows_RefusesWithItsOwnReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(*report.ReportRun)
		says string
	}{
		{
			name: "a run that did not finish",
			run:  func(r *report.ReportRun) { r.Status = report.RunStatusFailed },
			says: "did not finish successfully",
		},
		{
			name: "a run made before the sidecar existed",
			run:  func(r *report.ReportRun) { r.RowsKey = "" },
			says: "before its rows were stored",
		},
		{
			name: "a run whose artifact has expired",
			run: func(r *report.ReportRun) {
				r.ArtifactExpiresAt = timeutils.NowUnix() - 1
			},
			says: "have expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			run := storedRun()
			tt.run(run)
			store := &stubRowsStorage{body: []byte(storedEnvelope)}

			_, err := rowsService(run, store).ReadRunRows(t.Context(), rowsRequest())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.says)
			assert.Empty(t, store.downloaded, "a refused run is never fetched")
		})
	}
}

// A report big enough to exhaust the process is a report nobody can compare
// anyway, and the refusal says what to do about it.
func TestReadRunRows_RefusesAnObjectOverTheCap(t *testing.T) {
	t.Parallel()

	store := &stubRowsStorage{body: []byte(storedEnvelope), size: maxRunRowsBytes + 1}

	_, err := rowsService(storedRun(), store).ReadRunRows(t.Context(), rowsRequest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too large to read back")
	assert.Empty(t, store.downloaded, "the size is checked before the read, not after")
}

// The stat can be stale, or unset on some backends, so the cap is enforced
// again on the bytes themselves — otherwise a backend that reports zero would
// wave through an object of any size.
func TestReadRunRows_RefusesAnOversizedBodyDespiteTheStat(t *testing.T) {
	t.Parallel()

	oversized := `{"meta":{},"schema":[{"id":"a","type":"string"}],"rows":[["` +
		strings.Repeat("x", maxRunRowsBytes) + `"]],"summary":{}}`
	store := &stubRowsStorage{body: []byte(oversized), size: 12}

	_, err := rowsService(storedRun(), store).ReadRunRows(t.Context(), rowsRequest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too large to read back")
}

func TestReadRunRows_ReportsAMissingObject(t *testing.T) {
	t.Parallel()

	store := &stubRowsStorage{statErr: errors.New("NoSuchKey")}

	_, err := rowsService(storedRun(), store).ReadRunRows(t.Context(), rowsRequest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer available")
}

// A sidecar whose rows do not line up with its schema cannot be compared
// positionally, and quietly returning it would mean comparing one column
// against another.
func TestReadRunRows_RefusesAMalformedEnvelope(t *testing.T) {
	t.Parallel()

	short := `{"meta":{},"schema":[{"id":"a","type":"string"},{"id":"b","type":"int"}],` +
		`"rows":[["only-one"]],"summary":{}}`
	store := &stubRowsStorage{body: []byte(short)}

	_, err := rowsService(storedRun(), store).ReadRunRows(t.Context(), rowsRequest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not be read")
}

func TestReadRunRows_RefusesBytesThatAreNotAnEnvelope(t *testing.T) {
	t.Parallel()

	store := &stubRowsStorage{body: []byte("not json at all")}

	_, err := rowsService(storedRun(), store).ReadRunRows(t.Context(), rowsRequest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not be read")
}
