import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { foldArrivals } from "../intelligence-model";
import { ArrivalGrid } from "../when-section";

beforeEach(() => {
  vi.useFakeTimers({
    toFake: ["setTimeout", "clearTimeout", "requestAnimationFrame", "cancelAnimationFrame"],
  });
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("ArrivalGrid", () => {
  const grid = foldArrivals([
    // Monday 6–8a: five arrivals, three detained.
    { weekday: 1, hour: 6, stopCount: 1, detainedCount: 1 },
    { weekday: 1, hour: 7, stopCount: 4, detainedCount: 2 },
    // Tuesday 8–10a: one arrival, detained.
    { weekday: 2, hour: 9, stopCount: 1, detainedCount: 1 },
  ]);

  it("shows the share detained in a cell with enough arrivals", () => {
    render(<ArrivalGrid grid={grid} />);

    expect(
      screen.getByRole("cell", { name: "Mon 6–8a: 5 arrivals, 60% detained" }),
    ).toHaveTextContent("60%");
  });

  it("still shows the share of a cell with a single arrival", () => {
    render(<ArrivalGrid grid={grid} />);

    const cell = screen.getByRole("cell", { name: "Tue 8–10a: 1 arrival, 100% detained" });
    expect(cell).toHaveTextContent("100%");
  });

  it("hatches an hour nobody arrived in and marks it with a dash", () => {
    render(<ArrivalGrid grid={grid} />);

    const cell = screen.getByRole("cell", { name: "Wed 12–6a: no arrivals" });
    expect(cell).toHaveTextContent("–");
  });

  it("opens a tooltip with the arrivals and the share detained on hover", () => {
    render(<ArrivalGrid grid={grid} />);

    fireEvent.pointerEnter(screen.getByRole("cell", { name: /^Mon 6–8a/ }), { clientX: 100 });
    act(() => {
      vi.runAllTimers();
    });

    const tip = screen.getByRole("tooltip");
    expect(tip).toHaveTextContent("Mon 6–8a");
    expect(tip).toHaveTextContent("5 arrivals");
    expect(tip).toHaveTextContent("60% detained");
  });

  it("says there were no arrivals when an empty hour is hovered", () => {
    render(<ArrivalGrid grid={grid} />);

    fireEvent.pointerEnter(screen.getByRole("cell", { name: "Wed 12–6a: no arrivals" }), {
      clientX: 100,
    });
    act(() => {
      vi.runAllTimers();
    });

    expect(screen.getByRole("tooltip")).toHaveTextContent("Wed 12–6aNo arrivals");
  });

  it("captions the hatched cells", () => {
    render(<ArrivalGrid grid={grid} />);

    expect(screen.getByText("Hatched cells had no arrivals.")).toBeInTheDocument();
  });
});
