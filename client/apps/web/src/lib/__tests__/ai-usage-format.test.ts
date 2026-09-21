import { describe, expect, it } from "vitest";
import { formatLatency, formatTokens, formatUsd } from "../ai-usage-format";

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
