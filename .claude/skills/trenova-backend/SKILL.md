---
name: trenova-backend
description: How to design and write backend Go code in Trenova (services/tms, shared/). Use before adding or changing an entity, migration, repository, service, validator, GraphQL schema or resolver, REST handler, Temporal workflow or activity, cache, config section, or background sweep, and when reviewing or planning such a change. Covers the engineering stance (scale, no backwards compatibility, complete features, propose better designs), where code goes, and the pattern file for each layer.
---

# Writing backend code in Trenova

The patterns live in `docs/engineering/backend/`; this skill is the order to read them in and
the stance to hold while you do.

## The stance

- **You own the design.** Before writing code, decide whether the requested approach is the best
  one. If you can name a better design (a different boundary, a stored projection instead of a
  per-read computation, a debounced workflow instead of a per-event one, a breaking rename that
  removes a wart), build it and explain the trade-off. Ask only when the decision is genuinely
  the maintainer's, and lead with your recommendation.
- **Scale is the default requirement**, not an optimisation pass: hundreds of thousands of users,
  thousands of tenants, millions of rows per tenant. Run `scale.md` against your design before
  writing it.
- **Break things cleanly.** Nothing has launched. Rename and reshape across the schema, GraphQL,
  routes, payloads and config in one change, delete the old path, keep no shims.
- **Complete.** Validation, authorization, audit, realtime, error paths, tests, the agent tool or
  write-coverage entry, and the client, in the same change.
- **DRY by search.** Before writing a helper, port, query, loader or job, search `shared/`,
  `pkg/` and the module for one that already does it.

## Read first

1. `docs/engineering/backend/README.md`: where code goes and the end-to-end order.
2. The pattern file for each layer you touch: `data-access.md`, `services.md`, `api.md`,
   `jobs.md`, `testing.md`, `go.md`, and `scale.md` always.
3. The subsystem's doc from `docs/engineering/README.md`. Its "rule that bites" column lists the
   invariants that compile and still break production.

## When the patterns are wrong

If a pattern stops you doing the better thing, or you invent a mechanism the next change should
reuse, update the pattern file in the same change and say so in your report. Keep pattern files
short: a rule with one canonical example path, not a tutorial.

## Before you hand off

Review the diff against `scale.md` and the layer files: queries per row, missing indexes,
unbounded lists, network calls in transactions, missing authz or tenant scoping, swallowed
errors, duplicated helpers, generic types leaking through exported APIs. Fix what you find. Then
report what changed and why, any design you chose over the one asked for, any pattern you
changed, and the exact checks the change needs, for example:

- `go test -tags=nomsgpack ./<changed packages>/...` (add `integration` for repositories)
- `task lint` (golangci-lint, archlint)
- `task gqlgen`, `task gqlschema-diff`, `pnpm --filter @trenova/graphql codegen` (schema changed)
- `task db-migrate`, `task db-stamp` and `task db-lint` (migration or seed changed)
- the generators in `generated-artifacts.md` that the change touches
