package agenttoolcatalog

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

var Module = fx.Module("agent-tool-catalog", fx.Provide(NewFromRegistries))

type Params struct {
	fx.In

	QueryTools  serviceports.AgentQueryToolRegistry
	ActionTools serviceports.AgentToolRegistry
}

// NewFromRegistries indexes both registries once at startup. The catalog is
// immutable afterwards, which is what lets every turn rank against it without a
// lock or a rebuild.
func NewFromRegistries(p Params) *Catalog {
	query := p.QueryTools.Descriptors()
	action := p.ActionTools.Descriptors()

	descriptors := make([]serviceports.AgentToolDescriptor, 0, len(query)+len(action))
	descriptors = append(descriptors, query...)
	descriptors = append(descriptors, action...)

	return New(descriptors)
}
