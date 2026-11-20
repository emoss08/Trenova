package partner

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/textexpr"
)

var (
	segmentIDPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,2}$`)
	bindPattern      = regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)
)

var groupOf = map[string]string{
	"204": "SM",
	"210": "IM",
	"214": "QM",
	"990": "GF",
	"997": "FA",
	"999": "FA",
}

var generatedSets = []string{"204"}

type Duplicates string

const (
	DuplicatesNew    Duplicates = "new"
	DuplicatesResend Duplicates = "resend"
)

type Template struct {
	Set        string          `yaml:"set"`
	Purposes   map[Kind]string `yaml:"purposes"`
	Duplicates Duplicates      `yaml:"duplicates"`
	Load       Profile         `yaml:"load"`
	Segments   []Entry         `yaml:"segments"`
}

type Entry struct {
	Seg      string            `yaml:"seg"`
	El       []any             `yaml:"el"`
	When     string            `yaml:"when"`
	On       []Kind            `yaml:"on"`
	Optional bool              `yaml:"optional"`
	Each     string            `yaml:"each"`
	As       string            `yaml:"as"`
	Body     []Entry           `yaml:"body"`
	path     string            `yaml:"-"`
	elements [][]textexpr.Text `yaml:"-"`
	when     []string          `yaml:"-"`
	negate   bool              `yaml:"-"`
	each     []string          `yaml:"-"`
}

type Segment struct {
	ID       string
	Elements [][]string
}

func (t *Template) Group() string {
	return groupOf[t.Set]
}

func (t *Template) validate(check *problems, p *Partner) {
	if !slices.Contains(generatedSets, t.Set) {
		check.add("set", "%q is not a set the simulator generates (%v)", t.Set, generatedSets)
	}
	if !p.SendsSet(t.Set) {
		check.add("set", "%s is not in partner.yaml sends", t.Set)
	}
	if _, ok := t.Purposes[KindOriginal]; !ok {
		check.add("purposes.original", "is required")
	}
	for kind, code := range t.Purposes {
		if !slices.Contains(Kinds, kind) {
			check.add("purposes."+string(kind), "%q is not a variation kind", kind)
		}
		if len(code) != 2 {
			check.add("purposes."+string(kind), "%q is not a two-character purpose code", code)
		}
	}
	switch t.Duplicates {
	case "", DuplicatesNew:
	case DuplicatesResend:
		if t.Purposes[KindDuplicate] != t.Purposes[KindOriginal] {
			check.add(
				"purposes.duplicate",
				"must equal purposes.original: a resend repeats the original's bytes",
			)
		}
	default:
		check.add("duplicates", "%q is not one of new, resend", t.Duplicates)
	}
	t.Load.validate(check, p.Location())
	if len(t.Segments) == 0 {
		check.add("segments", "lists no segment")
	}
	compileEntries(check, "segments", t.Segments)
}

func compileEntries(check *problems, path string, entries []Entry) {
	for index := range entries {
		entry := &entries[index]
		entry.path = fmt.Sprintf("%s[%d]", path, index)
		if entry.When != "" {
			when := strings.TrimSpace(entry.When)
			entry.negate = strings.HasPrefix(when, "!")
			entry.when = strings.Split(strings.TrimPrefix(when, "!"), ".")
		}
		for _, kind := range entry.On {
			if !slices.Contains(Kinds, kind) {
				check.add(entry.path+".on", "%q is not a variation kind", kind)
			}
		}
		if entry.Each != "" {
			entry.each = strings.Split(entry.Each, ".")
			if !bindPattern.MatchString(entry.As) {
				check.add(entry.path+".as", "%q must name the item, such as stop", entry.As)
			}
		}
		switch {
		case entry.Seg != "" && len(entry.Body) > 0:
			check.add(entry.path, "has both seg and body")
		case entry.Seg != "":
			compileSegment(check, entry)
		case len(entry.Body) > 0:
			if len(entry.El) > 0 {
				check.add(entry.path+".el", "belongs to a seg, not a body")
			}
			compileEntries(check, entry.path+".body", entry.Body)
		default:
			check.add(entry.path, "needs seg or body")
		}
	}
}

func compileSegment(check *problems, entry *Entry) {
	if !segmentIDPattern.MatchString(entry.Seg) || entry.Seg == "ST" || entry.Seg == "SE" {
		check.add(
			entry.path+".seg",
			"%q is not a segment the template writes (ST and SE are the writer's)",
			entry.Seg,
		)
	}
	entry.elements = make([][]textexpr.Text, 0, len(entry.El))
	for position, raw := range entry.El {
		where := fmt.Sprintf("%s.el[%d]", entry.path, position)
		var sources []string
		switch typed := raw.(type) {
		case []any:
			for _, component := range typed {
				sources = append(sources, textexpr.Format(component))
			}
		default:
			sources = []string{textexpr.Format(typed)}
		}
		components := make([]textexpr.Text, 0, len(sources))
		for _, source := range sources {
			compiled, err := textexpr.Parse(source)
			if err != nil {
				check.add(where, "%v", err)
				continue
			}
			components = append(components, compiled)
		}
		entry.elements = append(entry.elements, components)
	}
}

type renderer struct {
	kind   Kind
	sparse bool
	out    []Segment
}

func (t *Template) render(view *scope, kind Kind, sparse bool) ([]Segment, error) {
	r := &renderer{kind: kind, sparse: sparse}
	if err := r.entries(t.Segments, view); err != nil {
		return nil, err
	}
	return r.out, nil
}

func (r *renderer) entries(entries []Entry, view *scope) error {
	for index := range entries {
		if err := r.entry(&entries[index], view); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) entry(entry *Entry, view *scope) error {
	if entry.Optional && r.sparse {
		return nil
	}
	if len(entry.On) > 0 && !slices.Contains(entry.On, r.kind) {
		return nil
	}
	if entry.when != nil {
		value, err := view.Resolve(entry.when)
		if err != nil {
			return fmt.Errorf("%s.when: %w", entry.path, err)
		}
		if textexpr.Truthy(value) == entry.negate {
			return nil
		}
	}
	if entry.each == nil {
		return r.write(entry, view)
	}
	value, err := view.Resolve(entry.each)
	if err != nil {
		return fmt.Errorf("%s.each: %w", entry.path, err)
	}
	items, ok := value.([]any)
	if value != nil && !ok {
		return fmt.Errorf("%s.each: %s is not a list", entry.path, entry.Each)
	}
	for _, item := range items {
		if err := r.write(entry, view.with(entry.As, item)); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) write(entry *Entry, view *scope) error {
	if entry.Seg == "" {
		return r.entries(entry.Body, view)
	}
	segment := Segment{ID: entry.Seg, Elements: make([][]string, 0, len(entry.elements))}
	for position, components := range entry.elements {
		values := make([]string, 0, len(components))
		for _, component := range components {
			value, err := component.Render(view)
			if err != nil {
				return fmt.Errorf("%s.el[%d]: %w", entry.path, position, err)
			}
			values = append(values, value)
		}
		segment.Elements = append(segment.Elements, values)
	}
	r.out = append(r.out, segment)
	return nil
}
