import { act, cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useShipmentBoardStore } from "../../board/store";
import { useMapFocusStore } from "../map-focus-store";
import { buildMapShipments } from "../map-shipment-model";
import { ShipmentMapRail } from "../shipment-map-rail";
import { matchShipmentsToAlerts } from "../weather-alert-matching";
import { alert, PLACES, shipment, square } from "./fixtures";

const { dallas, memphis, chicago, houston } = PLACES;

afterEach(() => {
  cleanup();
  act(() => {
    useMapFocusStore.getState().reset();
    useShipmentBoardStore.getState().setHighlightId(null);
  });
});

const models = buildMapShipments([
  shipment(
    "moving",
    [
      { at: dallas, status: "Completed" },
      { at: memphis, type: "Delivery" },
    ],
    { stage: "Moving", eta: { estimatedArrival: 2000, slackMinutes: 40, verdict: "OnTime" } },
  ),
  shipment(
    "late",
    [
      { at: houston, status: "Completed" },
      { at: chicago, type: "Delivery" },
    ],
    { stage: "Late", eta: { estimatedArrival: 1800, slackMinutes: -90, verdict: "Late" } },
  ),
]);

const memphisFlood = alert("memphis-flood", [square(memphis.lat, memphis.lng, 0.3)]);
const farFire = alert("far-fire", [square(47, -120, 0.5)], {
  event: "Red Flag Warning",
  category: "fire",
});

function renderRail({ alertsOn = false } = {}) {
  const features = alertsOn ? [memphisFlood, farFire] : [];
  return render(
    <ShipmentMapRail
      models={models}
      matches={matchShipmentsToAlerts(models, features)}
      alertsOn={alertsOn}
      features={features}
      allFeatures={features}
      hiddenAlertCategories={[]}
      onToggleAlertCategory={vi.fn()}
    />,
  );
}

function sectionLabels() {
  return screen.getAllByRole("region").map((section) => section.getAttribute("aria-label"));
}

describe("ShipmentMapRail", () => {
  it("lists late loads before moving ones", () => {
    renderRail();

    expect(sectionLabels()).toEqual(["Needs attention", "Moving"]);
  });

  it("opens a shipment in the map's card when its row is selected, and closes it on a second click", () => {
    renderRail();
    const row = screen.getByRole("button", { name: /PRO-moving/ });

    fireEvent.click(row);
    expect(useMapFocusStore.getState().focus).toEqual({ kind: "shipment", id: "moving" });
    expect(row.getAttribute("aria-pressed")).toBe("true");

    fireEvent.click(row);
    expect(useMapFocusStore.getState().focus).toBeNull();
  });

  it("lights the hovered shipment's mark", () => {
    renderRail();

    fireEvent.mouseEnter(screen.getByRole("button", { name: /PRO-late/ }));
    expect(useShipmentBoardStore.getState().highlightId).toBe("late");

    fireEvent.mouseLeave(screen.getByRole("button", { name: /PRO-late/ }));
    expect(useShipmentBoardStore.getState().highlightId).toBeNull();
  });

  it("narrows to a picked cluster or lane until Show all", () => {
    renderRail();
    act(() =>
      useMapFocusStore.getState().setScope({ ids: ["late"], label: "1 shipment near Houston, TX" }),
    );

    expect(screen.getByText("1 shipment near Houston, TX")).toBeTruthy();
    expect(screen.queryByRole("button", { name: /PRO-moving/ })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Show all" }));
    expect(screen.getByRole("button", { name: /PRO-moving/ })).toBeTruthy();
  });

  it("finds a shipment by city", () => {
    renderRail();

    fireEvent.change(screen.getByRole("searchbox", { name: "Search shipments" }), {
      target: { value: "chicago" },
    });

    expect(screen.getByRole("button", { name: /PRO-late/ })).toBeTruthy();
    expect(screen.queryByRole("button", { name: /PRO-moving/ })).toBeNull();
  });

  it("says when a shipment's next stop is inside a public alert", () => {
    renderRail({ alertsOn: true });

    const row = screen.getByRole("button", { name: /PRO-moving/ });
    expect(within(row).getByText("Flood Warning at the next stop")).toBeTruthy();
    expect(
      within(screen.getByRole("button", { name: /PRO-late/ })).queryByText(/Warning/),
    ).toBeNull();
  });

  it("lists alerts on our routes first, each with how many shipments it affects", () => {
    renderRail({ alertsOn: true });
    act(() => useMapFocusStore.getState().setRailTab("alerts"));

    expect(sectionLabels()).toEqual(["On your routes", "Elsewhere"]);
    const onRoutes = screen.getByRole("region", { name: "On your routes" });
    expect(within(onRoutes).getByText("Flood Warning")).toBeTruthy();
    expect(within(onRoutes).getByText("1 shipment affected")).toBeTruthy();
    const elsewhere = screen.getByRole("region", { name: "Elsewhere" });
    expect(within(elsewhere).getByText("Red Flag Warning")).toBeTruthy();
    expect(within(elsewhere).getByText("No shipments affected")).toBeTruthy();
  });

  it("hides the alerts tab while public alerts are off", () => {
    renderRail();

    expect(screen.queryByRole("tab", { name: /Weather alerts/ })).toBeNull();
  });
});
