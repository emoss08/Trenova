// Pure helpers: nothing here needs a browser.
import { describe, expect, it } from "vitest";
import {
  chartValue,
  fitText,
  laneOf,
  niceTicks,
  signedCount,
  sparseIndexes,
  tickValue,
  toNumber,
} from "../../blocks/chart-values";

describe("chart values", () => {
  it("reads a decimal string and refuses what is not a number", () => {
    expect(toNumber("0.638")).toBe(0.638);
    expect(toNumber(null)).toBeNull();
    expect(toNumber(undefined)).toBeNull();
    expect(toNumber("")).toBeNull();
    expect(toNumber("n/a")).toBeNull();
  });

  it("writes money whole above a thousand and to the tenth of a cent below a dollar", () => {
    expect(chartValue(412300, "money", "USD")).toBe("$412,300");
    expect(chartValue(0.781, "money", "USD")).toBe("$0.781");
    expect(chartValue(12.5, "money", "USD")).toBe("$12.50");
    expect(chartValue(0.882, "percent", "")).toBe("88.2%");
    expect(chartValue(15600, "duration", "")).toBe("4h 20m");
    expect(chartValue(-3600, "duration", "")).toBe("−1h");
  });

  it("writes ticks with no more digits than their step needs", () => {
    expect(tickValue(0.6, 0.1, "money", "USD")).toBe("$0.60");
    expect(tickValue(400000, 100000, "money", "USD")).toBe("$400,000");
    expect(tickValue(0.9, 0.05, "percent", "")).toBe("90%");
  });

  it("finds a few round ticks that hold every value", () => {
    const { ticks, step } = niceTicks(0.548, 0.781, 4);
    expect(step).toBe(0.1);
    expect(ticks.map((tick) => Math.round(tick * 100) / 100)).toEqual([0.5, 0.6, 0.7, 0.8]);
    expect(niceTicks(5, 5, 3).ticks.length).toBeGreaterThan(1);
  });

  it("signs a balance with a real minus", () => {
    expect([signedCount(2), signedCount(-4), signedCount(0)]).toEqual(["+2", "−4", "0"]);
  });

  it("labels the first point and every few after it", () => {
    expect(sparseIndexes(9, 5)).toEqual([0, 2, 4, 6, 8]);
    expect(sparseIndexes(3, 5)).toEqual([0, 1, 2]);
    expect(sparseIndexes(0, 5)).toEqual([]);
  });

  it("cuts a name to its room with an ellipsis", () => {
    expect(fitText("Northline Grocers", 200, 6.6)).toBe("Northline Grocers");
    expect(fitText("Northline Grocers", 40, 6.6)).toBe("North…");
  });

  it("stacks dots at the same value in lanes, a crowd spread over the lanes it has", () => {
    expect(laneOf([100, 100, 100], 10)).toEqual([47, 36, 58]);
    expect(laneOf([100, 300], 10)).toEqual([47, 47]);
    expect(laneOf([100, 100, 100], 10, [47, 36])).toEqual([47, 36, 47]);
  });
});
