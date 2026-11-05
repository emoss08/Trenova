package authservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
)

func TestSameBrowser(t *testing.T) {
	t.Parallel()

	state := &tenantports.SSOLoginState{BrowserBinding: tokenutils.Hash("started-here")}

	assert.True(t, sameBrowser(state, "started-here"))
	assert.False(t, sameBrowser(state, "another-browser"))
	assert.False(t, sameBrowser(state, ""), "a callback with no cookie is not the same browser")
	assert.False(
		t,
		sameBrowser(&tenantports.SSOLoginState{}, ""),
		"a state with no binding never matches",
	)
}
