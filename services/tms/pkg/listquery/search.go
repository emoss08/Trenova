package listquery

import (
	"strconv"

	"github.com/emoss08/trenova/pkg/searchtext"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

func (p *plan) search(query string) keel.Pred {
	if query == "" {
		return nil
	}

	if p.cat.Vector >= 0 {
		return searchtext.MatchVector(p.vector(), query)
	}

	return anyTerm(query, p.textSearch)
}

func (p *plan) textSearch(query string) keel.Pred {
	if len(p.cat.Search) > 0 {
		return p.configuredSearch(query)
	}

	return p.columnSearch(query)
}

// anyTerm is match for the query, or for any of the identifiers it lists.
func anyTerm(query string, match func(string) keel.Pred) keel.Pred {
	terms := searchtext.IdentifierTerms(query)
	if terms == nil {
		return match(query)
	}
	alternatives := make([]keel.Pred, 0, len(terms))
	for _, term := range terms {
		if pred := match(term); pred != nil {
			alternatives = append(alternatives, pred)
		}
	}
	if len(alternatives) == 0 {
		return nil
	}

	return keel.Or(alternatives...)
}

func (p *plan) rank(query string) keel.Expr {
	if query == "" || p.cat.Vector < 0 {
		return nil
	}

	return searchtext.RankVector(p.vector(), query)
}

func (p *plan) vector() keel.TSVectorCol {
	return keel.TSVectorCol{Col: p.baseColumn(p.cat.Vector)}
}

func (p *plan) configuredSearch(query string) keel.Pred {
	alternatives := make([]keel.Pred, 0, len(p.cat.Search))
	_, numeric := strconv.ParseFloat(query, 64)

	for _, sc := range p.cat.Search {
		expr := keel.DynOf(p.baseColumn(sc.Index))
		switch sc.Kind {
		case domaintypes.FieldTypeText, domaintypes.FieldTypeComposite:
			alternatives = append(
				alternatives,
				textOf(expr, &p.cat.Table.Columns[sc.Index]).ContainsFold(query),
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
	if len(p.cat.Strings) == 0 {
		return nil
	}

	alternatives := make([]keel.Pred, 0, len(p.cat.Strings))
	for _, index := range p.cat.Strings {
		alternatives = append(alternatives, keel.DynOf(p.baseColumn(index)).ContainsFold(query))
	}

	return keel.Or(alternatives...)
}
