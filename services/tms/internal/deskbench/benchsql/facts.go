package benchsql

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
)

const (
	maxFactRows = 50
	factTimeout = "10s"
)

type FactSource func(ctx context.Context, session *deskbench.Session, statement string) ([]string, error)

type Database struct {
	db *postgres.Connection
}

func New(db *postgres.Connection) *Database {
	return &Database{db: db}
}

func (d *Database) FactValues(
	ctx context.Context,
	session *deskbench.Session,
	statement string,
) ([]string, error) {
	ctx = session.Context(ctx)

	return dbtx.Read(ctx, d.db, func(ctx context.Context) ([]string, error) {
		db := d.db.Keel(ctx)
		if _, err := keel.Exec(ctx, db, boundStatementTimeoutSQL); err != nil {
			return nil, fmt.Errorf("bound the fact query: %w", err)
		}

		values := make([]string, 0, maxFactRows)
		err := keel.QueryRows(ctx, db, func(row pgx.CollectableRow) error {
			if len(values) >= maxFactRows || len(row.FieldDescriptions()) == 0 {
				return nil
			}
			cells, err := row.Values()
			if err != nil {
				return err
			}
			values = append(values, cellText(cells[0]))
			return nil
		}, statement)
		if err != nil {
			return nil, err
		}

		return values, nil
	})
}

const boundStatementTimeoutSQL = "SET LOCAL statement_timeout = '" + factTimeout + "'"

func cellText(cell any) string {
	switch v := cell.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.DateOnly)
	case driver.Valuer:
		value, err := v.Value()
		if err != nil {
			return fmt.Sprint(v)
		}
		return cellText(value)
	case map[string]any, []any:
		text, err := sonic.MarshalString(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return text
	default:
		return fmt.Sprint(v)
	}
}

func Quote(runID pulid.ID) string {
	return "'" + strings.ReplaceAll(runID.String(), "'", "''") + "'"
}
