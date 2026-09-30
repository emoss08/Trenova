package gqlexec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

const registryTestSchema = `
interface Named { name: String! }
type Thing implements Named { id: ID! name: String! }
union Any = Thing
type Query { thing: Thing any: Any }
`

type thingResolver interface {
	Thing() string
}

type thingRoot struct{}

func (thingRoot) Thing() string { return "thing" }

func registryShards() []*Shard {
	return []*Shard{
		{
			Name: "things",
			Objects: []*Object{
				{Name: "Thing", Implementors: []string{"Thing", "Named"}},
				{Name: "Query", Implementors: []string{"Query"}, Root: true},
			},
			Fields: []Fields{
				{Object: "Thing", Fields: []*Field{{Name: "id"}, {Name: "name"}}},
				{Object: "Query", Fields: []*Field{{Name: "thing"}}},
			},
			Abstracts: []Abstract{{Name: "Named"}, {Name: "Any"}},
			Resolvers: []ResolverRequirement{{
				Root: "Query",
				Check: func(r any) bool {
					_, ok := r.(thingResolver)
					return ok
				},
			}},
		},
		{
			Name:   "extensions",
			Fields: []Fields{{Object: "Query", Fields: []*Field{{Name: "any"}}}},
		},
	}
}

func registrySchema(t *testing.T) *ast.Schema {
	t.Helper()

	schema, err := gqlparser.LoadSchema(&ast.Source{Name: "registry", Input: registryTestSchema})
	require.NoError(t, err)
	return schema
}

func TestCompile_MergesFieldsAcrossShards(t *testing.T) {
	t.Parallel()

	reg, err := Compile(registryShards())
	require.NoError(t, err)

	query := reg.objects["Query"]
	require.NotNil(t, query)
	assert.NotNil(t, query.field("thing"))
	assert.NotNil(t, query.field("any"))
	assert.NoError(t, reg.validate(registrySchema(t), map[string]func() any{
		"Query": func() any { return thingRoot{} },
	}))
}

func TestCompile_RejectsDuplicates(t *testing.T) {
	t.Parallel()

	tests := map[string]func(shards []*Shard) []*Shard{
		"shard": func(shards []*Shard) []*Shard {
			return append(shards, &Shard{Name: "things"})
		},
		"object": func(shards []*Shard) []*Shard {
			return append(shards, &Shard{Name: "dup", Objects: []*Object{{Name: "Thing"}}})
		},
		"field": func(shards []*Shard) []*Shard {
			return append(shards, &Shard{
				Name:   "dup",
				Fields: []Fields{{Object: "Thing", Fields: []*Field{{Name: "id"}}}},
			})
		},
		"abstract": func(shards []*Shard) []*Shard {
			return append(shards, &Shard{Name: "dup", Abstracts: []Abstract{{Name: "Any"}}})
		},
		"input": func(shards []*Shard) []*Shard {
			shards[0].Inputs = []Input{{Name: "In"}}
			return append(shards, &Shard{Name: "dup", Inputs: []Input{{Name: "In"}}})
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := Compile(mutate(registryShards()))
			assert.ErrorContains(t, err, "registered twice")
		})
	}
}

func TestCompile_RejectsExtensionOfUnknownObject(t *testing.T) {
	t.Parallel()

	shards := append(registryShards(), &Shard{
		Name:   "orphan",
		Fields: []Fields{{Object: "Missing", Fields: []*Field{{Name: "id"}}}},
	})

	_, err := Compile(shards)
	assert.ErrorContains(t, err, `extends unknown object "Missing"`)
}

func TestValidate_ReportsMissingExecutors(t *testing.T) {
	t.Parallel()

	shards := registryShards()
	shards[0].Fields[0].Fields = shards[0].Fields[0].Fields[:1]
	shards[0].Abstracts = shards[0].Abstracts[:1]

	reg, err := Compile(shards)
	require.NoError(t, err)

	err = reg.validate(registrySchema(t), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field Thing.name has no executor")
	assert.Contains(t, err.Error(), "abstract type Any has no executor")
}

func TestValidate_ReportsUnimplementedResolvers(t *testing.T) {
	t.Parallel()

	reg, err := Compile(registryShards())
	require.NoError(t, err)

	err = reg.validate(registrySchema(t), map[string]func() any{
		"Query": func() any { return struct{}{} },
	})
	assert.ErrorContains(t, err, "resolver Query does not implement the fields its schema declares")

	err = reg.validate(registrySchema(t), map[string]func() any{})
	assert.ErrorContains(t, err, "resolver root Query is not bound")
}

func TestValidate_SkipsResolverChecksWithoutResolvers(t *testing.T) {
	t.Parallel()

	reg, err := Compile(registryShards())
	require.NoError(t, err)

	assert.NoError(t, reg.validate(registrySchema(t), nil))
	assert.NoError(t, reg.validate(registrySchema(t), map[string]func() any{
		"Query": func() any { return nil },
	}))
}
