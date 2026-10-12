import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CountdownCell } from "../countdown-cell";

const NOW = Date.UTC(2026, 9, 8, 12, 0, 0);
const nowSeconds = NOW / 1000;

describe("CountdownCell", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(NOW);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows a dash when there is nothing to count down to", () => {
    render(<CountdownCell target={null} />);

    expect(screen.getByText("—")).toBeTruthy();
  });

  it("counts down to a moment ahead in a neutral tone", () => {
    render(<CountdownCell target={nowSeconds + 3 * 3600 + 20 * 60} />);

    const cell = screen.getByText(/^in /);
    expect(cell.textContent).toBe("in 3h 20m");
    expect(cell.dataset.tone).toBe("neutral");
  });

  it("warns inside the warning window and turns to danger inside the danger window", () => {
    const { rerender } = render(<CountdownCell target={nowSeconds + 45 * 60} />);
    expect(screen.getByText(/^in /).dataset.tone).toBe("warning");

    rerender(<CountdownCell target={nowSeconds + 10 * 60} />);
    expect(screen.getByText(/^in /).dataset.tone).toBe("danger");
  });

  it("says how late a moment that has passed is", () => {
    render(<CountdownCell target={nowSeconds - 90 * 60} />);

    const cell = screen.getByText(/late$/);
    expect(cell.textContent).toBe("1h 30m late");
    expect(cell.dataset.tone).toBe("danger");
  });

  it("honours a column's own thresholds", () => {
    render(
      <CountdownCell target={nowSeconds + 3 * 3600} warnWithinMinutes={240} dangerWithinMinutes={0} />,
    );

    expect(screen.getByText(/^in /).dataset.tone).toBe("warning");
  });

  it("moves with the minute clock", () => {
    render(<CountdownCell target={nowSeconds + 10 * 60} />);
    expect(screen.getByText(/^in /).textContent).toBe("in 10m");

    act(() => {
      vi.advanceTimersByTime(60_000);
    });

    expect(screen.getByText(/^in /).textContent).toBe("in 9m");
  });

  it("shows seconds in the last minute and moves every second", () => {
    render(<CountdownCell target={nowSeconds + 30} />);
    expect(screen.getByText(/^in /).textContent).toBe("in 30s");

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(screen.getByText(/^in /).textContent).toBe("in 29s");
  });
});
