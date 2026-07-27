import { describe, expect, it } from "vitest";
import {
  formatCountdown,
  formatDetentionMinutes,
  freeTimeConsumedPercent,
  sortDeskEntries,
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
    freeTimeExpiresAt?: number;
  } = {},
): DeskEntry {
  const {
    urgency = "Normal",
    amountAtRisk = 0,
    grossAmount = amountAtRisk,
    roundedMinutes = 0,
    freeMinutesGranted = 120,
    clockStartAt = NOW,
    freeTimeExpiresAt = NOW + 7200,
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
      freeTimeExpiresAt,
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

describe("freeTimeConsumedPercent", () => {
  it("is zero at the moment the clock starts", () => {
    expect(freeTimeConsumedPercent(entry(), NOW)).toBe(0);
  });

  it("is half way through the budget", () => {
    expect(freeTimeConsumedPercent(entry(), NOW + 3600)).toBe(50);
  });

  it("clamps once free time is exhausted", () => {
    expect(freeTimeConsumedPercent(entry(), NOW + 36000)).toBe(100);
  });

  it("treats a zero allowance as immediately consumed", () => {
    expect(freeTimeConsumedPercent(entry({ freeMinutesGranted: 0 }), NOW)).toBe(100);
  });

  it("never reports negative consumption for a future clock start", () => {
    expect(freeTimeConsumedPercent(entry(), NOW - 3600)).toBe(0);
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

    expect(sorted.map((e) => e.urgency)).toEqual([
      "NoticeDueSoon",
      "Lost",
      "Normal",
    ]);
  });

  it("does not mutate the input", () => {
    const input = [
      entry({ urgency: "Accruing" }),
      entry({ urgency: "NoticeOverdue" }),
    ];
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
