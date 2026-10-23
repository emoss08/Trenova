package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestList_JoinsItemsAsASentenceListsThem(t *testing.T) {
	t.Parallel()

	ctx := WithLocale(t.Context(), EN)
	assert.Empty(t, List(ctx, nil))
	assert.Equal(t, "code", List(ctx, []string{"code"}))
	assert.Equal(t, "code and name", List(ctx, []string{"code", "name"}))
	assert.Equal(t, "code, name and address", List(ctx, []string{"code", "name", "address"}))
}
