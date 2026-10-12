import { act, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DataTableSearchSuggestions } from "@trenova/shared/types/data-table";
import DataTableSearch from "../data-table-search";

function suggestions(overrides: Partial<DataTableSearchSuggestions> = {}) {
  const late = vi.fn();
  const moving = vi.fn();
  const value: DataTableSearchSuggestions = {
    title: "Quick filters",
    items: [
      { key: "late", label: "Late", count: 4, onSelect: late },
      { key: "moving", label: "Moving", count: 1200, selected: true, onSelect: moving },
    ],
    ...overrides,
  };
  return { value, late, moving };
}

function input() {
  return screen.getByRole("combobox");
}

describe("DataTableSearch suggestions", () => {
  it("offers the suggestions with their counts when the empty field is focused", () => {
    const { value, late } = suggestions();
    render(<DataTableSearch value="" onChange={vi.fn()} suggestions={value} />);

    expect(screen.queryByRole("listbox")).toBeNull();
    fireEvent.focus(input());

    const list = screen.getByRole("listbox", { name: "Quick filters" });
    expect(list.textContent).toContain("Late");
    expect(list.textContent).toContain("1,200");
    expect(screen.getByRole("option", { name: /Moving/ }).getAttribute("aria-selected")).toBe(
      "true",
    );

    fireEvent.click(screen.getByRole("option", { name: /Late/ }));
    expect(late).toHaveBeenCalledOnce();
  });

  it("tells the host when the list opens, so it can fetch the counts", () => {
    const onOpenChange = vi.fn();
    const { value } = suggestions({ onOpenChange });
    render(<DataTableSearch value="" onChange={vi.fn()} suggestions={value} />);

    fireEvent.focus(input());
    expect(onOpenChange).toHaveBeenLastCalledWith(true);
  });

  it("puts the list away once the person starts typing", () => {
    const { value } = suggestions();
    render(<DataTableSearch value="" onChange={vi.fn()} suggestions={value} />);

    fireEvent.focus(input());
    fireEvent.change(input(), { target: { value: "acme" } });
    expect(screen.queryByRole("listbox")).toBeNull();
  });

  it("walks the list with the arrow keys and picks with Enter", () => {
    const { value, moving } = suggestions();
    render(<DataTableSearch value="" onChange={vi.fn()} suggestions={value} />);

    fireEvent.focus(input());
    fireEvent.keyDown(input(), { key: "ArrowDown" });
    fireEvent.keyDown(input(), { key: "ArrowDown" });
    fireEvent.keyDown(input(), { key: "Enter" });
    expect(moving).toHaveBeenCalledOnce();
  });

  it("draws no list for a table without suggestions", () => {
    render(<DataTableSearch value="" onChange={vi.fn()} />);
    fireEvent.focus(screen.getByRole("textbox"));
    expect(screen.queryByRole("listbox")).toBeNull();
  });
});

describe("DataTableSearch shortcut", () => {
  it("focuses the field from anywhere on the page", () => {
    render(<DataTableSearch value="" onChange={vi.fn()} shortcut="/" />);

    act(() => {
      window.dispatchEvent(new KeyboardEvent("keydown", { key: "/", bubbles: true }));
    });
    expect(document.activeElement).toBe(screen.getByRole("textbox"));
  });

  it("leaves the key alone while the person types somewhere else", () => {
    render(
      <>
        <input aria-label="Notes" />
        <DataTableSearch value="" onChange={vi.fn()} shortcut="/" />
      </>,
    );
    const notes = screen.getByLabelText("Notes");
    notes.focus();

    fireEvent.keyDown(notes, { key: "/" });
    expect(document.activeElement).toBe(notes);
  });
});
