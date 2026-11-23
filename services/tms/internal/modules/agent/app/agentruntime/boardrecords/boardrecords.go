package boardrecords

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const ExampleNumber = "A-1001"

const (
	recordsKey     = "records"
	recordKey      = "record"
	minLabelRunes  = 4
	maxNamedInNote = 3
	maxDepth       = 6
)

type Read struct {
	refs  []agentpagecontext.EntityRef
	pages []listPage
}

type listPage struct {
	ids   map[string]bool
	total int
	more  bool
}

type Swap struct {
	From agentpagecontext.EntityRef
	To   agentpagecontext.EntityRef
}

var numberedBy = []struct {
	resource permission.Resource
	idKey    string
	labelKey string
}{
	{resource: permission.ResourceShipment, idKey: "shipmentId", labelKey: "proNumber"},
	{resource: permission.ResourceInvoice, idKey: "invoiceId", labelKey: "invoiceNumber"},
}

func Of(results []string) Read {
	read := Read{refs: make([]agentpagecontext.EntityRef, 0, 16)}
	seen := make(map[string]struct{}, 16)
	keep := func(ref agentpagecontext.EntityRef) {
		if !board.Supports(permission.Resource(ref.Type)) {
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
			page.total, page.more = longer(payload)
		}
		if page.more && len(page.ids) > 0 {
			read.pages = append(read.pages, page)
		}
	}

	return read
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

func CountRefusal(tool, figure string, shown, total int) string {
	whole := "it has more pages than were read"
	if total > 0 {
		whole = fmt.Sprintf("it holds %d in all", total)
	}

	return fmt.Sprintf(
		"Tool %q was not run: %s would read %d, the records shown, but they are part of a "+
			"longer list (%s), so it would state a number that is not true. If the words "+
			"mean all of them, write the list's own number as the result gave it, or read the "+
			"whole list first; if they mean only the records shown, count a group, as in "+
			"\"{count_1} of them\".",
		tool, figure, shown, whole,
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
			if resource, known := recordkind.ResourceOfIDPrefix(pulid.ID(id).Prefix()); known &&
				resource == pair.resource {
				keep(agentpagecontext.EntityRef{Type: resource.String(), ID: id, Label: label})
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
	shown map[permission.Resource][]pulid.ID,
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
		if _, kind := shown[permission.Resource(ref.Type)]; !kind || holds(shown, ref) {
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
	shown map[permission.Resource][]pulid.ID,
	words string,
	unshown []agentpagecontext.EntityRef,
) (Swap, bool) {
	if len(unshown) != 1 {
		return Swap{}, false
	}
	wanted := unshown[0]
	var nameless []agentpagecontext.EntityRef
	for _, id := range shown[permission.Resource(wanted.Type)] {
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

	return Swap{From: nameless[0], To: wanted}, true
}

func (s *Swap) Apply(arguments map[string]any) map[string]any {
	swapped, _ := rewrite(arguments, func(text string) (string, bool) {
		if strings.TrimSpace(text) != s.From.ID {
			return text, false
		}
		return s.To.ID, true
	})
	swapped, _ = rewriteFigures(swapped, func(_ string, _ int, mark string) string {
		left, right, found := strings.Cut(mark[1:len(mark)-1], ":")
		switch {
		case !found:
			return mark
		case strings.TrimSpace(right) == s.From.ID:
			return "{" + left + ":" + s.To.ID + "}"
		case strings.TrimSpace(left) == s.From.ID:
			return "{" + s.To.ID + ":" + right + "}"
		}
		return mark
	})

	return swapped
}

func Refusal(tool, holder string, unshown []agentpagecontext.EntityRef) string {
	parts := make([]string, 0, min(len(unshown), maxNamedInNote)+1)
	for _, missing := range unshown[:min(len(unshown), maxNamedInNote)] {
		parts = append(parts, fmt.Sprintf("%s (the %s %s)", missing.Label,
			permission.Resource(missing.Type).Noun(), missing.ID))
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

func holds(shown map[permission.Resource][]pulid.ID, ref agentpagecontext.EntityRef) bool {
	return slices.Contains(shown[permission.Resource(ref.Type)], pulid.ID(ref.ID))
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
			return value, false
		}
		return out, true
	case []any:
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
			return value, false
		}
		return out, true
	case string:
		if key == recordKey {
			return replace(typed)
		}
	}

	return value, false
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
) (shown map[permission.Resource][]pulid.ID, words string) {
	shown = make(map[permission.Resource][]pulid.ID, 1)
	texts := make([]string, 0, 8)
	for _, key := range []string{"title", "answer"} {
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
		for _, key := range []string{"title", "note"} {
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
			resource, known := recordkind.ResourceOfIDPrefix(id.Prefix())
			if known && !slices.Contains(shown[resource], id) {
				shown[resource] = append(shown[resource], id)
			}
		}
	}

	return shown, strings.Join(texts, "\n\n")
}

func (r Read) WithExample(text string, shown map[permission.Resource][]pulid.ID) string {
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
