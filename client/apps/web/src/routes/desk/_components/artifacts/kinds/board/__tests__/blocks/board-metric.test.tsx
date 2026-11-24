// jsdom computes no layout, so a sketch and its form are compared by the heights they set and
// the rows they draw; that the two stand the same height on screen needs a browser.
import { cleanup, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BoardMetric } from "../../blocks/board-metric";
import { FormSketch } from "../../blocks/sketches";
import { MetricSketch } from "../../board-skeleton";
import {
  agingChart,
  arrivalChart,
  chart,
  count,
  coverChart,
  lateChart,
  metric,
  NOW,
  onTimeChart,
  tractorsChart,
  usd,
  weeksChart,
} from "./fixtures";

vi.mock("@trenova/shared/types/generated/board-catalog", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("@trenova/shared/types/generated/board-catalog")>();
  return {
    ...actual,
    BOARD_METRIC_SKETCH: {
      ...actual.BOARD_METRIC_SKETCH,
      onTimeByWeek: { form: "trend" },
      lateByCustomer: { form: "bars", rows: 5 },
      driverCover: { form: "grid", rows: 6 },
    },
  };
});

afterEach(cleanup);

const draw = (read: ReturnType<typeof metric>): ReactElement => (
  <BoardMetric
    metric={read}
    answeredAt={NOW - 600}
    now={NOW}
    open={vi.fn()}
    opensInPlace={() => true}
  />
);

/** The height a drawn chart's box sets, or a sketch's box. */
function boxHeight(container: HTMLElement): string {
  const box = container.querySelector<HTMLElement>(".dk-ch-box, .dk-bd-wait-box");
  return box?.style.height ?? "";
}

describe("BoardMetric", () => {
  it.each([
    ["trend", onTimeChart()],
    ["bars", lateChart()],
    ["bars", agingChart()],
    ["bars", weeksChart()],
    ["columns", arrivalChart()],
    ["dots", tractorsChart()],
    ["spans", coverChart()],
  ])("draws a %s chart as one labelled image", (form, shaped) => {
    render(draw(metric({ form, title: "Exhibit", chart: shaped })));
    const image = screen.getByRole("img");
    expect(image.tagName.toLowerCase()).toBe("svg");
    expect(image.getAttribute("aria-label")).toMatch(/^Exhibit/);
  });

  it.each(["trend", "bars", "columns", "dots", "spans"])(
    "says a %s chart with no chart, or an empty one, has nothing to draw",
    (form) => {
      render(draw(metric({ form, chart: null })));
      expect(screen.getByText("The chart has nothing to draw.")).toBeTruthy();
      cleanup();
      render(draw(metric({ form, chart: chart() })));
      expect(screen.getByText("The chart has nothing to draw.")).toBeTruthy();
      expect(screen.queryByRole("img")).toBeNull();
    },
  );

  it("says a trend whose every point is missing has nothing to draw", () => {
    const blank = onTimeChart();
    blank.series = [{ name: "On time", role: "now", points: [null, null] }];
    render(draw(metric({ form: "trend", chart: blank })));
    expect(screen.getByText("The chart has nothing to draw.")).toBeTruthy();
  });

  it("says an empty log or table or checks so, and draws figures as a readout", () => {
    render(draw(metric({ form: "log" })));
    expect(screen.getByText("Nothing happened in this period.")).toBeTruthy();
    cleanup();
    render(draw(metric({ form: "table", table: { columns: [], rows: [] } })));
    expect(screen.getByText("The table has no rows now.")).toBeTruthy();
    cleanup();
    render(draw(metric({ form: "checks" })));
    expect(screen.getByText("There is nothing to check.")).toBeTruthy();
    cleanup();
    const { container } = render(
      draw(
        metric({
          form: "figures",
          figures: [
            {
              name: "billed",
              label: "Billed",
              value: usd("9840.00"),
              was: null,
              wasAt: null,
              tone: "",
            },
            { name: "waived", label: "Waived", value: null, was: null, wasAt: null, tone: "" },
          ],
        }),
      ),
    );
    expect([...container.querySelectorAll("dt")].map((node) => node.textContent)).toEqual([
      "Billed",
      "Waived",
    ]);
    expect(screen.getByText("$9,840.00")).toBeTruthy();
    expect(screen.getByText("Could not be read")).toBeTruthy();
  });

  it("renders nothing for figures it has none of", () => {
    const { container } = render(draw(metric({ form: "figures" })));
    expect(container.innerHTML).toBe("");
  });

  it("draws a split, a grid, a log and a table in their own parts", () => {
    const { container } = render(
      draw(
        metric({
          form: "split",
          split: { total: usd(10), parts: [{ label: "Ready", value: usd(10), kind: "used" }] },
        }),
      ),
    );
    expect(container.querySelector(".dk-bd-split")).not.toBeNull();
    cleanup();
    const log = render(
      draw(
        metric({
          form: "log",
          log: [{ at: NOW, tone: "", text: "Tender in", detail: "", records: [] }],
        }),
      ),
    );
    expect(log.container.querySelectorAll(".dk-bd-mlog li")).toHaveLength(1);
    cleanup();
    const table = render(
      draw(
        metric({
          form: "table",
          table: {
            columns: [{ label: "Week", kind: "value" }],
            rows: [[{ value: count(1), text: "", mark: "" }]],
          },
        }),
      ),
    );
    expect(table.container.querySelectorAll(".dk-bd-mtbl tbody tr")).toHaveLength(1);
  });
});

describe("the sketch a form reserves", () => {
  it.each([
    ["trend", onTimeChart(), 0],
    ["bars", lateChart(), 5],
    ["bars", agingChart(), 0],
    ["bars", weeksChart(), 0],
    ["columns", arrivalChart(), 0],
    ["dots", tractorsChart(), 0],
    ["spans", coverChart(), 2],
  ])("holds a %s chart's exact height", (form, shaped, rows) => {
    const drawn = render(draw(metric({ form, chart: shaped })));
    const height = boxHeight(drawn.container);
    cleanup();
    const sketch = render(<FormSketch form={form} rows={rows} />);
    expect(height).not.toBe("");
    expect(boxHeight(sketch.container)).toBe(height);
  });

  it("draws a grid, a table, a log and a split in the same boxes as the form", () => {
    const grid = render(<FormSketch form="grid" rows={2} />);
    expect(grid.container.querySelectorAll(".dk-bd-grid > .gh-a")).toHaveLength(2);
    expect(grid.container.querySelector(".dk-bd-scroll > .dk-bd-grid")).not.toBeNull();
    cleanup();
    const table = render(<FormSketch form="table" rows={3} />);
    expect(table.container.querySelectorAll(".dk-tbl.dk-bd-mtbl tbody tr")).toHaveLength(3);
    expect(table.container.querySelector(".dk-bd-scroll.dk-bd-bleed")).not.toBeNull();
    cleanup();
    const log = render(<FormSketch form="log" rows={4} />);
    expect(log.container.querySelectorAll(".dk-bd-log.dk-bd-mlog > li")).toHaveLength(4);
    cleanup();
    const split = render(<FormSketch form="split" rows={0} />);
    expect(split.container.querySelectorAll(".dk-bd-keys > span")).toHaveLength(3);
    expect(split.container.querySelector(".dk-bd-big")).not.toBeNull();
  });

  it("reserves the catalog's form for a metric, or the rows the kept reply drew", () => {
    const trend = render(<MetricSketch metric="onTimeByWeek" drawn={0} />);
    expect(boxHeight(trend.container)).toBe("156px");
    cleanup();
    const ranked = render(<MetricSketch metric="lateByCustomer" drawn={3} />);
    expect(boxHeight(ranked.container)).toBe(`${3 * 26 - 6}px`);
    cleanup();
    const grid = render(<MetricSketch metric="driverCover" drawn={0} />);
    expect(grid.container.querySelectorAll(".dk-bd-grid > .gh-a")).toHaveLength(6);
  });
});
