import { render, screen } from "@testing-library/react";
import { useTable } from "@tanstack/react-table";
import { dataTableFeatures } from "@trenova/shared/lib/table-features";
import type { RowSelectionState } from "@tanstack/react-table";
import { describe, expect, it } from "vitest";
import { flexRender } from "@tanstack/react-table";
import { createSelectionColumn } from "../data-table-selection-column";

type TestRow = { id: string; name: string };

const rows: TestRow[] = [
  { id: "1", name: "Alice" },
  { id: "2", name: "Bob" },
];

function SelectionHeader({ rowSelection }: { rowSelection: RowSelectionState }) {
  const table = useTable({
    features: dataTableFeatures,
    data: rows,
    columns: [createSelectionColumn<TestRow>()],
    getRowId: (row) => row.id,
    enableRowSelection: true,
    state: { rowSelection },
    onRowSelectionChange: () => {},
  });

  const header = table.getFlatHeaders()[0];
  return <>{flexRender(header.column.columnDef.header, header.getContext())}</>;
}

function headerCheckbox() {
  return screen.getByLabelText("Select all");
}

describe("selection column header checkbox", () => {
  it("is neither checked nor indeterminate when nothing is selected", () => {
    render(<SelectionHeader rowSelection={{}} />);
    const box = headerCheckbox();

    expect(box.getAttribute("data-checked")).toBeNull();
    expect(box.getAttribute("data-indeterminate")).toBeNull();
  });

  it("is indeterminate when only some rows are selected", () => {
    render(<SelectionHeader rowSelection={{ "1": true }} />);

    expect(headerCheckbox().getAttribute("data-indeterminate")).not.toBeNull();
  });

  it("is checked and NOT indeterminate when every row is selected", () => {
    render(<SelectionHeader rowSelection={{ "1": true, "2": true }} />);
    const box = headerCheckbox();

    expect(box.getAttribute("data-checked")).not.toBeNull();
    expect(box.getAttribute("data-indeterminate")).toBeNull();
  });
});
