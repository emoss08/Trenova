import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { useMapUIState } from "../use-map-ui-state";

beforeEach(() => {
  window.localStorage.clear();
});

describe("useMapUIState", () => {
  it("starts on selected routes, with delivered loads and no alert categories hidden", () => {
    const { result } = renderHook(() => useMapUIState());

    expect(result.current.routeMode).toBe("selected");
    expect(result.current.hiddenStages).toEqual(["Delivered"]);
    expect(result.current.hiddenAlertCategories).toEqual([]);
  });

  it("drops stored values this build does not know", () => {
    window.localStorage.setItem("shipment-map-route-mode", JSON.stringify("everything"));
    window.localStorage.setItem("shipment-map-hidden-stages", JSON.stringify(["Late", "Shipped"]));
    window.localStorage.setItem(
      "shipment-map-hidden-alert-categories",
      JSON.stringify(["fire", "volcano", 3]),
    );

    const { result } = renderHook(() => useMapUIState());

    expect(result.current.routeMode).toBe("selected");
    expect(result.current.hiddenStages).toEqual(["Late"]);
    expect(result.current.hiddenAlertCategories).toEqual(["fire"]);
  });

  it("hides a stage, shows it again, and remembers the choice", () => {
    const { result, unmount } = renderHook(() => useMapUIState());

    act(() => result.current.toggleStage("Scheduled"));
    expect(result.current.hiddenStages).toEqual(["Delivered", "Scheduled"]);

    act(() => result.current.toggleStage("Delivered"));
    expect(result.current.hiddenStages).toEqual(["Scheduled"]);

    unmount();
    const { result: reopened } = renderHook(() => useMapUIState());
    expect(reopened.current.hiddenStages).toEqual(["Scheduled"]);
  });

  it("remembers the route mode", () => {
    const { result, unmount } = renderHook(() => useMapUIState());

    act(() => result.current.setRouteMode("lanes"));
    unmount();

    const { result: reopened } = renderHook(() => useMapUIState());
    expect(reopened.current.routeMode).toBe("lanes");
  });
});
