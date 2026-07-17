//go:build integration

package reportcatalog_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func loadTableColumns(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	table string,
) map[string]string {
	t.Helper()

	var rows []struct {
		ColumnName string `bun:"column_name"`
		DataType   string `bun:"data_type"`
	}
	err := db.NewSelect().
		Table("information_schema.columns").
		Column("column_name", "data_type").
		Where("table_schema = current_schema()").
		Where("table_name = ?", table).
		Scan(ctx, &rows)
	require.NoError(t, err, "querying information_schema for table %s", table)

	columns := make(map[string]string, len(rows))
	for _, row := range rows {
		columns[row.ColumnName] = row.DataType
	}
	return columns
}

func typeClassCompatible(fieldType reportcatalog.FieldType, dataType string) bool {
	switch fieldType {
	case reportcatalog.FieldDecimal:
		switch dataType {
		case "numeric", "double precision", "real", "integer", "bigint":
			return true
		}
		return false
	case reportcatalog.FieldInt, reportcatalog.FieldEpoch:
		switch dataType {
		case "smallint", "integer", "bigint", "numeric":
			return true
		}
		return false
	case reportcatalog.FieldBool:
		return dataType == "boolean"
	case reportcatalog.FieldJSON:
		switch dataType {
		case "jsonb", "json", "ARRAY":
			return true
		}
		return false
	case reportcatalog.FieldString, reportcatalog.FieldEnum, reportcatalog.FieldRef:
		switch dataType {
		case "character varying", "text", "character", "USER-DEFINED", "uuid", "citext":
			return true
		}
		return false
	default:
		return false
	}
}

func TestCatalogMatchesLiveSchema(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	for i := range reportcatalog.Default.Entities {
		entity := &reportcatalog.Default.Entities[i]

		t.Run(entity.Key, func(t *testing.T) {
			columns := loadTableColumns(t, ctx, db, entity.Table.Name)
			require.NotEmpty(t, columns, "table %q does not exist in the migrated schema", entity.Table.Name)

			for j := range entity.Fields {
				field := &entity.Fields[j]
				dataType, exists := columns[field.Column.Name]
				require.True(t, exists,
					"catalog field %s.%s maps to column %q which does not exist on table %q",
					entity.Key, field.Key, field.Column.Name, entity.Table.Name)
				require.True(t, typeClassCompatible(field.Type, dataType),
					"catalog field %s.%s has type %q but column %s.%s is %q",
					entity.Key, field.Key, field.Type, entity.Table.Name, field.Column.Name, dataType)
			}

			if entity.Tenant.IsTenanted() {
				_, hasOrg := columns[entity.Tenant.OrganizationID]
				_, hasBU := columns[entity.Tenant.BusinessUnitID]
				require.True(t, hasOrg && hasBU,
					"tenant columns missing on table %q", entity.Table.Name)
			}

			if entity.OwnershipColumn != "" {
				_, hasOwner := columns[entity.OwnershipColumn]
				require.True(t, hasOwner,
					"ownership column %q missing on table %q", entity.OwnershipColumn, entity.Table.Name)
			}

			for j := range entity.Table.PrimaryKey {
				_, hasPK := columns[entity.Table.PrimaryKey[j]]
				require.True(t, hasPK,
					"primary key column %q missing on table %q", entity.Table.PrimaryKey[j], entity.Table.Name)
			}
		})
	}
}

func TestJoinGraphMatchesLiveSchema(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	tableColumns := make(map[string]map[string]string)
	columnsFor := func(table string) map[string]string {
		if cols, ok := tableColumns[table]; ok {
			return cols
		}
		cols := loadTableColumns(t, ctx, db, table)
		tableColumns[table] = cols
		return cols
	}

	for i := range reportcatalog.Default.Entities {
		entity := &reportcatalog.Default.Entities[i]

		for j := range entity.Edges {
			edge := &entity.Edges[j]
			target, ok := reportcatalog.Default.Entity(edge.Target)
			require.True(t, ok, "edge %s.%s targets unknown entity", entity.Key, edge.Name)

			if edge.Through == nil {
				sourceCols := columnsFor(entity.Table.Name)
				targetCols := columnsFor(target.Table.Name)
				for _, jp := range edge.Join {
					_, hasLocal := sourceCols[jp.Local]
					require.True(t, hasLocal,
						"edge %s.%s local column %q missing on %q",
						entity.Key, edge.Name, jp.Local, entity.Table.Name)
					_, hasRemote := targetCols[jp.Remote]
					require.True(t, hasRemote,
						"edge %s.%s remote column %q missing on %q",
						entity.Key, edge.Name, jp.Remote, target.Table.Name)
				}
				continue
			}

			throughCols := columnsFor(edge.Through.Table.Name)
			require.NotEmpty(t, throughCols,
				"m2m through table %q does not exist", edge.Through.Table.Name)
			sourceCols := columnsFor(entity.Table.Name)
			targetCols := columnsFor(target.Table.Name)
			for _, jp := range edge.Through.SourceJoin {
				_, hasLocal := sourceCols[jp.Local]
				require.True(t, hasLocal,
					"m2m edge %s.%s source column %q missing on %q",
					entity.Key, edge.Name, jp.Local, entity.Table.Name)
				_, hasThrough := throughCols[jp.Remote]
				require.True(t, hasThrough,
					"m2m edge %s.%s through column %q missing on %q",
					entity.Key, edge.Name, jp.Remote, edge.Through.Table.Name)
			}
			for _, jp := range edge.Through.TargetJoin {
				_, hasThrough := throughCols[jp.Local]
				require.True(t, hasThrough,
					"m2m edge %s.%s through column %q missing on %q",
					entity.Key, edge.Name, jp.Local, edge.Through.Table.Name)
				_, hasRemote := targetCols[jp.Remote]
				require.True(t, hasRemote,
					"m2m edge %s.%s target column %q missing on %q",
					entity.Key, edge.Name, jp.Remote, target.Table.Name)
			}
		}
	}
}
