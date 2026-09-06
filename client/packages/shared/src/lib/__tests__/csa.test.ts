import { describe, expect, it } from "vitest";
import {
  csaBarWidth,
  csaBasicLabel,
  csaBasicTone,
  safetyRatingTone,
  trendDirection,
  weekdayLabel,
} from "../csa";

describe("csaBasicLabel", () => {
  it("uses the agency's own words", () => {
    expect(csaBasicLabel("HOSCompliance")).toBe("Hours of Service");
    expect(csaBasicLabel("CrashIndicator")).toBe("Crash Indicator");
  });

  it("passes an unknown value through rather than blanking it", () => {
    expect(csaBasicLabel("SomethingNew")).toBe("SomethingNew");
  });
});

describe("csaBasicTone", () => {
  // The bar is relative to the fleet's own worst BASIC. An absolute threshold
  // would be meaningless: the FMCSA ranks a carrier against its peers, and
  // this data cannot.
  it("grades against the widest bar", () => {
    expect(csaBasicTone(90, 100)).toBe("critical");
    expect(csaBasicTone(40, 100)).toBe("warning");
    expect(csaBasicTone(10, 100)).toBe("muted");
  });

  it("is muted when there is nothing to compare against", () => {
    expect(csaBasicTone(0, 100)).toBe("muted");
    expect(csaBasicTone(50, 0)).toBe("muted");
  });
});

describe("csaBarWidth", () => {
  it("is the share of the widest bar", () => {
    expect(csaBarWidth(50, 100)).toBe(50);
    expect(csaBarWidth(100, 100)).toBe(100);
  });

  // A BASIC carrying one violation drawn as nothing at all reads as a BASIC
  // carrying none.
  it("keeps a visible sliver for a small figure", () => {
    expect(csaBarWidth(1, 1000)).toBe(3);
  });

  it("is nothing when the figure is nothing", () => {
    expect(csaBarWidth(0, 100)).toBe(0);
  });
});

describe("trendDirection", () => {
  it("compares the back half of the window to the front", () => {
    expect(trendDirection([1, 1, 4, 5])).toBe("up");
    expect(trendDirection([6, 5, 1, 1])).toBe("down");
  });

  // Two points are not a trend.
  it("is flat when there is not enough of a window", () => {
    expect(trendDirection([1, 9])).toBe("flat");
    expect(trendDirection([])).toBe("flat");
  });

  it("is flat when the movement is noise", () => {
    expect(trendDirection([2, 2, 2, 2])).toBe("flat");
  });
});

describe("safetyRatingTone", () => {
  it("grades the four ratings", () => {
    expect(safetyRatingTone("Excellent")).toBe("active");
    expect(safetyRatingTone("AtRisk")).toBe("inactive");
    expect(safetyRatingTone("Watch")).toBe("warning");
    expect(safetyRatingTone("Good")).toBe("secondary");
  });
});

describe("weekdayLabel", () => {
  // Zero is Sunday, matching the column the digest reads.
  it("starts the week on Sunday", () => {
    expect(weekdayLabel(0)).toBe("Sunday");
    expect(weekdayLabel(5)).toBe("Friday");
  });
});
