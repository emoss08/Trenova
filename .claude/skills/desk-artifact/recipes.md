# Desk artifact recipes

These are copyable shapes taken from merged cards. The fuel purchase and fuel card cards
(`fueldeskcardservice`, `cards/fuel-purchase/`, `cards/fuel-card/`) and the journal cards
(`journalcardservice`, `cards/journal-entry/`) are the reference implementations. Open them
next to these recipes.

## Server

### The port: aggregates and side reads the card needs

`internal/modules/<module>/ports/<entity>ports/deskcard.go`:

```go
type Sum<Thing>Request struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	<Owner>ID  pulid.ID              `json:"<owner>Id"`
	From       int64                 `json:"from"`
	To         int64                 `json:"to"`
}

type <Entity>DeskCardReader interface {
	Sum<Thing>(ctx context.Context, req *Sum<Thing>Request) ([]*<Thing>Row, error)
	List<Related>(ctx context.Context, req *List<Related>Request) ([]*<entity>.<Related>, error)
}
```

Implement it in `adapters/postgres/<entity>repository/deskcard.go` on the generated keel
tables, each method's body inside `dbtx.Read` and passing `r.db.Keel(ctx)` to every terminal. Every list
takes a `Limit`. Add `deskcard_integration_test.go` (`//go:build integration`) next to it.

### The card service

`internal/modules/<module>/app/<entity>deskcardservice/service.go`:

```go
type Params struct {
	fx.In

	Logger *zap.Logger
	Records <entity>ports.<Entity>Repository
	Desk    <entity>ports.<Entity>DeskCardReader
}

type recordReader interface {
	GetByID(ctx context.Context, req *<entity>ports.GetByIDRequest) (*<entity>.<Entity>, error)
}

type Service struct {
	l       *zap.Logger
	records recordReader
	desk    <entity>ports.<Entity>DeskCardReader
	now     func() time.Time
}

type Get<Entity>CardRequest struct {
	TenantInfo     pagination.TenantInfo
	ID             pulid.ID
	Include<Other> bool
}

type <Entity>Card struct {
	Record *<entity>.<Entity>
}

func (s *Service) Get<Entity>Card(ctx context.Context, req *Get<Entity>CardRequest) (*<Entity>Card, error)
```

- The service declares narrow interfaces for what it reads, so its tests use small fakes.
- `now` is injected so tests can pin the clock.
- Each permission-gated section is read only when its `Include…` flag is set.
- Register both in `module.go`: `<entity>repository.NewDeskCardReader` and
  `<entity>deskcardservice.New`.

### The resolver

```go
func (r *QueryResolver) <Entity>DeskCard(ctx context.Context, id string) (*<entity>deskcardservice.<Entity>Card, error) {
	authCtx, err := r.RequirePermission(ctx, permission.Resource<Entity>, permission.OpRead)
	if err != nil {
		return nil, err
	}

	recordID, err := gqlid.RequiredID("id", id)
	if err != nil {
		return nil, err
	}

	return r.<Entity>DeskCards.Get<Entity>Card(ctx, &<entity>deskcardservice.Get<Entity>CardRequest{
		TenantInfo:     resolvercore.TenantInfo(authCtx),
		ID:             recordID,
		Include<Other>: r.HasPermission(ctx, authCtx, permission.Resource<Other>, permission.OpRead),
	})
}
```

Then:

1. Add the service to `Params`, `Services` and `newServices` in `resolver/services.go`.
2. Bind the GraphQL types to the service structs in `gqlgen.yml`.
3. Run `go run ./internal/api/graphql/gqlexec/gen`, then
   `go generate ./internal/api/graphql/projection/...` and `./internal/api/writecoverage/...`.
4. Write resolver tests (`<entity>_desk_card_test.go`) with
   `mocks.AllowAllPermissionEngine` and `mocks.DenyResourcesPermissionEngine{Denied: …}`.
   Assert that a gated section is empty and that its reader was never called.

## Client

### The operation, the wrapper and the key

`client/packages/graphql/src/operations/<area>/desk.graphql`:

```graphql
query Desk<Entity>Card($id: ID!) {
  <entity>DeskCard(id: $id) {
    record { ...<Entity>Fields }
    # …every section the card draws
  }
}
```

`apps/web/src/lib/graphql/<area>-desk-cards.ts`:

```ts
export type Desk<Entity>Card = Omit<Raw, "record"> & { record: <Entity>FieldsFragment };

export async function fetchDesk<Entity>(id: string, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: Desk<Entity>CardDocument,
    operationName: "Desk<Entity>Card",
    variables: { id },
    signal: options?.signal,
  });
  const card = data.<entity>DeskCard;
  return { ...card, record: getFragmentData(<Entity>FieldsFragmentDoc, card.record) };
}
```

`apps/web/src/lib/queries/<entity>.ts`:

```ts
export const <entity> = createQueryKeys("<entity>", {
  card: (id: string) => ({
    queryKey: ["card", id],
    queryFn: async ({ signal }) => fetchDesk<Entity>(id, { signal }),
  }),
});
```

### The card's two surfaces

```tsx
export function Desk<Entity>({ recordId, order: rowOrder = NO_ROWS, onSelect, onBack, backLabel }: Props) {
  const t = useT();
  const query = useQuery({ ...<entity>.card(recordId), retry: false });
  const card = query.data;
  if (!card) {
    return query.isError ? (
      <CardUnreadable kind={t("<Entity>")} title={t("This <entity> could not be read")}
        error={query.error} onBack={onBack} backLabel={backLabel} />
    ) : (
      <Desk<Entity>Skeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }
  return <EntityCard key={card.record.id} card={card} rowOrder={rowOrder}
    onSelect={onSelect} onBack={onBack} backLabel={backLabel} />;
}

export function Desk<Entity>Artifact({ recordId }: RecordArtifactProps) {
  return <Desk<Entity> recordId={recordId} />;
}
```

Inside `EntityCard`:

1. Call `useListedRecords({ entity, order, backLabel: t("Back to {0}", title) })` for each kind
   the card names, and return `listed.opened` when one is open.
2. Draw `RecordCard className="dk-xyz"`, then `CardHead`, which takes the kicker, the stepper,
   the lead mark, the title, the subtitle, the figure, and the meta (`CardStatus` plus
   `CardFacts`).
3. Draw the signature section, then the `CardSection`s, then `CardDocuments` if the record
   carries documents, then `CardBar`, which holds the hint, the link to the record's page,
   and the actions.

### Registry entry

```tsx
const load<Entity> = preloadable(() => import("./<resource>/desk-<resource>"));

<record_kind>: {
  resource: Resource.<Entity>,
  Skeleton: Desk<Entity>Skeleton,
  inPlace: lazyComponent(load<Entity>, (module) => module.Desk<Entity>),
  artifact: lazyComponent(load<Entity>, (module) => module.Desk<Entity>Artifact),
},
```

Import the skeleton statically. It stays in the workspace chunk, so the wait for the code
looks the same as the wait for the record.

### An action that reuses the page's dialog, then follows up

The fuel card's Assign button is the pattern to copy:

1. The button opens the fuel cards page's own `AssignFuelCardDialog`.
2. The dialog's `onAssigned` callback runs the follow-up step (`postHeldRows`, a pure helper
   that takes the resolver and the error describer as arguments, so it can be tested).
3. It collects refusals into state and shows them with `CardSectionError`, then invalidates the
   card's keys and the page's list keys.
4. The button shows a busy label while the follow-up runs, and there is no toast.

## Tests

```ts
const fetchers = vi.hoisted(() => ({ card: vi.fn() }));
vi.mock("@/lib/graphql/<area>-desk-cards", () => ({ fetchDesk<Entity>: fetchers.card }));
vi.mock("../../<other>/desk-<other>", () => ({
  Desk<Other>: ({ recordId, backLabel }: { recordId: string; backLabel?: string }) => (
    <p>Other card {recordId} · {backLabel}</p>
  ),
  Desk<Other>Artifact: () => null,
}));

beforeEach(() => {
  seedPermissions({ [Resource.<Entity>]: Operation.Read });
  fetchers.card.mockReset().mockImplementation(async (id: string) => {
    const found = byId[id];
    if (!found) throw new Error("<Entity> not found");
    return structuredClone(found);
  });
});
afterEach(() => clearPermissions());
```

- Render the card inside `QueryClientProvider` (with `retry: false`) and `MemoryRouter`.
- Find sections with `getByRole("heading", { name, level: 3 }).closest("section")`.
- Assert exact text, and exact `href`s built with `recordPath`.

## Mock API

```js
// fixtures/desk-<area>.mjs
export function desk<Entities>(anchor) { return { <id>: { /* the query's shape, times from anchor */ } }; }
export const DESK_<AREA>_CONVERSATIONS = {
  "<slug>": { recordId: "<id>", title: "…", question: "…", reply: "… [Title](artifact:art_<slug>) …" },
};

// handlers/deskcards.mjs
Desk<Entity>Card: (state, variables) => {
  if (held(state)) return HOLD;
  const card = desk<Entities>(state.anchor)[variables.id];
  if (!card) throw new TriageError("<Entity> not found", { code: "NOT_FOUND", type: "not-found" });
  return { <entity>DeskCard: card };
},
```

Register the conversations in `fixtures/desk-cards.mjs` twice: once in `deskCardArtifacts`
and once in the turns list. Add the card to `CARDS` in `desk-cards-screenshots.mjs`, then run:

```
node e2e/mock-api/server.mjs &           # from client/apps/web
pnpm dev &                               # the web app on :5173
node e2e/mock-api/desk-cards-screenshots.mjs <outDir> <name>-loaded-light,<name>-attention-light,…
```
