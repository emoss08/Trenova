# Frontend GraphQL Migration Agent Playbook

## Goal

Move frontend-owned API usage to the first-party GraphQL contract in a way that multiple agents can execute in parallel without duplicating logic, breaking REST integrations, or bypassing existing services.

This document is for migrating client application data access. REST remains the external integration API unless the product owner explicitly changes that contract.

## Migration Principles

- GraphQL is a first-party client BFF, not a replacement for external REST contracts.
- Use existing domain services, repositories, validators, workflows, permission checks, and audit paths.
- Do not move business rules into GraphQL resolvers or client helpers.
- Keep client query keys and cache invalidation shapes stable whenever possible.
- Prefer explicit GraphQL documents per page/workflow over generic query builders.
- Keep generated output generated: run gqlgen and client codegen instead of hand-editing generated files.
- Persisted operations are mandatory for first-party client GraphQL operations.
- Preserve specialized REST flows when GraphQL is the wrong transport, such as multipart uploads, large file downloads, auth callbacks, webhooks, and third-party integration endpoints.

## Agent Work Contract

Before changing code, produce or update an endpoint inventory for the page/workflow:

| UI Surface | Current REST Call | Purpose | Proposed GraphQL Operation | Owner | Status |
| --- | --- | --- | --- | --- | --- |
| Example table | `GET /api/v1/resource/?limit=...` | list rows | `query ResourceTable` | Agent A | planned |
| Example save | `PUT /api/v1/resource/:id` | save form | `mutation UpdateResource` | Agent B | planned |

For each endpoint, classify it as:

- **Migrate**: frontend-owned read/write that maps cleanly to GraphQL.
- **Keep REST**: external integration, upload/download, auth callback, webhook, or streaming-specific endpoint.
- **Defer**: unclear ownership or contract; needs product/backend direction.

Agents should work in independent vertical slices: schema + resolver + client operation + client helper + UI wiring + tests for one workflow or endpoint group.

## Backend Pattern

Add schema under `services/tms/internal/api/graphql/schema/*.graphqls`.

Use:

- `query` fields for reads.
- `mutation` fields for writes/actions.
- Connection shapes for paginated table/list reads.
- Input objects for mutations with more than a few fields.
- Existing domain output types when already bound in `services/tms/gqlgen.yml`.
- Generated GraphQL models only for transport wrappers, inputs, connections, edges, and response envelopes.

Resolver rules:

- Inject needed services through `services/tms/internal/api/graphql/resolver/resolver.go`.
- Call existing service methods; do not call repositories directly unless that is already the established service boundary for that domain.
- Mirror REST permission checks using `requirePermission`.
- Build `pagination.TenantInfo` from the GraphQL auth context.
- Parse IDs with `pulid.MustParse`.
- Reuse existing mapping helpers where present.
- Keep relation/include booleans as hard gates when REST had explicit include flags.
- Return existing domain/service errors; do not swallow errors or convert them to generic strings.

For lists:

- Use the repo/service pagination model already used by REST.
- Return `edges`, `totalCount`, and `pageInfo`.
- Preserve the current client table semantics: offset tables can be normalized client-side into `GenericLimitOffsetResponse<T>`.

For mutations:

- Match REST parity first.
- Keep optimistic-locking fields such as `version`.
- Preserve audit/workflow side effects by using the same service command path as REST.
- Prefer patch mutations only when the existing domain service has patch semantics.

## Client Pattern

Add operations under `client/src/graphql/operations/<domain>/<page-or-workflow>.graphql`.

Use:

- Fragments that mirror the UI’s existing parsed type.
- Existing generated documents from `client/src/graphql/generated/graphql.ts`.
- `requestGraphQL` from `client/src/lib/graphql`.
- A domain helper under `client/src/lib/graphql/<domain>.ts` for non-table workflows.
- `defineDataTableGraphQLConfig` or the existing table GraphQL path for DataTable migrations.
- Existing Zod schemas via `safeParse` when the REST service previously parsed responses.

Do not create a second GraphQL client, global GraphQL service, or ad hoc fetch wrapper.

## Select Options

GraphQL select options use one generic option contract for autocomplete-style UI:

- Schema: `selectOptions(input: SelectOptionsInput!): SelectOptionConnection!`.
- Generic option: `id`, `label`, optional `description`, optional `meta`.
- Client helper: `client/src/lib/graphql/select-options.ts` normalizes the connection back to `GenericLimitOffsetResponse<SelectOption>`.
- Pagination: this first slice remains offset-backed to match existing REST select-options services. Edges expose opaque cursors for GraphQL connection consistency, but client pagination must use `first` and `offset` until a resource has a real service-level cursor query.
- Autocomplete migration: pass `graphql={{ resource: "RESOURCE_NAME" }}` while keeping the existing REST `link` as compatibility fallback.

Use this path for autocomplete/select field resources that already have a REST `select-options/` endpoint and an existing service-level select-options method. Do not use it for rich domain reads, table rows, uploads/downloads, or workflows that need domain-specific output.

Current files involved:

| Layer | File |
| --- | --- |
| Backend schema | `services/tms/internal/api/graphql/schema/select_options.graphqls` |
| Backend resolver registry and mappers | `services/tms/internal/api/graphql/resolver/select_options.go` |
| Backend generated resolver shim | `services/tms/internal/api/graphql/resolver/select_options.resolvers.go` |
| Client operation | `client/src/graphql/operations/select-options/options.graphql` |
| Client runtime helper | `client/src/lib/graphql/select-options.ts` |
| Client wrappers | `client/src/components/autocomplete-fields.tsx` |

Before adding a resource, answer these questions:

1. Is the resource already tenant-scoped in REST select-options?
2. Does REST require only an authenticated tenant context, or does it require a resource read permission?
3. Which existing service method performs the search? Use that method; do not bypass services or duplicate select SQL in the resolver.
4. How should a selected value be loaded by ID? Prefer an existing service batch helper for multi-ID resources; otherwise use the existing service `Get` method.
5. What is the minimum generic display shape? Keep it to `id`, `label`, `description`, and `meta`; do not expose a full domain object.

### Backend Steps

1. Add the enum value to `SelectOptionResource` in `services/tms/internal/api/graphql/schema/select_options.graphqls`.

   Use screaming-snake enum values, for example:

   ```graphql
   enum SelectOptionResource {
     EQUIPMENT_TYPE
     CUSTOMER
   }
   ```

2. Add any missing service dependency to `services/tms/internal/api/graphql/resolver/resolver.go`.

   Follow the existing resolver dependency-injection pattern. The resolver should call services, not repositories. If the service does not have the method you need, add the method at the service/repository boundary first instead of writing query logic in GraphQL.

3. Add a registry entry in `selectOptionRegistry()`.

   Auth-only tenant-scoped resources look like this:

   ```go
   gqlmodel.SelectOptionResourceCustomer: {
       resolve: r.resolveCustomerSelectOptions,
   },
   ```

   Permissioned resources should set `permissionResource`:

   ```go
   customerPermission := permission.ResourceCustomer

   gqlmodel.SelectOptionResourceCustomer: {
       permissionResource: &customerPermission,
       resolve:            r.resolveCustomerSelectOptions,
   },
   ```

   Define the local permission variable before the returned map. Do not add a generic pointer helper in a resolver file.

4. Add a resolver method for the resource.

   The resolver has two paths:

   - `ids` path: load selected options and ignore search pagination.
   - search path: call the existing service `SelectOptions` method with `req.selectQuery`.

   Batch helpers should preserve the requested ID order with `orderedSelectOptionItems`. Single `Get` loops are acceptable when that is the existing service shape.

   ```go
   func (r *Resolver) resolveCustomerSelectOptions(
       ctx context.Context,
       req selectOptionsRequest,
   ) (*gqlmodel.SelectOptionConnection, error) {
       if len(req.ids) > 0 {
           items := make([]selectOptionConnectionItem, 0, len(req.ids))
           for _, id := range req.ids {
               entity, err := r.customerService.Get(ctx, repositories.GetCustomerByIDRequest{
                   ID:         id,
                   TenantInfo: req.tenantInfo,
               })
               if err != nil {
                   return nil, err
               }
               items = append(items, customerSelectOptionItem(entity))
           }

           return selectOptionConnection(items, len(items), 0)
       }

       result, err := r.customerService.SelectOptions(ctx, req.selectQuery)
       if err != nil {
           return nil, err
       }

       return selectOptionListConnection(
           result,
           req.selectQuery.Pagination.SafeOffset(),
           customerSelectOptionItem,
       )
   }
   ```

5. Add mapper functions.

   Each option item must provide the generic option and an opaque cursor payload. Search queries must select `created_at` and `id`, otherwise cursor encoding will fail.

   ```go
   func customerSelectOptionItem(entity *customer.Customer) selectOptionConnectionItem {
       return selectOptionConnectionItemFor(
           customerSelectOption(entity),
           entity.CreatedAt,
           entity.ID,
       )
   }

   func customerSelectOption(entity *customer.Customer) *gqlmodel.SelectOption {
       return &gqlmodel.SelectOption{
           ID:          entity.ID.String(),
           Label:       entity.Name,
           Description: stringPtr(entity.Code),
           Meta: map[string]any{
               "code": entity.Code,
           },
       }
   }
   ```

   Mapper rules:

   - `label` is the text shown in the closed autocomplete.
   - `description` is optional supporting text.
   - `meta` is for small UI-only defaults, badges, colors, related IDs, or display hints.
   - Keep `meta` keys camelCase because they are consumed by TypeScript.
   - Do not include sensitive fields, full nested objects, audit fields, or large custom payloads.

6. Make the repository/service select-options query return cursor fields.

   If the existing select-options repository query uses explicit columns, include `created_at` along with `id` and the display fields used by the mapper. Keep the query tenant scoping and filters exactly aligned with REST behavior.

7. Add backend tests in `services/tms/internal/api/graphql/resolver/select_options_test.go`.

   Cover:

   - resource mapping into `pagination.SelectQueryRequest`,
   - `ids` lookup path when the resource has special ordering or metadata,
   - mapper output for `label`, `description`, and `meta`,
   - permission policy if the resource is permissioned,
   - resource-specific filters if the resource accepts `filters`.

### Frontend Steps

1. Use the shared GraphQL operation.

   The select-options operation already lives at `client/src/graphql/operations/select-options/options.graphql`. Do not add a resource-specific operation unless the resource no longer fits the generic `SelectOption` shape.

2. Regenerate client GraphQL artifacts after schema enum changes.

   ```bash
   cd client
   pnpm graphql:codegen
   ```

3. Add a stable config constant in `client/src/components/autocomplete-fields.tsx`.

   ```ts
   const customerSelectOptionsGraphQL = {
     resource: "CUSTOMER",
   } satisfies GraphQLSelectOptionsConfig;
   ```

   If the wrapper always needs filters, put stable filters here. If filters come from props, keep them in `extraSearchParams` so the shared autocomplete can merge them into GraphQL `filters`.

4. Migrate the wrapper to `SelectOption`.

   Keep the existing REST `link` for compatibility and pass the GraphQL config:

   ```tsx
   export function CustomerAutocompleteField<T extends FieldValues>({
     ...props
   }: Omit<
     AutocompleteFieldProps<SelectOption, T>,
     "link" | "renderOption" | "getOptionValue" | "getDisplayValue"
   >) {
     return (
       <ControlledAutocompleteField<SelectOption, T>
         {...props}
         link="/customers/select-options/"
         graphql={customerSelectOptionsGraphQL}
         getOptionValue={(option) => option.id}
         getDisplayValue={(option) => option.label}
         renderOption={(option) => (
           <div className="flex flex-col">
             <span>{option.label}</span>
             {option.description && (
               <span className="text-xs text-muted-foreground">
                 {option.description}
               </span>
             )}
           </div>
         )}
       />
     );
   }
   ```

5. Read resource metadata from `option.meta`.

   Use narrow helpers instead of casting full objects:

   ```ts
   function selectOptionMetaString(option: SelectOption, key: string) {
     const value = option.meta?.[key];
     return typeof value === "string" ? value : "";
   }
   ```

   For IDs or defaults, check the type before writing form state:

   ```ts
   const primaryWorkerId = option.meta?.primaryWorkerId;
   if (typeof primaryWorkerId === "string") {
     form.setValue("primaryWorkerId", primaryWorkerId);
   }
   ```

6. Keep non-migrated call sites on REST.

   Do not migrate `MultiSelectField`, table filters, or specialized form flows unless the specific wrapper requires it. The current slice is for shared autocomplete wrappers.

### Pagination and IDs Behavior

- Search uses `first` and `offset`.
- `ids` lookups set offset to `0` and return the selected option(s).
- GraphQL edges include opaque cursors generated from `{createdAt, id}`, but the client helper reports `pageInfo.mode: "offset"`.
- Do not use `pageInfo.endCursor` for select-options pagination until the backend resource uses a real cursor service/repository method.

### Filters

`SelectOptionsInput.filters` is a `JSON` object for resource-specific filters that REST currently accepts through query params.

Rules:

- Keep filters small and explicit.
- Validate or normalize resource-specific filters in the resolver before passing them to a service request.
- Do not pass arbitrary filter objects into query builders.
- Preserve existing REST names when possible so wrappers can keep `extraSearchParams`.

Example:

```go
func equipmentTypeClassesFilter(filters map[string]any) []string {
    value, ok := filters["classes"]
    if !ok {
        value, ok = filters["class"]
    }
    if !ok {
        return nil
    }

    // Convert only supported JSON shapes into service input.
}
```

### Common Mistakes

- Do not hand-edit generated files under `generated/` or `gqlmodel/`.
- Do not put resource-specific domain objects into `SelectOption.meta`.
- Do not bypass services from the GraphQL resolver.
- Do not mark select-options client pagination as cursor mode while the request still uses offset.
- Do not remove REST `link` props from wrappers; they are the compatibility fallback.
- Do not add generic utility helpers to resolver files. Reusable utilities belong under `shared/`.

### Verification Checklist

Run the backend generator and focused backend tests:

```bash
cd services/tms
go run github.com/99designs/gqlgen generate --config gqlgen.yml
go test ./internal/api/graphql/...
```

Run client codegen and focused frontend tests:

```bash
cd client
pnpm graphql:codegen
pnpm vitest run src/lib/__tests__/graphql.test.ts src/components/fields/autocomplete/autocomplete.test.tsx
```

For shared type/schema changes, include:

```bash
cd client
pnpm vitest run src/types/server.test.ts
```

Before handing off, run the normal client gates if the wrapper or shared autocomplete changed:

```bash
cd client
pnpm build
pnpm lint
```

### Short Checklist

1. Add the enum value to `SelectOptionResource` in `services/tms/internal/api/graphql/schema/select_options.graphqls`.
2. Add a resolver registry entry in `services/tms/internal/api/graphql/resolver/select_options.go`.
3. Route through the existing service method for search and through service `Get` or an existing service batch helper for `ids`.
4. Preserve REST permission policy. Use auth-only tenant scoping only for REST resources that already behave that way; set a registry read permission for resources that require `read`.
5. Add a mapper that fills the generic label/description/meta fields without exposing a richer domain object.
6. Add or update the client wrapper in `client/src/components/autocomplete-fields.tsx`.
7. Add the operation/test coverage and regenerate gqlgen plus client codegen.

Query wiring:

- Keep `queries.<domain>.<key>()` names and query-key shapes stable when possible.
- Swap `queryFn` to the GraphQL helper.
- Keep invalidation targets unchanged unless the data ownership changed.
- Leave unrelated REST methods in `apiService` until their UI surface is migrated.

Mutation wiring:

- Use GraphQL helpers in form/action mutation functions.
- Keep existing optimistic mutation hooks and cache updates.
- Preserve current toast/error/form behavior.
- Keep multipart upload and generated download flows on REST unless a signed-url GraphQL design is explicitly approved.

## Generated Artifacts

Backend schema changes:

```bash
cd services/tms
go run github.com/99designs/gqlgen generate --config gqlgen.yml
```

Client operation changes:

```bash
cd client
pnpm graphql:codegen
```

Expected generated paths:

- `services/tms/internal/api/graphql/generated/generated.go`
- `services/tms/internal/api/graphql/gqlmodel/models_gen.go`
- `client/src/graphql/generated/`
- `client/src/graphql/schema.graphql`
- `services/tms/internal/api/graphql/persisted-documents.json`

Do not hand-edit generated files. If generated output is wrong, fix schema/config/source operations and regenerate.

## Testing Expectations

Backend focused tests should cover:

- Resolver permission resource/operation.
- Tenant info and ID mapping.
- Include flags and pagination mapping.
- Mutation input mapping to service calls.
- Error propagation for invalid IDs or service failures where practical.

Client focused tests should cover:

- GraphQL helper uses the expected document, operation name, and variables.
- Helper unwraps and parses the GraphQL response.
- Existing query key still calls the GraphQL helper.
- Form/action mutation calls the GraphQL helper.
- REST fallback is preserved for endpoints intentionally left on REST.

Suggested focused commands:

```bash
cd services/tms
go test ./internal/api/graphql/resolver
go test ./internal/api/graphql/...
go test ./internal/bootstrap

cd client
pnpm vitest run <focused test files>
pnpm build
pnpm lint
```

## Parallel Agent Checklist

Each agent should report:

- Endpoint(s) migrated.
- Endpoint(s) intentionally left on REST and why.
- Schema files changed.
- Resolver/service paths used.
- Client operation/helper/query-key paths changed.
- Generated files updated.
- Verification commands and results.
- Runtime note if persisted operation changes require backend rebuild/restart.

## Common Pitfalls

- Do not duplicate REST handler logic in GraphQL resolvers.
- Do not bypass permission checks because GraphQL is already behind authenticated routing.
- Do not forget the backend persisted manifest embed requires rebuild/restart.
- Do not migrate external integration semantics just because a client page uses the same REST service.
- Do not remove REST methods that other pages or integrations still call.
- Do not normalize GraphQL responses in one-off components when a shared helper or parser already exists.
- Do not let page-level migration scope sprawl into unrelated workflows without an inventory row and owner.
