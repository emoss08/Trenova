import { describe, expect, it, vi } from "vitest";
import { constructTable, coreFeatures, tableFeatures } from "@tanstack/react-table";
import { storeReactivityBindings } from "@tanstack/table-core/store-reactivity-bindings";
import { cellEditingFeature, type CellEditCommitParams } from "../cell-editing-feature";

type TestRow = { id: string; name: string; status: string };

const features = tableFeatures({
  ...coreFeatures,
  coreReactivityFeature: storeReactivityBindings(),
  cellEditingFeature,
});

const rows: TestRow[] = [
  { id: "r1", name: "Alpha", status: "active" },
  { id: "r2", name: "Beta", status: "inactive" },
];

function createTestTable(options?: {
  enableCellEditing?: boolean;
  onCellEditCommit?: (params: CellEditCommitParams<TestRow>) => void | Promise<void>;
}) {
  return constructTable({
    features,
    data: rows,
    columns: [
      { accessorKey: "name", header: "Name", enableCellEditing: true },
      { accessorKey: "status", header: "Status" },
    ],
    getRowId: (row: TestRow) => row.id,
    enableCellEditing: options?.enableCellEditing ?? true,
    onCellEditCommit: options?.onCellEditCommit,
  });
}

function getCell(table: ReturnType<typeof createTestTable>, rowId: string, columnId: string) {
  const row = table.getRowModel().rows.find((r) => r.id === rowId);
  const cell = row?.getAllCells().find((c) => c.column.id === columnId);
  if (!cell) throw new Error(`cell ${rowId}/${columnId} not found`);
  return cell;
}

describe("cellEditingFeature", () => {
  it("initializes with no editing cell and registers the state slice", () => {
    const table = createTestTable();
    expect(table.getEditingCell()).toBeNull();
    const atoms = (table as unknown as { atoms: { cellEditing?: { get: () => unknown } } }).atoms;
    expect(atoms.cellEditing?.get()).toBeNull();
  });

  it("only allows editing on columns that opt in while the table option is enabled", () => {
    const table = createTestTable();
    expect(getCell(table, "r1", "name").getCanEdit()).toBe(true);
    expect(getCell(table, "r1", "status").getCanEdit()).toBe(false);

    const disabledTable = createTestTable({ enableCellEditing: false });
    expect(getCell(disabledTable, "r1", "name").getCanEdit()).toBe(false);
  });

  it("tracks the active editing cell through start/stop", () => {
    const table = createTestTable();
    const cell = getCell(table, "r1", "name");

    cell.startEditing();
    expect(table.getEditingCell()).toEqual({ rowId: "r1", columnId: "name" });
    expect(cell.getIsEditing()).toBe(true);
    expect(getCell(table, "r2", "name").getIsEditing()).toBe(false);

    cell.stopEditing();
    expect(table.getEditingCell()).toBeNull();
    expect(cell.getIsEditing()).toBe(false);
  });

  it("refuses to start editing a non-editable cell", () => {
    const table = createTestTable();
    getCell(table, "r1", "status").startEditing();
    expect(table.getEditingCell()).toBeNull();
  });

  it("moves editing to the most recently started cell", () => {
    const table = createTestTable();
    getCell(table, "r1", "name").startEditing();
    getCell(table, "r2", "name").startEditing();
    expect(table.getEditingCell()).toEqual({ rowId: "r2", columnId: "name" });
    expect(getCell(table, "r1", "name").getIsEditing()).toBe(false);
  });

  it("commits with row context and previous value, then stops editing", async () => {
    const onCellEditCommit = vi.fn();
    const table = createTestTable({ onCellEditCommit });
    const cell = getCell(table, "r1", "name");

    cell.startEditing();
    await cell.commitEdit("Alpha Prime");

    expect(onCellEditCommit).toHaveBeenCalledExactlyOnceWith({
      rowId: "r1",
      columnId: "name",
      value: "Alpha Prime",
      previousValue: "Alpha",
      row: rows[0],
    });
    expect(table.getEditingCell()).toBeNull();
  });

  it("keeps the editing state when the commit handler rejects", async () => {
    const onCellEditCommit = vi.fn().mockRejectedValue(new Error("validation failed"));
    const table = createTestTable({ onCellEditCommit });
    const cell = getCell(table, "r1", "name");

    cell.startEditing();
    await expect(cell.commitEdit("bad value")).rejects.toThrow("validation failed");
    expect(table.getEditingCell()).toEqual({ rowId: "r1", columnId: "name" });
  });

  it("ignores commits from cells that are not being edited", async () => {
    const onCellEditCommit = vi.fn();
    const table = createTestTable({ onCellEditCommit });

    await getCell(table, "r1", "name").commitEdit("ignored");
    expect(onCellEditCommit).not.toHaveBeenCalled();
  });

  it("supports controlled state via onCellEditingChange", () => {
    const changes: unknown[] = [];
    const table = constructTable({
      features,
      data: rows,
      columns: [{ accessorKey: "name", header: "Name", enableCellEditing: true }],
      getRowId: (row: TestRow) => row.id,
      enableCellEditing: true,
      onCellEditingChange: (updater) => {
        changes.push(typeof updater === "function" ? updater(null) : updater);
      },
    });

    getCell(table, "r1", "name").startEditing();
    expect(changes).toEqual([{ rowId: "r1", columnId: "name" }]);
  });
});
