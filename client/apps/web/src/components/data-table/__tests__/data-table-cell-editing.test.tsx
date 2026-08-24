import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { DataTable } from "../data-table";
import type { CellEditCommitFn } from "@trenova/shared/lib/cell-editing-feature";
import type { ColumnDef } from "@trenova/shared/types/data-table";

type TestRow = { id: string; name: string; status: string };

const testColumns: ColumnDef<TestRow>[] = [
  {
    accessorKey: "name",
    header: "Name",
    enableCellEditing: true,
    meta: { label: "Name", apiField: "name", filterType: "text" },
  },
  {
    accessorKey: "status",
    header: "Status",
    meta: { label: "Status", apiField: "status", filterType: "text" },
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
        { id: "1", name: "Alice", status: "active" },
        { id: "2", name: "Bob", status: "inactive" },
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

function renderTable(onCellEditCommit?: CellEditCommitFn<TestRow>) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <NuqsTestingAdapter hasMemory resetUrlUpdateQueueOnMount={false}>
        <DataTable<TestRow>
          columns={testColumns}
          name="test-table"
          queryKey="test"
          graphql={testGraphQLConfig}
          onCellEditCommit={onCellEditCommit}
        />
      </NuqsTestingAdapter>
    </QueryClientProvider>,
  );
}

describe("DataTable cell editing", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows edit affordances only on opted-in columns when a commit handler is provided", () => {
    renderTable(vi.fn());
    expect(screen.getAllByLabelText("Edit name")).toHaveLength(2);
    expect(screen.queryByLabelText("Edit status")).not.toBeInTheDocument();
  });

  it("shows no edit affordances without a commit handler", () => {
    renderTable();
    expect(screen.queryByLabelText("Edit name")).not.toBeInTheDocument();
  });

  it("commits an edited value with full cell context on Enter", async () => {
    const onCellEditCommit = vi.fn().mockResolvedValue(undefined);
    renderTable(onCellEditCommit);

    fireEvent.click(screen.getAllByLabelText("Edit name")[0]);
    const input = await screen.findByRole("textbox", { name: "Edit name" });
    expect(input).toHaveValue("Alice");

    fireEvent.change(input, { target: { value: "Alice Prime" } });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => {
      expect(onCellEditCommit).toHaveBeenCalledExactlyOnceWith({
        rowId: "1",
        columnId: "name",
        value: "Alice Prime",
        previousValue: "Alice",
        row: { id: "1", name: "Alice", status: "active" },
      });
    });
    await waitFor(() => {
      expect(screen.queryByRole("textbox", { name: "Edit name" })).not.toBeInTheDocument();
    });
  });

  it("cancels the edit on Escape without committing", async () => {
    const onCellEditCommit = vi.fn();
    renderTable(onCellEditCommit);

    fireEvent.click(screen.getAllByLabelText("Edit name")[0]);
    const input = await screen.findByRole("textbox", { name: "Edit name" });
    fireEvent.change(input, { target: { value: "discarded" } });
    fireEvent.keyDown(input, { key: "Escape" });

    await waitFor(() => {
      expect(screen.queryByRole("textbox", { name: "Edit name" })).not.toBeInTheDocument();
    });
    expect(onCellEditCommit).not.toHaveBeenCalled();
    expect(screen.getByText("Alice")).toBeInTheDocument();
  });

  it("closes without committing when the value is unchanged", async () => {
    const onCellEditCommit = vi.fn();
    renderTable(onCellEditCommit);

    fireEvent.click(screen.getAllByLabelText("Edit name")[0]);
    const input = await screen.findByRole("textbox", { name: "Edit name" });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => {
      expect(screen.queryByRole("textbox", { name: "Edit name" })).not.toBeInTheDocument();
    });
    expect(onCellEditCommit).not.toHaveBeenCalled();
  });

  it("keeps the editor open when the commit handler rejects", async () => {
    const onCellEditCommit = vi.fn().mockRejectedValue(new Error("validation failed"));
    renderTable(onCellEditCommit);

    fireEvent.click(screen.getAllByLabelText("Edit name")[0]);
    const input = await screen.findByRole("textbox", { name: "Edit name" });
    fireEvent.change(input, { target: { value: "bad" } });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => {
      expect(onCellEditCommit).toHaveBeenCalledOnce();
    });
    expect(screen.getByRole("textbox", { name: "Edit name" })).toBeInTheDocument();
  });

  it("only one cell edits at a time", async () => {
    renderTable(vi.fn());

    fireEvent.click(screen.getAllByLabelText("Edit name")[0]);
    await screen.findByRole("textbox", { name: "Edit name" });

    fireEvent.click(screen.getAllByLabelText("Edit name")[0]);
    await waitFor(() => {
      expect(screen.getAllByRole("textbox", { name: "Edit name" })).toHaveLength(1);
    });
  });
});
