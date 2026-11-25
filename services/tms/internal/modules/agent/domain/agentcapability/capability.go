package agentcapability

import (
	"regexp"
	"slices"
	"strings"
)

type Verb string

const (
	VerbList    Verb = "list"
	VerbGet     Verb = "get"
	VerbInclude Verb = "include"
	VerbCreate  Verb = "create"
	VerbUpdate  Verb = "update"
	VerbAct     Verb = "act"
	VerbRead    Verb = "read"
	VerbTurn    Verb = "turn"
)

type Access string

const (
	AccessRead  Access = "read"
	AccessWrite Access = "write"
	AccessTurn  Access = "turn"
)

const (
	RecordsQuery   = "records_query"
	RecordGet      = "record_get"
	DescribeEntity = "describe_entity"
	SearchTools    = "search_tools"
)

type Instead struct {
	Capability string
	When       string
}

type Facts struct {
	Name        string
	Description string
	Effect      string
	Write       bool
	Turn        bool
	Approval    string
	MaxApproval string
	Resource    string
	Operation   string
	SearchTerms []string
	Recipe      []string
}

type Capability struct {
	ID          string
	Name        string
	Area        string
	Entity      string
	Verb        Verb
	Access      Access
	Approval    string
	MaxApproval string
	Permission  string
	Summary     string
	UseWhen     string
	InsteadOf   []Instead
	Synonyms    []string
	Examples    []string
	Chain       []string
}

func (c *Capability) IsEntityRead() bool {
	return c.Name == RecordsQuery || c.Name == RecordGet
}

func EntityCapabilityID(entity string, verb Verb) string {
	return entity + "." + string(verb)
}

func ParseCapabilityID(id string) (entity string, verb Verb, ok bool) {
	entity, raw, found := strings.Cut(id, ".")
	if !found {
		return "", "", false
	}
	switch Verb(raw) {
	case VerbList, VerbGet:
		return entity, Verb(raw), true
	default:
		return "", "", false
	}
}

func Classify(facts *Facts) Capability {
	capability := Capability{
		ID:          facts.Name,
		Name:        facts.Name,
		Area:        AreaOf(facts.Name),
		Approval:    facts.Approval,
		MaxApproval: facts.MaxApproval,
		Permission:  facts.Resource + ":" + facts.Operation,
		Summary:     FirstSentence(facts.Description),
		Chain:       slices.Clone(facts.Recipe),
	}
	_, verb, object := splitName(facts.Name)
	capability.Entity = entityOf(facts.Name, object)
	switch {
	case facts.Turn:
		capability.Access, capability.Verb = AccessTurn, VerbTurn
	case facts.Write:
		capability.Access, capability.Verb = AccessWrite, writeVerb(verb)
	default:
		capability.Access, capability.Verb = AccessRead, VerbRead
	}
	if guide, ok := guidance[facts.Name]; ok {
		capability.apply(&guide)
	}
	if entity, ok := EntityNamed(capability.Entity); ok {
		capability.Synonyms = appendUnique(capability.Synonyms, entity.Synonyms...)
	}
	capability.Synonyms = appendUnique(capability.Synonyms, facts.SearchTerms...)

	return capability
}

func EntityReads(entity *Entity) []Capability {
	out := make([]Capability, 0, 2)
	if entity.List != "" {
		out = append(out, entityRead(entity, VerbList))
	}
	if entity.Get != "" {
		out = append(out, entityRead(entity, VerbGet))
	}

	return out
}

func entityRead(entity *Entity, verb Verb) Capability {
	title := strings.ToLower(entity.Title)
	capability := Capability{
		ID:       EntityCapabilityID(entity.Name, verb),
		Name:     RecordsQuery,
		Area:     entity.Area,
		Entity:   entity.Name,
		Verb:     verb,
		Access:   AccessRead,
		Approval: "AutoExecute",
		Synonyms: slices.Clone(entity.Synonyms),
	}
	if verb == VerbGet {
		capability.Name = RecordGet
		capability.Summary = "Read one or more " + title + " in full by id or number, with related data through include."
		capability.UseWhen = "You know which " + strings.ReplaceAll(entity.Name, "_", " ") + " you mean and need its details."
		if entity.List != "" {
			capability.InsteadOf = []Instead{{
				Capability: EntityCapabilityID(entity.Name, VerbList),
				When:       "you are looking for " + title + " by what they match, not by id.",
			}}
		}
	} else {
		capability.Summary = "Find, filter, count or group " + title + "."
		capability.UseWhen = "You need every " + strings.ReplaceAll(entity.Name, "_", " ") +
			" matching conditions, a count, or a list to pick from."
		if entity.Get != "" {
			capability.InsteadOf = []Instead{{
				Capability: EntityCapabilityID(entity.Name, VerbGet),
				When:       "you already have the id and need everything about it.",
			}}
		}
	}
	if guide, ok := guidance[capability.ID]; ok {
		capability.apply(&guide)
	}

	return capability
}

func (c *Capability) apply(guide *Guidance) {
	if guide.Summary != "" {
		c.Summary = guide.Summary
	}
	if guide.UseWhen != "" {
		c.UseWhen = guide.UseWhen
	}
	if len(guide.InsteadOf) > 0 {
		c.InsteadOf = slices.Clone(guide.InsteadOf)
	}
	c.Synonyms = appendUnique(c.Synonyms, guide.Synonyms...)
	c.Examples = appendUnique(c.Examples, guide.Examples...)
	if len(guide.Chain) > 0 {
		c.Chain = slices.Clone(guide.Chain)
	}
}

func Describe(name, description string) string {
	guide, ok := guidance[name]
	if !ok {
		return description
	}
	var b strings.Builder
	if guide.Summary != "" {
		b.WriteString(guide.Summary)
		rest := strings.TrimSpace(strings.TrimPrefix(description, FirstSentence(description)))
		if rest != "" {
			b.WriteByte(' ')
			b.WriteString(rest)
		}
	} else {
		b.WriteString(description)
	}
	if guide.UseWhen != "" {
		b.WriteString("\nUse this when ")
		b.WriteString(strings.TrimSuffix(lowerFirst(guide.UseWhen), "."))
		b.WriteByte('.')
	}
	for _, instead := range guide.InsteadOf {
		b.WriteString("\nUse ")
		b.WriteString(insteadName(instead.Capability))
		b.WriteString(" instead when ")
		b.WriteString(strings.TrimSuffix(lowerFirst(instead.When), "."))
		b.WriteByte('.')
	}

	return b.String()
}

func SearchTermsFor(name string, terms []string) []string {
	guide, ok := guidance[name]
	if !ok {
		return terms
	}

	return appendUnique(slices.Clone(terms), append(slices.Clone(guide.Synonyms), guide.Examples...)...)
}

func insteadName(id string) string {
	entity, verb, ok := ParseCapabilityID(id)
	if !ok {
		return id
	}
	if verb == VerbGet {
		return RecordGet + " (entity " + entity + ")"
	}

	return RecordsQuery + " (entity " + entity + ")"
}

func writeVerb(verb string) Verb {
	switch verb {
	case "create", "add", "record", "duplicate", "import", "draft", "build", "generate", "open", "issue", "schedule":
		return VerbCreate
	case "update", "set", "edit", "rename", "adjust", "rerate", "recalculate", "correct", "amend", "reschedule", "arrange":
		return VerbUpdate
	default:
		return VerbAct
	}
}

func entityOf(name, object string) string {
	if entity, _, ok := EntityBacking(name); ok {
		return entity.Name
	}
	area := AreaOf(name)
	if object == "" {
		return area
	}
	object = strings.TrimSuffix(object, "s")
	if _, ok := EntityNamed(object); ok {
		return object
	}
	if _, ok := EntityNamed(area + "_" + object); ok {
		return area + "_" + object
	}
	if _, ok := EntityNamed(object + "_" + area); ok {
		return object + "_" + area
	}

	return area
}

func AreaOf(name string) string {
	area, _, found := strings.Cut(name, "_")
	if !found {
		return "agent"
	}
	switch area {
	case "find", "search", "ask", "show", "publish", "request", "withdraw", "delegate", "name":
		return "agent"
	case "records", "record", "describe":
		return "records"
	}

	return area
}

func splitName(name string) (area, verb, object string) {
	area, rest, _ := strings.Cut(name, "_")
	verb, object, _ = strings.Cut(rest, "_")

	return area, verb, object
}

func FirstSentence(text string) string {
	text = strings.TrimSpace(text)
	for idx := 0; idx < len(text); idx++ {
		if text[idx] != '.' && text[idx] != '?' && text[idx] != '!' {
			continue
		}
		if idx+1 == len(text) || text[idx+1] == ' ' || text[idx+1] == '\n' {
			return text[:idx+1]
		}
	}

	return text
}

func lowerFirst(text string) string {
	if text == "" {
		return text
	}

	return strings.ToLower(text[:1]) + text[1:]
}

func appendUnique(list []string, values ...string) []string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(list, value) {
			continue
		}
		list = append(list, value)
	}

	return list
}

var readerWord = regexp.MustCompile(`\b[a-z]+_(?:list|get)(?:_[a-z_]+)?\b`)

func RewriteReaders(text string) string {
	if !strings.Contains(text, "_list") && !strings.Contains(text, "_get") {
		return text
	}

	return readerWord.ReplaceAllStringFunc(text, func(word string) string {
		entity, verb, ok := EntityBacking(word)
		if !ok {
			return word
		}
		if verb == VerbList {
			return RecordsQuery + " (entity " + entity.Name + ")"
		}

		return RecordGet + " (entity " + entity.Name + ")"
	})
}

func RewriteSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	out := make(map[string]any, len(schema))
	for key, value := range schema {
		switch typed := value.(type) {
		case string:
			if key == "description" {
				out[key] = RewriteReaders(typed)
				continue
			}
			out[key] = typed
		case map[string]any:
			out[key] = RewriteSchema(typed)
		case []any:
			items := make([]any, len(typed))
			for idx, item := range typed {
				if object, ok := item.(map[string]any); ok {
					items[idx] = RewriteSchema(object)
					continue
				}
				items[idx] = item
			}
			out[key] = items
		default:
			out[key] = value
		}
	}

	return out
}
