import { describe, expect, it } from "vitest";
import {
  countDeskFilters,
  deskClockTrack,
  deskFloorStats,
  formatCountdown,
  formatDetentionMinutes,
  matchesDeskFilter,
  matchesDeskSearch,
  nextDeskDeadline,
  sortDeskEntries,
  sortDeskEntriesBy,
  summarizeDesk,
} from "../detention";
import type { DeskEntry, DeskUrgency } from "../../types/detention";

const NOW = 1_767_225_600;

function entry(
  overrides: {
    urgency?: DeskUrgency;
    amountAtRisk?: number;
    grossAmount?: number;
    roundedMinutes?: number;
    freeMinutesGranted?: number;
    clockStartAt?: number;
    clockStopAt?: number | null;
    freeTimeExpiresAt?: number;
    noticeDeadlineAt?: number | null;
    locationName?: string;
    customerName?: string;
    shipmentProNumber?: string;
    stopType?: string;
  } = {},
): DeskEntry {
  const {
    urgency = "Normal",
    amountAtRisk = 0,
    grossAmount = amountAtRisk,
    roundedMinutes = 0,
    freeMinutesGranted = 120,
    clockStartAt = NOW,
    clockStopAt = null,
    freeTimeExpiresAt = NOW + 7200,
    noticeDeadlineAt = null,
    locationName = "",
    customerName = "",
    shipmentProNumber = "",
    stopType = "Delivery",
  } = overrides;

  return {
    urgency,
    amountAtRisk,
    minutesUntilFreeEnds: Math.round((freeTimeExpiresAt - NOW) / 60),
    minutesUntilNoticeDue: null,
    noticeWindowOpen: false,
    occurrence: {
      roundedMinutes,
      freeMinutesGranted,
      clockStartAt,
      clockStopAt,
      freeTimeExpiresAt,
      noticeDeadlineAt,
      isOpen: clockStopAt === null,
      locationName,
      customerName,
      shipmentProNumber,
      stopType,
      grossAmount,
      billableAmount: amountAtRisk,
    },
  } as unknown as DeskEntry;
}

describe("formatDetentionMinutes", () => {
  it("renders sub-hour durations in minutes", () => {
    expect(formatDetentionMinutes(45)).toBe("45m");
  });

  it("renders whole hours without a minute remainder", () => {
    expect(formatDetentionMinutes(120)).toBe("2h");
  });

  it("renders mixed durations", () => {
    expect(formatDetentionMinutes(195)).toBe("3h 15m");
  });

  it("renders negative durations by magnitude", () => {
    expect(formatDetentionMinutes(-90)).toBe("1h 30m");
  });

  it("renders zero", () => {
    expect(formatDetentionMinutes(0)).toBe("0m");
  });
});

describe("formatCountdown", () => {
  it("renders a pending deadline", () => {
    expect(formatCountdown(25)).toBe("in 25m");
  });

  it("renders an elapsed deadline as overdue", () => {
    expect(formatCountdown(-10)).toBe("10m overdue");
  });

  it("treats the exact deadline as overdue so it is not ignored", () => {
    expect(formatCountdown(0)).toBe("0m overdue");
  });

  it("renders an absent deadline", () => {
    expect(formatCountdown(null)).toBe("—");
    expect(formatCountdown(undefined)).toBe("—");
  });
});

describe("sortDeskEntries", () => {
  it("puts a savable notice deadline ahead of a larger accruing charge", () => {
    const sorted = sortDeskEntries([
      entry({ urgency: "Accruing", amountAtRisk: 900 }),
      entry({ urgency: "NoticeOverdue", amountAtRisk: 50 }),
    ]);

    expect(sorted[0]?.urgency).toBe("NoticeOverdue");
  });

  it("ranks by exposure within the same urgency", () => {
    const sorted = sortDeskEntries([
      entry({ urgency: "Accruing", amountAtRisk: 100 }),
      entry({ urgency: "Accruing", amountAtRisk: 500 }),
    ]);

    expect(sorted[0]?.amountAtRisk).toBe(500);
  });

  it("sinks unrecoverable claims below actionable ones", () => {
    const sorted = sortDeskEntries([
      entry({ urgency: "Lost", amountAtRisk: 5000 }),
      entry({ urgency: "Normal", amountAtRisk: 1 }),
      entry({ urgency: "NoticeDueSoon", amountAtRisk: 1 }),
    ]);

    expect(sorted.map((e) => e.urgency)).toEqual(["NoticeDueSoon", "Lost", "Normal"]);
  });

  it("does not mutate the input", () => {
    const input = [entry({ urgency: "Accruing" }), entry({ urgency: "NoticeOverdue" })];
    const before = input.map((e) => e.urgency);

    sortDeskEntries(input);

    expect(input.map((e) => e.urgency)).toEqual(before);
  });
});

describe("summarizeDesk", () => {
  it("separates exposure that is still collectable from what is already lost", () => {
    const summary = summarizeDesk([
      entry({ urgency: "Accruing", amountAtRisk: 300, roundedMinutes: 60 }),
      entry({ urgency: "NoticeDueSoon", amountAtRisk: 100, roundedMinutes: 15 }),
      entry({ urgency: "Lost", amountAtRisk: 0, grossAmount: 450 }),
      entry({ urgency: "Normal" }),
    ]);

    expect(summary.total).toBe(4);
    expect(summary.accruing).toBe(2);
    expect(summary.noticesDue).toBe(1);
    expect(summary.lost).toBe(1);
    expect(summary.amountAtRisk).toBe(400);
    expect(summary.amountLost).toBe(450);
  });

  it("counts an overdue notice as due", () => {
    const summary = summarizeDesk([entry({ urgency: "NoticeOverdue" })]);
    expect(summary.noticesDue).toBe(1);
  });

  it("handles an empty desk", () => {
    const summary = summarizeDesk([]);

    expect(summary.total).toBe(0);
    expect(summary.amountAtRisk).toBe(0);
  });
});

describe("matchesDeskFilter", () => {
  it("puts both notice states in the notice lane", () => {
    expect(matchesDeskFilter(entry({ urgency: "NoticeOverdue" }), "notice")).toBe(true);
    expect(matchesDeskFilter(entry({ urgency: "NoticeDueSoon" }), "notice")).toBe(true);
  });

  it("keeps a stop still inside its free time out of the accruing lane", () => {
    expect(matchesDeskFilter(entry({ urgency: "FreeTimeEnding" }), "accruing")).toBe(false);
    expect(matchesDeskFilter(entry({ urgency: "FreeTimeEnding" }), "free")).toBe(true);
  });

  it("matches everything in the all lane", () => {
    expect(matchesDeskFilter(entry({ urgency: "Lost" }), "all")).toBe(true);
  });
});

describe("countDeskFilters", () => {
  it("assigns every stop to exactly one lane", () => {
    const counts = countDeskFilters([
      entry({ urgency: "NoticeOverdue" }),
      entry({ urgency: "NoticeDueSoon" }),
      entry({ urgency: "Accruing" }),
      entry({ urgency: "FreeTimeEnding" }),
      entry({ urgency: "Normal" }),
      entry({ urgency: "Lost" }),
    ]);

    expect(counts.all).toBe(6);
    expect(counts.notice).toBe(2);
    expect(counts.accruing).toBe(1);
    expect(counts.free).toBe(2);
    expect(counts.lost).toBe(1);
    expect(counts.notice + counts.accruing + counts.free + counts.lost).toBe(counts.all);
  });

  it("reports zeroes for an empty desk", () => {
    expect(countDeskFilters([])).toEqual({
      all: 0,
      notice: 0,
      accruing: 0,
      free: 0,
      lost: 0,
    });
  });
});

describe("matchesDeskSearch", () => {
  const target = entry({
    locationName: "Kroger DC #42",
    customerName: "Kroger Co",
    shipmentProNumber: "PRO10482",
    stopType: "Delivery",
  });

  it("matches an empty or whitespace term", () => {
    expect(matchesDeskSearch(target, "")).toBe(true);
    expect(matchesDeskSearch(target, "   ")).toBe(true);
  });

  it("matches the facility, the customer, the PRO, and the stop type case-insensitively", () => {
    expect(matchesDeskSearch(target, "kroger dc")).toBe(true);
    expect(matchesDeskSearch(target, "KROGER CO")).toBe(true);
    expect(matchesDeskSearch(target, "10482")).toBe(true);
    expect(matchesDeskSearch(target, "deliv")).toBe(true);
  });

  it("rejects a term that appears nowhere", () => {
    expect(matchesDeskSearch(target, "walmart")).toBe(false);
  });
});

describe("nextDeskDeadline", () => {
  it("prefers the notice deadline when it lands first", () => {
    const target = entry({ noticeDeadlineAt: NOW + 1800, freeTimeExpiresAt: NOW + 7200 });
    expect(nextDeskDeadline(target)).toBe(NOW + 1800);
  });

  it("falls back to free time expiry when there is no notice", () => {
    expect(nextDeskDeadline(entry({ freeTimeExpiresAt: NOW + 7200 }))).toBe(NOW + 7200);
  });

  it("keeps free time expiry when the notice deadline is later", () => {
    const target = entry({ noticeDeadlineAt: NOW + 9000, freeTimeExpiresAt: NOW + 7200 });
    expect(nextDeskDeadline(target)).toBe(NOW + 7200);
  });
});

describe("sortDeskEntriesBy", () => {
  it("orders by exposure, largest first", () => {
    const sorted = sortDeskEntriesBy(
      [
        entry({ urgency: "Normal", amountAtRisk: 10 }),
        entry({ urgency: "Lost", amountAtRisk: 900 }),
        entry({ urgency: "Accruing", amountAtRisk: 200 }),
      ],
      "exposure",
    );

    expect(sorted.map((item) => item.amountAtRisk)).toEqual([900, 200, 10]);
  });

  it("orders by dwell, longest on site first", () => {
    const sorted = sortDeskEntriesBy(
      [
        entry({ clockStartAt: NOW - 600 }),
        entry({ clockStartAt: NOW - 7200 }),
        entry({ clockStartAt: NOW - 3600 }),
      ],
      "dwell",
    );

    expect(sorted.map((item) => item.occurrence.clockStartAt)).toEqual([
      NOW - 7200,
      NOW - 3600,
      NOW - 600,
    ]);
  });

  it("orders by the soonest deadline of either kind", () => {
    const sorted = sortDeskEntriesBy(
      [
        entry({ freeTimeExpiresAt: NOW + 3600 }),
        entry({ freeTimeExpiresAt: NOW + 7200, noticeDeadlineAt: NOW + 600 }),
      ],
      "deadline",
    );

    expect(nextDeskDeadline(sorted[0]!)).toBe(NOW + 600);
  });

  it("breaks ties by urgency", () => {
    const sorted = sortDeskEntriesBy(
      [
        entry({ urgency: "Lost", amountAtRisk: 100 }),
        entry({ urgency: "NoticeOverdue", amountAtRisk: 100 }),
      ],
      "exposure",
    );

    expect(sorted[0]?.urgency).toBe("NoticeOverdue");
  });

  it("delegates the urgency sort and leaves the input untouched", () => {
    const input = [entry({ urgency: "Accruing" }), entry({ urgency: "NoticeOverdue" })];

    expect(sortDeskEntriesBy(input, "urgency")[0]?.urgency).toBe("NoticeOverdue");
    expect(input[0]?.urgency).toBe("Accruing");
  });
});

describe("deskClockTrack", () => {
  it("starts empty with the free-time boundary short of the end", () => {
    const track = deskClockTrack(entry(), NOW);

    expect(track.elapsedPercent).toBe(0);
    expect(track.accruedPercent).toBe(0);
    expect(track.isAccruing).toBe(false);
    expect(track.freePercent).toBeCloseTo(92.6, 0);
    expect(track.noticePercent).toBeNull();
  });

  it("splits free time from accrual once the clock runs past it", () => {
    const track = deskClockTrack(entry(), NOW + 10_800);

    expect(track.isAccruing).toBe(true);
    expect(track.freePercent).toBeCloseTo(61.7, 0);
    expect(track.elapsedPercent).toBeCloseTo(92.6, 0);
    expect(track.accruedPercent).toBeCloseTo(30.9, 0);
  });

  it("freezes a departed stop at its stop time", () => {
    const track = deskClockTrack(entry({ clockStopAt: NOW + 3600 }), NOW + 100_000);

    expect(track.elapsedPercent).toBeCloseTo(46.3, 0);
    expect(track.isAccruing).toBe(false);
  });

  it("places the notice deadline and reports once it has passed", () => {
    const target = entry({ noticeDeadlineAt: NOW + 1800 });

    expect(deskClockTrack(target, NOW).noticePercent).toBeCloseTo(23.1, 0);
    expect(deskClockTrack(target, NOW).noticePassed).toBe(false);
    expect(deskClockTrack(target, NOW + 3600).noticePassed).toBe(true);
  });

  it("never reports a negative elapsed share for a clock that has not started", () => {
    const track = deskClockTrack(entry(), NOW - 3600);

    expect(track.elapsedPercent).toBe(0);
    expect(track.accruedPercent).toBe(0);
  });

  it("keeps the whole bar within bounds for a stop far past every deadline", () => {
    const track = deskClockTrack(entry({ noticeDeadlineAt: NOW + 600 }), NOW + 864_000);

    expect(track.elapsedPercent).toBeLessThanOrEqual(100);
    expect(track.freePercent + track.accruedPercent).toBeLessThanOrEqual(100);
  });
});

describe("deskFloorStats", () => {
  it("averages and ranks dwell across the floor", () => {
    const stats = deskFloorStats(
      [
        entry({ clockStartAt: NOW - 3600, locationName: "Kroger DC" }),
        entry({ clockStartAt: NOW - 7200, locationName: "Target RDC" }),
      ],
      NOW,
    );

    expect(stats.averageOnSiteMinutes).toBe(90);
    expect(stats.longestOnSiteMinutes).toBe(120);
    expect(stats.longestLocationName).toBe("Target RDC");
  });

  it("measures a departed stop to its stop time, not to now", () => {
    const stats = deskFloorStats(
      [entry({ clockStartAt: NOW - 7200, clockStopAt: NOW - 3600 })],
      NOW,
    );

    expect(stats.longestOnSiteMinutes).toBe(60);
  });

  it("reports zeroes for an empty floor", () => {
    expect(deskFloorStats([], NOW)).toEqual({
      averageOnSiteMinutes: 0,
      longestOnSiteMinutes: 0,
      longestLocationName: "",
    });
  });
});
