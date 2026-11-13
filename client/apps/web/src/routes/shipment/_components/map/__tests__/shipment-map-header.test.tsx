import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MAP_STAGES, type MapStage } from "../map-shipment-model";
import { ShipmentMapHeader } from "../shipment-map-header";

const counts = Object.fromEntries(MAP_STAGES.map((stage) => [stage, 0])) as Record<
  MapStage,
  number
>;
counts.Late = 7;
counts.NeedsCoverage = 63;
counts.Moving = 1;

function renderHeader(compact: boolean, onToggleStage = vi.fn()) {
  render(
    <ShipmentMapHeader
      compact={compact}
      counts={counts}
      hiddenStages={[]}
      onToggleStage={onToggleStage}
      routeMode="selected"
      onRouteModeChange={() => {}}
      dataUpdatedAt={0}
      controls={<button type="button">Zoom to fit</button>}
    />,
  );
  return onToggleStage;
}

afterEach(cleanup);

describe("ShipmentMapHeader", () => {
  it("carries the title, the route modes and the live state on the workspace's row", () => {
    renderHeader(false);

    expect(screen.getByText("Live map")).toBeInTheDocument();
    expect(screen.getByLabelText("Routes shown")).toBeInTheDocument();
    expect(screen.getByText(/synced/)).toBeInTheDocument();
  });

  // A tile has one row for the header. The title is the tile's own, and the route
  // modes and the live state come back in full screen, so none of them is drawn.
  it("keeps only the stage chips and the map controls on a tile's row", () => {
    renderHeader(true);

    expect(screen.queryByText("Live map")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Routes shown")).not.toBeInTheDocument();
    expect(screen.queryByText(/synced/)).not.toBeInTheDocument();

    const chips = within(screen.getByRole("group", { name: "Show stages" })).getAllByRole("button");
    expect(chips.map((chip) => chip.textContent)).toEqual([
      "Needs attention7",
      "Needs coverage63",
      "Moving1",
    ]);
    expect(screen.getByRole("button", { name: "Zoom to fit" })).toBeInTheDocument();
  });

  it("lays a tile's chips on one line that scrolls rather than wraps", () => {
    renderHeader(true);

    const group = screen.getByRole("group", { name: "Show stages" });
    expect(group.className).toContain("w-max");
    expect(group.className).not.toContain("flex-wrap");
    for (const chip of within(group).getAllByRole("button")) {
      expect(chip.className).toContain("shrink-0");
    }
  });

  it("still filters by stage from a tile's chips", () => {
    const onToggleStage = renderHeader(true);

    fireEvent.click(screen.getByRole("button", { name: /Needs coverage/ }));

    expect(onToggleStage).toHaveBeenCalledWith("NeedsCoverage");
  });
});
