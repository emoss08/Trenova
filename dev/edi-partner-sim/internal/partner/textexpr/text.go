package textexpr

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Text struct {
	parts []part
}

type part struct {
	literal string
	path    []string
	filters []filter
}

type filter struct {
	name string
	arg  string
}

var filterNames = []string{
	"date", "date6", "time", "zone", "dollars", "upper", "lower", "trunc", "zip5", "zpad", "map",
}

func Parse(source string) (Text, error) {
	var out Text
	rest := source
	for rest != "" {
		open := strings.IndexByte(rest, '{')
		if open < 0 {
			out.parts = append(out.parts, part{literal: rest})
			break
		}
		if open > 0 {
			out.parts = append(out.parts, part{literal: rest[:open]})
		}
		closing := strings.IndexByte(rest[open:], '}')
		if closing < 0 {
			return Text{}, fmt.Errorf("%q has a { with no }", source)
		}
		parsed, err := parseExpr(rest[open+1 : open+closing])
		if err != nil {
			return Text{}, err
		}
		out.parts = append(out.parts, parsed)
		rest = rest[open+closing+1:]
	}
	return out, nil
}

func parseExpr(source string) (part, error) {
	pieces := strings.Split(source, "|")
	path := strings.TrimSpace(pieces[0])
	if path == "" {
		return part{}, fmt.Errorf("{%s} names no value", source)
	}
	parsed := part{path: strings.Split(path, ".")}
	for _, raw := range pieces[1:] {
		name, arg, _ := strings.Cut(strings.TrimSpace(raw), ":")
		if !slices.Contains(filterNames, name) {
			return part{}, fmt.Errorf(
				"{%s}: %q is not one of the filters %v",
				source,
				name,
				filterNames,
			)
		}
		if err := checkFilterArg(name, arg); err != nil {
			return part{}, fmt.Errorf("{%s}: %w", source, err)
		}
		parsed.filters = append(parsed.filters, filter{name: name, arg: arg})
	}
	return parsed, nil
}

func checkFilterArg(name, arg string) error {
	switch name {
	case "trunc", "zpad":
		if n, err := strconv.Atoi(arg); err != nil || n < 1 {
			return fmt.Errorf("%s takes a length, such as %s:4", name, name)
		}
	case "map":
		for pair := range strings.SplitSeq(arg, ",") {
			if from, _, ok := strings.Cut(pair, "="); !ok || from == "" {
				return fmt.Errorf("map takes from=to pairs, such as map:pickup=LD,delivery=UL")
			}
		}
	default:
		if arg != "" {
			return fmt.Errorf("%s takes no argument", name)
		}
	}
	return nil
}

type Resolver interface {
	Resolve(path []string) (any, error)
}

func (t Text) Render(view Resolver) (string, error) {
	var out strings.Builder
	for _, piece := range t.parts {
		if piece.path == nil {
			out.WriteString(piece.literal)
			continue
		}
		value, err := view.Resolve(piece.path)
		if err != nil {
			return "", err
		}
		for _, f := range piece.filters {
			if value, err = f.apply(value); err != nil {
				return "", fmt.Errorf("{%s|%s}: %w", strings.Join(piece.path, "."), f.name, err)
			}
		}
		out.WriteString(Format(value))
	}
	return out.String(), nil
}

func (f filter) apply(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch f.name {
	case "date", "date6", "time", "zone":
		at, ok := value.(time.Time)
		if !ok {
			return nil, fmt.Errorf("%v is not a time", value)
		}
		return formatTime(f.name, at), nil
	case "dollars":
		cents, ok := asInt(value)
		if !ok {
			return nil, fmt.Errorf("%v is not an amount in cents", value)
		}
		return fmt.Sprintf("%d.%02d", cents/100, abs(cents%100)), nil
	case "zpad":
		number, ok := asInt(value)
		if !ok {
			return nil, fmt.Errorf("%v is not a whole number", value)
		}
		width, _ := strconv.Atoi(f.arg)
		if number < 0 {
			return fmt.Sprintf("-%0*d", width-1, -number), nil
		}
		return fmt.Sprintf("%0*d", width, number), nil
	}
	text := Format(value)
	switch f.name {
	case "upper":
		return strings.ToUpper(text), nil
	case "lower":
		return strings.ToLower(text), nil
	case "trunc":
		width, _ := strconv.Atoi(f.arg)
		return text[:min(len(text), width)], nil
	case "zip5":
		return text[:min(len(text), 5)], nil
	default:
		for pair := range strings.SplitSeq(f.arg, ",") {
			from, to, _ := strings.Cut(pair, "=")
			if from == text {
				return to, nil
			}
		}
		return nil, fmt.Errorf("%q is not in map:%s", text, f.arg)
	}
}

func formatTime(name string, at time.Time) string {
	switch name {
	case "date":
		return at.Format("20060102")
	case "date6":
		return at.Format("060102")
	case "time":
		return at.Format("1504")
	default:
		abbreviation := at.Format("MST")
		if len(abbreviation) == 3 && abbreviation[2] == 'T' {
			return abbreviation[:1] + "T"
		}
		return "UT"
	}
}

func Format(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		if typed {
			return "Y"
		}
		return "N"
	case time.Time:
		return typed.Format(time.RFC3339)
	default:
		return fmt.Sprint(typed)
	}
}

func asInt(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	default:
		return 0, false
	}
}

func abs(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func Truthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return typed != ""
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case bool:
		return typed
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}
