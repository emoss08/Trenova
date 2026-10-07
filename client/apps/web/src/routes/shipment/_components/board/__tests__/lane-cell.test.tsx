import { act, cleanup, render, screen } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import { Profiler } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { LaneCell } from "../cells/lane-cell";
import { useShipmentBoardStore } from "../store";

afterEach(() => {
  cleanup();
  act(() => useShipmentBoardStore.getState().setHighlightId(null));
});

function shipment(id: string, origin: string): Shipment {
  return {
    id,
    stage: "NeedsCoverage",
    moves: [
      {
        stops: [
          { type: "Pickup", location: { city: origin } },
          { type: "Delivery", location: { city: `${origin} end` } },
        ],
      },
    ],
  } as unknown as Shipment;
}

function renderLanes() {
  const commits: Record<string, number> = { a: 0, b: 0 };
  render(
    <>
      <Profiler id="a" onRender={() => (commits.a += 1)}>
        <LaneCell shipment={shipment("shp_a", "Dallas")} />
      </Profiler>
      <Profiler id="b" onRender={() => (commits.b += 1)}>
        <LaneCell shipment={shipment("shp_b", "Tulsa")} />
      </Profiler>
    </>,
  );
  commits.a = 0;
  commits.b = 0;
  return commits;
}

describe("LaneCell", () => {
  it("marks the lane the map is pointing at", () => {
    renderLanes();
    act(() => useShipmentBoardStore.getState().setHighlightId("shp_a"));

    expect(screen.getByText("Dallas").className).toContain("text-brand");
    expect(screen.getByText("Tulsa").className).not.toContain("text-brand");
  });

  it("redraws only the lanes whose highlight changed", () => {
    const commits = renderLanes();

    act(() => useShipmentBoardStore.getState().setHighlightId("shp_a"));
    expect(commits).toEqual({ a: 1, b: 0 });

    act(() => useShipmentBoardStore.getState().setHighlightId("shp_c"));
    expect(commits).toEqual({ a: 2, b: 0 });
  });
});
