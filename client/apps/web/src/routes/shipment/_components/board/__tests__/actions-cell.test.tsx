import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import {
  DataTableRowActionsContext,
  DataTableRowStateContext,
} from "@/contexts/data-table-row-context";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ActionsCell } from "../cells/actions-cell";

afterEach(cleanup);

const row = { id: "shp_1", original: { id: "shp_1" } as Shipment } as Row<Shipment>;

function renderCell({
  isExpanded = false,
  actions = [],
  onToggleExpanded = vi.fn(),
}: {
  isExpanded?: boolean;
  actions?: RowAction<Shipment>[];
  onToggleExpanded?: () => void;
}) {
  return render(
    <DataTableRowActionsContext value={actions}>
      <DataTableRowStateContext value={{ isExpanded, isCursor: false, isSelected: false }}>
        <ActionsCell row={row} onToggleExpanded={onToggleExpanded} />
      </DataTableRowStateContext>
    </DataTableRowActionsContext>,
  );
}

describe("ActionsCell", () => {
  it("offers to expand a closed row and toggles it", () => {
    const onToggleExpanded = vi.fn();
    renderCell({ onToggleExpanded });

    const toggle = screen.getByRole("button", { name: "Expand row" });
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    fireEvent.click(toggle);
    expect(onToggleExpanded).toHaveBeenCalledOnce();
  });

  it("reads its row's open state from the row it sits in", () => {
    renderCell({ isExpanded: true });

    const toggle = screen.getByRole("button", { name: "Collapse row" });
    expect(toggle.getAttribute("aria-expanded")).toBe("true");
  });

  it("lists the table's row actions, hiding the ones that do not apply", async () => {
    const run = vi.fn();
    renderCell({
      actions: [
        { id: "edit", label: "Edit", onClick: run },
        { id: "cancel", label: "Cancel shipment", onClick: vi.fn(), hidden: () => true },
      ],
    });

    fireEvent.click(screen.getByRole("button", { name: "Row actions" }));
    fireEvent.click(await screen.findByRole("menuitem", { name: "Edit" }));
    expect(run).toHaveBeenCalledWith(row);
    expect(screen.queryByRole("menuitem", { name: "Cancel shipment" })).toBeNull();
  });

  it("draws no actions menu when the table has no row actions", () => {
    renderCell({});
    expect(screen.queryByRole("button", { name: "Row actions" })).toBeNull();
  });
});
