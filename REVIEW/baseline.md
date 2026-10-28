# Baseline

Taken at `36d426b9` (the tip of `master` when the review started), before any change on
this branch. The commands are the repository's own (`services/tms/Taskfile.yml`,
`client/package.json`, `.github/workflows/*.yml`).

## Summary

| Check | Command | Result at baseline |
|---|---|---|
| Go unit tests | `GOFLAGS=-tags=nomsgpack go test -p 2 ./...` (services/tms) | **12 packages failing** out of 864 with tests (852 pass) |
| Go integration tests | `go test -tags=integration,nomsgpack …` | not run locally at baseline (needs Docker images; CI job red) |
| Go lint | `golangci-lint run ./...`, `go run ./scripts/archlint` | not established: the CI Lint job fails at "Set up Go" before linting |
| Client lint | `pnpm lint`, `lint:design`, `lint:skeletons`, `check:single-copies` | **0 errors** in all four |
| Client typecheck | `pnpm typecheck` (6 packages) | **0 errors** |
| Client tests | `pnpm test` | **5 files / 6 tests failing** in `@trenova/web` (1,170 files, 9,816 tests); dash, shared and edition all pass |
| Dependency audit (client) | `pnpm audit` | 1 high (happy-dom), 2 low (katex, deepmerge) |

CI at the baseline commit: Test Go had Unit and Integration failing and Lint and the
no-pgvector job dying at "Set up Go"; Test Client had only Test failing; Agent Evals failed.

## Go unit tests, by area

A first full run filled the disk (14 GB build cache) and 385 more packages failed with
`no space left on device`; those were re-run with `-p 2` and all passed. The 12 real
failures:

| Area | Package | Failing tests | Cause found |
|---|---|---|---|
| GraphQL guard | `internal/api/graphql/gqlext` | `TestEveryLinkedFieldIsDeclared` | a client operation selected Worker fields outside the reference view |
| GraphQL / build | `infrastructure/agentcompletion/completionrouter` | build failed | pointer receiver on a type stored by value in a map |
| Config | `internal/infrastructure/config` | `TestLoad`, `TestSetDefaults` | tests used the flat rate-limit keys removed in `9c6ef059` |
| Storage | `internal/infrastructure/minio` | 17 `TestClient_*` and `TestTemporalPayloadStore_*` | Docker Hub rate limit (429) pulling the test image: environmental |
| Agent | `agent/app/agentquerytoolservice` | `TestGetShipment_SaysWhenCommentsAndHoldsAreLeftOutForAccess`, `TestWorkerGet_ReadsTheCardAndFoldsComplianceIntoAssignability` | tests predate `07712204` and `b8a5934b` |
| Agent | `agent/app/agentredteam` | `TestRedTeam` | case fixture used a parameter renamed in `ed9ef342` |
| Agent | `agent/app/agentruntime` | `TestPointToTables_PointsEveryReprintToItsOwnTable` (reply pass) | assertion superseded by `b8a5934b` |
| Agent | `agent/app/agenttoolcatalog` | `TestEveryToolDescribesItselfWellEnoughToBeChosen` | two tool descriptions over their length limits |
| Agent | `agent/app/agenttoolservice`, `agent/domain/agentdefinition` | `TestEveryToolADeskRunsOnIsOneAnAgentMayCall`, `TestTemplates_TheNewDesksHoldWhatTheirWorkNeedsAndNothingThatLeaves` | desk templates gained tools outside the agent allow-list |
| Agent evals | `agent/app/agentevalgate` | `TestGuideSelectionAgainstFloorsHybrid`, `TestToolSelectionAgainstFloorsHybrid` | embedding fixture stale (needs a re-record against Ollama) |
| Desk bench | `internal/deskbench` | `TestScenarioSplits_MatchTheLedger` | split ledger out of date with tuned scenarios |

## Client tests (`@trenova/web`)

| File | Test |
|---|---|
| `components/data-table/__tests__/data-table-board.test.tsx` | gives a pinned cell the row's field fill when the row is hovered, under the cursor or open |
| `components/data-table/__tests__/data-table-totals-row.test.tsx` | keeps the reserved space in step when the row grows |
| `routes/agent-control/_components/quality/__tests__/quality-tab.test.tsx` | opens what a person saw when they rated an answer down |
| `routes/desk/_components/artifacts/kinds/decision/__tests__/desk-decision-body.test.tsx` | says nothing changed and reads as not carried out |
| `routes/desk/_components/decisions/__tests__/decision-flow-refused.test.tsx` | does not say it again when the decisions are read back with the change failed |
| same file (flaky: failed once in three isolated runs) | says at once that nothing was changed and takes the change off the queue |

## Areas reviewed

Phase 2 split the code by risk and reviewed each area in parallel: authentication and
permissions, the public (unauthenticated) surface, REST handlers, GraphQL, data access and
tenancy, money (billing, settlement, accounting), documents and uploads, integrations
(EDI, AS2, webhooks, providers), operations (shipments, dispatch, recurring), the agent and
AI module, the frontend, and infrastructure and deployment.
