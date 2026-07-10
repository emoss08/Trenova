# DataTable GraphQL Connection Migration — Per-Entity Recipe

Proven on `accessorialCharge`. Mirror `equipmentType` as the canonical template. Production-grade, no stubs.

Backend root: `services/tms`  •  Frontend root: `client`

## Canonical reference files (READ FIRST per entity)
- Schema: `services/tms/internal/api/graphql/schema/equipment_type.graphqls`
- Resolver query methods: `.../resolver/equipment_type.resolvers.go` (EquipmentTypes + EquipmentType, ~L125-160)
- Resolver glue: `.../resolver/equipmenttypemapping.go` (columns + connectionToModel; DROP the `Classes`/`equipmentClassStrings` bits — those are equipment-specific)
- Repo cursor List: `.../postgres/repositories/equipmenttyperepository/equipmenttype.go` (List + applyCursorPageFilters + applyTotalCountFilters + applyEquipmentTypeColumns)
- Ports: `.../ports/repositories/equipmenttype.go` (request struct)
- resolver.go registration: `.../resolver/resolver.go` (grep equipmentTypeService)
- gqlgen bindings: `services/tms/gqlgen.yml` (EquipmentType ~L147; enum bindings ~L36)
- Frontend config: `client/src/lib/graphql/equipment-table.ts`
- Frontend operation: `client/src/graphql/operations/equipment/table.graphql`
- Frontend data-table config helper: `client/src/lib/graphql/data-table.ts` (defineDataTableGraphQLConfig)

## Per-entity steps (each existing config entity already has domain/ports/repo/service for REST — do NOT break REST)

### Backend
1. **Domain** (`internal/core/domain/<entity>/<entity>.go`):
   - Embed `pagination.CursorValueSet` with tag ``json:"-" bun:",embed"`` right after `bun.BaseModel`. **(REQUIRED — cursor scan columns; without it: runtime "cursor list item does not expose cursor values".)**
   - Ensure `GetCreatedAt() int64` method exists (satisfies `pagination.CursorEntity`). Add if missing.
   - Ensure `GetPostgresSearchConfig()` + `GetTableName()` exist (usually already there).
   - `goimports -w` the file (adds pagination import).
2. **Ports** (`internal/core/ports/repositories/<entity>.go`): add
   `List<Entity>ConnectionRequest{ Filter *pagination.QueryOptions; Cursor pagination.CursorInfo; <Entity>Columns []string }`
   and `ListConnection(ctx, *List<Entity>ConnectionRequest) (*pagination.CursorListResult[*<entity>.<Entity>], error)` to the interface. KEEP existing offset `List`.
3. **Repo** (`.../<entity>repository/<entity>.go`): add `ListConnection` + `applyCursorPageFilters` + `applyTotalCountFilters` + `apply<Entity>Columns`, mirroring equipmentType (use `buncolgen.<Entity>Table.Alias`, `querybuilder.ApplyCursorFilters`/`ApplyFiltersWithoutSort`, `dbhelper.CursorList`). KEEP existing offset List.
4. **Service** (`.../<entity>service/service.go`): add `ListConnection` delegating to repo.
5. **Schema** `.../schema/<entity>.graphqls`: add (or extend existing file) `type <Entity>` (all domain fields + enums), `<Entity>Edge`, `<Entity>Connection`, and `extend type Query { <entities>(input: DataTableConnectionInput!): <Entity>Connection!  <entity>(id: ID!): <Entity> }`. Reuse shared `EntityStatus`. Define new enums only if not already declared elsewhere.
6. **gqlgen.yml**: bind `<Entity>` to domain model + any new enums. Fields with no gqlgen marshaler (e.g. `decimal.Decimal` amount) → mark as resolver field (see accessorial: amount → Float! resolver using `InexactFloat64`; nullable enum → resolver returning nil for empty).
7. **Run gqlgen**: `cd services/tms && go run github.com/99designs/gqlgen generate`
8. **Resolver glue** `.../resolver/<entity>mapping.go`: `<entity>Columns(ctx, "edges.node")` using `projection.<Entity>Spec`; `<entity>ConnectionToModel` using generic `entityCursorConnection`.
9. **Fill resolver stubs** in generated `.../resolver/<entity>.resolvers.go`: mirror equipment_type body (requirePermission → dataTableConnectionFromGraphQL → service.ListConnection → connectionToModel). Verify permission const via `grep permission.Resource<Entity>`.
10. **Register service** in `resolver.go` (Params field, struct field, New()).
11. **Projection spec**: the projection generator (`task generate`) is BROKEN on master (unrelated ShipmentEvent error). Hand-add `<Entity>Spec` to `.../projection/specs_gen.go`, mirroring `EquipmentTypeSpec` exactly (uses `buncolgen.<Entity>FieldMap`, same AlwaysColumns, all fields + businessUnit/organization relations).
12. **Mock**: hand-edit `internal/testutil/mocks/mock_<Entity>Repository.go` — add `ListConnection` block mirroring the file's existing `List` block (method + _Call type + Expecter + Run/Return/RunAndReturn). Also fix any hand-written fakes implementing the interface (grep `fake<Entity>Repo`). **DO NOT run mockery (crashes).**
13. Verify compilation:
   - `cd services/tms && go build ./...` (non-test code).
   - **DO NOT run the whole test suite** (`go test ./...` — even `-count=0` — causes issues; forbidden).
   - Instead, catch test-only interface breaks PROACTIVELY: for each entity, after adding `ListConnection` to the interface, `grep -rln "Mock<Entity>Repository\|<lowercase>Repository) List(" --include=*.go` is unreliable; do: `grep -rln "<Entity>Repository" services/tms --include=*_test.go` and `grep -rl "mock_<Entity>Repository" services/tms`. Update the generated mock `internal/testutil/mocks/mock_<Entity>Repository.go` (append `ListConnection` block mirroring the existing `List` block) AND every hand-written test double (types like `stub<Entity>Repository`, `fake<Entity>Repo` in *_test.go) by adding a `ListConnection` method.
   - Compile ONLY the specific affected test packages to confirm, e.g. `go vet ./internal/core/services/<entity>service/... ./internal/api/handlers/<entity>handler/...` plus any package your grep found referencing the interface. Never `./...` for tests.

### Frontend
14. **Operation** `client/src/graphql/operations/<domain>/table.graphql`: add row fragment (fields matching columns/type) + `query <Entity>Table($input: DataTableConnectionInput!) { <entities>(input:$input){ edges{node{...<Entity>TableRowFields}} totalCount pageInfo{...DataTablePageInfoFields} } }`.
15. **Frontend codegen**: `cd client && pnpm graphql:codegen` → generates `<Entity>TableDocument` + `<Entity>TableQueryVariables`.
16. **Config** in `client/src/lib/graphql/<entity>-table.ts` (or shared registry): `defineDataTableGraphQLConfig<<Entity>, <Entity>TableQueryVariables>({ document, operationName:"<Entity>Table", connectionKey:"<entities>" })`.
17. **Table file**: remove `link=` and `exportModelName=`, add `graphql={<config>}`.
18. Verify: `cd client && pnpm tsc --noEmit` shows no error for the table; `cd services/tms && go build ./...` passes.

## Gotchas (learned)
- CursorValueSet embed is mandatory (step 1).
- Mockery crashes — hand-edit mocks (step 12) + hand-fix any `fake*Repo` test doubles.
- Projection generator broken — hand-add spec (step 11).
- decimal/other non-marshalable fields → gqlgen resolver fields.
- Some entities already have a `.graphqls` WITH a connection (e.g. equipmentManufacturer) → frontend-only (steps 14-18). Some have a `.graphqls` WITHOUT connection → add connection + rest.
