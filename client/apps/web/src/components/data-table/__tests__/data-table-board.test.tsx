import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import React from "react";
import { DataTable } from "../data-table";
import type {
  ColumnDef,
  DataTableGroupKey,
  DataTableGrouping,
} from "@trenova/shared/types/data-table";

type TestRow = { id: string; name: string; stage: number };

const testColumns: ColumnDef<TestRow>[] = [{ accessorKey: "name", header: "Name" }];
const testGraphQLConfig = {
  document:
    "query TestTable($input: DataTableConnectionInput!) { tests(input: $input) { totalCount } }",
  operationName: "TestTable",
  connectionKey: "tests",
};
const { defaultQueryResult, useDataTableQueryMock } = vi.hoisted(() => {
  const defaultQueryResult = {
    data: {
      results: [
        { id: "1", name: "Alice", stage: 1 },
        { id: "2", name: "Bob", stage: 1 },
        { id: "3", name: "Cara", stage: 3 },
      ],
      count: 3,
    },
    isLoading: false,
    isError: false,
    error: null,
  };
  return {
    defaultQueryResult,
    useDataTableQueryMock: vi.fn((..._args: unknown[]): unknown => defaultQueryResult),
  };
});

vi.mock("@/hooks/use-permission", () => ({
  // Both hooks: the config manager inside the table reaches for the
  // single-operation one, and a factory missing it throws only on the runs
  // that get far enough to render it.
  usePermission: () => ({ allowed: true, isLoading: false }),
  usePermissions: () => ({
    canRead: true,
    canCreate: true,
    canUpdate: true,
    canExport: true,
    canImport: true,
    isLoading: false,
  }),
}));

vi.mock("@/hooks/data-table/use-data-table-query", () => ({
  useDataTableQuery: useDataTableQueryMock,
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    tableConfiguration: {
      default: () => ({ queryKey: ["tableConfig-default"], queryFn: () => null }),
      all: () => ({
        queryKey: ["tableConfig-all"],
        queryFn: () => ({ results: [], count: 0 }),
      }),
    },
  },
}));

vi.mock("@trenova/shared/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T): T => value,
}));

vi.mock("@/lib/data-table", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/data-table")>()),
  initializeFilterItemsFromFieldFilters: () => [],
  initializeFilterItemsFromFilterGroups: () => [],
  updateSortField: (_sort: unknown, field: string, direction: unknown) => [{ field, direction }],
}));

// The toolbar lazy-loads its panels, and nothing here waits for them, so
// each module graph finished loading in the background of whichever test
// happened to be running and held its event loop for seconds. On a loaded
// CI runner that pushed an unrelated cursor test past its timeout. Loading
// them once, up front, keeps that cost out of every test.
beforeAll(async () => {
  await Promise.all([
    import("@/components/data-table/data-table-search"),
    import("@/components/data-table/data-table-filter-builder"),
    import("@/components/data-table/data-table-sort-builder"),
    import("@/components/data-table/data-table-format-builder"),
    import("@/components/data-table/data-table-display-menu"),
    import("@/components/data-table/data-table-config-manager"),
    import("@/components/data-table/data-table-export-dialog"),
    import("@/components/data-table/data-table-ask"),
  ]);
}, 60_000);

function createQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
}

function renderDataTable(props?: Partial<React.ComponentProps<typeof DataTable<TestRow>>>) {
  const queryClient = createQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <NuqsTestingAdapter hasMemory>
        <DataTable<TestRow>
          columns={testColumns}
          name="test-table"
          queryKey="test"
          graphql={testGraphQLConfig}
          {...props}
        />
      </NuqsTestingAdapter>
    </QueryClientProvider>,
  );
}

function lastQueryOptions() {
  const calls = useDataTableQueryMock.mock.calls as unknown[][];
  return calls[calls.length - 1][3] as {
    sort: { field: string; direction: string }[];
    fieldFilters: { field: string; operator: string; value: unknown }[];
  };
}

function grouping(overrides: Partial<DataTableGrouping<TestRow>> = {}): DataTableGrouping<TestRow> {
  return {
    field: "stage",
    groups: [
      { key: 1, label: "Late", count: 12, aggregate: "$9,400" },
      { key: 2, label: "Needs coverage", count: 4 },
      { key: 3, label: "Moving", count: 30 },
    ],
    getGroupKey: (row) => row.stage,
    collapsedKeys: [],
    onToggleGroup: vi.fn(),
    ...overrides,
  };
}

function press(key: string) {
  act(() => {
    window.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true }));
  });
}

describe("DataTable grouping", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("sorts by the group field first and draws a header with the group's totals", () => {
    renderDataTable({ grouping: grouping() });

    expect(lastQueryOptions().sort[0]).toEqual({ field: "stage", direction: "asc" });
    const late = document.querySelector('tr[data-group-key="1"]');
    expect(late?.textContent).toContain("Late");
    expect(late?.textContent).toContain("12");
    expect(late?.textContent).toContain("$9,400");
    expect(document.querySelector('tr[data-group-key="3"]')?.textContent).toContain("Moving");
    expect(document.querySelector('tr[data-group-key="2"]')).toBeNull();
  });

  it("asks the server to leave out collapsed groups and still shows their headers", () => {
    renderDataTable({ grouping: grouping({ collapsedKeys: [2] }) });

    expect(lastQueryOptions().fieldFilters).toContainEqual({
      field: "stage",
      operator: "notin",
      value: [2],
    });
    const collapsed = document.querySelector('tr[data-group-key="2"]');
    expect(collapsed?.getAttribute("data-collapsed")).toBe("true");
  });

  it("keeps a just-opened group's header in place while its rows load", () => {
    const { rerender } = renderDataTable({ grouping: grouping({ collapsedKeys: [2] }) });
    useDataTableQueryMock.mockImplementation(() => ({
      data: {
        results: [
          { id: "1", name: "Alice", stage: 1 },
          { id: "3", name: "Cara", stage: 3 },
        ],
        count: 2,
      },
      isLoading: false,
      isPlaceholderData: true,
      isError: false,
      error: null,
    }));

    rerender(
      <QueryClientProvider client={createQueryClient()}>
        <NuqsTestingAdapter hasMemory>
          <DataTable<TestRow>
            columns={testColumns}
            name="test-table"
            queryKey="test"
            graphql={testGraphQLConfig}
            grouping={grouping({ collapsedKeys: [] })}
          />
        </NuqsTestingAdapter>
      </QueryClientProvider>,
    );

    const opened = document.querySelector('tr[data-group-key="2"]');
    expect(opened).not.toBeNull();
    expect(opened?.getAttribute("data-collapsed")).toBeNull();
    expect(screen.getByText("Cara")).toBeTruthy();
    useDataTableQueryMock.mockImplementation(() => defaultQueryResult);
  });

  it("folds a group's rows away the moment it collapses", () => {
    renderDataTable({ grouping: grouping({ collapsedKeys: [1] }) });

    expect(document.querySelector('tr[data-group-key="1"]')?.getAttribute("data-collapsed")).toBe(
      "true",
    );
    expect(screen.queryByText("Alice")).toBeNull();
    expect(screen.getByText("Cara")).toBeTruthy();
  });

  it("toggles a group from its header", () => {
    const onToggleGroup = vi.fn<(key: DataTableGroupKey) => void>();
    renderDataTable({ grouping: grouping({ onToggleGroup }) });

    fireEvent.click(screen.getByRole("button", { name: "Collapse Late" }));
    expect(onToggleGroup).toHaveBeenCalledWith(1);
  });
});

function ExpandingTable({
  withKeyboard = false,
  onEdit,
}: {
  withKeyboard?: boolean;
  onEdit?: (row: TestRow) => void;
}) {
  const [expanded, setExpanded] = React.useState<string | null>(null);
  const [cursor, setCursor] = React.useState<string | null>(null);
  return (
    <DataTable<TestRow>
      columns={testColumns}
      name="test-table"
      queryKey="test"
      graphql={testGraphQLConfig}
      expansion={{
        expandedRowId: expanded,
        onExpandedRowIdChange: setExpanded,
        renderExpandedRow: (row, { collapse }) => (
          <div data-testid="panel">
            Details for {row.original.name}
            <button onClick={collapse}>Close panel</button>
          </div>
        ),
      }}
      keyboard={
        withKeyboard
          ? {
              enabled: true,
              cursorRowId: cursor,
              onCursorRowIdChange: setCursor,
              rowShortcuts: onEdit ? [{ key: "e", run: onEdit }] : undefined,
            }
          : undefined
      }
    />
  );
}

function renderExpanding(withKeyboard = false, onEdit?: (row: TestRow) => void) {
  return render(
    <QueryClientProvider client={createQueryClient()}>
      <NuqsTestingAdapter hasMemory>
        <ExpandingTable withKeyboard={withKeyboard} onEdit={onEdit} />
      </NuqsTestingAdapter>
    </QueryClientProvider>,
  );
}

describe("DataTable expansion", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("opens one row at a time beneath itself when the row is clicked", () => {
    renderExpanding();

    fireEvent.click(screen.getByText("Alice"));
    expect(screen.getByTestId("panel").textContent).toContain("Details for Alice");

    fireEvent.click(screen.getByText("Bob"));
    expect(screen.getAllByTestId("panel")).toHaveLength(1);
    expect(screen.getByTestId("panel").textContent).toContain("Details for Bob");
    const bobRow = document.getElementById("2");
    expect(bobRow?.getAttribute("aria-expanded")).toBe("true");
    expect(bobRow?.nextElementSibling?.getAttribute("data-expanded-for")).toBe("2");
  });

  it("shuts the row from the panel's own collapse and from a second click", () => {
    renderExpanding();
    fireEvent.click(screen.getByText("Alice"));
    fireEvent.click(screen.getByText("Close panel"));
    expect(screen.queryByTestId("panel")).toBeNull();

    fireEvent.click(screen.getByText("Alice"));
    fireEvent.click(screen.getAllByText("Alice")[0]);
    expect(screen.queryByTestId("panel")).toBeNull();
  });

  it("moves a page-level cursor with J and opens the cursor row with Enter", () => {
    renderExpanding(true);

    press("j");
    expect(document.getElementById("1")?.getAttribute("data-cursor")).toBe("true");
    press("j");
    expect(document.getElementById("2")?.getAttribute("data-cursor")).toBe("true");
    press("Enter");
    expect(screen.getByTestId("panel").textContent).toContain("Details for Bob");
    press("j");
    expect(screen.getByTestId("panel").textContent).toContain("Details for Cara");
    press("Escape");
    expect(screen.queryByTestId("panel")).toBeNull();
  });
});

describe("DataTable row shortcuts", () => {
  it("hands the cursor row's data to a row shortcut", () => {
    const onEdit = vi.fn();
    renderExpanding(true, onEdit);
    press("j");
    press("j");
    press("e");
    expect(onEdit).toHaveBeenCalledWith({ id: "2", name: "Bob", stage: 1 });
  });
});

describe("DataTable slots", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("draws an alternate view in place of the rows and pager, with the table's query", () => {
    const render = vi.fn(() => <div data-testid="timeline">timeline</div>);
    renderDataTable({ alternateView: { active: true, render } });

    expect(screen.getByTestId("timeline")).toBeTruthy();
    expect(screen.queryByText("Alice")).toBeNull();
    expect(screen.queryByText("Rows per page")).toBeNull();
    expect(render).toHaveBeenCalledWith({
      queryOptions: expect.objectContaining({ query: "", sort: [] }),
    });
  });

  it("puts toolbar and footer slots where the table's own controls sit", () => {
    renderDataTable({
      toolbar: {
        chips: { items: [{ key: "late", label: "Late", onRemove: vi.fn() }], onClear: vi.fn() },
        trailing: <button>View switch</button>,
        end: <button>Panel toggle</button>,
      },
      footerLeading: <button>Shortcuts</button>,
      pageSizeOptions: [10, 25, 50],
    });

    expect(screen.getByText("Late")).toBeTruthy();
    expect(screen.getByText("View switch")).toBeTruthy();
    expect(screen.getByText("Panel toggle")).toBeTruthy();
    expect(screen.getByText("Shortcuts")).toBeTruthy();
  });

  it("lets a narrow host collapse the built-in toolbar controls", async () => {
    renderDataTable({
      toolbar: { responsive: { label: "label-narrow", secondary: "secondary-narrow" } },
    });

    const sortLabel = await screen.findByText("Sort");
    expect(sortLabel.className).toContain("label-narrow");
    const filterLabel = await screen.findByText("Filter");
    expect(filterLabel.className).toContain("label-narrow");
    const display = await screen.findByRole("button", { name: /Display/ });
    expect(display.closest(".secondary-narrow")).toBeTruthy();
  });

  it("gives an alternate view the table's density", () => {
    renderDataTable({
      initialDensity: "compact",
      alternateView: { active: true, render: () => <div data-testid="timeline" /> },
    });

    expect(screen.getByTestId("timeline").closest('[data-density="compact"]')).toBeTruthy();
  });

  it("is the page body's only child, so a list page bleeds it edge to edge", () => {
    const queryClient = createQueryClient();
    render(
      <QueryClientProvider client={queryClient}>
        <NuqsTestingAdapter hasMemory>
          <div data-slot="page-body" data-testid="page-body">
            <DataTable<TestRow>
              columns={testColumns}
              name="test-table"
              queryKey="test"
              graphql={testGraphQLConfig}
            />
          </div>
        </NuqsTestingAdapter>
      </QueryClientProvider>,
    );

    const pageBody = screen.getByTestId("page-body");
    expect(pageBody.children).toHaveLength(1);
    expect(pageBody.firstElementChild?.getAttribute("data-slot")).toBe("data-table");
    expect(pageBody.querySelectorAll('[data-slot="data-table"]')).toHaveLength(1);
  });

  it("starts at the density it is given", () => {
    renderDataTable({ initialDensity: "compact" });
    expect(document.querySelector('table[data-density="compact"]')).toBeTruthy();
  });
});

describe("DataTable column pinning", () => {
  beforeEach(() => {
    useDataTableQueryMock.mockClear();
  });

  it("pins the columns the table asks for from the first paint", () => {
    renderDataTable({ initialColumnPinning: { left: ["name"], right: [] } });

    const cell = screen.getByText("Alice").closest("td");
    expect(cell?.className).toContain("sticky");
  });

  it("keeps a pinned head above the resize handles of the heads scrolling under it", () => {
    renderDataTable({ initialColumnPinning: { left: ["name"], right: [] } });

    const head = screen.getByText("Name").closest("th");
    expect(head?.className).toContain("z-20");
    expect(head?.className).not.toContain("z-10");
  });
});
