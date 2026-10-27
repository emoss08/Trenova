package listquery

import (
	"context"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	fuzzyThreshold     = "0.5"
	fuzzyMinWordLength = 4
	fuzzyMaxWords      = 4
	fuzzyThresholdSQL  = "SELECT set_config('pg_trgm.word_similarity_threshold', $1, true)"
)

var wordSimilarity = keel.DeclareFunc2[keel.Float64Expr, keel.StringValue, keel.StringValue](
	"word_similarity",
)

func SearchFuncs() []keel.DeclaredFunc {
	return []keel.DeclaredFunc{wordSimilarity}
}

type fuzzyMatch struct {
	where keel.Pred
	score keel.Float64Expr
}

func fuzzyWords(query string) []string {
	if hasWebsearchOperators(query) {
		return nil
	}

	words := make([]string, 0, fuzzyMaxWords)
	for _, word := range strings.FieldsFunc(strings.ToLower(query), wordBreak) {
		if len(words) == fuzzyMaxWords {
			break
		}
		if utf8.RuneCountInString(word) >= fuzzyMinWordLength && !slices.Contains(words, word) {
			words = append(words, word)
		}
	}

	return words
}

func wordBreak(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func (p *plan) fuzzy(filter *pagination.QueryOptions) (fuzzyMatch, bool) {
	if len(p.cat.fuzzy) == 0 {
		return fuzzyMatch{}, false
	}

	words := fuzzyWords(queryOf(filter))
	if len(words) == 0 {
		return fuzzyMatch{}, false
	}

	targets := p.fuzzyTargets()
	preds := make([]keel.Pred, 0, len(words))
	for _, word := range words {
		alternatives := make([]keel.Pred, 0, len(targets))
		for _, target := range targets {
			alternatives = append(alternatives, keel.Raw("? <% ?", keel.Text(word), target))
		}
		preds = append(preds, keel.Or(alternatives...))
	}

	phrase := keel.Text(strings.Join(words, " "))
	score := wordSimilarity.Call(phrase, targets[0])
	for _, target := range targets[1:] {
		score = score.Greatest(wordSimilarity.Call(phrase, target))
	}

	return fuzzyMatch{where: keel.And(preds...), score: score}, true
}

func (p *plan) fuzzyTargets() []keel.StringExpr {
	targets := make([]keel.StringExpr, 0, len(p.cat.fuzzy))
	for _, columns := range p.cat.fuzzy {
		expr := keel.DynOf(p.baseColumn(columns[0]))
		for _, index := range columns[1:] {
			expr = expr.Concat(keel.DynConst(" ")).Concat(keel.DynOf(p.baseColumn(index)))
		}
		targets = append(targets, keel.StringOf(expr))
	}

	return targets
}

func useFuzzyThreshold(ctx context.Context, db *keel.DB) error {
	_, err := keel.Exec(ctx, db, fuzzyThresholdSQL, fuzzyThreshold)
	return err
}

func queryOf(filter *pagination.QueryOptions) string {
	if filter == nil {
		return ""
	}

	return filter.Query
}

func fuzzyList[T any](
	ctx context.Context,
	db *keel.DB,
	p *plan,
	read *fuzzyRead[T],
) (*pagination.ListResult[*T], error) {
	if err := useFuzzyThreshold(ctx, db); err != nil {
		return nil, err
	}

	rows, counted, err := listRows(
		p,
		read.unmatched.Where(read.match.where),
		read.req,
		read.match.score,
	)
	if err != nil {
		return nil, err
	}

	result, err := listAndCount(ctx, db, rows, counted)
	if err != nil {
		return nil, err
	}
	result.Fuzzy = result.Total > 0

	return result, nil
}

type fuzzyRead[T any] struct {
	match     fuzzyMatch
	unmatched *keel.Query[*T]
	req       *Request[T]
}

func fuzzyConnection[T any](
	ctx context.Context,
	db *keel.DB,
	read *fuzzyRead[T],
	page keysetPage,
) (*pagination.CursorListResult[*T], error) {
	if err := useFuzzyThreshold(ctx, db); err != nil {
		return nil, err
	}

	result, err := connectionPage(ctx, db, read.unmatched.Where(read.match.where), read.req, page)
	if err != nil {
		return nil, err
	}
	result.Fuzzy = len(result.Items) > 0

	return result, nil
}
