package bootstrap_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/emoss08/trenova/internal/bootstrap/apiapp"
	"github.com/emoss08/trenova/internal/bootstrap/core"
	"github.com/emoss08/trenova/internal/bootstrap/workerapp"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aitrainingports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/stretchr/testify/assert"
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

func TestWiring_APIProcessesNeverHoldADocumentConverter(t *testing.T) {
	t.Parallel()

	for name, processOptions := range map[string]fx.Option{
		"api":        fx.Options(bootstrap.Options(), bootstrap.APIOptions()),
		"api-binary": fx.Options(core.Options(), apiapp.Options()),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := fx.ValidateApp(
				processOptions,
				fx.Invoke(func(kernelports.DocumentConverter) {}),
			)
			require.Error(t, err, "the API only enqueues extraction; it never calls the sidecar")
			assert.Contains(t, err.Error(), "kernelports.DocumentConverter")
		})
	}
}

func TestWiring_APIBinaryGraphResolves(t *testing.T) {
	t.Parallel()

	require.NoError(t, fx.ValidateApp(
		core.Options(),
		apiapp.Options(),
	))
}

func TestWiring_WorkerBinaryGraphResolves(t *testing.T) {
	t.Parallel()

	require.NoError(t, fx.ValidateApp(
		core.Options(),
		workerapp.Modules(),
		workerapp.Options(),
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
		"api":           fx.Options(bootstrap.Options(), bootstrap.APIOptions()),
		"worker":        fx.Options(bootstrap.Options(), bootstrap.WorkerOptions()),
		"api-binary":    fx.Options(core.Options(), apiapp.Options()),
		"worker-binary": fx.Options(core.Options(), workerapp.Modules(), workerapp.Options()),
	}

	workerReachable := fx.Invoke(func(extractionevalports.ExtractionPredictor) {})

	for name, processOptions := range options {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if name != "api-binary" {
				processOptions = fx.Options(processOptions, workerReachable)
			}
			if strings.HasPrefix(name, "worker") {
				processOptions = fx.Options(
					processOptions,
					fx.Invoke(func(kernelports.DocumentConverter) {}),
				)
			}

			require.NoError(t, fx.ValidateApp(
				processOptions,
				fx.Invoke(
					func(
						agentports.AgentEventPublisher,
						watchtowerports.WatchtowerProjector,
						// A trajectory recorder is optional at every use site, so
						// an installation missing it records nothing and says
						// nothing — the exact failure this file exists to catch.
						// It matters most in the worker, where the background runs
						// that had no durable account of themselves execute.
						agentports.AgentRunEventRecorder,
						agentports.QueryVectorizer,
						agentports.CatalogVectorIndex,
						agentports.RetrievalIndexer,
						agentports.RetrievalSearcher,
						agentports.MemoryVectorSearcher,
						agentports.MemoryRanker,
						kernelports.WorkflowSignalStarter,
						// Previews are optional to the runtime, the decision
						// services and the resolvers; without them nothing is
						// previewed, no baseline is kept and no digest checked.
						agentports.ProposalPreviewService,
						agentports.RecordLabeler,
						aicorrectionports.AICorrectionService,
						agentqualityports.EvaluationBudget,
						extractionevalports.ExtractionEvalRunStarter,
						extractionevalports.ExtractionEvalRunner,
						aitrainingports.AITrainingHistoryService,
						ediports.EditionInfo,
						platformcatalogports.EntitlementProvider,
						platformcatalogports.BillingProvider,
						platformcatalogports.UsageProvider,
						platformplanports.PlanService,
						platformplanports.QuotaGuard,
						// The change feed compares edited documents only when
						// the drift check is there; without it an edit waits
						// for the nightly run.
						accountingsyncports.AccountingDriftRechecker,
						accountingsyncports.AccountingDriftChecker,
						accountingsyncports.AccountingSyncDispatcher,
						proposalrecorder.PlanStore,
						agentports.AgentTrustService,
						agentports.AgentProposalNotifier,
						agentports.AgentActivityPublisher,
						inboundmessageports.TriageRuleEvaluator,
						inboundmessageports.TriageRuleTeacher,
						inboundmessageports.SenderHintProvider,
						inboundmessageports.TriageBriefingInvalidator,
						triageports.TriageCache,
						triageports.TriageConflictChecker,
						triageports.InboundCorrectionRecorder,
						triageports.InboundModelAvailability,
						triageports.TenderOverlapFinder,
						detentionports.BilledDetentionCreditor,
					) {
					},
				),
			))
		})
	}
}
