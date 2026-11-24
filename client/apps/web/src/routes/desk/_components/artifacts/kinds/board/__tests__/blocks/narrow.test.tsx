// jsdom measures nothing, so the chart's width is set by mocking the hook that measures it, and
// collisions are checked with the same estimated text widths the layout uses. Whether those
// estimates hold for the real fonts (and nothing overlaps on screen) needs a browser.
import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BarsChart } from "../../blocks/chart-bars";
import { ColumnsChart } from "../../blocks/chart-columns";
import { DotsChart } from "../../blocks/chart-dots";
import {
  overlaps,
  textBox,
  type Anchor,
  type Box,
  type TextClass,
} from "../../blocks/chart-labels";
import { SpansChart } from "../../blocks/chart-spans";
import { TrendChart } from "../../blocks/chart-trend";
import type { DeskBoardChart } from "../../blocks/metric-types";
import {
  agingChart,
  arrivalChart,
  chart,
  chartRow,
  count,
  DAY,
  lateChart,
  NOW,
  onTimeChart,
  series,
  tick,
  usd,
  weeksChart,
} from "./fixtures";

const measured = vi.hoisted(() => ({ width: 349 }));
vi.mock("@trenova/shared/hooks/use-element-width", () => ({
  useElementWidth: () => [() => {}, measured.width],
}));

afterEach(cleanup);

/** Each text's own words (not its `<title>`), class and box, as the layout estimates it. */
function textBoxes(container: HTMLElement): { text: string; box: Box }[] {
  return [...container.querySelectorAll("svg text")].map((node) => {
    const text = [...node.childNodes]
      .filter((child) => child.nodeType === Node.TEXT_NODE)
      .map((child) => child.textContent)
      .join("");
    const cls = (node.getAttribute("class") ?? "tk").split(" ")[0] as TextClass;
    const anchor = (node.getAttribute("text-anchor") ?? "start") as Anchor;
    const box = textBox(text, cls, {
      x: Number(node.getAttribute("x")),
      y: Number(node.getAttribute("y")),
      anchor,
    });
    return { text, box };
  });
}

/** No two texts overlap and none leaves the chart's width. */
function expectNoCollisions(container: HTMLElement, width: number) {
  const boxes = textBoxes(container);
  for (const { text, box } of boxes) {
    expect({ text, inside: box.x0 >= -0.5 && box.x1 <= width + 0.5 }).toEqual({
      text,
      inside: true,
    });
  }
  for (let a = 0; a < boxes.length; a += 1) {
    for (let b = a + 1; b < boxes.length; b += 1) {
      expect({
        pair: [boxes[a].text, boxes[b].text],
        overlap: overlaps(boxes[a].box, boxes[b].box),
      }).toEqual({ pair: [boxes[a].text, boxes[b].text], overlap: false });
    }
  }
}

/** Sixty tractors, every fourth marked, named by their unit numbers, around a fleet line. */
function crowdedTractors(): DeskBoardChart {
  return chart({
    unit: "money",
    currency: "USD",
    rows: Array.from({ length: 60 }, (_, index) => {
      const cost = (0.48 + ((index * 37) % 61) / 1000).toFixed(3);
      const marked = index % 4 === 0;
      const unit = String(1092 + index * 7);
      return chartRow({
        label: unit,
        value: usd(cost),
        marked,
        dots: [{ at: null, value: cost, marked, ghost: index === 59, label: unit }],
      });
    }),
    band: { from: null, to: null, low: "0.489", high: "0.597" },
    ref: { value: "0.543", label: "Fleet" },
  });
}

function crowdedCover(): DeskBoardChart {
  const names = [
    "Ridgeline Freight Systems",
    "Copperhead Transport Logistics",
    "Blue Line Freight",
    "Sandoval Trucking Company",
    "Great Plains Refrigerated",
    "Northwoods Carrier Services",
  ];
  return chart({
    unit: "count",
    ticks: [2, 6, 10, 14, 18, 22, 26, 29, 31].map((days, index) => ({
      at: NOW + days * DAY,
      label: "",
      marked: index === 7,
    })),
    rows: names.map((label, index) =>
      chartRow({
        label,
        spans: [{ from: NOW - 60 * DAY, to: NOW + (6 + index * 4) * DAY, kind: "cover" }],
        dots:
          index === 0
            ? [
                {
                  at: NOW + 9 * DAY,
                  value: null,
                  marked: true,
                  ghost: false,
                  label: "SHP-CAR-000123 Dec 2",
                },
              ]
            : [],
      }),
    ),
  });
}

function crowdedRanked(): DeskBoardChart {
  return chart({
    unit: "count",
    rows: [
      chartRow({ label: "Rodriguez & Evans Distribution", value: count(14), marked: true }),
      chartRow({ label: "Harbor Paper Company of Louisiana", value: count(7) }),
      chartRow({ label: "38 other customers", value: count(71), rest: true }),
    ],
  });
}

function fuelRun(): DeskBoardChart {
  const run = Array.from({ length: 22 }, (_, index) => 0.47 + index * 0.001);
  return chart({
    unit: "money",
    currency: "USD",
    ticks: run.map((_, index) => tick(`Nov ${index + 1}`)),
    series: [series("Run", "now", [...run.slice(0, 21), 0.492])],
    ref: { value: "0.488", label: "October" },
  });
}

describe.each([300, 349, 606])("every chart at %ipx", (width) => {
  it.each([
    ["trend", () => <TrendChart chart={onTimeChart()} title="On time" />],
    ["trend, a long run", () => <TrendChart chart={fuelRun()} title="Fuel" />],
    ["bars, ranked", () => <BarsChart chart={lateChart()} title="Late" />],
    ["bars, ranked with long names", () => <BarsChart chart={crowdedRanked()} title="Late" />],
    ["bars, period on period", () => <BarsChart chart={agingChart()} title="Aging" />],
    ["bars, stacked", () => <BarsChart chart={weeksChart()} title="Weeks" />],
    ["columns", () => <ColumnsChart chart={arrivalChart()} title="Arrival" />],
    ["dots, crowded", () => <DotsChart chart={crowdedTractors()} title="Tractors" />],
    ["spans, crowded", () => <SpansChart chart={crowdedCover()} title="Cover" now={NOW} />],
  ])("draws %s with no text over another or out of the chart", (_, draw) => {
    measured.width = width;
    const { container } = render(draw());
    expect(container.querySelector("svg")?.getAttribute("width")).toBe(String(width));
    expectNoCollisions(container, width);
  });
});

describe("crowded charts", () => {
  it("names at most four marked tractors, the furthest from the line first, and keeps every dot in the band", () => {
    measured.width = 349;
    const { container } = render(<DotsChart chart={crowdedTractors()} title="Tractors" />);
    const named = [...container.querySelectorAll("text.lb.w, text.lb:not(.w)")].filter(
      (node) => !node.textContent?.startsWith("Fleet"),
    );
    expect(named.length).toBeLessThanOrEqual(4);
    expect(container.querySelectorAll("circle.dot.w")).toHaveLength(15);
    expect(container.querySelectorAll("circle.dot.gh")).toHaveLength(1);
    for (const dot of container.querySelectorAll("circle")) {
      const cy = Number(dot.getAttribute("cy"));
      const cx = Number(dot.getAttribute("cx"));
      expect(cy).toBeGreaterThanOrEqual(12);
      expect(cy).toBeLessThanOrEqual(78);
      expect(cx).toBeGreaterThanOrEqual(40);
      expect(cx).toBeLessThanOrEqual(349 - 42);
    }
    expect(container.querySelector("rect.band.q")).not.toBeNull();
  });

  it("sets a load's name clear of the row names and keeps the day its cover ends on the axis", () => {
    measured.width = 349;
    const { container } = render(<SpansChart chart={crowdedCover()} title="Cover" now={NOW} />);
    const names = [...container.querySelectorAll("text.nm")];
    const column = Math.max(
      ...names.map(
        (node) =>
          textBox(node.childNodes[node.childNodes.length - 1]?.textContent ?? "", "nm", {
            x: 0,
            y: 0,
            anchor: "start",
          }).x1,
      ),
    );
    const label = container.querySelector("text.lb.w");
    if (label) {
      const box = textBox(label.textContent ?? "", "lb", {
        x: Number(label.getAttribute("x")),
        y: Number(label.getAttribute("y")),
        anchor: (label.getAttribute("text-anchor") ?? "start") as Anchor,
      });
      expect(box.x0).toBeGreaterThan(column);
    }
    const kept = [...container.querySelectorAll("text.tk")].map((node) => node.textContent);
    const boundDay = new Date((NOW + 29 * DAY) * 1000).toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
    });
    expect(kept).toContain(boundDay);
  });

  it("draws the catch-all row light and gives long names the room the bars can spare", () => {
    measured.width = 606;
    const { container } = render(<BarsChart chart={crowdedRanked()} title="Late" />);
    const names = [...container.querySelectorAll("text.nm")];
    expect(names[0].textContent).toBe("Rodriguez & Evans Distribution");
    expect(names[2].getAttribute("class")).toBe("nm q");
    expect(container.querySelectorAll("rect.bar.was")).toHaveLength(1);
    cleanup();
    measured.width = 300;
    const narrow = render(<BarsChart chart={crowdedRanked()} title="Late" />);
    const cut = narrow.container.querySelector("text.nm");
    expect(cut?.querySelector("title")?.textContent).toBe("Rodriguez & Evans Distribution");
  });

  it("keeps the latest point's value off the line", () => {
    measured.width = 349;
    const { container } = render(<TrendChart chart={fuelRun()} title="Fuel" />);
    const latest = [...container.querySelectorAll("text.lb")].find(
      (node) => node.textContent === "$0.492",
    );
    expect(latest).toBeDefined();
    const line = container.querySelector("polyline.ln");
    const points = (line?.getAttribute("points") ?? "")
      .split(" ")
      .map((pair) => pair.split(",").map(Number));
    const box = textBox("$0.492", "lb", {
      x: Number(latest?.getAttribute("x")),
      y: Number(latest?.getAttribute("y")),
      anchor: (latest?.getAttribute("text-anchor") ?? "start") as Anchor,
    });
    for (const [x, y] of points) {
      expect(x >= box.x0 && x <= box.x1 && y >= box.y0 && y <= box.y1).toBe(false);
    }
  });
});
