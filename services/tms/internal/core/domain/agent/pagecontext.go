package agent

import (
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxPagePathLength   = 500
	maxPageTitleLength  = 200
	maxPageEntityLength = 100

	MaxPageViewQueryLength   = 200
	MaxPageViewFieldFilters  = 20
	MaxPageViewFilterGroups  = 5
	MaxPageViewSortFields    = 5
	MaxPageViewSelectionIDs  = 25
	MaxPageViewKPIs          = 12
	MaxPageViewColumns       = 40
	MaxPageViewFieldLength   = 100
	MaxPageViewKPILabelLen   = 80
	MaxPageViewKPIValueLen   = 60
	MaxPageViewFilterValueLn = 500
)

// PageContext is what the person was looking at when they asked. It is data
// about the page, never an instruction, and only the listed record kinds are
// accepted so a client cannot smuggle arbitrary text in as a "record".
type PageContext struct {
	Path       string `json:"path"`
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
	Title      string `json:"title"`
	// View is what a list page was showing: its filters, sort, selection and
	// figures. Nil for a page that is not a table.
	View *PageView `json:"view,omitempty"`
}

// PageView is a table the person had in front of them, expressed in the same
// shape the list tools take, so "these rows" can be re-run as a query rather
// than guessed at from the words.
type PageView struct {
	Resource       string                    `json:"resource"`
	Query          string                    `json:"query,omitempty"`
	FieldFilters   []domaintypes.FieldFilter `json:"fieldFilters,omitempty"`
	FilterGroups   []domaintypes.FilterGroup `json:"filterGroups,omitempty"`
	Sort           []domaintypes.SortField   `json:"sort,omitempty"`
	Selection      *PageSelection            `json:"selection,omitempty"`
	KPIs           []PageKPI                 `json:"kpis,omitempty"`
	VisibleColumns []string                  `json:"visibleColumns,omitempty"`
	// RowCount is how many rows the table reported in total; nil when the
	// table had not counted yet.
	RowCount *int `json:"rowCount,omitempty"`
}

// PageSelection is the rows the person had checked: how many, and the first
// few identifiers so a question about "these" has something to look up.
type PageSelection struct {
	Count int      `json:"count"`
	IDs   []string `json:"ids,omitempty"`
}

// PageKPI is one figure shown above a table, as the person read it.
type PageKPI struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Sub   string `json:"sub,omitempty"`
}

var knownPageEntityTypes = map[string]struct{}{
	"shipment":             {},
	"shipment_move":        {},
	"order":                {},
	"worker":               {},
	"tractor":              {},
	"trailer":              {},
	"customer":             {},
	"carrier":              {},
	"location":             {},
	"billing_queue_item":   {},
	"invoice":              {},
	"document":             {},
	"rate_matrix":          {},
	"agent_definition":     {},
	"insight":              {},
	"report":               {},
	"report_run":           {},
	"dashboard":            {},
	"driver_settlement":    {},
	"detention_occurrence": {},
	"carrier_intel_event":  {},
	"agent_run":            {},
	"agent_proposal":       {},
	"service_failure":      {},
	"worker_credential":    {},
	"edi_inbound_file":     {},
	"weather_alert":        {},
}

func KnownPageEntityTypes() []string {
	kinds := make([]string, 0, len(knownPageEntityTypes))
	for kind := range knownPageEntityTypes {
		kinds = append(kinds, kind)
	}

	return kinds
}

func IsKnownPageEntityType(kind string) bool {
	_, ok := knownPageEntityTypes[kind]
	return ok
}

var pageResources = permission.NewRegistry()

// IsKnownPageResource reports whether a table's resource names something the
// permission registry knows, which is the set of resources a list page can
// actually show.
func IsKnownPageResource(resource string) bool {
	return resource != "" && pageResources.HasResource(resource)
}

// Validate reports field errors under the given prefix (for example
// "context"), so a client can show them next to the request that carried them.
func (p *PageContext) Validate(prefix string, multiErr *errortypes.MultiError) {
	field := fieldNamer(prefix)

	path := strings.TrimSpace(p.Path)
	switch {
	case path == "":
		multiErr.Add(field("path"), errortypes.ErrRequired, "Path is required")
	case !strings.HasPrefix(path, "/"):
		multiErr.Add(field("path"), errortypes.ErrInvalid, "Path must be an application path")
	case len(path) > maxPagePathLength:
		multiErr.Add(field("path"), errortypes.ErrInvalid, "Path is too long")
	}

	if len(p.Title) > maxPageTitleLength {
		multiErr.Add(field("title"), errortypes.ErrInvalid, "Title is too long")
	}

	if p.EntityType != "" && !IsKnownPageEntityType(p.EntityType) {
		multiErr.Add(field("entityType"), errortypes.ErrInvalid, "Unknown record type")
	}

	if len(p.EntityID) > maxPageEntityLength {
		multiErr.Add(field("entityId"), errortypes.ErrInvalid, "Record identifier is too long")
	}

	if p.EntityID != "" && p.EntityType == "" {
		multiErr.Add(field("entityType"), errortypes.ErrRequired, "Record type is required with a record identifier")
	}

	if p.View != nil {
		p.View.Validate(field("view"), multiErr)
	}
}

// Validate bounds a table view to what a page could have shown: a known
// resource, operators the query layer accepts, and lists short enough that a
// client cannot use the view as a channel for arbitrary text.
func (v *PageView) Validate(prefix string, multiErr *errortypes.MultiError) {
	field := fieldNamer(prefix)

	if !IsKnownPageResource(strings.TrimSpace(v.Resource)) {
		multiErr.Add(field("resource"), errortypes.ErrInvalid, "Unknown table resource")
	}

	if len(v.Query) > MaxPageViewQueryLength {
		multiErr.Add(field("query"), errortypes.ErrInvalid, "Search text is too long")
	}

	if len(v.FieldFilters) > MaxPageViewFieldFilters {
		multiErr.Add(field("fieldFilters"), errortypes.ErrInvalid, "Too many filters")
	} else {
		validatePageFilters(field("fieldFilters"), v.FieldFilters, multiErr)
	}

	if len(v.FilterGroups) > MaxPageViewFilterGroups {
		multiErr.Add(field("filterGroups"), errortypes.ErrInvalid, "Too many filter groups")
	} else {
		for i, group := range v.FilterGroups {
			groupField := field("filterGroups") + "[" + strconv.Itoa(i) + "].filters"
			if len(group.Filters) > MaxPageViewFieldFilters {
				multiErr.Add(groupField, errortypes.ErrInvalid, "Too many filters")
				continue
			}
			validatePageFilters(groupField, group.Filters, multiErr)
		}
	}

	if len(v.Sort) > MaxPageViewSortFields {
		multiErr.Add(field("sort"), errortypes.ErrInvalid, "Too many sort fields")
	} else {
		for i, sort := range v.Sort {
			sortField := field("sort") + "[" + strconv.Itoa(i) + "]"
			if !validFieldName(sort.Field) {
				multiErr.Add(sortField+".field", errortypes.ErrInvalid, "Sort field is invalid")
			}
			if !sort.Direction.IsValid() {
				multiErr.Add(sortField+".direction", errortypes.ErrInvalid, "Sort direction is invalid")
			}
		}
	}

	if v.Selection != nil {
		if v.Selection.Count < 0 {
			multiErr.Add(field("selection.count"), errortypes.ErrInvalid, "Selection count cannot be negative")
		}
		if len(v.Selection.IDs) > MaxPageViewSelectionIDs {
			multiErr.Add(field("selection.ids"), errortypes.ErrInvalid, "Too many selected identifiers")
		} else {
			for i, id := range v.Selection.IDs {
				if _, err := pulid.Parse(id); err != nil {
					multiErr.Add(
						field("selection.ids")+"["+strconv.Itoa(i)+"]",
						errortypes.ErrInvalid,
						"Selected identifier is invalid",
					)
				}
			}
		}
		if v.Selection.Count < len(v.Selection.IDs) {
			multiErr.Add(field("selection.count"), errortypes.ErrInvalid, "Selection count is below the identifiers sent")
		}
	}

	if len(v.KPIs) > MaxPageViewKPIs {
		multiErr.Add(field("kpis"), errortypes.ErrInvalid, "Too many figures")
	} else {
		for i, kpi := range v.KPIs {
			kpiField := field("kpis") + "[" + strconv.Itoa(i) + "]"
			if strings.TrimSpace(kpi.Label) == "" {
				multiErr.Add(kpiField+".label", errortypes.ErrRequired, "Figure label is required")
			}
			if len(kpi.Label) > MaxPageViewKPILabelLen {
				multiErr.Add(kpiField+".label", errortypes.ErrInvalid, "Figure label is too long")
			}
			if len(kpi.Value) > MaxPageViewKPIValueLen {
				multiErr.Add(kpiField+".value", errortypes.ErrInvalid, "Figure value is too long")
			}
			if len(kpi.Sub) > MaxPageViewKPILabelLen {
				multiErr.Add(kpiField+".sub", errortypes.ErrInvalid, "Figure note is too long")
			}
		}
	}

	if len(v.VisibleColumns) > MaxPageViewColumns {
		multiErr.Add(field("visibleColumns"), errortypes.ErrInvalid, "Too many columns")
	} else {
		for i, column := range v.VisibleColumns {
			if !validFieldName(column) {
				multiErr.Add(field("visibleColumns")+"["+strconv.Itoa(i)+"]", errortypes.ErrInvalid, "Column name is invalid")
			}
		}
	}

	if v.RowCount != nil && *v.RowCount < 0 {
		multiErr.Add(field("rowCount"), errortypes.ErrInvalid, "Row count cannot be negative")
	}
}

func validatePageFilters(prefix string, filters []domaintypes.FieldFilter, multiErr *errortypes.MultiError) {
	for i, filter := range filters {
		filterField := prefix + "[" + strconv.Itoa(i) + "]"
		if !validFieldName(filter.Field) {
			multiErr.Add(filterField+".field", errortypes.ErrInvalid, "Filter field is invalid")
		}
		if !filter.Operator.IsValid() {
			multiErr.Add(filterField+".operator", errortypes.ErrInvalid, "Filter operator is invalid")
		}
		if len(FormatFilterValue(filter.Value)) > MaxPageViewFilterValueLn {
			multiErr.Add(filterField+".value", errortypes.ErrInvalid, "Filter value is too long")
		}
	}
}

// validFieldName accepts the dotted, snake or camel identifiers the query
// layer resolves, and nothing that could read as prose in a prompt.
func validFieldName(name string) bool {
	if name == "" || len(name) > MaxPageViewFieldLength {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '.':
		default:
			return false
		}
	}

	return true
}

// FormatFilterValue writes a filter's value the way a person would read it
// in a chip: scalars as they are, lists comma-separated, and nothing else.
func FormatFilterValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []string:
		return strings.Join(v, ", ")
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, FormatFilterValue(item))
		}

		return strings.Join(parts, ", ")
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, key+"="+FormatFilterValue(v[key]))
		}

		return strings.Join(parts, " ")
	case bool:
		if v {
			return "true"
		}

		return "false"
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

// Normalized trims what a person cannot see and drops a record reference the
// validator would reject, so the stored context is exactly what was used.
func (p *PageContext) Normalized() *PageContext {
	if p == nil {
		return nil
	}

	return &PageContext{
		Path:       strings.TrimSpace(p.Path),
		EntityType: strings.TrimSpace(p.EntityType),
		EntityID:   strings.TrimSpace(p.EntityID),
		Title:      strings.TrimSpace(p.Title),
		View:       p.View.normalized(),
	}
}

func (v *PageView) normalized() *PageView {
	if v == nil {
		return nil
	}

	out := &PageView{
		Resource:       strings.TrimSpace(v.Resource),
		Query:          strings.TrimSpace(v.Query),
		FieldFilters:   v.FieldFilters,
		FilterGroups:   v.FilterGroups,
		Sort:           v.Sort,
		VisibleColumns: v.VisibleColumns,
		RowCount:       v.RowCount,
	}
	if v.Selection != nil && (v.Selection.Count > 0 || len(v.Selection.IDs) > 0) {
		out.Selection = &PageSelection{Count: v.Selection.Count, IDs: v.Selection.IDs}
	}
	if len(v.KPIs) > 0 {
		out.KPIs = make([]PageKPI, 0, len(v.KPIs))
		for _, kpi := range v.KPIs {
			out.KPIs = append(out.KPIs, PageKPI{
				Label: strings.TrimSpace(kpi.Label),
				Value: strings.TrimSpace(kpi.Value),
				Sub:   strings.TrimSpace(kpi.Sub),
			})
		}
	}

	return out
}

// Empty reports whether the view says nothing a prompt should carry.
func (v *PageView) Empty() bool {
	return v == nil || (v.Query == "" && len(v.FieldFilters) == 0 && len(v.FilterGroups) == 0 &&
		len(v.Sort) == 0 && v.Selection == nil && len(v.KPIs) == 0 && v.RowCount == nil)
}

func fieldNamer(prefix string) func(string) string {
	return func(name string) string {
		if prefix == "" {
			return name
		}

		return prefix + "." + name
	}
}
