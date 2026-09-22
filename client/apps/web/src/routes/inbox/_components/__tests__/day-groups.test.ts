import { describe, expect, it } from "vitest";
import { groupByDay } from "../day-groups";

function localSeconds(year: number, month: number, day: number, hour = 12): number {
  return Math.floor(new Date(year, month - 1, day, hour).getTime() / 1000);
}

describe("groupByDay", () => {
  // A Wednesday afternoon, in whatever zone the test runs in: the buckets are
  // local days, so the fixtures are built from local dates too.
  const now = localSeconds(2026, 9, 23, 15);

  it("buckets by local day: today, yesterday, this week by weekday, then by date", () => {
    const rows = [
      { id: "a", receivedAt: localSeconds(2026, 9, 23, 9) },
      { id: "b", receivedAt: localSeconds(2026, 9, 23, 0) },
      { id: "c", receivedAt: localSeconds(2026, 9, 22, 23) },
      { id: "d", receivedAt: localSeconds(2026, 9, 20, 8) },
      { id: "e", receivedAt: localSeconds(2026, 9, 1, 8) },
    ];

    const groups = groupByDay(rows, now);

    expect(groups.map((group) => group.bucket)).toEqual(["today", "yesterday", "week", "earlier"]);
    expect(groups.map((group) => group.items.map((item) => item.id))).toEqual([
      ["a", "b"],
      ["c"],
      ["d"],
      ["e"],
    ]);
    expect(groups[2].day).toBe(localSeconds(2026, 9, 20, 0));
  });

  it("keeps two earlier days apart rather than folding them into one heading", () => {
    const groups = groupByDay(
      [
        { id: "x", receivedAt: localSeconds(2026, 8, 30, 10) },
        { id: "y", receivedAt: localSeconds(2026, 8, 29, 10) },
      ],
      now,
    );

    expect(groups).toHaveLength(2);
    expect(groups.every((group) => group.bucket === "earlier")).toBe(true);
  });

  it("puts mail stamped a little in the future under today rather than losing it", () => {
    // A sender's clock that runs ahead is common; the message still arrived.
    const groups = groupByDay([{ id: "f", receivedAt: now + 90 }], now);

    expect(groups).toEqual([
      {
        bucket: "today",
        day: localSeconds(2026, 9, 23, 0),
        items: [{ id: "f", receivedAt: now + 90 }],
      },
    ]);
  });

  it("draws the day line where the reader's zone draws it, not the browser's", () => {
    // A zone eight hours ahead of UTC: 20:00 UTC on the 22nd is already the
    // 23rd there, so it is today, not yesterday.
    const offset = 8 * 3600;
    const zoned = (seconds: number) => Math.floor((seconds + offset) / 86_400) * 86_400 - offset;
    const nowUtc = Date.UTC(2026, 8, 23, 6) / 1000;
    const lateUtc = Date.UTC(2026, 8, 22, 20) / 1000;
    const earlierUtc = Date.UTC(2026, 8, 22, 12) / 1000;

    const groups = groupByDay(
      [
        { id: "late", receivedAt: lateUtc },
        { id: "earlier", receivedAt: earlierUtc },
      ],
      nowUtc,
      zoned,
    );

    expect(groups.map((group) => [group.bucket, group.items.map((item) => item.id)])).toEqual([
      ["today", ["late"]],
      ["yesterday", ["earlier"]],
    ]);
  });

  it("has no groups for no mail", () => {
    expect(groupByDay([], now)).toEqual([]);
  });
});
