import { buildTimelineLoads, type TimelineLoad } from "@/lib/shipment-board/timeline";
import type { Shipment } from "@trenova/shared/types/shipment";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TimelineBar } from "../timeline-bar";
import { TimelineHoverCard } from "../timeline-hover-card";
import { createTimelineScale } from "../timeline-scale";

/* Loads are built from shipments shaped as the ShipmentScheduleFields fragment
   selects them. Times are Unix seconds from a fixed Tuesday in UTC. */
const TUE = Date.UTC(2026, 10, 3) / 1000;
const HOUR = 3_600;
const at = (hours: number) => TUE + hours * HOUR;
const scale = createTimelineScale("day", [TUE, TUE + 24 * HOUR]);
/** The card is 288px wide and centered on the pointer. */
const HALF_CARD = 144;

function shipment(id: string, from: string, to: string, startHour: number): Shipment {
  const stop = (stopId: string, city: string, hour: number, type: string) => ({
    id: stopId,
    type,
    status: "New",
    scheduledWindowStart: at(hour),
    scheduledWindowEnd: at(hour + 1),
    actualArrival: null,
    actualDeparture: null,
    location: { city, state: { abbreviation: "TX" } },
  });
  return {
    id,
    proNumber: `PRO-${id}`,
    stage: "Moving",
    status: "InTransit",
    eta: null,
    tenderStatus: null,
    customer: { id: `cus_${id}`, name: `Customer ${id}` },
    moves: [
      {
        id: `mov_${id}`,
        status: "New",
        coverageType: "unassigned",
        stops: [
          stop(`${id}_a`, from, startHour, "Pickup"),
          stop(`${id}_b`, to, startHour + 6, "Delivery"),
        ],
      },
    ],
  } as unknown as Shipment;
}

const [laredo, houston] = buildTimelineLoads([
  shipment("1", "Laredo", "Dallas", 2),
  shipment("2", "Houston", "Austin", 12),
]);

function renderBars(loads: TimelineLoad[]) {
  return render(
    <TimelineHoverCard scale={scale}>
      {loads.map((load) => (
        <TimelineBar
          key={load.id}
          load={load}
          scale={scale}
          top={0}
          height={20}
          caption="timing"
          selected={false}
          onSelect={() => {}}
        />
      ))}
    </TimelineHoverCard>,
  );
}

const bar = (name: string) => screen.getByRole("button", { name });
const card = () => screen.queryByRole("tooltip");

beforeEach(() => {
  vi.useFakeTimers({
    toFake: ["setTimeout", "clearTimeout", "requestAnimationFrame", "cancelAnimationFrame"],
  });
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("TimelineHoverCard", () => {
  it("opens on a bar after a short rest, with the load's stops and who has it", () => {
    renderBars([laredo]);
    fireEvent.pointerEnter(bar("PRO-1, Laredo → Dallas"), { clientX: 400 });

    act(() => vi.advanceTimersByTime(79));
    expect(card()).toBeNull();

    act(() => vi.advanceTimersByTime(1));
    expect(card()?.textContent).toContain("PRO-1");
    expect(card()?.textContent).toContain("Customer 1");
    expect(card()?.textContent).toContain("Not covered");
  });

  it("follows the pointer along the bar", () => {
    renderBars([laredo]);
    const target = bar("PRO-1, Laredo → Dallas");
    fireEvent.pointerEnter(target, { clientX: 400 });
    act(() => vi.advanceTimersByTime(80));
    expect(card()?.style.left).toBe(`${400 - HALF_CARD}px`);

    fireEvent.pointerMove(target, { clientX: 600 });
    act(() => vi.advanceTimersToNextFrame());
    expect(card()?.style.left).toBe(`${600 - HALF_CARD}px`);
  });

  it("opens where the pointer is when it moved while the card was waiting", () => {
    renderBars([laredo]);
    const target = bar("PRO-1, Laredo → Dallas");
    fireEvent.pointerEnter(target, { clientX: 300 });
    fireEvent.pointerMove(target, { clientX: 500 });
    act(() => vi.advanceTimersByTime(80));

    expect(card()?.style.left).toBe(`${500 - HALF_CARD}px`);
  });

  it("switches to the next bar at once while open", () => {
    renderBars([laredo, houston]);
    fireEvent.pointerEnter(bar("PRO-1, Laredo → Dallas"), { clientX: 400 });
    act(() => vi.advanceTimersByTime(80));

    fireEvent.pointerLeave(bar("PRO-1, Laredo → Dallas"));
    fireEvent.pointerEnter(bar("PRO-2, Houston → Austin"), { clientX: 700 });

    expect(card()?.textContent).toContain("PRO-2");
    expect(card()?.style.left).toBe(`${700 - HALF_CARD}px`);
  });

  it("closes shortly after the pointer leaves the bars", () => {
    renderBars([laredo]);
    fireEvent.pointerEnter(bar("PRO-1, Laredo → Dallas"), { clientX: 400 });
    act(() => vi.advanceTimersByTime(80));

    fireEvent.pointerLeave(bar("PRO-1, Laredo → Dallas"));
    act(() => vi.advanceTimersByTime(90));
    expect(card()).toBeNull();
  });

  it("closes on Escape and as Space picks the load up", () => {
    renderBars([laredo]);
    const target = bar("PRO-1, Laredo → Dallas");
    fireEvent.pointerEnter(target, { clientX: 400 });
    act(() => vi.advanceTimersByTime(80));
    fireEvent.keyDown(target, { key: "Escape" });
    expect(card()).toBeNull();

    fireEvent.pointerLeave(target);
    fireEvent.pointerEnter(target, { clientX: 400 });
    act(() => vi.advanceTimersByTime(80));
    fireEvent.keyDown(target, { key: " " });
    expect(card()).toBeNull();
  });

  it("closes when the hovered bar leaves the sheet, as a refetch that drops its load does", () => {
    const view = renderBars([laredo, houston]);
    fireEvent.pointerEnter(bar("PRO-1, Laredo → Dallas"), { clientX: 400 });
    act(() => vi.advanceTimersByTime(80));
    expect(card()?.textContent).toContain("PRO-1");

    view.rerender(
      <TimelineHoverCard scale={scale}>
        <TimelineBar
          load={houston}
          scale={scale}
          top={0}
          height={20}
          caption="timing"
          selected={false}
          onSelect={() => {}}
        />
      </TimelineHoverCard>,
    );
    expect(card()).toBeNull();
  });

  it("closes as the pointer presses the bar, so a drag never carries it", () => {
    renderBars([laredo]);
    fireEvent.pointerEnter(bar("PRO-1, Laredo → Dallas"), { clientX: 400 });
    act(() => vi.advanceTimersByTime(80));

    fireEvent.pointerDown(bar("PRO-1, Laredo → Dallas"));
    expect(card()).toBeNull();
  });
});
