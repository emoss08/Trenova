# GraphQL Executor

The TMS GraphQL server keeps gqlgen for everything except the executor. The schema, the
`gqlgen.yml` bindings, `@goField`, the generated models, the resolver stubs, the resolver
interfaces, the HTTP handler, and every handler extension are gqlgen's. The code that walks a
selection set and calls resolvers is Trenova's own: a small runtime,
`services/tms/internal/api/graphql/gqlexec`, plus one generated package per schema file.
Resolvers are split the same way, one package per schema file, and each package depends
only on the services its own code uses. Both are produced by `task gqlgen`, which runs
`internal/api/graphql/gqlexec/gen`.

## Why

gqlgen writes the whole executor into one Go package. At Trenova's size — about 1,900 types
and 10,800 fields — that package was 545,000 lines. Go compiles a package on one core. It
imported 112 domain packages, so almost any domain edit recompiled all of it. It also peaked
at about 7 GB of RAM, which is why `LOW_RESOURCE` mode existed.

Almost all of that code repeats one generic algorithm for each field. Each plain struct
field got about 50 lines: a resolve function, a field-context function, and a case in its
object's switch. `gqlexec` implements the algorithm once and describes each field with one
table entry. Only the code that has to be typed is still generated: argument and input
decoding, scalar and enum marshalling, and resolver calls.

| `go build ./cmd/cli`, 4 cores, empty build cache | gqlgen | gqlexec, one resolver package | gqlexec, resolver packages |
| --- | --- | --- | --- |
| Cold build | 354 s | 276 s | 274 s |
| Rebuild after editing a domain struct | 253 s | 166 s | 152 s |
| Rebuild after editing a service | 75 s | 74 s | 36 s |
| Rebuild after editing a resolver | 76 s | 75 s | 29 s |
| Largest single compile (peak RAM) | executor, 7.1 GB | resolver, 5.4 GB | test mocks, 2.3 GB |
| Generated executor | 545,000 lines, 1 package | 235,000 lines, 120 packages | same |

The runtime is also cheaper. On one pass over the parity workload below, it used 11% less
time, 20% less memory and 17% fewer allocations than gqlgen's executor.

With the executor fixed, the hand-written `resolver` package became the largest compile:
about 40 s and 5.4 GB on its own. A first split by schema file made things worse. Every
package reached every service through one shared struct, so a service edit recompiled all
116 of them (162 s). The kept design gives each package a generated `Deps` struct holding
only the services that package names; see [Resolvers](#resolvers).

## Layout

```text
internal/api/graphql/
├── gqlexec/                 # runtime (hand-written, no domain imports)
│   ├── schema.go            # graphql.ExecutableSchema: operation dispatch, deferral
│   ├── exec.go              # objects, fields, root middleware, null propagation
│   ├── marshal.go           # list, input and adapter helpers used by generated code
│   ├── registry.go          # merges shards, validates them against the schema
│   ├── gen/                 # the generator (`task gqlgen`)
│   └── internal/e2e/        # a two-file schema exercising every code path
├── generated/
│   ├── root.generated.go    # ResolverRoot, *Resolver interfaces, Config, NewExecutableSchema
│   └── exec/<file>exec/     # one executor package per schema file
├── gqlmodel/models_gen.go   # gqlgen model plugin, unchanged
└── resolver/
    ├── services.go          # Params (fx), Services: every dependency a resolver may use
    ├── resolver.generated.go  # Resolver root: builds each package's Deps, wires ResolverRoot
    ├── base/                # Core (logger, permissions, tracing) and shared helpers
    ├── <file>resolver/      # one package per schema file
    │   ├── <file>.resolvers.go       # hand-written resolver methods (stubs generated)
    │   └── resolvers.generated.go    # Deps and the resolver types
    ├── mappers/             # mappergen output
    └── telematicsmapping/   # mapping shared by more than one resolver package
```

A schema file's executor package holds:

- The objects, interfaces, unions and inputs the file defines.
- The fields the file declares, including `extend type Query` and `extend type Mutation`.
  The 1,066 root operations are therefore spread across the files that declare them.
- An argument decoder for each field that takes arguments.
- The marshal and unmarshal functions that those fields and inputs use. They are copied into
  every shard that needs them, so shards never import each other.
- A small resolver interface per object, listing only the resolver methods that shard calls.

Types that cross files are looked up by name at runtime. An object is marshalled through
`ec.MarshalType("Trailer", v)`, and an input defined in another file is decoded through
`gqlexec.UnmarshalInput`. That lookup is what removes the compile-time dependency between
shards.

## How generation works

`task gqlgen`:

1. Loads `gqlgen.yml`, runs gqlgen's model plugin, and reloads packages. Reloading is what
   fixed the old "merging type systems failed" retry.
2. Binds the schema to Go types with gqlgen's own `codegen.BuildData`. Bindings, autobind,
   `@goField`, initialisms and enum mappings therefore behave exactly as they did under
   gqlgen.
3. Writes the executor shards and `generated/root.generated.go`.
4. Writes resolver stubs into `resolver/<file>resolver/<file>.resolvers.go`, keeping
   existing method bodies, and regenerates each package's `Deps` from the `Services`
   fields that package's code names. It then writes the resolver root.
5. Builds the result.

If any step before the build fails, the previous executor, models and resolver files are put
back exactly as they were.

## Resolvers

`resolver/services.go` is hand-written. `Params` is the fx parameter struct, `Services`
embeds `*base.Core` and holds every service a resolver may use, and `newServices` fills it.
To give resolvers a new service, add it to all three.

Each resolver package gets a generated `Deps`:

```go
type Deps struct {
	*base.Core
	ShipmentService services.ShipmentService
	// ...only the Services fields this package's code names
}

type QueryResolver struct{ *Deps }
```

The generator finds a package's dependencies by name. Any `x.Name` selector or `Name:` key in
the package that matches a `Services` field adds it, so writing `r.ShipmentService` in a
resolver method and running `task gqlgen` is all it takes. The list can include a field the
package does not really use; that only widens the package's imports.

Rules that keep the split paying off:

- Code shared by several resolver packages goes in `base` only when it needs nothing but
  `Core`. Everything in `base` recompiles all resolver packages when it changes. Mapping
  shared by a few packages gets its own small package, like `telematicsmapping`.
- Resolver packages never import each other.
- A resolver method may not have the same name as a `Services` field the package uses: the
  method would hide the field. The generator refuses this and names both; rename the field.

In tests, build the package's `Deps` directly:

```go
r := &QueryResolver{&Deps{
	Core:            &base.Core{L: zap.NewNop(), PermissionEngine: engine},
	ShipmentService: shipments,
}}
```

Tests in the root package use `FromServices(&Services{...})`.

The 50 or so service packages reachable from shared packages (`gqlmodel`, `loaders`,
middleware, `ports/services`) still recompile every resolver package when edited. Domain
edits also still reach most of them; that cascade comes from the domain package graph, not
from GraphQL.

## How a request runs

`generated.NewExecutableSchema` builds a `gqlexec.Schema` from the registry (compiled once
per process). gqlgen's handler parses, validates and runs extensions exactly as before, then
calls `Exec`. The runtime:

1. Collects fields with `graphql.CollectFields`, as gqlgen's generated code did.
2. For each field, builds the same `graphql.FieldContext` gqlgen built, with the same
   `IsMethod`, `IsResolver`, `Args` and `Child`. It then resolves the field through
   `graphql.ResolveField`, the helper gqlgen's own generated code calls. Field middleware,
   error paths, panic recovery and the "must not be null" rule therefore come from gqlgen
   itself.
3. Runs resolver fields and context-taking methods concurrently. Mutation fields run in
   order. Lists of objects marshal concurrently up to `exec.worker_limit`.
4. Propagates nulls exactly as gqlgen does. A non-null field that fails nulls its parent.

Extensions that read `graphql.GetFieldContext`, `CollectFieldsCtx`, root field contexts or
`fc.Result` (projection, cost budget, observability, feature access) see the same values as
before.

At startup the registry checks that every object, field, interface and union in the schema
has an executor, and that the resolver root implements every resolver interface a shard
declares. If something is missing, `NewExecutableSchema` panics with the list instead of
failing on the first request that reaches it.

## Proof of equivalence

The executor was checked against gqlgen's. Both were driven with identical, deterministic
fixture resolvers: generated fakes that return filled values, nil pointers and errors, keyed
by field and arguments. The inputs were:

- all 958 persisted client operations, with generated variables;
- one synthetic query or mutation per root field that selects every field, three levels
  deep, including fragments on each concrete type of every interface and union;
- the full introspection query.

Each ran under three fixture seeds: 6,072 requests covering 9,564 of the 10,844 object
fields. The `data` payloads were byte-for-byte identical and the errors identical after
sorting (concurrent fields may report errors in either order). To check that the harness
could catch real differences, two bugs were planted in the runtime, one in null propagation
and one in non-null list elements. They produced 2,289 and 875 mismatches.

The harness needs gqlgen's executor, so it is not kept in the repository. The durable
regression tests are `gqlexec/internal/e2e` (end-to-end behaviour on a two-file schema) and
`gqlexec/registry_test.go`.

## What is not supported

The generator refuses these gqlgen features and names the field, instead of generating code
that behaves differently:

- subscriptions
- runtime directives: any directive on a field, argument or input that is not
  `skip_runtime`, and any `QUERY`, `MUTATION` or `FIELD` location directive
- batch resolvers (`batch: true`)
- complexity functions (`omit_complexity` must stay `true`; the cost budget uses
  `querycost`, not gqlgen's complexity root)
- `omit_panic_handler`
- federation

`@defer` goes through the same deferral code path gqlgen's generated executor used. No
client operation uses it, and neither the parity run nor the end-to-end tests cover it.

`graphql.UnmarshalInputFromContext` no longer works. gqlgen's executor rebuilt the map
behind it, with reflection over every input type, on every request. Nothing in Trenova calls
it.

To support one of these, add it to the runtime and the shard template, then extend
`gqlexec/internal/e2e` to cover it. Do not work around the generator's refusal.

## Working on it

- `task gqlgen` regenerates everything and then builds it. Never run
  `go tool gqlgen generate`.
- The executor templates in `gqlexec/gen/*.gotpl` are ports of gqlgen's `type.gotpl`,
  `input.gotpl`, `args.gotpl`, `field.gotpl` and `interface.gotpl`. When upgrading gqlgen,
  diff those upstream templates between the two versions and carry any behaviour change
  across. Then run `go test ./internal/api/graphql/...`.
- Regenerate the end-to-end fixture after changing the templates. `task gqlgen-check` and
  CI do this too:

  ```bash
  cd services/tms/internal/api/graphql/gqlexec/internal/e2e
  go run github.com/emoss08/trenova/internal/api/graphql/gqlexec/gen
  ```
