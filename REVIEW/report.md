# Trenova review: bugs, security, tests and lint

Branch `claude/sweet-johnson-ei4krw`, from `36d426b9`. Every fix is its own commit with a
regression test; the commit hashes below are on this branch. The method and the baseline
numbers are in [baseline.md](baseline.md).

## 1. Baseline and final state

| Check | Baseline (`36d426b9`) | Final (this branch) |
|---|---|---|
| Go unit tests (`services/tms`, `-tags=nomsgpack`) | 12 packages failing | FINAL_GO_UNIT |
| Go integration tests in the 11 packages this branch touches | not run (CI red) | all 11 pass on PostGIS 18 and Redis 8, including the two new ones |
| golangci-lint v2.12.2 as CI runs it (`--new-from-rev`), and archlint | not established (CI Lint dies at "Set up Go") | 0 new issues; archlint clean. A full run without `--new-from-rev` reports 704 issues, all pre-existing (CI gates on new issues only) |
| Codegen checks (gqlexec + e2e, agentread, schema stamp, projections, keel tables, safety doc, write coverage) | not run | all pass with no diff |
| Client lint, lint:design, lint:skeletons, single copies | 0 errors | 0 errors |
| Client typecheck (6 packages) | 0 errors | 0 errors |
| Client tests | `@trenova/web` 5 files / 6 tests failing | all pass: web 1,170 files / 9,816 tests, dash 14 / 69 (one test added), shared 152 / 1,784, edition 1 / 3 |
| Client codegen:check, guide:check, access:check, i18n check | not run | pass |
| `pnpm audit` | 1 high, 2 low | 2 low (no fix exists for one; see §6) |
| Rust `cargo fmt --check` (services, desktop/capture) | not run | pass (no Rust changed; clippy and tests not run here, see below) |
| `govulncheck` | not run | could not run: the environment's network policy denies `vuln.go.dev` |

What is still red and why:

- `internal/deskbench` `TestScenarioSplits_MatchTheLedger`: the split ledger needs a
  decision, not a fix (§5).
- `agent/app/agentevalgate` `Test*AgainstFloorsHybrid`: the stored query embeddings were
  recorded with an earlier embedding model and need re-recording against Ollama
  `nomic-embed-text`, which this environment cannot reach. Re-record with `-update`, then
  run without it.
- `internal/infrastructure/minio` fails in a clean run here only because Docker Hub rate
  limits the image pull (429). With the same image pulled through a registry mirror, all
  of its tests pass.

Not run here: Rust clippy and tests (no Rust changed; the imaging crates need the native
libraries in the imaging dev image, see `docs/engineering/rust.md`), the full 159-package
integration suite (only the 11 packages this branch touches were run, on plain PostGIS
without pgvector), and `govulncheck` (blocked host, and not part of CI either). CI runs
the first two.

## 2. Security findings

Ranked by impact. "Fixed" names the commit; everything else is documented with the change
it needs. Line numbers are at the baseline commit.

### Critical

None found. Tenancy holds: every repository path checked runs under `dbtx` with RLS, every
`WithSystem` site re-applies its tenant predicate, and no tenant id is taken from user or
model input.

### High

| # | Finding | Location | Impact | Status |
|---|---|---|---|---|
| H1 | A user with `user:update` could change another user's email (including an administrator's), then send a password reset or reset their MFA to take the account. No check compared the target's access with the actor's. | `userservice/patch.go:103`, `userresolver`, `userhandler` reset-password and MFA routes | Account takeover inside the tenant, privilege escalation to admin | **Fixed** `8243ca21`: changing another person's sign-in needs the actor's own access to cover theirs (sensitivity and every resource grant), and user management in each of the target's other organizations |
| H2 | `organization:update` could configure Okta and Microsoft SSO through the legacy routes, and `AllowFederatedMFA` was written as `true` whatever the request said. | `organizationhandler/handler.go:74-93`, `ssoconfigrepository/ssoconfig.go:218` | An org admin who is not an identity admin points sign-in at their own IdP, asserts any member's email and skips MFA | **Fixed** `113400f4`: routes need `identity_provider` read, create and update; the flag is saved as sent |
| H3 | API key rotation had no delegation check: `api_key:update` rotated anyone's key and received a token carrying the creator's power. | `apikeyservice/service.go:250-303`, `apikeyhandler/handler.go` | Privilege escalation through another user's key | **Fixed** `bd140ae8`: a rotation is checked against the rotator's own access; the handler now passes the acting user (update and rotate always failed before, and audited a nil actor) |
| H4 | AS2 inbound inflated compressed content with an unbounded `io.ReadAll`, before any signature check, on a public route. | `shared/as2/compress.go:81`, `message.go:303` | Unauthenticated memory exhaustion (decompression bomb) | **Fixed** `c519f8b8`: inflate reads through a limit (64 MiB default, configurable) and fails with `ErrDecompressedTooLarge` |
| H5 | Driver settlement could pay the same pay events twice: `MarkSettled` ignored how many rows it changed, so two settlements built at once both took the same accrued events. The carrier cost-event path only refused zero rows. | `driversettlementrepository/payevent.go:411-421`, `carriersettlementrepository/costevent.go:351` | Double payment to drivers and carriers | **Fixed** `e662121c`: ids are deduplicated and every one must change state, or the whole mark is refused |
| H6 | Customer payments posted, applied and reversed into Closed, Permanently Closed and (under the strict policy) Locked fiscal periods. | `customerpaymentservice/validator.go:117,297,439` | Silent changes to closed books | **Fixed** `2f573a93` |
| H7 | Invoice draft attachments accepted any document in the tenant and mailed it to any address. | `invoiceservice/drafts.go:54-63`, `invoicehandler` | A biller emails out a driver's drug test result or injury file | **Fixed** `8480b3cf`: each attachment needs document read and read on the record it belongs to |
| H8 | REST document reads check only `document:read`: no read on the owning record, and assistant-thread documents skip the visibility rule. `/documents/select-options/` has no permission check. | `documenthandler/handler.go:159-333`, `documentservice` List/Get/GetByResource/Download/View/Preview/PageImage, `documentextractionservice.GetContent` | Anyone with document read opens worker files (DOT drug and alcohol, clearinghouse, injury, leave, discipline); any user lists document names | **Not fixed**: needs the owner check `documentmailservice.requireReadable` already has, applied to every read path and to list filters, which means an owner-resource column to filter lists by. Too wide for a review commit; see §6 |
| H9 | The realtime stream sends full entities (carrier tax id, customer, user records) to every signed-in user in the tenant, regardless of permission. | `realtimebroker/subscriber.go:159`; publishers in `carrierservice`, `customerservice`, `userservice` | Data exposure across permission boundaries | **Not fixed**: publishers should send ids and let clients refetch through checked reads; a design change across every publisher |
| H10 | SSRF through an integration's `baseUrl`: no URL validation, and plain HTTP clients rather than `httpsafe`. Responses are reflected (PC*MILER test-connection error, email `LastError`). | `integrationservice.UpdateConfig`, `testers.go`, `shared/pcmiler`, email providers, `rampconnector`, `exchangerateservice`, `fuelsurchargeservice/eia.go`, samsara | Reads internal services and cloud metadata from the API host | **Not fixed**: route every integration client through `httpsafe` and validate the URL on save; touches eight clients |
| H11 | Foreign-currency invoices and payments post to the GL unconverted. | `invoiceledger.go:120,226`, `customerpaymentservice/service.go:192-277` | Misstated ledger balances | **Not fixed**: needs the functional-currency rule decided (rate source, rate date, realised FX) |

### Medium

| # | Finding | Location | Status |
|---|---|---|---|
| M1 | Changing your password left every other session signed in. | `userservice/service.go:642` | **Fixed** `74959baf` |
| M2 | Permission cache race: a check that computed from the old roles could write them back after `UserChanged` cleared the cache, and serve them for 30 minutes. | `permission/engine.go:1071-1135`, `permissioncache.go` | **Fixed** `bac53cea` (per-user epoch) |
| M3 | Bulk driver settlement actions checked `update` for Submit, Approve and Post; the single mutations need `submit` or `approve`. | `driversettlementresolver:596-601` | **Fixed** `41df2694` |
| M4 | Timesheet `RecordTimeEntry` with an existing entry id rewrote any worker's entry (authz checked the input worker, the service kept the stored one); `DeleteTimeEntry` ignored team scope. | `timesheetresolver`, `timesheetservice/service.go:391-402` | **Fixed** `fe693c7b` |
| M5 | A driver taken off a load kept access to its documents, comments and expenses. | `driverportalrepository/portal.go:487-504` | **Fixed** `59d1ce89` |
| M6 | The driver portal never cleared its query cache at logout, idle timeout or login: the next driver on a shared device saw the last one's pay, hours and profile. | `apps/dash` | **Fixed** `fe4625de` |
| M7 | Upload session `lineageId` was not checked against the record: an upload could become a new version of another record's document. | `documentuploadservice/service.go:182-193` | **Fixed** `5b50581a` |
| M8 | Cancelling a finished upload session deleted the live document's object. | `documentuploadservice/service.go:567-600` | **Fixed** `7accc4d1` |
| M9 | Exchange-rate routes had no permission checks; refresh calls a paid provider, settlement-quotes writes. Billing profile read had none either. | `exchangeratehandler`, `customerhandler` | **Fixed** `739abaa5` |
| M10 | CSV formula injection in the assistant's table export. | `assistantservice/artifact_export.go:225` | **Fixed** `2931b6af` |
| M11 | `selectOptions` took any number of ids and read each one. | `selectoptionsresolver/selectoptions.go:45` | **Fixed** `6850125e` |
| M12 | `/role-assignments/` had no permission check. | `roleassignmenthandler/handler.go:36-46` | **Fixed** `565bd94a` |
| M13 | EnforceSSO can be bypassed: password login is checked against the target organization only at login; switching organization checks membership only. | `authservice/service.go:1129-1166`, `userservice/service.go:265-325` | Not fixed: the switch should re-run the login policy of the organization switched into, and a password session refused there |
| M14 | AS2 inbound accepts unsigned EDI when the partner has no certificate or `requireSignedInbound` is off; async MDNs go to any URL with internal error text; AS2 identifier pairs are first come, first served globally. | `editransport/as2.go:129-133`, `ediinboundservice/as2.go` | Not fixed: refuse unsigned inbound by default, send async MDNs only to the partner's configured URL |
| M15 | JSON bodies are unbounded on public and protected routes (`ShouldBindJSON` without a limit). | `authhandler`, `passwordreset`, tender, rate confirmation, tracking, driver portal | Not fixed: a global body-limit middleware with named exemptions (the capture handler already has the helper) |
| M16 | Production start does not refuse placeholder secrets; a missing `APP_ENV` runs as development (debug, introspection, no HSTS). | `config/loader.go:211-216,757-802` | Not fixed: changes how operators deploy; recommend refusing both |
| M17 | Detention's per-day cap ignores frozen occurrences. | `detentionservice/service.go:212-218` | Not fixed: needs the billing rule confirmed |
| M18 | Timesheet workweek is anchored to UTC. | `timesheetservice/service.go:566,684`, `schedulingservice/rota.go:56` | Not fixed: needs the organization's time zone threaded through overtime |
| M19 | Write-only integration secrets are kept when the destination URL changes. | `secretconfig/codec.go:92-98`, `ediservice/profiles.go`, `factoringservice/transport.go` | Not fixed: apply AI providers' `SameCredentialTarget` rule |
| M20 | Unbounded `io.ReadAll` on provider responses. | `modeladapter`, `anthropicclient.go:102`, pcmiler, ramp, testers, exchange rate, EIA | Not fixed: wrap in `io.LimitReader` per client |
| M21 | Capture print-job split buffers every page in API memory. | `captureservice/batches.go:531-591` | Not fixed: stream through the imaging worker |
| M22 | Rate quote routes persist rows behind `rate_quote:read`, `carrierIds` unbounded, client-set shipment id. | `ratequotehandler`, `rateengine/service.go:439-447` | Not fixed |
| M23 | Manual journals and journal reversals can be approved by the person who made them. | `manualjournalservice/service.go:316-418`, `journalreversalservice` | Not fixed: a segregation-of-duties setting |
| M24 | The generic journal reversal can reverse a subledger's journal. | `journalreversalservice/validator.go:29-78` | Not fixed |
| M25 | Customer payment post and PayWorkerNow have no durable idempotency key. | `customerpaymentservice`, settlement | Not fixed |
| M26 | The web app keeps recent records, recent pages, desk searches, upload blobs and the full user in browser storage, keyed by org not user, and does not clear them at sign-out. | `recent-records-store.ts`, `recent-pages-store.ts`, `desk-settings-store.ts`, `document-upload-store.ts`, `auth-store.ts` | Not fixed: breaks the rule in `web-build.md`; each store needs to move off storage or into `clearSessionCache()` |
| M27 | Agent module: scheduled conversations and resumed waits run as a deactivated or locked user; portal live view hands any `portal_automation:read` user a vendor browser signed in with org credentials; agent definitions are saved without checking the editor holds the tools' permissions; proposal list and get skip the private-thread rule. | `conversationscheduleservice`, `agentwaitservice`, `agentextensionhandler`, `agentdefinitionservice`, `agentproposalrepository` | Not fixed; each is its own change |

### Low

Fixed: tokens in `webhookToken` and `mailboxToken` route params now redacted from logs
(`565bd94a`); the update check moved off the public routes and behind `organization:update`
(`5af8b829`); a dev dependency with a high-severity advisory (`e812f3f4`).

Not fixed: report builder shows other users' private report definitions within the tenant;
user and email-profile select options expose emails; `MyYtdPay` computes every worker then
filters; forgot-password timing differs for unknown emails; a replaced carrier's tracking
link stays readable after the move closes; `webhookToken` is tenant-settable and not unique;
`Accept: text/event-stream` skips the request timeout on any route; all RFC1918 proxies
trusted by default; Caddy CSP
allows `unsafe-eval`; dev compose binds `0.0.0.0`; `native-capture.yml` uses unpinned
actions with signing secrets; print-load-plan's inline script is blocked by CSP; three debug
`console.log`s; presigned URLs return sealed ciphertext; empty content types skip the allow
list; SSO state is not bound to the browser; the watchtower hand-off starts any enabled agent
without `MayUseAgent`; triage load instructions are stored as a system comment, so the
agent's taint tracking treats outside text as internal.

## 3. Bugs fixed

| Bug | File | Commit |
|---|---|---|
| A payment journal's header totals excluded the short pay, so the journal never balanced in manual mode; an apply with only a short pay wrote a zero line and returned 500 | `customerpaymentservice/service.go`, `plan_payment.go` | `5e465033` |
| The self-host config was refused at startup (edition-only `platform.controlPlane`, removed rate-limit keys) | `deploy/selfhost/config/config.yaml` | `fdc24d30` |
| `completionrouter` did not compile (pointer receiver on a value stored in a map) | `agentports/agentbatch.go` | `64d08f7c` |
| A business-day shift drifted by an hour across daylight saving | `recurringshipment/schedule.go` | `37ecfd13` |
| API key update and rotate always failed (no acting user passed) | `apikeyhandler/handler.go` | `bd140ae8` |
| SSO config ignored the `AllowFederatedMFA` the admin chose | `ssoconfigrepository/ssoconfig.go` | `113400f4` |
| The version handler's protected routes were never mounted and `/system/version` was registered twice | `versionhandler/handler.go`, `router.go` | `5af8b829` |
| After a forced password change, every request was refused until the next sign-in: the session kept `MustChangePassword` | `userservice/service.go` | `dbda1d93` |
| A config without `database.sslMode` failed to start: the default `prefer` is not one of the modes the config validates; the default is now `require` | `config/loader.go` | `fffefc56` |
| A failed session revocation after a password change skipped the audit entry | `userservice/service.go` | `d4970183` |
| Two tool descriptions exceeded their length limits | `carriertools/wait_tools.go`, `board_tools.go` | `64d08f7c` |
| The desk credential list read Worker fields outside the reference view | `worker-credential/desk.graphql`, `worker_record_reads.go` | `218e3c03` |

Every security fix in §2 marked **Fixed** is also a bug fix, with its own regression test.

## 4. Test changes

### The test was stale, so it was updated

| Test | Reason |
|---|---|
| `config` `TestLoad`, `TestSetDefaults` | the flat rate-limit keys were replaced by per-scope policies in `9c6ef059` |
| `agentquerytoolservice` `TestWorkerGet_…` (renamed `…FoldsComplianceIntoDispatchable`) | `b8a5934b` moved compliance from `canBeAssigned` to `dispatchable` |
| `agentquerytoolservice` `TestGetShipment_SaysWhenCommentsAndHoldsAreLeftOutForAccess` | `07712204` let the agent read comments and holds; the agent wording is now tested directly |
| `agentredteam` case `03_record_note.yaml` | fixture called `shipment_get` with `shipmentId`, renamed to `shipmentIds` in `ed9ef342`, so it errored and read nothing |
| `agentruntime` reply pass | the assertion "Call the carriers first." was superseded by the linked sentence in `b8a5934b` |
| web `data-table-board` pinned cell | pinned cells take the row's field fill since `44a7d8d3` |
| web `data-table-totals-row` | the fake `ResizeObserver` kept one global callback, which the ScrollArea's overwrote |
| web `quality-tab` | the answer sheet is modal, so the table behind it is `aria-hidden` |
| web `desk-decision-body` | mocked query factories lacked the `_def` scope the cache helpers read since `b8a5934b` |
| web `decision-flow-refused` | the queue list became virtualized and jsdom lays nothing out; the test stubs layout with the shared helper |
| `userservice`, `userhandler` ChangeMyPassword tests | the call takes the session scope now (and assert the new behaviour) |
| permission engine and cache fakes | `Epoch` takes the user and returns both epochs |
| `userresolver` harness | built the user service without the permission engine production always wires, which the sign-in check of `8243ca21` reads |
| `config` "database defaults" | asserted the `prefer` default the config's own validation refused; the new `TestAConfigWithoutAnSSLModeLoadsWithTLSRequired` failed against it |

### The code was wrong, so it was fixed

| Test | Fix |
|---|---|
| `gqlext` `TestEveryLinkedFieldIsDeclared` | the client operation stopped selecting fields outside the reference view |
| `agenttoolcatalog` `TestEveryToolDescribesItselfWellEnoughToBeChosen` | two descriptions shortened |
| `agenttoolservice` / `agentdefinition` template tests | two tools outside the agent allow-list removed from desk templates (marked `// REVIEW:`, §5) |
| `completionrouter` build | value receiver |
| `config` `TestSelfHostConfigFileLoads` (new) | the self-host file itself was invalid |

No test was deleted, skipped or loosened, and no lint suppression was added.

## 5. Needs decision

- **Desk template tools.** `carrier_set_status` was added to CarrierRiskDesk (its test says
  a risk desk must not change a carrier's status) and `masterdata_create_commodity` to
  IntakeDesk (not on the agent allow-list). Both are removed and marked `// REVIEW:` in
  `agentdefinition/tools.go`. Either keep them out, or add the tool to the allow-list and
  change the test that forbids it.
- **Desk credential list.** The operation no longer selects `Worker.canBeAssigned`,
  `assignmentBlocked` and `profile.complianceStatus`. If the desk needs them, make the fields
  nullable and `@requiresRead` so they can leave the reference view. That is a schema change.
- **Desk bench ledger.** `TestScenarioSplits_MatchTheLedger` fails because the messy-*
  scenarios were tuned against in run `20261027-messy`. Moving them to the tuning split, or
  replacing them with fresh held-out scenarios, is the maintainer's call.
- **Compliance Block.** `ComplianceEnforcementLevel` Block is never enforced on
  `AssignToMove`/`Reassign`; only the check route reads it, while the console treats it as
  a soft override. I recommend enforcing it on the server, with an override permission and
  an audited reason.
- **HOS reads.** Worker HOS queries ignore team scope and have no limit; the shipment board
  needs every state, so bound them by page and add the scope check.
- **API key revive on rotate.** Rotating a revoked key revives it; the test treats that as
  intended, so it was kept. Confirm.
- **Behaviour changes made, for review:** password change now ends other sessions; SSO
  routes need `identity_provider` rather than `organization:update`; exchange-rate, billing
  profile, role assignment and update-check routes now need permissions; a user patch that
  changes another person's sign-in needs the actor to cover their access.

## 6. Suggested follow-ups

1. Close H8: an owner-resource column on documents and one `requireReadable` on every read
   and list path.
2. H9: realtime events carry ids, not entities.
3. H10 and M20: one HTTP client factory for integrations, built on `httpsafe`, with a body
   limit, and URL validation on save.
4. M15: a global request-body limit middleware.
5. A REST counterpart to `authzlint` so a protected route without a permission check fails
   the build; several findings above (M9, M12) were exactly that gap.
6. Re-record the agent eval embeddings and settle the desk bench ledger so CI is green.
7. `katex` (low) is fixed by upgrading `remark-math`/`rehype-katex` to versions on katex
   0.18.2 or later. `deepmerge` (low) has no patched release; it is only reached through dev
   tooling (`shadcn`, rollup).
8. `pdf-lib` is unmaintained; it only generates cover sheets.
9. Add `govulncheck` to CI (Rust already runs `cargo deny`); the Go scan could not run
   here. GitHub reports open Dependabot alerts on the default branch
   (https://github.com/emoss08/Trenova/security/dependabot); triage them alongside it.
10. Burn down the 704 existing golangci-lint issues by linter, starting with `gosec`
    (50+) and `nilerr` (9), which are the ones that can hide bugs.

## Checks to run on the PR

- `cd services/tms && task test && task test-integration && task lint`
- the codegen job's steps (`task gqlgen`, `task db-stamp`, `task keel-gen`, the safety doc
  and write-coverage generators) should leave no diff
- `cd client && pnpm lint && pnpm typecheck && pnpm test`,
  `pnpm --filter @trenova/graphql codegen:check`, `guide:check`, `access:check`
- `node i18n/tools/i18n.mjs check`
- `task rust:fmt && task rust:lint && task rust:test` (no Rust changed)
