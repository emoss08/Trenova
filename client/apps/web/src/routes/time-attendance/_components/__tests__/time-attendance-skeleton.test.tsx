import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { TimeAttendanceSkeleton } from "../time-attendance-skeleton";

afterEach(cleanup);

const hidden = { hidden: true } as const;

function stripCells(root: HTMLElement) {
  const strip = within(root).getByRole("group", hidden);
  return strip.firstElementChild?.children.length;
}

describe("TimeAttendanceSkeleton", () => {
  // The skeleton is the route's Suspense fallback and the console's own first
  // read. It has to be the console's shape, or the page jumps when either lands.
  it("outlines the strip, the tabs and the board of everybody on the clock", () => {
    render(<TimeAttendanceSkeleton showPayroll />);

    const root = screen.getByLabelText("Loading time and attendance");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(stripCells(root)).toBe(4);
    expect(within(root).getAllByRole("listitem", hidden)).toHaveLength(3);
  });

  it("leaves the payroll figure out for somebody who may not run payroll", () => {
    render(<TimeAttendanceSkeleton showPayroll={false} />);

    expect(stripCells(screen.getByLabelText("Loading time and attendance"))).toBe(3);
  });

  it("keeps its stand-ins out of the accessibility tree and carries no text", () => {
    render(<TimeAttendanceSkeleton showPayroll />);

    expect(screen.queryByRole("group")).toBeNull();
    expect(screen.queryAllByRole("listitem")).toHaveLength(0);
    expect(screen.getByLabelText("Loading time and attendance")).toHaveTextContent("");
  });
});
