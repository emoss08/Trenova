import { describe, expect, it } from "vitest";
import {
  bucketIndexOf,
  bucketRange,
  entriesByDay,
  isOffHour,
  localHour,
  niceStep,
  percentChange,
} from "../audit-overview-model";

const CHICAGO = "America/Chicago";

function bucket(at: number) {
  return { at, people: 0, agents: 0, integrations: 0, jobs: 0 };
}

describe("niceStep", () => {
  it("puts three to five gridlines under the tallest stack", () => {
    expect(niceStep(620)).toBe(200);
    expect(niceStep(90)).toBe(50);
    expect(niceStep(7)).toBe(2);
    expect(niceStep(0)).toBe(1);
  });
});

describe("bucketRange", () => {
  const hours = [0, 3600, 7200, 10_800].map(bucket);

  it("covers the dragged buckets in either direction, ending where the next one starts", () => {
    expect(bucketRange(hours, 2, 1, "Hour")).toEqual({ from: 3600, to: 10_799 });
  });

  it("gives the last bucket its own length", () => {
    expect(bucketRange(hours, 3, 3, "Hour")).toEqual({ from: 10_800, to: 14_399 });
  });

  it("keeps a day's 25 hours when the clocks fall back", () => {
    const nov1 = Date.UTC(2026, 10, 1, 5) / 1000;
    const nov2 = Date.UTC(2026, 10, 2, 6) / 1000;
    expect(bucketRange([bucket(nov1), bucket(nov2)], 0, 0, "Day")).toEqual({
      from: nov1,
      to: nov2 - 1,
    });
  });
});

describe("bucketIndexOf", () => {
  it("finds the bucket an instant falls in, and none before the first", () => {
    const hours = [100, 200, 300].map(bucket);
    expect(bucketIndexOf(hours, 250)).toBe(1);
    expect(bucketIndexOf(hours, 300)).toBe(2);
    expect(bucketIndexOf(hours, 99)).toBe(-1);
  });
});

describe("off hours", () => {
  it("reads the hour in the organization's zone", () => {
    const ninePmChicago = Date.UTC(2026, 10, 6, 3) / 1000;
    expect(localHour(ninePmChicago, CHICAGO)).toBe(21);
    expect(isOffHour(ninePmChicago, CHICAGO)).toBe(true);
  });

  it("counts 6 AM as working hours and 8 PM as off hours, as the server does", () => {
    expect(isOffHour(Date.UTC(2026, 10, 6, 12) / 1000, CHICAGO)).toBe(false);
    expect(isOffHour(Date.UTC(2026, 10, 7, 2) / 1000, CHICAGO)).toBe(true);
  });
});

describe("percentChange", () => {
  it("compares to the window before, or says nothing when there was none", () => {
    expect(percentChange(48_212, 42_980)).toBe(12);
    expect(percentChange(10, 0)).toBeNull();
  });
});

describe("entriesByDay", () => {
  const HOUR = 3600;
  const nov5 = Date.UTC(2026, 10, 5, 6) / 1000;
  const hourly = (from: number, count: number) =>
    Array.from({ length: count }, (_, i) => ({ ...bucket(from + i * HOUR), people: 1, jobs: 2 }));

  it("sums every series of each whole local day, and today so far", () => {
    expect(entriesByDay(hourly(nov5, 26), CHICAGO)).toEqual(
      new Map([
        ["2026-11-05", 72],
        ["2026-11-06", 6],
      ]),
    );
  });

  it("leaves out a day the window starts partway through", () => {
    expect(entriesByDay(hourly(nov5 + 6 * HOUR, 20), CHICAGO)).toEqual(
      new Map([["2026-11-06", 6]]),
    );
  });

  it("leaves out days whose edges miss the buckets in another time zone", () => {
    const days = [nov5, nov5 + 24 * HOUR].map((at) => ({ ...bucket(at), people: 5 }));
    expect(entriesByDay(days, "Asia/Kolkata")).toEqual(new Map());
  });
});
