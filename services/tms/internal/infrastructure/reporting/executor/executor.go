package executor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ services.ReportDatasetExecutor = (*Executor)(nil)

type Params struct {
	fx.In

	DB     *postgres.ReportingConnection
	Config *config.Config
	Logger *zap.Logger
}

type Executor struct {
	db  *postgres.ReportingConnection
	cfg *config.ReportingConfig
	l   *zap.Logger
}

func New(p Params) services.ReportDatasetExecutor {
	return &Executor{
		db:  p.DB,
		cfg: p.Config.GetReportingConfig(),
		l:   p.Logger.Named("reporting.executor"),
	}
}

func (x *Executor) Open(
	ctx context.Context,
	req *services.OpenReportDatasetRequest,
) (services.ReportDatasetReader, error) {
	maxRows := req.MaxRows
	if maxRows <= 0 {
		maxRows = x.cfg.GetMaxRows()
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = x.cfg.GetStatementTimeout()
	}

	queryCtx, cancel := context.WithTimeout(ctx, timeout)

	tx, err := x.db.DB().BeginTx(queryCtx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("begin read-only report transaction: %w", err)
	}

	//nolint:rowserrcheck // rows.Err is checked by datasetReader.Next on stream end
	rows, err := tx.QueryContext(queryCtx, req.Compiled.SQL, req.Compiled.Args...)
	if err != nil {
		_ = tx.Rollback()
		cancel()
		return nil, fmt.Errorf("execute report query: %w", err)
	}

	return &datasetReader{
		schema:   req.Compiled.Columns,
		rows:     rows,
		tx:       tx,
		cancel:   cancel,
		maxRows:  maxRows,
		compiled: req.Compiled,
		db:       x.db,
		timeout:  timeout,
	}, nil
}

type datasetReader struct {
	schema    []services.ReportResultColumn
	rows      *sql.Rows
	tx        bun.Tx
	cancel    context.CancelFunc
	maxRows   int64
	count     int64
	truncated bool
	closed    bool

	compiled   *services.CompiledReportQuery
	db         *postgres.ReportingConnection
	timeout    time.Duration
	totals     services.ReportRow
	totalsDone bool
}

func (r *datasetReader) Schema() []services.ReportResultColumn { return r.schema }

func (r *datasetReader) RowCount() int64 { return r.count }

func (r *datasetReader) Truncated() bool { return r.truncated }

func (r *datasetReader) Next(ctx context.Context) (services.ReportRow, error) {
	if r.closed {
		return nil, io.EOF
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.count >= r.maxRows {
		if r.rows.Next() {
			r.truncated = true
		}
		return nil, io.EOF
	}

	if !r.rows.Next() {
		if err := r.rows.Err(); err != nil {
			return nil, fmt.Errorf("stream report rows: %w", err)
		}
		return nil, io.EOF
	}

	raw := make([]any, len(r.schema))
	scanTargets := make([]any, len(r.schema))
	for i := range raw {
		scanTargets[i] = &raw[i]
	}
	if err := r.rows.Scan(scanTargets...); err != nil {
		return nil, fmt.Errorf("scan report row: %w", err)
	}

	row := make(services.ReportRow, len(r.schema))
	for i := range r.schema {
		value, err := decodeValue(r.schema[i].Type, raw[i])
		if err != nil {
			return nil, fmt.Errorf("decode column %q: %w", r.schema[i].ID, err)
		}
		row[i] = value
	}

	r.count++
	return row, nil
}

// Totals runs the compiled grand-total query on first call and memoizes it, so
// a renderer that asks for totals at end-of-stream pays for one extra scan and
// a caller that never asks pays for none.
func (r *datasetReader) Totals(ctx context.Context) (services.ReportRow, error) {
	if r.totalsDone || r.compiled == nil || !r.compiled.HasTotals() {
		return r.totals, nil
	}
	r.totalsDone = true

	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	tx, err := r.db.DB().BeginTx(queryCtx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin read-only totals transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(queryCtx, r.compiled.TotalsSQL, r.compiled.TotalsArgs...)

	raw := make([]any, len(r.schema))
	scanTargets := make([]any, len(r.schema))
	for i := range raw {
		scanTargets[i] = &raw[i]
	}
	if err := row.Scan(scanTargets...); err != nil {
		return nil, fmt.Errorf("scan report totals: %w", err)
	}

	totals := make(services.ReportRow, len(r.schema))
	for i := range r.schema {
		value, err := decodeValue(r.schema[i].Type, raw[i])
		if err != nil {
			return nil, fmt.Errorf("decode totals column %q: %w", r.schema[i].ID, err)
		}
		totals[i] = value
	}

	r.totals = totals
	return totals, nil
}

func (r *datasetReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	err := r.rows.Close()
	if rbErr := r.tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) && err == nil {
		err = rbErr
	}
	r.cancel()
	return err
}

//nolint:nilnil // a nil value is the legitimate representation of SQL NULL
func decodeValue(fieldType reportcatalog.FieldType, raw any) (any, error) {
	if raw == nil {
		return nil, nil
	}

	//nolint:exhaustive // string-like types share the default decoding
	switch fieldType {
	case reportcatalog.FieldInt, reportcatalog.FieldEpoch:
		return decodeInt(raw)
	case reportcatalog.FieldDecimal:
		return decodeDecimal(raw)
	case reportcatalog.FieldBool:
		return decodeBool(raw)
	default:
		return decodeString(raw), nil
	}
}

func decodeString(raw any) string {
	switch v := raw.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return fmt.Sprint(v)
	}
}

func decodeInt(raw any) (int64, error) {
	switch v := raw.(type) {
	case int64:
		return v, nil
	case []byte:
		return strconv.ParseInt(string(v), 10, 64)
	case string:
		return strconv.ParseInt(v, 10, 64)
	case float64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unexpected integer representation %T", raw)
	}
}

func decodeDecimal(raw any) (decimal.Decimal, error) {
	switch v := raw.(type) {
	case []byte:
		return decimal.NewFromString(string(v))
	case string:
		return decimal.NewFromString(v)
	case int64:
		return decimal.NewFromInt(v), nil
	case float64:
		return decimal.NewFromFloat(v), nil
	default:
		return decimal.Decimal{}, fmt.Errorf("unexpected numeric representation %T", raw)
	}
}

func decodeBool(raw any) (bool, error) {
	switch v := raw.(type) {
	case bool:
		return v, nil
	case []byte:
		return parseBoolText(string(v))
	case string:
		return parseBoolText(v)
	default:
		return false, fmt.Errorf("unexpected boolean representation %T", raw)
	}
}

func parseBoolText(s string) (bool, error) {
	switch s {
	case "t", "true", "TRUE", "T", "1":
		return true, nil
	case "f", "false", "FALSE", "F", "0":
		return false, nil
	default:
		return false, fmt.Errorf("unexpected boolean text %q", s)
	}
}
