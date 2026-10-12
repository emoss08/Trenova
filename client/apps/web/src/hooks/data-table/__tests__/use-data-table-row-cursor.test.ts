import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useDataTableRowCursor, type DataTableRowCursorParams } from "../use-data-table-row-cursor";

function setup({
  cursorRowId,
  expandedRowId,
  ...overrides
}: Partial<DataTableRowCursorParams> = {}) {
  const state = {
    cursor: cursorRowId ?? null,
    expanded: expandedRowId ?? null,
    selected: new Set<string>(),
  };
  const onCursorRowIdChange = vi.fn((id: string | null) => {
    state.cursor = id;
  });
  const onExpandedRowIdChange = vi.fn((id: string | null) => {
    state.expanded = id;
  });
  const onToggleSelect = vi.fn((id: string) => {
    if (state.selected.has(id)) state.selected.delete(id);
    else state.selected.add(id);
  });
  const onClearSelection = vi.fn(() => state.selected.clear());

  const params = (): DataTableRowCursorParams => ({
    enabled: true,
    rowIds: ["a", "b", "c"],
    cursorRowId: state.cursor,
    onCursorRowIdChange,
    expandedRowId: state.expanded,
    onExpandedRowIdChange,
    hasSelection: () => state.selected.size > 0,
    onToggleSelect,
    onClearSelection,
    ...overrides,
  });

  const hook = renderHook(() => useDataTableRowCursor(params()));
  const press = (key: string, init: KeyboardEventInit = {}, target: EventTarget = window) => {
    act(() => {
      target.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, ...init }));
    });
    hook.rerender();
  };

  return {
    state,
    press,
    onCursorRowIdChange,
    onExpandedRowIdChange,
    onToggleSelect,
    onClearSelection,
  };
}

afterEach(() => {
  document.body.innerHTML = "";
});

describe("useDataTableRowCursor", () => {
  it("starts on the first row and moves with J/K and the arrows, stopping at the ends", () => {
    const t = setup();
    t.press("j");
    expect(t.state.cursor).toBe("a");
    t.press("ArrowDown");
    t.press("j");
    t.press("j");
    expect(t.state.cursor).toBe("c");
    t.press("k");
    expect(t.state.cursor).toBe("b");
    t.press("ArrowUp");
    t.press("ArrowUp");
    expect(t.state.cursor).toBe("a");
  });

  it("starts from a row that is open even without a cursor", () => {
    const t = setup({ expandedRowId: "b" });
    t.press("j");
    expect(t.state.cursor).toBe("c");
  });

  it("carries an open row along with the cursor", () => {
    const t = setup({ cursorRowId: "a", expandedRowId: "a" });
    t.press("j");
    expect(t.state.cursor).toBe("b");
    expect(t.state.expanded).toBe("b");
  });

  it("toggles the cursor row open and shut with Enter", () => {
    const t = setup({ cursorRowId: "b" });
    t.press("Enter");
    expect(t.state.expanded).toBe("b");
    t.press("Enter");
    expect(t.state.expanded).toBeNull();
  });

  it("toggles selection of the cursor row with X", () => {
    const t = setup({ cursorRowId: "c" });
    t.press("x");
    expect(t.onToggleSelect).toHaveBeenCalledWith("c");
  });

  it("pins or unpins the cursor row with P", () => {
    const onTogglePin = vi.fn();
    const t = setup({ cursorRowId: "b", onTogglePin });
    t.press("p");
    expect(onTogglePin).toHaveBeenCalledWith("b");
  });

  it("does nothing on P when the table cannot pin rows or there is no cursor", () => {
    const onTogglePin = vi.fn();
    setup({ onTogglePin }).press("p");
    expect(onTogglePin).not.toHaveBeenCalled();
  });

  it("backs out one step per Escape: collapse, then clear selection, then drop the cursor", () => {
    const t = setup({ cursorRowId: "a", expandedRowId: "a" });
    t.state.selected.add("b");
    t.press("Escape");
    expect(t.state.expanded).toBeNull();
    expect(t.state.selected.size).toBe(1);
    t.press("Escape");
    expect(t.onClearSelection).toHaveBeenCalledTimes(1);
    expect(t.state.cursor).toBe("a");
    t.press("Escape");
    expect(t.state.cursor).toBeNull();
  });

  it("ignores keys typed into a field, inside a dialog, or with a modifier", () => {
    const t = setup({ cursorRowId: "a" });
    const input = document.createElement("input");
    document.body.append(input);
    t.press("j", {}, input);

    const dialog = document.createElement("div");
    dialog.setAttribute("role", "dialog");
    const inner = document.createElement("button");
    dialog.append(inner);
    document.body.append(dialog);
    t.press("j", {}, inner);

    t.press("j", { metaKey: true });
    t.press("x", { ctrlKey: true });
    t.press("Enter", { altKey: true });

    expect(t.onCursorRowIdChange).not.toHaveBeenCalled();
    expect(t.onToggleSelect).not.toHaveBeenCalled();
    expect(t.onExpandedRowIdChange).not.toHaveBeenCalled();
  });

  it("does nothing while disabled or with no rows", () => {
    const off = setup({ enabled: false });
    off.press("j");
    expect(off.onCursorRowIdChange).not.toHaveBeenCalled();

    const empty = setup({ rowIds: [] });
    empty.press("j");
    empty.press("Enter");
    expect(empty.onCursorRowIdChange).not.toHaveBeenCalled();
    expect(empty.onExpandedRowIdChange).not.toHaveBeenCalled();
  });

  it("re-anchors on the first row when the cursor's row left the page", () => {
    const t = setup({ cursorRowId: "gone" });
    t.press("j");
    expect(t.state.cursor).toBe("a");
  });
});

describe("useDataTableRowCursor row shortcuts", () => {
  it("runs a shortcut against the cursor row, matching its modifiers exactly", () => {
    const edit = vi.fn();
    const copyPro = vi.fn();
    const copyLink = vi.fn();
    const t = setup({
      cursorRowId: "b",
      shortcuts: [
        { key: "e", run: edit },
        { key: "c", alt: true, run: copyPro },
        { key: "l", mod: true, run: copyLink },
      ],
    });

    t.press("e");
    expect(edit).toHaveBeenCalledWith("b");
    t.press("c");
    expect(copyPro).not.toHaveBeenCalled();
    t.press("c", { altKey: true, code: "KeyC" });
    expect(copyPro).toHaveBeenCalledWith("b");
    t.press("l", { ctrlKey: true });
    t.press("l", { metaKey: true });
    expect(copyLink).toHaveBeenCalledTimes(2);
    t.press("e", { altKey: true });
    expect(edit).toHaveBeenCalledTimes(1);
  });

  it("reads Alt chords by physical key, since Option rewrites the character on a Mac", () => {
    const copyPro = vi.fn();
    const t = setup({ cursorRowId: "a", shortcuts: [{ key: "c", alt: true, run: copyPro }] });
    t.press("ç", { altKey: true, code: "KeyC" });
    expect(copyPro).toHaveBeenCalledWith("a");
  });

  it("runs nothing without a cursor or open row", () => {
    const edit = vi.fn();
    const t = setup({ shortcuts: [{ key: "e", run: edit }] });
    t.press("e");
    expect(edit).not.toHaveBeenCalled();
  });
});
