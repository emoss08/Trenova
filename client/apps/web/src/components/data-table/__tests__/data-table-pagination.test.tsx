import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { Table } from "@trenova/shared/types/data-table";
import { DataTablePagination } from "../_components/data-table-pagination";

type Row = { id: string };

function offsetTable(page: {
  pageIndex: number;
  pageSize: number;
  rowCount: number;
}): Table<Row> {
  return {
    state: { pagination: { pageIndex: page.pageIndex, pageSize: page.pageSize } },
    getPageCount: () => Math.max(1, Math.ceil(page.rowCount / page.pageSize)),
    getRowCount: () => page.rowCount,
    getRowModel: () => ({ rows: [] }),
  } as unknown as Table<Row>;
}

describe("DataTablePagination over offset pages", () => {
  afterEach(() => {
    cleanup();
  });

  it("counts the pages of a counted total and offers the last one", () => {
    const { container } = render(
      <DataTablePagination
        table={offsetTable({ pageIndex: 1, pageSize: 25, rowCount: 60 })}
        currentPageRowCount={25}
        totalCountExact
      />,
    );

    expect(container).toHaveTextContent("Showing 26 to 50 of 60 results");
    expect(container).toHaveTextContent(/Page\s*2\s*of\s*3/);
    expect(screen.getByLabelText("Go to next page")).not.toBeDisabled();
    expect(screen.getByLabelText("Go to last page")).toBeInTheDocument();
  });

  it("pages an estimated total by full pages and never jumps to a last page", () => {
    const { container } = render(
      <DataTablePagination
        table={offsetTable({ pageIndex: 2, pageSize: 25, rowCount: 1_250_000 })}
        currentPageRowCount={25}
        totalCountExact={false}
      />,
    );

    expect(container).toHaveTextContent("Showing 51 to 75 of about 1.3M results");
    expect(container).toHaveTextContent(/Page\s*3/);
    expect(container).not.toHaveTextContent(/Page\s*3\s*of/);
    expect(screen.getByLabelText("Go to next page")).not.toBeDisabled();
    expect(screen.queryByLabelText("Go to last page")).not.toBeInTheDocument();
  });

  it("ends an estimated list at a short page", () => {
    render(
      <DataTablePagination
        table={offsetTable({ pageIndex: 4, pageSize: 25, rowCount: 1_250_000 })}
        currentPageRowCount={7}
        totalCountExact={false}
      />,
    );

    expect(screen.getByLabelText("Go to next page")).toBeDisabled();
  });
});
