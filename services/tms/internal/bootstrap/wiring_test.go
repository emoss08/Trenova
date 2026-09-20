package bootstrap_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

/*
The dependency graph has to resolve for every process, not just compile.

A tool that asked for *workerptoservice.Service compiled, unit-tested and ran
in the API — and took the Temporal worker down on startup, because that service
is provided through fx.As and only the port is in the graph. Nothing before the
process started could tell: go build type-checks the constructor, not whether
anything supplies its argument.

fx.ValidateApp walks the graph without constructing anything or touching a
database, so both entry points can be checked in a unit test at no cost. Had
this existed, the breakage would have been a red test rather than a failed
deploy.
*/
func TestWiring_APIGraphResolves(t *testing.T) {
	t.Parallel()

	require.NoError(t, fx.ValidateApp(
		bootstrap.Options(),
		bootstrap.APIOptions(),
	))
}

func TestWiring_WorkerGraphResolves(t *testing.T) {
	t.Parallel()

	require.NoError(t, fx.ValidateApp(
		bootstrap.Options(),
		bootstrap.WorkerOptions(),
	))
}
