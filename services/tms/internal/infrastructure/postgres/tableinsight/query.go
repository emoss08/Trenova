package tableinsight

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

const (
	DefaultFacetLimit  = 20
	MaxFacetLimit      = 100
	MaxAggregateFields = 20
	MaxSeriesGroups    = 200
	MaxSeriesPeriods   = 60

	aggregatesPerField = 4
	maxMatchCapacity   = 1024
)

type Base interface {
	facet(
		ctx context.Context,
		db *keel.DB,
		columns Columns,
		req *permissionports.TableFacetRequest,
	) (*permissionports.TableFacetResult, error)
	aggregate(
		ctx context.Context,
		db *keel.DB,
		columns Columns,
		req *permissionports.TableAggregateRequest,
	) (*permissionports.TableAggregateResult, error)
	series(
		ctx context.Context,
		db *keel.DB,
		columns Columns,
		req *permissionports.TableSeriesRequest,
	) (*permissionports.TableSeriesResult, error)
	matchingIDs(
		ctx context.Context,
		db *keel.DB,
		id keel.Col,
		limit int,
	) (*permissionports.TableMatchResult, error)
}

func Typed[T any](q *keel.Query[T]) Base {
	return &typedBase[T]{q: q}
}

type typedBase[T any] struct {
	q *keel.Query[T]
}

type facetRow struct {
	Value *string
	Count int
	Total int
}

func (b *typedBase[T]) facet(
	ctx context.Context,
	db *keel.DB,
	columns Columns,
	req *permissionports.TableFacetRequest,
) (*permissionports.TableFacetResult, error) {
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

	value := keel.DynOf(column.Col).Cast(keel.TypeString)
	total := keel.DynOf(keel.Sum(keel.Count()).OverWindow(keel.Window{})).Cast(keel.TypeInt64)
	rows, err := keel.Into[facetRow](ctx, db, b.q.Clone().
		Pick(value, keel.Count(), total).
		Group(column.Col).
		Order(keel.Count().Desc(), keel.Asc(value).NullsLast()).
		Limit(limit))
	if err != nil {
		return nil, err
	}

	result := &permissionports.TableFacetResult{
		Buckets: make([]permissionports.TableFacetBucket, 0, len(rows)),
	}
	for _, row := range rows {
		result.Buckets = append(result.Buckets, permissionports.TableFacetBucket{
			Value: row.Value,
			Count: row.Count,
		})
		result.Total = row.Total
	}

	return result, nil
}

func (b *typedBase[T]) aggregate(
	ctx context.Context,
	db *keel.DB,
	columns Columns,
	req *permissionports.TableAggregateRequest,
) (*permissionports.TableAggregateResult, error) {
	if len(req.Fields) > MaxAggregateFields {
		return nil, errortypes.NewValidationError(
			"fields",
			errortypes.ErrInvalid,
			fmt.Sprintf("A table can total at most %d columns at once", MaxAggregateFields),
		)
	}

	values := make([]any, 0, 1+aggregatesPerField*len(req.Fields))
	values = append(values, keel.Count())
	for idx, field := range req.Fields {
		column, ok := columns[field]
		if !ok || !column.Summable {
			return nil, errortypes.NewValidationError(
				fmt.Sprintf("fields[%d]", idx),
				errortypes.ErrInvalid,
				fmt.Sprintf("This table cannot total %q", field),
			)
		}
		values = append(values,
			keel.Sum(column.Col),
			keel.Avg(column.Col),
			keel.Min(column.Col),
			keel.Max(column.Col),
		)
	}

	totals, err := keel.IntoOne[[]*string](ctx, db, b.q.Clone().Pick(textArray(values)))
	if err != nil {
		return nil, err
	}
	if len(totals) != len(values) {
		return nil, fmt.Errorf(
			"read table totals: %d values for %d totals",
			len(totals),
			len(values),
		)
	}

	result := &permissionports.TableAggregateResult{
		Values: make([]permissionports.TableAggregateValue, 0, len(req.Fields)),
	}
	count, err := parseInsightDecimal(totals[0])
	if err != nil {
		return nil, err
	}
	if count != nil {
		result.Count = int(count.IntPart())
	}

	for idx, field := range req.Fields {
		value := permissionports.TableAggregateValue{Field: field}
		offset := 1 + idx*aggregatesPerField
		for at, target := range []**decimal.Decimal{&value.Sum, &value.Average, &value.Min, &value.Max} {
			parsed, pErr := parseInsightDecimal(totals[offset+at])
			if pErr != nil {
				return nil, pErr
			}
			*target = parsed
		}
		result.Values = append(result.Values, value)
	}

	return result, nil
}

func textArray(values []any) keel.RawExpr {
	return keel.Raw(
		"ARRAY["+strings.TrimSuffix(strings.Repeat("(?)::text, ", len(values)), ", ")+"]",
		values...,
	)
}

type seriesRow struct {
	Group  string
	Bucket int
	Value  string
}

type seriesExpressions struct {
	group keel.Col
	date  keel.Col
	value keel.Expr
}

func seriesColumns(
	columns Columns,
	req *permissionports.TableSeriesRequest,
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

	exprs := &seriesExpressions{group: group.Col, date: date.Col, value: keel.Count()}
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
	exprs.value = keel.DynOf(keel.Sum(value.Col)).Coalesce(0)

	return exprs, nil
}

func seriesPeriods(req *permissionports.TableSeriesRequest) (int, error) {
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

func (b *typedBase[T]) series(
	ctx context.Context,
	db *keel.DB,
	columns Columns,
	req *permissionports.TableSeriesRequest,
) (*permissionports.TableSeriesResult, error) {
	exprs, err := seriesColumns(columns, req)
	if err != nil {
		return nil, err
	}
	if len(req.GroupValues) == 0 {
		return &permissionports.TableSeriesResult{}, nil
	}
	periods, err := seriesPeriods(req)
	if err != nil {
		return nil, err
	}

	group := keel.DynOf(exprs.group).Cast(keel.TypeString)
	bucket := keel.Raw("width_bucket(?, ?::bigint[])", exprs.date, req.Bounds)
	rows, err := keel.Into[seriesRow](ctx, db, b.q.Clone().
		Pick(group, bucket, keel.DynOf(exprs.value).Cast(keel.TypeString)).
		Where(
			keel.DynOf(exprs.group).In(keel.DynLit(req.GroupValues)),
			keel.DynOf(exprs.date).Gte(req.Bounds[0]),
			keel.DynOf(exprs.date).Lt(req.Bounds[periods]),
		).
		Group(group, bucket))
	if err != nil {
		return nil, err
	}

	return seriesResult(req.GroupValues, rows, periods)
}

func seriesResult(
	groupValues []string,
	rows []seriesRow,
	periods int,
) (*permissionports.TableSeriesResult, error) {
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

	result := &permissionports.TableSeriesResult{
		Groups: make([]permissionports.TableSeriesGroup, 0, len(byGroup)),
	}
	for _, value := range groupValues {
		points, found := byGroup[value]
		if !found {
			continue
		}
		result.Groups = append(result.Groups, permissionports.TableSeriesGroup{
			Value:  value,
			Points: points,
		})
		delete(byGroup, value)
	}

	return result, nil
}

type matchRow struct {
	ID    string
	Total int
}

func (b *typedBase[T]) matchingIDs(
	ctx context.Context,
	db *keel.DB,
	id keel.Col,
	limit int,
) (*permissionports.TableMatchResult, error) {
	if limit <= 0 {
		return nil, errortypes.NewValidationError(
			"limit",
			errortypes.ErrInvalid,
			"Say how many rows to find",
		)
	}

	rows, err := keel.Into[matchRow](ctx, db, b.q.Clone().
		Pick(keel.DynOf(id).Cast(keel.TypeString), keel.Count().OverWindow(keel.Window{})).
		Order(id.Asc()).
		Limit(limit))
	if err != nil {
		return nil, err
	}

	result := &permissionports.TableMatchResult{
		IDs: make([]string, 0, min(len(rows), maxMatchCapacity)),
	}
	for _, row := range rows {
		result.IDs = append(result.IDs, row.ID)
		result.Total = row.Total
	}

	return result, nil
}

func parseInsightDecimal(raw *string) (*decimal.Decimal, error) {
	if raw == nil || *raw == "" {
		return nil, nil //nolint:nilnil // SQL NULL: a total over no rows has no value
	}

	parsed, err := decimal.NewFromString(*raw)
	if err != nil {
		return nil, fmt.Errorf("read table total %q: %w", *raw, err)
	}

	return &parsed, nil
}
