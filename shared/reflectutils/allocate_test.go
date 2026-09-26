package reflectutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/reflectutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type leaf struct {
	Name string
}

type holder struct {
	First  *leaf
	Second *leaf
	Kept   *leaf
	Value  leaf
	Names  []string
	hidden *leaf
}

func TestAllocatePointersFillsEveryNilExportedPointer(t *testing.T) {
	t.Parallel()

	kept := &leaf{Name: "kept"}
	target := holder{Kept: kept}

	require.NoError(t, reflectutils.AllocatePointers(&target))

	require.NotNil(t, target.First)
	require.NotNil(t, target.Second)
	assert.NotSame(t, target.First, target.Second)
	assert.Same(t, kept, target.Kept)
	assert.Nil(t, target.Names)
	assert.Nil(t, target.hidden)
}

func TestAllocatePointersRejectsANonStructPointer(t *testing.T) {
	t.Parallel()

	var value holder
	require.ErrorIs(t, reflectutils.AllocatePointers(value), reflectutils.ErrNotStructPointer)

	count := 3
	require.ErrorIs(t, reflectutils.AllocatePointers(&count), reflectutils.ErrNotStructPointer)

	var missing *holder
	require.ErrorIs(t, reflectutils.AllocatePointers(missing), reflectutils.ErrNotStructPointer)
}
