package bootstrap_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/emoss08/trenova/internal/core/ports/services"
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

/*
An optional dependency that nothing provides resolves to nil and the service
carries on without it. That is the point of `optional:"true"` — a projector or
a publisher is never worth failing a boot over — but it also means a desk can
be wired to a publisher that was never in the graph and simply never fire,
with no error anywhere. These assert the two best-effort ports are really
supplied, in both processes, by asking for them without the optional tag.
*/
func TestWiring_BestEffortPortsAreActuallyProvided(t *testing.T) {
	t.Parallel()

	options := map[string]fx.Option{
		"api":    bootstrap.APIOptions(),
		"worker": bootstrap.WorkerOptions(),
	}

	for name, processOptions := range options {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, fx.ValidateApp(
				bootstrap.Options(),
				processOptions,
				fx.Invoke(
					func(
						services.AgentEventPublisher,
						services.WatchtowerProjector,
						// A trajectory recorder is optional at every use site, so
						// an installation missing it records nothing and says
						// nothing — the exact failure this file exists to catch.
						// It matters most in the worker, where the background runs
						// that had no durable account of themselves execute.
						services.AgentRunEventRecorder,
						services.QueryVectorizer,
						services.CatalogVectorIndex,
						services.RetrievalIndexer,
						services.RetrievalSearcher,
						services.MemoryVectorSearcher,
						services.MemoryRanker,
						services.WorkflowSignalStarter,
						// Previews are optional to the runtime, the decision
						// services and the resolvers; without them nothing is
						// previewed, no baseline is kept and no digest checked.
						services.ProposalPreviewService,
						services.RecordLabeler,
						services.AICorrectionService,
						services.EvaluationBudget,
						services.ExtractionPredictor,
						services.ExtractionEvalRunStarter,
						services.ExtractionEvalRunner,
					) {
					},
				),
			))
		})
	}
}
