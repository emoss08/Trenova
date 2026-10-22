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
