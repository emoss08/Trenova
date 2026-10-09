package dbhelper

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

const (
	DefaultFacetLimit  = 20
	MaxFacetLimit      = 100
	MaxAggregateFields = 20
	MaxSeriesGroups    = 200
	MaxSeriesPeriods   = 60
)

type InsightColumn struct {
	Column    buncolgen.Column
	Facetable bool
	Summable  bool
	Timeline  bool
}

type InsightColumns map[string]InsightColumn

func (c InsightColumns) Fields() []repositories.TableInsightField {
	fields := make([]repositories.TableInsightField, 0, len(c))
	for name := range c {
		column := c[name]
		fields = append(fields, repositories.TableInsightField{
			Name:      name,
			Facetable: column.Facetable,
			Summable:  column.Summable,
			Timeline:  column.Timeline,
		})
	}
	slices.SortFunc(fields, func(a, b repositories.TableInsightField) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return fields
}

type facetRow struct {
	Value *string `bun:"value"`
	Count int     `bun:"count"`
	Total int     `bun:"total"`
}

func Facet(
	ctx context.Context,
	base *bun.SelectQuery,
	columns InsightColumns,
	req *repositories.TableFacetRequest,
) (*repositories.TableFacetResult, error) {
	column, ok := columns[req.Field]
	if !ok || !column.Facetable {
		return nil, errortypes.NewValidationError(
			"field",
			errortypes.ErrInvalid,
			fmt.Sprintf("This table cannot be counted by %q", req.Field),
		)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = DefaultFacetLimit
	}
	limit = min(limit, MaxFacetLimit)

	expr := column.Column.Qualified()
	rows := make([]facetRow, 0, limit)
	err := base.
		ColumnExpr(expr+"::text AS value").
		ColumnExpr("count(*) AS count").
		ColumnExpr("(sum(count(*)) OVER ())::bigint AS total").
		GroupExpr(expr).
		OrderExpr("count DESC, value ASC NULLS LAST").
		Limit(limit).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	result := &repositories.TableFacetResult{
		Buckets: make([]repositories.TableFacetBucket, 0, len(rows)),
	}
	for _, row := range rows {
		result.Buckets = append(result.Buckets, repositories.TableFacetBucket{
			Value: row.Value,
			Count: row.Count,
		})
		result.Total = row.Total
	}

	return result, nil
}

func Aggregate(
	ctx context.Context,
	base *bun.SelectQuery,
	columns InsightColumns,
	req *repositories.TableAggregateRequest,
) (*repositories.TableAggregateResult, error) {
	if len(req.Fields) > MaxAggregateFields {
		return nil, errortypes.NewValidationError(
			"fields",
			errortypes.ErrInvalid,
			fmt.Sprintf("A table can total at most %d columns at once", MaxAggregateFields),
		)
	}

	for idx, field := range req.Fields {
		if column, ok := columns[field]; !ok || !column.Summable {
			return nil, errortypes.NewValidationError(
				fmt.Sprintf("fields[%d]", idx),
				errortypes.ErrInvalid,
				fmt.Sprintf("This table cannot total %q", field),
			)
		}
	}

	query := base.ColumnExpr("count(*)::text AS count")
	for idx, field := range req.Fields {
		column := columns[field]
		expr := column.Column.Qualified()
		query = query.
			ColumnExpr(fmt.Sprintf("sum(%s)::text AS f%d_sum", expr, idx)).
			ColumnExpr(fmt.Sprintf("avg(%s)::text AS f%d_avg", expr, idx)).
			ColumnExpr(fmt.Sprintf("min(%s)::text AS f%d_min", expr, idx)).
			ColumnExpr(fmt.Sprintf("max(%s)::text AS f%d_max", expr, idx))
	}

	row := make(map[string]any, 1+4*len(req.Fields))
	if err := query.Scan(ctx, &row); err != nil {
		return nil, err
	}

	count, err := parseInsightDecimal(row["count"])
	if err != nil {
		return nil, err
	}
	result := &repositories.TableAggregateResult{
		Values: make([]repositories.TableAggregateValue, 0, len(req.Fields)),
	}
	if count != nil {
		result.Count = int(count.IntPart())
	}

	for idx, field := range req.Fields {
		value := repositories.TableAggregateValue{Field: field}
		for suffix, target := range map[string]**decimal.Decimal{
			"sum": &value.Sum,
			"avg": &value.Average,
			"min": &value.Min,
			"max": &value.Max,
		} {
			parsed, pErr := parseInsightDecimal(row[fmt.Sprintf("f%d_%s", idx, suffix)])
			if pErr != nil {
				return nil, pErr
			}
			*target = parsed
		}
		result.Values = append(result.Values, value)
	}

	return result, nil
}

type seriesRow struct {
	Group  string `bun:"grp"`
	Bucket int    `bun:"bucket"`
	Value  string `bun:"value"`
}

type seriesExpressions struct {
	group string
	date  string
	value string
}

func seriesColumns(
	columns InsightColumns,
	req *repositories.TableSeriesRequest,
) (*seriesExpressions, error) {
	group, ok := columns[req.GroupField]
	if !ok || !group.Facetable {
		return nil, errortypes.NewValidationError(
			"groupField",
			errortypes.ErrInvalid,
			fmt.Sprintf("This table cannot be grouped by %q", req.GroupField),
		)
	}
	date, ok := columns[req.DateField]
	if !ok || !date.Timeline {
		return nil, errortypes.NewValidationError(
			"dateField",
			errortypes.ErrInvalid,
			fmt.Sprintf("This table cannot be charted over %q", req.DateField),
		)
	}

	exprs := &seriesExpressions{
		group: group.Column.Qualified(),
		date:  date.Column.Qualified(),
		value: "count(*)",
	}
	if req.ValueField == "" {
		return exprs, nil
	}
	value, ok := columns[req.ValueField]
	if !ok || !value.Summable {
		return nil, errortypes.NewValidationError(
			"valueField",
			errortypes.ErrInvalid,
			fmt.Sprintf("This table cannot total %q", req.ValueField),
		)
	}
	exprs.value = fmt.Sprintf("coalesce(sum(%s), 0)", value.Column.Qualified())
	return exprs, nil
}

func seriesPeriods(req *repositories.TableSeriesRequest) (int, error) {
	if len(req.GroupValues) > MaxSeriesGroups {
		return 0, errortypes.NewValidationError(
			"groupValues",
			errortypes.ErrInvalid,
			fmt.Sprintf("A chart can follow at most %d rows at once", MaxSeriesGroups),
		)
	}
	periods := len(req.Bounds) - 1
	if periods < 1 || periods > MaxSeriesPeriods {
		return 0, errortypes.NewValidationError(
			"periods",
			errortypes.ErrInvalid,
			fmt.Sprintf("A chart covers between 1 and %d periods", MaxSeriesPeriods),
		)
	}
	return periods, nil
}

func seriesResult(
	groupValues []string,
	rows []seriesRow,
	periods int,
) (*repositories.TableSeriesResult, error) {
	byGroup := make(map[string][]decimal.Decimal, len(groupValues))
	for _, value := range groupValues {
		if _, seen := byGroup[value]; seen {
			continue
		}
		points := make([]decimal.Decimal, periods)
		for idx := range points {
			points[idx] = decimal.Zero
		}
		byGroup[value] = points
	}
	for _, row := range rows {
		points, found := byGroup[row.Group]
		if !found || row.Bucket < 1 || row.Bucket > periods {
			continue
		}
		parsed, err := decimal.NewFromString(row.Value)
		if err != nil {
			return nil, fmt.Errorf("read chart value %q: %w", row.Value, err)
		}
		points[row.Bucket-1] = parsed
	}

	result := &repositories.TableSeriesResult{
		Groups: make([]repositories.TableSeriesGroup, 0, len(byGroup)),
	}
	for _, value := range groupValues {
		points, found := byGroup[value]
		if !found {
			continue
		}
		result.Groups = append(result.Groups, repositories.TableSeriesGroup{
			Value:  value,
			Points: points,
		})
		delete(byGroup, value)
	}
	return result, nil
}

func Series(
	ctx context.Context,
	base *bun.SelectQuery,
	columns InsightColumns,
	req *repositories.TableSeriesRequest,
) (*repositories.TableSeriesResult, error) {
	exprs, err := seriesColumns(columns, req)
	if err != nil {
		return nil, err
	}
	if len(req.GroupValues) == 0 {
		return &repositories.TableSeriesResult{}, nil
	}
	periods, err := seriesPeriods(req)
	if err != nil {
		return nil, err
	}

	rows := make([]seriesRow, 0, len(req.GroupValues)*periods)
	err = base.
		ColumnExpr(exprs.group+"::text AS grp").
		ColumnExpr("width_bucket("+exprs.date+", ?::bigint[]) AS bucket", pgdialect.Array(req.Bounds)).
		ColumnExpr(exprs.value+"::text AS value").
		Where(exprs.group+" IN (?)", bun.List(req.GroupValues)).
		Where(exprs.date+" >= ?", req.Bounds[0]).
		Where(exprs.date+" < ?", req.Bounds[periods]).
		GroupExpr("grp").
		GroupExpr("bucket").
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	return seriesResult(req.GroupValues, rows, periods)
}

type matchRow struct {
	ID    string `bun:"id"`
	Total int    `bun:"total"`
}

func MatchingIDs(
	ctx context.Context,
	base *bun.SelectQuery,
	idColumn *buncolgen.Column,
	limit int,
) (*repositories.TableMatchResult, error) {
	if limit <= 0 {
		return nil, errortypes.NewValidationError(
			"limit",
			errortypes.ErrInvalid,
			"Say how many rows to find",
		)
	}

	expr := idColumn.Qualified()
	rows := make([]matchRow, 0, min(limit, 1024))
	err := base.
		ColumnExpr(expr+"::text AS id").
		ColumnExpr("(count(*) OVER ())::bigint AS total").
		OrderExpr(expr+" ASC").
		Limit(limit).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	result := &repositories.TableMatchResult{IDs: make([]string, 0, len(rows))}
	for _, row := range rows {
		result.IDs = append(result.IDs, row.ID)
		result.Total = row.Total
	}
	return result, nil
}

func parseInsightDecimal(raw any) (*decimal.Decimal, error) {
	var text string
	switch value := raw.(type) {
	case nil:
		return nil, nil //nolint:nilnil // SQL NULL: a total over no rows has no value
	case string:
		text = value
	case []byte:
		text = string(value)
	default:
		text = fmt.Sprint(value)
	}
	if text == "" {
		return nil, nil //nolint:nilnil // an empty total has no value
	}

	parsed, err := decimal.NewFromString(text)
	if err != nil {
		return nil, fmt.Errorf("read table total %q: %w", text, err)
	}
	return &parsed, nil
}
