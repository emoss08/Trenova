package boardrecords

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const ExampleNumber = "A-1001"

const (
	recordsKey     = "records"
	titleKey       = "title"
	forKey         = "for"
	recordKey      = "record"
	minLabelRunes  = 4
	maxNamedInNote = 3
	maxDepth       = 6
)

type Read struct {
	refs    []agentpagecontext.EntityRef
	pages   []listPage
	parents map[string]string
	turn    map[string]bool
}

type listPage struct {
	ids   map[string]bool
	total int
	more  bool
	table pulid.ID
}

type Swap struct {
	From agentpagecontext.EntityRef
	To   agentpagecontext.EntityRef
}

var numberedBy = []struct {
	kind     recordkind.Kind
	idKey    string
	labelKey string
}{
	{
		kind:     recordkind.Kind(permission.ResourceShipment),
		idKey:    "shipmentId",
		labelKey: "proNumber",
	},
	{
		kind:     recordkind.Kind(permission.ResourceInvoice),
		idKey:    "invoiceId",
		labelKey: "invoiceNumber",
	},
}

func Of(results []string) Read {
	read := Read{
		refs:    make([]agentpagecontext.EntityRef, 0, 16),
		parents: make(map[string]string, 4),
	}
	seen := make(map[string]struct{}, 16)
	keep := func(ref agentpagecontext.EntityRef) {
		if !boardregistry.Catalog().Supports(recordkind.Kind(ref.Type)) {
			return
		}
		if _, dup := seen[ref.ID]; dup {
			return
		}
		seen[ref.ID] = struct{}{}
		ref.Label = strings.TrimSpace(ref.Label)
		read.refs = append(read.refs, ref)
	}
	for _, content := range results {
		page := listPage{ids: make(map[string]bool, 8)}
		keepOnPage := func(ref agentpagecontext.EntityRef) {
			page.ids[ref.ID] = true
			keep(ref)
		}
		for _, ref := range toolresult.ReturnedRecords(content) {
			keepOnPage(ref)
		}
		if payload, ok := toolresult.Payload(content); ok {
			numberedRows(payload, 0, keepOnPage)
			partsIn(payload, 0, read.parents)
			page.total, page.more = longer(payload)
			page.table = keptTable(content)
		}
		if page.more && len(page.ids) > 0 {
			read.pages = append(read.pages, page)
		}
	}

	return read
}

func (r Read) ThisTurn(results []string) Read {
	r.turn = make(map[string]bool, 16)
	for _, ref := range Of(results).refs {
		r.turn[ref.ID] = true
	}

	return r
}

func longer(payload any) (int, bool) {
	fields, isObject := payload.(map[string]any)
	if !isObject {
		return 0, false
	}
	total, _ := fields["total"].(float64)
	count, _ := fields["count"].(float64)
	more, _ := fields["hasMore"].(bool)

	return int(total), more || (total > 0 && total > count)
}

func (r Read) Longer(ids []pulid.ID) (int, bool) {
	for _, page := range r.pages {
		if !slices.ContainsFunc(ids, func(id pulid.ID) bool { return page.ids[id.String()] }) {
			continue
		}
		if page.total > 0 && page.total <= len(ids) {
			continue
		}
		return page.total, true
	}

	return 0, false
}

func (r Read) WholeTyped(words string, shown int) bool {
	for _, page := range r.pages {
		if page.total <= shown {
			continue
		}
		for _, written := range []string{strconv.Itoa(page.total), grouped(page.total)} {
			if stringutils.ContainsWords(words, written) {
				return true
			}
		}
	}

	return false
}

func grouped(n int) string {
	digits := strconv.Itoa(n)
	var b strings.Builder
	for idx, digit := range digits {
		if idx > 0 && (len(digits)-idx)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(digit)
	}

	return b.String()
}

func CountRefusal(tool, figure string, kind recordkind.Kind, shown, total int) string {
	whole := "it has more pages than were read"
	if total > 0 {
		whole = fmt.Sprintf("it holds %d in all", total)
	}

	return fmt.Sprintf(
		"Tool %q was not run: %s would read %d, the records shown, but they are part of a "+
			"longer list (%s), so it would state a number that is not true. Where the words "+
			"mean the whole list, write {all}: it reads the list's own number, and again "+
			"whenever the board is read (\"{all} %s\"). Where they mean only the "+
			"records shown, say so: \"these {count}\", \"the {count} shown here\"; a "+
			"count of part of the whole list is a figure Trenova keeps (a metric's) or none.",
		tool, figure, shown, whole, inflect.PluralNoun(kind.Noun()),
	)
}

func UnkeptRefusal(tool string) string {
	return fmt.Sprintf(
		"Tool %q was not run: it would count a list longer than the records read, and that "+
			"list reported no total it can be read again by, so the number would not be true. "+
			"Count only the records the board shows and say so in the words, or read the list "+
			"with a tool that reports its total first.",
		tool,
	)
}

func numberedRows(value any, depth int, keep func(agentpagecontext.EntityRef)) {
	if depth > maxDepth {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		for _, pair := range numberedBy {
			id, _ := typed[pair.idKey].(string)
			label, _ := typed[pair.labelKey].(string)
			if id == "" || strings.TrimSpace(label) == "" || !pulid.LooksLike(id) {
				continue
			}
			if kind, known := recordkind.OfIDPrefix(pulid.ID(id).Prefix()); known &&
				kind == pair.kind {
				keep(agentpagecontext.EntityRef{Type: string(kind), ID: id, Label: label})
			}
		}
		for _, name := range slices.Sorted(maps.Keys(typed)) {
			numberedRows(typed[name], depth+1, keep)
		}
	case []any:
		for _, element := range typed {
			numberedRows(element, depth+1, keep)
		}
	}
}

func DecodeLists(arguments map[string]any, keys ...string) (map[string]any, bool) {
	var out map[string]any
	for _, key := range keys {
		text, isText := arguments[key].(string)
		text = strings.TrimSpace(text)
		if !isText || !strings.HasPrefix(text, "[") {
			continue
		}
		var decoded []any
		if err := sonic.UnmarshalString(text, &decoded); err != nil {
			continue
		}
		if out == nil {
			out = maps.Clone(arguments)
		}
		out[key] = decoded
	}
	if out == nil {
		return arguments, false
	}

	return out, true
}

func (r Read) Numbers(arguments map[string]any) (map[string]any, bool) {
	return rewrite(arguments, r.idFor)
}

func (r Read) idFor(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || pulid.LooksLike(trimmed) {
		return text, false
	}
	found := ""
	for _, ref := range r.refs {
		if ref.Label == "" || !strings.EqualFold(ref.Label, trimmed) {
			continue
		}
		if found != "" && found != ref.ID {
			return text, false
		}
		found = ref.ID
	}

	return found, found != ""
}

func (r Read) Unshown(
	shown map[recordkind.Kind][]pulid.ID,
	words string,
) []agentpagecontext.EntityRef {
	shownLabels := make(map[string]struct{}, len(shown))
	for _, ref := range r.refs {
		if holds(shown, ref) && ref.Label != "" {
			shownLabels[strings.ToLower(ref.Label)] = struct{}{}
		}
	}
	unshown := make([]agentpagecontext.EntityRef, 0, 1)
	for _, ref := range r.refs {
		if _, kind := shown[recordkind.Kind(ref.Type)]; !kind || holds(shown, ref) {
			continue
		}
		if _, same := shownLabels[strings.ToLower(ref.Label)]; same || !named(words, ref.Label) {
			continue
		}
		unshown = append(unshown, ref)
	}

	return unshown
}

func (r Read) SwapFor(
	shown map[recordkind.Kind][]pulid.ID,
	words string,
	unshown []agentpagecontext.EntityRef,
) (Swap, bool) {
	if len(unshown) != 1 {
		return Swap{}, false
	}
	wanted := unshown[0]
	var nameless []agentpagecontext.EntityRef
	for _, id := range shown[recordkind.Kind(wanted.Type)] {
		ref, known := r.ref(id.String())
		if !known || utf8.RuneCountInString(ref.Label) < minLabelRunes {
			return Swap{}, false
		}
		if !named(words, ref.Label) {
			nameless = append(nameless, ref)
		}
	}
	if len(nameless) != 1 {
		return Swap{}, false
	}
	if r.turn[nameless[0].ID] {
		return Swap{From: nameless[0], To: wanted}, false
	}

	return Swap{From: nameless[0], To: wanted}, true
}

func SwapRefusal(tool, holder string, swap *Swap) string {
	return fmt.Sprintf(
		"Tool %q was not run: its words name %s (the %s %s), but no %s holds it, while a %s "+
			"holds %s (%s), which this turn read and the words never name. Say which you mean: "+
			"put %s's id in a %s of its own, or name %s in the words, or take %s out of them.",
		tool, swap.To.Label, recordkind.Kind(swap.To.Type).Noun(), swap.To.ID, holder, holder,
		swap.From.Label, swap.From.ID, swap.To.Label, holder, swap.From.Label, swap.To.Label,
	)
}

func (s *Swap) Apply(arguments map[string]any) map[string]any {
	return ReplaceIDs(arguments, map[string]string{s.From.ID: s.To.ID})
}

func ReplaceIDs(arguments map[string]any, ids map[string]string) map[string]any {
	swapped, _ := rewrite(arguments, func(text string) (string, bool) {
		to, found := ids[strings.TrimSpace(text)]
		if !found {
			return text, false
		}
		return to, true
	})
	swapped, _ = rewriteFigures(swapped, func(_ string, _ int, mark string) string {
		left, right, found := strings.Cut(mark[1:len(mark)-1], ":")
		if !found {
			return mark
		}
		if to, known := ids[strings.TrimSpace(right)]; known {
			return "{" + left + ":" + to + "}"
		}
		if to, known := ids[strings.TrimSpace(left)]; known {
			return "{" + to + ":" + right + "}"
		}
		return mark
	})

	return swapped
}

func Refusal(tool, holder string, unshown []agentpagecontext.EntityRef) string {
	parts := make([]string, 0, min(len(unshown), maxNamedInNote)+1)
	for _, missing := range unshown[:min(len(unshown), maxNamedInNote)] {
		parts = append(parts, fmt.Sprintf("%s (the %s %s)", missing.Label,
			recordkind.Kind(missing.Type).Noun(), missing.ID))
	}
	if len(unshown) > maxNamedInNote {
		parts = append(parts, fmt.Sprintf("%d more", len(unshown)-maxNamedInNote))
	}

	return fmt.Sprintf(
		"Tool %q was not run: its words name %s, but no %s holds it, so the answer would speak "+
			"of a record it does not show. Put each id in the %s it belongs to, in place of any "+
			"record you did not mean, or take the number out of the words.",
		tool, strings.Join(parts, ", "), holder, holder,
	)
}

func (r Read) ref(id string) (agentpagecontext.EntityRef, bool) {
	for _, ref := range r.refs {
		if ref.ID == id {
			return ref, true
		}
	}

	return agentpagecontext.EntityRef{}, false
}

func holds(shown map[recordkind.Kind][]pulid.ID, ref agentpagecontext.EntityRef) bool {
	return slices.Contains(shown[recordkind.Kind(ref.Type)], pulid.ID(ref.ID))
}

func named(words, label string) bool {
	return utf8.RuneCountInString(label) >= minLabelRunes &&
		stringutils.ContainsWords(words, label)
}

func rewrite(
	arguments map[string]any,
	replace func(string) (string, bool),
) (map[string]any, bool) {
	replaced, changed := rewriteIn(arguments, "", replace)
	if !changed {
		return arguments, false
	}
	out, _ := replaced.(map[string]any)

	return out, true
}

func rewriteIn(value any, key string, replace func(string) (string, bool)) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return rewriteObject(typed, replace)
	case []any:
		return rewriteList(typed, key, replace)
	case string:
		if key == recordKey || key == forKey {
			return replace(typed)
		}
	}

	return value, false
}

func rewriteObject(
	typed map[string]any,
	replace func(string) (string, bool),
) (any, bool) {
	var out map[string]any
	for _, name := range slices.Sorted(maps.Keys(typed)) {
		next, changed := rewriteIn(typed[name], name, replace)
		if !changed {
			continue
		}
		if out == nil {
			out = maps.Clone(typed)
		}
		out[name] = next
	}
	if out == nil {
		return typed, false
	}

	return out, true
}

func rewriteList(typed []any, key string, replace func(string) (string, bool)) (any, bool) {
	var out []any
	for idx, element := range typed {
		var next any
		var changed bool
		if text, isText := element.(string); isText && key == recordsKey {
			next, changed = replace(text)
		} else {
			next, changed = rewriteIn(element, "", replace)
		}
		if !changed {
			continue
		}
		if out == nil {
			out = slices.Clone(typed)
		}
		out[idx] = next
	}
	if out == nil {
		return typed, false
	}

	return out, true
}

var figureMark = regexp.MustCompile(`\{([^{}\s]{1,60})\}`)

func (r Read) FigureNumbers(arguments map[string]any) (map[string]any, bool) {
	return rewriteFigures(arguments, func(_ string, _ int, mark string) string {
		return r.figureID(mark)
	})
}

func rewriteFigures(
	arguments map[string]any,
	figure func(text string, at int, mark string) string,
) (map[string]any, bool) {
	out, changed := figuresIn(arguments, "answer", figure)
	groups, isList := out["groups"].([]any)
	if !isList {
		return out, changed
	}
	var cloned []any
	for idx, entry := range groups {
		fields, isObject := entry.(map[string]any)
		if !isObject {
			continue
		}
		next, noted := figuresIn(fields, "note", figure)
		if !noted {
			continue
		}
		if cloned == nil {
			cloned = slices.Clone(groups)
		}
		cloned[idx] = next
	}
	if cloned == nil {
		return out, changed
	}
	if !changed {
		out = maps.Clone(out)
	}
	out["groups"] = cloned

	return out, true
}

func figuresIn(
	fields map[string]any,
	key string,
	figure func(text string, at int, mark string) string,
) (map[string]any, bool) {
	text, isText := fields[key].(string)
	if !isText {
		return fields, false
	}
	var built strings.Builder
	last := 0
	for _, found := range figureMark.FindAllStringIndex(text, -1) {
		built.WriteString(text[last:found[0]])
		built.WriteString(figure(text, found[0], text[found[0]:found[1]]))
		last = found[1]
	}
	built.WriteString(text[last:])
	rewritten := built.String()
	if rewritten == text {
		return fields, false
	}
	out := maps.Clone(fields)
	out[key] = rewritten

	return out, true
}

func (r Read) figureID(mark string) string {
	inner := mark[1 : len(mark)-1]
	left, right, found := strings.Cut(inner, ":")
	if !found {
		return mark
	}
	if id, ok := r.idFor(right); ok {
		return "{" + left + ":" + id + "}"
	}
	if id, ok := r.idFor(left); ok {
		return "{" + right + ":" + id + "}"
	}

	return mark
}

func AnswerShown(
	arguments map[string]any,
) (shown map[recordkind.Kind][]pulid.ID, words string) {
	shown = make(map[recordkind.Kind][]pulid.ID, 1)
	texts := make([]string, 0, 8)
	for _, key := range []string{titleKey, "answer"} {
		if text, isText := arguments[key].(string); isText {
			texts = append(texts, text)
		}
	}
	groups, _ := arguments["groups"].([]any)
	for _, entry := range groups {
		fields, isObject := entry.(map[string]any)
		if !isObject {
			continue
		}
		for _, key := range []string{titleKey, "note"} {
			if text, isText := fields[key].(string); isText {
				texts = append(texts, text)
			}
		}
		listed, _ := fields[recordsKey].([]any)
		for _, element := range listed {
			text, _ := element.(string)
			id := pulid.ID(strings.TrimSpace(text))
			if !pulid.LooksLike(id.String()) {
				continue
			}
			kind, known := recordkind.OfIDPrefix(id.Prefix())
			if known && !slices.Contains(shown[kind], id) {
				shown[kind] = append(shown[kind], id)
			}
		}
	}

	return shown, strings.Join(texts, "\n\n")
}

func (r Read) WithExample(text string, shown map[recordkind.Kind][]pulid.ID) string {
	if !strings.Contains(text, ExampleNumber) {
		return text
	}
	for _, ids := range shown {
		for _, id := range ids {
			if ref, known := r.ref(id.String()); known && ref.Label != "" {
				return strings.ReplaceAll(text, ExampleNumber, ref.Label)
			}
		}
	}

	return text
}

func MetricsListRecords(names []string) bool {
	for _, name := range names {
		metric, known := boardregistry.Catalog().MetricNamed(name)
		if !known {
			continue
		}
		switch metric.Form {
		case boardcatalog.FormTrend, boardcatalog.FormColumns, boardcatalog.FormSplit,
			boardcatalog.FormFigures:
		case boardcatalog.FormBars, boardcatalog.FormGrid, boardcatalog.FormDots,
			boardcatalog.FormSpans, boardcatalog.FormLog, boardcatalog.FormChecks,
			boardcatalog.FormTable:
			return true
		}
	}

	return false
}

func GroupMetrics(arguments map[string]any) []string {
	groups, _ := arguments["groups"].([]any)
	names := make([]string, 0, len(groups))
	for _, entry := range groups {
		fields, _ := entry.(map[string]any)
		if name, isText := fields["metric"].(string); isText && strings.TrimSpace(name) != "" {
			names = append(names, strings.TrimSpace(name))
		}
	}

	return names
}
