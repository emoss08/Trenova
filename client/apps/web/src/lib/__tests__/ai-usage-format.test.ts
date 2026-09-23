import { describe, expect, it } from "vitest";
import { formatLatency, formatTokens, formatUsd, formatWorkDuration } from "../ai-usage-format";

describe("formatLatency", () => {
  it("reads in the unit a person uses", () => {
    expect(formatLatency(640)).toBe("640 ms");
    expect(formatLatency(1840)).toBe("1.8 s");
    expect(formatLatency(12400)).toBe("12 s");
    expect(formatLatency(-1)).toBe("—");
  });
});

describe("formatUsd", () => {
  // A single turn is usually a fraction of a cent; "$0.00" would say it was free.
  it("keeps the cents that matter", () => {
    expect(formatUsd("0.0042")).toBe("$0.0042");
    expect(formatUsd(1.5)).toBe("$1.50");
    expect(formatUsd("0")).toBe("$0.00");
    expect(formatUsd(null)).toBeNull();
    expect(formatUsd("")).toBeNull();
    expect(formatUsd("not-a-number")).toBeNull();
  });
});

describe("formatTokens", () => {
  it("stops counting once the number stops being countable", () => {
    expect(formatTokens(842)).toBe("842");
    expect(formatTokens(48_200)).toBe("48k");
    expect(formatTokens(2_350_000)).toBe("2.4M");
    expect(formatTokens(12_000_000)).toBe("12M");
  });
});

describe("formatWorkDuration", () => {
  // Steps are stamped in whole seconds, so a step inside one reads as under
  // a second rather than as a zero that looks like nothing happened.
  it("reads a step's time at the resolution it was stamped", () => {
    expect(formatWorkDuration(0)).toBe("<1s");
    expect(formatWorkDuration(0.4)).toBe("<1s");
    expect(formatWorkDuration(14)).toBe("14s");
    expect(formatWorkDuration(60)).toBe("1m");
    expect(formatWorkDuration(125)).toBe("2m 5s");
    expect(formatWorkDuration(3600)).toBe("1h");
    expect(formatWorkDuration(3780)).toBe("1h 3m");
    expect(formatWorkDuration(Number.NaN)).toBe("<1s");
  });
});
