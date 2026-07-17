package compiler

import (
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

type joinedEntity struct {
	alias       string
	entity      *reportcatalog.Entity
	parentAlias string
	edge        *reportcatalog.Edge
}

type lateralMeasure struct {
	column    *validatedColumn
	innerName string
}

type lateralPlan struct {
	alias    string
	path     reportcatalog.ResolvedPath
	measures []*lateralMeasure
}

type joinPlan struct {
	aliases      map[string]string
	joins        []joinedEntity
	laterals     map[string]*lateralPlan
	lateralOrder []string
}

func (p *joinPlan) aliasFor(pathKey string) string {
	return p.aliases[pathKey]
}

func (c *Compiler) planJoins(v *validatedDef) (*joinPlan, error) {
	plan := &joinPlan{
		aliases:  map[string]string{"": "t0"},
		laterals: make(map[string]*lateralPlan),
	}

	toOnePrefixes := make(map[string][]string)
	lateralPaths := make(map[string]reportcatalog.ResolvedPath)

	for _, ref := range v.refs {
		if !ref.toMany {
			for i := 1; i <= len(ref.ref.Path); i++ {
				prefix := ref.ref.Path[:i]
				toOnePrefixes[reportcatalog.PathKey(prefix)] = prefix
			}
			continue
		}

		firstToMany := firstToManyIndex(ref.path)
		for i := 1; i <= firstToMany; i++ {
			prefix := ref.ref.Path[:i]
			toOnePrefixes[reportcatalog.PathKey(prefix)] = prefix
		}
	}

	for i := range v.columns {
		col := &v.columns[i]
		if col.spec.Kind == report.ColumnKindMeasure && col.ref.toMany {
			lateralPaths[col.ref.pathKey] = col.ref.path
		}
	}

	prefixKeys := make([]string, 0, len(toOnePrefixes))
	for key := range toOnePrefixes {
		prefixKeys = append(prefixKeys, key)
	}
	sort.Strings(prefixKeys)

	for i, key := range prefixKeys {
		alias := fmt.Sprintf("t%d", i+1)
		plan.aliases[key] = alias

		path := toOnePrefixes[key]
		parentKey := reportcatalog.PathKey(path[:len(path)-1])
		parentAlias, ok := plan.aliases[parentKey]
		if !ok {
			return nil, fmt.Errorf("join planning: parent prefix %q not aliased", parentKey)
		}

		_, resolved, err := c.catalog.ResolvePath(v.entity.Key, path)
		if err != nil {
			return nil, err
		}
		lastStep := resolved.Steps[len(resolved.Steps)-1]

		plan.joins = append(plan.joins, joinedEntity{
			alias:       alias,
			entity:      lastStep.Entity,
			parentAlias: parentAlias,
			edge:        lastStep.Edge,
		})
	}

	c.planLaterals(v, plan, lateralPaths)

	if len(plan.laterals) > c.limits.maxToManySubqueries {
		return nil, fmt.Errorf(
			"report requires %d to-many subqueries, exceeding the maximum of %d",
			len(plan.laterals), c.limits.maxToManySubqueries,
		)
	}
	if len(plan.joins) > c.limits.maxToOneJoins {
		return nil, fmt.Errorf(
			"report requires %d joins, exceeding the maximum of %d",
			len(plan.joins), c.limits.maxToOneJoins,
		)
	}

	return plan, nil
}

func (c *Compiler) planLaterals(
	v *validatedDef,
	plan *joinPlan,
	lateralPaths map[string]reportcatalog.ResolvedPath,
) {
	lateralKeys := make([]string, 0, len(lateralPaths))
	for key := range lateralPaths {
		lateralKeys = append(lateralKeys, key)
	}
	sort.Strings(lateralKeys)

	for i, key := range lateralKeys {
		plan.laterals[key] = &lateralPlan{
			alias: fmt.Sprintf("l%d", i),
			path:  lateralPaths[key],
		}
		plan.lateralOrder = append(plan.lateralOrder, key)
	}

	aggIndex := 0
	for i := range v.columns {
		col := &v.columns[i]
		if col.spec.Kind != report.ColumnKindMeasure || !col.ref.toMany {
			continue
		}
		lateral := plan.laterals[col.ref.pathKey]
		lateral.measures = append(lateral.measures, &lateralMeasure{
			column:    col,
			innerName: fmt.Sprintf("agg_%d", aggIndex),
		})
		aggIndex++
	}
}

func firstToManyIndex(path reportcatalog.ResolvedPath) int {
	for i, step := range path.Steps {
		if step.Edge.Cardinality.IsToMany() {
			return i
		}
	}
	return len(path.Steps)
}

func lateralPrefixKey(path []string, firstToMany int) string {
	return strings.Join(path[:firstToMany], ".")
}
