package dbdialect

import (
	"errors"
	"fmt"
	"strings"
)

type Kind string

const (
	Postgres = Kind("postgres")
	SQLite   = Kind("sqlite")
)

const DefaultKind = Postgres

var ErrUnknownKind = errors.New("unknown database driver")

func Parse(value string) (Kind, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(Postgres), "postgresql", "pgx", "pg":
		return Postgres, nil
	case string(SQLite), "sqlite3":
		return SQLite, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnknownKind, value)
	}
}

func (k Kind) String() string { return string(k) }

func (k Kind) IsPostgres() bool { return k == Postgres }

func (k Kind) IsSQLite() bool { return k == SQLite }

func (k Kind) DisplayName() string {
	switch k {
	case Postgres:
		return "PostgreSQL"
	case SQLite:
		return "SQLite"
	default:
		return string(k)
	}
}

type Capability string

const (
	CapPostGIS           = Capability("postgis")
	CapFullTextSearch    = Capability("full_text_search")
	CapChangeDataCapture = Capability("change_data_capture")
	CapExactDecimal      = Capability("exact_decimal")
	CapMaterializedView  = Capability("materialized_view")
	CapAdvisoryLock      = Capability("advisory_lock")
	CapRowLocking        = Capability("row_locking")
	CapExclusionConstr   = Capability("exclusion_constraint")
	CapTrigram           = Capability("trigram")
	CapConcurrentIndex   = Capability("concurrent_index")
	CapPartitionedTable  = Capability("partitioned_table")
	CapSessionDiagnostic = Capability("session_diagnostics")
)

var postgresOnly = map[Capability]struct{}{
	CapPostGIS:           {},
	CapFullTextSearch:    {},
	CapChangeDataCapture: {},
	CapExactDecimal:      {},
	CapMaterializedView:  {},
	CapAdvisoryLock:      {},
	CapRowLocking:        {},
	CapExclusionConstr:   {},
	CapTrigram:           {},
	CapConcurrentIndex:   {},
	CapPartitionedTable:  {},
	CapSessionDiagnostic: {},
}

func (k Kind) Supports(capability Capability) bool {
	if k.IsPostgres() {
		return true
	}

	_, restricted := postgresOnly[capability]

	return !restricted
}

func (c Capability) DisplayName() string {
	switch c {
	case CapPostGIS:
		return "PostGIS spatial queries"
	case CapFullTextSearch:
		return "Postgres full-text search"
	case CapChangeDataCapture:
		return "logical-replication change data capture"
	case CapExactDecimal:
		return "exact decimal arithmetic"
	case CapMaterializedView:
		return "materialized views"
	case CapAdvisoryLock:
		return "advisory locks"
	case CapRowLocking:
		return "SELECT ... FOR UPDATE row locking"
	case CapExclusionConstr:
		return "exclusion constraints"
	case CapTrigram:
		return "trigram similarity search"
	case CapConcurrentIndex:
		return "concurrent index creation"
	case CapPartitionedTable:
		return "declarative table partitioning"
	case CapSessionDiagnostic:
		return "database session diagnostics"
	default:
		return string(c)
	}
}

// NowEpoch returns the SQL expression for the current Unix timestamp as an
// integer. PostgreSQL has no portable spelling of this and SQLite has no
// extract(), so query builders must ask for it rather than hardcoding either.
func (k Kind) NowEpoch() string {
	if k.IsSQLite() {
		return "unixepoch()"
	}

	return "extract(epoch from current_timestamp)::bigint"
}
