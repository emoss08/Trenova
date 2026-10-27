package tarifftypes_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestDeclsEqual_ComparesByNameAndValue(t *testing.T) {
	t.Parallel()

	a := []tarifftypes.InputDecl{
		{Name: "rate", Type: "Money/mi", Descriptor: rules.RawJSON(`{"kind":"money","unit":"mi"}`), Required: true},
		{Name: "minimumCharge", Type: "Money", Default: rules.RawJSON(`{"$q":"0","u":"$"}`)},
	}
	reordered := []tarifftypes.InputDecl{
		{Name: "minimumCharge", Type: "Money", Default: rules.RawJSON(`{"u":"$", "$q":"0"}`)},
		{Name: "rate", Type: "Money/mi", Descriptor: rules.RawJSON(`{"unit":"mi","kind":"money"}`), Required: true, Doc: "docs differ"},
	}
	assert.True(t, tarifftypes.DeclsEqual(a, reordered), "order, key order and docs do not matter")

	changed := []tarifftypes.InputDecl{a[0], {Name: "minimumCharge", Type: "Money"}}
	assert.False(t, tarifftypes.DeclsEqual(a, changed), "a dropped default is a change")
	assert.False(t, tarifftypes.DeclsEqual(a, a[:1]))
}

func TestImportsEqual(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("fl_")
	a := []tarifftypes.Import{{Path: "org.x", LibraryID: id, VersionNumber: 1}}
	assert.True(t, tarifftypes.ImportsEqual(a, []tarifftypes.Import{{Path: "org.x", LibraryID: id, VersionNumber: 1}}))
	assert.False(t, tarifftypes.ImportsEqual(a, []tarifftypes.Import{{Path: "org.x", LibraryID: id, VersionNumber: 2}}))
	assert.False(t, tarifftypes.ImportsEqual(a, nil))
}

func TestDeclsEqual_TreatsAConstraintAsPartOfTheDeclaration(t *testing.T) {
	t.Parallel()

	bounded := func(maxInclusive bool, max string) []tarifftypes.InputDecl {
		return []tarifftypes.InputDecl{{
			Name: "rate",
			Type: "Money/mi",
			Constraint: &rules.TariffConstraint{Range: &rules.TariffRangeConstraint{
				Min:          rules.RawJSON(`{"$q":"0.5","u":"$/mi"}`),
				Max:          rules.RawJSON(max),
				MaxInclusive: maxInclusive,
			}},
		}}
	}
	open := []tarifftypes.InputDecl{{Name: "rate", Type: "Money/mi"}}

	assert.True(t, tarifftypes.DeclsEqual(
		bounded(true, `{"$q":"15","u":"$/mi"}`),
		bounded(true, `{"u":"$/mi","$q":"15.00"}`),
	))
	assert.False(t, tarifftypes.DeclsEqual(open, bounded(true, `{"$q":"15","u":"$/mi"}`)),
		"adding a constraint is a change")
	assert.False(t, tarifftypes.DeclsEqual(
		bounded(true, `{"$q":"15","u":"$/mi"}`),
		bounded(false, `{"$q":"15","u":"$/mi"}`),
	), "an excluded maximum is a change")
	assert.True(t, tarifftypes.DeclsEqual(bounded(false, ``), bounded(false, `null`)),
		"an open end reads the same absent or null")
	assert.False(t, tarifftypes.DeclsEqual(bounded(false, ``), bounded(false, `{"$q":"9","u":"$/mi"}`)))

	listed := func(values ...string) []tarifftypes.InputDecl {
		oneOf := make([]rules.RawJSON, 0, len(values))
		for _, value := range values {
			oneOf = append(oneOf, rules.RawJSON(value))
		}
		return []tarifftypes.InputDecl{{
			Name:       "service",
			Type:       "Text",
			Constraint: &rules.TariffConstraint{OneOf: oneOf},
		}}
	}
	assert.True(t, tarifftypes.DeclsEqual(listed(`"A"`, `"B"`), listed(`"A"`, `"B"`)))
	assert.False(t, tarifftypes.DeclsEqual(listed(`"A"`, `"B"`), listed(`"A"`)))
	assert.False(t, tarifftypes.DeclsEqual(listed(`"A"`, `"B"`), listed(`"A"`, `"C"`)))
}

func TestInputDeclsFromKeepsTheConstraint(t *testing.T) {
	t.Parallel()

	constraint := &rules.TariffConstraint{OneOf: []rules.RawJSON{rules.RawJSON(`"A"`)}}
	decls := tarifftypes.InputDeclsFrom([]rules.TariffInput{{
		Name:       "service",
		Type:       "Text",
		Constraint: constraint,
	}})
	assert.Same(t, constraint, decls[0].Constraint)
}
