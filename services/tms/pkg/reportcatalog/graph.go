package reportcatalog

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen"
)

type Cardinality string

const (
	CardinalityOne  Cardinality = "one"
	CardinalityMany Cardinality = "many"
	CardinalityM2M  Cardinality = "m2m"
)

func (c Cardinality) IsToMany() bool {
	return c == CardinalityMany || c == CardinalityM2M
}

type JoinPair struct {
	Local  string
	Remote string
}

type ThroughJoin struct {
	Table      buncolgen.TableInfo
	SourceJoin []JoinPair
	TargetJoin []JoinPair
	Tenant     TenantColumns
}

type Edge struct {
	Name        string
	Label       string
	Source      string
	Target      string
	Cardinality Cardinality
	Join        []JoinPair
	Through     *ThroughJoin
	Traversable bool
}

var (
	ErrUnknownEntity      = errors.New("reportcatalog: unknown entity")
	ErrUnknownEdge        = errors.New("reportcatalog: unknown edge")
	ErrEdgeNotTraversable = errors.New("reportcatalog: edge not traversable")
)

type PathStep struct {
	Edge   *Edge
	Entity *Entity
}

type ResolvedPath struct {
	Steps []PathStep
}

func (p ResolvedPath) Terminal(base *Entity) *Entity {
	if len(p.Steps) == 0 {
		return base
	}
	return p.Steps[len(p.Steps)-1].Entity
}

func (p ResolvedPath) MaxCardinality() Cardinality {
	for _, step := range p.Steps {
		if step.Edge.Cardinality.IsToMany() {
			return step.Edge.Cardinality
		}
	}
	return CardinalityOne
}

func (p ResolvedPath) CrossesToMany() bool {
	return p.MaxCardinality() != CardinalityOne
}

func PathKey(path []string) string {
	return strings.Join(path, ".")
}

func (c *Catalog) ResolvePath(baseEntity string, path []string) (*Entity, ResolvedPath, error) {
	base, ok := c.Entity(baseEntity)
	if !ok {
		return nil, ResolvedPath{}, fmt.Errorf("%w: %q", ErrUnknownEntity, baseEntity)
	}

	resolved := ResolvedPath{Steps: make([]PathStep, 0, len(path))}
	current := base

	for _, edgeName := range path {
		edge, edgeOK := current.Edge(edgeName)
		if !edgeOK {
			return nil, ResolvedPath{}, fmt.Errorf(
				"%w: %q on entity %q (path %q)",
				ErrUnknownEdge, edgeName, current.Key, PathKey(path),
			)
		}
		if !edge.Traversable {
			return nil, ResolvedPath{}, fmt.Errorf(
				"%w: %q on entity %q (path %q)",
				ErrEdgeNotTraversable, edgeName, current.Key, PathKey(path),
			)
		}

		target, targetOK := c.Entity(edge.Target)
		if !targetOK {
			return nil, ResolvedPath{}, fmt.Errorf(
				"%w: edge %q targets %q which is not in the catalog",
				ErrUnknownEntity, edgeName, edge.Target,
			)
		}

		resolved.Steps = append(resolved.Steps, PathStep{Edge: edge, Entity: target})
		current = target
	}

	return base, resolved, nil
}
