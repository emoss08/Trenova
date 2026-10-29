# Go style guide

How Go is written in `services/tms` and `shared/`, for people and agents alike. The baseline is
the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md); this page is
what it leaves open, decided for Trenova. Some of it is adapted from Google's
[Go](https://google.github.io/styleguide/go/) and [Rust](https://google.github.io/styleguide/rustguide.html)
guides where their reasoning holds for a codebase of this size. `task lint` (golangci-lint,
archlint, pkgsizelint and the repository's own linters) enforces most of it.

## Principles

When two styles both work, pick by these, in order. A later one never overrides an earlier one.

1. **Clarity.** A reader can tell what the code does and why without running it. Code is read
   far more often than it is written.
2. **Build cost.** Every package is a compile unit. Code is arranged so an edit recompiles as
   little as possible: small packages, narrow exported surfaces, nothing generic in exported
   signatures. With roughly 2,000 packages, a careless edit to a widely imported package costs
   every developer a minute on each reload.
3. **Simplicity.** The least mechanism that does the job: the language, then the standard
   library, then `shared/` and `pkg/`, then a maintained dependency, then something new.
4. **Concision.** No repetition, no ceremony, no abstraction without a caller.
5. **Consistency.** It reads like the code around it. Consistency breaks ties; it never justifies
   spreading a pattern this page rejects.

## Packages

### Small packages

**Prefer small packages: one abstraction each** (one entity, one service, one family of ports,
one family of tools), ideally a handful of files. If an abstraction can live in its own package,
put it there. Decide first whether two APIs need to share a package at all; only then decide
how to arrange the files inside one.

Why it matters here:

- **A package compiles as a whole.** Any edit to it re-type-checks every file in it. A
  36,000-line package recompiles 36,000 lines on a one-line change.
- **An exported change recompiles every importer.** Go skips rebuilding an importer when a
  package's export data (its exported names, types, constants and inlinable bodies) is
  unchanged, so an edit to unexported code stays local. Anything exported that changes
  recompiles every direct importer, and theirs in turn wherever the change shows through their
  own exported API. A package with 500 importers that also holds the code people edit most is
  the most expensive thing in the build.
- **What a package exports, every importer pays to load.** A widely imported package should
  export only what most importers use.

**Budgets** (lines of hand-written, non-test Go per package):

| Package | Budget | Why |
|---|---|---|
| `domain/*`, `ports/*`, `*contract`, `pkg/*`, `shared/*` | 1,500 | other modules import these; an edit ripples |
| everything else | 3,000 | `app`, `adapters`, `jobs`, `api`, `infrastructure`, `cmd`: few importers, but each edit recompiles the whole package |

**How to split.** Split along an abstraction, not along a line count:

1. Find the core: the main type and everything it needs transitively. In a domain package that
   is the aggregate and the value types its fields use (`worker.Worker`, its profile, and the
   status enums the profile carries).
2. Everything outside the core moves to packages that import the core. Group by what changes
   together and who uses it (`workerdot`, `workersafety`, `pto`).
3. Pull what changes often and what few importers need out of widely imported packages. The
   permission vocabulary (`permission.Resource`, `permission.OpRead`), which 500 packages
   import, is separate from the resource catalog (`permissioncatalog.Registry`), which the
   authorization service and its tests import.
4. A method lives with its type, so a type whose methods are separate use cases (prompt
   building, rendering, rating) keeps its data and invariants. The use cases become functions in
   their own package that take the type.

**Exceptions.** A package may exceed its budget only when Go forces the code together, and the
reason is recorded in `services/tms/.go-pkgsize.yml`:

- **Mutual references.** Types that refer to each other must share a package, because Go forbids
  import cycles. An aggregate whose models hold relations in both directions
  (`shipment.Shipment`, its moves, stops and assignments) is one package. Keep only the
  aggregate there; everything that merely uses it moves out.
- **A large foundational library** whose surface is one abstraction.

**Don't over-split.** Go has no re-exports: don't alias types from another package to keep an
old import path working; change the callers. Don't split so that one half must export its
internals for the other. Don't cut along file boundaries into packages that import each other's
everything. Every package also compiles its own copy of the generic instantiations it can see
([Generics and build time](#generics-and-build-time)), so a split must narrow what each package
sees. To hide a package tree's implementation, use an `internal/` subdirectory.

**Enforcement.** `go run ./scripts/pkgsizelint` (part of `task lint` and CI) measures every
package in `services/tms` and `shared` and fails on any package over its budget that has no entry
under `exceptions` in `.go-pkgsize.yml`. An exception names one of the reasons above and caps the
package at a ceiling; a package that grows past its ceiling fails the lint.
`go run ./scripts/pkgsizelint -update` lowers each ceiling to the package's current size and
removes exceptions that are back within budget. Never raise a ceiling to fit new code: move out
whatever does not have to share the package.

### Names

- **Name a package for what it provides**, so the call site reads well:
  `permissioncatalog.NewRegistry()`, `llmports.Message`. Never `util`, `common`, `helpers`,
  `base`, `misc` or `types`. When you split a package with such a name, rename the pieces.
- **Domain package names are unique across modules**, because keel names its generated table
  package after the domain package (`domain/timesheet` generates `timesheetdb`).
- **Ports packages end in `ports`, contract packages in `contract`, test helpers in `test`**, as
  in [module-architecture.md](../module-architecture.md).
- **Don't repeat the package in its names:** `retrievalports.QueryVector`, not
  `retrievalports.RetrievalQueryVector`.

### Files

There is no one-type-per-file rule. A file is focused enough that a maintainer can guess where
something lives, and a file of thousands of lines is a sign the package needs splitting, not
that the file does. Name a file after its primary type or concern (`timesheet.go`,
`operationdefs.go`).

## Code

- **No comments.** Names carry the meaning. A doc in `docs/engineering/` carries the why. The one
  exception is data: the doc comment of a reportable keel model (its table tag carries
  `report=`) and of its fields is the report catalog's description of the entity and the field,
  which keel gen reads and the builder and the assistant show
  ([reporting.md](../reporting.md#descriptions-are-doc-comments)).
- **JSON is `sonic`.** `encoding/json` is forbidden by lint.
- **Errors** wrap with `fmt.Errorf("…: %w", err)` or are an `errortypes` error the caller is meant
  to see ([services.md](services.md#errors)). Never discard one; never log and return the same
  error.
- **A params struct past three or four parameters** (`type CreateShipmentParams struct { … }`).
  An fx constructor always takes an `fx.In` params struct.
- **Value receivers** unless the method mutates or the struct is large.
- **Concrete or generic types over `any`.** `any` is for genuinely heterogeneous data at a
  boundary (a JSON column, a template context).
- **Preallocate** slices and maps whose size is known: `make([]T, 0, len(in))`.
- **Utilities live in `shared/`** (`shared/sliceutils`, `shared/stringutils`, `shared/maputils`,
  `shared/timeutils`, `shared/money`, `shared/pulid`, `shared/parallel`, …). Search there before
  writing one; add a package there when no category fits. A helper in a domain, service or
  handler file is a review failure, and so is a private copy of a constant `shared/` already
  has (`timeutils.SecondsPerDay`).
- **No dependency checks that DI already guarantees.** A nil check on an fx-provided dependency,
  or on a field that validation or route binding already requires, is noise.
- **Tests use `t.Context()`**, never `context.Background()`.
- **Format** with `goimports` (no `-local`) and `golines -m 100`, on the files you changed.
  Imports are two groups: the standard library, then everything else.

## Generics and build time

A generic type's methods are compiled again in every package that can see the type. So no
exported type, and no signature of an exported function or method, mentions a concrete generic
type that has methods (`*tenantedvalidation.Validator[*X]`, `*dataloadgen.Loader[K,V]`):
hold it behind a generic interface (`tenantedvalidation.EntityValidator[*X]`) or in an
unexported type reached only through package-level functions. An unexported field of an exported
type still spreads it. `sync.Map` is a generic hash trie underneath; a widely reachable type uses
a plain map behind a mutex. The measurements and the fixes already made are in "Generics and
build time" in [module-architecture.md](../module-architecture.md).

## Dependencies

A new third-party module needs a reason the PR states: what it does that `shared/` or the
standard library cannot, its maintenance, and its licence. Wrap a vendor SDK in an adapter behind
a port so the rest of the code never imports it.
