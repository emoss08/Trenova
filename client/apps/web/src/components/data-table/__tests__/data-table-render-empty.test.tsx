import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { DataTable } from "../data-table";

type TestRow = { id: string; name: string; amount: number };

const testColumns: ColumnDef<TestRow>[] = [
  { accessorKey: "name", header: "Name" },
  {
    accessorKey: "amount",
    header: () => <div className="text-right">Amount</div>,
    meta: { label: "Amount due", filterType: "number" },
  },
];
const testGraphQLConfig = {
  document:
    "query TestTable($input: DataTableConnectionInput!) { tests(input: $input) { totalCount } }",
  operationName: "TestTable",
  connectionKey: "tests",
};

const queryResult = vi.hoisted(() => ({
  current: { data: { results: [] as TestRow[], count: 0 }, isLoading: false, isError: false },
}));

vi.mock("@/hooks/use-permission", () => ({
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
  useDataTableQuery: () => ({ ...queryResult.current, error: null }),
}));
vi.mock("@/lib/queries", () => ({
  queries: {
    tableConfiguration: {
      default: () => ({ queryKey: ["tableConfig-default"], queryFn: () => null }),
      all: () => ({ queryKey: ["tableConfig-all"], queryFn: () => ({ results: [], count: 0 }) }),
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
}));

type Props = Partial<React.ComponentProps<typeof DataTable<TestRow>>>;

function renderTable(props: Props = {}, searchParams = "", onUrlUpdate?: OnUrlUpdateFunction) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  render(
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter hasMemory searchParams={searchParams} onUrlUpdate={onUrlUpdate}>
        <DataTable<TestRow>
          columns={testColumns}
          name="Test record"
          queryKey="test"
          graphql={testGraphQLConfig}
          {...props}
        />
      </NuqsTestingAdapter>
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  queryResult.current = {
    data: { results: [], count: 0 },
    isLoading: false,
    isError: false,
  };
});

describe("DataTable empty state", () => {
  // An empty page is not a table with no rows: the header row and the pager
  // give way to a sketch of the table, drawn from its own visible columns.
  it("replaces the table with a sketch named after the record when nothing is recorded", () => {
    renderTable();

    expect(screen.getByRole("heading", { name: "No test records yet" })).toBeInTheDocument();
    expect(screen.queryByRole("columnheader")).not.toBeInTheDocument();
    expect(screen.queryByText(/Rows per page/)).not.toBeInTheDocument();
    expect(screen.queryByText("No data available")).not.toBeInTheDocument();
  });

  it("offers to add the first record when the table can create one", async () => {
    const onAddRecord = vi.fn();
    const user = userEvent.setup();
    renderTable({ onAddRecord });

    await user.click(screen.getByRole("button", { name: "Add Test record" }));
    expect(onAddRecord).toHaveBeenCalledTimes(1);
  });

  it("does not offer to add when creation is switched off", () => {
    renderTable({ onAddRecord: vi.fn(), enableCreateAction: false });

    expect(screen.queryByRole("button", { name: "Add Test record" })).not.toBeInTheDocument();
  });

  it("offers to clear a search that emptied the table instead of adding", async () => {
    const onUrlUpdate = vi.fn();
    const user = userEvent.setup();
    renderTable({ onAddRecord: vi.fn() }, "?query=alice", onUrlUpdate);

    expect(screen.getByRole("heading", { name: "Nothing matches" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add Test record" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));

    const last = onUrlUpdate.mock.calls.at(-1)?.[0] as { searchParams: URLSearchParams };
    expect(last.searchParams.get("query")).toBeNull();
  });

  it("lets a table draw its own empty state and tells it whether filters are on", () => {
    renderTable({
      renderEmptyState: ({ hasActiveFilters }) => (
        <p>{hasActiveFilters ? "custom filtered" : "custom empty"}</p>
      ),
    });

    expect(screen.getByText("custom empty")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "No test records yet" })).not.toBeInTheDocument();
  });

  it("keeps the table once there are rows", () => {
    queryResult.current = {
      data: { results: [{ id: "1", name: "Alice", amount: 5 }], count: 1 },
      isLoading: false,
      isError: false,
    };
    renderTable();

    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "No test records yet" })).not.toBeInTheDocument();
  });
});
