package writecoverage

import (
	"path/filepath"

	"github.com/emoss08/trenova/internal/api"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy/registered"
)

const (
	MappingFile         = "writecoverage.yml"
	MappingDisplayPath  = "services/tms/internal/api/writecoverage/" + MappingFile
	DocumentFile        = "../../../../../" + DocumentDisplayPath
	DocumentDisplayPath = "docs/engineering/agent-write-coverage.md"
	GenerateCommand     = "go generate ./internal/api/writecoverage/..."

	schemaDir   = "../graphql/schema"
	resolverDir = "../graphql/resolver"
	handlersDir = "../handlers"
)

type Report struct {
	Writes    []Write
	Mapping   Mapping
	Tools     map[string]serviceports.ToolPolicy
	ToolOrder []string
}

func Load(dir string) (Report, error) {
	table, err := api.RouteTable()
	if err != nil {
		return Report{}, err
	}

	writes, err := Enumerate(Sources{
		SchemaDir:   filepath.Join(dir, schemaDir),
		ResolverDir: filepath.Join(dir, resolverDir),
		HandlersDir: filepath.Join(dir, handlersDir),
		Routes:      table,
	})
	if err != nil {
		return Report{}, err
	}

	mapping, err := LoadMapping(filepath.Join(dir, MappingFile))
	if err != nil {
		return Report{}, err
	}

	catalog, err := registered.Catalog()
	if err != nil {
		return Report{}, err
	}
	policies := catalog.All()
	tools := make(map[string]serviceports.ToolPolicy, len(policies))
	order := make([]string, 0, len(policies))
	for idx := range policies {
		tools[policies[idx].Name] = policies[idx]
		order = append(order, policies[idx].Name)
	}

	return Report{Writes: writes, Mapping: mapping, Tools: tools, ToolOrder: order}, nil
}

func (r *Report) Problems() []string {
	return Check(r.Writes, r.Mapping, r.Tools)
}

func (r *Report) Pending() int {
	count := 0
	for idx := range r.Writes {
		if decision, ok := r.Mapping.Writes[r.Writes[idx].Key]; ok &&
			decision.State() == StatePending {
			count++
		}
	}

	return count
}
