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
  mutationFn: (values: FormValues) => apiService.thingService.create(values),
  onSuccess: () => {
    void queryClient.invalidateQueries({ queryKey: thing._def });
    toast.success(t("Saved"));
  },
});
```

## A heavy library, loaded at the moment of use

```ts
mutationFn: async (values) => {
  const { buildCoverSheetPdf } = await import("@/lib/capture-cover-sheet"); // brings pdf-lib
  return buildCoverSheetPdf(sheets, wording);
},
```

Keep what the caller needs synchronously (types, the function that opens the result) in a
separate light module (`capture-cover-sheet-print.ts`), so the static import stays small.

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
