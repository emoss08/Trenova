package agentquerytoolservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
)

const nearMissProbes = 4

type nearMissRequest struct {
	params   *serviceports.QueryToolParams
	opts     *pagination.QueryOptions
	gate     *fieldGate
	criteria *filtercatalog.Criteria
	filters  []domaintypes.FieldFilter
}

func (t *listTool) nearMiss(ctx context.Context, req *nearMissRequest) string {
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
		rows, err := t.fetchRows(ctx, req.params, &probe, req.gate, req.criteria)
		if err != nil {
			return false, false
		}

		return len(rows) > 0, true
	}

	entity := t.spec.entityPlural
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

	if len(loosened) == 0 {
		return fmt.Sprintf(
			"No %s matched %s, and dropping any one of these still finds none: together they "+
				"rule everything out. Other %s exist, so loosen more than one, or ask the person "+
				"what they meant.",
			entity, applied, entity,
		)
	}

	return fmt.Sprintf(
		"No %s matched %s. Dropping just one of these finds some: %s. Loosen that one rather "+
			"than trying new values one by one.",
		entity, applied, strings.Join(loosened, "; "),
	)
}

func (t *listTool) termLabels(
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
