import { fireEvent, render, screen } from "@testing-library/react";
import { setUserDatePreferences } from "@trenova/shared/lib/date";
import type { SingleFilterItem } from "@trenova/shared/types/data-table";
import { afterEach, describe, expect, it, vi } from "vitest";
import DataTableFilterChips from "../data-table-filter-chips";

function dateFilter(operator: SingleFilterItem["operator"], value: unknown): SingleFilterItem {
  return {
    id: "time",
    type: "filter",
    field: "timestamp",
    apiField: "timestamp",
    label: "Time",
    operator,
    value,
    filterType: "date",
  };
}

function chipText(filter: SingleFilterItem) {
  const { container } = render(
    <DataTableFilterChips
      filters={[filter]}
      onFiltersChange={vi.fn()}
      query=""
      onClearQuery={vi.fn()}
    />,
  );
  return (container.textContent ?? "").replace(/\s/g, " ");
}

describe("DataTableFilterChips with a date filter", () => {
  afterEach(() => {
    setUserDatePreferences({});
  });

  it("reads a range of hours as the date and its times", () => {
    setUserDatePreferences({ timezone: "America/Denver", timeFormat: "12-hour" });

    const text = chipText(dateFilter("daterange", { from: 1_793_908_800, to: 1_793_912_399 }));

    expect(text).toBe("Time is between Nov 5, 2026, 1:00 – 1:59 PM");
  });

  it("reads a range of whole days as dates", () => {
    setUserDatePreferences({ timezone: "America/Denver" });

    const text = chipText(dateFilter("daterange", { from: 1_793_689_200, to: 1_793_862_000 }));

    expect(text).toBe("Time is between Nov 3 – 5, 2026");
  });

  it("reads a single day as its date", () => {
    setUserDatePreferences({ timezone: "America/Denver" });

    expect(chipText(dateFilter("gte", 1_793_862_000))).toBe(
      "Time greater than or equal Nov 5, 2026",
    );
  });

  it("keeps a count of days as the number", () => {
    expect(chipText(dateFilter("lastndays", 7))).toBe("Time in last N days 7");
  });
});

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
