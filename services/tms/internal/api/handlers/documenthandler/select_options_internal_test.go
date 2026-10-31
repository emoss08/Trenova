package documenthandler

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	firstDocumentID  = "doc_01JQ0000000000000000000001"
	secondDocumentID = "doc_01JQ0000000000000000000002"
)

func TestSelectOptionIDs_ReadsEachIDOnce(t *testing.T) {
	t.Parallel()

	ids, err := selectOptionIDs(
		[]string{firstDocumentID, " " + secondDocumentID + " ", firstDocumentID},
	)

	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{firstDocumentID, secondDocumentID}, ids)
}

func TestSelectOptionIDs_AsksForNothingWithoutIDs(t *testing.T) {
	t.Parallel()

	ids, err := selectOptionIDs(nil)

	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestSelectOptionIDs_RefusesAnInvalidID(t *testing.T) {
	t.Parallel()

	_, err := selectOptionIDs([]string{firstDocumentID, "nope"})

	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestSelectOptionIDs_RefusesMoreThanOnePageOfIDs(t *testing.T) {
	t.Parallel()

	values := strings.Split(strings.Repeat(firstDocumentID+",", pagination.MaxLimit+1), ",")

	_, err := selectOptionIDs(values[:pagination.MaxLimit+1])

	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}
