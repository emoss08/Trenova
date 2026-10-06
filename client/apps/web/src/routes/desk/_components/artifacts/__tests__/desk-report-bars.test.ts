import type { AssistantArtifact } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { barsOf, footTotals, gridOf, versionNote } from "../desk-table-body";

const t = ((text: string, ...args: unknown[]) =>
  text.replace(/\{0, plural, one \{# ([^}]*)\} other \{# ([^}]*)\}\}/u, (_m, one, other) =>
    args[0] === 1 ? `1 ${one}` : `${String(args[0])} ${other}`,
  )) as never;

function report(rows: unknown[][], extra: Record<string, unknown> = {}): AssistantArtifact {
  return {
    id: "a1",
    kind: "report_preview",
    payload: {
      columns: [
        { id: "customer", label: "Customer", type: "text" },
        { id: "hours", label: "Hours", type: "number" },
        { id: "cost", label: "Cost", type: "currency" },
      ],
      rows: rows.map(([customer, hours, cost]) => ({ customer, hours, cost })),
      rowCount: rows.length,
      ...extra,
    },
  } as unknown as AssistantArtifact;
}

describe("report bars", () => {
  it("draws a short labelled report as bars", () => {
    const bars = barsOf(
      gridOf(
        report([
          ["Acme", 4.5, 300],
          ["Globex", 2, 120],
        ]),
      ),
    );
    expect(bars?.label.key).toBe("customer");
    expect(bars?.bar.key).toBe("hours");
    expect(bars?.extra?.key).toBe("cost");
  });

  it("keeps a table when a figure is missing or below zero", () => {
    expect(
      barsOf(
        gridOf(
          report([
            ["Acme", -1, 300],
            ["Globex", 2, 120],
          ]),
        ),
      ),
    ).toBeNull();
    expect(
      barsOf(
        gridOf(
          report([
            ["Acme", null, 300],
            ["Globex", 2, 120],
          ]),
        ),
      ),
    ).toBeNull();
    expect(barsOf(gridOf(report([["Acme", 1, 300]])))).toBeNull();
  });
});

describe("footer totals", () => {
  it("sums money over the rows showing when the source sent none", () => {
    const grid = gridOf(
      report([
        ["Acme", 4.5, 300],
        ["Globex", 2, 120.5],
      ]),
    );
    expect(footTotals(grid, grid.rows)).toEqual({ cost: 420.5 });
    expect(footTotals(grid, grid.rows.slice(1))).toEqual({ cost: 120.5 });
  });
});

describe("version notes", () => {
  it("says what a version changed", () => {
    const first = report([
      ["Acme", 4.5, 300],
      ["Globex", 2, 120],
    ]);
    const second = report([
      ["Acme", 5, 300],
      ["Globex", 2, 120],
      ["Initech", 1, 50],
    ]);
    expect(versionNote(first, null, t)).toBe("First read");
    expect(versionNote(second, first, t)).toBe("1 row added · 1 value changed");
    expect(versionNote(first, first, t)).toBe("Nothing changed");
  });
});
