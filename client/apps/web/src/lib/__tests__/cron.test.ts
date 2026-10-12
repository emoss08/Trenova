import { describe, expect, it } from "vitest";
import { cronFiresAt, cronRunsBetween, describeCron, parseCronSchedule, wallClockAt } from "../cron";

describe("describeCron", () => {
  it("describes daily schedules", () => {
    expect(describeCron("0 8 * * *")).toBe("Daily at 8:00 AM");
    expect(describeCron("30 17 * * *")).toBe("Daily at 5:30 PM");
    expect(describeCron("0 0 * * *")).toBe("Daily at 12:00 AM");
    expect(describeCron("0 12 * * *")).toBe("Daily at 12:00 PM");
  });

  it("describes weekday schedules", () => {
    expect(describeCron("0 8 * * 1-5")).toBe("Weekdays at 8:00 AM");
  });

  it("describes weekly schedules", () => {
    expect(describeCron("0 8 * * 1")).toBe("Weekly on Monday at 8:00 AM");
    expect(describeCron("15 9 * * 0")).toBe("Weekly on Sunday at 9:15 AM");
    expect(describeCron("0 8 * * 7")).toBe("Weekly on Sunday at 8:00 AM");
    expect(describeCron("0 8 * * 1,5")).toBe("Weekly on Monday and Friday at 8:00 AM");
    expect(describeCron("0 8 * * 1,3,5")).toBe(
      "Weekly on Monday, Wednesday, and Friday at 8:00 AM",
    );
  });

  it("describes monthly schedules", () => {
    expect(describeCron("0 8 1 * *")).toBe("Monthly on the 1st at 8:00 AM");
    expect(describeCron("0 8 2 * *")).toBe("Monthly on the 2nd at 8:00 AM");
    expect(describeCron("0 8 3 * *")).toBe("Monthly on the 3rd at 8:00 AM");
    expect(describeCron("0 8 11 * *")).toBe("Monthly on the 11th at 8:00 AM");
    expect(describeCron("0 8 21 * *")).toBe("Monthly on the 21st at 8:00 AM");
  });

  it("falls back to null for shapes it cannot describe", () => {
    expect(describeCron("*/5 * * * *")).toBeNull();
    expect(describeCron("0 8 * 6 *")).toBeNull();
    expect(describeCron("0 8 1 * 1")).toBeNull();
    expect(describeCron("0 8,12 * * *")).toBeNull();
    expect(describeCron("not a cron")).toBeNull();
    expect(describeCron("")).toBeNull();
    expect(describeCron("0 25 * * *")).toBeNull();
    expect(describeCron("0 8 32 * *")).toBeNull();
    expect(describeCron("0 8 * * 8")).toBeNull();
  });
});

describe("parseCronSchedule", () => {
  it("reads lists, ranges, steps and names the way the server does", () => {
    const schedule = parseCronSchedule("*/15 6-8,17 * JAN-MAR mon-fri");
    expect(schedule).not.toBeNull();
    expect([...schedule!.minutes]).toEqual([0, 15, 30, 45]);
    expect([...schedule!.hours]).toEqual([6, 7, 8, 17]);
    expect([...schedule!.months]).toEqual([1, 2, 3]);
    expect([...schedule!.daysOfWeek]).toEqual([1, 2, 3, 4, 5]);
    expect(schedule!.anyDayOfMonth).toBe(true);
    expect(schedule!.anyDayOfWeek).toBe(false);
  });

  it("treats 7 as Sunday and a start with a step as running to the end", () => {
    expect([...parseCronSchedule("0 0 * * 7")!.daysOfWeek]).toEqual([0]);
    expect([...parseCronSchedule("50/5 * * * *")!.minutes]).toEqual([50, 55]);
  });

  it("refuses what the server would", () => {
    for (const bad of ["", "0 9 * *", "0 24 * * *", "60 1 * * *", "0 17 L * *", "0 9 * * 8", "*/0 * * * *", "5-1 * * * *"]) {
      expect(parseCronSchedule(bad)).toBeNull();
    }
  });
});

describe("cronFiresAt", () => {
  const at = { year: 2026, month: 10, day: 9, weekday: 5, hour: 9, minute: 0 };

  it("needs both days when either is a star, and either day when both are set", () => {
    expect(cronFiresAt(parseCronSchedule("0 9 * * 5")!, at)).toBe(true);
    expect(cronFiresAt(parseCronSchedule("0 9 * * 1")!, at)).toBe(false);
    expect(cronFiresAt(parseCronSchedule("0 9 1 * 5")!, at)).toBe(true);
    expect(cronFiresAt(parseCronSchedule("0 9 1 * 1")!, at)).toBe(false);
    expect(cronFiresAt(parseCronSchedule("0 9 9 * 1")!, at)).toBe(true);
  });
});

describe("cronRunsBetween", () => {
  it("finds weekday mornings in the agent's time zone, not the reader's", () => {
    const from = Date.UTC(2026, 9, 5, 0, 0) / 1000;
    const until = from + 7 * 86400;
    const runs = cronRunsBetween(parseCronSchedule("0 6 * * 1-5")!, "America/Chicago", from, until);
    expect(runs).toHaveLength(5);
    for (const run of runs) {
      const wall = wallClockAt(run, "America/Chicago");
      expect([wall.hour, wall.minute]).toEqual([6, 0]);
      expect(wall.weekday).toBeGreaterThanOrEqual(1);
      expect(wall.weekday).toBeLessThanOrEqual(5);
    }
    expect(new Date(runs[0]! * 1000).toISOString()).toBe("2026-10-05T11:00:00.000Z");
  });

  it("leaves out a run before the window opens", () => {
    const from = Date.UTC(2026, 9, 5, 11, 30) / 1000;
    const runs = cronRunsBetween(parseCronSchedule("0 * * * *")!, "UTC", from, from + 3 * 3600);
    expect(runs.map((run) => new Date(run * 1000).toISOString())).toEqual([
      "2026-10-05T12:00:00.000Z",
      "2026-10-05T13:00:00.000Z",
      "2026-10-05T14:00:00.000Z",
    ]);
  });
});
