package datatableinsightresolver

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/app/tableinsightservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/shopspring/decimal"
)

func insightScope(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	filter *gqlmodel.DataTableConnectionInput,
	options map[string]any,
) (permissionports.TableInsightScope, error) {
	connection, err := gqlconnection.DataTableConnectionFromGraphQL(
		ctx,
		filter,
		resolvercore.TenantInfo(authCtx),
	)
	if err != nil {
		return permissionports.TableInsightScope{}, err
	}

	return permissionports.TableInsightScope{Filter: connection.Filter, Options: options}, nil
}

func decimalString(value *decimal.Decimal) *string {
	if value == nil {
		return nil
	}
	text := value.String()
	return &text
}

func facetsToModel(
	field string,
	result *permissionports.TableFacetResult,
) *gqlmodel.DataTableFacets {
	buckets := make([]*gqlmodel.DataTableFacetBucket, 0, len(result.Buckets))
	for _, bucket := range result.Buckets {
		buckets = append(buckets, &gqlmodel.DataTableFacetBucket{
			Value: bucket.Value,
			Count: bucket.Count,
		})
	}
	return &gqlmodel.DataTableFacets{Field: field, Buckets: buckets, Total: result.Total}
}

func aggregatesToModel(result *permissionports.TableAggregateResult) *gqlmodel.DataTableAggregates {
	values := make([]*gqlmodel.DataTableAggregateValue, 0, len(result.Values))
	for _, value := range result.Values {
		values = append(values, &gqlmodel.DataTableAggregateValue{
			Field:   value.Field,
			Sum:     decimalString(value.Sum),
			Average: decimalString(value.Average),
			Min:     decimalString(value.Min),
			Max:     decimalString(value.Max),
		})
	}
	return &gqlmodel.DataTableAggregates{Count: result.Count, Values: values}
}

func fieldsToModel(fields []permissionports.TableInsightField) []*gqlmodel.DataTableInsightField {
	models := make([]*gqlmodel.DataTableInsightField, 0, len(fields))
	for _, field := range fields {
		models = append(models, &gqlmodel.DataTableInsightField{
			Name:      field.Name,
			Facetable: field.Facetable,
			Summable:  field.Summable,
			Timeline:  field.Timeline,
		})
	}
	return models
}

func seriesRequest(
	input *gqlmodel.DataTableSeriesInput,
	scope permissionports.TableInsightScope,
) *tableinsightservice.SeriesRequest {
	req := &tableinsightservice.SeriesRequest{
		Scope:       scope,
		GroupField:  input.GroupField,
		GroupValues: input.GroupValues,
		DateField:   input.DateField,
		Interval:    tableinsightservice.SeriesInterval(strings.ToLower(string(input.Interval))),
		Periods:     input.Periods,
		Now:         time.Now(),
	}
	if input.ValueField != nil {
		req.ValueField = *input.ValueField
	}
	if input.Timezone != nil {
		req.Timezone = *input.Timezone
	}
	return req
}

func seriesToModel(result *tableinsightservice.SeriesResult) *gqlmodel.DataTableSeries {
	groups := make([]*gqlmodel.DataTableSeriesGroup, 0, len(result.Groups))
	for _, group := range result.Groups {
		points := make([]string, 0, len(group.Points))
		for _, point := range group.Points {
			points = append(points, point.String())
		}
		groups = append(groups, &gqlmodel.DataTableSeriesGroup{Value: group.Value, Points: points})
	}
	starts := make([]int, 0, len(result.PeriodStarts))
	for _, start := range result.PeriodStarts {
		starts = append(starts, int(start))
	}
	return &gqlmodel.DataTableSeries{PeriodStarts: starts, Groups: groups}
}
