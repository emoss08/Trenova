import {
  CHART_LABEL_KEY,
  CHART_X_KEY,
  analysisChartPoints,
  analysisPieSlices,
  chartHasValues,
} from "@/lib/analysis-chart";
import type { AnalysisChart } from "@/types/analysis";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";

const t = ((template: string) => template) as TranslateFn;

function chartOf(overrides: Partial<AnalysisChart> = {}): AnalysisChart {
  return {
    kind: "bar",
    title: "Open by customer",
    x: { label: "Customer", kind: "text", values: ["Acme", "Peak", null] },
    series: [
      { name: "Open", values: [36912.16, null, 120] },
      { name: "Open", values: [1, 2] },
    ],
    valueKind: "money",
    currency: "USD",
    stacked: false,
    horizontal: false,
    ...overrides,
  };
}

describe("analysisChartPoints", () => {
  it("gives each value along the axis a point, every series under its own key", () => {
    expect(analysisChartPoints(chartOf(), t)).toEqual([
      { [CHART_LABEL_KEY]: "Acme", s0: 36912.16, s1: 1 },
      { [CHART_LABEL_KEY]: "Peak", s0: null, s1: 2 },
      { [CHART_LABEL_KEY]: "—", s0: 120, s1: null },
    ]);
  });

  it("keeps a scatter point's x as a number, and none for an x that is not one", () => {
    const points = analysisChartPoints(
      chartOf({
        kind: "scatter",
        x: { label: "Miles", kind: "number", values: [120, "far"] },
        series: [{ name: "MPG", values: [6.8, 7.1] }],
      }),
      t,
    );

    expect(points.map((point) => point[CHART_X_KEY])).toEqual([120, null]);
    expect(points.map((point) => point.s0)).toEqual([6.8, 7.1]);
  });
});

describe("analysisPieSlices", () => {
  it("draws only the slices above zero and counts the rest", () => {
    const points = analysisChartPoints(
      chartOf({
        kind: "pie",
        x: { label: "", kind: "text", values: ["Acme", "Peak", "Credit", "Unknown"] },
        series: [{ name: "Share", values: [0.6, 0.4, -0.1, null] }],
      }),
      t,
    );

    const { slices, left } = analysisPieSlices(points);

    expect(slices.map((slice) => slice[CHART_LABEL_KEY])).toEqual(["Acme", "Peak"]);
    expect(left).toBe(2);
  });
});

describe("chartHasValues", () => {
  it("is false only when no series has a number", () => {
    expect(chartHasValues(chartOf())).toBe(true);
    expect(chartHasValues(chartOf({ series: [{ name: "Open", values: [null, null] }] }))).toBe(
      false,
    );
    expect(chartHasValues(chartOf({ series: [] }))).toBe(false);
  });
});
