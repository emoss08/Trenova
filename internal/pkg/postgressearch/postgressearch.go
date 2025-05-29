package postgressearch

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/uptrace/bun"
)

// Constants for search configuration
const (
	defaultSimilarityThreshold = 0.3
	wildcardPattern            = "%"
)

// DefaultConfig provides sensible defaults
var DefaultConfig = infra.PostgresSearchConfig{
	MinLength: 2,
	MaxTerms:  6,
}

func BuildSearchQuery[T infra.PostgresSearchable](q *bun.SelectQuery, query string, entity T) *bun.SelectQuery {
	config := entity.GetPostgresSearchConfig()

	if len(strings.TrimSpace(query)) < config.MinLength {
		return q
	}

	terms := strings.Fields(query)
	if len(terms) > config.MaxTerms {
		terms = terms[:config.MaxTerms]
	}

	var tsQueryBuilder strings.Builder
	tsQueryBuilder.Grow(len(query) + len(terms)*3)

	for i, term := range terms {
		if i > 0 {
			tsQueryBuilder.WriteString(" | ")
		}
		tsQueryBuilder.WriteString(term)
	}
	tsqueryStr := tsQueryBuilder.String()
	tsqueryWithWildcard := tsqueryStr + ":*"

	tableAlias := config.TableAlias
	tableAliasWithDot := tableAlias + "."

	q = q.ColumnExpr(tableAliasWithDot + "*")

	rankExpr := fmt.Sprintf(
		`ts_rank(%ssearch_vector, to_tsquery('simple', ?)) AS rank`,
		tableAliasWithDot,
	)
	q = q.ColumnExpr(rankExpr, tsqueryWithWildcard)

	whereParts, whereArgs := buildSearchConditions(&config, tableAliasWithDot, query, tsqueryStr)

	var searchCondBuilder strings.Builder
	searchCondBuilder.WriteString("(")
	for i, part := range whereParts {
		if i > 0 {
			searchCondBuilder.WriteString(" OR ")
		}
		searchCondBuilder.WriteString(part)
	}
	searchCondBuilder.WriteString(")")

	q = q.Where(searchCondBuilder.String(), whereArgs...)

	orderParts, orderArgs := buildOrderingConditions(&config, tableAliasWithDot, query)

	for i, orderPart := range orderParts {
		if i < len(orderArgs) {
			q = q.OrderExpr(orderPart, orderArgs[i])
		} else {
			q = q.OrderExpr(orderPart)
		}
	}

	return q
}

func buildSearchConditions(config *infra.PostgresSearchConfig, tableAliasWithDot, query, tsqueryStr string) (whereParts []string, whereArgs []any) {
	whereParts = make([]string, 0, len(config.Fields)+1)
	whereArgs = make([]any, 0, len(config.Fields)*2+1)

	whereParts = append(whereParts,
		fmt.Sprintf("%ssearch_vector @@ to_tsquery('simple', ?)", tableAliasWithDot))
	whereArgs = append(whereArgs, tsqueryStr+":*")

	if config.UsePartialMatch {
		queryWithWildcards := wildcardPattern + query + wildcardPattern

		for _, field := range config.Fields {
			switch field.Type {
			case infra.PostgresSearchTypeArray:
				whereParts = append(whereParts,
					fmt.Sprintf("%s%s @> ?", tableAliasWithDot, field.Name))
				whereArgs = append(whereArgs, queryWithWildcards)
			case infra.PostgresSearchTypeComposite, infra.PostgresSearchTypeNumber:
				whereParts = append(whereParts,
					fmt.Sprintf("%s%s ILIKE ?", tableAliasWithDot, field.Name))
				whereArgs = append(whereArgs, queryWithWildcards)
			case infra.PostgresSearchTypeText:
				whereParts = append(whereParts,
					fmt.Sprintf("(%s%s ILIKE ? OR similarity(%s%s, ?) > %g)",
						tableAliasWithDot, field.Name,
						tableAliasWithDot, field.Name, defaultSimilarityThreshold))
				whereArgs = append(whereArgs, queryWithWildcards, query)

			case infra.PostgresSearchTypeEnum:
				whereParts = append(whereParts,
					fmt.Sprintf("%s%s::text = ?", tableAliasWithDot, field.Name))
				whereArgs = append(whereArgs, query)
			}
		}
	}

	return whereParts, whereArgs
}

func buildOrderingConditions(config *infra.PostgresSearchConfig, tableAliasWithDot, query string) (orderParts []string, orderArgs []any) {
	orderParts = make([]string, 0, len(config.Fields)*2+1)
	orderArgs = make([]any, 0, len(config.Fields)*2)

	for _, field := range config.Fields {
		if field.Type == infra.PostgresSearchTypeComposite || field.Type == infra.PostgresSearchTypeNumber {
			orderParts = append(orderParts,
				fmt.Sprintf("CASE WHEN %s%s = ? THEN 1 ELSE 0 END DESC",
					tableAliasWithDot, field.Name))
			orderArgs = append(orderArgs, query)
		}
	}

	queryWithSuffix := query + wildcardPattern
	for _, field := range config.Fields {
		if field.Type == infra.PostgresSearchTypeComposite || field.Type == infra.PostgresSearchTypeNumber {
			orderParts = append(orderParts,
				fmt.Sprintf("CASE WHEN %s%s ILIKE ? THEN 1 ELSE 0 END DESC",
					tableAliasWithDot, field.Name))
			orderArgs = append(orderArgs, queryWithSuffix)
		}
	}

	orderParts = append(orderParts, "rank DESC NULLS LAST")

	return orderParts, orderArgs
}

// Helper function for trigger generation
func BuildTSVectorUpdate(fields []infra.PostgresSearchableField) string {
	parts := make([]string, 0, len(fields))

	for _, field := range fields {
		dict := field.Dictionary
		if dict == "" {
			dict = "english"
			if field.Type == infra.PostgresSearchTypeComposite || field.Type == infra.PostgresSearchTypeNumber {
				dict = "simple"
			}
		}

		var partBuilder strings.Builder
		partBuilder.Grow(100) // Estimate space needed

		partBuilder.WriteString("setweight(to_tsvector('")
		partBuilder.WriteString(dict)
		partBuilder.WriteString("', COALESCE(")

		if field.Type == infra.PostgresSearchTypeEnum {
			partBuilder.WriteString("CAST(")
			partBuilder.WriteString(field.Name)
			partBuilder.WriteString(" AS text)")
		} else {
			partBuilder.WriteString(field.Name)
		}

		partBuilder.WriteString(", '')), '")
		partBuilder.WriteString(field.Weight)
		partBuilder.WriteString("')")

		parts = append(parts, partBuilder.String())
	}

	return strings.Join(parts, " || ")
}
