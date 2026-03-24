package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/gtc/internal/core/domain"
	"github.com/emoss08/gtc/internal/core/ports"
	"go.uber.org/zap"
)

type fakeTailReader struct {
	startLSN     string
	currentLSN   string
	handler      ports.TransactionHandler
	transactions []domain.TransactionRecords
	advanceCalls []string
	advanceErr   error
}

func (f *fakeTailReader) Start(ctx context.Context, startLSN string, handler ports.TransactionHandler) error {
	f.startLSN = startLSN
	f.currentLSN = startLSN
	f.handler = handler
	for _, tx := range f.transactions {
		if err := handler(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeTailReader) Stop(ctx context.Context) error { return nil }
func (f *fakeTailReader) AdvanceLSN(lsn string) error {
	f.advanceCalls = append(f.advanceCalls, lsn)
	if f.advanceErr != nil {
		return f.advanceErr
	}
	f.currentLSN = lsn
	return nil
}
func (f *fakeTailReader) CurrentLSN() string { return f.currentLSN }

type fakeSnapshotReader struct {
	currentLSN string
	runCalls   int
	record     domain.SourceRecord
}

func (f *fakeSnapshotReader) CurrentLSN(ctx context.Context) (string, error) {
	return f.currentLSN, nil
}

func (f *fakeSnapshotReader) Run(
	ctx context.Context,
	bindings []domain.SnapshotBinding,
	handler ports.RecordHandler,
) error {
	return f.run(ctx, bindings, handler)
}

func (f *fakeSnapshotReader) Backfill(
	ctx context.Context,
	bindings []domain.SnapshotBinding,
	handler ports.RecordHandler,
) error {
	return f.run(ctx, bindings, handler)
}

func (f *fakeSnapshotReader) run(
	ctx context.Context,
	bindings []domain.SnapshotBinding,
	handler ports.RecordHandler,
) error {
	f.runCalls++
	record := f.record
	if record.NewData == nil {
		record = domain.SourceRecord{
			Operation: domain.OperationSnapshot,
			NewData: map[string]any{
				"id":               "snapshot-1",
				"organization_id":  "org_1",
				"business_unit_id": "bu_1",
			},
			Metadata: domain.RecordMetadata{Snapshot: true},
		}
	}

	for _, binding := range bindings {
		record.Schema = binding.Schema
		record.Table = binding.Table
		if err := handler(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeSnapshotReader) HealthCheck(ctx context.Context) error { return nil }

type fakeCheckpointStore struct {
	bootstrapLSN string
	walLSN       string
	saveCalls    []string
	saveErr      error
	onSaveWALLSN func(lsn string)
}

func (f *fakeCheckpointStore) Ensure(ctx context.Context) error      { return nil }
func (f *fakeCheckpointStore) HealthCheck(ctx context.Context) error { return nil }
func (f *fakeCheckpointStore) LoadBootstrapLSN(ctx context.Context) (string, error) {
	return f.bootstrapLSN, nil
}
func (f *fakeCheckpointStore) SaveBootstrapLSN(ctx context.Context, lsn string) error {
	f.bootstrapLSN = lsn
	return nil
}
func (f *fakeCheckpointStore) LoadWALLSN(ctx context.Context) (string, error) { return f.walLSN, nil }
func (f *fakeCheckpointStore) SaveWALLSN(ctx context.Context, lsn string) error {
	if f.onSaveWALLSN != nil {
		f.onSaveWALLSN(lsn)
	}
	if f.saveErr != nil {
		return f.saveErr
	}
	f.walLSN = lsn
	f.saveCalls = append(f.saveCalls, lsn)
	return nil
}
func (f *fakeCheckpointStore) LoadSnapshotProgress(
	ctx context.Context,
	tableName string,
) (ports.SnapshotProgress, error) {
	return ports.SnapshotProgress{TableName: tableName}, nil
}
func (f *fakeCheckpointStore) SaveSnapshotProgress(
	ctx context.Context,
	progress ports.SnapshotProgress,
) error {
	return nil
}

type fakeMetadataStore struct {
	metadata map[string]domain.TableMetadata
}

func (f *fakeMetadataStore) LoadTableMetadata(
	ctx context.Context,
	schema string,
	table string,
) (domain.TableMetadata, error) {
	metadata, ok := f.metadata[schema+"."+table]
	if !ok {
		return domain.TableMetadata{}, errors.New("missing metadata")
	}
	return metadata, nil
}

type fakeSink struct {
	mu       sync.Mutex
	kind     domain.DestinationKind
	writes   int
	delay    time.Duration
	failures int
	writeLog []string
}

func (f *fakeSink) Kind() domain.DestinationKind         { return f.kind }
func (f *fakeSink) Name() string                         { return string(f.kind) }
func (f *fakeSink) Initialize(ctx context.Context) error { return nil }
func (f *fakeSink) Write(ctx context.Context, projection domain.Projection, record domain.SourceRecord) error {
	if f.delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(f.delay):
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failures > 0 {
		f.failures--
		return errors.New("sink failed")
	}

	f.writes++
	f.writeLog = append(f.writeLog, projection.Name+":"+record.FullTableName()+":"+recordID(record))
	return nil
}
func (f *fakeSink) HealthCheck(ctx context.Context) error { return nil }
func (f *fakeSink) Shutdown(ctx context.Context) error    { return nil }

type fakeDLQ struct {
	mu      sync.Mutex
	records []domain.DeadLetterRecord
}

func (f *fakeDLQ) Write(ctx context.Context, entry domain.DeadLetterRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.records = append(f.records, entry)
	return nil
}

func baseRuntimeParams(
	tailer *fakeTailReader,
	snapshotter *fakeSnapshotReader,
	checkpoints *fakeCheckpointStore,
	metadata *fakeMetadataStore,
	sinks ...ports.Sink,
) RuntimeParams {
	return RuntimeParams{
		TailReader:      tailer,
		Snapshotter:     snapshotter,
		Checkpoints:     checkpoints,
		MetadataStore:   metadata,
		ProcessTimeout:  time.Second,
		WorkerCount:     2,
		WorkerQueueSize: 8,
		RetryMax:        3,
		RetryBackoff:    time.Millisecond,
		Sinks:           sinks,
		Logger:          zap.NewNop(),
	}
}

func recordID(record domain.SourceRecord) string {
	if record.NewData != nil {
		if value, ok := record.NewData["id"]; ok {
			return fmt.Sprint(value)
		}
	}
	if record.OldData != nil {
		if value, ok := record.OldData["id"]; ok {
			return fmt.Sprint(value)
		}
	}
	return ""
}

func TestRuntimeBootstrapsSnapshotsAndTailing(t *testing.T) {
	t.Parallel()

	tailer := &fakeTailReader{}
	snapshotter := &fakeSnapshotReader{currentLSN: "0/16B6C50"}
	checkpoints := &fakeCheckpointStore{}
	metadataStore := &fakeMetadataStore{metadata: map[string]domain.TableMetadata{
		"public.shipments": {
			Schema:      "public",
			Table:       "shipments",
			PrimaryKeys: []string{"id", "organization_id", "business_unit_id"},
		},
	}}
	meiliSink := &fakeSink{kind: domain.DestinationMeilisearch}
	redisSink := &fakeSink{kind: domain.DestinationRedisJSON}

	params := baseRuntimeParams(tailer, snapshotter, checkpoints, metadataStore, meiliSink, redisSink)
	params.Projections = []domain.Projection{
		{
			Name:         "shipment-search",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:  domain.DestinationMeilisearch,
				Index: "shipments",
			},
		},
		{
			Name:         "shipment-cache",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:        domain.DestinationRedisJSON,
				KeyTemplate: "cache:shipments",
			},
		},
	}

	runtime, err := NewRuntime(params)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("runtime.Start returned error: %v", err)
	}

	if tailer.startLSN != "0/16B6C50" {
		t.Fatalf("expected tail start lsn to equal bootstrap lsn, got %s", tailer.startLSN)
	}
	if checkpoints.bootstrapLSN != "0/16B6C50" {
		t.Fatalf("expected bootstrap lsn to be saved, got %s", checkpoints.bootstrapLSN)
	}
	if checkpoints.walLSN != "0/16B6C50" {
		t.Fatalf("expected initial wal lsn to be saved, got %s", checkpoints.walLSN)
	}
	if snapshotter.runCalls != 1 {
		t.Fatalf("expected one snapshot run, got %d", snapshotter.runCalls)
	}
	if meiliSink.writes == 0 || redisSink.writes == 0 {
		t.Fatalf("expected snapshot records to be written to both sinks")
	}
	if got := runtime.projections[0].PrimaryKeys; len(got) != 3 {
		t.Fatalf("expected runtime to resolve composite keys, got %v", got)
	}
}

func TestRuntimeUpdatesWALCheckpointAfterTransaction(t *testing.T) {
	t.Parallel()

	tailer := &fakeTailReader{
		transactions: []domain.TransactionRecords{
			{
				LSN:       "0/18",
				CommitLSN: "0/20",
				Records: []domain.SourceRecord{
					{
						Operation: domain.OperationUpdate,
						Schema:    "public",
						Table:     "shipments",
						NewData:   map[string]any{"id": "shp_1"},
					},
				},
			},
		},
	}
	snapshotter := &fakeSnapshotReader{currentLSN: "0/10"}
	checkpoints := &fakeCheckpointStore{bootstrapLSN: "0/10", walLSN: "0/10"}
	metadataStore := &fakeMetadataStore{metadata: map[string]domain.TableMetadata{
		"public.shipments": {Schema: "public", Table: "shipments", PrimaryKeys: []string{"id"}},
	}}
	meiliSink := &fakeSink{kind: domain.DestinationMeilisearch}

	params := baseRuntimeParams(tailer, snapshotter, checkpoints, metadataStore, meiliSink)
	params.Projections = []domain.Projection{
		{
			Name:         "shipment-search",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:  domain.DestinationMeilisearch,
				Index: "shipments",
			},
		},
	}

	runtime, err := NewRuntime(params)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("runtime.Start returned error: %v", err)
	}

	if checkpoints.walLSN != "0/20" {
		t.Fatalf("expected wal checkpoint to advance to commit lsn, got %s", checkpoints.walLSN)
	}
	if tailer.CurrentLSN() != "0/20" {
		t.Fatalf("expected tailer lsn to advance to commit lsn, got %s", tailer.CurrentLSN())
	}
}

func TestRuntimeSuppressesIgnoredOnlyUpdates(t *testing.T) {
	t.Parallel()

	tailer := &fakeTailReader{
		transactions: []domain.TransactionRecords{
			{
				CommitLSN: "0/20",
				Records: []domain.SourceRecord{
					{
						Operation: domain.OperationUpdate,
						Schema:    "public",
						Table:     "shipments",
						OldData: map[string]any{
							"id":            "shp_1",
							"updated_at":    "1",
							"version":       1,
							"search_vector": "a",
						},
						NewData: map[string]any{
							"id":            "shp_1",
							"updated_at":    "2",
							"version":       2,
							"search_vector": "b",
						},
					},
				},
			},
		},
	}
	snapshotter := &fakeSnapshotReader{currentLSN: "0/10"}
	checkpoints := &fakeCheckpointStore{bootstrapLSN: "0/10", walLSN: "0/10"}
	metadataStore := &fakeMetadataStore{metadata: map[string]domain.TableMetadata{
		"public.shipments": {Schema: "public", Table: "shipments", PrimaryKeys: []string{"id"}},
	}}
	streamSink := &fakeSink{kind: domain.DestinationRedisStream}

	params := baseRuntimeParams(tailer, snapshotter, checkpoints, metadataStore, streamSink)
	params.Projections = []domain.Projection{
		{
			Name:           "shipment-stream",
			SourceSchema:   "public",
			SourceTable:    "shipments",
			IgnoredUpdates: []string{"updated_at", "version", "search_vector"},
			Destination: domain.Destination{
				Kind:   domain.DestinationRedisStream,
				Stream: "cdc:shipments",
			},
		},
	}

	runtime, err := NewRuntime(params)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("runtime.Start returned error: %v", err)
	}

	if streamSink.writes != 1 {
		t.Fatalf("expected only snapshot write, got %d", streamSink.writes)
	}
}

func TestRuntimeEnforcesProcessTimeout(t *testing.T) {
	t.Parallel()

	tailer := &fakeTailReader{}
	snapshotter := &fakeSnapshotReader{currentLSN: "0/10"}
	checkpoints := &fakeCheckpointStore{bootstrapLSN: "0/10", walLSN: "0/10"}
	metadataStore := &fakeMetadataStore{metadata: map[string]domain.TableMetadata{
		"public.shipments": {Schema: "public", Table: "shipments", PrimaryKeys: []string{"id"}},
	}}
	slowSink := &fakeSink{kind: domain.DestinationMeilisearch, delay: 50 * time.Millisecond}

	params := baseRuntimeParams(tailer, snapshotter, checkpoints, metadataStore, slowSink)
	params.ProcessTimeout = 10 * time.Millisecond
	params.Projections = []domain.Projection{
		{
			Name:         "shipment-search",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:  domain.DestinationMeilisearch,
				Index: "shipments",
			},
		},
	}

	runtime, err := NewRuntime(params)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	if err := runtime.Start(context.Background()); err == nil {
		t.Fatalf("expected timeout during snapshot initialization")
	}
}

func TestRuntimeRetriesTransientSinkFailures(t *testing.T) {
	t.Parallel()

	tailer := &fakeTailReader{}
	snapshotter := &fakeSnapshotReader{currentLSN: "0/10"}
	checkpoints := &fakeCheckpointStore{}
	metadataStore := &fakeMetadataStore{metadata: map[string]domain.TableMetadata{
		"public.shipments": {Schema: "public", Table: "shipments", PrimaryKeys: []string{"id"}},
	}}
	flakySink := &fakeSink{kind: domain.DestinationMeilisearch, failures: 2}

	params := baseRuntimeParams(tailer, snapshotter, checkpoints, metadataStore, flakySink)
	params.Projections = []domain.Projection{
		{
			Name:         "shipment-search",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:  domain.DestinationMeilisearch,
				Index: "shipments",
			},
		},
	}

	runtime, err := NewRuntime(params)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("runtime.Start returned error: %v", err)
	}

	if flakySink.writes != 1 {
		t.Fatalf("expected retry to eventually succeed, got %d writes", flakySink.writes)
	}
}

func TestRuntimeWritesDeadLetterOnExhaustedRetries(t *testing.T) {
	t.Parallel()

	tailer := &fakeTailReader{}
	snapshotter := &fakeSnapshotReader{currentLSN: "0/10"}
	checkpoints := &fakeCheckpointStore{}
	metadataStore := &fakeMetadataStore{metadata: map[string]domain.TableMetadata{
		"public.shipments": {Schema: "public", Table: "shipments", PrimaryKeys: []string{"id"}},
	}}
	failingSink := &fakeSink{kind: domain.DestinationMeilisearch, failures: 5}
	dlq := &fakeDLQ{}

	params := baseRuntimeParams(tailer, snapshotter, checkpoints, metadataStore, failingSink)
	params.DeadLetter = dlq
	params.RetryMax = 2
	params.Projections = []domain.Projection{
		{
			Name:         "shipment-search",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:  domain.DestinationMeilisearch,
				Index: "shipments",
			},
		},
	}

	runtime, err := NewRuntime(params)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	if err := runtime.Start(context.Background()); err == nil {
		t.Fatalf("expected runtime to fail after exhausted retries")
	}

	if len(dlq.records) != 1 {
		t.Fatalf("expected one dead-letter entry, got %d", len(dlq.records))
	}
}

func TestRuntimePreservesCheckpointOrderAcrossTransactions(t *testing.T) {
	t.Parallel()

	tailer := &fakeTailReader{
		transactions: []domain.TransactionRecords{
			{
				CommitLSN: "0/20",
				Records: []domain.SourceRecord{
					{
						Operation: domain.OperationUpdate,
						Schema:    "public",
						Table:     "shipments",
						NewData:   map[string]any{"id": "shp_1"},
					},
				},
			},
			{
				CommitLSN: "0/30",
				Records: []domain.SourceRecord{
					{
						Operation: domain.OperationUpdate,
						Schema:    "public",
						Table:     "shipments",
						NewData:   map[string]any{"id": "shp_2"},
					},
				},
			},
		},
	}
	snapshotter := &fakeSnapshotReader{currentLSN: "0/10"}
	checkpoints := &fakeCheckpointStore{bootstrapLSN: "0/10", walLSN: "0/10"}
	metadataStore := &fakeMetadataStore{metadata: map[string]domain.TableMetadata{
		"public.shipments": {Schema: "public", Table: "shipments", PrimaryKeys: []string{"id"}},
	}}
	meiliSink := &fakeSink{kind: domain.DestinationMeilisearch}

	params := baseRuntimeParams(tailer, snapshotter, checkpoints, metadataStore, meiliSink)
	params.Projections = []domain.Projection{
		{
			Name:         "shipment-search",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:  domain.DestinationMeilisearch,
				Index: "shipments",
			},
		},
	}

	runtime, err := NewRuntime(params)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("runtime.Start returned error: %v", err)
	}

	if len(checkpoints.saveCalls) != 2 {
		t.Fatalf("expected two transactional checkpoint saves, got %v", checkpoints.saveCalls)
	}
	if checkpoints.saveCalls[0] != "0/20" || checkpoints.saveCalls[1] != "0/30" {
		t.Fatalf("unexpected checkpoint order: %v", checkpoints.saveCalls)
	}
	if got := tailer.advanceCalls; len(got) != 2 || got[0] != "0/20" || got[1] != "0/30" {
		t.Fatalf("unexpected tailer lsn advances: %v", got)
	}
}

func TestRuntimeDoesNotAdvanceTailLSNDuringCheckpointSave(t *testing.T) {
	t.Parallel()

	tailer := &fakeTailReader{
		transactions: []domain.TransactionRecords{
			{
				CommitLSN: "0/20",
				Records: []domain.SourceRecord{
					{
						Operation: domain.OperationUpdate,
						Schema:    "public",
						Table:     "shipments",
						NewData:   map[string]any{"id": "shp_1"},
					},
				},
			},
		},
	}
	snapshotter := &fakeSnapshotReader{currentLSN: "0/10"}
	checkpoints := &fakeCheckpointStore{bootstrapLSN: "0/10", walLSN: "0/10"}
	checkpoints.onSaveWALLSN = func(lsn string) {
		if current := tailer.CurrentLSN(); current != "0/10" {
			t.Fatalf("expected tailer lsn to remain at prior checkpoint during save, got %s", current)
		}
		if lsn != "0/20" {
			t.Fatalf("expected checkpoint save lsn 0/20, got %s", lsn)
		}
	}
	metadataStore := &fakeMetadataStore{metadata: map[string]domain.TableMetadata{
		"public.shipments": {Schema: "public", Table: "shipments", PrimaryKeys: []string{"id"}},
	}}
	meiliSink := &fakeSink{kind: domain.DestinationMeilisearch}

	params := baseRuntimeParams(tailer, snapshotter, checkpoints, metadataStore, meiliSink)
	params.Projections = []domain.Projection{
		{
			Name:         "shipment-search",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:  domain.DestinationMeilisearch,
				Index: "shipments",
			},
		},
	}

	runtime, err := NewRuntime(params)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("runtime.Start returned error: %v", err)
	}
}

func TestRuntimeReprocessesTransactionAfterCheckpointFailureOnRestart(t *testing.T) {
	t.Parallel()

	checkpoints := &fakeCheckpointStore{bootstrapLSN: "0/10", walLSN: "0/10"}
	metadataStore := &fakeMetadataStore{metadata: map[string]domain.TableMetadata{
		"public.shipments": {Schema: "public", Table: "shipments", PrimaryKeys: []string{"id"}},
	}}
	meiliSink := &fakeSink{kind: domain.DestinationMeilisearch}
	transaction := domain.TransactionRecords{
		CommitLSN: "0/20",
		Records: []domain.SourceRecord{
			{
				Operation: domain.OperationUpdate,
				Schema:    "public",
				Table:     "shipments",
				NewData:   map[string]any{"id": "shp_1"},
			},
		},
	}

	firstTailer := &fakeTailReader{transactions: []domain.TransactionRecords{transaction}}
	firstSnapshotter := &fakeSnapshotReader{currentLSN: "0/10"}
	saveAttempts := 0
	checkpoints.onSaveWALLSN = func(lsn string) {
		saveAttempts++
		if saveAttempts == 1 {
			checkpoints.saveErr = errors.New("checkpoint failed")
			return
		}
		checkpoints.saveErr = nil
	}

	firstParams := baseRuntimeParams(firstTailer, firstSnapshotter, checkpoints, metadataStore, meiliSink)
	firstParams.Projections = []domain.Projection{
		{
			Name:         "shipment-search",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:  domain.DestinationMeilisearch,
				Index: "shipments",
			},
		},
	}

	firstRuntime, err := NewRuntime(firstParams)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	if err := firstRuntime.Start(context.Background()); err == nil {
		t.Fatalf("expected first runtime start to fail when checkpoint save fails")
	}

	if checkpoints.walLSN != "0/10" {
		t.Fatalf("expected checkpoint to remain at 0/10 after failure, got %s", checkpoints.walLSN)
	}
	if meiliSink.writes != 2 {
		t.Fatalf("expected snapshot and transaction write before crash, got %d", meiliSink.writes)
	}

	secondTailer := &fakeTailReader{transactions: []domain.TransactionRecords{transaction}}
	secondSnapshotter := &fakeSnapshotReader{currentLSN: "0/10"}
	secondParams := baseRuntimeParams(secondTailer, secondSnapshotter, checkpoints, metadataStore, meiliSink)
	secondParams.Projections = firstParams.Projections

	secondRuntime, err := NewRuntime(secondParams)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	startLSN, err := secondRuntime.determineStartLSN(context.Background())
	if err != nil {
		t.Fatalf("determineStartLSN returned error: %v", err)
	}
	if startLSN != "0/10" {
		t.Fatalf("expected restart start lsn to remain at 0/10, got %s", startLSN)
	}

	if err := secondRuntime.Start(context.Background()); err != nil {
		t.Fatalf("expected restarted runtime to replay transaction successfully, got %v", err)
	}

	if secondTailer.startLSN != "0/10" {
		t.Fatalf("expected restarted tailer to start from 0/10, got %s", secondTailer.startLSN)
	}
	if checkpoints.walLSN != "0/20" {
		t.Fatalf("expected replayed transaction to advance checkpoint to 0/20, got %s", checkpoints.walLSN)
	}
	txWrites := 0
	for _, entry := range meiliSink.writeLog {
		if entry == "shipment-search:public.shipments:shp_1" {
			txWrites++
		}
	}
	if txWrites != 2 {
		t.Fatalf("expected transaction at 0/20 to be replayed twice across restart, got %d writes (%v)", txWrites, meiliSink.writeLog)
	}
}

func TestRuntimeDoesNotAdvanceTailLSNWhenCheckpointSaveFails(t *testing.T) {
	t.Parallel()

	tailer := &fakeTailReader{
		transactions: []domain.TransactionRecords{
			{
				CommitLSN: "0/20",
				Records: []domain.SourceRecord{
					{
						Operation: domain.OperationUpdate,
						Schema:    "public",
						Table:     "shipments",
						NewData:   map[string]any{"id": "shp_1"},
					},
				},
			},
		},
	}
	snapshotter := &fakeSnapshotReader{currentLSN: "0/10"}
	checkpoints := &fakeCheckpointStore{
		bootstrapLSN: "0/10",
		walLSN:       "0/10",
		saveErr:      errors.New("checkpoint failed"),
	}
	metadataStore := &fakeMetadataStore{metadata: map[string]domain.TableMetadata{
		"public.shipments": {Schema: "public", Table: "shipments", PrimaryKeys: []string{"id"}},
	}}
	meiliSink := &fakeSink{kind: domain.DestinationMeilisearch}

	params := baseRuntimeParams(tailer, snapshotter, checkpoints, metadataStore, meiliSink)
	params.Projections = []domain.Projection{
		{
			Name:         "shipment-search",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:  domain.DestinationMeilisearch,
				Index: "shipments",
			},
		},
	}

	runtime, err := NewRuntime(params)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	if err := runtime.Start(context.Background()); err == nil {
		t.Fatalf("expected runtime.Start to fail when checkpoint save fails")
	}

	if got := len(tailer.advanceCalls); got != 0 {
		t.Fatalf("expected no tailer lsn advances, got %v", tailer.advanceCalls)
	}
	if tailer.CurrentLSN() != "0/10" {
		t.Fatalf("expected tailer lsn to remain at prior checkpoint, got %s", tailer.CurrentLSN())
	}
}

func TestRuntimeFailsWhenTailLSNAdvanceFails(t *testing.T) {
	t.Parallel()

	tailer := &fakeTailReader{
		advanceErr: errors.New("advance failed"),
		transactions: []domain.TransactionRecords{
			{
				CommitLSN: "0/20",
				Records: []domain.SourceRecord{
					{
						Operation: domain.OperationUpdate,
						Schema:    "public",
						Table:     "shipments",
						NewData:   map[string]any{"id": "shp_1"},
					},
				},
			},
		},
	}
	snapshotter := &fakeSnapshotReader{currentLSN: "0/10"}
	checkpoints := &fakeCheckpointStore{bootstrapLSN: "0/10", walLSN: "0/10"}
	metadataStore := &fakeMetadataStore{metadata: map[string]domain.TableMetadata{
		"public.shipments": {Schema: "public", Table: "shipments", PrimaryKeys: []string{"id"}},
	}}
	meiliSink := &fakeSink{kind: domain.DestinationMeilisearch}

	params := baseRuntimeParams(tailer, snapshotter, checkpoints, metadataStore, meiliSink)
	params.Projections = []domain.Projection{
		{
			Name:         "shipment-search",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:  domain.DestinationMeilisearch,
				Index: "shipments",
			},
		},
	}

	runtime, err := NewRuntime(params)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	if err := runtime.Start(context.Background()); err == nil {
		t.Fatalf("expected runtime.Start to fail when tail lsn advance fails")
	}

	if checkpoints.walLSN != "0/20" {
		t.Fatalf("expected checkpoint to be saved before advance failure, got %s", checkpoints.walLSN)
	}
	if tailer.CurrentLSN() != "0/10" {
		t.Fatalf("expected tailer lsn to remain at prior checkpoint, got %s", tailer.CurrentLSN())
	}
}

func TestRuntimeBackfillFiltersProjection(t *testing.T) {
	t.Parallel()

	tailer := &fakeTailReader{}
	snapshotter := &fakeSnapshotReader{currentLSN: "0/10"}
	checkpoints := &fakeCheckpointStore{}
	metadataStore := &fakeMetadataStore{metadata: map[string]domain.TableMetadata{
		"public.shipments": {Schema: "public", Table: "shipments", PrimaryKeys: []string{"id"}},
	}}
	meiliSink := &fakeSink{kind: domain.DestinationMeilisearch}
	redisSink := &fakeSink{kind: domain.DestinationRedisJSON}

	params := baseRuntimeParams(tailer, snapshotter, checkpoints, metadataStore, meiliSink, redisSink)
	params.Projections = []domain.Projection{
		{
			Name:         "shipment-search",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:  domain.DestinationMeilisearch,
				Index: "shipments",
			},
		},
		{
			Name:         "shipment-cache",
			SourceSchema: "public",
			SourceTable:  "shipments",
			Destination: domain.Destination{
				Kind:        domain.DestinationRedisJSON,
				KeyTemplate: "cache:shipments",
			},
		},
	}

	runtime, err := NewRuntime(params)
	if err != nil {
		t.Fatalf("NewRuntime returned error: %v", err)
	}

	if err := runtime.Backfill(context.Background(), []string{"shipment-search"}, nil); err != nil {
		t.Fatalf("Backfill returned error: %v", err)
	}

	if meiliSink.writes != 1 {
		t.Fatalf("expected one search write, got %d", meiliSink.writes)
	}
	if redisSink.writes != 0 {
		t.Fatalf("expected no cache writes, got %d", redisSink.writes)
	}
}
