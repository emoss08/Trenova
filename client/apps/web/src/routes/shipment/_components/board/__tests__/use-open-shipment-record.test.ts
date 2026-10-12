import { act, renderHook } from "@testing-library/react";
import { withNuqsTestingAdapter, type UrlUpdateEvent } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useOpenShipmentRecord } from "../use-open-shipment-record";

const BOARD_URL = "?view=timeline&group=false&qf=late&collapsed=2";

function renderOpenRecord(searchParams = BOARD_URL) {
  const updates: UrlUpdateEvent[] = [];
  const { result } = renderHook(() => useOpenShipmentRecord(), {
    wrapper: withNuqsTestingAdapter({
      searchParams,
      onUrlUpdate: (event) => updates.push(event),
    }),
  });
  return { open: result.current, updates };
}

async function flushUrlUpdates() {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 100));
  });
}

describe("useOpenShipmentRecord", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("expands the row and opens the editor without leaving the page", async () => {
    const windowOpen = vi.spyOn(window, "open").mockReturnValue(null);
    const { open, updates } = renderOpenRecord();

    act(() => open("shp_01"));
    await flushUrlUpdates();

    expect(windowOpen).not.toHaveBeenCalled();
    const last = updates.at(-1)!;
    expect(last.searchParams.get("expanded")).toBe("shp_01");
    expect(last.searchParams.get("panelType")).toBe("edit");
    expect(last.searchParams.get("panelEntityId")).toBe("shp_01");
  });

  it("keeps the board's view, grouping, filters and collapsed groups", async () => {
    const { open, updates } = renderOpenRecord();

    act(() => open("shp_01"));
    await flushUrlUpdates();

    const last = updates.at(-1)!;
    expect(last.searchParams.get("view")).toBe("timeline");
    expect(last.searchParams.get("group")).toBe("false");
    expect(last.searchParams.get("qf")).toBe("late");
    expect(last.searchParams.get("collapsed")).toBe("2");
  });

  it("switches to another shipment when one is already open", async () => {
    const { open, updates } = renderOpenRecord(
      `${BOARD_URL}&expanded=shp_01&panelType=edit&panelEntityId=shp_01`,
    );

    act(() => open("shp_02"));
    await flushUrlUpdates();

    const last = updates.at(-1)!;
    expect(last.searchParams.get("expanded")).toBe("shp_02");
    expect(last.searchParams.get("panelEntityId")).toBe("shp_02");
    expect(last.searchParams.getAll("expanded")).toHaveLength(1);
  });
});
