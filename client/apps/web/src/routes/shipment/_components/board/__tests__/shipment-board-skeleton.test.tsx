import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { ShipmentBoardCapabilities } from "@/lib/graphql/shipment-board";
import { BoardCapabilitiesProvider } from "@/lib/shipment-board/capabilities";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BOARD_CELL_SKELETONS } from "../board-cell-skeletons";
import { getColumns, SHIPMENT_HIDDEN_COLUMNS } from "../shipment-columns";
import { boardSkeletonColumns, ShipmentBoardSkeleton } from "../shipment-board-skeleton";

const t = ((message: string) => message) as TranslateFn;

const CAPABILITIES: ShipmentBoardCapabilities = {
  ai: true,
  operationType: "both",
  hos: false,
  maps: false,
};

function renderSkeleton(searchParams = "") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter searchParams={searchParams}>
          <BoardCapabilitiesProvider value={CAPABILITIES}>
            <ShipmentBoardSkeleton />
          </BoardCapabilitiesProvider>
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

afterEach(cleanup);

describe("the board's skeleton columns", () => {
  const shown = getColumns({ t, onToggleExpanded: vi.fn() }).filter(
    (column) => SHIPMENT_HIDDEN_COLUMNS[column.id ?? ""] !== false,
  );

  it("are the columns the board opens with, in order, at their sizes", () => {
    const drawn = boardSkeletonColumns(t).map(({ id, label, size, sortable }) => ({
      id,
      label,
      size,
      sortable,
    }));

    expect(drawn).toEqual(
      shown.map((column) => ({
        id: column.id,
        // The actions column's head is for screen readers, so nothing is drawn for it.
        label:
          column.id === "actions"
            ? ""
            : typeof column.header === "string"
              ? column.header
              : column.meta?.label,
        size: column.size,
        sortable: column.meta?.sortable !== false,
      })),
    );
  });

  it("each have the cell shape the table's loading rows draw too", () => {
    for (const column of shown) {
      expect(column.meta?.skeleton).toBe(BOARD_CELL_SKELETONS[column.id ?? ""]);
      expect(column.meta?.skeleton).toBeTruthy();
    }
  });
});

describe("ShipmentBoardSkeleton", () => {
  it("draws the table's header, a group and a page of rows", () => {
    renderSkeleton();

    expect(screen.getByLabelText("Loading data...")).toHaveAttribute("aria-busy", "true");
    expect(screen.getByText("Lane")).toBeInTheDocument();
    expect(document.querySelectorAll("tbody tr")).toHaveLength(26);
  });

  it("draws no group row on a board that is not grouped", () => {
    renderSkeleton("?group=none");

    expect(document.querySelectorAll("tbody tr")).toHaveLength(25);
  });

  it("draws the timeline's skeleton when the address names the timeline", () => {
    renderSkeleton("?view=timeline");

    expect(screen.getByLabelText("Loading the timeline")).toBeInTheDocument();
    expect(screen.queryByText("Lane")).toBeNull();
  });

  it("draws neither table nor timeline when the address names the map", () => {
    renderSkeleton("?view=map");

    expect(screen.queryByLabelText("Loading the timeline")).toBeNull();
    expect(screen.queryByText("Lane")).toBeNull();
  });
});
