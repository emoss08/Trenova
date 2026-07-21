# Mission: Full overhaul of the Trenova accounting/AR module — design + analytics + customer payments

You are overhauling the accounting section of the Trenova TMS (Go monorepo + React/TS client).
The current AR pages look utilitarian and lack the analytics/functionality that transportation
accounting systems (McLeod LoadMaster/PowerBroker, TruckMate, NetSuite AR) provide. Do a complete
redesign AND a full-stack upgrade: new GraphQL API for AR/GL/payments + new analytics aggregations,
and migrate every page off REST onto GraphQL. This is an enterprise app — production-grade, fully
featured, no stubs or "v1" shortcuts. Read surrounding code before writing new code and match it.

## Pages in scope
1. Accounting Dashboard — client/src/routes/accounting-dashboard/page.tsx (271 lines, all inline)
2. AR Aging — client/src/routes/ar-aging/page.tsx (raw `<table>`, 6 KPI cards, no charts)
3. Customer Ledger — client/src/routes/customer-ledger/page.tsx (raw ID textbox, hand-rolled table)
4. AR Open Items — client/src/routes/ar-open-items/page.tsx (best of the current set; filter bar + summary cards + raw table)
5. NEW: Customer Payments — full cash-application page (no UI exists today; only types/customer-payment.ts + services/customer-payment.ts with getById)

## Current state = ground truth (do not re-derive)
- All AR pages are REST via @tanstack/react-query + a query-key factory: client/src/lib/queries/ar.ts
  and client/src/services/ar.ts (api.get to /accounting/accounts-receivable/*). Migrate these reads to GraphQL.
- Every AR page rebuilds its own `<table>` markup and its own KPI/summary card. There is NO chart anywhere
  in accounting today. The reusable DataTable stack (client/src/components/data-table/*) is NOT used by
  any AR page — adopt it (or ui/table) instead of raw `<table>`.
- Money is stored in MINOR UNITS (integer cents) — divide by 100. Use the existing
  client/src/components/accounting/amount-display.tsx (`<AmountDisplay variant="auto" />`) for all currency.
- Dates are UNIX SECONDS — new Date(x * 1000).
- Reusable accounting components already exist and MUST be reused, not reinvented:
  client/src/components/accounting/{amount-display,source-drill-down-link,accounting-status-badge,
  financial-report-section,fiscal-period-selector}.tsx. SettlementStatus badge lives in
  client/src/components/status-badge.tsx (PlainSettlementStatusBadge). Page shell = PageLayout from
  client/src/components/navigation/sidebar-layout.tsx (takes pageHeaderProps {title, description, actions}).

## Design system + quality bar
- shadcn/ui, style "base-nova", baseColor zinc, lucide icons (components.json). Primitives in client/src/components/ui/*.
- Charts: recharts 3.9.2 wrapped by the shadcn chart primitive client/src/components/ui/chart.tsx
  (ChartContainer, ChartTooltip, ChartTooltipContent, ChartConfig, ChartLegend). @nivo/bar is also available.
- Match or exceed these existing polished pages (study them first):
  * Quality bar for charts/dashboard: client/src/routes/fuel-management/_components/fuel-dashboard.tsx
    (ChartContainer + recharts LineChart, themed series, range toggles 13w/26w/52w, KPI cards with
    TrendingUp/Down deltas, GraphQL-backed).
  * Quality bar for layout/tabs/filter-bar: client/src/routes/reports/page.tsx + _components/*
    (tabbed underline header, nuqs URL-state filters, sort/category/status Selects, debounced search,
    PageLayout with header actions, motion/react animation, per-category colored icon tiles).
- Aesthetic target: Linear/Vercel-grade polish with REAL motion (motion/react) — flat/static UI is
  unacceptable. Dense but legible financial layouts. 6–8 visuals per dashboard screen max.

## Hard constraints (project + owner preferences — non-negotiable)
- Forms: use the useWatch hook, NEVER watch() from useForm.
- Entity references (customer, invoice, GL account, worker, etc.): use the autocomplete fields in
  client/src/components/autocomplete-fields.tsx (e.g. CustomerAutocompleteField). NEVER a raw ID text
  box or a static `<select>`. (Fix customer-ledger's raw ID textbox specifically.)
- Every `<Select>` must receive items={...} as {value,label}[] or it renders raw values.
- Do NOT add colored left-border accents to cards/list items.
- New client features use GraphQL for EVERYTHING including mutations — never REST. Pass a mutationFn
  into form panels. (This is why payments writes get new GraphQL mutations below.)
- Go: hexagonal architecture (domain in core/, adapters in infrastructure/), Bun ORM, sonic for JSON
  (encoding/json is lint-forbidden), no comments in code, group 4+ params into a named struct, utilities
  go in shared/ not domain files. Follow Uber Go style.
- Do NOT run mockery/generate-mocks (crashes the machine) — hand-edit mocks if needed.

## Backend work (full-stack — none of this GraphQL exists today)
The domains already exist (services/tms/internal/core/domain/{invoice,customerpayment,customerledger,
glaccount,journalentry,accounttype}); the SERVICES exist too:
- customerpaymentservice: PostAndApply, ApplyUnapplied, Reverse (services/.../customerpaymentservice/service.go)
- accountsreceivableservice (read-only): ListCustomerLedger, ListOpenItems, GetCustomerStatement,
  GetCustomerAging, GetAgingSummary.
What's missing is the GraphQL surface and the new analytics aggregations. Follow the fuel-surcharge
GraphQL pattern end-to-end (schema → gqlgen generate → resolver → wire in resolver.go):
- Reference schema: services/tms/internal/api/graphql/schema/fuel_surcharge.graphqls
- gqlgen config: services/tms/gqlgen.yml ; regenerate with the project's gqlgen task, NOT mockery.
- Add *.graphqls for: AR aging + aging snapshots/history, AR open items, customer ledger, customer
  payments (list/detail + mutations), and the accounting dashboard analytics.
- Resolvers should call the EXISTING services above where possible. Add NEW service methods (in the
  appropriate accounting/AR service, business logic in core/) for analytics that don't exist yet:
  * DSO time series (rolling, e.g. 13/26/52 weeks) and current DSO + trend delta
  * Collection Effectiveness Index (CEI)
  * Average Days to Pay (ADP)
  * Aging distribution snapshot + historical aging trend (stacked over time)
  * Rolling ~90-day cash-flow forecast (expected vs actual collections, based on due dates + payment history)
  * Bad-debt / write-off ratio, dispute/short-pay rate
  * Top-N overdue customers by open $, and per-customer risk (credit utilization, 12-mo payment trend,
    delinquency score)
- Payment mutations (wrap existing service methods): postAndApplyCustomerPayment, applyUnappliedCustomerPayment,
  reverseCustomerPayment; plus queries customerPayments(connection) and customerPayment(id). Keep the
  existing REST handlers intact for now, but the CLIENT must use the new GraphQL.
- Everything must respect existing money-in-minor-units, fiscal-period-closed guards, idempotency, and
  AccountingControl posting mode (Automatic vs Manual) already implemented in the services.

## Client GraphQL workflow (follow exactly)
1. Write operations in client/src/graphql/operations/<feature>/operations.graphql (fragments + queries +
   mutations). Use the DataTableConnectionInput! + edges/node/pageInfo/totalCount convention for lists.
2. Run: pnpm graphql:codegen  (generates client/src/graphql/generated/*, persisted-documents.json — auto-synced).
3. Thin service in client/src/lib/graphql/<feature>.ts calling requestGraphQL({document, operationName, variables}).
   Canonical example: client/src/lib/graphql/fuel-surcharge.ts.
4. Query-key factory module under client/src/lib/queries/, consumed via useQuery(queries.<feature>.<op>()).
5. Mutations via useMutation with the GraphQL mutationFn, passed into the form panels.

## Per-page requirements (transportation-grade analytics)

### 1. Accounting Dashboard (executive AR command center)
Replace the 4 flat KPI cards + link grids with a real dashboard (6–8 visuals):
- Top KPI row with trend deltas: Total AR Outstanding, Current DSO (+ trend arrow, target <45 days),
  CEI gauge, Unapplied Cash, Overdue % of AR. Reuse the fuel-dashboard KPI-card-with-delta pattern.
- AR Aging distribution: donut/stacked bar across 0–30 / 31–60 / 61–90 / 90+ (green→yellow→red).
- DSO trend line chart with 13w/26w/52w range toggle (like fuel dashboard).
- Rolling 90-day cash-flow forecast: expected vs actual collections (line/area).
- Aging trend over time (stacked area by bucket).
- Top 10 overdue customers by open $ (ranked list w/ drill-down to customer ledger).
- Collections worklist / attention panel: promise-to-pay, disputes/short-pays, threshold alerts at
  15 & 30 days overdue.
- Keep quick navigation but make it secondary, not the centerpiece.

### 2. AR Aging
- Replace raw `<table>` with the DataTable stack: sortable, filterable, sticky totals footer, column config.
- Add a summary header: aging distribution chart + KPI tiles (total, % current, % overdue, DSO, CEI).
- Color-coded aging buckets, per-customer rows drilling into customer ledger.
- Filters: customer (autocomplete), as-of date, business unit; export.
- Optional transportation cut: aging by customer segment / by branch if data supports it.

### 3. Customer Ledger
- Replace the raw ID textbox with CustomerAutocompleteField.
- Ledger as a clean statement: running balance, debit/credit, document + SourceDrillDownLink, event type,
  aging of each open item.
- Header: customer AR snapshot (open balance, credit limit + utilization gauge, DSO, avg days to pay,
  12-mo payment-trend sparkline, oldest open invoice).
- Actions: record payment (opens the payments flow prefilled for this customer), view statement, export.

### 4. AR Open Items
- Keep its good filter bar; upgrade table to the DataTable stack (sort/filter/column config/sticky totals).
- Add per-invoice aging badge, short-pay/dispute indicators, and a "select rows → Apply Payment" bulk action
  that hands the selected open invoices to the payments cash-application flow.
- Summary cards: total open, current, overdue, avg age, count. Small aging chart.

### 5. Customer Payments (NEW — full cash-application workflow)
- List page: DataTable of payments (filter by customer/date/method/status), summary tiles
  (posted today, unapplied cash, reversed), status via existing badges.
- Record Payment panel (GraphQL mutation postAndApplyCustomerPayment): CustomerAutocompleteField →
  payment method Select (items: ACH/Check/Wire/Card/Cash/Other) → amount, payment date, accounting date,
  reference → auto-suggested list of that customer's OPEN invoices with checkboxes and editable applied
  amounts. Support split across multiple invoices, short-pay/deduction codes, and unapplied-cash remainder.
  Live-validate: total applied + deductions + unapplied == payment amount; block over-application and
  closed fiscal periods (surface server field errors via the errortypes multi-error → form fields).
- Apply Unapplied panel (applyUnappliedCustomerPayment): take an existing payment's unapplied cash and
  apply it to open invoices.
- Payment detail drawer: payment header, applications table (invoice, applied, short-pay), linked journal
  batch (SourceDrillDownLink to the GL posting), and a Reverse action (reverseCustomerPayment) with
  confirmation. Show the resulting settlement-status change on affected invoices.
- Use useWatch for all form state; useMutation with GraphQL mutationFn passed into the panel.

## Verification (must pass before you consider it done)
- Client typecheck: cd client && npx tsc -b --force  (NOT `pnpm tsc --noEmit` — it's false-green).
- Client lint: cd client && pnpm lint
- GraphQL: pnpm graphql:codegen runs clean and persisted-documents.json is synced.
- Go: from services/tms — task lint and task test. Regenerate gqlgen (never mockery).
- Local verification only via build/unit tests/FX boot/psql — do NOT curl a live server (the sandbox
  kills live servers).
- Manually confirm: recording a payment updates invoice SettlementStatus (Unpaid→PartiallyPaid→Paid),
  posts a balanced GL journal, and reversing undoes both.

Work through the pages in this order (dashboard analytics + backend API first since everything depends on
it, then aging → open items → ledger → payments). Keep each PR-sized change coherent and fully featured.
