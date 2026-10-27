import type { ReportRunDeskCard, ReportRunDeskParameter } from "@/lib/graphql/report-desk-card";
import { describe, expect, it } from "vitest";
import {
  durationText,
  parameterText,
  reportsPath,
  sheetWindow,
  sourceOfPreview,
  sourceOfRun,
  windowText,
} from "../report-run-card";

const t = ((text: string, ...args: unknown[]) =>
  text
    .replace(/\{(\d), plural, one \{# ([^}]*)\} other \{# ([^}]*)\}\}/gu, (_m, at, one, other) =>
      args[Number(at)] === 1 ? `1 ${one}` : `${String(args[Number(at)])} ${other}`,
    )
    .replace(/\{(\d)\}/gu, (_m, at) => String(args[Number(at)]))) as never;

const CHICAGO = "America/Chicago";
const at = (iso: string) => Math.floor(Date.parse(iso) / 1000);

function parameter(overrides: Partial<ReportRunDeskParameter>): ReportRunDeskParameter {
  return {
    name: "windowDays",
    label: "Window (days)",
    type: "int",
    value: 30,
    isDefault: false,
    refEntity: null,
    windowStart: null,
    windowEnd: null,
    ...overrides,
  };
}

describe("windowText", () => {
  it("writes a window from its first day to its last, the end being exclusive", () => {
    const start = at("2026-09-27T05:00:00Z");
    const end = at("2026-10-28T05:00:00Z");
    expect(windowText(start, end, t, { timeZone: CHICAGO })).toBe("Sep 27 – Oct 27");
    expect(windowText(start, end, t, { timeZone: CHICAGO, withYear: true })).toBe(
      "Sep 27 – Oct 27, 2026",
    );
  });

  it("names both years when a window crosses one", () => {
    expect(
      windowText(at("2025-10-26T05:00:00Z"), at("2026-10-26T05:00:00Z"), t, { timeZone: CHICAGO }),
    ).toBe("Oct 26, 2025 – Oct 25, 2026");
  });

  it("writes a single day once", () => {
    expect(
      windowText(at("2026-10-27T05:00:00Z"), at("2026-10-28T05:00:00Z"), t, { timeZone: CHICAGO }),
    ).toBe("Oct 27");
  });
});

describe("parameterText", () => {
  it("adds the window a relative parameter resolved to", () => {
    expect(
      parameterText(
        parameter({
          windowStart: at("2026-09-27T05:00:00Z"),
          windowEnd: at("2026-10-28T05:00:00Z"),
        }),
        t,
        CHICAGO,
      ),
    ).toBe("30 · Sep 27 – Oct 27");
  });

  it("lists a few values, counts the rest, and counts records it cannot name", () => {
    expect(parameterText(parameter({ value: ["Unpaid", "PartiallyPaid"] }), t)).toBe(
      "Unpaid, PartiallyPaid",
    );
    expect(parameterText(parameter({ value: ["A", "B", "C", "D", "E"] }), t)).toBe(
      "A, B, C and 2 more",
    );
    expect(parameterText(parameter({ value: ["cus_1", "cus_2"], refEntity: "customer" }), t)).toBe(
      "2 selected",
    );
    expect(parameterText(parameter({ value: "cus_1", refEntity: "customer" }), t)).toBe(
      "1 selected",
    );
  });

  it("says a parameter left empty applies to anything, and a flag yes or no", () => {
    expect(parameterText(parameter({ value: null }), t)).toBe("Any");
    expect(parameterText(parameter({ value: [] }), t)).toBe("Any");
    expect(parameterText(parameter({ value: true }), t)).toBe("Yes");
    expect(parameterText(parameter({ value: false }), t)).toBe("No");
  });
});

describe("reportsPath", () => {
  it("opens a saved report by id and a built-in one by key", () => {
    expect(reportsPath({ definitionId: "rdef_1", cannedKey: null })).toBe(
      "/reports/explore/rdef_1",
    );
    expect(reportsPath({ definitionId: null, cannedKey: "revenue by customer" })).toBe(
      "/reports/explore?canned=revenue%20by%20customer",
    );
    expect(reportsPath({ definitionId: "", cannedKey: "" })).toBe("");
  });
});

describe("durationText", () => {
  it("writes seconds under a minute and minutes past it", () => {
    expect(durationText(3200, t)).toBe("3.2 s");
    expect(durationText(64_400, t)).toBe("1 min 4 s");
  });
});

function card(overrides: Partial<ReportRunDeskCard> = {}): ReportRunDeskCard {
  return {
    run: { rowCount: 214, truncated: false } as ReportRunDeskCard["run"],
    report: { name: "", description: "", category: "", definitionId: null, cannedKey: "k" },
    parameters: [],
    timezone: CHICAGO,
    sample: {
      columns: [
        {
          id: "customer",
          label: "Customer",
          type: "string",
          format: null,
          display: null as never,
        },
      ],
      rows: [["Acme"]],
      totals: null,
      more: false,
    },
    withheld: null,
    previous: null,
    schedule: null,
    chart: null,
    sort: [],
    ...overrides,
  };
}

describe("sourceOfRun", () => {
  it("knows the file holds more rows than the sample when the run returned more", () => {
    const source = sourceOfRun(card());
    expect(source?.rowCount).toBe(214);
    expect(source?.more).toBe(true);
    expect(sourceOfRun(card({ sample: null }))).toBeNull();
  });
});

describe("sheetWindow", () => {
  it("is the first parameter that resolved to a window", () => {
    expect(
      sheetWindow(
        card({
          parameters: [
            parameter({ name: "statuses", value: ["Posted"] }),
            parameter({ windowStart: 10, windowEnd: 20 }),
          ],
        }),
      ),
    ).toEqual({ start: 10, end: 20 });
    expect(sheetWindow(card())).toBeNull();
  });
});

describe("sourceOfPreview", () => {
  it("reads a preview's catalog types, and knows when it read fewer rows than came back", () => {
    const source = sourceOfPreview({
      name: "",
      dataset: "invoice",
      columns: [
        { id: "customer", label: "Customer", type: "string", format: null },
        { id: "revenue", label: "Revenue", type: "decimal", format: "money" },
      ] as never,
      rows: [["Acme", "10"]],
      totals: null,
      rowCount: 63,
      truncated: false,
    });
    expect(source.columns.map((column) => column.numeric)).toEqual([false, true]);
    expect(source.columns[1].hint).toBe("money");
    expect(source.more).toBe(true);
  });
});
