// Package productguide is Trenova's description of itself: every page a
// signed-in person can open, where it sits, what guards it, what it is for and
// how to do things on it, and where each kind of record opens.
//
// catalog_gen.json is generated from the web app's router, navigation config,
// page headers and record-link registry, plus the guides under
// docs/product-guide; see docs/engineering/product-guide.md. It is embedded, so
// a server always carries the guide that matches the app it was built with.
package productguide

import (
	_ "embed"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/bytedance/sonic"
)

//go:embed catalog_gen.json
var raw []byte

// Requirement is a permission a page's loader demands.
type Requirement struct {
	Resource  string `json:"resource"`
	Operation string `json:"operation"`
}

// Task is one thing a person does on a page, as numbered steps.
type Task struct {
	Title    string   `json:"title"`
	Keywords []string `json:"keywords"`
	Steps    []string `json:"steps"`
}

// CreateAction is the page's own create entry point: the address parameters
// that open its create form.
type CreateAction struct {
	Label string            `json:"label"`
	Query map[string]string `json:"query"`
}

type Page struct {
	Path         string        `json:"path"`
	Name         string        `json:"name"`
	Module       string        `json:"module"`
	Breadcrumb   []string      `json:"breadcrumb"`
	Description  string        `json:"description"`
	Summary      string        `json:"summary"`
	Requires     []Requirement `json:"requires"`
	Capabilities []string      `json:"capabilities"`
	Aliases      []string      `json:"aliases"`
	Tasks        []Task        `json:"tasks"`
	Notes        string        `json:"notes"`
	Related      []string      `json:"related"`
	Covers       []string      `json:"covers"`
	CreateAction *CreateAction `json:"createAction"`
	InNavigation bool          `json:"inNavigation"`
}

// Location is where the page sits, as a person reads it in the sidebar.
func (p *Page) Location() string {
	return strings.Join(p.Breadcrumb, " › ")
}

type Module struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// RecordLink is where one kind of record opens. "{id}" stands for the
// record's id in the path or in a parameter's value.
type RecordLink struct {
	Entity string            `json:"entity"`
	Label  string            `json:"label"`
	Path   string            `json:"path"`
	Params map[string]string `json:"params"`
}

type Catalog struct {
	Version string       `json:"version"`
	Modules []Module     `json:"modules"`
	Pages   []Page       `json:"pages"`
	Records []RecordLink `json:"records"`

	byPath     map[string]int
	byResource map[string]int
	covered    []coveredRoute
	records    map[string]int
	modules    map[string]int
	prefixes   []int
}

type coveredRoute struct {
	pattern *regexp.Regexp
	page    int
}

const (
	recordIDToken = "{id}"
	readOperation = "read"
)

// preferredListPage decides which of two pages guarded by the same resource is
// the one that resource lives on: the one in the navigation, then the one
// nearest the top of its module.
func preferredListPage(candidate, current *Page) bool {
	if candidate.InNavigation != current.InNavigation {
		return candidate.InNavigation
	}
	if len(candidate.Path) != len(current.Path) {
		return len(candidate.Path) < len(current.Path)
	}

	return candidate.Path < current.Path
}

// Default is the catalog this server was built with.
var Default = mustLoad(raw)

func mustLoad(data []byte) *Catalog {
	catalog, err := Load(data)
	if err != nil {
		panic(fmt.Sprintf("productguide: %v", err))
	}

	return catalog
}

// Load reads a catalog and indexes it.
func Load(data []byte) (*Catalog, error) {
	catalog := new(Catalog)
	if err := sonic.Unmarshal(data, catalog); err != nil {
		return nil, fmt.Errorf("read the product guide catalog: %w", err)
	}

	catalog.byPath = make(map[string]int, len(catalog.Pages))
	catalog.byResource = make(map[string]int, len(catalog.Pages))
	catalog.records = make(map[string]int, len(catalog.Records))
	catalog.modules = make(map[string]int, len(catalog.Modules))
	catalog.prefixes = make([]int, 0, len(catalog.Pages))

	for i := range catalog.Pages {
		page := &catalog.Pages[i]
		catalog.byPath[page.Path] = i
		catalog.prefixes = append(catalog.prefixes, i)
		for _, covered := range page.Covers {
			catalog.covered = append(catalog.covered, coveredRoute{
				pattern: routePattern(covered),
				page:    i,
			})
		}
	}
	// Longest path first, so a nested page is found before the one it sits
	// under.
	sort.SliceStable(catalog.prefixes, func(a, b int) bool {
		return len(catalog.Pages[catalog.prefixes[a]].Path) > len(catalog.Pages[catalog.prefixes[b]].Path)
	})
	for i := range catalog.Pages {
		page := &catalog.Pages[i]
		for _, requirement := range page.Requires {
			if requirement.Operation != readOperation {
				continue
			}
			current, seen := catalog.byResource[requirement.Resource]
			if !seen || preferredListPage(page, &catalog.Pages[current]) {
				catalog.byResource[requirement.Resource] = i
			}
		}
	}
	for i := range catalog.Records {
		catalog.records[catalog.Records[i].Entity] = i
	}
	for i := range catalog.Modules {
		catalog.modules[catalog.Modules[i].ID] = i
	}

	return catalog, nil
}

func routePattern(path string) *regexp.Regexp {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			segments[i] = "[^/]+"
			continue
		}
		segments[i] = regexp.QuoteMeta(segment)
	}

	return regexp.MustCompile("^" + strings.Join(segments, "/") + "$")
}

// Page is the page at exactly this path.
func (c *Catalog) Page(path string) (*Page, bool) {
	i, ok := c.byPath[normalize(path)]
	if !ok {
		return nil, false
	}

	return &c.Pages[i], true
}

// PageForPath is the page a location belongs to: the page itself, a route its
// guide covers (a detail page, a create page), or the nearest page it sits
// under. A query string or fragment is ignored.
func (c *Catalog) PageForPath(location string) (*Page, bool) {
	path := normalize(stripQuery(location))
	if page, ok := c.Page(path); ok {
		return page, true
	}
	for _, covered := range c.covered {
		if covered.pattern.MatchString(path) {
			return &c.Pages[covered.page], true
		}
	}
	for _, i := range c.prefixes {
		page := &c.Pages[i]
		if page.Path != "/" && strings.HasPrefix(path, page.Path+"/") {
			return page, true
		}
	}

	return nil, false
}

// PageForResource is the page records of a permission resource are listed and
// opened on: the page whose loader guards on reading it, preferring the one
// in the navigation.
func (c *Catalog) PageForResource(resource string) (*Page, bool) {
	i, ok := c.byResource[resource]
	if !ok {
		return nil, false
	}

	return &c.Pages[i], true
}

// Module is a module's label and description.
func (c *Catalog) Module(id string) (*Module, bool) {
	i, ok := c.modules[id]
	if !ok {
		return nil, false
	}

	return &c.Modules[i], true
}

// Record is where one kind of record opens.
func (c *Catalog) Record(entity string) (*RecordLink, bool) {
	i, ok := c.records[entity]
	if !ok {
		return nil, false
	}

	return &c.Records[i], true
}

// RecordPath is the address that opens one record, with any extra parameters
// (a tab, say) added.
func (c *Catalog) RecordPath(entity, id string, extra map[string]string) (string, bool) {
	link, ok := c.Record(entity)
	if !ok || strings.TrimSpace(id) == "" {
		return "", false
	}

	path := strings.ReplaceAll(link.Path, recordIDToken, url.PathEscape(id))
	params := make(url.Values, len(link.Params)+len(extra))
	for key, value := range link.Params {
		params.Set(key, strings.ReplaceAll(value, recordIDToken, id))
	}
	for key, value := range extra {
		params.Set(key, value)
	}
	if len(params) == 0 {
		return path, true
	}

	return path + "?" + params.Encode(), true
}

// ListPath is the page a kind of record is listed on, without opening one.
func (c *Catalog) ListPath(entity string) (string, bool) {
	link, ok := c.Record(entity)
	if !ok || strings.Contains(link.Path, recordIDToken) {
		return "", false
	}

	return link.Path, true
}

// RecordPath is Default.RecordPath.
func RecordPath(entity, id string) (string, bool) {
	return Default.RecordPath(entity, id, nil)
}

func stripQuery(location string) string {
	if i := strings.IndexAny(location, "?#"); i >= 0 {
		return location[:i]
	}

	return location
}

func normalize(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "/" {
		return "/"
	}

	return strings.TrimRight(trimmed, "/")
}
