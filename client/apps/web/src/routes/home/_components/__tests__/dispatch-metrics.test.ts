import { describe, expect, it } from "vitest";
import { defaultAnalyticsData } from "@/lib/shipment-analytics";
import { resolveMetric } from "../metric-values";
import type { HomeData } from "../use-home-data";

function homeData(dispatch: (typeof defaultAnalyticsData)["dispatch"]): HomeData {
  return {
    shipmentAnalytics: { ...defaultAnalyticsData, dispatch },
    shipmentAnalyticsLoading: false,
    shipmentAnalyticsReady: true,
    attention: undefined,
    attentionLoading: false,
    receivables: undefined,
    receivablesLoading: false,
  };
}

/*
 * The dispatch board's counts of the next day's moves are KPI tiles too, read
 * from the same summary the board and an agent's board figures read, and each
 * opens the dispatch console.
 */
describe("dispatch metrics", () => {
  const data = homeData({ moves: 41, late: 7, dueNow: 3, dueToday: 12, uncovered: 9 });

  it("draws each count on its own tile, linked to the dispatch console", () => {
    expect(
      ["movesLate", "movesDueNow", "movesDueToday", "movesUncovered"].map((key) => {
        const metric = resolveMetric(key, data);
        return [metric?.label, metric?.display, metric?.href];
      }),
    ).toEqual([
      ["Late moves", "7", "/dispatch/console"],
      ["Moves due now", "3", "/dispatch/console"],
      ["Moves due today", "12", "/dispatch/console"],
      ["Uncovered moves", "9", "/dispatch/console"],
    ]);
  });

  it("says how many moves the late count is out of", () => {
    expect(resolveMetric("movesLate", data)?.sub).toBe("of 41 moves in the next day");
  });

  it("waits rather than drawing a count before the figures arrive", () => {
    expect(
      resolveMetric("movesLate", { ...data, shipmentAnalyticsLoading: true }),
    ).toBeNull();
  });
});
