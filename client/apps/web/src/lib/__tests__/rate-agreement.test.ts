import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  agreementWindow,
  daysUntil,
  nextLaneOffset,
  orderedAgreements,
  type AgreementTerm,
} from "@/lib/rate-agreement";

/** Fills a message's numbered places the way the catalog would, plurals read as their `other` arm. */
const t = ((message: string, ...args: unknown[]) =>
  message
    .replace(/\{(\d+), plural, one \{[^}]*\} other \{([^}]*)\}\}/gu, (_, index, other: string) =>
      other.replace("#", String(args[Number(index)])),
    )
    .replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

const DAY = 86_400;
/** Oct 11, 2026 12:00 UTC. */
const NOW = 1_791_720_000;

type Agreement = AgreementTerm & { id: string };

function agreement(overrides: Partial<Agreement> = {}): Agreement {
  return {
    id: "ra_1",
    status: "Active",
    effectiveFrom: NOW - 200 * DAY,
    effectiveTo: NOW + 200 * DAY,
    autoRenew: false,
    renewalNoticeDays: 30,
    ...overrides,
  };
}

describe("daysUntil", () => {
  it("counts whole days ahead, and below zero once the day has gone", () => {
    expect(daysUntil(NOW + 12 * DAY, NOW)).toBe(12);
    expect(daysUntil(NOW + 1, NOW)).toBe(1);
    expect(daysUntil(NOW - DAY, NOW)).toBe(-1);
  });
});

describe("agreementWindow", () => {
  it("runs on with no end, and through a far end", () => {
    expect(agreementWindow(agreement({ effectiveTo: null }), NOW, t)).toEqual({
      text: "No end date",
      state: "ok",
    });
    expect(agreementWindow(agreement(), NOW, t).text).toMatch(/^Through /u);
  });

  it("flags an end inside the renewal notice, unless it renews by itself", () => {
    expect(agreementWindow(agreement({ effectiveTo: NOW + 12 * DAY }), NOW, t)).toEqual({
      text: "Ends in 12 days",
      state: "warn",
    });
    expect(
      agreementWindow(agreement({ effectiveTo: NOW + 12 * DAY, autoRenew: true }), NOW, t),
    ).toMatchObject({ state: "ok" });
  });

  it("says when an agreement ended, has not started, or is not active", () => {
    expect(agreementWindow(agreement({ effectiveTo: NOW - DAY }), NOW, t).state).toBe("bad");
    expect(agreementWindow(agreement({ effectiveFrom: NOW + DAY }), NOW, t).text).toMatch(
      /^Starts /u,
    );
    expect(agreementWindow(agreement({ status: "InReview" }), NOW, t)).toEqual({
      text: "In review",
      state: null,
    });
  });
});

describe("orderedAgreements", () => {
  it("puts the agreements in force first, each group newest first", () => {
    const ordered = orderedAgreements([
      agreement({ id: "old_draft", status: "Draft", effectiveFrom: NOW - 10 * DAY }),
      agreement({ id: "old_active", effectiveFrom: NOW - 300 * DAY }),
      agreement({ id: "new_active", effectiveFrom: NOW - 20 * DAY }),
      agreement({ id: "new_draft", status: "Draft", effectiveFrom: NOW - DAY }),
    ]);
    expect(ordered.map((entry) => entry.id)).toEqual([
      "new_active",
      "old_active",
      "new_draft",
      "old_draft",
    ]);
  });
});

describe("nextLaneOffset", () => {
  const page = (offset: number, rows: number, count: number, countExact = true) => ({
    offset,
    limit: 8,
    count,
    countExact,
    results: Array.from({ length: rows }, (_, index) => index),
  });

  it("starts the next page where this one ended, until every lane is read", () => {
    expect(nextLaneOffset(page(0, 8, 20))).toBe(8);
    expect(nextLaneOffset(page(16, 4, 20))).toBeUndefined();
    expect(nextLaneOffset(page(0, 8, 8))).toBeUndefined();
  });

  // Lanes retired while someone pages leave the count ahead of the rows;
  // an empty page ends the reading rather than asking for the same page again.
  it("stops at an empty page even when the count says there are more", () => {
    expect(nextLaneOffset(page(24, 0, 30))).toBeUndefined();
  });

  // An estimate can sit below the lanes there are, so it never ends the reading:
  // a full page asks for the next, a short one is the last.
  it("pages past an estimated count by full pages", () => {
    expect(nextLaneOffset(page(16_000, 8, 10_001, false))).toBe(16_008);
    expect(nextLaneOffset(page(16_008, 3, 10_001, false))).toBeUndefined();
  });
});
