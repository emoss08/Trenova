import { describe, expect, it } from "vitest";
import {
  clockToMinutes,
  DAY_LABELS,
  dayMaskToDays,
  daysToDayMask,
  describeShiftPattern,
  formatShiftWindow,
  minutesToClock,
  rotaStateTone,
  startOfRotaWeek,
  summariseRotaRow,
  swapActionsFor,
} from "../scheduling";

describe("dayMaskToDays", () => {
  it("reads the mask from Sunday", () => {
    expect(dayMaskToDays("0111110")).toEqual([1, 2, 3, 4, 5]);
    expect(dayMaskToDays("1000001")).toEqual([0, 6]);
  });

  // A mask that is not seven characters of 0 or 1 is not a pattern, and
  // guessing at one would roster somebody onto days nobody chose.
  it("treats a malformed mask as no days", () => {
    expect(dayMaskToDays("MTWTF")).toEqual([]);
    expect(dayMaskToDays("")).toEqual([]);
    expect(dayMaskToDays("0111112")).toEqual([]);
  });
});

describe("daysToDayMask", () => {
  it("round-trips through the mask", () => {
    expect(daysToDayMask([1, 2, 3, 4, 5])).toBe("0111110");
    expect(daysToDayMask([])).toBe("0000000");
    expect(dayMaskToDays(daysToDayMask([0, 6]))).toEqual([0, 6]);
  });

  it("ignores a day that is not a day of the week", () => {
    expect(daysToDayMask([1, 9, -2])).toBe("0100000");
  });
});

describe("minutesToClock", () => {
  it("renders a time of day", () => {
    expect(minutesToClock(360)).toBe("06:00");
    expect(minutesToClock(0)).toBe("00:00");
    expect(minutesToClock(1439)).toBe("23:59");
  });

  // A shift that ends at 02:00 is ten hours long, not minus four, so the end
  // minute arrives past 1440 and has to wrap for display.
  it("wraps past midnight", () => {
    expect(minutesToClock(1800)).toBe("06:00");
  });
});

describe("formatShiftWindow", () => {
  it("shows the start, the end and the length", () => {
    expect(formatShiftWindow(360, 600)).toBe("06:00–16:00 · 10h");
  });

  it("marks a window that runs into the next day", () => {
    expect(formatShiftWindow(1200, 600)).toBe("20:00–06:00 (+1) · 10h");
  });

  it("renders a part hour", () => {
    expect(formatShiftWindow(360, 90)).toBe("06:00–07:30 · 1h 30m");
  });
});

describe("describeShiftPattern", () => {
  it("names the working days", () => {
    expect(describeShiftPattern("0111110", 1)).toBe("Mon–Fri");
    expect(describeShiftPattern("1000001", 1)).toBe("Sun, Sat");
  });

  it("collapses a full week", () => {
    expect(describeShiftPattern("1111111", 1)).toBe("Every day");
  });

  // The rotation is what makes an A/B pair alternate, so a pattern that
  // rotates has to say so or two drivers look double-booked.
  it("says how long the rotation is", () => {
    expect(describeShiftPattern("0111110", 2)).toBe("Mon–Fri · 2-week rotation");
  });

  it("says nothing when there are no working days", () => {
    expect(describeShiftPattern("0000000", 1)).toBe("No working days");
  });
});

describe("startOfRotaWeek", () => {
  it("resolves any day back to the Sunday that starts its week", () => {
    // 2026-09-09 is a Wednesday; 2026-09-06 is the Sunday before it.
    const wednesday = Date.UTC(2026, 8, 9, 14, 30) / 1000;
    const sunday = Date.UTC(2026, 8, 6) / 1000;
    expect(startOfRotaWeek(wednesday)).toBe(sunday);
    expect(startOfRotaWeek(sunday)).toBe(sunday);
  });
});

describe("summariseRotaRow", () => {
  const day = (state: string, scheduled: boolean, isConflict = false) => ({
    date: 0,
    state,
    scheduled,
    startMinute: 360,
    durationMinutes: 600,
    preference: null,
    assignmentCount: 0,
    isConflict,
  });

  it("counts the days worked and the hours they add up to", () => {
    const summary = summariseRotaRow({
      days: [
        day("Scheduled", true),
        day("Assigned", true),
        day("Off", false),
        day("TimeOff", true, true),
      ],
    });

    expect(summary.scheduledDays).toBe(3);
    expect(summary.conflicts).toBe(1);
    // The day somebody is signed off is not an hour anybody is working.
    expect(summary.hours).toBe(20);
  });

  it("is empty for an empty week", () => {
    expect(summariseRotaRow({ days: [] })).toEqual({
      scheduledDays: 0,
      conflicts: 0,
      hours: 0,
    });
  });
});

describe("rotaStateTone", () => {
  it("gives every state a tone", () => {
    for (const state of ["Off", "Scheduled", "Assigned", "TimeOff", "Leave", "Unavailable"]) {
      expect(rotaStateTone(state)).toBeTruthy();
    }
  });

  it("falls back rather than rendering nothing for an unknown state", () => {
    expect(rotaStateTone("Something else")).toBe(rotaStateTone("Off"));
  });
});

describe("DAY_LABELS", () => {
  it("is indexed from Sunday, like the mask", () => {
    expect(DAY_LABELS).toHaveLength(7);
    expect(DAY_LABELS[0]).toBe("Sun");
    expect(DAY_LABELS[6]).toBe("Sat");
  });
});

describe("swapActionsFor", () => {
  // Only the driver a swap was offered to can answer it, and only the one who
  // proposed it can take it back. Offering both to both sides would send the
  // server a request it is going to refuse.
  it("offers the answer to the driver it was offered to", () => {
    expect(swapActionsFor({ status: "Proposed", outgoing: false })).toEqual(["accept", "decline"]);
  });

  it("offers only a withdrawal to the driver who proposed it", () => {
    expect(swapActionsFor({ status: "Proposed", outgoing: true })).toEqual(["withdraw"]);
  });

  // A swap the colleague has accepted is waiting on the office, and the
  // proposer can still take it back until the office answers.
  it("lets the proposer withdraw a swap that has been accepted", () => {
    expect(swapActionsFor({ status: "Accepted", outgoing: true })).toEqual(["withdraw"]);
    expect(swapActionsFor({ status: "Accepted", outgoing: false })).toEqual([]);
  });

  it("offers nothing on a swap that has finished", () => {
    for (const status of ["Approved", "Rejected", "Declined", "Withdrawn"]) {
      expect(swapActionsFor({ status, outgoing: true })).toEqual([]);
      expect(swapActionsFor({ status, outgoing: false })).toEqual([]);
    }
  });
});

describe("clockToMinutes", () => {
  it("is the inverse of minutesToClock", () => {
    expect(clockToMinutes("06:30")).toBe(390);
    expect(clockToMinutes("00:00")).toBe(0);
    expect(clockToMinutes(minutesToClock(1439))).toBe(1439);
  });

  // A malformed time is refused rather than rostering somebody onto minute NaN.
  it("refuses anything that is not a clock time", () => {
    expect(clockToMinutes("")).toBe(-1);
    expect(clockToMinutes("25:00")).toBe(-1);
    expect(clockToMinutes("6:75")).toBe(-1);
    expect(clockToMinutes("six")).toBe(-1);
  });
});
