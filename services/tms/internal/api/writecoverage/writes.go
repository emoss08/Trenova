package writecoverage

import (
	"cmp"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/schemadiff"
	"github.com/gin-gonic/gin"
)

type Kind string

const (
	KindMutation = Kind("mutation")
	KindRoute    = Kind("route")

	mutationPrefix   = "mutation "
	mutationReceiver = "mutationResolver"
	resolverReceiver = "Resolver"
	handlerReceiver  = "Handler"
	handlerSuffix    = "handler"
	schemaExtension  = ".graphqls"
)

var (
	writeMethods = map[string]struct{}{
		http.MethodPost:   {},
		http.MethodPut:    {},
		http.MethodPatch:  {},
		http.MethodDelete: {},
	}
	handlerMethod = regexp.MustCompile(`\(\*?` + handlerReceiver + `\)\.(\w+)`)
)

type Route struct {
	Method string
	Path   string
}

func (r Route) String() string {
	return r.Method + " " + r.Path
}

type Endpoint struct {
	Handler string
	Routes  []Route
	Calls   []string
}

type Write struct {
	Endpoint

	Key    string
	Kind   Kind
	Domain string
	Twins  []Endpoint
}

type Sources struct {
	SchemaDir   string
	ResolverDir string
	HandlersDir string
	Routes      gin.RoutesInfo
}

func Enumerate(src Sources) ([]Write, error) {
	mutations, err := enumerateMutations(src.SchemaDir, src.ResolverDir)
	if err != nil {
		return nil, err
	}

	routes, err := enumerateRoutes(src.Routes, src.HandlersDir)
	if err != nil {
		return nil, err
	}

	writes := mergeTwins(mutations, routes)
	sortWrites(writes)

	return writes, nil
}

func MutationKey(name string) string {
	return mutationPrefix + name
}

func enumerateMutations(schemaDir, resolverDir string) ([]Write, error) {
	schema, err := schemadiff.LoadDir(schemaDir)
	if err != nil {
		return nil, err
	}
	if schema.Mutation == nil {
		return nil, fmt.Errorf("schema in %s declares no Mutation type", schemaDir)
	}

	index, err := indexPackage(resolverDir, resolverReceiver, mutationReceiver)
	if err != nil {
		return nil, err
	}
	resolvers := make(map[string]string, len(index.methods))
	for name := range index.methods {
		resolvers[strings.ToLower(name)] = name
	}

	writes := make([]Write, 0, len(schema.Mutation.Fields))
	for _, field := range schema.Mutation.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}

		source := ""
		if field.Position != nil && field.Position.Src != nil {
			source = field.Position.Src.Name
		}

		write := Write{
			Key:    MutationKey(field.Name),
			Kind:   KindMutation,
			Domain: domainName(strings.TrimSuffix(source, schemaExtension)),
		}
		if method, ok := resolvers[strings.ToLower(field.Name)]; ok {
			write.Handler = mutationReceiver + "." + method
			write.Calls = index.calls(method)
		}
		writes = append(writes, write)
	}

	return writes, nil
}

type handlerName struct {
	pkg    string
	method string
}

func (h handlerName) String() string {
	return h.pkg + "." + h.method
}

func parseHandler(name string) (handlerName, bool) {
	base := name[strings.LastIndex(name, "/")+1:]
	pkg, rest, found := strings.Cut(base, ".")
	if !found {
		return handlerName{pkg: base}, false
	}

	matches := handlerMethod.FindAllStringSubmatch(rest, -1)
	if len(matches) == 0 {
		return handlerName{pkg: pkg, method: rest}, false
	}

	return handlerName{pkg: pkg, method: matches[len(matches)-1][1]}, true
}

func enumerateRoutes(table gin.RoutesInfo, handlersDir string) ([]Write, error) {
	grouped := make(map[string]*Write)
	indexes, err := newHandlerIndexes(handlersDir)
	if err != nil {
		return nil, err
	}

	for _, info := range table {
		if _, write := writeMethods[info.Method]; !write {
			continue
		}

		name, isMethod := parseHandler(info.Handler)
		route := Route{Method: info.Method, Path: info.Path}

		if existing, ok := grouped[info.Handler]; ok {
			existing.Routes = append(existing.Routes, route)
			continue
		}

		write := &Write{
			Kind:     KindRoute,
			Domain:   domainName(strings.TrimSuffix(name.pkg, handlerSuffix)),
			Endpoint: Endpoint{Handler: name.String(), Routes: []Route{route}},
		}
		if isMethod {
			calls, callsErr := indexes.calls(name)
			if callsErr != nil {
				return nil, callsErr
			}
			write.Calls = calls
		}
		grouped[info.Handler] = write
	}

	writes := make([]Write, 0, len(grouped))
	for _, write := range grouped {
		slices.SortFunc(write.Routes, compareRoutes)
		write.Key = write.Routes[0].String()
		writes = append(writes, *write)
	}

	return writes, nil
}

type handlerIndexes struct {
	dir      string
	packages map[string]struct{}
	built    map[string]*receiverIndex
}

func newHandlerIndexes(dir string) (*handlerIndexes, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	packages := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			packages[entry.Name()] = struct{}{}
		}
	}

	return &handlerIndexes{
		dir:      dir,
		packages: packages,
		built:    make(map[string]*receiverIndex, len(packages)),
	}, nil
}

func (h *handlerIndexes) calls(name handlerName) ([]string, error) {
	if _, ok := h.packages[name.pkg]; !ok {
		return nil, nil
	}

	index, ok := h.built[name.pkg]
	if !ok {
		built, err := indexPackage(filepath.Join(h.dir, name.pkg), handlerReceiver)
		if err != nil {
			return nil, err
		}
		h.built[name.pkg] = built
		index = built
	}

	return index.calls(name.method), nil
}

func mergeTwins(mutations, routes []Write) []Write {
	bySignature := make(map[string][]int, len(mutations))
	for idx := range mutations {
		if len(mutations[idx].Calls) == 0 {
			continue
		}
		signature := strings.Join(mutations[idx].Calls, "\n")
		bySignature[signature] = append(bySignature[signature], idx)
	}

	writes := make([]Write, 0, len(mutations)+len(routes))
	for idx := range routes {
		route := routes[idx]
		twin, found := twinOf(&route, mutations, bySignature)
		if !found {
			writes = append(writes, route)
			continue
		}
		mutations[twin].Twins = append(mutations[twin].Twins, route.Endpoint)
	}

	for idx := range mutations {
		slices.SortFunc(mutations[idx].Twins, func(a, b Endpoint) int {
			return compareRoutes(a.Routes[0], b.Routes[0])
		})
	}

	return append(writes, mutations...)
}

func twinOf(route *Write, mutations []Write, bySignature map[string][]int) (int, bool) {
	if len(route.Calls) == 0 {
		return 0, false
	}

	matches := bySignature[strings.Join(route.Calls, "\n")]
	if len(matches) > 1 {
		_, method, _ := strings.Cut(route.Handler, ".")
		named := make([]int, 0, len(matches))
		for _, idx := range matches {
			mutation := strings.TrimPrefix(mutations[idx].Key, mutationPrefix)
			if strings.HasPrefix(strings.ToLower(mutation), strings.ToLower(method)) {
				named = append(named, idx)
			}
		}
		matches = named
	}
	if len(matches) != 1 {
		return 0, false
	}

	return matches[0], true
}

func sortWrites(writes []Write) {
	sort.Slice(writes, func(i, j int) bool {
		return compareWrites(&writes[i], &writes[j]) < 0
	})
}

func domainName(raw string) string {
	return strings.ToLower(strings.ReplaceAll(raw, "_", ""))
}

func compareRoutes(a, b Route) int {
	return cmp.Or(
		cmp.Compare(len(a.Path), len(b.Path)),
		strings.Compare(a.Path, b.Path),
		strings.Compare(a.Method, b.Method),
	)
}

func compareWrites(a, b *Write) int {
	return cmp.Or(
		strings.Compare(a.Domain, b.Domain),
		strings.Compare(string(a.Kind), string(b.Kind)),
		strings.Compare(a.Key, b.Key),
	)
}
