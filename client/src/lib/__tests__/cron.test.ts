import { describe, expect, it } from "vitest";
import { describeCron } from "../cron";

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
