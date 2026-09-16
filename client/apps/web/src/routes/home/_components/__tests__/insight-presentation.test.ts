import type { Insight, InsightMetric } from "@/types/insight";
import { describe, expect, it } from "vitest";
import {
  formatMetricChange,
  formatMetricValue,
  insightBody,
  isChangeAdverse,
  isStale,
  primaryMetrics,
  sortInsights,
} from "../widgets/insight-presentation";

function metric(overrides: Partial<InsightMetric> = {}): InsightMetric {
  return {
    key: "onTimePercent",
    label: "On-time delivery",
    value: "82.4",
    unit: "Percent",
    direction: "LowerIsWorse",
    baseline: null,
    baselineLabel: "",
    ...overrides,
  };
}

function insight(overrides: Partial<Insight> = {}): Insight {
  return {
    id: "inst_1",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    detectorKey: "ontime-decline",
    category: "ServiceQuality",
    severity: "Warning",
    status: "Active",
    dedupeKey: "ontime-decline:cus_1",
    subject: "Acme Foods",
    headline: "On-time delivery for Acme Foods is 82.4%",
    narrative: "",
    recommendation: "",
    narrated: false,
    metrics: [metric()],
    links: [],
    windowStart: 1_700_000_000,
    windowEnd: 1_702_592_000,
    detectedAt: 1_702_592_000,
    staleAt: 1_702_678_400,
    dismissedAt: null,
    dismissedById: "",
    dismissReason: "",
    modelIdentifier: "",
    providerId: "",
    version: 0,
    createdAt: 1_702_592_000,
    updatedAt: 1_702_592_000,
    ...overrides,
  };
}

describe("formatMetricValue", () => {
  it("renders each unit the way that unit reads", () => {
    expect(formatMetricValue(metric({ value: "82.4", unit: "Percent" }))).toBe("82.4%");
    expect(formatMetricValue(metric({ value: "21", unit: "Days" }))).toBe("21d");
    expect(formatMetricValue(metric({ value: "90", unit: "Hours" }))).toBe("90h");
    expect(formatMetricValue(metric({ value: "14", unit: "Count" }))).toBe("14");
    expect(formatMetricValue(metric({ value: "28000", unit: "Miles" }))).toBe("28,000 mi");
  });

  it("renders money as money", () => {
    const formatted = formatMetricValue(metric({ value: "31240.00", unit: "Currency" }));

    expect(formatted).toContain("31,240");
    expect(formatted).toMatch(/\$/u);
  });

  // A card that says "NaN unbilled" is worse than one that admits it does not
  // know. The server sends decimal strings, and a malformed one must not
  // become a confident-looking figure.
  it("admits it cannot read an unparseable value rather than printing NaN", () => {
    expect(formatMetricValue(metric({ value: "not a number" }))).toBe("—");
    expect(formatMetricValue(metric({ value: "" }))).toBe("—");
  });

  it("does not print trailing zeroes on a whole number", () => {
    expect(formatMetricValue(metric({ value: "95.0", unit: "Percent" }))).toBe("95%");
  });
});

describe("formatMetricChange", () => {
  it("shows the movement against a baseline with its direction", () => {
    expect(formatMetricChange(metric({ value: "82.4", baseline: "93.1" }))).toBe("−10.7%");
    expect(formatMetricChange(metric({ value: "97", baseline: "88" }))).toBe("+9%");
  });

  // Nothing to compare against means nothing to draw, rather than an arrow
  // pointing at zero.
  it("returns nothing when there is no baseline", () => {
    expect(formatMetricChange(metric({ baseline: null }))).toBeNull();
  });

  it("returns nothing when the value has not moved", () => {
    expect(formatMetricChange(metric({ value: "93.1", baseline: "93.1" }))).toBeNull();
  });

  it("returns nothing when either side cannot be read", () => {
    expect(formatMetricChange(metric({ value: "unknown", baseline: "93.1" }))).toBeNull();
    expect(formatMetricChange(metric({ value: "82.4", baseline: "unknown" }))).toBeNull();
  });
});

describe("isChangeAdverse", () => {
  // Rising revenue and rising detention are not the same news, and a renderer
  // cannot tell them apart without being told.
  it("reads the direction rather than the sign", () => {
    expect(
      isChangeAdverse(metric({ value: "82.4", baseline: "93.1", direction: "LowerIsWorse" })),
    ).toBe(true);
    expect(
      isChangeAdverse(metric({ value: "97", baseline: "93.1", direction: "LowerIsWorse" })),
    ).toBe(false);
    expect(
      isChangeAdverse(metric({ value: "28", baseline: "12", direction: "HigherIsWorse" })),
    ).toBe(true);
    expect(
      isChangeAdverse(metric({ value: "8", baseline: "12", direction: "HigherIsWorse" })),
    ).toBe(false);
  });

  it("treats a neutral metric as never adverse", () => {
    expect(isChangeAdverse(metric({ value: "140", baseline: "40", direction: "Neutral" }))).toBe(
      false,
    );
  });
});

describe("insightBody", () => {
  // A card headed "insight" invites more trust than a table, so the reader is
  // told which sentences a model wrote.
  it("marks narration a model produced as generated", () => {
    const body = insightBody(
      insight({ narrated: true, narrative: "Deliveries are arriving late more often." }),
    );

    expect(body.generated).toBe(true);
    expect(body.text).toBe("Deliveries are arriving late more often.");
  });

  // The detector's own wording is not generated text and must not be labelled
  // as though a model wrote it.
  it("does not claim detector wording was generated", () => {
    const body = insightBody(insight({ narrated: false, narrative: "Plain detector text." }));

    expect(body.generated).toBe(false);
  });

  it("has no body when nothing was written", () => {
    expect(insightBody(insight({ narrative: "   " })).text).toBe("");
  });
});

describe("sortInsights", () => {
  it("puts the most urgent finding first", () => {
    const sorted = sortInsights([
      insight({ id: "a", severity: "Info", detectedAt: 300 }),
      insight({ id: "b", severity: "Critical", detectedAt: 100 }),
      insight({ id: "c", severity: "Warning", detectedAt: 200 }),
    ]);

    expect(sorted.map((entry) => entry.id)).toEqual(["b", "c", "a"]);
  });

  it("puts the newest first within a severity", () => {
    const sorted = sortInsights([
      insight({ id: "old", severity: "Warning", detectedAt: 100 }),
      insight({ id: "new", severity: "Warning", detectedAt: 500 }),
    ]);

    expect(sorted.map((entry) => entry.id)).toEqual(["new", "old"]);
  });

  it("does not mutate the list it was given", () => {
    const original = [
      insight({ id: "a", severity: "Info" }),
      insight({ id: "b", severity: "Critical" }),
    ];

    sortInsights(original);

    expect(original.map((entry) => entry.id)).toEqual(["a", "b"]);
  });
});

describe("isStale", () => {
  it("compares against the refresh horizon", () => {
    const entry = insight({ staleAt: 1_000 });

    expect(isStale(entry, 1_000)).toBe(false);
    expect(isStale(entry, 1_001)).toBe(true);
  });

  // The absence of a refresh promise is not evidence the numbers are old.
  it("treats an unset horizon as fresh", () => {
    expect(isStale(insight({ staleAt: 0 }), Number.MAX_SAFE_INTEGER)).toBe(false);
  });
});

describe("primaryMetrics", () => {
  it("shows at most two numbers on a card", () => {
    const entry = insight({
      metrics: [metric({ key: "one" }), metric({ key: "two" }), metric({ key: "three" })],
    });

    expect(primaryMetrics(entry).map((item) => item.key)).toEqual(["one", "two"]);
  });

  it("copes with a finding that arrived without metrics", () => {
    expect(primaryMetrics(insight({ metrics: null }))).toEqual([]);
  });
});
