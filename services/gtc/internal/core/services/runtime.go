package services

import (
	"context"
	"fmt"
	"hash/fnv"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emoss08/gtc/internal/core/domain"
	"github.com/emoss08/gtc/internal/core/ports"
	"github.com/emoss08/gtc/internal/infrastructure/metrics"
	"github.com/jackc/pglogrepl"
	"go.uber.org/zap"
)

type tailHealthProvider interface {
	HealthStatuses() map[string]bool
}

type Runtime struct {
	tailReader      ports.TailReader
	snapshotter     ports.SnapshotReader
	checkpoints     ports.CheckpointStore
	metadataStore   ports.MetadataStore
	dlqWriter       ports.DeadLetterWriter
	projections     []domain.Projection
	sinks           map[domain.DestinationKind]ports.Sink
	processTimeout  time.Duration
	workerCount     int
	workerQueueSize int
	retryMax        int
	retryBackoff    time.Duration
	logger          *zap.Logger
	ready           atomic.Bool
	prepareMu       sync.Mutex
	prepared        bool
	statusMu        sync.RWMutex
	statuses        map[string]bool
	processor       *transactionProcessor
}

type RuntimeParams struct {
	TailReader      ports.TailReader
	Snapshotter     ports.SnapshotReader
	Checkpoints     ports.CheckpointStore
	MetadataStore   ports.MetadataStore
	DeadLetter      ports.DeadLetterWriter
	Projections     []domain.Projection
	Sinks           []ports.Sink
	ProcessTimeout  time.Duration
	WorkerCount     int
	WorkerQueueSize int
	RetryMax        int
	RetryBackoff    time.Duration
	Logger          *zap.Logger
}

func NewRuntime(params RuntimeParams) (*Runtime, error) {
	sinks := make(map[domain.DestinationKind]ports.Sink, len(params.Sinks))
	for _, sink := range params.Sinks {
		if _, exists := sinks[sink.Kind()]; exists {
			return nil, fmt.Errorf("duplicate sink kind %q", sink.Kind())
		}
		sinks[sink.Kind()] = sink
	}

	if len(params.Projections) == 0 {
		return nil, fmt.Errorf("no projections configured")
	}
	if params.MetadataStore == nil {
		return nil, fmt.Errorf("metadata store is required")
	}
	if params.ProcessTimeout <= 0 {
		return nil, fmt.Errorf("process timeout must be greater than zero")
	}
	if params.WorkerCount <= 0 {
		return nil, fmt.Errorf("worker count must be greater than zero")
	}
	if params.WorkerQueueSize <= 0 {
		return nil, fmt.Errorf("worker queue size must be greater than zero")
	}
	if params.RetryMax <= 0 {
		return nil, fmt.Errorf("retry max must be greater than zero")
	}
	if params.RetryBackoff <= 0 {
		return nil, fmt.Errorf("retry backoff must be greater than zero")
	}

	return &Runtime{
		tailReader:      params.TailReader,
		snapshotter:     params.Snapshotter,
		checkpoints:     params.Checkpoints,
		metadataStore:   params.MetadataStore,
		dlqWriter:       params.DeadLetter,
		projections:     slices.Clone(params.Projections),
		sinks:           sinks,
		processTimeout:  params.ProcessTimeout,
		workerCount:     params.WorkerCount,
		workerQueueSize: params.WorkerQueueSize,
		retryMax:        params.RetryMax,
		retryBackoff:    params.RetryBackoff,
		logger:          params.Logger.Named("runtime"),
		statuses:        make(map[string]bool),
	}, nil
}

func (r *Runtime) Start(ctx context.Context) error {
	r.ready.Store(false)

	if err := r.prepare(ctx, true); err != nil {
		return err
	}

	startLSN, err := r.determineStartLSN(ctx)
	if err != nil {
		return err
	}

	if err := r.runSnapshots(ctx); err != nil {
		return err
	}

	if checkpointLSN, loadErr := r.checkpoints.LoadWALLSN(ctx); loadErr == nil && checkpointLSN == "" {
		if err := r.checkpoints.SaveWALLSN(ctx, startLSN); err != nil {
			return fmt.Errorf("save initial wal checkpoint: %w", err)
		}
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	processor := newTransactionProcessor(runCtx, r)
	r.processor = processor

	processor.start()

	r.ready.Store(true)
	r.logger.Info("starting WAL tail", zap.String("start_lsn", startLSN))
	err = r.tailReader.Start(runCtx, startLSN, r.handleTransaction)
	r.ready.Store(false)
	if err == nil {
		processor.drain()
	} else {
		processor.stop()
	}
	r.processor = nil

	if procErr := processor.err(); procErr != nil {
		return procErr
	}
	if err != nil {
		return err
	}
	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	r.ready.Store(false)

	if r.processor != nil {
		r.processor.stop()
	}

	var stopErr error
	if err := r.tailReader.Stop(ctx); err != nil {
		stopErr = err
	}

	for _, sink := range r.sinks {
		if err := sink.Shutdown(ctx); err != nil && stopErr == nil {
			stopErr = err
		}
	}
	r.prepareMu.Lock()
	r.prepared = false
	r.prepareMu.Unlock()

	return stopErr
}

func (r *Runtime) Validate(ctx context.Context) error {
	return r.prepare(ctx, false)
}

func (r *Runtime) Backfill(ctx context.Context, projectionNames []string, tableNames []string) error {
	if err := r.prepare(ctx, false); err != nil {
		return err
	}

	projections, err := r.filterProjections(projectionNames, tableNames)
	if err != nil {
		return err
	}

	return r.snapshotter.Backfill(ctx, uniqueBindings(projections), func(runCtx context.Context, record domain.SourceRecord) error {
		return r.handleRecordWithProjections(runCtx, projections, record)
	})
}

func (r *Runtime) ReplayDeadLetters(ctx context.Context, entries []domain.DeadLetterRecord) error {
	if err := r.prepare(ctx, false); err != nil {
		return err
	}

	for _, entry := range entries {
		projection, err := r.projectionByName(entry.Projection)
		if err != nil {
			return err
		}
		if err := r.writeProjection(ctx, projection, entry.Record); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runtime) IsReady() bool {
	return r.ready.Load()
}

func (r *Runtime) SinkStatuses() map[string]bool {
	r.statusMu.RLock()
	defer r.statusMu.RUnlock()

	statuses := make(map[string]bool, len(r.statuses))
	for name, healthy := range r.statuses {
		statuses[name] = healthy
	}

	return statuses
}

func (r *Runtime) HealthCheck(ctx context.Context) map[string]bool {
	statuses := make(map[string]bool, len(r.sinks)+2)

	statuses["postgres_snapshot"] = r.snapshotter.HealthCheck(ctx) == nil
	statuses["checkpoints"] = r.checkpoints.HealthCheck(ctx) == nil

	if provider, ok := r.tailReader.(tailHealthProvider); ok {
		for name, healthy := range provider.HealthStatuses() {
			statuses[name] = healthy
		}
	}

	for _, sink := range r.sinks {
		healthy := sink.HealthCheck(ctx) == nil
		statuses[sink.Name()] = healthy
		r.setStatus(sink.Name(), healthy)
	}

	return statuses
}

func (r *Runtime) determineStartLSN(ctx context.Context) (string, error) {
	walLSN, err := r.checkpoints.LoadWALLSN(ctx)
	if err != nil {
		return "", fmt.Errorf("load wal checkpoint: %w", err)
	}
	if walLSN != "" {
		return walLSN, nil
	}

	bootstrapLSN, err := r.checkpoints.LoadBootstrapLSN(ctx)
	if err != nil {
		return "", fmt.Errorf("load bootstrap lsn: %w", err)
	}
	if bootstrapLSN != "" {
		return bootstrapLSN, nil
	}

	bootstrapLSN, err = r.snapshotter.CurrentLSN(ctx)
	if err != nil {
		return "", fmt.Errorf("capture bootstrap lsn: %w", err)
	}

	if err := r.checkpoints.SaveBootstrapLSN(ctx, bootstrapLSN); err != nil {
		return "", fmt.Errorf("save bootstrap lsn: %w", err)
	}

	return bootstrapLSN, nil
}

func (r *Runtime) runSnapshots(ctx context.Context) error {
	bindings := uniqueBindings(r.projections)
	if len(bindings) == 0 {
		return nil
	}

	r.logger.Info("running snapshot phase", zap.Int("table_count", len(bindings)))
	if err := r.snapshotter.Run(ctx, bindings, r.handleRecord); err != nil {
		return fmt.Errorf("run snapshots: %w", err)
	}

	return nil
}

func (r *Runtime) handleTransaction(ctx context.Context, tx domain.TransactionRecords) error {
	if r.processor == nil {
		return fmt.Errorf("transaction processor is not running")
	}

	if err := r.processor.err(); err != nil {
		return err
	}

	return r.processor.enqueue(ctx, tx)
}

func (r *Runtime) handleRecord(ctx context.Context, record domain.SourceRecord) error {
	return r.handleRecordWithProjections(ctx, r.matchingProjections(record.FullTableName()), record)
}

func (r *Runtime) handleRecordWithProjections(
	ctx context.Context,
	projections []domain.Projection,
	record domain.SourceRecord,
) error {
	if len(projections) == 0 {
		r.logger.Debug("no matching projections",
			zap.String("table", record.FullTableName()),
			zap.String("operation", string(record.Operation)),
		)
		return nil
	}

	for _, projection := range projections {
		if shouldSuppressRecord(projection, record) {
			changedFields := domain.ChangedFields(record.OldData, record.NewData)
			r.logger.Debug("suppressed record",
				zap.String("projection", projection.Name),
				zap.String("table", record.FullTableName()),
				zap.String("sink", string(projection.Destination.Kind)),
				zap.Strings("changedFields", changedFields),
				zap.Strings("ignoredUpdates", projection.IgnoredUpdates),
				zap.Bool("hasOldData", record.OldData != nil),
				zap.Bool("hasNewData", record.NewData != nil),
			)
			continue
		}

		r.logger.Debug("writing projection",
			zap.String("projection", projection.Name),
			zap.String("table", record.FullTableName()),
			zap.String("sink", string(projection.Destination.Kind)),
		)

		if err := r.writeProjection(ctx, projection, record); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runtime) writeProjection(ctx context.Context, projection domain.Projection, record domain.SourceRecord) error {
	sink, ok := r.sinks[projection.Destination.Kind]
	if !ok {
		return fmt.Errorf("projection %s uses unknown sink kind %q", projection.Name, projection.Destination.Kind)
	}

	var lastErr error
	for attempt := 1; attempt <= r.retryMax; attempt++ {
		writeCtx, cancel := context.WithTimeout(ctx, r.processTimeout)
		err := sink.Write(writeCtx, projection, record)
		cancel()
		if err == nil {
			r.setStatus(sink.Name(), true)
			return nil
		}

		lastErr = err
		r.setStatus(sink.Name(), false)
		r.logger.Warn("projection write failed",
			zap.String("projection", projection.Name),
			zap.String("sink", sink.Name()),
			zap.String("operation", record.Operation.String()),
			zap.String("table", record.FullTableName()),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", r.retryMax),
			zap.Error(err),
		)
		if attempt == r.retryMax {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.retryBackoff):
		}
	}

	if r.dlqWriter != nil {
		entry := domain.DeadLetterRecord{
			TransactionID: record.Metadata.TransactionID,
			CommitLSN:     record.Metadata.CommitLSN,
			Projection:    projection.Name,
			Error:         lastErr.Error(),
			Attempts:      r.retryMax,
			Record:        record,
			CreatedAt:     time.Now().UTC(),
		}
		if err := r.dlqWriter.Write(ctx, entry); err != nil {
			return fmt.Errorf("projection %s failed and dlq write failed: %w", projection.Name, err)
		}
		r.logger.Error("projection sent to dlq",
			zap.String("projection", projection.Name),
			zap.String("table", record.FullTableName()),
			zap.String("operation", record.Operation.String()),
			zap.String("commit_lsn", record.Metadata.CommitLSN),
			zap.Uint32("transaction_id", record.Metadata.TransactionID),
			zap.Error(lastErr),
		)
	}

	return fmt.Errorf("projection %s failed after %d attempts: %w", projection.Name, r.retryMax, lastErr)
}

func (r *Runtime) matchingProjections(fullTableName string) []domain.Projection {
	matches := make([]domain.Projection, 0, len(r.projections))
	for _, projection := range r.projections {
		if projection.FullTableName() == fullTableName {
			matches = append(matches, projection)
		}
	}

	return matches
}

func (r *Runtime) setStatus(name string, healthy bool) {
	r.statusMu.Lock()
	defer r.statusMu.Unlock()
	r.statuses[name] = healthy
}

func (r *Runtime) prepare(ctx context.Context, ensureCheckpoints bool) error {
	r.prepareMu.Lock()
	defer r.prepareMu.Unlock()
	if r.prepared {
		return nil
	}

	if err := r.resolveProjections(ctx); err != nil {
		return fmt.Errorf("resolve projections: %w", err)
	}

	if ensureCheckpoints {
		if err := r.checkpoints.Ensure(ctx); err != nil {
			return fmt.Errorf("ensure checkpoints: %w", err)
		}
	}

	for _, sink := range r.sinks {
		if err := sink.Initialize(ctx); err != nil {
			return fmt.Errorf("initialize %s: %w", sink.Name(), err)
		}
		r.setStatus(sink.Name(), true)
	}

	r.prepared = true
	return nil
}

func (r *Runtime) resolveProjections(ctx context.Context) error {
	resolved := make([]domain.Projection, 0, len(r.projections))
	metadataCache := make(map[string]domain.TableMetadata, len(r.projections))

	for _, projection := range r.projections {
		key := projection.FullTableName()
		metadata, ok := metadataCache[key]
		if !ok {
			var err error
			metadata, err = r.metadataStore.LoadTableMetadata(ctx, projection.SourceSchema, projection.SourceTable)
			if err != nil {
				return err
			}
			metadataCache[key] = metadata
		}

		if len(projection.PrimaryKeys) > 0 && !domain.EqualStringSlices(projection.PrimaryKeys, metadata.PrimaryKeys) {
			return fmt.Errorf(
				"projection %s primary keys %v do not match discovered keys %v",
				projection.Name,
				projection.PrimaryKeys,
				metadata.PrimaryKeys,
			)
		}

		projection.PrimaryKeys = slices.Clone(metadata.PrimaryKeys)
		resolved = append(resolved, projection)
	}

	r.projections = resolved
	return nil
}

func uniqueBindings(projections []domain.Projection) []domain.SnapshotBinding {
	seen := make(map[string]struct{}, len(projections))
	bindings := make([]domain.SnapshotBinding, 0, len(projections))

	for _, projection := range projections {
		key := projection.FullTableName()
		if _, exists := seen[key]; exists {
			continue
		}

		bindings = append(bindings, domain.SnapshotBinding{
			Schema:      projection.SourceSchema,
			Table:       projection.SourceTable,
			PrimaryKeys: slices.Clone(projection.PrimaryKeys),
		})
		seen[key] = struct{}{}
	}

	return bindings
}

func (r *Runtime) filterProjections(projectionNames []string, tableNames []string) ([]domain.Projection, error) {
	if len(projectionNames) == 0 && len(tableNames) == 0 {
		return slices.Clone(r.projections), nil
	}

	projectionSet := make(map[string]struct{}, len(projectionNames))
	for _, name := range projectionNames {
		projectionSet[name] = struct{}{}
	}
	tableSet := make(map[string]struct{}, len(tableNames))
	for _, name := range tableNames {
		tableSet[name] = struct{}{}
	}

	filtered := make([]domain.Projection, 0)
	for _, projection := range r.projections {
		if len(projectionSet) > 0 {
			if _, ok := projectionSet[projection.Name]; ok {
				filtered = append(filtered, projection)
				continue
			}
		}
		if len(tableSet) > 0 {
			if _, ok := tableSet[projection.FullTableName()]; ok {
				filtered = append(filtered, projection)
			}
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no projections matched the requested filters")
	}

	return filtered, nil
}

func (r *Runtime) projectionByName(name string) (domain.Projection, error) {
	for _, projection := range r.projections {
		if projection.Name == name {
			return projection, nil
		}
	}

	return domain.Projection{}, fmt.Errorf("unknown projection %q", name)
}

func shouldSuppressRecord(projection domain.Projection, record domain.SourceRecord) bool {
	if record.Operation != domain.OperationUpdate {
		return false
	}
	if len(projection.IgnoredUpdates) == 0 {
		return false
	}

	changedFields := domain.ChangedFields(record.OldData, record.NewData)
	if len(changedFields) == 0 {
		return true
	}

	ignored := make(map[string]struct{}, len(projection.IgnoredUpdates))
	for _, field := range projection.IgnoredUpdates {
		ignored[field] = struct{}{}
	}

	for _, field := range changedFields {
		if _, ok := ignored[field]; !ok {
			return false
		}
	}

	return true
}

type transactionProcessor struct {
	ctx       context.Context
	cancel    context.CancelFunc
	runtime   *Runtime
	input     chan txEnvelope
	results   chan partitionResult
	workers   []chan partitionJob
	errMu     sync.Mutex
	fatalErr  error
	wg        sync.WaitGroup
	seq       uint64
	closeOnce sync.Once
}

type txEnvelope struct {
	seq uint64
	tx  domain.TransactionRecords
}

type txState struct {
	tx      domain.TransactionRecords
	pending int
	err     error
	done    bool
}

type partitionJob struct {
	seq       uint64
	tx        domain.TransactionRecords
	partition string
	records   []domain.SourceRecord
}

type partitionResult struct {
	seq uint64
	err error
}

func newTransactionProcessor(ctx context.Context, runtime *Runtime) *transactionProcessor {
	runCtx, cancel := context.WithCancel(ctx)
	workers := make([]chan partitionJob, runtime.workerCount)
	for idx := range workers {
		workers[idx] = make(chan partitionJob, runtime.workerQueueSize)
	}

	return &transactionProcessor{
		ctx:     runCtx,
		cancel:  cancel,
		runtime: runtime,
		input:   make(chan txEnvelope, runtime.workerQueueSize),
		results: make(chan partitionResult, runtime.workerQueueSize*runtime.workerCount),
		workers: workers,
	}
}

func (p *transactionProcessor) start() {
	p.wg.Add(1)
	go p.run()

	for idx := range p.workers {
		p.wg.Add(1)
		go p.worker(p.workers[idx])
	}
}

func (p *transactionProcessor) stop() {
	p.closeInput()
	p.cancel()
	p.wg.Wait()
}

func (p *transactionProcessor) drain() {
	p.closeInput()
	p.wg.Wait()
}

func (p *transactionProcessor) err() error {
	p.errMu.Lock()
	defer p.errMu.Unlock()
	return p.fatalErr
}

func (p *transactionProcessor) setErr(err error) {
	if err == nil {
		return
	}

	p.errMu.Lock()
	defer p.errMu.Unlock()
	if p.fatalErr == nil {
		p.fatalErr = err
		p.cancel()
	}
}

func (p *transactionProcessor) closeInput() {
	p.closeOnce.Do(func() {
		close(p.input)
	})
}

func (p *transactionProcessor) enqueue(ctx context.Context, tx domain.TransactionRecords) error {
	if err := p.err(); err != nil {
		return err
	}

	envelope := txEnvelope{
		seq: atomic.AddUint64(&p.seq, 1),
		tx:  tx,
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.ctx.Done():
		if err := p.err(); err != nil {
			return err
		}
		return p.ctx.Err()
	case p.input <- envelope:
		return nil
	}
}

func (p *transactionProcessor) run() {
	defer p.wg.Done()
	defer func() {
		for _, worker := range p.workers {
			close(worker)
		}
	}()

	states := make(map[uint64]*txState)
	var nextAck uint64 = 1
	input := p.input

	for {
		select {
		case <-p.ctx.Done():
			return
		case envelope, ok := <-input:
			if !ok {
				input = nil
				if len(states) == 0 {
					return
				}
				continue
			}
			state := &txState{tx: envelope.tx}
			jobs := partitionRecords(envelope.seq, envelope.tx)
			state.pending = len(jobs)
			state.done = len(jobs) == 0
			states[envelope.seq] = state

			for _, job := range jobs {
				workerIdx := partitionIndex(job.partition, len(p.workers))
				select {
				case <-p.ctx.Done():
					return
				case p.workers[workerIdx] <- job:
				}
			}
		case result := <-p.results:
			state, ok := states[result.seq]
			if !ok {
				continue
			}
			state.pending--
			if result.err != nil && state.err == nil {
				state.err = result.err
			}
			if state.pending == 0 {
				state.done = true
			}

			for {
				ackState, ok := states[nextAck]
				if !ok || !ackState.done {
					break
				}
				if ackState.err != nil {
					p.setErr(ackState.err)
					return
				}
				if ackState.tx.CommitLSN != "" {
					if err := p.runtime.checkpoints.SaveWALLSN(p.ctx, ackState.tx.CommitLSN); err != nil {
						p.setErr(fmt.Errorf("save wal checkpoint: %w", err))
						return
					}
					if checkpointLSN, err := pglogrepl.ParseLSN(ackState.tx.CommitLSN); err == nil {
						metrics.CheckpointLSNBytes.Set(float64(uint64(checkpointLSN)))
					}
					if err := p.runtime.tailReader.AdvanceLSN(ackState.tx.CommitLSN); err != nil {
						p.setErr(fmt.Errorf("advance tail lsn: %w", err))
						return
					}
				}
				delete(states, nextAck)
				nextAck++
			}
			if input == nil && len(states) == 0 {
				return
			}
		}
	}
}

func (p *transactionProcessor) worker(jobs <-chan partitionJob) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}

			var err error
			for _, record := range job.records {
				record.Metadata.CommitLSN = job.tx.CommitLSN
				record.Metadata.Timestamp = job.tx.Timestamp
				record.Metadata.TransactionID = job.tx.TransactionID
				if err = p.runtime.handleRecord(p.ctx, record); err != nil {
					break
				}
			}

			select {
			case <-p.ctx.Done():
				return
			case p.results <- partitionResult{seq: job.seq, err: err}:
			}
		}
	}
}

func partitionRecords(seq uint64, tx domain.TransactionRecords) []partitionJob {
	grouped := make(map[string][]domain.SourceRecord)
	order := make([]string, 0)

	for _, record := range tx.Records {
		key := record.FullTableName()
		if _, exists := grouped[key]; !exists {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], record)
	}

	jobs := make([]partitionJob, 0, len(order))
	for _, key := range order {
		jobs = append(jobs, partitionJob{
			seq:       seq,
			tx:        tx,
			partition: key,
			records:   grouped[key],
		})
	}

	return jobs
}

func partitionIndex(partition string, workerCount int) int {
	if workerCount <= 1 {
		return 0
	}

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(partition))
	return int(hasher.Sum32() % uint32(workerCount))
}
