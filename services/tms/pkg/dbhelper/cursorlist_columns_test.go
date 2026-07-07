package dbhelper

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestTemplateCursorSQLIncludesBaseColumns(t *testing.T) {
	db := bun.NewDB(new(sql.DB), pgdialect.New())
	filter := &pagination.QueryOptions{}
	entities := make([]*edi.EDITemplate, 0)

	q := db.NewSelect().
		Model(&entities).
		ColumnExpr(buncolgen.EDITemplateTable.All())
	q, err := querybuilder.ApplyCursorFilters(q, "et", filter, pagination.CursorInfo{}, (*edi.EDITemplate)(nil))
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range filter.CursorColumns {
		q = q.ColumnExpr("? AS ?", bun.Safe(column.SQLExpression), bun.Ident(column.Alias))
	}
	sqlText := q.String()
	t.Log(sqlText)
	if !strings.Contains(sqlText, "\"et\".*") && !strings.Contains(sqlText, "et.*") {
		t.Fatalf("expected base table star in SELECT, got: %s", sqlText)
	}
	if !strings.Contains(sqlText, "__cursor_value_0") {
		t.Fatalf("expected cursor value column in SELECT, got: %s", sqlText)
	}
}
