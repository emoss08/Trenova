# GraphQL Developer Guide

This guide explains how to build first-party client features with GraphQL in Trenova.
REST remains the external integration API. Use GraphQL for client-owned reads and
mutations where typed documents, generated variables, persisted operations, and
selection-aware backend queries make the feature easier to maintain.

## DX Comparison

| Concern | REST today | GraphQL today |
| --- | --- | --- |
| First call site | Usually one `link`, `url`, or `apiService` method. Easy to start. | Requires schema, resolver, operation document, codegen, persisted manifest, and client helper wiring. More setup. |
| Type safety | Mostly hand-maintained TypeScript types and Zod parsing. Drift is easy if a handler changes. | Generated operation variables and result types catch drift during `pnpm graphql:codegen` / `pnpm build`. |
| Table lists | DataTable already knows REST query params and response shape. | DataTable can use GraphQL via `defineDataTableGraphQLConfig`, but the operation must expose a connection and the config must normalize variables. |
| Mutations | Generic form panels can call `api.post` / `api.put` from a URL. Very low ceremony. | Use generated mutation documents through a service method, then pass that method as `mutationFn` to the form panel. |
| Backend implementation | Handler, service, repository are established and visible. | Resolver should be thin and call the same service path as REST; schema and gqlgen add extra files. |
| Field selection | REST usually returns endpoint-defined shapes. Simple, sometimes over-fetches. | GraphQL documents request only needed fields; backend projection can select only requested columns. |
| Contract drift | Found by tests or runtime parsing. | Found earlier by gqlgen, client codegen, persisted manifest checks, and TypeScript. |
| Best fit | External consumers, uploads/downloads, webhooks, auth callbacks, simple endpoint reuse. | First-party client tables, page-specific reads, typed workflow mutations, and selection-sensitive data. |

The practical tradeoff is simple: REST is faster to start; GraphQL is safer and
more expressive once the rails are in place. For client-owned pages, prefer
GraphQL when the feature has a table, related fields, or more than one UI
workflow around the same resource.

## Rules

- Do not expose GraphQL as the external public API unless product direction changes.
- Do not bypass services in GraphQL resolvers. Resolvers call services; services own validation, permissions-adjacent business behavior, workflows, and audit side effects.
- Keep REST behavior stable when REST is still an integration contract.
- Use explicit GraphQL documents. Do not build ad hoc string queries in components.
- Regenerate gqlgen and client artifacts. Do not hand-edit generated files.
- Keep persisted documents synced to `services/tms/internal/api/graphql/persisted-documents.json`.
- Keep uploads, downloads, webhooks, auth callbacks, and third-party integration endpoints on REST unless a dedicated GraphQL design is approved.

## Building A New GraphQL Resource

Use Equipment Type as the reference shape for a table plus create/update/patch/bulk mutations.

### 1. Backend Schema

Add or update schema files under:

```text
services/tms/internal/api/graphql/schema/
```

For a DataTable-backed list, new resources should use the shared connection input:

```graphql
input DataTableConnectionInput {
  first: Int = 20
  after: String
  query: String
  fieldFilters: [FieldFilterInput!]
  filterGroups: [FilterGroupInput!]
  sort: [SortFieldInput!]
}
```

The resource list should expose a connection:

```graphql
type EquipmentTypeEdge {
  node: EquipmentType!
  cursor: String!
}

type EquipmentTypeConnection {
  edges: [EquipmentTypeEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

extend type Query {
  equipmentTypes(input: DataTableConnectionInput!, classes: [EquipmentClass!]): EquipmentTypeConnection!
  equipmentType(id: ID!): EquipmentType!
}
```

Mutations should use typed inputs:

```graphql
extend type Mutation {
  createEquipmentType(input: EquipmentTypeInput!): EquipmentType!
  updateEquipmentType(id: ID!, input: EquipmentTypeInput!): EquipmentType!
  patchEquipmentType(id: ID!, input: EquipmentTypePatchInput!): EquipmentType!
}
```

### 2. Backend Generation

Run gqlgen after schema or resolver contract changes:

```bash
cd services/tms
task gqlgen
```

Use the check task when validating drift:

```bash
cd services/tms
task gqlgen-check
```

`task generate` runs `go generate ./...`; it does not replace gqlgen.

### 3. Resolver Shape

Resolvers should be thin:

1. Call `r.requirePermission(ctx, resource, operation)`.
2. Convert GraphQL input into the service/repository request shape.
3. Call the existing service.
4. Return domain objects or connection wrappers.

For list queries, use the shared GraphQL mapping helper for `DataTableConnectionInput` and pass cursor info into the service/repository path.

For mutations, preserve REST behavior by using the same service methods REST uses. Do not duplicate validation or workflow rules in the resolver.

### 4. Projection

For table lists, preserve GraphQL field selection. The resolver should derive requested `edges.node` fields and pass columns into the repository. The repository should fall back to full columns only when no projection was provided.

The goal is:

- GraphQL requests select only requested fields plus cursor-required fields.
- REST callers keep their existing response behavior.
- Cursor values come from DB-projected sort values, not from partially hydrated structs.

### 5. Client Operation Documents

Add operations under:

```text
client/src/graphql/operations/<domain>/
```

Example:

```graphql
query EquipmentTypeTable($input: DataTableConnectionInput!, $classes: [EquipmentClass!]) {
  equipmentTypes(input: $input, classes: $classes) {
    edges {
      node {
        id
        status
        code
        description
        class
        color
        interiorLength
        version
        createdAt
        updatedAt
      }
    }
    totalCount
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}

mutation CreateEquipmentType($input: EquipmentTypeInput!) {
  createEquipmentType(input: $input) {
    id
    status
    code
    version
  }
}
```

Prefer fragments when the same row shape is used by list, get, create, update, and patch operations.

### 6. Client Codegen

Run:

```bash
cd client
pnpm graphql:codegen
```

Codegen updates:

- `client/src/graphql/generated/`
- `client/src/graphql/schema.graphql`
- `services/tms/internal/api/graphql/persisted-documents.json`

Use the drift check:

```bash
cd client
pnpm graphql:codegen:check
```

The backend embeds the persisted manifest. Rebuild or restart the backend after persisted operation changes.

### 7. Table Wiring

Create a table GraphQL config near related table configs:

```ts
export const equipmentTableGraphQLConfigs = {
  equipmentType: defineDataTableGraphQLConfig<
    EquipmentType,
    EquipmentTypeTableQueryVariables
  >({
    document: EquipmentTypeTableDocument,
    operationName: "EquipmentTypeTable",
    connectionKey: "equipmentTypes",
    buildVariables: ({ pageSize, options }) => ({
      input: {
        first: pageSize,
        after: options?.cursor || undefined,
        query: options?.query || undefined,
        fieldFilters: options?.fieldFilters ?? [],
        filterGroups: options?.filterGroups ?? [],
        sort: options?.sort ?? [],
      },
    }),
  }),
};
```

Then pass it to `DataTable`:

```tsx
<DataTable<EquipmentType>
  name="Equipment Type"
  link="/equipment-types/"
  queryKey="equipment-type-list"
  columns={columns}
  graphql={equipmentTableGraphQLConfigs.equipmentType}
  TablePanel={EquipmentTypePanel}
/>
```

Keep `link` while REST fallback/export behavior still depends on it.

### 8. Mutation Wiring

Add domain service methods that call generated documents through `requestGraphQL`:

```ts
public async create(data: EquipmentType) {
  const response = await requestGraphQL({
    document: CreateEquipmentTypeDocument,
    operationName: "CreateEquipmentType",
    variables: { input: toEquipmentTypeInput(data) },
  });

  return safeParse(equipmentTypeSchema, response.createEquipmentType, "EquipmentType");
}
```

Use a small mapper when the UI/domain row includes fields GraphQL input should not send, such as `id`, tenant IDs, timestamps, or server-managed metadata.

Use the generic form panels through `mutationFn`:

```tsx
<FormCreatePanel
  form={form}
  queryKey="equipment-type-list"
  title="Equipment Type"
  formComponent={<EquipTypeForm />}
  mutationFn={(values) => apiService.equipmentTypeService.create(values)}
/>
```

Avoid local casts in panel call sites. Let the form schema and panel generics infer the submitted value type.

## Verification

For a table plus mutation migration, run the focused path first:

```bash
cd services/tms
task gqlgen-check
go test ./internal/api/graphql/... ./internal/api/handlers/<handler-package> ./internal/core/services/<service-package>
```

```bash
cd client
pnpm graphql:codegen:check
pnpm vitest run src/hooks/data-table/__tests__/use-data-table-query.test.ts src/lib/__tests__/graphql.test.ts
pnpm build
```

Finish with:

```bash
git diff --check
```

## Common Mistakes

- Adding GraphQL schema and forgetting gqlgen.
- Adding client operations and forgetting to sync the backend persisted manifest.
- Calling repositories directly from resolvers.
- Duplicating REST validation in GraphQL instead of calling services.
- Returning full DB columns for GraphQL table reads when projection is available.
- Removing REST fallback props before every caller has moved.
- Sending row-only fields into GraphQL input variables.
- Treating GraphQL as easier because the component code is shorter. Most of the value comes from generated contracts and drift checks, so the generation path is part of the feature.
