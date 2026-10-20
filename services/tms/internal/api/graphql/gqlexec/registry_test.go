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
				{Object: "Query", Fields: []*Field{{Name: "thing"}, {Name: "__schema"}, {Name: "__type"}}},
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

const operationsTestSchema = `
type Thing { id: ID! name: String! owner: Owner }
type Owner { id: ID! name: String! }
input ThingFilter { name: String, nested: NestedFilter }
input NestedFilter { id: ID }
type Query { thing(filter: ThingFilter): Thing other: Owner }
`

func operationsSchema(t *testing.T) *ast.Schema {
	t.Helper()

	schema, err := gqlparser.LoadSchema(&ast.Source{Name: "operations", Input: operationsTestSchema})
	require.NoError(t, err)
	return schema
}

func operationsDocument(t *testing.T, schema *ast.Schema, query string) *ast.QueryDocument {
	t.Helper()

	doc, err := gqlparser.LoadQuery(schema, query)
	require.Empty(t, err)
	return doc
}

func thingShards(withOwner, withNested bool) []*Shard {
	inputs := []Input{{Name: "ThingFilter"}}
	if withNested {
		inputs = append(inputs, Input{Name: "NestedFilter"})
	}
	shards := []*Shard{{
		Name: "things",
		Objects: []*Object{
			{Name: "Thing", Implementors: []string{"Thing"}},
			{Name: "Query", Implementors: []string{"Query"}, Root: true},
		},
		Fields: []Fields{
			{Object: "Thing", Fields: []*Field{{Name: "id"}, {Name: "name"}, {Name: "owner"}}},
			{Object: "Query", Fields: []*Field{{Name: "thing"}}},
		},
		Inputs: inputs,
	}}
	if withOwner {
		shards = append(shards, &Shard{
			Name:    "owners",
			Objects: []*Object{{Name: "Owner", Implementors: []string{"Owner"}}},
			Fields:  []Fields{{Object: "Owner", Fields: []*Field{{Name: "id"}, {Name: "name"}}}},
		})
	}
	return shards
}

func TestValidateOperations_AcceptsAnExecutorHoldingOnlyWhatItsOperationsReach(t *testing.T) {
	t.Parallel()

	schema := operationsSchema(t)
	reg, err := Compile(thingShards(false, true))
	require.NoError(t, err)

	doc := operationsDocument(t, schema, `query Things { thing { id name } }`)

	assert.NoError(t, reg.validateOperations(schema, []*ast.QueryDocument{doc}, nil),
		"Query.other and Owner are never reached, so their shards are not needed")
	assert.Error(t, reg.validate(schema, nil), "the whole schema is not executable")
}

func TestValidateOperations_ReportsAReachedTypeWithNoExecutor(t *testing.T) {
	t.Parallel()

	schema := operationsSchema(t)
	reg, err := Compile(thingShards(false, true))
	require.NoError(t, err)

	doc := operationsDocument(t, schema,
		`query Things { thing { id ...OwnerFields } } fragment OwnerFields on Thing { owner { name } }`)

	err = reg.validateOperations(schema, []*ast.QueryDocument{doc}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "object Owner has no executor")
}

func TestValidateOperations_ReportsAnInputItCannotDecode(t *testing.T) {
	t.Parallel()

	schema := operationsSchema(t)
	reg, err := Compile(thingShards(true, false))
	require.NoError(t, err)

	doc := operationsDocument(t, schema, `query Things($f: ThingFilter) { thing(filter: $f) { id } }`)

	err = reg.validateOperations(schema, []*ast.QueryDocument{doc}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "input NestedFilter has no decoder")
}

func TestNewSchema_ValidatesOnlyItsOperationsWhenGiven(t *testing.T) {
	t.Parallel()

	schema := operationsSchema(t)
	reg, err := Compile(thingShards(false, true))
	require.NoError(t, err)
	doc := operationsDocument(t, schema, `query Things { thing { id } }`)

	assert.NotPanics(t, func() {
		NewSchema(SchemaConfig{Registry: reg, Parsed: schema, Operations: []*ast.QueryDocument{doc}})
	})
	assert.Panics(t, func() {
		NewSchema(SchemaConfig{Registry: reg, Parsed: schema})
	})
}

func TestCompile_RegistriesOverTheSameShardsDoNotShareObjects(t *testing.T) {
	t.Parallel()

	shards := registryShards()
	whole, err := Compile(shards)
	require.NoError(t, err)
	part, err := Compile(shards[:1])
	require.NoError(t, err)

	assert.NotNil(t, whole.objects["Query"].field("any"),
		"compiling a second registry leaves the first one's fields alone")
	assert.Nil(t, part.objects["Query"].field("any"))
}

// A field selected on an interface runs on whichever object the value is, so
// each possible object's executor must hold it.
func TestValidateOperations_ChecksAnInterfaceFieldOnEveryPossibleObject(t *testing.T) {
	t.Parallel()

	schema := gqlparser.MustLoadSchema(&ast.Source{Input: `
interface Named { name: String }
type Dog implements Named { name: String }
type Cat implements Named { name: String }
type Query { named: Named }
`})
	shard := func(catName bool) []*Shard {
		catFields := []*Field{}
		if catName {
			catFields = append(catFields, &Field{Name: "name"})
		}
		return []*Shard{{
			Name: "pets",
			Objects: []*Object{
				{Name: "Dog", Implementors: []string{"Dog", "Named"}},
				{Name: "Cat", Implementors: []string{"Cat", "Named"}},
				{Name: "Query", Implementors: []string{"Query"}, Root: true},
			},
			Fields: []Fields{
				{Object: "Dog", Fields: []*Field{{Name: "name"}}},
				{Object: "Cat", Fields: catFields},
				{Object: "Query", Fields: []*Field{{Name: "named"}}},
			},
			Abstracts: []Abstract{{Name: "Named"}},
		}}
	}
	doc := operationsDocument(t, schema, `query Named { named { name } }`)

	complete, err := Compile(shard(true))
	require.NoError(t, err)
	assert.NoError(t, complete.validateOperations(schema, []*ast.QueryDocument{doc}, nil))

	missing, err := Compile(shard(false))
	require.NoError(t, err)
	err = missing.validateOperations(schema, []*ast.QueryDocument{doc}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field Cat.name has no executor")
}
