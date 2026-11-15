import { TestRouter } from "@trenova/shared/testing/test-router";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { DataTable } from "../data-table";
import type { CellEditCommitFn } from "@trenova/shared/lib/cell-editing-feature";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { seedDataTableQueries } from "@/test/data-table-queries";

type TestRow = { id: string; name: string; status: string };

const testColumns: ColumnDef<TestRow>[] = [
  {
    accessorKey: "name",
    header: "Name",
    enableCellEditing: true,
    meta: {
      label: "Name",
      apiField: "name",
      filterType: "text",
      editLeading: (row: TestRow) => <span data-testid="name-mark">{row.name.charAt(0)}</span>,
    },
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

// One answer, kept across renders, as React Query keeps an unchanged result.
const useDataTableQueryMock = vi.hoisted(() => {
  const result = {
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
  };
  return vi.fn(() => result);
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

vi.mock("@/hooks/data-table/use-data-table-query", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/data-table/use-data-table-query")>()),
  useDataTableQuery: useDataTableQueryMock,
}));

vi.mock("@/lib/queries/bulk-edit", async () => ({
  bulkEdit: (await import("@/test/data-table-queries")).dataTableQueryMocks.bulkEdit,
}));
vi.mock("@/lib/queries/table-layout", async () => ({
  tableLayout: (await import("@/test/data-table-queries")).dataTableQueryMocks.tableLayout,
}));
vi.mock("@/lib/queries/data-table-insight", async () => ({
  dataTableInsight: (await import("@/test/data-table-queries")).dataTableQueryMocks
    .dataTableInsight,
}));
vi.mock("@/lib/queries/table-configuration", async () => ({
  tableConfiguration: (await import("@/test/data-table-queries")).dataTableQueryMocks
    .tableConfiguration,
}));

vi.mock("@trenova/shared/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T): T => value,
}));

function renderTable(onCellEditCommit?: CellEditCommitFn<TestRow>) {
  const queryClient = seedDataTableQueries(
    new QueryClient({
      defaultOptions: { queries: { retry: false, gcTime: 0 } },
    }),
  );
  return render(
    <QueryClientProvider client={queryClient}>
      <TestRouter>
        <DataTable<TestRow>
          columns={testColumns}
          name="test-table"
          queryKey="test"
          graphql={testGraphQLConfig}
          onCellEditCommit={onCellEditCommit}
        />
      </TestRouter>
    </QueryClientProvider>,
  );
}

describe("DataTable cell editing", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows edit affordances only on opted-in columns when a commit handler is provided", () => {
    renderTable(vi.fn());
    expect(screen.getAllByLabelText("Edit Name")).toHaveLength(2);
    expect(screen.queryByLabelText("Edit Status")).not.toBeInTheDocument();
  });

  it("shows no edit affordances without a commit handler", () => {
    renderTable();
    expect(screen.queryByLabelText("Edit Name")).not.toBeInTheDocument();
  });

  it("commits an edited value with full cell context on Enter", async () => {
    const onCellEditCommit = vi.fn().mockResolvedValue(undefined);
    renderTable(onCellEditCommit);

    fireEvent.click(screen.getAllByLabelText("Edit Name")[0]);
    const input = await screen.findByRole("textbox", { name: "Edit Name" });
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
      expect(screen.queryByRole("textbox", { name: "Edit Name" })).not.toBeInTheDocument();
    });
  });

  it("edits a line of text in the cell itself, filling it, with no field box around it", async () => {
    renderTable(vi.fn());

    fireEvent.click(screen.getAllByLabelText("Edit Name")[0]);
    const input = await screen.findByRole("textbox", { name: "Edit Name" });
    const cell = input.closest('[role="gridcell"]');

    const field = input.closest('[data-slot="cell-inline-editor"]');
    expect(input).not.toHaveAttribute("data-slot", "input");
    expect(input).toHaveClass("border-0", "bg-transparent", "outline-none", "text-sm");
    expect(field).toHaveClass("absolute", "inset-0", "px-(--cell-px)");
    expect(cell).toHaveClass("bg-muted/50");
    expect(cell).not.toHaveClass("overflow-visible");
    // The hidden copy of the text holds the row at its height while the field floats over it.
    expect(cell?.querySelector('[aria-hidden="true"]')?.textContent).toBe("Alice");
  });

  it("keeps what the column draws before the text beside it while the text is edited", async () => {
    renderTable(vi.fn());

    fireEvent.click(screen.getAllByLabelText("Edit Name")[0]);
    const input = await screen.findByRole("textbox", { name: "Edit Name" });
    const field = input.closest('[data-slot="cell-inline-editor"]');
    const mark = screen.getByTestId("name-mark");

    expect(mark.textContent).toBe("A");
    expect(field?.firstElementChild).toBe(mark);
  });

  it("cancels the edit on Escape without committing", async () => {
    const onCellEditCommit = vi.fn();
    renderTable(onCellEditCommit);

    fireEvent.click(screen.getAllByLabelText("Edit Name")[0]);
    const input = await screen.findByRole("textbox", { name: "Edit Name" });
    fireEvent.change(input, { target: { value: "discarded" } });
    fireEvent.keyDown(input, { key: "Escape" });

    await waitFor(() => {
      expect(screen.queryByRole("textbox", { name: "Edit Name" })).not.toBeInTheDocument();
    });
    expect(onCellEditCommit).not.toHaveBeenCalled();
    expect(screen.getByText("Alice")).toBeInTheDocument();
  });

  it("closes without committing when the value is unchanged", async () => {
    const onCellEditCommit = vi.fn();
    renderTable(onCellEditCommit);

    fireEvent.click(screen.getAllByLabelText("Edit Name")[0]);
    const input = await screen.findByRole("textbox", { name: "Edit Name" });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => {
      expect(screen.queryByRole("textbox", { name: "Edit Name" })).not.toBeInTheDocument();
    });
    expect(onCellEditCommit).not.toHaveBeenCalled();
  });

  it("keeps the editor open when the commit handler rejects", async () => {
    const onCellEditCommit = vi.fn().mockRejectedValue(new Error("validation failed"));
    renderTable(onCellEditCommit);

    fireEvent.click(screen.getAllByLabelText("Edit Name")[0]);
    const input = await screen.findByRole("textbox", { name: "Edit Name" });
    fireEvent.change(input, { target: { value: "bad" } });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => {
      expect(onCellEditCommit).toHaveBeenCalledOnce();
    });
    expect(screen.getByRole("textbox", { name: "Edit Name" })).toBeInTheDocument();
  });

  it("only one cell edits at a time", async () => {
    renderTable(vi.fn());

    fireEvent.click(screen.getAllByLabelText("Edit Name")[0]);
    await screen.findByRole("textbox", { name: "Edit Name" });

    fireEvent.click(screen.getAllByLabelText("Edit Name")[0]);
    await waitFor(() => {
      expect(screen.getAllByRole("textbox", { name: "Edit Name" })).toHaveLength(1);
    });
  });
});
