package dbpool

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const postgisTypesSQL = `SELECT typname, oid, typarray FROM pg_type WHERE typname IN ('geometry', 'geography')`

func registerPostGIS(ctx context.Context, conn *pgx.Conn) error {
	rows, err := conn.Query(ctx, postgisTypesSQL)
	if err != nil {
		return fmt.Errorf("read postgis types: %w", err)
	}

	types := make([]*pgtype.Type, 0, 4)
	for rows.Next() {
		var name string
		var oid, array uint32
		if err = rows.Scan(&name, &oid, &array); err != nil {
			rows.Close()
			return fmt.Errorf("scan postgis type: %w", err)
		}
		element := &pgtype.Type{Name: name, OID: oid, Codec: pgtype.ByteaCodec{}}
		types = append(types, element, &pgtype.Type{
			Name:  "_" + name,
			OID:   array,
			Codec: &pgtype.ArrayCodec{ElementType: element},
		})
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return fmt.Errorf("read postgis types: %w", err)
	}

	for _, t := range types {
		conn.TypeMap().RegisterType(t)
	}

	return nil
}
