import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { HolidayCalendarSkeleton, HolidayMonthsSkeleton } from "../holiday-calendar-skeleton";

afterEach(cleanup);

const DAYS_IN_2024 = 366;
const WEEKDAYS = 7;
const MONTHS = 12;

describe("HolidayCalendarSkeleton", () => {
  // A month card is as tall as its weeks, so the outline draws each month's
  // days from the weekday it starts on, the way the loaded card does.
  it("draws twelve months with a cell for every day of the year", () => {
    render(<HolidayMonthsSkeleton year={2024} />);

    const grid = screen.getByLabelText("Loading the calendar for 2024");
    expect(grid).toHaveAttribute("aria-busy", "true");
    expect(grid.children).toHaveLength(MONTHS);
    const cells = grid.querySelectorAll(".size-6");
    expect(cells).toHaveLength(DAYS_IN_2024);
    const september = grid.children[8];
    const pads = Array.from(september.querySelectorAll(".grid > span")).filter(
      (span) => span.childElementCount === 0,
    );
    expect(pads).toHaveLength(0);
    const june = grid.children[5];
    const junePads = Array.from(june.querySelectorAll(".grid > span")).filter(
      (span) => span.childElementCount === 0,
    );
    expect(junePads).toHaveLength(6);
    expect(june.querySelectorAll(".grid > span")).toHaveLength(WEEKDAYS + 6 + 30);
  });

  it("outlines this year under the toolbar and carries no text", () => {
    vi.useFakeTimers({ now: new Date(Date.UTC(2026, 9, 14)) });
    try {
      render(<HolidayCalendarSkeleton canCreate />);

      const root = screen.getByLabelText("Loading the holiday calendar");
      expect(root).toHaveAttribute("aria-busy", "true");
      expect(root).toHaveTextContent("");
      expect(root.querySelectorAll(".size-6")).toHaveLength(365);
    } finally {
      vi.useRealTimers();
    }
  });
});
