import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import React from "react";
import { DataTable } from "../data-table";
import { ControlsProvider, useControls } from "@/contexts/control-context";
import { DataTableProvider, useDataTable } from "@/contexts/data-table-context";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useTable } from "@tanstack/react-table";
import { dataTableFeatures } from "@trenova/shared/lib/table-features";
import { DataTablePagination } from "../_components/data-table-pagination";

type TestRow = { id: string; name: string };

const testColumns: ColumnDef<TestRow>[] = [{ accessorKey: "name", header: "Name" }];
const testGraphQLConfig = {
  document:
    "query TestTable($input: DataTableConnectionInput!) { tests(input: $input) { totalCount } }",
  operationName: "TestTable",
  connectionKey: "tests",
};
const { useDataTableQueryMock, defaultQueryResult } = vi.hoisted(() => {
  const defaultQueryResult = {
    data: {
      results: [
        { id: "1", name: "Alice" },
        { id: "2", name: "Bob" },
      ],
      count: 2,
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

// ── DataTable integration tests ────────────────────────────────────────

describe("DataTable", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders row data from the query", () => {
    renderDataTable();
    expect(screen.getAllByText("Alice").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Bob").length).toBeGreaterThanOrEqual(1);
  });

  it("renders column headers", () => {
    renderDataTable();
    expect(screen.getAllByText("Name").length).toBeGreaterThanOrEqual(1);
  });

  it("renders without a TablePanel", () => {
    renderDataTable({ TablePanel: undefined });
    expect(screen.getAllByText("Alice").length).toBeGreaterThanOrEqual(1);
  });

  it("passes the required GraphQL config through to the query hook", () => {
    const graphql = {
      document:
        "query TestTable($input: DataTableConnectionInput!) { tests(input: $input) { totalCount } }",
      operationName: "TestTable",
      connectionKey: "tests",
      extraVariables: { includeDetails: true },
    };

    renderDataTable({ graphql });

    const graphqlCall = (useDataTableQueryMock.mock.calls as unknown[][]).find(
      (call) => call[1] === graphql,
    );
    expect(graphqlCall).toEqual([
      "test",
      graphql,
      { pageIndex: 0, pageSize: 10 },
      expect.objectContaining({
        query: "",
        fieldFilters: [],
        filterGroups: [],
        sort: [],
      }),
      true,
    ]);
  });

  it("survives multiple parent re-renders without crashing", () => {
    const queryClient = createQueryClient();

    function App() {
      const [tick, setTick] = React.useState(0);
      return (
        <>
          <button data-testid="rerender" onClick={() => setTick((t) => t + 1)}>
            tick {tick}
          </button>
          <DataTable<TestRow>
            columns={testColumns}
            name="rerender-test"
            queryKey="rerender"
            graphql={testGraphQLConfig}
          />
        </>
      );
    }

    render(
      <QueryClientProvider client={queryClient}>
        <NuqsTestingAdapter hasMemory>
          <App />
        </NuqsTestingAdapter>
      </QueryClientProvider>,
    );

    expect(screen.getAllByText("Alice").length).toBeGreaterThanOrEqual(1);

    const rerenderButton = screen.getByTestId("rerender");
    act(() => {
      for (let i = 0; i < 3; i++) {
        fireEvent.click(rerenderButton);
      }
    });

    expect(screen.getAllByText("Alice").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Bob").length).toBeGreaterThanOrEqual(1);
  });
});

// ── Cursor pagination total count ──────────────────────────────────────

type CursorPage = {
  results: TestRow[];
  totalCount: number | null;
  endCursor: string | null;
};

function cursorQueryResult(page: CursorPage) {
  return {
    data: {
      results: page.results,
      count: page.totalCount ?? page.results.length,
      next: null,
      prev: null,
      pageInfo: {
        mode: "cursor" as const,
        hasNextPage: page.endCursor != null,
        endCursor: page.endCursor,
        totalCount: page.totalCount,
      },
    },
    isLoading: false,
    isError: false,
    error: null,
  };
}

function mockCursorPages(pages: Record<string, CursorPage>) {
  useDataTableQueryMock.mockImplementation((...args: unknown[]) => {
    const options = args[3] as { cursor?: string } | undefined;
    const page = pages[options?.cursor ?? ""];
    if (!page) {
      throw new Error(`Unexpected cursor ${String(options?.cursor)}`);
    }
    return cursorQueryResult(page);
  });
}

const firstPage: CursorPage = {
  results: [
    { id: "1", name: "Alice" },
    { id: "2", name: "Bob" },
  ],
  totalCount: 30,
  endCursor: "cursor-page-2",
};

const secondPageWithoutTotal: CursorPage = {
  results: [
    { id: "3", name: "Carol" },
    { id: "4", name: "Dave" },
  ],
  totalCount: null,
  endCursor: "cursor-page-3",
};

describe("DataTable cursor total count", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    useDataTableQueryMock.mockImplementation(() => defaultQueryResult);
  });

  it("keeps the total from the first page when a later page omits it", async () => {
    mockCursorPages({ "": firstPage, "cursor-page-2": secondPageWithoutTotal });

    const { container } = renderDataTable();

    expect(container).toHaveTextContent("Showing 1 to 2 of 30 results");
    expect(container).toHaveTextContent(/Page\s*1\s*of\s*3/);

    act(() => {
      fireEvent.click(screen.getByLabelText("Go to next page"));
    });

    expect(await screen.findAllByText("Carol")).not.toHaveLength(0);
    expect(container).toHaveTextContent("Showing 11 to 12 of 30 results");
    expect(container).toHaveTextContent(/Page\s*2\s*of\s*3/);
    expect(screen.getByLabelText("Go to next page")).not.toBeDisabled();
  });

  it("offers select-all-matching on a later page using the remembered total", async () => {
    mockCursorPages({ "": firstPage, "cursor-page-2": secondPageWithoutTotal });

    renderDataTable({ enableRowSelection: true });

    act(() => {
      fireEvent.click(screen.getByLabelText("Go to next page"));
    });
    expect(await screen.findAllByText("Carol")).not.toHaveLength(0);

    act(() => {
      fireEvent.click(screen.getByLabelText("Select all"));
    });

    expect(screen.getByText("Select all 30 matching")).toBeInTheDocument();
  });

  it("keeps the remembered total when returning to a first page that omits it", async () => {
    mockCursorPages({ "": firstPage, "cursor-page-2": secondPageWithoutTotal });

    const { container } = renderDataTable();

    act(() => {
      fireEvent.click(screen.getByLabelText("Go to next page"));
    });
    expect(await screen.findAllByText("Carol")).not.toHaveLength(0);

    mockCursorPages({
      "": { ...firstPage, totalCount: null },
      "cursor-page-2": secondPageWithoutTotal,
    });

    act(() => {
      fireEvent.click(screen.getByLabelText("Go to previous page"));
    });

    expect(await screen.findAllByText("Alice")).not.toHaveLength(0);
    expect(container).toHaveTextContent("Showing 1 to 2 of 30 results");
  });

  it("forgets the remembered total when the query scope changes", async () => {
    mockCursorPages({ "": firstPage, "cursor-page-2": secondPageWithoutTotal });
    const queryClient = createQueryClient();
    const tree = (graphql: typeof testGraphQLConfig) => (
      <QueryClientProvider client={queryClient}>
        <NuqsTestingAdapter hasMemory>
          <DataTable<TestRow>
            columns={testColumns}
            name="test-table"
            queryKey="test"
            graphql={graphql}
          />
        </NuqsTestingAdapter>
      </QueryClientProvider>
    );

    const { container, rerender } = render(tree(testGraphQLConfig));

    act(() => {
      fireEvent.click(screen.getByLabelText("Go to next page"));
    });
    expect(await screen.findAllByText("Carol")).not.toHaveLength(0);
    expect(container).toHaveTextContent("of 30");

    mockCursorPages({
      "": { results: [{ id: "9", name: "Zed" }], totalCount: null, endCursor: null },
      "cursor-page-2": secondPageWithoutTotal,
    });

    rerender(tree({ ...testGraphQLConfig, operationName: "OtherTestTable" }));

    expect(await screen.findAllByText("Zed")).not.toHaveLength(0);
    expect(container).toHaveTextContent("Showing 1 to 1 results");
    expect(container).not.toHaveTextContent("of 30");
  });
});

// ── DataTableProvider context memoization tests ────────────────────────

describe("DataTableProvider memoization", () => {
  it("delivers updated context when isLoading changes", () => {
    function Consumer() {
      const ctx = useDataTable();
      return <span data-testid="loading-state">{ctx.isLoading ? "loading" : "ready"}</span>;
    }

    function TestApp() {
      const [loading, setLoading] = React.useState(true);
      const table = useTable({
        features: dataTableFeatures,
        data: [] as TestRow[],
        columns: testColumns,
        getRowId: (row) => row.id,
      });

      return (
        <>
          <button data-testid="toggle-loading" onClick={() => setLoading(false)}>
            finish
          </button>
          <DataTableProvider table={table} columns={testColumns} isLoading={loading}>
            <Consumer />
          </DataTableProvider>
        </>
      );
    }

    render(<TestApp />);

    expect(screen.getByTestId("loading-state")).toHaveTextContent("loading");

    act(() => {
      fireEvent.click(screen.getByTestId("toggle-loading"));
    });

    expect(screen.getByTestId("loading-state")).toHaveTextContent("ready");
  });

  it("context value is referentially stable when all props are stable", () => {
    const contextValues: unknown[] = [];

    function ValueCapture() {
      const ctx = useDataTable();
      contextValues.push(ctx);
      return null;
    }

    const stableTable = {} as any;
    const stableCallbacks = {
      openPanelCreate: () => {},
      openPanelEdit: () => {},
      closePanel: () => {},
    };

    function Parent() {
      const [tick, setTick] = React.useState(0);
      return (
        <>
          <button data-testid="tick-stable" onClick={() => setTick((t) => t + 1)}>
            tick {tick}
          </button>
          <DataTableProvider
            table={stableTable}
            columns={testColumns}
            isLoading={false}
            {...stableCallbacks}
          >
            <ValueCapture />
          </DataTableProvider>
        </>
      );
    }

    render(<Parent />);

    const firstValue = contextValues[0];
    expect(firstValue).toBeDefined();

    act(() => {
      screen.getByTestId("tick-stable").click();
    });
    act(() => {
      screen.getByTestId("tick-stable").click();
    });

    expect(contextValues.length).toBeGreaterThanOrEqual(3);
    for (let i = 1; i < contextValues.length; i++) {
      expect(contextValues[i]).toBe(firstValue);
    }
  });

  it("context value changes when isLoading prop changes", () => {
    const contextValues: unknown[] = [];

    function ValueCapture() {
      const ctx = useDataTable();
      contextValues.push(ctx);
      return null;
    }

    const stableTable = {} as any;

    function Parent() {
      const [loading, setLoading] = React.useState(true);
      return (
        <>
          <button data-testid="toggle" onClick={() => setLoading(false)}>
            toggle
          </button>
          <DataTableProvider table={stableTable} columns={testColumns} isLoading={loading}>
            <ValueCapture />
          </DataTableProvider>
        </>
      );
    }

    render(<Parent />);

    const firstValue = contextValues[0];
    expect((firstValue as any).isLoading).toBe(true);

    act(() => {
      screen.getByTestId("toggle").click();
    });

    const lastValue = contextValues[contextValues.length - 1];
    expect((lastValue as any).isLoading).toBe(false);
    expect(lastValue).not.toBe(firstValue);
  });

  it("context value updates when useTable is used (table ref changes each render)", () => {
    const contextValues: unknown[] = [];

    function ValueCapture() {
      const ctx = useDataTable();
      contextValues.push(ctx);
      return null;
    }

    function WithRealTable() {
      const [tick, setTick] = React.useState(0);
      const table = useTable({
        features: dataTableFeatures,
        data: [{ id: "1", name: "Alice" }] as TestRow[],
        columns: testColumns,
        getRowId: (row) => row.id,
      });

      return (
        <>
          <button data-testid="tick-rt" onClick={() => setTick((t) => t + 1)}>
            tick {tick}
          </button>
          <DataTableProvider table={table} columns={testColumns} isLoading={false}>
            <ValueCapture />
          </DataTableProvider>
        </>
      );
    }

    render(<WithRealTable />);

    act(() => {
      fireEvent.click(screen.getByTestId("tick-rt"));
    });

    // useTable creates a new object each render, so the context
    // value WILL change — this is expected and documented in the plan.
    // The useMemo still protects against changes from other parent state
    // that doesn't affect the provider's props.
    expect(contextValues.length).toBeGreaterThanOrEqual(2);
    expect((contextValues[0] as any).isLoading).toBe(false);
    expect((contextValues[contextValues.length - 1] as any).isLoading).toBe(false);
  });
});

describe("DataTablePagination", () => {
  it("renders cursor pagination without total count or last-page controls", () => {
    function CursorPaginationHarness() {
      const table = useTable({
        features: dataTableFeatures,
        data: [
          { id: "11", name: "Cursor A" },
          { id: "12", name: "Cursor B" },
        ] as TestRow[],
        columns: testColumns,
        manualPagination: true,
        pageCount: 3,
        rowCount: 30,
        state: {
          pagination: {
            pageIndex: 1,
            pageSize: 10,
          },
        },
      });

      return (
        <DataTablePagination table={table} mode="cursor" hasNextPage currentPageRowCount={2} />
      );
    }

    const { container } = render(<CursorPaginationHarness />);
    const view = within(container);

    expect(container).toHaveTextContent("Showing 11 to 12 results");
    expect(view.queryByLabelText("Go to first page")).not.toBeInTheDocument();
    expect(view.queryByLabelText("Go to last page")).not.toBeInTheDocument();
    expect(view.getByLabelText("Go to next page")).not.toBeDisabled();
    expect(container).not.toHaveTextContent("of 30");
  });
});

// ── ControlsProvider memoization ───────────────────────────────────────

describe("ControlsProvider memoization", () => {
  // Explicit timeout: the suite's parallel import phase runs to two minutes
  // on a loaded CI runner, and this file's worker can start this test with
  // most of the default five seconds already spent.
  it("context value is referentially stable when open does not change", { timeout: 15_000 }, () => {
    const capturedValues: unknown[] = [];

    function Capture() {
      const ctx = useControls();
      capturedValues.push(ctx);
      return <span>{ctx.open ? "open" : "closed"}</span>;
    }

    function Parent() {
      const [tick, setTick] = React.useState(0);
      return (
        <>
          <button data-testid="tick-ctrl" onClick={() => setTick((t) => t + 1)}>
            tick {tick}
          </button>
          <ControlsProvider>
            <Capture />
          </ControlsProvider>
        </>
      );
    }

    render(<Parent />);

    expect(capturedValues.length).toBeGreaterThanOrEqual(1);
    const first = capturedValues[0];

    fireEvent.click(screen.getByTestId("tick-ctrl"));
    fireEvent.click(screen.getByTestId("tick-ctrl"));

    expect(capturedValues.length).toBeGreaterThanOrEqual(2);
    for (let i = 1; i < capturedValues.length; i++) {
      expect(capturedValues[i]).toBe(first);
    }
  });
});
