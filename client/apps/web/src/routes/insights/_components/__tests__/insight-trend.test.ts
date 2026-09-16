import type { Insight, InsightMetric } from "@/types/insight";
import { describe, expect, it } from "vitest";
import { buildTrend, normalizePoints, primaryTrendMetricKey } from "../insight-trend";

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

describe("buildTrend", () => {
  // History is read newest-first; a chart is drawn oldest-first. Reversing here
  // rather than in the component is one fewer thing to get wrong.
  it("puts the series in the order a chart is read", () => {
    const current = insight({ detectedAt: 300, metrics: [metric({ value: "82" })] });
    const history = [
      insight({ id: "b", detectedAt: 200, metrics: [metric({ value: "88" })] }),
      insight({ id: "a", detectedAt: 100, metrics: [metric({ value: "93" })] }),
    ];

    const trend = buildTrend(current, history, "onTimePercent");

    expect(trend?.points).toEqual([
      { detectedAt: 100, value: 93 },
      { detectedAt: 200, value: 88 },
      { detectedAt: 300, value: 82 },
    ]);
  });

  // The sign alone does not say whether news is good: rising detention and
  // rising on-time percentage are opposites.
  it("reads the direction against what the metric says is bad", () => {
    const falling = buildTrend(
      insight({ detectedAt: 200, metrics: [metric({ value: "82", direction: "LowerIsWorse" })] }),
      [insight({ detectedAt: 100, metrics: [metric({ value: "93" })] })],
      "onTimePercent",
    );
    expect(falling?.direction).toBe("worsening");

    const rising = buildTrend(
      insight({
        detectedAt: 200,
        metrics: [metric({ key: "dwell", value: "90", direction: "HigherIsWorse" })],
      }),
      [insight({ detectedAt: 100, metrics: [metric({ key: "dwell", value: "40" })] })],
      "dwell",
    );
    expect(rising?.direction).toBe("worsening");
  });

  it("recognises movement in the right direction as improving", () => {
    const trend = buildTrend(
      insight({ detectedAt: 200, metrics: [metric({ value: "95", direction: "LowerIsWorse" })] }),
      [insight({ detectedAt: 100, metrics: [metric({ value: "84" })] })],
      "onTimePercent",
    );

    expect(trend?.direction).toBe("improving");
    expect(trend?.change).toBe(11);
  });

  // A finding seen once has no direction. Drawing it as flat would assert
  // stability nobody has observed.
  it("says the direction is unknown for a finding seen only once", () => {
    const trend = buildTrend(insight(), [], "onTimePercent");

    expect(trend?.direction).toBe("unknown");
    expect(trend?.change).toBeNull();
    expect(trend?.points).toHaveLength(1);
  });

  it("separates a genuinely level series from an unknown one", () => {
    const trend = buildTrend(
      insight({ detectedAt: 200, metrics: [metric({ value: "88" })] }),
      [insight({ detectedAt: 100, metrics: [metric({ value: "88" })] })],
      "onTimePercent",
    );

    expect(trend?.direction).toBe("steady");
  });

  it("never calls a neutral metric worsening", () => {
    const trend = buildTrend(
      insight({
        detectedAt: 200,
        metrics: [metric({ key: "moves", value: "400", direction: "Neutral" })],
      }),
      [insight({ detectedAt: 100, metrics: [metric({ key: "moves", value: "100" })] })],
      "moves",
    );

    expect(trend?.direction).toBe("steady");
  });

  // Detectors add and drop metrics between releases. A missing one plotted as
  // zero would draw a collapse that never happened.
  it("skips a refresh that did not carry the metric rather than plotting zero", () => {
    const current = insight({ detectedAt: 300, metrics: [metric({ value: "82" })] });
    const history = [
      insight({ id: "b", detectedAt: 200, metrics: [metric({ key: "somethingElse" })] }),
      insight({ id: "a", detectedAt: 100, metrics: [metric({ value: "93" })] }),
    ];

    const trend = buildTrend(current, history, "onTimePercent");

    expect(trend?.points).toEqual([
      { detectedAt: 100, value: 93 },
      { detectedAt: 300, value: 82 },
    ]);
  });

  it("skips a reading that cannot be parsed", () => {
    const trend = buildTrend(
      insight({ detectedAt: 200, metrics: [metric({ value: "82" })] }),
      [insight({ detectedAt: 100, metrics: [metric({ value: "" })] })],
      "onTimePercent",
    );

    expect(trend?.points).toHaveLength(1);
  });

  it("has no trend for a metric the finding does not carry", () => {
    expect(buildTrend(insight(), [], "nothingLikeThis")).toBeNull();
  });
});

describe("normalizePoints", () => {
  it("scales a series into the drawable range", () => {
    const normalized = normalizePoints([
      { detectedAt: 1, value: 10 },
      { detectedAt: 2, value: 20 },
      { detectedAt: 3, value: 30 },
    ]);

    expect(normalized).toEqual([0, 0.5, 1]);
  });

  // Dividing by a zero range puts every point at NaN and draws nothing. A level
  // line is the truth.
  it("draws a flat series on the midline rather than as NaN", () => {
    const normalized = normalizePoints([
      { detectedAt: 1, value: 42 },
      { detectedAt: 2, value: 42 },
    ]);

    expect(normalized).toEqual([0.5, 0.5]);
  });

  it("copes with a single point and with none", () => {
    expect(normalizePoints([{ detectedAt: 1, value: 7 }])).toEqual([0.5]);
    expect(normalizePoints([])).toEqual([]);
  });
});

describe("primaryTrendMetricKey", () => {
  it("charts the number the detector led with", () => {
    const entry = insight({
      metrics: [metric({ key: "unbilledAmount" }), metric({ key: "shipments" })],
    });

    expect(primaryTrendMetricKey(entry)).toBe("unbilledAmount");
  });

  // Guessing would draw a chart of something arbitrary.
  it("charts nothing for a finding with no metrics", () => {
    expect(primaryTrendMetricKey(insight({ metrics: null }))).toBeNull();
  });
});
