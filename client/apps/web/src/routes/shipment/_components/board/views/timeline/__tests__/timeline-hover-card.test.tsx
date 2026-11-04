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
/** The card is 288px wide and sits right of and below the pointer. */
const CARD_WIDTH = 288;
const BESIDE_X = 14;
const BESIDE_Y = 18;

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

  it("sits beside the pointer and follows it in both directions over the bar", () => {
    renderBars([laredo]);
    const target = bar("PRO-1, Laredo → Dallas");
    fireEvent.pointerEnter(target, { clientX: 400, clientY: 200 });
    act(() => vi.advanceTimersByTime(80));
    expect(card()?.style.left).toBe(`${400 + BESIDE_X}px`);
    expect(card()?.style.top).toBe(`${200 + BESIDE_Y}px`);

    fireEvent.pointerMove(target, { clientX: 600, clientY: 210 });
    act(() => vi.advanceTimersToNextFrame());
    expect(card()?.style.left).toBe(`${600 + BESIDE_X}px`);
    expect(card()?.style.top).toBe(`${210 + BESIDE_Y}px`);
  });

  it("moves to the other side of the pointer where the window ends", () => {
    renderBars([laredo]);
    const x = window.innerWidth - 20;
    const y = window.innerHeight - 4;
    fireEvent.pointerEnter(bar("PRO-1, Laredo → Dallas"), { clientX: x, clientY: y });
    act(() => vi.advanceTimersByTime(80));

    expect(card()?.style.left).toBe(`${x - BESIDE_X - CARD_WIDTH}px`);
    expect(card()?.style.top).toBe(`${y - BESIDE_X}px`);
  });

  it("stays inside the window when the point it opens at lies past it", () => {
    renderBars([laredo]);
    fireEvent.pointerEnter(bar("PRO-1, Laredo → Dallas"), {
      clientX: window.innerWidth + 900,
      clientY: 200,
    });
    act(() => vi.advanceTimersByTime(80));

    expect(card()?.style.left).toBe(`${window.innerWidth - CARD_WIDTH - 8}px`);
  });

  it("opens where the pointer is when it moved while the card was waiting", () => {
    renderBars([laredo]);
    const target = bar("PRO-1, Laredo → Dallas");
    fireEvent.pointerEnter(target, { clientX: 300, clientY: 100 });
    fireEvent.pointerMove(target, { clientX: 500, clientY: 120 });
    act(() => vi.advanceTimersByTime(80));

    expect(card()?.style.left).toBe(`${500 + BESIDE_X}px`);
    expect(card()?.style.top).toBe(`${120 + BESIDE_Y}px`);
  });

  it("switches to the next bar at once while open", () => {
    renderBars([laredo, houston]);
    fireEvent.pointerEnter(bar("PRO-1, Laredo → Dallas"), { clientX: 400 });
    act(() => vi.advanceTimersByTime(80));

    fireEvent.pointerLeave(bar("PRO-1, Laredo → Dallas"));
    fireEvent.pointerEnter(bar("PRO-2, Houston → Austin"), { clientX: 700, clientY: 300 });

    expect(card()?.textContent).toContain("PRO-2");
    expect(card()?.style.left).toBe(`${700 + BESIDE_X}px`);
    expect(card()?.style.top).toBe(`${300 + BESIDE_Y}px`);
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
