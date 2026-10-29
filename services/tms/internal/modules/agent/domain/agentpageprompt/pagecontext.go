package agentpageprompt

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	pageContextOpenTag  = "<page_context>"
	pageContextCloseTag = "</page_context>"
	pageViewOpenTag     = "<page_view>"
	pageViewCloseTag    = "</page_view>"
	pageDraftOpenTag    = "<page_draft>"
	pageDraftCloseTag   = "</page_draft>"
)

func DescribeSurface(surface agentpagecontext.Surface, guide *agentturn.RuntimePage) string {
	switch surface {
	case agentpagecontext.SurfaceDesk:
		line := "- Where the person is talking to you: the Desk, Trenova's full-page workspace " +
			"for conversations with its AI agents, at /desk. They are in the Desk now, not on " +
			"another page of Trenova."
		if guide == nil || strings.TrimSpace(guide.Summary) == "" {
			return line
		}

		return line + " What the Desk is for:\n" + strings.TrimSpace(guide.Summary)
	case agentpagecontext.SurfaceAssistant:
		return "- Where the person is talking to you: the assistant panel, which opens over " +
			"whichever page of Trenova they are on and can float in a corner, dock to the side " +
			"or fill the screen. Open in Desk carries the conversation over to the Desk, " +
			"Trenova's full-page workspace for conversations with its AI agents, at /desk."
	case agentpagecontext.SurfacePalette:
		return "- Where the person is talking to you: the command palette (Ctrl+K, or ⌘K on a " +
			"Mac), the box that opens over whichever page of Trenova they are on to find " +
			"records, pages and commands across the whole organization. A question typed into " +
			"it comes to you, and your answer appears in the palette under it, so keep it " +
			"short. The question is about what it names, anywhere in Trenova, not about the " +
			"page under the palette: look up the records, people, places and figures it names " +
			"with your tools, and answer about that page only when the question points at it " +
			"(\"this\", \"here\", \"these\"). The records the palette's search found that the " +
			"question names, and a record they picked in the palette before asking, come as " +
			"mentions: read those first. Continue in Desk carries " +
			"the conversation over to the Desk, Trenova's full-page workspace for conversations " +
			"with its AI agents, at /desk."
	default:
		return ""
	}
}

func DescribePage(
	page *agentturn.PageContext,
	guide *agentturn.RuntimePage,
	surface agentpagecontext.Surface,
) string {
	var builder strings.Builder
	switch surface { //nolint:exhaustive // the assistant panel and no named surface both sit on the page
	case agentpagecontext.SurfaceDesk:
		builder.WriteString("- The page of Trenova the person had open before the Desk:\n")
	case agentpagecontext.SurfacePalette:
		builder.WriteString("- The page of Trenova open under the command palette. It is where " +
			"the person was when they asked, not what they asked about:\n")
	default:
		builder.WriteString("- What the person is looking at right now:\n")
	}
	builder.WriteString(pageContextOpenTag)
	builder.WriteString("\npath: ")
	builder.WriteString(stringutils.FenceText(page.Path, pageContextCloseTag))
	if guide != nil {
		builder.WriteString("\npage: ")
		builder.WriteString(stringutils.FenceText(guide.Name, pageContextCloseTag))
		if guide.Location != "" && guide.Location != guide.Name {
			builder.WriteString(" (")
			builder.WriteString(stringutils.FenceText(guide.Location, pageContextCloseTag))
			builder.WriteString(")")
		}
		if guide.Summary != "" {
			builder.WriteString("\nabout: ")
			builder.WriteString(stringutils.FenceText(guide.Summary, pageContextCloseTag))
		}
	}
	if page.EntityType != "" {
		builder.WriteString("\nrecord: ")
		builder.WriteString(stringutils.FenceText(page.EntityType, pageContextCloseTag))
		if page.EntityID != "" {
			builder.WriteString(" ")
			builder.WriteString(stringutils.FenceText(page.EntityID, pageContextCloseTag))
		}
	}
	if title := strings.TrimSpace(page.Title); title != "" {
		builder.WriteString("\ntitle: ")
		builder.WriteString(stringutils.FenceText(title, pageContextCloseTag))
	}
	builder.WriteString("\n")
	builder.WriteString(pageContextCloseTag)

	return builder.String()
}

// DescribePageView writes the table the person had in front of them in the
// shape the list tools take, and says so: a question about "these rows" is
// answered by running the same query, not by reading the filters as facts.
// Under the palette the table is only where the person was, so its query is
// run only for a question that points at it.
func DescribePageView(view *agentpagecontext.PageView, surface agentpagecontext.Surface) string {
	var builder strings.Builder
	if surface == agentpagecontext.SurfacePalette {
		builder.WriteString(
			"- The table on that page, as the person has it filtered. Only when the " +
				"question points at these rows, call the matching list tool with the same " +
				"filters rather than describing the filters themselves:\n",
		)
	} else {
		builder.WriteString(
			"- The table on that page, as the person has it filtered. To answer about " +
				"these rows, call the matching list tool with the same filters rather than " +
				"describing the filters themselves:\n",
		)
	}
	builder.WriteString(pageViewOpenTag)
	builder.WriteString("\nresource: ")
	builder.WriteString(stringutils.FenceText(view.Resource, pageViewCloseTag))
	if view.Query != "" {
		builder.WriteString("\nsearch: ")
		builder.WriteString(stringutils.FenceText(view.Query, pageViewCloseTag))
	}
	for _, filter := range view.FieldFilters {
		builder.WriteString("\nfilter: ")
		builder.WriteString(describeFilter(filter))
	}
	for i, group := range view.FilterGroups {
		for _, filter := range group.Filters {
			builder.WriteString("\nfilter (group ")
			builder.WriteString(strconv.Itoa(i + 1))
			builder.WriteString(", any of): ")
			builder.WriteString(describeFilter(filter))
		}
	}
	for _, sort := range view.Sort {
		builder.WriteString("\nsort: ")
		builder.WriteString(stringutils.FenceText(sort.Field, pageViewCloseTag))
		builder.WriteString(" ")
		builder.WriteString(string(sort.Direction))
	}
	if view.RowCount != nil {
		builder.WriteString("\nrows matching: ")
		if view.RowCountEstimated {
			builder.WriteString("about ")
		}
		builder.WriteString(strconv.Itoa(*view.RowCount))
	}
	if view.Selection != nil && view.Selection.Count > 0 {
		builder.WriteString("\nselected: ")
		builder.WriteString(strconv.Itoa(view.Selection.Count))
		if len(view.Selection.IDs) > 0 {
			builder.WriteString(" (ids: ")
			builder.WriteString(strings.Join(view.Selection.IDs, ", "))
			if view.Selection.Count > len(view.Selection.IDs) {
				builder.WriteString(", …")
			}
			builder.WriteString(")")
		}
	}
	for _, kpi := range view.KPIs {
		builder.WriteString("\nfigure: ")
		builder.WriteString(stringutils.FenceText(kpi.Label, pageViewCloseTag))
		builder.WriteString(" = ")
		builder.WriteString(stringutils.FenceText(kpi.Value, pageViewCloseTag))
		if kpi.Sub != "" {
			builder.WriteString(" (")
			builder.WriteString(stringutils.FenceText(kpi.Sub, pageViewCloseTag))
			builder.WriteString(")")
		}
	}
	if len(view.VisibleColumns) > 0 {
		builder.WriteString("\ncolumns shown: ")
		builder.WriteString(strings.Join(view.VisibleColumns, ", "))
	}
	builder.WriteString("\n")
	builder.WriteString(pageViewCloseTag)

	return builder.String()
}

func describeFilter(filter domaintypes.FieldFilter) string {
	line := stringutils.FenceText(
		filter.Field,
		pageViewCloseTag,
	) + " " + string(
		filter.Operator,
	)
	if value := agentpagecontext.FormatFilterValue(filter.Value); value != "" {
		line += " " + stringutils.FenceText(value, pageViewCloseTag)
	}

	return line
}
