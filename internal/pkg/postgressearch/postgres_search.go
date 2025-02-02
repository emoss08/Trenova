package postgressearch

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/uptrace/bun"
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

	// Build the tsquery string
	tsqueryStr := strings.Join(terms, " | ")

	// Select all fields from the main table first
	q = q.ColumnExpr(config.TableAlias + ".*")

	// Add ts_rank as an additional column
	rankExpr := fmt.Sprintf(
		`ts_rank(%s.search_vector, to_tsquery('simple', ?)) AS rank`,
		config.TableAlias,
	)
	q = q.ColumnExpr(rankExpr, tsqueryStr+":*")

	// Build search conditions
	var whereParts []string
	var whereArgs []any

	// Add full-text search condition
	whereParts = append(whereParts,
		fmt.Sprintf("%s.search_vector @@ to_tsquery('simple', ?)", config.TableAlias))
	whereArgs = append(whereArgs, tsqueryStr+":*")

	if config.UsePartialMatch {
		for _, field := range config.Fields {
			switch field.Type {
			case infra.PostgresSearchTypeComposite, infra.PostgresSearchTypeNumber:
				// Use ILIKE for pattern matching
				whereParts = append(whereParts,
					fmt.Sprintf("%s.%s ILIKE ?", config.TableAlias, field.Name))
				whereArgs = append(whereArgs, "%"+query+"%")

			case infra.PostgresSearchTypeText:
				// Use both ILIKE and similarity for text fields
				whereParts = append(whereParts,
					fmt.Sprintf("(%s.%s ILIKE ? OR similarity(%s.%s, ?) > 0.3)",
						config.TableAlias, field.Name,
						config.TableAlias, field.Name))
				whereArgs = append(whereArgs, "%"+query+"%", query)

			case infra.PostgresSearchTypeEnum:
				// Exact matching for enums
				whereParts = append(whereParts,
					fmt.Sprintf("%s.%s::text = ?", config.TableAlias, field.Name))
				whereArgs = append(whereArgs, query)
			}
		}
	}

	searchCond := "(" + strings.Join(whereParts, " OR ") + ")"
	q = q.Where(searchCond, whereArgs...)

	// Build ordering expression
	var orderParts []string
	var orderArgs []any

	// Order by exact matches first
	for _, field := range config.Fields {
		if field.Type == infra.PostgresSearchTypeComposite || field.Type == infra.PostgresSearchTypeNumber {
			orderParts = append(orderParts,
				fmt.Sprintf("CASE WHEN %s.%s = ? THEN 1 ELSE 0 END DESC",
					config.TableAlias, field.Name))
			orderArgs = append(orderArgs, query)
		}
	}

	// Then order by prefix matches
	for _, field := range config.Fields {
		if field.Type == infra.PostgresSearchTypeComposite || field.Type == infra.PostgresSearchTypeNumber {
			orderParts = append(orderParts,
				fmt.Sprintf("CASE WHEN %s.%s ILIKE ? THEN 1 ELSE 0 END DESC",
					config.TableAlias, field.Name))
			orderArgs = append(orderArgs, query+"%")
		}
	}

	// Finally order by rank
	orderParts = append(orderParts, "rank DESC NULLS LAST")

	// Apply ordering
	for i, orderPart := range orderParts {
		if i < len(orderArgs) {
			q = q.OrderExpr(orderPart, orderArgs[i])
		} else {
			q = q.OrderExpr(orderPart)
		}
	}

	return q
}

// Helper function for trigger generation
func BuildTSVectorUpdate(fields []infra.PostgresSearchableField) string {
	var parts []string

	for _, field := range fields {
		dict := field.Dictionary
		if dict == "" {
			dict = "english"
			if field.Type == infra.PostgresSearchTypeComposite || field.Type == infra.PostgresSearchTypeNumber {
				dict = "simple"
			}
		}

		part := fmt.Sprintf("setweight(to_tsvector('%s', COALESCE(", dict)
		if field.Type == infra.PostgresSearchTypeEnum {
			part += "CAST(" + field.Name + " AS text)"
		} else {
			part += field.Name
		}
		part += ", '')), '" + field.Weight + "')"
		parts = append(parts, part)
	}

	return strings.Join(parts, " || ")
}
