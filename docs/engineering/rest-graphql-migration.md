# REST to GraphQL migration (working plan)

Delete this file when the migration is finished, after moving the lasting rule (what stays REST and why) into
[backend/api.md](backend/api.md).

**Status: paused after batch 1.** The "Open issues to check first" below were resolved on 2026-11-01
(i18n, page-access manifests, client tests, lint and bundle budget). Batches 2–14 have not started; the
routes they list still serve the client over REST. Resume at batch 2 with the per-batch loop.

## Goal

Every REST route the web app, driver portal or shared client calls moves to GraphQL (queries for reads and
for POSTs that write nothing, mutations for writes), except the routes listed under "What stays REST".
Each converted route is deleted with its handler code and tests; a handler with no routes left is deleted
with its wiring.

## Done

- 206 REST routes no client called were deleted, with 11 handler packages and the service methods only
  they called (invoice-from-order, permission simulator, manual service-failure create, comment typing
  signal, share-candidate lookup, EDI profile get/list, detention occurrence list; restorable from git).
- Batch 1 (reference-data CRUD: accessorial charges, account types, commodities, customers, equipment
  manufacturers, fleet codes, hazardous materials, hazmat segregation rules, hold reasons, document types,
  factoring companies, location categories) is converted: 41 routes, 12 handler packages deleted, Go vet,
  tests and golangci-lint clean on the touched packages, `task gqlgen`, projections, write coverage and
  client codegen regenerated.

## Open issues to check first

- Client unit tests at the time of handoff: 73 failures in 23 files, mostly `routes/billing-queue`,
  `routes/desk/_components/artifacts/cards/*` (settlements, fuel card, detention occurrence, invoice),
  `routes/dispatch-console`, `hooks/realtime-desk-flood`, `hooks/data-table/use-data-table-layout`,
  `components/navigation/user-menu`, `organization-settings-loading`. They sit mostly in areas of the recent
  WIP redesign commits; confirm which predate this migration (run them at the parent commit) and fix the ones
  it caused.
- `pnpm --filter @trenova/web typecheck` has two errors outside this work:
  `routes/shipment/_components/rating-breakdown-popover.tsx` (`bill.currency`) and
  `routes/shipment/_components/move/__tests__/move-headline.test.ts` (`addressLine: null`).
- `pkg/pageaccess/screens_gen.json` / `manifest_gen.json` still list deleted REST paths: run
  `pnpm --filter @trenova/web access:generate`, then the manifest build
  (`go run ./internal/api/graphql/gqlext/refgen` from services/tms if that is the generator; check
  `internal/api/pagemanifest`).
- `task i18n` for new caller-facing strings, then `task i18n-check`.
- The projection generator (`go generate ./internal/api/graphql/projection/...`) was failing at the
  WIP commit; `BillingQueueHeadline`, `CarrierIntelRuleSetting` (DTOs) and virtuals
  `BillingQueueItem.headline`, `RateAgreementAccessorial.chargeCode/chargeDescription`,
  `RateAgreementFuelBinding.programName` were added so it runs. Keep it green after each batch.

## Per-batch loop

1. Convert the batch following the playbook below.
2. From services/tms: `go run ./internal/api/graphql/gqlexec/gen` then `go run ./internal/api/graphql/agentread/gen`
   (that is `task gqlgen`), `go generate ./internal/api/graphql/projection/...`,
   `go generate ./internal/api/writecoverage/...`. All with `GOFLAGS=-tags=nomsgpack`.
3. From client: `pnpm --filter @trenova/graphql codegen`.
4. `GOFLAGS=-tags=nomsgpack go vet` and `go test` on the touched packages plus
   `./internal/api/writecoverage/ ./internal/api/graphql/authzlint/ ./internal/api/middleware/
   ./internal/api/responseguard/ ./internal/api ./internal/bootstrap/modules/api`.
5. `golangci-lint run --new-from-rev=<base>` and `golangci-lint run --enable-only=unused,whitespace` on the
   touched packages; `golangci-lint fmt <files>` for formatting.
6. `pnpm --filter @trenova/web typecheck`, and `pnpm exec vitest related --run <changed files>` from
   client/apps/web (run it in the background: it can exceed ten minutes).
7. Commit the batch.

## Playbook

Repo: /home/wolf/projects/Trenova. Server: services/tms (Go). Client: client/ (pnpm).
Read first: AGENTS.md, docs/engineering/backend/api.md, docs/engineering/graphql-executor.md,
.claude/skills/trenova-backend/SKILL.md and .claude/skills/trenova-frontend/SKILL.md.
Run builds, vet, tests and lint **one package (or a small explicit list) at a time**, never the whole
module (a full build takes ~30 minutes and the machine can run out of memory). Never run two heavy jobs
at once. Format only the files you changed, by path.

## What stays REST (do not convert)
- Auth/session: everything under `/auth/`, `/users/me/change-password`, `/users/me/switch-organization`.
- Unauthenticated: public token links (tender-offers, rate-confirmation-links, carrier-tracking-links,
  shipment-tracking-links, shared-documents), driver portal invitations preview/accept, `/system/public-config`,
  `/system/version`.
- Called by third parties or the desktop app: webhooks, AS2, `/capture/` device + pairing routes.
- Binary transfer: anything that uploads a file (multipart or upload sessions, including the whole
  `/documents/uploads/` protocol), or returns a file/bytes (download, view, preview, content, page image,
  PDF, CSV/export, transcript, recording, template download). A route that returns JSON *about* a file
  is not binary and converts.
- Server-sent event streams (`text/event-stream`), and presence join/leave tied to the realtime connection.
Report every route you keep, with the reason.

## Converting a route
For each REST route the client uses (listed in your brief):

1. **Find an existing GraphQL operation first.** Search `internal/api/graphql/schema/*.graphqls` for a
   query/mutation over the same service call (e.g. `createTrailer` already exists while the client still
   POSTs `/trailers/`). If one exists, only switch the client and delete the route.
2. **Otherwise add one** to the schema file of that domain (`extend type Query`/`extend type Mutation`
   or the file's `type Query`/`type Mutation`):
   - Reads → `Query`. **A POST that writes nothing (preview, calculate, inspect, explain, match,
     backtest, compare, readiness) is a `Query`**, never a mutation: GraphQL mutations are refused on
     read-only plans (`gqlext/readonly.go`), queries are not.
   - Writes → `Mutation`, verb-first names (`approveFormulaTemplate`, `bulkUpdateCustomerStatus`).
   - Reuse existing GraphQL types and domain bindings (`gqlgen.yml models:`). A new output type bound to
     an existing Go struct goes in `gqlgen.yml` (pattern: `ShipmentCustomerBillingRules` →
     `customerports.ShipmentBillingRules`); if it is a DTO with no keel table, add it to
     `nonProjectionObjects` in `internal/api/graphql/projection/gen/generator.go` with a reason.
     Prefer `input XInput` types for mutation arguments. Scalars: `Timestamp` for unix seconds,
     `Decimal` for money/quantities, `JSON` for free-form maps, `ID` for ids.
   - Lists the client pages through → connections with `DataTableConnectionInput` like the domain's
     existing table query; small bounded lists → plain lists.
3. **Resolver** in `internal/api/graphql/resolver/<schemafile>resolver/<schemafile>.resolvers.go`,
   matching the generated signature gqlgen will produce (args in schema order, `ctx` first, model
   types from `gqlmodel` or the bound Go type). Start with the **same permission check the REST route
   had** (`authCtx, err := r.RequirePermission(ctx, permission.ResourceX, permission.OpY)`), scope with
   `resolvercore.TenantInfo(authCtx)`, take the actor from `actorutil.FromAuthContext(authCtx)`, call the
   **same service method the REST handler called**. Parse ids with `pulid.MustParse` / `gqlid.ParseIDs`.
   A service the package does not have yet: use it as `r.ServiceName` where `ServiceName` is a field in
   `internal/api/graphql/resolver/services.go` `Services`; if it is not there, add it to `Params`,
   `Services` and `newServices` in that file (the generator wires `Deps`). Never import another
   resolver package. Self-scoped operations (`My<Thing>`) are listed in
   `internal/api/graphql/selfscoped` with a reason. Mutations allowed while the plan is read-only (only
   a person's own preferences/settings, stopping their own Desk turn) go in the allowlist in
   `internal/api/graphql/gqlext/readonly.go` (add one if missing: a `map[string]struct{}` of mutation
   field names consulted in `MutateOperationContext`).
4. **Write coverage**: every new mutation gets an entry in `internal/api/writecoverage/writecoverage.yml`
   keyed `"mutation <name>":`, carrying over the decision of the REST route it replaces (tools/exempt/
   pending); delete the REST route's entry. Queries get none.
5. **Delete the REST route**: its registration in the handler's `RegisterRoutes`, the handler method,
   now-unused helpers/types in the handler package, and handler tests for it. If the handler has no
   routes left, delete the package and its wiring in `internal/api/router.go` and
   `internal/bootstrap/modules/api/handlers.go`. Remove its path from `readOnlyWriteAllowedPaths` in
   `internal/api/middleware/readonly.go` if listed. Keep service/port/repo methods the resolver now calls.
6. **Tests**: add resolver tests in the resolver package (`*_test.go`, package-internal) for each new
   resolver: permission checked (use `resolvertest.RecordingPermissionEngine` /
   `mocks.DenyResourcesPermissionEngine`), input mapped to the service request, output mapped. Build
   resolvers as `&QueryResolver{&Deps{Core: &resolvercore.Core{PermissionEngine: engine}, X: svc}}`
   (`&MutationResolver{...}` for mutations). Follow existing tests in the same package.

## Client
7. Add operations in `client/packages/graphql/src/operations/<domain>/<name>.graphql` (follow the
   existing files there). Generated names after codegen: `<OperationName>Document`,
   `<OperationName>Query`/`<OperationName>Mutation` and `...Variables`, imported from
   `@trenova/graphql/generated/graphql`.
8. Switch every caller: services in `client/apps/web/src/services/*.ts` (and `client/apps/dash`,
   `client/packages/shared`) call `requestGraphQL({ document, operationName, variables, signal })` from
   `@trenova/shared/lib/graphql`; put GraphQL helpers in `client/apps/web/src/lib/graphql/<domain>.ts`
   when the domain has one. Query wrappers take and forward `{ signal }`. Form panels pass
   `mutationFn` instead of `url` (see `routes/worker-credential-type/_components/credential-type-panel.tsx`).
   Keep return shapes the callers expect, or update the callers. Delete REST service methods and REST
   types/schemas nothing uses.
9. Update client tests that mocked the REST service (mock `@trenova/shared/lib/graphql` `requestGraphQL`
   by `operationName`, or the lib/graphql helper) and add tests for changed behaviour.
10. Format the files you changed by path: `gofmt -w`, `goimports -w`, `golines -m 100 -w` for Go;
    `client/apps/web/node_modules/.bin/oxfmt <files>` (run from `client/`) for TS. Go: no comments
    (except keel model docs), `sonic` not `encoding/json`, `errortypes` for caller-facing errors.

## Report back
List: GraphQL operations added (name, kind, permission), existing operations reused, REST routes
deleted, REST routes kept and why, services.go fields added, writecoverage entries changed, files
changed, anything you could not finish or are unsure of (be specific).


## Remaining batches

Routes each client calls, by handler (paths omit `/api/v1`). Re-check a handler's routes against the code
before converting: other work may have changed them.

### Batch 2: Places, equipment, workers, carriers, rating zones, custom fields, distance

```text
## locationhandler
  GET /locations/
  POST /locations/
  PATCH /locations/:locationID/
  POST /locations/bulk-update-status/
## servicetypehandler
  POST /service-types/
  PATCH /service-types/:serviceTypeID/
  PUT /service-types/:serviceTypeID/
  POST /service-types/bulk-update-status/
## shipmenttypehandler
  POST /shipment-types/
  PATCH /shipment-types/:shipmentTypeID/
  PUT /shipment-types/:shipmentTypeID/
  POST /shipment-types/bulk-update-status/
## tractorhandler
  POST /tractors/
  PATCH /tractors/:tractorID/
  PUT /tractors/:tractorID/
  POST /tractors/bulk-update-status/
## trailerhandler
  POST /trailers/
  PATCH /trailers/:trailerID/
  PUT /trailers/:trailerID/
  POST /trailers/:trailerID/locate/
  POST /trailers/bulk-update-status/
## workerhandler
  POST /workers/
  PATCH /workers/:workerID/
  PUT /workers/:workerID/
## carrierhandler
  GET /carriers/
  POST /carriers/
  GET /carriers/:carrierID
  PATCH /carriers/:carrierID/
  PUT /carriers/:carrierID/
  POST /carriers/bulk-update-status/
## ratezonehandler
  GET /rate-zones/
  POST /rate-zones/
  GET /rate-zones/:rateZoneID/
  PUT /rate-zones/:rateZoneID/
## customfieldhandler
  POST /custom-fields/definitions/
  DELETE /custom-fields/definitions/:definitionID/
  PATCH /custom-fields/definitions/:definitionID/
  PUT /custom-fields/definitions/:definitionID/
## distanceoverridehandler
  POST /distance-overrides/
  DELETE /distance-overrides/:distanceOverrideID/
  PATCH /distance-overrides/:distanceOverrideID/
  PUT /distance-overrides/:distanceOverrideID/
## distanceprofilehandler
  POST /distance-profiles/
  DELETE /distance-profiles/:distanceProfileID/
  PATCH /distance-profiles/:distanceProfileID/
  PUT /distance-profiles/:distanceProfileID/
  POST /distance-profiles/:distanceProfileID/set-default/
## servicefailurereasoncodehandler
  GET /service-failure-reason-codes/
  POST /service-failure-reason-codes/
  GET /service-failure-reason-codes/:reasonCodeID/
  PUT /service-failure-reason-codes/:reasonCodeID/
  POST /service-failure-reason-codes/:reasonCodeID/activate/
  POST /service-failure-reason-codes/:reasonCodeID/archive/
  POST /service-failure-reason-codes/reorder/
## storedmileagehandler
  DELETE /stored-mileages/:storedMileageID/
## sequenceconfighandler
  GET /sequence-configs/
  PUT /sequence-configs/
```

### Batch 3: Controls, settings, preferences, search, analytics, system

```text
## accountingcontrolhandler
  GET /accounting-controls/
  PUT /accounting-controls/
## billingcontrolhandler
  GET /billing-controls/
  PUT /billing-controls/
## dataentrycontrolhandler
  GET /data-entry-controls/
  PUT /data-entry-controls/
## dataretentionhandler
  GET /data-retention/
  PUT /data-retention/
## dispatchcontrolhandler
  GET /dispatch-controls/
  PUT /dispatch-controls/
## distancecontrolhandler
  GET /distance-controls/
  PATCH /distance-controls/
## documentcontrolhandler
  GET /document-controls/
  PUT /document-controls/
## invoiceadjustmentcontrolhandler
  GET /invoice-adjustment-controls/
  PUT /invoice-adjustment-controls/
## shipmentcontrolhandler
  GET /shipment-controls/
  PUT /shipment-controls/
## notificationpreferencehandler
  GET /users/me/notification-preferences/
  PUT /users/me/notification-preferences/
## pagefavoritehandler
  GET /page-favorites/
  GET /page-favorites/check
  POST /page-favorites/toggle
## casechecklisthandler
  GET /case-checklists/
  PUT /case-checklists/
  DELETE /case-checklists/:templateID/
## weatheralerthandler
  GET /weather-alerts/
  GET /weather-alerts/:alertID/
## versionhandler
  POST /system/check-updates
  GET /system/update-status
  GET /system/version
## googlemapshandler
  POST /google-maps/autocomplete/
## exchangeratehandler
  GET /exchange-rates/convert
  GET /exchange-rates/latest
  POST /exchange-rates/refresh
  POST /exchange-rates/settlement-quotes
## pushhandler
  GET /push/public-key/
  DELETE /push/subscriptions/
  POST /push/subscriptions/
## onboardinghandler
  GET /onboarding/
  POST /onboarding/complete/
## searchhandler
  GET /search/global/
## analyticshandler
  GET /analytics/
## tablequeryhandler
  GET /tables/
  POST /tables/:resource/compose/
## tablechangealerthandler
  GET /tca/allowlisted-tables/
  GET /tca/subscriptions/
  POST /tca/subscriptions/
  DELETE /tca/subscriptions/:id
  GET /tca/subscriptions/:id
  PUT /tca/subscriptions/:id
  PATCH /tca/subscriptions/:id/pause
  PATCH /tca/subscriptions/:id/resume
```

### Batch 4: Accounting

```text
## fiscalperiodhandler
  PUT /fiscal-periods/:fiscalPeriodID/activate/
  GET /fiscal-periods/:fiscalPeriodID/close-blockers/
  PUT /fiscal-periods/:fiscalPeriodID/close/
  PUT /fiscal-periods/:fiscalPeriodID/lock/
  PUT /fiscal-periods/:fiscalPeriodID/reopen/
  PUT /fiscal-periods/:fiscalPeriodID/unlock/
## fiscalyearhandler
  POST /fiscal-years/
  PUT /fiscal-years/:fiscalYearID/
  PUT /fiscal-years/:fiscalYearID/activate/
  GET /fiscal-years/:fiscalYearID/close-preview/
  PUT /fiscal-years/:fiscalYearID/close/
  PUT /fiscal-years/:fiscalYearID/reopen/
## glbalancehandler
  GET /accounting/statements/balance-sheet/:fiscalPeriodID/
  GET /accounting/statements/income-statement/:fiscalPeriodID/
  GET /accounting/trial-balance/:fiscalPeriodID/
## journalreversalhandler
  GET /accounting/journal-reversals/
  POST /accounting/journal-reversals/
  GET /accounting/journal-reversals/:reversalID/
  POST /accounting/journal-reversals/:reversalID/approve/
  POST /accounting/journal-reversals/:reversalID/cancel/
  POST /accounting/journal-reversals/:reversalID/post/
  POST /accounting/journal-reversals/:reversalID/reject/
## manualjournalhandler
  GET /accounting/manual-journals/
  GET /accounting/manual-journals/:requestID/
  POST /accounting/manual-journals/:requestID/approve/
  POST /accounting/manual-journals/:requestID/cancel/
  POST /accounting/manual-journals/:requestID/post/
  POST /accounting/manual-journals/:requestID/reject/
  POST /accounting/manual-journals/:requestID/submit/
  POST /accounting/manual-journals/drafts/
  PUT /accounting/manual-journals/drafts/:requestID/
## accountsreceivablehandler
  GET /accounting/accounts-receivable/customers/:customerID/ledger/
  GET /accounting/accounts-receivable/customers/:customerID/statement/
## collectionhandler
  GET /accounting/collections/customers/:customerID/
  POST /accounting/collections/customers/:customerID/activities/
  PUT /accounting/collections/customers/:customerID/collector/
  PUT /accounting/collections/customers/:customerID/follow-up/
  POST /accounting/collections/customers/:customerID/reminder/
  GET /accounting/collections/worklist/
## customerstatementhandler
  GET /accounting/accounts-receivable/customers/:customerID/statement/pdf/
  POST /accounting/accounts-receivable/customers/:customerID/statement/send/
  GET /accounting/accounts-receivable/customers/:customerID/statements/
## bankreceiptbatchhandler
  GET /accounting/bank-receipt-batches/
  POST /accounting/bank-receipt-batches/
  GET /accounting/bank-receipt-batches/:batchID/
## bankreceipthandler
  POST /accounting/bank-receipts/
  GET /accounting/bank-receipts/:receiptID/
  POST /accounting/bank-receipts/:receiptID/match/
  GET /accounting/bank-receipts/:receiptID/suggestions/
  GET /accounting/bank-receipts/exceptions/
  GET /accounting/bank-receipts/summary/
## bankreceiptworkitemhandler
  GET /accounting/bank-receipt-work-items/
  GET /accounting/bank-receipt-work-items/:workItemID/
  POST /accounting/bank-receipt-work-items/:workItemID/assign/
  POST /accounting/bank-receipt-work-items/:workItemID/dismiss/
  POST /accounting/bank-receipt-work-items/:workItemID/resolve/
  POST /accounting/bank-receipt-work-items/:workItemID/start-review/
```

### Batch 5: Invoices and billing queue

```text
## invoicehandler
  GET /billing/invoices/
  GET /billing/invoices/:invoiceID/
  PATCH /billing/invoices/:invoiceID/
  GET /billing/invoices/:invoiceID/email-attempts/
  POST /billing/invoices/:invoiceID/generate-pdf/
  GET /billing/invoices/:invoiceID/post-preview/
  POST /billing/invoices/:invoiceID/post/
  POST /billing/invoices/:invoiceID/resend/
  GET /billing/invoices/:invoiceID/send-plan/
  POST /billing/invoices/:invoiceID/send/
  POST /billing/invoices/from-shipments/
## invoicerunhandler
  GET /billing/statements/
  GET /billing/statements/:customerID/
  POST /billing/statements/:customerID/bill/
## invoicesharehandler
  GET /billing/invoices/:invoiceID/shares/
  POST /billing/invoices/:invoiceID/shares/
  GET /billing/invoices/:invoiceID/shares/candidates/
## invoiceadjustmenthandler
  GET /billing/invoice-adjustments/:adjustmentID/
  POST /billing/invoice-adjustments/:adjustmentID/approve/
  POST /billing/invoice-adjustments/:adjustmentID/reject/
  GET /billing/invoice-adjustments/approvals/
  GET /billing/invoice-adjustments/batches/
  GET /billing/invoice-adjustments/batches/:batchID/
  GET /billing/invoice-adjustments/correction-groups/:correctionGroupID/
  POST /billing/invoice-adjustments/drafts/
  PATCH /billing/invoice-adjustments/drafts/:adjustmentID/
  POST /billing/invoice-adjustments/drafts/:adjustmentID/preview/
  POST /billing/invoice-adjustments/drafts/:adjustmentID/submit/
  POST /billing/invoice-adjustments/preview/
  GET /billing/invoice-adjustments/reconciliation-exceptions/
  POST /billing/invoice-adjustments/submit/
  GET /billing/invoice-adjustments/summary/
## billingqueuehandler
  GET /billing-queue/
  GET /billing-queue/:itemID/
  GET /billing-queue/:itemID/activity/
  PUT /billing-queue/:itemID/assign/
  PUT /billing-queue/:itemID/charges/
  PUT /billing-queue/:itemID/invoice-details/
  POST /billing-queue/:itemID/issues/:issueID/resolve/
  POST /billing-queue/:itemID/issues/:issueID/undo/
  GET /billing-queue/:itemID/neighbors/
  GET /billing-queue/:itemID/packet-preview/
  POST /billing-queue/:itemID/post/
  POST /billing-queue/:itemID/reassign-charge/
  POST /billing-queue/:itemID/release/
  PUT /billing-queue/:itemID/status/
  GET /billing-queue/assist/
  GET /billing-queue/brief/
  POST /billing-queue/bulk/
  GET /billing-queue/bulk/:runID/
  POST /billing-queue/bulk/:runID/cancel/
  GET /billing-queue/filter-presets/
  POST /billing-queue/filter-presets/
  DELETE /billing-queue/filter-presets/:presetId/
  PUT /billing-queue/filter-presets/:presetId/
  GET /billing-queue/nova-generation/
  POST /billing-queue/pod-signatures/redetect/
  GET /billing-queue/recommendations/
  POST /billing-queue/search-intent/
  GET /billing-queue/stats/
  GET /billing-queue/summaries/
```

### Batch 6: Rating

```text
## rateagreementhandler
  GET /rate-agreements/
  POST /rate-agreements/
  GET /rate-agreements/:rateAgreementID/
  PUT /rate-agreements/:rateAgreementID/
  POST /rate-agreements/:rateAgreementID/approve/
  POST /rate-agreements/:rateAgreementID/archive/
  POST /rate-agreements/:rateAgreementID/duplicate/
  GET /rate-agreements/:rateAgreementID/lanes/
  GET /rate-agreements/:rateAgreementID/lanes/summary/
  POST /rate-agreements/:rateAgreementID/reject/
  POST /rate-agreements/:rateAgreementID/resume/
  GET /rate-agreements/:rateAgreementID/rules/
  POST /rate-agreements/:rateAgreementID/submit/
  POST /rate-agreements/:rateAgreementID/suspend/
  GET /rate-agreements/:rateAgreementID/versions/
  POST /rate-agreements/rate-increase/apply/
  POST /rate-agreements/rate-increase/preview/
## rateimporthandler
  GET /rate-imports/
  POST /rate-imports/
  GET /rate-imports/:rateImportID/
  POST /rate-imports/:rateImportID/commit/
  POST /rate-imports/:rateImportID/discard/
  GET /rate-imports/:rateImportID/rows/
  GET /rate-imports/template/
## ratematrixhandler
  GET /rate-matrices/
  POST /rate-matrices/
  GET /rate-matrices/:rateMatrixID/
  PUT /rate-matrices/:rateMatrixID/
  POST /rate-matrices/:rateMatrixID/approve/
  POST /rate-matrices/:rateMatrixID/archive/
  GET /rate-matrices/:rateMatrixID/cells/
  PUT /rate-matrices/:rateMatrixID/cells/
  POST /rate-matrices/:rateMatrixID/reject/
  POST /rate-matrices/:rateMatrixID/request-changes/
  POST /rate-matrices/:rateMatrixID/restore/
  GET /rate-matrices/:rateMatrixID/reviewers/
  POST /rate-matrices/:rateMatrixID/submit/
## ratequotehandler
  GET /rate-quotes/shipment/:shipmentID/
  GET /rate-quotes/shipment/:shipmentID/applied/
  POST /rate-quotes/shipment/:shipmentID/explain/
  POST /rate-quotes/shipment/:shipmentID/shop/
## ratesimulationhandler
  GET /rate-simulations/
  POST /rate-simulations/
  GET /rate-simulations/:rateSimulationID/
  GET /rate-simulations/:rateSimulationID/results/
## routingguidehandler
  POST /routing-guides/
  DELETE /routing-guides/:guideID/
  PUT /routing-guides/:guideID/
## jurisdictionrulehandler
  POST /jurisdiction-rule-overrides/
  DELETE /jurisdiction-rule-overrides/:overrideID/
  PUT /jurisdiction-rule-overrides/:overrideID/
  POST /jurisdiction-rules/
  PUT /jurisdiction-rules/:ruleID/
  POST /jurisdiction-rules/:ruleID/verify/
## detentionhandler
  GET /detention/desk/
```

### Batch 7: Formula templates and libraries

```text
## formulatemplatehandler
  GET /formula-templates/
  POST /formula-templates/
  GET /formula-templates/:templateID/
  PUT /formula-templates/:templateID/
  POST /formula-templates/:templateID/approve
  POST /formula-templates/:templateID/backtest
  GET /formula-templates/:templateID/compare
  POST /formula-templates/:templateID/fork
  POST /formula-templates/:templateID/impact
  GET /formula-templates/:templateID/lineage
  GET /formula-templates/:templateID/readiness
  POST /formula-templates/:templateID/reject
  POST /formula-templates/:templateID/request-changes
  GET /formula-templates/:templateID/review-diff
  POST /formula-templates/:templateID/review-note
  GET /formula-templates/:templateID/reviewers
  GET /formula-templates/:templateID/reviews
  POST /formula-templates/:templateID/rollback
  POST /formula-templates/:templateID/submit
  GET /formula-templates/:templateID/usage
  GET /formula-templates/:templateID/versions
  POST /formula-templates/:templateID/versions
  GET /formula-templates/:templateID/versions/:versionNumber
  PATCH /formula-templates/:templateID/versions/:versionNumber/effective-date
  PATCH /formula-templates/:templateID/versions/:versionNumber/tags
  GET /formula-templates/:templateID/versions/scheduled
  POST /formula-templates/ai/describe-inputs
  POST /formula-templates/ai/explain
  POST /formula-templates/ai/explain-test
  POST /formula-templates/ai/suggest-tests
  POST /formula-templates/ai/thread/
  POST /formula-templates/bulk-update-status
  GET /formula-templates/context
  POST /formula-templates/context/cells
  POST /formula-templates/duplicate
  POST /formula-templates/import
  POST /formula-templates/install-standards
  POST /formula-templates/scaffold-test
  GET /formula-templates/standards
  POST /formula-templates/test
## formulalibraryhandler
  POST /formula-libraries/
  PUT /formula-libraries/:libraryID/
  POST /formula-libraries/:libraryID/approve
  POST /formula-libraries/:libraryID/archive
  GET /formula-libraries/:libraryID/compare
  GET /formula-libraries/:libraryID/readiness
  POST /formula-libraries/:libraryID/reject
  POST /formula-libraries/:libraryID/request-changes
  GET /formula-libraries/:libraryID/review-diff
  GET /formula-libraries/:libraryID/reviewers
  POST /formula-libraries/:libraryID/rollback
  POST /formula-libraries/:libraryID/submit
  GET /formula-libraries/:libraryID/usages
  POST /formula-libraries/ai/thread/
  POST /formula-libraries/duplicate
```

### Batch 8: Shipments and dispatch

```text
## shipmenthandler
  POST /shipments/
  PUT /shipments/:shipmentID/
  GET /shipments/:shipmentID/customer-tracking/preview/
  GET /shipments/:shipmentID/eta/
  GET /shipments/:shipmentID/permit-assessment/
  POST /shipments/delay/
  GET /shipments/delayed/
## shipmentmovehandler
  POST /shipment-moves/:moveID/split/
  POST /shipment-moves/:moveID/stops/:stopID/record-actual/
## assignmenthandler
  GET /assignments/
  GET /assignments/:assignmentID/
  POST /assignments/check-worker-compliance/
  DELETE /shipment-moves/:moveID/assignment/
  POST /shipment-moves/:moveID/assignment/
  PUT /shipment-moves/:moveID/assignment/
## carrierassignmenthandler
  DELETE /shipment-moves/:moveID/carrier-assignment/
  POST /shipment-moves/:moveID/carrier-assignment/
  GET /shipment-moves/:moveID/carrier-assignment/preview/
## tenderhandler
  POST /tenders/:tenderID/bounce/
  POST /tenders/:tenderID/cancel/
  POST /tenders/offers/:offerID/respond/
  POST /tenders/spot/
  POST /tenders/waterfall/
## rateconfirmationhandler
  POST /rate-confirmations/:rateConfirmationID/confirm/
  POST /rate-confirmations/:rateConfirmationID/send/
  POST /rate-confirmations/:rateConfirmationID/void/
  GET /shipment-moves/:moveID/rate-confirmations/
  POST /shipment-moves/:moveID/rate-confirmations/
## recurringshipmenthandler
  POST /recurring-shipments/
  GET /recurring-shipments/:recurringShipmentID
  GET /recurring-shipments/:recurringShipmentID/
  PUT /recurring-shipments/:recurringShipmentID/
  POST /recurring-shipments/:recurringShipmentID/generate/
  GET /recurring-shipments/:recurringShipmentID/runs/
  PUT /recurring-shipments/:recurringShipmentID/status/
  POST /recurring-shipments/match/
## permithandler
  GET /shipments/:shipmentID/permit-requirements/
  POST /shipments/:shipmentID/permit-requirements/:requirementID/waive/
  GET /shipments/:shipmentID/permits/
  POST /shipments/:shipmentID/permits/
  PUT /shipments/:shipmentID/permits/:permitID/
## servicefailurehandler
  GET /service-failures/
  GET /service-failures/:serviceFailureID/
  PUT /service-failures/:serviceFailureID/
  POST /service-failures/:serviceFailureID/edi-214-payload/
  GET /service-failures/:serviceFailureID/edi-214-readiness/
  GET /service-failures/:serviceFailureID/edi-214-status/
  POST /service-failures/:serviceFailureID/resolve/
  POST /service-failures/:serviceFailureID/review/
  POST /service-failures/:serviceFailureID/void/
  POST /service-failures/bulk-evaluate/
  POST /service-failures/evaluate-shipment/:shipmentID/
  POST /service-failures/evaluate-stop/:shipmentID/:stopID/
```

### Batch 9: Documents, reports, insights, integrations

```text
## documenthandler
  GET /documents/
  DELETE /documents/:documentID/
  GET /documents/:documentID/
  POST /documents/:documentID/approve/
  POST /documents/:documentID/attach-to-shipment/
  GET /documents/:documentID/content/
  POST /documents/:documentID/delivery-facts/
  GET /documents/:documentID/download/
  GET /documents/:documentID/import-assistant/history/
  POST /documents/:documentID/import-assistant/thread/
  GET /documents/:documentID/pages/:pageNumber/image/
  GET /documents/:documentID/preview/
  GET /documents/:documentID/reading/
  POST /documents/:documentID/reject/
  POST /documents/:documentID/restore/
  GET /documents/:documentID/shipment-draft/
  POST /documents/:documentID/shipment-draft/reextract/
  POST /documents/:documentID/signature/
  GET /documents/:documentID/versions/
  GET /documents/:documentID/view/
  POST /documents/bulk-delete/
  GET /documents/extraction-status/
  GET /documents/resource/:resourceType/:resourceID/
  GET /documents/resource/:resourceType/:resourceID/packet-summary/
  POST /documents/upload-bulk/
  POST /documents/upload/
  POST /documents/uploads/
  GET /documents/uploads/:uploadSessionID/
  POST /documents/uploads/:uploadSessionID/cancel/
  POST /documents/uploads/:uploadSessionID/complete/
  POST /documents/uploads/:uploadSessionID/parts/
  GET /documents/uploads/active/
## documentpacketrulehandler
  GET /document-packet-rules/
  POST /document-packet-rules/
  DELETE /document-packet-rules/:ruleID/
  PUT /document-packet-rules/:ruleID/
## documenttemplatehandler
  GET /document-templates/versions/:versionID/preview.pdf
## reporthandler
  POST /reports/dashboards/:dashboardID/export/
  GET /reports/runs/:runID/download/
## insighthandler
  GET /insights/
  GET /insights/:insightID/
  POST /insights/:insightID/dismiss/
  POST /insights/:insightID/restore/
  GET /insights/browse/
## integrationhandler
  GET /integrations/:type/config/
  PUT /integrations/:type/config/
  GET /integrations/:type/runtime-config/
  POST /integrations/:type/test-connection/
  GET /integrations/catalog/
  POST /integrations/samsara/workers/sync/
  GET /integrations/samsara/workers/sync/:workflowID/
  GET /integrations/samsara/workers/sync/drift/
  POST /integrations/samsara/workers/sync/drift/detect/
  POST /integrations/samsara/workers/sync/drift/repair/
  GET /integrations/samsara/workers/sync/readiness/
```

### Batch 10: EDI and email

```text
## edihandler
  POST /edi/documents/generate/
  POST /edi/documents/preview/
  GET /edi/inbound-files/
  GET /edi/inbound-files/:fileID/
  GET /edi/inbound-files/:fileID/inspect/
  POST /edi/inbound-files/:fileID/reprocess/
  POST /edi/inbound-files/bulk-reprocess/
  GET /edi/messages/
  GET /edi/messages/:messageID/
  GET /edi/messages/:messageID/inspect/
  POST /edi/messages/:messageID/replay/
  POST /edi/messages/:messageID/retry-delivery/
  POST /edi/messages/bulk-retry-delivery/
  POST /edi/x12/inspect/
## ediladinghandler
  GET /edi/flows/instances/
  GET /edi/flows/instances/:instanceID/
  POST /edi/flows/instances/:instanceID/release/
  GET /edi/lading/catalog/
  GET /edi/lading/context/
  POST /edi/lading/projects/
  GET /edi/lading/projects/:projectID/
  POST /edi/lading/projects/:projectID/drafts/
  PUT /edi/lading/projects/:projectID/settings/
  GET /edi/lading/projects/:projectID/versions/
  GET /edi/lading/projects/by-partner/:partnerID/
  GET /edi/lading/replays/:replayID/
  GET /edi/lading/versions/:versionID/
  PUT /edi/lading/versions/:versionID/
  POST /edi/lading/versions/:versionID/activate/
  POST /edi/lading/versions/:versionID/archive/
  POST /edi/lading/versions/:versionID/certify/
  GET /edi/lading/versions/:versionID/replays/
  POST /edi/lading/versions/:versionID/replays/
  POST /edi/lading/versions/:versionID/run/
  GET /edi/lading/versions/:versionID/shadow-results/
  DELETE /edi/lading/versions/:versionID/shadow/
  POST /edi/lading/versions/:versionID/shadow/
  POST /edi/lading/versions/:versionID/test/
## edisetuphandler
  POST /edi/communication-profiles/
  PUT /edi/communication-profiles/:profileID/
  POST /edi/communication-profiles/:profileID/poll/
  POST /edi/communication-profiles/:profileID/test-connection/
  POST /edi/communication-profiles/inspect-certificate/
  GET /edi/connections/
  POST /edi/connections/
  GET /edi/connections/:connectionID/
  POST /edi/connections/:connectionID/accept/
  POST /edi/connections/:connectionID/reject/
  POST /edi/connections/:connectionID/revoke/
  POST /edi/connections/:connectionID/suspend/
  PUT /edi/mapping-profiles/:profileID/items/
  DELETE /edi/mapping-profiles/:profileID/items/:mappingItemID/
  GET /edi/partners/
  POST /edi/partners/
  GET /edi/partners/:partnerID/
  PUT /edi/partners/:partnerID/
  GET /edi/partners/:partnerID/mapping-profile/
  PUT /edi/partners/:partnerID/mapping-profile/
  DELETE /edi/partners/:partnerID/mapping-profile/items/:mappingItemID/
  GET /edi/partners/:partnerID/readiness/
  POST /edi/partners/internal-pairs/
## editransferhandler
  POST /edi/load-tenders/
  GET /edi/shipment-links/
  GET /edi/shipment-links/:linkID/
  GET /edi/transfer-changes/
  GET /edi/transfer-changes/:changeID/
  POST /edi/transfer-changes/:changeID/apply/
  POST /edi/transfer-changes/:changeID/reject/
  GET /edi/transfers/
  GET /edi/transfers/:transferID/
  POST /edi/transfers/:transferID/approve/
  POST /edi/transfers/:transferID/cancel/
  POST /edi/transfers/:transferID/expire/
  GET /edi/transfers/:transferID/mapping-preview/
  POST /edi/transfers/:transferID/reject/
  POST /edi/transfers/bulk-approve/
  POST /edi/transfers/bulk-reject/
## emailhandler
  GET /email-logs/
  GET /email-logs/:messageID/
  GET /email-profiles/
  POST /email-profiles/
  DELETE /email-profiles/:profileID/
  GET /email-profiles/:profileID/
  PUT /email-profiles/:profileID/
  POST /email-profiles/:profileID/test-send/
  GET /email-profiles/assignments/
  PUT /email-profiles/assignments/
  GET /email-suppressions/
  POST /email-suppressions/
  DELETE /email-suppressions/:suppressionID/
```

### Batch 11: Users, roles, IAM, organizations, API keys

```text
## userhandler
  GET /users/
  GET /users/:userID/
  DELETE /users/:userID/mfa/
  GET /users/:userID/organization-memberships/
  PUT /users/:userID/organization-memberships/
  GET /users/:userID/profile-picture/
  POST /users/:userID/reset-password/
  GET /users/:userID/role-assignments/
  GET /users/avatars/
  POST /users/bulk-update-status/
  GET /users/me/
  PUT /users/me/assistant-visibility/
  PUT /users/me/availability/
  POST /users/me/change-password/
  GET /users/me/mfa/
  POST /users/me/mfa/recovery-codes/
  POST /users/me/mfa/totp/confirm/
  POST /users/me/mfa/totp/disable/
  POST /users/me/mfa/totp/enroll/
  GET /users/me/organizations/
  DELETE /users/me/profile-picture/
  POST /users/me/profile-picture/
  PATCH /users/me/profile/
  GET /users/me/sessions/
  DELETE /users/me/sessions/:sessionID/
  POST /users/me/sessions/revoke-others/
  PATCH /users/me/settings/
  GET /users/me/sign-ins/
  POST /users/me/switch-organization/
## rolehandler
  GET /roles/
  POST /roles/
  DELETE /roles/:roleID
  GET /roles/:roleID
  PUT /roles/:roleID
  POST /roles/:roleID/assignments
  GET /roles/:roleID/impact
  DELETE /roles/:roleID/permissions/:permID
  PUT /roles/:roleID/permissions/:permID
  DELETE /roles/assignments/:assignmentID
  GET /roles/constraints
  POST /roles/constraints
  DELETE /roles/constraints/:constraintID
  PUT /roles/constraints/:constraintID
  GET /roles/hierarchy
  POST /roles/hierarchy
  DELETE /roles/hierarchy/:edgeID
  GET /roles/preflight
## permissionhandler
  GET /me/permissions
  POST /me/permissions/check
  GET /me/permissions/version
  GET /permissions/bundles
  GET /permissions/operations
  GET /permissions/resources
## iamhandler
  GET /organizations/:id/iam/access-policies
  POST /organizations/:id/iam/access-policies
  DELETE /organizations/:id/iam/access-policies/:policyId
  PUT /organizations/:id/iam/access-policies/:policyId
  GET /organizations/:id/iam/access-settings
  PUT /organizations/:id/iam/access-settings
  GET /organizations/:id/iam/auth-events
  GET /organizations/:id/iam/external-identities
  GET /organizations/:id/iam/identity-providers
  POST /organizations/:id/iam/identity-providers
  DELETE /organizations/:id/iam/identity-providers/:providerId
  PUT /organizations/:id/iam/identity-providers/:providerId
  GET /organizations/:id/iam/mfa-authenticators
  GET /organizations/:id/iam/provisioning-audit
  GET /organizations/:id/iam/risk-decisions
  GET /organizations/:id/iam/scim/directories
  POST /organizations/:id/iam/scim/directories
  DELETE /organizations/:id/iam/scim/directories/:directoryId
  PUT /organizations/:id/iam/scim/directories/:directoryId
  GET /organizations/:id/iam/scim/directories/:directoryId/group-role-mappings
  POST /organizations/:id/iam/scim/directories/:directoryId/group-role-mappings
  DELETE /organizations/:id/iam/scim/directories/:directoryId/group-role-mappings/:mappingId
  PUT /organizations/:id/iam/scim/directories/:directoryId/group-role-mappings/:mappingId
  GET /organizations/:id/iam/scim/directories/:directoryId/tokens
  POST /organizations/:id/iam/scim/directories/:directoryId/tokens
  POST /organizations/:id/iam/scim/tokens/:tokenId/revoke
  GET /organizations/:id/iam/security-briefing
## organizationhandler
  GET /organizations/:id
  PUT /organizations/:id
  DELETE /organizations/:id/logo
  GET /organizations/:id/logo
  POST /organizations/:id/logo
  GET /organizations/:id/microsoft-sso
  PUT /organizations/:id/microsoft-sso
  GET /organizations/:id/okta-sso
  PUT /organizations/:id/okta-sso
## apikeyhandler
  GET /api-keys/
  POST /api-keys/
  GET /api-keys/:apiKeyID/
  PUT /api-keys/:apiKeyID/
  POST /api-keys/:apiKeyID/revoke/
  POST /api-keys/:apiKeyID/rotate/
  GET /api-keys/allowed-resources
```

### Batch 12: Agents, AI providers, command palette

```text
## agentdefinitionhandler
  GET /agent-definitions/
  POST /agent-definitions/
  DELETE /agent-definitions/:agentID/
  GET /agent-definitions/:agentID/
  PUT /agent-definitions/:agentID/
  GET /agent-definitions/:agentID/budget/
  GET /agent-definitions/:agentID/trust/
  POST /agent-definitions/dry-run/
  GET /agent-definitions/event-kinds/
  POST /agent-definitions/preview-prompt/
  GET /agent-definitions/system/:systemKey/
  GET /agent-definitions/templates/
  GET /agent-definitions/tools/
## agentextensionhandler
  GET /agent-extensions/:type/config/
  PUT /agent-extensions/:type/config/
  POST /agent-extensions/:type/test/
  GET /agent-extensions/catalog/
  GET /agent-portals/runs/
  GET /agent-portals/runs/:runID/
  GET /agent-portals/runs/:runID/live/
  GET /agent-portals/runs/:runID/recording/
  POST /agent-portals/runs/:runID/sign-in-code/
  GET /agent-portals/sites/
  POST /agent-portals/sites/
  DELETE /agent-portals/sites/:siteID/
  PUT /agent-portals/sites/:siteID/
  GET /agent-web-watches/
  GET /agent-web-watches/:watchID/findings/
  POST /agent-web-watches/:watchID/stop/
## agentrunhandler
  POST /agent-runs/
  GET /agent-runs/:runID/
  GET /agent-runs/:runID/transcript/
## agentanalysishandler
  GET /agent-analysis/runs/
  GET /agent-analysis/runs/:runID/
## aiproviderhandler
  POST /ai-providers/
  DELETE /ai-providers/:providerID/
  PUT /ai-providers/:providerID/
  POST /ai-providers/:providerID/test/
  GET /ai-providers/catalog/
## palettehandler
  POST /assistant/ask/
  DELETE /assistant/ask/:askID/
  POST /assistant/ask/:askID/keep/
  GET /assistant/ask/:askID/stream/
```

### Batch 13: Desk / assistant

```text
## assistanthandler
  GET /assistant/agents/:agentID/commands/
  GET /assistant/mentions/
  GET /assistant/providers/
  GET /assistant/schedules/
  DELETE /assistant/schedules/:scheduleID/
  PATCH /assistant/schedules/:scheduleID/
  POST /assistant/schedules/:scheduleID/run/
  GET /assistant/search/
  GET /assistant/threads/
  POST /assistant/threads/
  DELETE /assistant/threads/:threadID/
  GET /assistant/threads/:threadID/
  PATCH /assistant/threads/:threadID/
  GET /assistant/threads/:threadID/artifacts/
  GET /assistant/threads/:threadID/artifacts/:artifactID/
  GET /assistant/threads/:threadID/artifacts/:artifactID/export.csv
  GET /assistant/threads/:threadID/artifacts/:artifactID/export/
  POST /assistant/threads/:threadID/artifacts/:artifactID/pin/
  POST /assistant/threads/:threadID/artifacts/:artifactID/restore/
  POST /assistant/threads/:threadID/artifacts/:artifactID/rewrite/
  GET /assistant/threads/:threadID/artifacts/:artifactID/rows/
  POST /assistant/threads/:threadID/artifacts/:artifactID/run/
  POST /assistant/threads/:threadID/artifacts/:artifactID/versions/
  GET /assistant/threads/:threadID/artifacts/by-slug/:slug/
  GET /assistant/threads/:threadID/budget/
  DELETE /assistant/threads/:threadID/case/
  GET /assistant/threads/:threadID/case/
  PUT /assistant/threads/:threadID/case/
  POST /assistant/threads/:threadID/case/await-reply/
  POST /assistant/threads/:threadID/case/snooze/
  POST /assistant/threads/:threadID/case/ticks/
  POST /assistant/threads/:threadID/case/wake/
  GET /assistant/threads/:threadID/chapters/
  POST /assistant/threads/:threadID/chapters/
  DELETE /assistant/threads/:threadID/chapters/:chapterID/
  PATCH /assistant/threads/:threadID/chapters/:chapterID/
  POST /assistant/threads/:threadID/compact/
  POST /assistant/threads/:threadID/handoff/
  GET /assistant/threads/:threadID/messages/
  POST /assistant/threads/:threadID/messages/
  GET /assistant/threads/:threadID/plans/
  GET /assistant/threads/:threadID/proposals/
  PUT /assistant/threads/:threadID/proposals/:proposalID/edits/
  GET /assistant/threads/:threadID/queue/
  POST /assistant/threads/:threadID/queue/
  DELETE /assistant/threads/:threadID/queue/:itemID/
  PATCH /assistant/threads/:threadID/queue/:itemID/
  POST /assistant/threads/:threadID/queue/:itemID/send/
  PUT /assistant/threads/:threadID/queue/order/
  POST /assistant/threads/:threadID/read/
  POST /assistant/threads/:threadID/requests/
  GET /assistant/threads/:threadID/schedules/
  POST /assistant/threads/:threadID/schedules/
  GET /assistant/threads/:threadID/transcript/
  POST /assistant/threads/:threadID/turns/
  GET /assistant/threads/:threadID/turns/active/
  GET /assistant/threads/:threadID/waits/
  POST /assistant/threads/:threadID/waits/:waitID/cancel/
  GET /assistant/threads/shelves/
  POST /assistant/turns/:turnID/stop/
  GET /assistant/turns/:turnID/stream/
  GET /assistant/turns/active/
```

### Batch 14: Driver portal

```text
## driverportalhandler
  POST /portal/credentials/:credentialID/document/
  GET /portal/document-types/
  POST /portal/expenses/:expenseID/receipt/
  POST /portal/invitations/accept
  GET /portal/invitations/preview
  GET /portal/loads/:shipmentID/documents/
  POST /portal/loads/:shipmentID/documents/
  GET /portal/profile/document-types/
  GET /portal/profile/documents/
  POST /portal/profile/documents/
```
