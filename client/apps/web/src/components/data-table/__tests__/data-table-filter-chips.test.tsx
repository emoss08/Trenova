import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import DataTableFilterChips from "../data-table-filter-chips";

describe("DataTableFilterChips with the host's own filters", () => {
  it("shows them beside the table's filters and removes one at a time", () => {
    const removeLate = vi.fn();
    render(
      <DataTableFilterChips
        filters={[]}
        onFiltersChange={vi.fn()}
        query=""
        onClearQuery={vi.fn()}
        extraChips={{
          items: [
            { key: "late", label: "Late", onRemove: removeLate },
            { key: "reefer", label: "Reefer", onRemove: vi.fn() },
          ],
          onClear: vi.fn(),
        }}
      />,
    );

    expect(screen.getByText("Late")).toBeTruthy();
    expect(screen.getByText("Reefer")).toBeTruthy();
    fireEvent.click(screen.getAllByRole("button", { name: "Remove filter" })[0]);
    expect(removeLate).toHaveBeenCalledOnce();
  });

  it("clears them with everything else", () => {
    const onClear = vi.fn();
    const onFiltersChange = vi.fn();
    const onClearQuery = vi.fn();
    render(
      <DataTableFilterChips
        filters={[]}
        onFiltersChange={onFiltersChange}
        query="acme"
        onClearQuery={onClearQuery}
        extraChips={{
          items: [{ key: "late", label: "Late", onRemove: vi.fn() }],
          onClear,
        }}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Clear all" }));
    expect(onClear).toHaveBeenCalledOnce();
    expect(onClearQuery).toHaveBeenCalledOnce();
    expect(onFiltersChange).toHaveBeenCalledWith([]);
  });
});
