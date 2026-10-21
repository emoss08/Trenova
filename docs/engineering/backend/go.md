# Go style

The baseline is the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md);
`task lint` (golangci-lint, archlint and the repository's own linters) enforces most of what
follows. This page is what the guide leaves open, decided for Trenova.

## Code

- **No comments.** Names carry the meaning. A doc in `docs/engineering/` carries the why.
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
  handler file is a review failure.
- **No dependency checks that DI already guarantees.** A nil check on an fx-provided dependency,
  or on a field that validation or route binding already requires, is noise.
- **Tests use `t.Context()`**, never `context.Background()`.
- Format with `goimports` and `golines`, on the files you changed.

## Generics and build time

A generic type's methods are compiled again in every package that can see the type. So no
exported type, and no signature of an exported function or method, mentions a concrete generic
type that has methods (`*validationframework.TenantedValidator[*X]`, `*dataloadgen.Loader[K,V]`):
hold it behind a generic interface (`validationframework.TenantedEntityValidator[*X]`) or in an
unexported type reached only through package-level functions. An unexported field of an exported
type still spreads it. `sync.Map` is a generic hash trie underneath; a widely reachable type uses
a plain map behind a mutex. The measurements and the fixes already made are in "Generics and
build time" in [module-architecture.md](../module-architecture.md).

## Dependencies

A new third-party module needs a reason the PR states: what it does that `shared/` or the
standard library cannot, its maintenance, and its licence. Wrap a vendor SDK in an adapter behind
a port so the rest of the code never imports it.
