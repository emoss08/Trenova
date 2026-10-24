package listquery

import (
	"strconv"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const minPrefixSearchLength = 2

func (p *plan) search(query string) keel.Pred {
	if query == "" {
		return nil
	}

	if p.cat.vector >= 0 {
		return p.vector().Match(query, searchOptions(query)...)
	}

	if len(p.cat.search) > 0 {
		return p.configuredSearch(query)
	}

	return p.columnSearch(query)
}

func (p *plan) rank(query string) keel.Expr {
	if query == "" || p.cat.vector < 0 {
		return nil
	}

	return p.vector().Rank(query, searchOptions(query)...)
}

func (p *plan) vector() keel.TSVectorCol {
	return keel.TSVectorCol{Col: p.baseColumn(p.cat.vector)}
}

func searchOptions(query string) []keel.SearchOption {
	modes := keel.SearchModes(keel.WebSearch)
	if !hasWebsearchOperators(query) && len(query) >= minPrefixSearchLength {
		modes = keel.SearchModes(keel.WebSearch, keel.Prefix)
	}

	return []keel.SearchOption{keel.SearchConfig("simple", "english"), modes}
}

func hasWebsearchOperators(query string) bool {
	if strings.Contains(query, "\"") {
		return true
	}

	if strings.Contains(query, " OR ") || strings.Contains(query, " or ") {
		return true
	}

	for word := range strings.FieldsSeq(query) {
		if strings.HasPrefix(word, "-") && len(word) > 1 {
			return true
		}
	}

	return false
}

func (p *plan) configuredSearch(query string) keel.Pred {
	alternatives := make([]keel.Pred, 0, len(p.cat.search))
	_, numeric := strconv.ParseFloat(query, 64)

	for _, sc := range p.cat.search {
		expr := keel.DynOf(p.baseColumn(sc.index))
		switch sc.kind {
		case domaintypes.FieldTypeText, domaintypes.FieldTypeComposite:
			alternatives = append(
				alternatives,
				textOf(expr, &p.cat.table.Columns[sc.index]).ContainsFold(query),
			)
		case domaintypes.FieldTypeEnum:
			alternatives = append(alternatives, expr.Cast(keel.TypeString).ContainsFold(query))
		case domaintypes.FieldTypeNumber:
			if numeric == nil {
				alternatives = append(alternatives, expr.Cast(keel.TypeString).Eq(query))
			}
		case domaintypes.FieldTypeBoolean, domaintypes.FieldTypeDate:
		}
	}

	if len(alternatives) == 0 {
		return nil
	}

	return keel.Or(alternatives...)
}

func textOf(expr keel.DynExpr, col *schema.Column) keel.DynExpr {
	if len(col.Enum) > 0 || col.Kind != schema.KindString {
		return expr.Cast(keel.TypeString)
	}

	return expr
}

func (p *plan) columnSearch(query string) keel.Pred {
	if len(p.cat.strings) == 0 {
		return nil
	}

	alternatives := make([]keel.Pred, 0, len(p.cat.strings))
	for _, index := range p.cat.strings {
		alternatives = append(alternatives, keel.DynOf(p.baseColumn(index)).ContainsFold(query))
	}

	return keel.Or(alternatives...)
}
