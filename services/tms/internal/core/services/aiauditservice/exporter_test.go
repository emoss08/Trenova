package aiauditservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"io"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// countingWriter records every write, so a test can tell a streamed file from
// one assembled in memory and written at the end.
type countingWriter struct {
	buf     bytes.Buffer
	writes  int
	largest int
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.writes++
	w.largest = max(w.largest, len(p))

	return w.buf.Write(p)
}

func seededLedger(t *testing.T, tenantInfo pagination.TenantInfo, rows int) *fakeLedger {
	t.Helper()

	ledger := newFakeLedger()
	events := make([]*aiaudit.AIAuditEvent, 0, rows)
	for i := range rows {
		events = append(events, &aiaudit.AIAuditEvent{
			SourceKey:     "usage:" + strconv.Itoa(i),
			OccurredAt:    testNow - int64(rows-i),
			Kind:          aiaudit.KindModelCall,
			Outcome:       aiaudit.OutcomeSucceeded,
			PrincipalType: aiaudit.PrincipalUser,
			PrincipalID:   "usr_1",
			Reason:        "=HYPERLINK(\"http://x\")",
			Arguments:     map[string]any{"n": strconv.Itoa(i)},
			Purpose:       aiaudit.PurposeLive,
		})
	}
	_, err := ledger.Append(t.Context(), &repositories.AppendAIAuditEventsRequest{
		TenantInfo: tenantInfo,
		Events:     events,
		Sign:       testKeyring(true).Signer(),
		KeyID:      "k1",
		Now:        testNow,
	})
	require.NoError(t, err)

	return ledger
}

func exportFor(
	tenantInfo pagination.TenantInfo,
	format aiaudit.ExportFormat,
) *aiaudit.AIAuditExport {
	return &aiaudit.AIAuditExport{
		ID:                pulid.MustNew(aiaudit.ExportIDPrefix),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		RequestedByUserID: pulid.MustNew("usr_"),
		Format:            format,
		Filters:           &aiaudit.ExportFilter{},
		RangeFrom:         testNow - 1_000_000,
		RangeTo:           testNow,
	}
}

func writeExport(
	t *testing.T,
	ledger *fakeLedger,
	export *aiaudit.AIAuditExport,
) (*countingWriter, *ExportStats) {
	t.Helper()

	summary, err := ledger.Summarize(t.Context(), scopeOf(export))
	require.NoError(t, err)
	out := &countingWriter{}
	stats, err := NewExporter(ledger, testKeyring(true)).Write(t.Context(), out, &ExportSpec{
		Export:      export,
		Summary:     summary,
		GeneratedAt: time.Unix(testNow, 0),
	})
	require.NoError(t, err)

	return out, stats
}

func TestExporter_JSONStreamsTheEnvelopeAndChain(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ledger := seededLedger(t, tenantInfo, 2500)
	export := exportFor(tenantInfo, aiaudit.ExportFormatJSON)

	out, stats := writeExport(t, ledger, export)

	assert.Greater(t, out.writes, 2, "the file is written as it is read, not at the end")
	assert.LessOrEqual(t, out.largest, exportBufferSize)
	assert.Equal(t, int64(2500), stats.Rows)
	assert.True(t, stats.Complete)
	sum := sha256.Sum256(out.buf.Bytes())
	assert.Equal(t, hex.EncodeToString(sum[:]), stats.SHA256)
	assert.Equal(t, int64(out.buf.Len()), stats.Bytes)

	var file struct {
		Export struct {
			Format string `json:"format"`
			ID     string `json:"id"`
			Chain  struct {
				KeyID    string `json:"keyId"`
				Signed   bool   `json:"signed"`
				FirstSeq int64  `json:"firstSeq"`
				LastSeq  int64  `json:"lastSeq"`
				Complete bool   `json:"complete"`
			} `json:"chain"`
		} `json:"export"`
		Events []struct {
			Seq       int64  `json:"seq"`
			PrevHash  string `json:"prevHash"`
			Hash      string `json:"hash"`
			HashKeyID string `json:"hashKeyId"`
		} `json:"events"`
	}
	require.NoError(t, sonic.Unmarshal(out.buf.Bytes(), &file))

	assert.Equal(t, ExportEnvelopeFormat, file.Export.Format)
	assert.Equal(t, export.ID.String(), file.Export.ID)
	assert.Equal(t, "k1", file.Export.Chain.KeyID)
	assert.True(t, file.Export.Chain.Signed)
	assert.Equal(t, int64(1), file.Export.Chain.FirstSeq)
	assert.Equal(t, int64(2500), file.Export.Chain.LastSeq)
	assert.True(t, file.Export.Chain.Complete)
	require.Len(t, file.Events, 2500)
	for i := 1; i < len(file.Events); i++ {
		assert.Equal(t, file.Events[i-1].Hash, file.Events[i].PrevHash)
	}
	assert.Equal(t, "k1", file.Events[0].HashKeyID)
}

func TestExporter_CSVKeepsTheColumnContractAndDefusesFormulas(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ledger := seededLedger(t, tenantInfo, 3)

	out, stats := writeExport(t, ledger, exportFor(tenantInfo, aiaudit.ExportFormatCSV))

	records, err := csv.NewReader(bytes.NewReader(out.buf.Bytes())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 4)
	assert.Equal(t, exportColumnNames(), records[0])
	assert.Equal(t, int64(3), stats.Rows)

	columns := map[string]int{}
	for i, name := range records[0] {
		columns[name] = i
	}
	first := records[1]
	assert.Equal(t, "1", first[columns["seq"]])
	assert.Equal(t, `'=HYPERLINK("http://x")`, first[columns["reason"]])
	assert.Equal(t, `{"n":"0"}`, first[columns["arguments"]])
	assert.Equal(t, aiaudit.GenesisHash, first[columns["prev_hash"]])
	assert.Equal(t, records[1][columns["hash"]], records[2][columns["prev_hash"]])
}

func TestChainComplete_OnlyForAnUnfilteredContiguousRange(t *testing.T) {
	t.Parallel()

	contiguous := &repositories.AIAuditEventSummary{Count: 10, FirstSeq: 5, LastSeq: 14}
	gapped := &repositories.AIAuditEventSummary{Count: 9, FirstSeq: 5, LastSeq: 14}
	filtered := &aiaudit.ExportFilter{FieldFilters: []domaintypes.FieldFilter{{
		Field: "kind", Operator: "eq", Value: "ToolCall",
	}}}

	assert.True(t, ChainComplete(&aiaudit.ExportFilter{}, contiguous))
	assert.True(t, ChainComplete(nil, contiguous))
	assert.False(t, ChainComplete(&aiaudit.ExportFilter{}, gapped))
	assert.False(t, ChainComplete(filtered, contiguous))
	assert.False(t, ChainComplete(nil, &repositories.AIAuditEventSummary{}))
}

type fakeStorage struct {
	storage.Client

	mu      sync.Mutex
	objects map[string][]byte
}

func (f *fakeStorage) Upload(
	_ context.Context,
	params *storage.UploadParams,
) (*storage.FileInfo, error) {
	body, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.objects == nil {
		f.objects = map[string][]byte{}
	}
	f.objects[params.Key] = body

	return &storage.FileInfo{Key: params.Key, Size: int64(len(body))}, nil
}

func (f *fakeStorage) GetPresignedURL(
	_ context.Context,
	params *storage.PresignedURLParams,
) (string, error) {
	return "https://files.example/" + params.Key, nil
}

func (f *fakeStorage) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.objects, key)

	return nil
}

type fakeExportRepo struct {
	repositories.AIAuditExportRepository

	mu      sync.Mutex
	exports map[pulid.ID]*aiaudit.AIAuditExport
}

func (f *fakeExportRepo) Create(
	_ context.Context,
	export *aiaudit.AIAuditExport,
) (*aiaudit.AIAuditExport, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if export.ID.IsNil() {
		export.ID = pulid.MustNew(aiaudit.ExportIDPrefix)
	}
	copied := *export
	f.exports[export.ID] = &copied

	return export, nil
}

func (f *fakeExportRepo) Update(
	_ context.Context,
	export *aiaudit.AIAuditExport,
) (*aiaudit.AIAuditExport, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	export.Version++
	copied := *export
	f.exports[export.ID] = &copied

	return export, nil
}

func (f *fakeExportRepo) GetByID(
	_ context.Context,
	req repositories.GetAIAuditExportRequest,
) (*aiaudit.AIAuditExport, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	copied := *f.exports[req.ID]

	return &copied, nil
}

type fakeStarter struct {
	serviceports.WorkflowStarter

	started []string
}

func (f *fakeStarter) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	_ ...any,
) (client.WorkflowRun, error) {
	f.started = append(f.started, options.ID+"|"+workflow.(string)+"|"+options.TaskQueue)

	return nil, nil
}

func testExports(
	ledger *fakeLedger,
	syncMax int,
) (*Exports, *fakeExportRepo, *fakeStarter, *fakeNotifier, *fakeAudit) {
	repo := &fakeExportRepo{exports: map[pulid.ID]*aiaudit.AIAuditExport{}}
	starter := &fakeStarter{}
	notifier := &fakeNotifier{}
	audited := &fakeAudit{}

	return NewExports(ExportsParams{
		Ledger:    ledger,
		Exports:   repo,
		Source:    &fakeSource{users: map[pulid.ID]string{}},
		Exporter:  NewExporter(ledger, testKeyring(true)),
		Storage:   &fakeStorage{},
		Workflows: starter,
		Notifier:  notifier,
		Audit:     audited,
		Config:    &config.AIAuditExportConfig{SyncMaxRows: syncMax, MaxRows: 100},
		Now:       func() time.Time { return time.Unix(testNow, 0) },
		Logger:    zap.NewNop(),
	}), repo, starter, notifier, audited
}

func actorFor(tenantInfo pagination.TenantInfo) *serviceports.RequestActor {
	user := pulid.MustNew("usr_")

	return &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    user,
		UserID:         user,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
}

func TestExports_ASmallExportIsWrittenAtOnceAndAudited(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ledger := seededLedger(t, tenantInfo, 5)
	exports, _, starter, notifier, audited := testExports(ledger, 10)
	actor := actorFor(tenantInfo)

	export, err := exports.Request(t.Context(), &serviceports.RequestAIAuditExportRequest{
		Actor:  actor,
		Format: aiaudit.ExportFormatCSV,
		From:   testNow - 1000,
		To:     testNow,
	})
	require.NoError(t, err)

	assert.Equal(t, aiaudit.ExportStatusSucceeded, export.Status)
	assert.Equal(t, int64(5), export.RowCount)
	assert.Len(t, export.SHA256, 64)
	assert.True(t, export.ChainComplete)
	assert.Empty(t, starter.started)
	require.Len(t, notifier.personal, 1)
	assert.Equal(t, serviceports.AIAuditExportReadyEvent, notifier.personal[0].EventType)
	require.Len(t, audited.entries, 1)
	assert.True(t, audited.entries[0].Critical)

	download, err := exports.Download(t.Context(), &serviceports.GetAIAuditExportDownloadRequest{
		Actor:    actor,
		ExportID: export.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, export.SHA256, download.SHA256)
	assert.Equal(t, testNow+60, download.ExpiresAt)
	assert.Len(t, audited.entries, 2, "the download is audited too")

	_, err = exports.Download(t.Context(), &serviceports.GetAIAuditExportDownloadRequest{
		Actor:    actorFor(tenantInfo),
		ExportID: export.ID,
	})
	require.Error(t, err, "nobody but the requester is handed the file")
}

func TestExports_ALargeExportRunsOnTheReportQueue(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ledger := seededLedger(t, tenantInfo, 20)
	exports, _, starter, _, _ := testExports(ledger, 10)

	export, err := exports.Request(t.Context(), &serviceports.RequestAIAuditExportRequest{
		Actor:  actorFor(tenantInfo),
		Format: aiaudit.ExportFormatJSON,
		From:   testNow - 1000,
		To:     testNow,
	})
	require.NoError(t, err)

	assert.Equal(t, aiaudit.ExportStatusRunning, export.Status)
	require.Len(t, starter.started, 1)
	assert.Contains(t, starter.started[0], serviceports.AIAuditExportWorkflowName+"|report-queue")

	finished, err := exports.Run(t.Context(), tenantInfo, export.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, aiaudit.ExportStatusSucceeded, finished.Status)
	assert.Equal(t, int64(20), finished.RowCount)
}

func TestExports_AnExportPastTheCapIsRefused(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ledger := seededLedger(t, tenantInfo, 150)
	exports, repo, _, _, _ := testExports(ledger, 10)

	_, err := exports.Request(t.Context(), &serviceports.RequestAIAuditExportRequest{
		Actor:  actorFor(tenantInfo),
		Format: aiaudit.ExportFormatJSON,
		From:   testNow - 1000,
		To:     testNow,
	})
	require.Error(t, err)
	assert.Empty(t, repo.exports)
}
