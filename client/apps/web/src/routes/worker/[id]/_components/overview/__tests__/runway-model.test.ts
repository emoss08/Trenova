import { describe, expect, it } from "vitest";
import {
  clearanceItemAction,
  clearanceSummary,
  placeRunwayLabels,
  runwayBand,
  runwayLater,
  runwayMarkers,
  runwayRangeDays,
  runwayTicks,
  runwayTone,
  runwayX,
  type RunwayMarker,
  type WorkerClearance,
  type WorkerClearanceItem,
} from "../runway-model";

const DAY = 86_400;
/** Nov 22, 2026 at noon UTC: the sample record's today. */
const AS_OF = Date.UTC(2026, 10, 22, 12) / 1000;

function item(
  overrides: Partial<WorkerClearanceItem> & { key: string; daysUntil: number },
): WorkerClearanceItem {
  return {
    lane: "License",
    source: "Credential",
    name: overrides.key,
    dueAt: AS_OF + overrides.daysUntil * DAY,
    effect: "Due",
    rule: null,
    hazmatOnly: false,
    regulation: null,
    required: true,
    destination: {
      tab: "Qualification",
      view: "Credentials",
      record: "Credential",
      recordId: `cred_${overrides.key}`,
    },
    ...overrides,
  };
}

/** The sample record from DESIGN.md §6, counted from Nov 22, 2026. */
const SAMPLE_ITEMS: WorkerClearanceItem[] = [
  item({
    key: "clearinghouse",
    name: "Clearinghouse annual query",
    lane: "ReviewsAndQueries",
    source: "ClearinghouseQuery",
    daysUntil: 17,
    destination: { tab: "Qualification", view: "DrugAndAlcohol", record: null, recordId: null },
  }),
  item({
    key: "refresher",
    name: "Hazmat awareness refresher",
    lane: "Training",
    source: "Training",
    daysUntil: 26,
    destination: { tab: "Development", view: "Training", record: "Training", recordId: "trn_1" },
  }),
  item({
    key: "medical",
    name: "Medical card",
    lane: "Medical",
    daysUntil: 84,
    effect: "StopsAssignment",
    rule: "RequiredCredential",
    regulation: "49 CFR 391.45",
  }),
  item({ key: "mvr", name: "MVR review", lane: "ReviewsAndQueries", daysUntil: 101 }),
  item({
    key: "hazmat",
    name: "Hazmat endorsement",
    daysUntil: 220,
    effect: "StopsAssignment",
    rule: "DispatchCompliance",
    hazmatOnly: true,
  }),
  item({
    key: "defensive",
    name: "Defensive driving",
    lane: "Training",
    source: "Training",
    daysUntil: 263,
    destination: { tab: "Development", view: "Training", record: "Training", recordId: "trn_2" },
  }),
  item({ key: "twic", name: "TWIC card", daysUntil: 869 }),
  item({
    key: "cdl",
    name: "CDL",
    daysUntil: 1243,
    effect: "StopsAssignment",
    rule: "RequiredCredential",
  }),
];

function clearance(overrides: Partial<WorkerClearance> = {}): WorkerClearance {
  const items = overrides.items ?? SAMPLE_ITEMS;
  return {
    cleared: true,
    clearedThrough: AS_OF + 84 * DAY,
    notClearedSince: null,
    blocks: [],
    itemsTruncated: false,
    firstStop: {
      key: "medical",
      name: "Medical card",
      source: "Credential",
      dueAt: AS_OF + 84 * DAY,
      daysUntil: 84,
    },
    items,
    ...overrides,
  };
}

describe("runwayRangeDays", () => {
  it("spans 6, 12 and 24 calendar months from today, leap day included", () => {
    expect(runwayRangeDays(AS_OF, 6)).toBe(181);
    expect(runwayRangeDays(AS_OF, 12)).toBe(365);
    expect(runwayRangeDays(AS_OF, 24)).toBe(731);
  });

  it("ends on the last day of a shorter month", () => {
    expect(runwayRangeDays(Date.UTC(2026, 7, 31, 12) / 1000, 6)).toBe(181);
  });
});

describe("runwayX and runwayTone", () => {
  it("places a date by its share of the range, lapsed on the Today line and late at the edge", () => {
    expect(runwayX(84, 365)).toBeCloseTo(84 / 365, 10);
    expect(runwayX(-3, 365)).toBe(0);
    expect(runwayX(400, 365)).toBe(1);
  });

  it("tones by distance: over 90 days, 31 to 90, 30 or less, lapsed", () => {
    expect([91, 90, 31, 30, 0, -1].map(runwayTone)).toEqual([
      "far",
      "soon",
      "soon",
      "near",
      "near",
      "lapsed",
    ]);
  });
});

describe("runwayTicks", () => {
  it("ticks the first of every month over 12 months, the year on January", () => {
    const ticks = runwayTicks(AS_OF, 365);
    expect(ticks).toHaveLength(12);
    expect(ticks[0]).toEqual({ at: Date.UTC(2026, 11, 1) / 1000, x: 9 / 365, withYear: false });
    expect(ticks[1]).toMatchObject({ at: Date.UTC(2027, 0, 1) / 1000, withYear: true });
    expect(ticks[2]).toMatchObject({ at: Date.UTC(2027, 1, 1) / 1000, withYear: false });
    expect(ticks[11]).toMatchObject({ at: Date.UTC(2027, 10, 1) / 1000 });
  });

  it("ticks every third month over 24 months, the year on the first label in each new year", () => {
    const ticks = runwayTicks(AS_OF, 730);
    expect(ticks.map((tick) => new Date(tick.at * 1000).toISOString().slice(0, 7))).toEqual([
      "2026-12",
      "2027-03",
      "2027-06",
      "2027-09",
      "2027-12",
      "2028-03",
      "2028-06",
      "2028-09",
    ]);
    expect(ticks.map((tick) => tick.withYear)).toEqual([
      false,
      true,
      false,
      false,
      false,
      true,
      false,
      false,
    ]);
  });
});

describe("runwayBand", () => {
  it("runs from today to the first hard stop", () => {
    const band = runwayBand(clearance(), 365);
    expect(band?.stop?.key).toBe("medical");
    expect(band?.edge).toBeCloseTo(84 / 365, 10);
    expect(band?.beyond).toBe(false);
    expect(band?.flagFlipped).toBe(false);
  });

  it("is not ended by a hazmat-only stop that comes first", () => {
    const items = [
      item({
        key: "hazmat",
        daysUntil: 40,
        effect: "StopsAssignment",
        rule: "DispatchCompliance",
        hazmatOnly: true,
      }),
      ...SAMPLE_ITEMS.filter((entry) => entry.key !== "hazmat"),
    ];
    const band = runwayBand(clearance({ items }), 365);
    expect(band?.stop?.key).toBe("medical");
    const hazmat = runwayMarkers(clearance({ items }), { variant: "lanes", rangeDays: 365 }).find(
      (marker) => marker.item.key === "hazmat",
    );
    expect(hazmat).toMatchObject({ solid: true, endsBand: false });
  });

  it("moves inward on a wider range and flips its flag near the right edge", () => {
    expect(runwayBand(clearance(), 730)?.edge).toBeCloseTo(84 / 730, 10);
    const late = item({
      key: "late",
      daysUntil: 320,
      effect: "StopsAssignment",
      rule: "RequiredCredential",
    });
    const band = runwayBand(clearance({ items: [late], firstStop: null }), 365);
    expect(band?.stop?.key).toBe("late");
    expect(band?.flagFlipped).toBe(true);
  });

  it("runs off the edge when the stop is past the range or there is none", () => {
    expect(runwayBand(clearance(), 61)).toMatchObject({ edge: 1, beyond: true });
    expect(runwayBand(clearance({ items: [], firstStop: null }), 365)).toEqual({
      edge: 1,
      stop: null,
      beyond: true,
      flagFlipped: false,
    });
  });

  it("uses the server's first stop when it lies past the capped items", () => {
    const band = runwayBand(
      clearance({
        items: SAMPLE_ITEMS.filter((entry) => entry.effect === "Due"),
        itemsTruncated: true,
      }),
      365,
    );
    expect(band?.stop?.key).toBe("medical");
  });

  it("is absent when the worker is not cleared", () => {
    expect(runwayBand(clearance({ cleared: false, clearedThrough: null }), 365)).toBeNull();
  });
});

describe("runwayMarkers and runwayLater", () => {
  it("puts each item in its lane, solid for a stop, and what is past the range in Later", () => {
    const markers = runwayMarkers(clearance(), { variant: "lanes", rangeDays: 365 });
    expect(
      markers.map((marker) => [marker.item.key, marker.lane, marker.inRange, marker.solid]),
    ).toEqual([
      ["clearinghouse", 2, true, false],
      ["refresher", 3, true, false],
      ["medical", 1, true, true],
      ["mvr", 2, true, false],
      ["hazmat", 0, true, true],
      ["defensive", 3, true, false],
      ["twic", 0, false, false],
      ["cdl", 0, false, true],
    ]);
    expect(markers.find((marker) => marker.item.key === "medical")?.endsBand).toBe(true);
    const later = runwayLater(markers, 4);
    expect(later.map((lane) => lane.map((marker) => marker.item.key))).toEqual([
      ["twic", "cdl"],
      [],
      [],
      [],
    ]);
  });

  it("moves markers inward on 24 months and fewer sit in Later on 6", () => {
    const wide = runwayMarkers(clearance(), { variant: "lanes", rangeDays: 730 });
    expect(wide.find((marker) => marker.item.key === "hazmat")?.x).toBeCloseTo(220 / 730, 10);
    const narrow = runwayMarkers(clearance(), { variant: "lanes", rangeDays: 181 });
    expect(runwayLater(narrow, 4)[0]?.map((marker) => marker.item.key)).toEqual([
      "hazmat",
      "twic",
      "cdl",
    ]);
  });

  it("draws one lane of the sources asked for in the single variant", () => {
    const markers = runwayMarkers(clearance(), {
      variant: "single",
      rangeDays: 365,
      sources: ["Credential"],
    });
    expect(markers.map((marker) => marker.item.key)).toEqual([
      "medical",
      "mvr",
      "hazmat",
      "twic",
      "cdl",
    ]);
    expect(new Set(markers.map((marker) => marker.lane))).toEqual(new Set([0]));
  });

  it("puts a lapsed item on the Today line in the lapsed tone", () => {
    const lapsed = item({
      key: "old",
      daysUntil: -12,
      effect: "StopsAssignment",
      rule: "RequiredCredential",
    });
    const [marker] = runwayMarkers(
      clearance({ cleared: false, items: [lapsed], firstStop: null }),
      {
        variant: "lanes",
        rangeDays: 365,
      },
    );
    expect(marker).toMatchObject({ x: 0, tone: "lapsed", inRange: true, endsBand: false });
  });
});

describe("placeRunwayLabels", () => {
  function marker(key: string, x: number, lane = 0): RunwayMarker {
    return {
      item: item({ key, daysUntil: Math.round(x * 365) }),
      lane,
      x,
      inRange: true,
      tone: "far",
      solid: false,
      endsBand: false,
    };
  }
  const measure = (_: WorkerClearanceItem, short: boolean) => (short ? 30 : 100);

  it("sets a label above, then below when the one before runs into it", () => {
    const labels = placeRunwayLabels(
      [marker("a", 0.125), marker("b", 0.25), marker("c", 0.25, 1)],
      {
        width: 400,
        measure,
        sides: ["above", "below"],
      },
    );
    expect(labels.get("a")).toEqual({ side: "above", flip: false, short: false, hidden: false });
    expect(labels.get("b")).toEqual({ side: "below", flip: false, short: false, hidden: false });
    expect(labels.get("c")).toEqual({ side: "above", flip: false, short: false, hidden: false });
  });

  it("flips to the left near the right edge, shortens when only that fits, and drops the label last", () => {
    const labels = placeRunwayLabels(
      [marker("a", 0.5), marker("b", 0.85), marker("c", 0.9), marker("d", 0.9125)],
      { width: 400, measure, sides: ["above", "below"] },
    );
    expect(labels.get("a")).toEqual({ side: "above", flip: false, short: false, hidden: false });
    expect(labels.get("b")).toEqual({ side: "below", flip: true, short: false, hidden: false });
    expect(labels.get("c")).toEqual({ side: "above", flip: false, short: true, hidden: false });
    expect(labels.get("d")).toEqual({ side: "above", flip: false, short: true, hidden: true });
  });

  it("uses only the rows it is given", () => {
    const labels = placeRunwayLabels([marker("a", 0), marker("b", 0)], {
      width: 400,
      measure,
      sides: ["above"],
    });
    expect(labels.get("b")?.hidden).toBe(true);
  });

  it("leaves markers past the range unlabelled", () => {
    const late = { ...marker("late", 1), inRange: false };
    expect(placeRunwayLabels([late], { width: 400, measure, sides: ["above"] }).has("late")).toBe(
      false,
    );
  });
});

describe("clearanceSummary", () => {
  it("counts what comes due before the first stop", () => {
    expect(clearanceSummary(clearance())).toMatchObject({
      stop: { key: "medical" },
      dueBefore: 2,
      hazmatStopsBefore: 0,
    });
  });

  it("counts hazmat-only stops before it apart and leaves out what has lapsed", () => {
    const items = [
      item({ key: "lapsed-query", daysUntil: -5, lane: "ReviewsAndQueries" }),
      item({
        key: "hazmat",
        daysUntil: 40,
        effect: "StopsAssignment",
        rule: "DispatchCompliance",
        hazmatOnly: true,
      }),
      ...SAMPLE_ITEMS.filter((entry) => entry.key !== "hazmat"),
    ];
    expect(clearanceSummary(clearance({ items }))).toMatchObject({
      dueBefore: 3,
      hazmatStopsBefore: 1,
    });
  });

  it("counts everything ahead when nothing stops the driver", () => {
    const items = SAMPLE_ITEMS.filter((entry) => entry.effect === "Due");
    expect(clearanceSummary(clearance({ items, firstStop: null }))).toMatchObject({
      stop: null,
      dueBefore: 0,
      dueAhead: 5,
    });
  });
});

describe("clearanceItemAction", () => {
  it("renews a credential on file and adds one never recorded", () => {
    expect(clearanceItemAction(SAMPLE_ITEMS[2]!)).toEqual({
      action: "renew",
      target: {
        tab: "qualification",
        view: "credentials",
        form: "renew-credential",
        credential: "cred_medical",
      },
    });
    expect(
      clearanceItemAction(
        item({
          key: "missing",
          daysUntil: 0,
          destination: { tab: "Qualification", view: "Credentials", record: null, recordId: null },
        }),
      ),
    ).toEqual({
      action: "add-credential",
      target: { tab: "qualification", view: "credentials", form: "add-credential" },
    });
  });

  it("logs the query, opens a course where it is and starts a review not yet open", () => {
    expect(clearanceItemAction(SAMPLE_ITEMS[0]!)).toEqual({
      action: "log-query",
      target: { tab: "qualification", view: "drug-and-alcohol", form: "log-query" },
    });
    expect(clearanceItemAction(SAMPLE_ITEMS[1]!)).toEqual({
      action: "open-training",
      target: { tab: "development", view: "training", training: "trn_1" },
    });
    expect(
      clearanceItemAction(
        item({
          key: "review",
          daysUntil: 50,
          source: "PerformanceReview",
          destination: { tab: "Development", view: "Reviews", record: null, recordId: null },
        }),
      ),
    ).toEqual({
      action: "start-review",
      target: { tab: "development", view: "reviews", form: "start-review" },
    });
  });
});
