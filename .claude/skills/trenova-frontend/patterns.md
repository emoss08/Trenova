# Frontend patterns

Copy these shapes; each is taken from code that follows the rules in SKILL.md.

## A list page (`routes/commodity/`)

```tsx
// routes/commodity/page.tsx
const Table = lazy(() => import("./_components/commodity-table"));

export function CommoditiesPage() {
  const t = useT();
  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Commodities"),
        description: t("Manage and configure commodities for your organization"),
      }}
    >
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
```

```tsx
// routes/commodity/_components/commodity-table.tsx
export default function CommodityTable() {
  const t = useT();
  const columns = useMemo(() => getColumns(t), [t]);
  return (
    <DataTable<CommodityRow>
      name="Commodity"
      emptyTitle={t("No commodities yet")}
      queryKey="commodity-list"
      graphql={commodityTableGraphQLConfig}
      resource={Resource.Commodity}
      columns={columns}
      TablePanel={CommodityPanel}
    />
  );
}
```

```ts
// lib/graphql/commodity-table.ts
export const commodityTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: CommodityTableDocument,
  operationName: "CommodityTable",
  connectionKey: "commodities",
});
export type CommodityRow = DataTableConfigRow<typeof commodityTableGraphQLConfig>;
```

The panel is `FormCreatePanel`/`FormEditPanel` with a `useForm({ resolver:
zodResolver(commoditySchema), defaultValues })` (see `commodity-panel.tsx`).

## A lazy section with its own skeleton

```tsx
// overview-skeleton.tsx: tiny, primitives only, never imports the section
export function OverviewSkeleton() {
  const t = useT();
  return (
    <div aria-busy aria-label={t("Loading the overview")} className="grid gap-4">
      <div aria-hidden className="contents">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    </div>
  );
}

// page.tsx
const overview = lazyComponent(
  () => import("./_components/overview/overview-tab"),
  (module) => module.default,
);

<LazyComponent fallback={<OverviewSkeleton />}>
  <overview.Component />
</LazyComponent>;
// Start the download on intent: onPointerEnter={overview.preload}
```

Inside `overview-tab.tsx`, while its own `useQuery` is pending, draw the same skeleton body.

## A query key factory and its fetcher

```ts
// lib/graphql/tractor-card.ts
export async function fetchDeskTractor(id: string, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: DeskTractorCardDocument,
    operationName: "DeskTractorCard",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.tractor) {
    throw new Error("Tractor not found");
  }
  return data.tractor;
}

// lib/queries/tractor.ts: one factory per module, imported directly
export const tractor = createQueryKeys("tractor", {
  card: (tractorId: string) => ({
    queryKey: ["card", tractorId],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchDeskTractor(tractorId, { signal }),
  }),
});

// a component
import { tractor } from "@/lib/queries/tractor";
const { data } = useQuery(tractor.card(id));
// alias only when a local of the same name exists: import { tractor as tractorQueries }
```

## A mutation from a form

```tsx
const save = useApiMutation({
  form,                       // server field errors land on the form's fields
  resourceName: "Cover sheet",
  mutationFn: (values: FormValues) => thingService.create(values),
  onSuccess: () => {
    void queryClient.invalidateQueries({ queryKey: thing._def });
    toast.success(t("Saved"));
  },
});
```

## A button that waits for the server

The control shows its work and cannot fire twice; the write it calls refuses to run on top
of itself. `useApiMutation` reports a failure, so the handler only has to hold and let go.

```tsx
const remove = useApiMutation({
  scope: { id: `thing-remove:${thing.id}` },   // a second call waits instead of racing
  resourceName: "Thing",
  mutationFn: () => thingService.remove(thing.id),
  onSuccess: () => void queryClient.invalidateQueries({ queryKey: thingQueries._def }),
});

<Button
  isLoading={remove.isPending}                  // spinner, and disabled while pending
  onClick={() => void remove.mutateAsync().catch(() => undefined)}
>
  {t("Remove")}
</Button>
```

When the work is a chain (save, then refetch, then a toast), keep one in-flight guard where
the chain starts and let the caller await it:

```tsx
const changing = useRef(false);
const keep = async (changes: Changes) => {
  if (changing.current) return false;          // two clicks can land before a re-render
  changing.current = true;
  try {
    await save(changes);
    await queryClient.invalidateQueries({ queryKey: thingQueries.detail(id).queryKey });
    return true;
  } catch {
    return false;                               // the save already reported it
  } finally {
    changing.current = false;
  }
};
```

A control that has no label to swap (a checkbox, an icon button, a toggle) is replaced by a
`Spinner` in a box of the same size while it saves, so the row does not move
(`billing-queue-documents-tab.tsx`, the "In packet" checkbox). Tests click it twice and
assert one request (`billing-queue-packet-preview.test.tsx`, "removes a document once").

## A heavy library, loaded at the moment of use

```ts
mutationFn: async (values) => {
  const { buildCoverSheetPdf } = await import("@/lib/capture-cover-sheet"); // brings pdf-lib
  return buildCoverSheetPdf(sheets, wording);
},
```

Keep what the caller needs synchronously (types, the function that opens the result) in a
separate light module (`capture-cover-sheet-print.ts`), so the static import stays small.

## A rule the rules module answers

Money splits, report formatting, lane keys and matrix coverage come from the WebAssembly
module (`@trenova/shared/rules/*`, `docs/engineering/rules-engine.md`), never from a TS copy.

```tsx
// Render: suspend until loaded, under the section's own skeleton; one call for the whole list.
<LazyComponent fallback={<ChargeLinesSkeleton />}>
  <ChargeLines charges={charges} />
</LazyComponent>;

function ChargeLines({ charges }: { charges: Charge[] }) {
  const rules = useRules();
  const { totals } = chargeTotals(rules, { charges }); // decimal strings; Number() only to format
}

// A form: the resolver awaits the module, so validation never runs before it has loaded.
useForm({
  resolver: rulesResolver((rules) => zodResolver(withChargeAllocations(schema, rules, total))),
});
```

A message the module returns is shown with `translateMessage(message, t)`.

## A sheet of hundreds of rows (`board/views/timeline/timeline-canvas.tsx`)

- **Virtualize by row with TanStack Virtual.** The sheet is a list of
  `{ key, height, header?, node }` lines, each at a known height, placed absolutely inside a
  `ScrollArea` viewport; only the lines in view plus an overscan are mounted. Lines are measured
  from the sticky axis's bottom edge (no `scrollMargin`) and drawn `AXIS_HEIGHT` lower, so the
  first line in range is the first one below the axis. A `rangeExtractor` keeps that line's
  group header mounted and renders it in flow as `sticky` (negative bottom margin): it pins the
  moment it reaches the axis, and the next header slides over it. `scrollPaddingStart`/`End`
  keep a revealed row clear of the pinned header and the bottom edge. The component is
  `"use no memo"`: the virtualizer is a mutable object.
- **Fill the width.** A range narrower than the sheet stretches to fill it
  (`createTimelineScale(range, days, minWidth)` with the measured width), or a wide or
  zoomed-out window shows empty sheet past the range's end.
- **One hover card for the whole sheet, not a tooltip per row.** A per-row `Tooltip` mounts a
  provider, root and trigger hundreds of times. `timeline-hover-card.tsx` is the shared
  `PointerHoverCard`, fed by a tiny external store: a bar calls `enter`/`move`/`leave` from context and holds no state, so
  hovering redraws only the card, and following the pointer writes its position to the DOM once a
  frame. It opens after a short rest, switches bars at once while open, opens on keyboard focus
  only, and closes on scroll, Escape, pointer-down and when its bar unmounts (`release`).
- **A scroll fade never covers a sticky cell.** Offset the mask (`maskClassName`) past the
  sticky axis, the pinned header and the label column.
- **Hover and the clock stay out of the parent.** A row reads its own hover from the store
  (`useShipmentBoardStore((s) => s.highlightId === id)`), and only the part that moves with time
  calls `useNow`. A parent that subscribes to either redraws every row on each change.
- **A sticky cell's fill is opaque** (`bg-surface-selected`, never a `/10` tint), or what scrolls
  under it shows through.

## Views of one set of records (`board/shipment-board.tsx`)

Table, timeline and map are three views of the same rows, so switching between them
redraws nothing: a view mounts the first time it is opened and then stays, in `Activity`.

```tsx
const [opened, setOpened] = useState<ReadonlySet<View>>(() => new Set([view]));
if (!opened.has(view)) setOpened(new Set(opened).add(view));

{opened.has("timeline") ? (
  <Activity mode={view === "timeline" ? "visible" : "hidden"}>…</Activity>
) : null}
```

Guard with `opened`: `Activity` renders a hidden child anyway, so an unguarded one loads
its chunk (the Maps script, here) on a page that never shows it. A hidden tree loses its
effects, which is what stops its queries and timers, and it keeps its state. A browser
replays a CSS animation on anything that returns from `display: none`, so an entrance must
not run twice: a row records that it has risen (`markRowEntered` in `data-table-body.tsx`).

## Dragging cards between lanes (`billing-queue-document-board.tsx`)

- **dnd-kit, never native drag.** HTML5 drag gives no live preview, drags nothing in Firefox
  from a `<button>`, and drops "before the card under the pointer", so a card can never move
  one place right. One `DndContext` holds every lane; each lane is a `SortableContext` plus a
  `useDroppable` for its empty space.
- **Clicks still select.** `MouseSensor` with `activationConstraint: { distance: 6 }` and
  `TouchSensor` with a short `delay`; spread only `listeners` on the activator, never
  `attributes`, which overwrite your own `aria-pressed` and role.
- **Show the landing before the drop.** The card crosses lanes in local state in `onDragOver`
  (pure helpers in `billing-queue-board-lanes.ts`), the dragged card's own spot is a dashed
  placeholder, and a `DragOverlay` carries a lifted copy. Save once in `onDragEnd`, and only if
  the order changed. A lane whose order isn't kept uses a no-op strategy (`() => null`) so its
  cards hold still.
- **A selected card carries the selection.** Fold the rest of the group out of the lanes on
  drag start, show the count on the lifted card, and land them after it on drop
  (`lanesWithoutGroup`/`lanesWithGroup`). A drop over nothing changes nothing.
- **Name the drag for screen readers** through `accessibility.announcements` with `t()`; the
  library's defaults are English.
- **Two docks at once stack.** `SelectionDock` takes a slot from `useSelectionDockSlot`
  (`components/selection-dock-stack.tsx`) and rises one step per open dock below it; it stays
  where it renders, so a dock inside a modal sheet is still inside it.

## Showing a file

```tsx
// A PDF: the browser's own viewer, fetched with the session
<PdfFrame
  endpoint={documentContentEndpoint(documentId, "view")}
  title={fileName}
  page={3}
  loading={<DocumentBodySkeleton />}
/>

// An image the API serves
<img src={documentContentUrl(documentId, "view")} alt={fileName} />

// Code
<HighlightedCode code={json} language="json" query={search} />
```

## Text

```tsx
const t = useT();
t("Saved");                                            // plain
t("{0} saved", name);                                  // a value in a sentence
t("{0, plural, one {# stop} other {# stops}}", count); // plural
const rt = useRichT();
rt("You've used <b>{0}</b> of this month's allowance", { b: (c) => <b>{c}</b> }, pct);

// outside a component
toast.error(translate("Failed to update status"));

// a module-level map
const STATUS_LABELS = defineLabels({ InReview: "In review", Approved: "Approved" });
STATUS_LABELS[status]; // current language at every read
```

## A test

```tsx
const mocks = vi.hoisted(() => ({ download: vi.fn() }));
vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, download: mocks.download } };
});

it("draws the page's outline while the PDF downloads", () => {
  mocks.download.mockImplementation(() => new Promise<never>(() => {}));
  render(<Panel documentId="doc_1" fileName="ratecon.pdf" />, { wrapper: QueryWrapper });
  expect(screen.getByLabelText("Loading the document")).toHaveAttribute("aria-busy", "true");
  expect(mocks.download).toHaveBeenCalledWith("/documents/doc_1/view/", "document.pdf", {
    signal: expect.any(AbortSignal),
  });
});
```
