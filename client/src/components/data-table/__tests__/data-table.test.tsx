import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import React from "react";
import { DataTable } from "../data-table";
import { DataTableProvider, useDataTable } from "@/contexts/data-table-context";
import type { ColumnDef } from "@tanstack/react-table";
import { useReactTable, getCoreRowModel } from "@tanstack/react-table";

type TestRow = { id: string; name: string };

const testColumns: ColumnDef<TestRow>[] = [{ accessorKey: "name", header: "Name" }];

vi.mock("@/hooks/use-permission", () => ({
  usePermissions: () => ({
    canRead: true,
    canCreate: true,
    canUpdate: true,
    canExport: true,
    canImport: true,
    isLoading: false,
    isPlatformAdmin: false,
    isOrgAdmin: false,
  }),
}));

vi.mock("@/hooks/data-table/use-data-table-query", () => ({
  useDataTableQuery: () => ({
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
  }),
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

vi.mock("@/lib/api", () => ({
  api: { get: vi.fn().mockResolvedValue(null) },
}));

vi.mock("@/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T): T => value,
}));

vi.mock("@/lib/data-table", () => ({
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
          link="/trailers/"
          queryKey="test"
          exportModelName="Test"
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

  it("survives multiple parent re-renders without crashing", async () => {
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
            link="/trailers/"
            queryKey="rerender"
            exportModelName="Test"
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

    const user = userEvent.setup();
    for (let i = 0; i < 3; i++) {
      await user.click(screen.getByTestId("rerender"));
    }

    await waitFor(() => {
      expect(screen.getAllByText("Alice").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("Bob").length).toBeGreaterThanOrEqual(1);
    });
  });
});

// ── DataTableProvider context memoization tests ────────────────────────

describe("DataTableProvider memoization", () => {
  it("delivers updated context when isLoading changes", async () => {
    function Consumer() {
      const ctx = useDataTable();
      return <span data-testid="loading-state">{ctx.isLoading ? "loading" : "ready"}</span>;
    }

    function TestApp() {
      const [loading, setLoading] = React.useState(true);
      const table = useReactTable({
        data: [] as TestRow[],
        columns: testColumns,
        getCoreRowModel: getCoreRowModel(),
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

    const user = userEvent.setup();
    await user.click(screen.getByTestId("toggle-loading"));

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

  it("context value updates when useReactTable is used (table ref changes each render)", async () => {
    const contextValues: unknown[] = [];

    function ValueCapture() {
      const ctx = useDataTable();
      contextValues.push(ctx);
      return null;
    }

    function WithRealTable() {
      const [tick, setTick] = React.useState(0);
      const table = useReactTable({
        data: [{ id: "1", name: "Alice" }] as TestRow[],
        columns: testColumns,
        getCoreRowModel: getCoreRowModel(),
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

    const user = userEvent.setup();
    await user.click(screen.getByTestId("tick-rt"));

    // useReactTable creates a new object each render, so the context
    // value WILL change — this is expected and documented in the plan.
    // The useMemo still protects against changes from other parent state
    // that doesn't affect the provider's props.
    expect(contextValues.length).toBeGreaterThanOrEqual(2);
    expect((contextValues[0] as any).isLoading).toBe(false);
    expect((contextValues[contextValues.length - 1] as any).isLoading).toBe(false);
  });
});

// ── ControlsProvider memoization ───────────────────────────────────────

describe("ControlsProvider memoization", () => {
  it("context value is referentially stable when open does not change", async () => {
    const { ControlsProvider, useControls } = await import("@/contexts/control-context");
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

    const user = userEvent.setup();
    await user.click(screen.getByTestId("tick-ctrl"));
    await user.click(screen.getByTestId("tick-ctrl"));

    expect(capturedValues.length).toBeGreaterThanOrEqual(2);
    for (let i = 1; i < capturedValues.length; i++) {
      expect(capturedValues[i]).toBe(first);
    }
  });
});
