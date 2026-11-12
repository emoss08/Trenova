package remedy_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/remedy"
	"github.com/stretchr/testify/assert"
)

func TestOf_ReadsTheRemedyARefusalNamesThroughItsWrapping(t *testing.T) {
	refusal := remedy.Mark(
		errortypes.NewBusinessError("No email profile is set up"),
		remedy.SetUpEmailProfile,
	)

	kind, ok := remedy.Of(fmt.Errorf("send the update: %w", refusal))

	assert.True(t, ok)
	assert.Equal(t, remedy.SetUpEmailProfile, kind)
}

func TestOf_NamesNoRemedyForARefusalThatHasNone(t *testing.T) {
	_, plain := remedy.Of(errortypes.NewBusinessError("The stop is already complete"))
	_, other := remedy.Of(errors.New("connection reset"))
	_, none := remedy.Of(nil)

	assert.False(t, plain)
	assert.False(t, other)
	assert.False(t, none)
}
