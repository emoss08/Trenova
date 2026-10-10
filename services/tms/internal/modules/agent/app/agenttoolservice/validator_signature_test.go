package agenttoolservice

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

/*
The runtime and the proposal executor find a tool's checks by asserting
serviceports.ToolValidator, whose Validate takes a context and the execute
params. A method named Validate with any other signature compiles, passes its
own unit tests, and is never called: six tools shipped that way, so a dashboard
tile on a missing report or a schedule with a bad cron was proposed, approved,
and only then failed.

The type assertion is silent by design, so the check lives here.
*/
func TestEveryToolValidateMethodIsOneTheRuntimeCalls(t *testing.T) {
	t.Parallel()

	method := regexp.MustCompile(`func \(t \*(\w+)\) Validate\(([^)]*)`)
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	wrong := make([]string, 0)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, readErr := os.ReadFile(name)
		require.NoError(t, readErr)

		for _, match := range method.FindAllStringSubmatch(string(source), -1) {
			if !strings.Contains(match[2], "context.Context") {
				wrong = append(wrong, match[1]+" in "+name)
			}
		}
	}

	require.Emptyf(t, wrong,
		"these tools define Validate with a signature serviceports.ToolValidator does not "+
			"match, so nothing ever calls it: %v", wrong,
	)
}
