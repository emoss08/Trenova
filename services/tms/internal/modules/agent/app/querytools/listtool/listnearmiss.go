package listtool

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
)

const nearMissProbes = 4

type nearMissRequest struct {
	params   *agenttoolports.QueryToolParams
	opts     *pagination.QueryOptions
	gate     *fieldgate.Gate
	criteria *filtercatalog.Criteria
	filters  []domaintypes.FieldFilter
}

func (t *Tool) nearMiss(ctx context.Context, req *nearMissRequest) string {
	query := req.opts.Query
	if query == "" && len(req.filters) == 0 {
		return ""
	}

	found := func(text string, filters []domaintypes.FieldFilter) (bool, bool) {
		probe := *req.opts
		probe.Query = text
		probe.FieldFilters = filters
		probe.Sort = nil
		probe.Pagination = pagination.Info{Limit: 1}
		fetched, err := t.fetchRows(ctx, &listFetch{
			params:   req.params,
			opts:     &probe,
			cursor:   pagination.CursorInfo{Limit: 1},
			gate:     req.gate,
			criteria: req.criteria,
		})
		if err != nil {
			return false, false
		}

		return len(fetched.Rows) > 0, true
	}

	entity := t.Spec.EntityPlural
	applied := strings.Join(req.criteria.Terms(), " and ")

	anything, ok := found("", nil)
	if !ok {
		return ""
	}
	if !anything {
		return fmt.Sprintf(
			"There are no %s you can see in this organization at all, so no other wording or "+
				"filter will find one. Tell the person so rather than searching again.",
			entity,
		)
	}

	labels := t.termLabels(req.criteria, query, req.filters)
	loosened := make([]string, 0, len(labels))
	if query != "" {
		if hit, probed := found("", req.filters); probed && hit {
			loosened = append(loosened, labels[0])
		}
	}
	offset := 0
	if query != "" {
		offset = 1
	}
	for idx := range req.filters {
		if idx >= nearMissProbes {
			break
		}
		without := slices.Delete(slices.Clone(req.filters), idx, idx+1)
		if hit, probed := found(query, without); probed && hit {
			loosened = append(loosened, labels[offset+idx])
		}
	}

	return nearMissNote(entity, applied, loosened)
}

// nearMissNote says what ruled every row out. A term the person asked for is
// the question, not a guess to loosen: "loosen that one" read as an
// instruction made an agent drop "delivered today" from "what was delivered
// today?" and answer a different question, and then propose keeping that as
// a lesson. None matching is an answer.
func nearMissNote(entity, applied string, loosened []string) string {
	if len(loosened) == 0 {
		return fmt.Sprintf(
			"No %s matched %s, and dropping any one of these still finds none: together they "+
				"rule everything out. Other %s exist. If the person asked for all of these, "+
				"none matching is the answer: say so. Otherwise loosen more than one, or ask "+
				"the person what they meant.",
			entity, applied, entity,
		)
	}

	return fmt.Sprintf(
		"No %s matched %s. Without just one of these, some do: %s. If the person asked for "+
			"it, none matching is the answer: say so rather than searching without it. Loosen "+
			"it only when you chose it yourself, and loosen it rather than trying new values "+
			"one by one.",
		entity, applied, strings.Join(loosened, "; "),
	)
}

func (t *Tool) termLabels(
	criteria *filtercatalog.Criteria,
	query string,
	filters []domaintypes.FieldFilter,
) []string {
	expected := len(filters)
	if query != "" {
		expected++
	}

	terms := criteria.Terms()
	if len(terms) == expected {
		return terms
	}

	labels := make([]string, 0, expected)
	if query != "" {
		labels = append(labels, fmt.Sprintf("text matching %q", query))
	}
	for _, filter := range filters {
		labels = append(labels, "the filter on "+filter.Field)
	}

	return labels
}
