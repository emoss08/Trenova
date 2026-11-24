// jsdom draws no layout: these hold the marks, classes, labels and heights; how the charts sit
// at a real width (text never clipped, lanes clear of each other) needs a browser.
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { BarsChart } from "../../blocks/chart-bars";
import { ColumnsChart } from "../../blocks/chart-columns";
import { DotsChart } from "../../blocks/chart-dots";
import { SpansChart } from "../../blocks/chart-spans";
import { TrendChart } from "../../blocks/chart-trend";
import {
  agingChart,
  arrivalChart,
  chart,
  chartRow,
  count,
  coverChart,
  lateChart,
  NOW,
  onTimeChart,
  series,
  tick,
  tractorsChart,
  weeksChart,
} from "./fixtures";

afterEach(cleanup);

const texts = (container: HTMLElement, selector: string) =>
  [...container.querySelectorAll(selector)].map((node) => node.textContent);

describe("TrendChart", () => {
  it("draws the slip marked in a band, the target and this week as a ghost, and says the headline", () => {
    const { container } = render(
      <TrendChart chart={onTimeChart()} title="On time by week, this quarter" />,
    );
    const svg = screen.getByRole("img");
    expect(svg.getAttribute("aria-label")).toBe(
      "On time by week, this quarter, Sep 28 to Nov 16: latest 92.4%; lowest 88.2%, Nov 2; 95% so far in the period still running",
    );
    expect(svg.getAttribute("height")).toBe("156");
    expect(container.querySelectorAll("circle.dot.w")).toHaveLength(1);
    expect(container.querySelectorAll("circle.dot.gh")).toHaveLength(1);
    expect(container.querySelectorAll("line.ln.gh")).toHaveLength(1);
    expect(container.querySelectorAll("rect.band")).toHaveLength(1);
    expect(texts(container, "text.lb.w")).toEqual(["Nov 2", "88.2%"]);
    expect(texts(container, "text.lb:not(.w)")).toEqual(["92.4%"]);
    expect(container.querySelector("line.tgt")).not.toBeNull();
    expect(texts(container, "text.tk")).toContain("Target 92%");
  });

  it("breaks the line where a point is missing and draws one point alone", () => {
    const gapped = chart({
      unit: "percent",
      ticks: [tick("W1"), tick("W2"), tick("W3")],
      series: [series("On time", "now", [0.9, null, 0.92])],
    });
    const { container } = render(<TrendChart chart={gapped} title="On time" />);
    expect(container.querySelectorAll("polyline.ln")).toHaveLength(2);
    cleanup();

    const single = chart({
      unit: "percent",
      ticks: [tick("W1")],
      series: [series("On time", "now", [0.9])],
    });
    render(<TrendChart chart={single} title="On time" />);
    expect(screen.getByRole("img").getAttribute("aria-label")).toBe("On time: 90%");
  });

  it("draws an earlier series lighter than the series now", () => {
    const two = chart({
      unit: "count",
      ticks: [tick("A"), tick("B")],
      series: [series("Last year", "was", [3, 4]), series("This year", "now", [5, 6])],
    });
    const { container } = render(<TrendChart chart={two} title="Loads" />);
    expect(container.querySelectorAll("polyline.ln.was")).toHaveLength(1);
    expect(container.querySelectorAll("polyline.ln:not(.was)")).toHaveLength(1);
  });
});

describe("BarsChart", () => {
  it("ranks rows with the marked one warn and its count in the label", () => {
    const { container } = render(<BarsChart chart={lateChart()} title="Late by customer" />);
    const svg = screen.getByRole("img");
    expect(svg.getAttribute("aria-label")).toBe(
      "Late by customer: Northline Grocers at 17, the most of 5 rows",
    );
    expect(svg.getAttribute("height")).toBe(String(5 * 26 - 6));
    expect(container.querySelectorAll("rect.bar")).toHaveLength(5);
    expect(container.querySelectorAll("rect.bar.w")).toHaveLength(1);
    expect(texts(container, "text.nm")).toEqual([
      "Northline Grocers",
      "Acme Foods",
      "Harbor Paper Co.",
      "Delta Tile Supply",
      "3 other customers",
    ]);
  });

  it("says a withheld row is not the reader's instead of drawing it as zero", () => {
    const withheld = lateChart();
    withheld.rows[1] = chartRow({ label: "Acme Foods", value: null });
    const { container } = render(<BarsChart chart={withheld} title="Late by customer" />);
    expect(container.querySelectorAll("rect.bar")).toHaveLength(4);
    expect(texts(container, "text.tk")).toEqual(["Not yours to read"]);
  });

  it("draws a negative value left of zero, its value written before it", () => {
    const centred = chart({
      unit: "count",
      rows: [
        chartRow({ label: "Gained", value: count(10) }),
        chartRow({ label: "Lost", value: count(-5) }),
      ],
    });
    const { container } = render(<BarsChart chart={centred} title="Change" />);
    const [gained, lost] = [...container.querySelectorAll("rect.bar")].map((bar) =>
      Number(bar.getAttribute("x")),
    );
    expect(lost).toBeLessThan(gained);
    const label = [...container.querySelectorAll("text.lb")].find(
      (node) => node.textContent === "-5" || node.textContent === "−5",
    );
    expect(label?.getAttribute("text-anchor")).toBe("end");
  });

  it("sets last month light beside this month, the bucket that grew warn with both values", () => {
    const { container } = render(<BarsChart chart={agingChart()} title="Aging" />);
    const svg = screen.getByRole("img");
    expect(svg.getAttribute("aria-label")).toBe("Aging: 31–60 from $41,650 to $74,920");
    expect(svg.getAttribute("height")).toBe("166");
    expect(container.querySelectorAll("rect.bar.was")).toHaveLength(5);
    expect(container.querySelectorAll("rect.bar.w")).toHaveLength(1);
    expect(texts(container, "text.lb.w")).toEqual(["Nov $74,920", "31–60"]);
    expect(texts(container, "text.tk")).toContain("Oct $41,650");
  });

  it("stacks the batches with the emphasised week dark, its total, and the series named beside", () => {
    const { container } = render(<BarsChart chart={weeksChart()} title="Five weeks" />);
    expect(screen.getByRole("img").getAttribute("aria-label")).toBe(
      "Five weeks: $142,860 at Nov 16",
    );
    expect(container.querySelectorAll("rect.bar.now")).toHaveLength(1);
    expect(container.querySelectorAll("rect.bar.was")).toHaveLength(5);
    expect(texts(container, "text.lb")).toEqual(["$142,860", "Nov 16"]);
    expect(texts(container, "text.tk")).toEqual(
      expect.arrayContaining(["net", "deductions", "Oct 19"]),
    );
  });
});

describe("ColumnsChart", () => {
  it("draws each column solid to the allowance and hatched past it, the longest named", () => {
    const { container } = render(<ColumnsChart chart={arrivalChart()} title="Romeo, MI" />);
    const svg = screen.getByRole("img");
    expect(svg.getAttribute("aria-label")).toBe(
      "Romeo, MI: the most 4h 20m, 8 AM, against 2h allowed",
    );
    expect(svg.getAttribute("height")).toBe("134");
    expect(container.querySelectorAll("rect.bar")).toHaveLength(5);
    const hatched = [...container.querySelectorAll("rect")].filter((rect) =>
      rect.getAttribute("fill")?.startsWith("url(#"),
    );
    expect(hatched).toHaveLength(5);
    expect(container.querySelector("pattern")?.id).toBe(
      hatched[0].getAttribute("fill")?.slice(5, -1),
    );
    expect(texts(container, "text.lb.w")).toEqual(["4h 20m", "8 AM"]);
    expect(texts(container, "text.tk")).toContain("Free 2h");
  });

  it("leaves a missing column out and hatches nothing under the allowance", () => {
    const { container } = render(
      <ColumnsChart chart={arrivalChart([3600, null, 15600, 7200, 5400])} title="Romeo, MI" />,
    );
    expect(container.querySelectorAll("rect.bar")).toHaveLength(4);
    expect(
      [...container.querySelectorAll("rect")].filter((rect) => rect.getAttribute("fill")),
    ).toHaveLength(1);
  });
});

describe("DotsChart", () => {
  it("draws every tractor, the outliers marked and named, against the fleet's line and band", () => {
    const { container } = render(<DotsChart chart={tractorsChart()} title="Each tractor" />);
    const svg = screen.getByRole("img");
    expect(svg.getAttribute("aria-label")).toBe(
      "Each tractor: 8 dots against Fleet $0.638; marked T-118 $0.781, T-104 $0.742",
    );
    expect(svg.getAttribute("height")).toBe("106");
    expect(container.querySelectorAll("circle.dot.q")).toHaveLength(6);
    expect(container.querySelectorAll("circle.dot.w")).toHaveLength(2);
    expect(texts(container, "text.lb.w")).toEqual(["T-118", "T-104"]);
    expect(texts(container, "text.lb:not(.w)")).toEqual(["Fleet $0.638"]);
    expect(container.querySelectorAll("rect.band.q")).toHaveLength(1);
  });
});

describe("SpansChart", () => {
  it("draws each cover to its end, the mark where it ends, and the load past it named", () => {
    const { container } = render(<SpansChart chart={coverChart()} title="Cover" now={NOW} />);
    const svg = screen.getByRole("img");
    expect(svg.getAttribute("aria-label")).toBe(
      "Cover: 2 rows; 1 past its bound: S26111040188306 delivers Dec 1",
    );
    expect(svg.getAttribute("height")).toBe(String(2 * 34 + 30));
    expect(container.querySelectorAll("line.cov")).toHaveLength(2);
    expect(container.querySelectorAll("line.bnd")).toHaveLength(2);
    expect(container.querySelectorAll("line.unc")).toHaveLength(2);
    expect(container.querySelectorAll("circle.dot.w")).toHaveLength(1);
    expect(texts(container, "text.lb.w")).toEqual(["S26111040188306 delivers Dec 1"]);
    expect(texts(container, "text.lb:not(.w)")).toEqual(["Today"]);
  });
});
