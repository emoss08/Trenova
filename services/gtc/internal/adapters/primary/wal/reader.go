package wal

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emoss08/gtc/internal/core/ports"
	"github.com/emoss08/gtc/internal/infrastructure/metrics"
	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"go.uber.org/zap"
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type Config struct {
	DatabaseURL         string
	SlotName            string
	PublicationName     string
	PublicationTables   []string
	StandbyTimeout      time.Duration
	ReconnectBackoff    time.Duration
	MaxReconnectBackoff time.Duration
	AutoCreateSlot      bool
	AutoCreatePub       bool
	InactiveSlotAction  string
	MaxLagBytes         int64
	SlotRetryInterval   time.Duration
	SlotRetryTimeout    time.Duration
}

type Reader struct {
	config       Config
	logger       *zap.Logger
	conn         *pgconn.PgConn
	connMu       sync.Mutex
	decoder      *Decoder
	clientLSN    atomic.Uint64
	shutdown     chan struct{}
	shutdownOnce sync.Once
	slotHealthMu sync.RWMutex
	slotHealth   slotHealthState
	monitorMu    sync.Mutex
	monitorStop  chan struct{}
}

const slotMonitorInterval = time.Minute

type slotState struct {
	Exists   bool
	Active   bool
	LagBytes int64
}

type slotHealthState struct {
	Active bool
	LagOK  bool
	Known  bool
}

func NewReader(cfg Config, logger *zap.Logger) *Reader {
	return &Reader{
		config:   cfg,
		logger:   logger.Named("wal_reader"),
		decoder:  NewDecoder(),
		shutdown: make(chan struct{}),
		slotHealth: slotHealthState{
			Active: true,
			LagOK:  true,
		},
	}
}

func (r *Reader) Start(ctx context.Context, startLSN string, handler ports.TransactionHandler) error {
	r.logger.Info("starting WAL reader",
		zap.String("slot_name", r.config.SlotName),
		zap.String("publication", r.config.PublicationName),
		zap.Duration("standby_timeout", r.config.StandbyTimeout),
	)

	backoff := r.config.ReconnectBackoff
	if backoff == 0 {
		backoff = time.Second
	}
	maxBackoff := r.config.MaxReconnectBackoff
	if maxBackoff == 0 {
		maxBackoff = 30 * time.Second
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.shutdown:
			return nil
		default:
		}

		if err := r.connect(ctx); err != nil {
			r.logger.Error("connection failed", zap.Error(err))
			r.waitWithBackoff(ctx, backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		if err := r.setupReplication(ctx, startLSN); err != nil {
			r.logger.Error("replication setup failed", zap.Error(err))
			r.stopSlotMonitor()
			r.closeConnection(ctx)
			r.waitWithBackoff(ctx, backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		backoff = r.config.ReconnectBackoff
		if backoff == 0 {
			backoff = time.Second
		}

		r.logger.Info("WAL streaming started, listening for changes")
		if err := r.streamLoop(ctx, handler); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			r.logger.Error("stream loop failed, will reconnect",
				zap.Error(err),
				zap.Duration("backoff", backoff),
			)
			r.stopSlotMonitor()
			r.closeConnection(ctx)
			r.waitWithBackoff(ctx, backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}
		return nil
	}
}

func (r *Reader) waitWithBackoff(ctx context.Context, backoff time.Duration) {
	select {
	case <-ctx.Done():
	case <-r.shutdown:
	case <-time.After(backoff):
	}
}

func (r *Reader) closeConnection(ctx context.Context) {
	r.connMu.Lock()
	defer r.connMu.Unlock()
	if r.conn != nil {
		_ = r.conn.Close(ctx)
		r.conn = nil
	}
}

func (r *Reader) connect(ctx context.Context) error {
	r.logger.Debug("connecting to PostgreSQL")

	conn, err := pgconn.Connect(ctx, r.config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	r.connMu.Lock()
	r.conn = conn
	r.connMu.Unlock()
	r.logger.Info("connected to PostgreSQL")
	return nil
}

func (r *Reader) setupReplication(ctx context.Context, startLSN string) error {
	r.logger.Debug("identifying system")

	sysident, err := pglogrepl.IdentifySystem(ctx, r.conn)
	if err != nil {
		return fmt.Errorf("identify system failed: %w", err)
	}

	r.logger.Info("system identified",
		zap.String("system_id", sysident.SystemID),
		zap.Int("timeline", int(sysident.Timeline)),
		zap.String("xlog_pos", sysident.XLogPos.String()),
		zap.String("db_name", sysident.DBName),
	)

	if err := r.ensurePublication(ctx); err != nil {
		return err
	}

	slotLSN, err := r.ensureReplicationSlot(ctx)
	if err != nil {
		return err
	}

	effectiveLSN := slotLSN
	if startLSN != "" {
		parsed, parseErr := pglogrepl.ParseLSN(startLSN)
		if parseErr != nil {
			return fmt.Errorf("parse start lsn: %w", parseErr)
		}
		effectiveLSN = parsed
	}

	if err := r.startReplicationWithRetry(ctx, effectiveLSN); err != nil {
		return err
	}

	if err := r.verifySlotActivity(ctx); err != nil {
		return err
	}

	r.clientLSN.Store(uint64(effectiveLSN))
	r.startSlotMonitor(ctx)
	r.logger.Info("replication started", zap.String("lsn", effectiveLSN.String()))
	return nil
}

func (r *Reader) ensurePublication(ctx context.Context) error {
	if !validIdentifier.MatchString(r.config.PublicationName) {
		return fmt.Errorf("invalid publication name: %q", r.config.PublicationName)
	}

	checkResult := r.conn.Exec(ctx, fmt.Sprintf(
		"SELECT 1 FROM pg_publication WHERE pubname = '%s'",
		r.config.PublicationName,
	))
	rows, err := checkResult.ReadAll()
	if err != nil {
		return fmt.Errorf("check publication failed: %w", err)
	}

	if len(rows) > 0 && len(rows[0].Rows) > 0 {
		if r.config.AutoCreatePub {
			if err := r.syncPublicationTables(ctx); err != nil {
				return err
			}
		}
		r.logger.Info("publication exists", zap.String("publication", r.config.PublicationName))
		return nil
	}

	if !r.config.AutoCreatePub {
		return fmt.Errorf(
			"publication %q does not exist and auto-create is disabled",
			r.config.PublicationName,
		)
	}

	r.logger.Info("creating publication", zap.String("publication", r.config.PublicationName))
	tableList, err := r.publicationTableList()
	if err != nil {
		return err
	}

	statement := fmt.Sprintf("CREATE PUBLICATION %s FOR TABLE %s", quoteIdentifier(r.config.PublicationName), tableList)
	result := r.conn.Exec(ctx, statement)
	if _, err = result.ReadAll(); err != nil {
		return fmt.Errorf("create publication failed: %w", err)
	}

	r.logger.Info("publication created", zap.String("publication", r.config.PublicationName))
	return nil
}

func (r *Reader) syncPublicationTables(ctx context.Context) error {
	if len(r.config.PublicationTables) == 0 {
		return fmt.Errorf("publication %q has no configured tables", r.config.PublicationName)
	}

	tableList, err := r.publicationTableList()
	if err != nil {
		return err
	}

	statement := fmt.Sprintf(
		"ALTER PUBLICATION %s SET TABLE %s",
		quoteIdentifier(r.config.PublicationName),
		tableList,
	)
	result := r.conn.Exec(ctx, statement)
	if _, err := result.ReadAll(); err != nil {
		return fmt.Errorf("sync publication tables: %w", err)
	}

	return nil
}

func (r *Reader) publicationTableList() (string, error) {
	tables := make([]string, 0, len(r.config.PublicationTables))
	for _, fullName := range r.config.PublicationTables {
		schema, table, err := splitFullTableName(fullName)
		if err != nil {
			return "", err
		}
		tables = append(tables, quoteQualifiedIdentifier(schema, table))
	}
	if len(tables) == 0 {
		return "", fmt.Errorf("no publication tables configured")
	}

	return strings.Join(tables, ", "), nil
}

func (r *Reader) ensureReplicationSlot(ctx context.Context) (pglogrepl.LSN, error) {
	if !validIdentifier.MatchString(r.config.SlotName) {
		return 0, fmt.Errorf("invalid slot name: %q", r.config.SlotName)
	}

	checkResult := r.conn.Exec(ctx, fmt.Sprintf(
		"SELECT confirmed_flush_lsn FROM pg_replication_slots WHERE slot_name = '%s'",
		r.config.SlotName,
	))
	rows, err := checkResult.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("check replication slot failed: %w", err)
	}

	if len(rows) > 0 && len(rows[0].Rows) > 0 && len(rows[0].Rows[0]) > 0 {
		lsnStr := string(rows[0].Rows[0][0])
		startLSN, parseErr := pglogrepl.ParseLSN(lsnStr)
		if parseErr != nil {
			return 0, fmt.Errorf("parse confirmed_flush_lsn failed: %w", parseErr)
		}
		r.logger.Info("replication slot exists",
			zap.String("slot_name", r.config.SlotName),
			zap.String("confirmed_flush_lsn", startLSN.String()),
		)
		return startLSN, nil
	}

	if !r.config.AutoCreateSlot {
		return 0, fmt.Errorf(
			"replication slot %q does not exist and auto-create is disabled",
			r.config.SlotName,
		)
	}

	r.logger.Info("creating replication slot", zap.String("slot_name", r.config.SlotName))
	slotResult, err := pglogrepl.CreateReplicationSlot(
		ctx,
		r.conn,
		r.config.SlotName,
		"pgoutput",
		pglogrepl.CreateReplicationSlotOptions{Temporary: false},
	)
	if err != nil {
		return 0, fmt.Errorf("create replication slot failed: %w", err)
	}

	startLSN, err := pglogrepl.ParseLSN(slotResult.ConsistentPoint)
	if err != nil {
		return 0, fmt.Errorf("parse consistent_point failed: %w", err)
	}

	r.logger.Info("replication slot created",
		zap.String("slot_name", r.config.SlotName),
		zap.String("consistent_point", startLSN.String()),
	)
	return startLSN, nil
}

func (r *Reader) verifySlotActivity(ctx context.Context) error {
	state, err := r.loadSlotState(ctx)
	if err != nil {
		return err
	}

	return r.observeSlotState(state, true)
}

func (r *Reader) startSlotMonitor(ctx context.Context) {
	r.monitorMu.Lock()
	defer r.monitorMu.Unlock()

	if r.monitorStop != nil {
		close(r.monitorStop)
	}

	stop := make(chan struct{})
	r.monitorStop = stop

	go func(stopCh <-chan struct{}) {
		ticker := time.NewTicker(slotMonitorInterval)
		defer ticker.Stop()

		for {
			pollCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			state, err := r.loadSlotState(pollCtx)
			cancel()
			if err != nil {
				r.logger.Error("replication slot monitor query failed",
					zap.String("slot_name", r.config.SlotName),
					zap.Error(err),
				)
			} else if obsErr := r.observeSlotState(state, false); obsErr != nil {
				r.logger.Error("replication slot monitor detected unhealthy state",
					zap.String("slot_name", r.config.SlotName),
					zap.Error(obsErr),
				)
			}

			select {
			case <-ctx.Done():
				return
			case <-r.shutdown:
				return
			case <-stopCh:
				return
			case <-ticker.C:
			}
		}
	}(stop)
}

func (r *Reader) stopSlotMonitor() {
	r.monitorMu.Lock()
	defer r.monitorMu.Unlock()

	if r.monitorStop != nil {
		close(r.monitorStop)
		r.monitorStop = nil
	}
}

func (r *Reader) loadSlotState(ctx context.Context) (slotState, error) {
	conn, err := pgx.Connect(ctx, r.monitorDatabaseURL())
	if err != nil {
		return slotState{}, fmt.Errorf("connect slot monitor: %w", err)
	}
	defer conn.Close(ctx)

	const query = `
		SELECT
			active,
			COALESCE(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn), 0)::bigint
		FROM pg_replication_slots
		WHERE slot_name = $1
	`

	var state slotState
	state.Exists = true
	if err := conn.QueryRow(ctx, query, r.config.SlotName).Scan(&state.Active, &state.LagBytes); err != nil {
		if err == pgx.ErrNoRows {
			return slotState{}, fmt.Errorf("replication slot %q does not exist", r.config.SlotName)
		}
		return slotState{}, fmt.Errorf("query replication slot state: %w", err)
	}

	return state, nil
}

func (r *Reader) observeSlotState(state slotState, startup bool) error {
	metrics.ReplicationSlotActive.WithLabelValues(r.config.SlotName).Set(boolToGauge(state.Active))
	metrics.ReplicationSlotLagBytes.WithLabelValues(r.config.SlotName).Set(float64(state.LagBytes))

	healthyActive := state.Active
	healthyLag := state.LagBytes <= r.config.MaxLagBytes

	r.slotHealthMu.Lock()
	r.slotHealth = slotHealthState{
		Active: healthyActive,
		LagOK:  healthyLag,
		Known:  true,
	}
	r.slotHealthMu.Unlock()

	if !healthyActive {
		fields := []zap.Field{
			zap.String("slot_name", r.config.SlotName),
			zap.Bool("active", state.Active),
			zap.Int64("lag_bytes", state.LagBytes),
			zap.String("inactive_slot_action", r.config.InactiveSlotAction),
		}
		if startup {
			r.logger.Error("replication slot is inactive after replication start", fields...)
			if r.config.InactiveSlotAction == "fail" {
				return fmt.Errorf("replication slot %q is inactive", r.config.SlotName)
			}
		} else {
			r.logger.Error("replication slot became inactive", fields...)
		}
	}

	if !healthyLag {
		r.logger.Error("replication slot lag exceeds threshold",
			zap.String("slot_name", r.config.SlotName),
			zap.Int64("lag_bytes", state.LagBytes),
			zap.Int64("max_lag_bytes", r.config.MaxLagBytes),
		)
	}

	return nil
}

func (r *Reader) startReplicationWithRetry(ctx context.Context, startLSN pglogrepl.LSN) error {
	pluginArguments := []string{
		"proto_version '2'",
		fmt.Sprintf("publication_names '%s'", r.config.PublicationName),
		"messages 'true'",
		"streaming 'true'",
	}

	deadline := time.Now().Add(r.config.SlotRetryTimeout)

	for {
		r.logger.Debug("starting replication", zap.String("start_lsn", startLSN.String()))

		err := pglogrepl.StartReplication(
			ctx,
			r.conn,
			r.config.SlotName,
			startLSN,
			pglogrepl.StartReplicationOptions{PluginArgs: pluginArguments},
		)
		if err == nil {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("start replication failed after timeout: %w", err)
		}

		r.logger.Warn("replication slot busy, retrying",
			zap.Error(err),
			zap.Duration("retry_in", r.config.SlotRetryInterval),
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.config.SlotRetryInterval):
		}
	}
}

func (r *Reader) streamLoop(ctx context.Context, handler ports.TransactionHandler) error {
	nextDeadline := time.Now().Add(r.config.StandbyTimeout)
	eventsProcessed := int64(0)

	for {
		select {
		case <-r.shutdown:
			r.logger.Info(
				"shutdown signal received",
				zap.Int64("events_processed", eventsProcessed),
			)
			return nil
		case <-ctx.Done():
			r.logger.Info("context cancelled", zap.Int64("events_processed", eventsProcessed))
			return ctx.Err()
		default:
		}

		if time.Now().After(nextDeadline) {
			if err := r.sendStandbyStatus(ctx); err != nil {
				r.logger.Error("failed to send standby status", zap.Error(err))
				return err
			}
			nextDeadline = time.Now().Add(r.config.StandbyTimeout)
		}

		msgCtx, cancel := context.WithDeadline(ctx, nextDeadline)
		rawMsg, err := r.conn.ReceiveMessage(msgCtx)
		cancel()

		if err != nil {
			if pgconn.Timeout(err) {
				continue
			}
			r.logger.Error("receive message failed", zap.Error(err))
			return fmt.Errorf("receive message failed: %w", err)
		}

		if errMsg, ok := rawMsg.(*pgproto3.ErrorResponse); ok {
			r.logger.Error("postgres WAL error",
				zap.String("severity", errMsg.Severity),
				zap.String("code", errMsg.Code),
				zap.String("message", errMsg.Message),
				zap.String("detail", errMsg.Detail),
			)
			return fmt.Errorf("postgres WAL error: %s", errMsg.Message)
		}

		result, err := r.decoder.Decode(rawMsg)
		if err != nil {
			r.logger.Error("decode failed", zap.Error(err))
			return fmt.Errorf("decode failed: %w", err)
		}

		if result.Transaction != nil {
			for _, event := range result.Transaction.Records {
				r.logger.Debug("event received",
					zap.String("operation", event.Operation.String()),
					zap.String("schema", event.Schema),
					zap.String("table", event.Table),
					zap.String("lsn", event.Metadata.LSN),
					zap.Int("xid", int(event.Metadata.TransactionID)),
				)
			}

			if err := handler(ctx, *result.Transaction); err != nil {
				r.logger.Error("handler failed",
					zap.Error(err),
					zap.String("commit_lsn", result.Transaction.CommitLSN),
					zap.Int("record_count", len(result.Transaction.Records)),
				)
				return fmt.Errorf("handler failed: %w", err)
			}
			eventsProcessed += int64(len(result.Transaction.Records))
		}
	}
}

func (r *Reader) AdvanceLSN(lsn string) error {
	parsedLSN, err := pglogrepl.ParseLSN(lsn)
	if err != nil {
		return fmt.Errorf("parse lsn: %w", err)
	}

	newLSN := uint64(parsedLSN)
	for {
		current := r.clientLSN.Load()
		if newLSN <= current {
			return nil
		}
		if r.clientLSN.CompareAndSwap(current, newLSN) {
			return nil
		}
	}
}

func (r *Reader) sendStandbyStatus(ctx context.Context) error {
	lsn := pglogrepl.LSN(r.clientLSN.Load())

	r.logger.Debug("sending standby status", zap.String("lsn", lsn.String()))

	return pglogrepl.SendStandbyStatusUpdate(
		ctx,
		r.conn,
		pglogrepl.StandbyStatusUpdate{WALWritePosition: lsn},
	)
}

func (r *Reader) Stop(ctx context.Context) error {
	r.logger.Info("stopping WAL reader")
	r.shutdownOnce.Do(func() { close(r.shutdown) })
	r.stopSlotMonitor()

	r.connMu.Lock()
	conn := r.conn
	r.conn = nil
	r.connMu.Unlock()

	if conn != nil {
		if err := conn.Close(ctx); err != nil {
			r.logger.Error("failed to close connection", zap.Error(err))
			return err
		}
	}

	r.logger.Info("WAL reader stopped")
	return nil
}

func (r *Reader) CurrentLSN() string {
	return pglogrepl.LSN(r.clientLSN.Load()).String()
}

func (r *Reader) HealthStatuses() map[string]bool {
	r.slotHealthMu.RLock()
	defer r.slotHealthMu.RUnlock()

	if !r.slotHealth.Known {
		return map[string]bool{"replication_slot": true}
	}

	return map[string]bool{
		"replication_slot": r.slotHealth.Active && r.slotHealth.LagOK,
	}
}

func (r *Reader) monitorDatabaseURL() string {
	parsed, err := url.Parse(r.config.DatabaseURL)
	if err != nil {
		return r.config.DatabaseURL
	}

	query := parsed.Query()
	query.Del("replication")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func boolToGauge(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func quoteIdentifier(value string) string {
	return pgx.Identifier{value}.Sanitize()
}

func quoteQualifiedIdentifier(schema string, table string) string {
	return pgx.Identifier{schema, table}.Sanitize()
}

func splitFullTableName(value string) (string, string, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid table name %q", value)
	}

	return parts[0], parts[1], nil
}
