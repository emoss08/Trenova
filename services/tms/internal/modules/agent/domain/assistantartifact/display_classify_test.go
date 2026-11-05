package assistantartifact

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAppPath_AcceptsPathsInsideTheApp(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/",
		"/billing/invoices",
		"/hr/workers?panelType=edit&panelEntityId=wrk_1",
		"/billing/invoices?next=//example.com",
		"/search?q=100%",
	} {
		assert.True(t, IsAppPath(path), "%q", path)
	}
}

func TestIsAppPath_RefusesWhatABrowserReadsAsAnotherHost(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"",
		"billing/invoices",
		"//example.com/x",
		"///example.com",
		"/\\example.com",
		"/\t/example.com",
		"/\n/example.com",
		"/\r\n/example.com",
		"/\t\\example.com",
		"https://example.com",
		"javascript:alert(1)",
	} {
		assert.False(t, IsAppPath(path), "%q", path)
	}
}
