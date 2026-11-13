import { outlook } from "@/lib/graphql/__tests__/freight-outlook-fixtures";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { BoardTide } from "../board-tide";
import { BARS_BASE, DRIVERS_LANE, LOADS_LANE, NOW_LABEL_Y, TIDE_HEIGHT } from "../board-tide-model";

const LABEL = 280;
const VALUE = 108;
const wide = { width: 1588, label: LABEL, value: VALUE, count: 12 };
const PLOT_RIGHT = wide.width - VALUE;

function renderTide(columns = wide, data = outlook()) {
  return render(
    <BoardTide outlook={data} columns={columns} horizonSlots={6} pinned={null} onPin={vi.fn()} />,
  );
}

function at(element: Element, attribute: string): number {
  return Number(element.getAttribute(attribute));
}

function textsIn(container: HTMLElement, content: string): SVGTextElement[] {
  return Array.from(container.querySelectorAll("text")).filter(
    (text) => text.textContent === content,
  );
}

describe("BoardTide", () => {
  it("names each line and each lane beside the plot, never inside it", () => {
    renderTide();

    for (const name of [
      "Network",
      "Drivers free",
      "Loads needing a driver",
      "No truck nearby",
      "Idle drivers",
    ]) {
      const text = screen.getByText(name);
      expect(at(text, "x"), name).toBeLessThan(LABEL);
    }
    for (const name of ["Drivers free", "Loads needing a driver", "No truck nearby"]) {
      expect(screen.getByText(name).getAttribute("text-anchor"), name).toBe("end");
    }
  });

  it("says what the top of each lane stands for, with its unit, at the right of the plot", () => {
    const data = outlook();
    renderTide(wide, {
      ...data,
      network: data.network.map((slot) => ({
        ...slot,
        need: 2,
        needLow: 0,
        needHigh: 5,
        drivers: 148,
      })),
    });

    const loadsTop = screen.getByText("5 loads");
    const driversTop = screen.getByText("150 drivers");
    expect(at(loadsTop, "x")).toBeGreaterThan(PLOT_RIGHT);
    expect(at(loadsTop, "y")).toBe(LOADS_LANE.top + 3.5);
    expect(at(driversTop, "x")).toBeGreaterThan(PLOT_RIGHT);
    expect(at(driversTop, "y")).toBe(DRIVERS_LANE.top + 3.5);
  });

  it("writes no bare numbers beside the plot", () => {
    const { container } = renderTide();

    const beside = Array.from(container.querySelectorAll("text")).filter(
      (text) => at(text, "x") > PLOT_RIGHT,
    );

    expect(beside.map((text) => text.textContent)).toEqual(["30 loads", "30 drivers"]);
  });

  it("counts a single load and a single driver in the singular", () => {
    const data = outlook();
    renderTide(wide, {
      ...data,
      network: data.network.map((slot) => ({
        ...slot,
        need: 0,
        needLow: 0,
        needHigh: 0.4,
        drivers: 1,
      })),
    });

    expect(screen.getByText("1 load")).toBeInTheDocument();
    expect(screen.getByText("1 driver")).toBeInTheDocument();
  });

  it("shows a few loads under many drivers two fifths of the way up their own lane, not flat", () => {
    const data = outlook();
    const { container } = renderTide(wide, {
      ...data,
      network: data.network.map((slot) => ({
        ...slot,
        need: 2,
        needLow: 0,
        needHigh: 5,
        drivers: 148,
      })),
    });

    const [loads, drivers] = Array.from(container.querySelectorAll("path.fill-none")).map((line) =>
      Number(line.getAttribute("d")?.match(/^M280,([\d.]+)/)?.[1]),
    );
    const loadsLane = LOADS_LANE.bottom - LOADS_LANE.top;
    const driversLane = DRIVERS_LANE.bottom - DRIVERS_LANE.top;

    expect(loads).toBeCloseTo(LOADS_LANE.bottom - (loadsLane * 2) / 5);
    expect(drivers).toBeCloseTo(DRIVERS_LANE.bottom - (driversLane * 148) / 150);
  });

  it("names each line where it starts, in its own lane", () => {
    renderTide();

    const loads = at(screen.getByText("Loads needing a driver"), "y");
    const drivers = at(screen.getByText("Drivers free"), "y");

    expect(loads).toBeGreaterThanOrEqual(LOADS_LANE.top);
    expect(loads).toBeLessThanOrEqual(LOADS_LANE.bottom + 3.5);
    expect(drivers).toBeGreaterThanOrEqual(DRIVERS_LANE.top);
    expect(drivers).toBeLessThanOrEqual(DRIVERS_LANE.bottom + 3.5);
    expect(at(screen.getByText("Drivers free"), "x")).toBe(
      at(screen.getByText("Idle drivers"), "x"),
    );
  });

  it("shades a window where the loads outrun the drivers, across both lanes", () => {
    const data = outlook();
    const { container } = renderTide(wide, {
      ...data,
      network: data.network.map((slot, index) =>
        index === 2 ? { ...slot, short: 3 } : { ...slot, short: 0.2 },
      ),
    });

    const shaded = Array.from(container.querySelectorAll("rect")).filter((rect) =>
      rect.getAttribute("class")?.includes("fill-danger"),
    );

    expect(shaded).toHaveLength(1);
    expect(at(shaded[0], "x")).toBe(LABEL + 200);
    expect(at(shaded[0], "width")).toBe(100);
    expect(at(shaded[0], "y")).toBeLessThan(LOADS_LANE.top);
    expect(at(shaded[0], "y") + at(shaded[0], "height")).toBeGreaterThan(DRIVERS_LANE.bottom);
    expect(at(shaded[0], "y") + at(shaded[0], "height")).toBeLessThan(BARS_BASE);
  });

  it("draws both lines from one edge of the plot to the other", () => {
    const { container } = renderTide();

    const lines = Array.from(container.querySelectorAll("path.fill-none"));
    expect(lines).toHaveLength(2);
    for (const line of lines) {
      const path = line.getAttribute("d") ?? "";
      expect(path.startsWith(`M${LABEL},`)).toBe(true);
      expect(path).toMatch(new RegExp(`[, ]${PLOT_RIGHT},[\\d.]+$`));
    }
  });

  it("puts the time now on a strip of its own, above the lines and after its line", () => {
    renderTide();

    const now = screen.getByText(/^Now /);

    expect(at(now, "y")).toBe(NOW_LABEL_Y);
    expect(at(now, "y")).toBeLessThan(LOADS_LANE.top);
    expect(at(now, "x")).toBe(LABEL);
    expect(now.getAttribute("text-anchor")).toBe("start");
  });

  it("writes each bar's count on it, loads above the line and drivers below", () => {
    const { container } = renderTide();

    const short = textsIn(container, "3");
    const idle = textsIn(container, "4");

    expect(short).toHaveLength(12);
    expect(idle).toHaveLength(12);
    expect(at(short[0], "x")).toBe(LABEL + 50);
    expect(at(short[0], "y")).toBeLessThan(at(idle[0], "y"));
    expect(at(short[0], "y")).toBeGreaterThan(DRIVERS_LANE.bottom);
    expect(at(idle[0], "y")).toBeLessThan(TIDE_HEIGHT);
  });

  it("draws no bar and no count for a window with nothing short and nobody idle", () => {
    const data = outlook();
    const { container } = renderTide(wide, {
      ...data,
      network: data.network.map((slot, index) =>
        index === 0 ? slot : { ...slot, uncovered: 0, idle: 0.3 },
      ),
    });

    expect(textsIn(container, "3")).toHaveLength(1);
    expect(textsIn(container, "4")).toHaveLength(1);
    expect(textsIn(container, "0")).toHaveLength(0);
    expect(container.querySelectorAll("rect.fill-warning, rect.fill-accent-sky")).toHaveLength(2);
  });

  it("leaves the counts off bars too narrow to carry them", () => {
    const { container } = renderTide({ ...wide, width: LABEL + VALUE + 12 * 20 });

    expect(textsIn(container, "3")).toHaveLength(0);
    expect(textsIn(container, "4")).toHaveLength(0);
    expect(container.querySelectorAll("rect.fill-warning")).toHaveLength(12);
  });

  it("holds its height while the board has not been measured", () => {
    const { container } = renderTide({ ...wide, width: 0 });

    expect(container.querySelector("svg")).toBeNull();
    expect(container.firstElementChild).toHaveStyle({ height: `${TIDE_HEIGHT}px` });
  });
});
