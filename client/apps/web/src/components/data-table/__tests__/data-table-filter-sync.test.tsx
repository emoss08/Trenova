import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { useQueryStates } from "nuqs";
import { DataTable } from "../data-table";
import { parseAsFieldFilters, parseAsFilterGroups } from "@/hooks/data-table/use-data-table-state";
import type { ColumnDef, FieldFilter } from "@trenova/shared/types/data-table";

type TestRow = { id: string; name: string };

const testColumns: ColumnDef<TestRow>[] = [
  {
    accessorKey: "name",
    header: "Name",
    meta: {
      label: "Name",
      apiField: "name",
      filterable: true,
      filterType: "text",
    },
  },
];

const testGraphQLConfig = {
  document:
    "query TestTable($input: DataTableConnectionInput!) { tests(input: $input) { totalCount } }",
  operationName: "TestTable",
  connectionKey: "tests",
};

const useDataTableQueryMock = vi.hoisted(() =>
  vi.fn(() => ({
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
  })),
);

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

vi.mock("@/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T): T => value,
}));

function ExternalFilterWriter({ fieldFilters }: { fieldFilters: FieldFilter[] }) {
  const [, setParams] = useQueryStates({
    fieldFilters: parseAsFieldFilters,
    filterGroups: parseAsFilterGroups,
  });
  return (
    <>
      <button
        type="button"
        data-testid="external-filter-write"
        onClick={() => void setParams({ fieldFilters })}
      >
        external write
      </button>
      <button
        type="button"
        data-testid="external-filter-clear"
        onClick={() => void setParams({ fieldFilters: [] })}
      >
        external clear
      </button>
    </>
  );
}

function renderHarness(externalFieldFilters: FieldFilter[]) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      {/* resetUrlUpdateQueueOnMount runs in the adapter's render body, so any
          re-render wipes updates queued between an external click and their
          timeout-0 flush — disable it to keep external writes reliable. */}
      <NuqsTestingAdapter hasMemory resetUrlUpdateQueueOnMount={false}>
        <ExternalFilterWriter fieldFilters={externalFieldFilters} />
        <DataTable<TestRow>
          columns={testColumns}
          name="test-table"
          queryKey="test"
          graphql={testGraphQLConfig}
        />
      </NuqsTestingAdapter>
    </QueryClientProvider>,
  );
}

describe("DataTable filter state sync (URL → draft)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders filter chips when the URL filter params change after mount", async () => {
    renderHarness([{ field: "name", operator: "contains", value: "Zed" }]);

    expect(screen.queryByText("Zed")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("external-filter-write"));

    expect(await screen.findByText("Zed")).toBeInTheDocument();
    expect(screen.getByText("contains")).toBeInTheDocument();
  });

  it("clears filter chips when the URL filter params are emptied externally", async () => {
    renderHarness([{ field: "name", operator: "contains", value: "Zed" }]);

    fireEvent.click(screen.getByTestId("external-filter-write"));
    expect(await screen.findByText("Zed")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("external-filter-clear"));
    await waitFor(() => {
      expect(screen.queryByText("Zed")).not.toBeInTheDocument();
    });
  });
});
