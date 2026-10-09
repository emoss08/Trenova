import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ColumnDef, DataTableExpansion, RowAction } from "@trenova/shared/types/data-table";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import React from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { DataTable } from "../data-table";
import { seedDataTableQueries } from "@/test/data-table-queries";

/**
 * What one interaction costs the table, in cell renders. A board of a hundred
 * rows draws fourteen hundred cells; an interaction that touches one row must
 * not redraw the other ninety-nine.
 */

type TestRow = { id: string; name: string; stage: number };

const ROW_COUNT = 100;
const COLUMN_COUNT = 14;

const {
  countCellRender,
  countHeaderRender,
  queryResult,
  useDataTableQueryMock,
} = vi.hoisted(() => {
  const results = Array.from({ length: 100 }, (_, index) => ({
    id: `r${index}`,
    name: `Row ${index}`,
    stage: index % 3,
  }));
  const queryResult = {
    data: { results, count: results.length },
    isLoading: false,
    isError: false,
    error: null,
  };
  return {
    countCellRender: vi.fn(),
    countHeaderRender: vi.fn(),
    queryResult,
    useDataTableQueryMock: vi.fn((..._args: unknown[]): unknown => queryResult),
  };
});

vi.mock("@/hooks/use-permission", () => ({
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

vi.mock("@/hooks/data-table/use-data-table-query", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/data-table/use-data-table-query")>()),
  useDataTableQuery: useDataTableQueryMock,
}));

vi.mock("@/lib/queries", async () => ({
  queries: {
    ...(await import("@/test/data-table-queries")).dataTableQueryMocks,
  },
}));

vi.mock("@trenova/shared/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T): T => value,
}));

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

afterEach(cleanup);

function CountingCell({ value }: { value: string }) {
  countCellRender();
  return <span>{value}</span>;
}

function CountingHeader({ label }: { label: string }) {
  countHeaderRender();
  return <span>{label}</span>;
}

// The first column draws its own head, so a render of the header row is counted.
const columns: ColumnDef<TestRow>[] = Array.from({ length: COLUMN_COUNT }, (_, index) => ({
  id: `c${index}`,
  header: index === 0 ? () => <CountingHeader label="Column 0" /> : `Column ${index}`,
  meta: index === 0 ? { sortable: false } : undefined,
  cell: ({ row }) => <CountingCell value={`${row.original.name}:${index}`} />,
}));

const graphql = {
  document: "query T($input: DataTableConnectionInput!) { tests(input: $input) { totalCount } }",
  operationName: "T",
  connectionKey: "tests",
};

const { slowAction } = vi.hoisted(() => {
  let settle: () => void = () => {};
  return {
    slowAction: {
      run: () =>
        new Promise<void>((resolve) => {
          settle = resolve;
        }),
      settle: () => settle(),
    },
  };
});

const rowActions: RowAction<TestRow>[] = [
  { id: "open", label: "Open", onClick: vi.fn() },
  { id: "bill", label: "Bill", onClick: () => slowAction.run() },
];

const renderExpandedRow: DataTableExpansion<TestRow>["renderExpandedRow"] = (row) => (
  <div data-testid="panel">Details for {row.original.name}</div>
);

const parent = { rerender: () => {} };

function Board() {
  const [, setTick] = React.useState(0);
  React.useEffect(() => {
    parent.rerender = () => setTick((tick) => tick + 1);
  }, []);
  const [expanded, setExpanded] = React.useState<string | null>(null);
  const [cursor, setCursor] = React.useState<string | null>(null);

  const expansion = React.useMemo(
    () => ({ expandedRowId: expanded, onExpandedRowIdChange: setExpanded, renderExpandedRow }),
    [expanded],
  );
  const keyboard = React.useMemo(
    () => ({ enabled: true, cursorRowId: cursor, onCursorRowIdChange: setCursor }),
    [cursor],
  );

  return (
    <DataTable<TestRow>
      columns={columns}
      name="render-cost"
      queryKey="render-cost"
      graphql={graphql}
      enableRowSelection
      contextMenuActions={rowActions}
      expansion={expansion}
      keyboard={keyboard}
      pageSizeOptions={[25, 50, 100]}
    />
  );
}

function renderBoard() {
  const queryClient = seedDataTableQueries(new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  }));
  render(
    <QueryClientProvider client={queryClient}>
      <NuqsTestingAdapter hasMemory searchParams="?pageSize=100">
        <Board />
      </NuqsTestingAdapter>
    </QueryClientProvider>,
  );
}

function press(key: string) {
  act(() => {
    window.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true }));
  });
}

function measure(interaction: () => void) {
  countCellRender.mockClear();
  interaction();
  return countCellRender.mock.calls.length;
}

describe("DataTable render cost", () => {
  beforeEach(() => {
    useDataTableQueryMock.mockImplementation(() => queryResult);
    renderBoard();
  });

  it("draws every cell once on the first paint", () => {
    expect(screen.getByText("Row 99:0")).toBeTruthy();
  });

  it("redraws no cells when the page above it re-renders", () => {
    expect(measure(() => act(() => parent.rerender()))).toBe(0);
  });

  it("redraws only the two rows a cursor step touches", () => {
    press("j");
    expect(measure(() => press("j"))).toBeLessThanOrEqual(2 * COLUMN_COUNT);
  });

  it("redraws only the row that opens", () => {
    expect(measure(() => fireEvent.click(screen.getByText("Row 40:0")))).toBeLessThanOrEqual(
      COLUMN_COUNT,
    );
    expect(screen.getByTestId("panel").textContent).toContain("Row 40");
  });

  it("redraws only the row whose checkbox is ticked", () => {
    const checkboxes = screen.getAllByRole("checkbox", { name: "Select row" });
    expect(measure(() => fireEvent.click(checkboxes[10]))).toBeLessThanOrEqual(COLUMN_COUNT);
    expect(checkboxes[10].getAttribute("aria-checked")).toBe("true");
  });

  it("ticks rows without redrawing the header row around them", () => {
    const checkboxes = screen.getAllByRole("checkbox", { name: "Select row" });
    countHeaderRender.mockClear();
    fireEvent.click(checkboxes[3]);
    fireEvent.click(checkboxes[4]);
    fireEvent.click(checkboxes[3]);

    expect(countHeaderRender).not.toHaveBeenCalled();
    expect(checkboxes[4].getAttribute("aria-checked")).toBe("true");
    expect(checkboxes[3].getAttribute("aria-checked")).toBe("false");
  });

  it("follows a resize drag without rendering the header row or the cells", async () => {
    const tableElement = document.querySelector<HTMLTableElement>('[data-slot="table"]')!;
    const width = () => tableElement.style.getPropertyValue("--col-c1-size");
    const handle = screen.getByRole("separator", { name: "Resize c1 column" });

    fireEvent.mouseDown(handle, { clientX: 200 });
    countHeaderRender.mockClear();
    countCellRender.mockClear();

    for (const clientX of [230, 260, 290]) {
      const before = width();
      fireEvent.mouseMove(document, { clientX });
      await waitFor(() => expect(width()).not.toBe(before));
    }

    expect(countHeaderRender).not.toHaveBeenCalled();
    expect(countCellRender).not.toHaveBeenCalled();
    act(() => {
      fireEvent.mouseUp(document, { clientX: 290 });
    });
  });

  it("still redraws every row when the data itself changes", () => {
    const next = {
      ...queryResult,
      data: {
        ...queryResult.data,
        results: queryResult.data.results.map((row) => ({ ...row, name: `${row.name}*` })),
      },
    };
    useDataTableQueryMock.mockImplementation(() => next);
    expect(measure(() => act(() => parent.rerender()))).toBe(ROW_COUNT * COLUMN_COUNT);
    expect(screen.getByText("Row 0*:0")).toBeTruthy();
  });

  it("redraws no cells while a row action runs and when it settles", async () => {
    fireEvent.contextMenu(screen.getByText("Row 7:0"));
    const bill = await screen.findByRole("menuitem", { name: "Bill" });

    expect(measure(() => fireEvent.click(bill))).toBe(0);
    fireEvent.contextMenu(screen.getByText("Row 7:0"));
    expect(
      (await screen.findByRole("menuitem", { name: "Bill" })).getAttribute("aria-disabled"),
    ).toBe("true");

    countCellRender.mockClear();
    await act(async () => {
      slowAction.settle();
      await Promise.resolve();
    });
    expect(countCellRender).not.toHaveBeenCalled();
  });
});
