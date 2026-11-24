package agentcapability

import (
	"slices"
	"strings"
)

type Include struct {
	Name        string
	Reader      string
	Key         string
	Summary     string
	FromSummary bool
}

type Entity struct {
	Name     string
	Area     string
	Title    string
	Synonyms []string
	List     string
	Get      string
	Includes []Include
	Aliases  map[string]string
}

func (e *Entity) Words() []string {
	words := strings.Fields(strings.ReplaceAll(e.Name, "_", " "))
	words = append(words, strings.Fields(strings.ToLower(e.Title))...)
	for _, synonym := range e.Synonyms {
		words = append(words, strings.Fields(synonym)...)
	}

	return words
}

func (e *Entity) Include(name string) (Include, bool) {
	idx := slices.IndexFunc(e.Includes, func(include Include) bool { return include.Name == name })
	if idx < 0 {
		return Include{}, false
	}

	return e.Includes[idx], true
}

func (e *Entity) IncludeNames() []string {
	names := make([]string, 0, len(e.Includes))
	for idx := range e.Includes {
		names = append(names, e.Includes[idx].Name)
	}

	return names
}

func Entities() []Entity {
	out := make([]Entity, len(entities))
	copy(out, entities)

	return out
}

func EntityNamed(name string) (Entity, bool) {
	idx := slices.IndexFunc(entities, func(entity Entity) bool { return entity.Name == name })
	if idx < 0 {
		return Entity{}, false
	}

	return entities[idx], true
}

func EntityNames() []string {
	names := make([]string, 0, len(entities))
	for idx := range entities {
		names = append(names, entities[idx].Name)
	}
	slices.Sort(names)

	return names
}

func EntityBacking(reader string) (entity Entity, verb Verb, ok bool) {
	for idx := range entities {
		switch reader {
		case entities[idx].List:
			return entities[idx], VerbList, true
		case entities[idx].Get:
			return entities[idx], VerbGet, true
		}
		for _, include := range entities[idx].Includes {
			if include.Reader == reader {
				return entities[idx], VerbInclude, true
			}
		}
	}

	return Entity{}, "", false
}

func IsBacking(reader string) bool {
	_, _, ok := EntityBacking(reader)
	return ok
}

func Backings() []string {
	seen := make(map[string]struct{}, len(entities)*2)
	out := make([]string, 0, len(entities)*2)
	add := func(name string) {
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for idx := range entities {
		add(entities[idx].List)
		add(entities[idx].Get)
		for _, include := range entities[idx].Includes {
			add(include.Reader)
		}
	}
	slices.Sort(out)

	return out
}
