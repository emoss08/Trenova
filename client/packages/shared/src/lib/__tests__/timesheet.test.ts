import { describe, expect, it } from "vitest";
import {
  buildPayrollCsv,
  elapsedMinutes,
  formatHours,
  manualEntryDefaults,
  paidMinutesFor,
  timesheetActionsFor,
  timesheetStatusTone,
  totalHours,
} from "../timesheet";

describe("formatHours", () => {
  it("renders minutes as hours", () => {
    expect(formatHours(0)).toBe("0h");
    expect(formatHours(60)).toBe("1h");
    expect(formatHours(2400)).toBe("40h");
  });

  // A part hour is what overtime is usually made of, so rounding it away would
  // lose the very thing somebody is checking.
  it("keeps the part hour", () => {
    expect(formatHours(90)).toBe("1h 30m");
    expect(formatHours(45)).toBe("45m");
  });

  it("handles a negative or missing value rather than rendering NaN", () => {
    expect(formatHours(-30)).toBe("0h");
    expect(formatHours(Number.NaN)).toBe("0h");
  });
});

describe("totalHours", () => {
  it("adds the three buckets", () => {
    expect(totalHours({ regularMinutes: 2400, overtimeMinutes: 300, paidLeaveMinutes: 480 })).toBe(
      3180,
    );
  });
});

describe("elapsedMinutes", () => {
  it("counts from the punch to now", () => {
    const now = 1_800_000_000;
    expect(elapsedMinutes(now - 5400, now)).toBe(90);
  });

  // A clock read a moment before the punch landed would otherwise show a
  // negative shift.
  it("never goes negative", () => {
    expect(elapsedMinutes(1_800_000_000, 1_799_999_000)).toBe(0);
  });
});

describe("timesheetActionsFor", () => {
  // A worker hands their own week over and nothing else. Offering them approve
  // would send the server a request it is going to refuse.
  it("offers the worker only the hand-over", () => {
    expect(timesheetActionsFor("Open", { isOwner: true, canApprove: true })).toEqual(["submit"]);
  });

  it("offers a manager the decision on a submitted week", () => {
    expect(timesheetActionsFor("Submitted", { isOwner: false, canApprove: true })).toEqual([
      "approve",
      "reject",
    ]);
  });

  it("offers nothing on a submitted week to somebody who cannot approve", () => {
    expect(timesheetActionsFor("Submitted", { isOwner: false, canApprove: false })).toEqual([]);
  });

  // The point of rejecting a week is to have it fixed, so it goes back round.
  it("offers the hand-over again on a rejected week", () => {
    expect(timesheetActionsFor("Rejected", { isOwner: true, canApprove: false })).toEqual([
      "submit",
    ]);
  });

  it("offers nothing on a week that has gone to payroll", () => {
    expect(timesheetActionsFor("Locked", { isOwner: true, canApprove: true })).toEqual([]);
  });
});

describe("timesheetStatusTone", () => {
  it("gives every status a tone", () => {
    for (const status of ["Open", "Submitted", "Approved", "Rejected", "Locked"]) {
      expect(timesheetStatusTone(status).label).toBeTruthy();
    }
  });

  it("falls back rather than rendering nothing", () => {
    expect(timesheetStatusTone("Something else")).toEqual(timesheetStatusTone("Open"));
  });
});

describe("buildPayrollCsv", () => {
  const rows = [
    {
      workerId: "wrk_1",
      workerName: "Ada Byron",
      periodStart: 1_800_000_000,
      periodEnd: 1_800_604_800,
      regularMinutes: 2400,
      overtimeMinutes: 300,
      paidLeaveMinutes: 480,
    },
  ];

  it("writes a header and one line per week, in hours", () => {
    const csv = buildPayrollCsv(rows);
    const lines = csv.trim().split("\n");

    expect(lines[0]).toBe(
      "worker_id,worker_name,period_start,period_end,regular_hours,overtime_hours,paid_leave_hours",
    );
    expect(lines[1]).toContain("wrk_1,Ada Byron,");
    // Payroll systems take hours to two places, not minutes.
    expect(lines[1]).toContain(",40.00,5.00,8.00");
  });

  // A name with a comma in it would otherwise split into two columns and shift
  // everybody's hours one place to the right.
  it("quotes a value containing a comma or a quote", () => {
    const csv = buildPayrollCsv([{ ...rows[0], workerName: 'Byron, Ada "Countess"' }]);
    expect(csv).toContain('"Byron, Ada ""Countess"""');
  });
});

describe("paidMinutesFor", () => {
  it("takes the unpaid break off the span", () => {
    expect(paidMinutesFor(1_000_000, 1_000_000 + 8 * 3600, 30)).toBe(450);
  });

  // The dialog previews the figure while somebody is still typing, so a
  // half-entered finish time must never read as negative pay.
  it("never goes below zero while the times are still being typed", () => {
    expect(paidMinutesFor(1_000_000, 900_000, 0)).toBe(0);
    expect(paidMinutesFor(1_000_000, 1_000_000 + 600, 30)).toBe(0);
  });
});

describe("manualEntryDefaults", () => {
  const dayStart = 1_800_000_000;

  it("starts a fresh entry on a working day rather than at midnight", () => {
    const values = manualEntryDefaults(null, { dayStart, now: dayStart + 10 * 3600 });
    expect(values.clockedInAt).toBe(dayStart + 8 * 3600);
    expect(values.clockedOutAt).toBe(dayStart + 16 * 3600);
    expect(values.breakMinutes).toBe(0);
    expect(values.reason).toBe("");
  });

  // A correction starts from what was punched, but the reason is the reason
  // for *this* change, not the one recorded last time.
  it("copies the punches of the entry being corrected and leaves the reason blank", () => {
    const values = manualEntryDefaults(
      {
        clockedInAt: dayStart + 7 * 3600,
        clockedOutAt: dayStart + 15 * 3600,
        breakMinutes: 45,
        note: "Yard shift",
        editReason: "Missed clock-out",
      },
      { dayStart, now: dayStart + 20 * 3600 },
    );
    expect(values).toEqual({
      clockedInAt: dayStart + 7 * 3600,
      clockedOutAt: dayStart + 15 * 3600,
      breakMinutes: 45,
      note: "Yard shift",
      reason: "",
    });
  });

  it("closes a still-open punch at the current time", () => {
    const now = dayStart + 11 * 3600;
    const values = manualEntryDefaults(
      {
        clockedInAt: dayStart + 6 * 3600,
        clockedOutAt: null,
        breakMinutes: 0,
        note: null,
        editReason: null,
      },
      { dayStart, now },
    );
    expect(values.clockedOutAt).toBe(now);
  });
});
