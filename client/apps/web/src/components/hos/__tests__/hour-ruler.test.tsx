import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HourRuler, hourRulerTone, hourRulerUsed } from "../hour-ruler";

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;

function fill(meter: HTMLElement): HTMLElement {
  const element = meter.firstElementChild;
  if (!(element instanceof HTMLElement)) throw new Error("no fill");
  return element;
}

describe("hourRulerTone", () => {
  it("is plain from an hour left, warning under it and danger at none", () => {
    expect(hourRulerTone(HOUR)).toBe("normal");
    expect(hourRulerTone(HOUR - 1)).toBe("warning");
    expect(hourRulerTone(1)).toBe("warning");
    expect(hourRulerTone(0)).toBe("danger");
    expect(hourRulerTone(-5 * MINUTE)).toBe("danger");
  });
});

describe("hourRulerUsed", () => {
  it("is the part of the limit gone, held between none and all", () => {
    expect(hourRulerUsed(6 * HOUR + 12 * MINUTE, 11 * HOUR)).toBeCloseTo(288 / 660, 10);
    expect(hourRulerUsed(-HOUR, 11 * HOUR)).toBe(1);
    expect(hourRulerUsed(12 * HOUR, 11 * HOUR)).toBe(0);
  });
});

describe("HourRuler", () => {
  it("says the time left against its limit and fills the part used", async () => {
    render(
      <HourRuler
        name="Drive"
        remainingMs={6 * HOUR + 12 * MINUTE}
        limitMs={11 * HOUR}
        caption="left of 11h"
      />,
    );

    expect(screen.getByText("Drive")).toBeInTheDocument();
    expect(screen.getByText("6h 12m")).toBeInTheDocument();
    expect(screen.getByText("left of 11h")).toBeInTheDocument();
    const meter = screen.getByRole("meter", { name: "Drive" });
    expect(meter).toHaveAttribute("aria-valuenow", "372");
    expect(meter).toHaveAttribute("aria-valuemax", "660");
    expect(meter).toHaveAttribute("aria-valuetext", "6h 12m left");
    expect(meter).toHaveAttribute("data-tone", "normal");
    expect(meter.style.getPropertyValue("--tick")).toBe("calc(100% / 11)");
    expect(fill(meter)).toHaveClass("bg-ink");
    await waitFor(() =>
      expect(fill(meter).style.transform).toBe(
        `scaleX(${(11 * HOUR - 372 * MINUTE) / (11 * HOUR)})`,
      ),
    );
  });

  it("turns warning under an hour left and says the minutes alone", () => {
    render(
      <HourRuler
        name="Break"
        remainingMs={42 * MINUTE}
        limitMs={8 * HOUR}
        caption="of 8h until break"
        size="large"
      />,
    );

    const meter = screen.getByRole("meter", { name: "Break" });
    expect(screen.getByText("42m")).toBeInTheDocument();
    expect(meter).toHaveAttribute("data-tone", "warning");
    expect(fill(meter)).toHaveClass("bg-warning");
  });

  it("turns danger with no time left", () => {
    render(<HourRuler name="Shift" remainingMs={0} limitMs={14 * HOUR} caption="of 14h shift" />);

    const meter = screen.getByRole("meter", { name: "Shift" });
    expect(screen.getByText("0m")).toBeInTheDocument();
    expect(meter).toHaveAttribute("data-tone", "danger");
    expect(fill(meter)).toHaveClass("bg-danger");
  });

  it("rules the cycle in tens of hours when told to", () => {
    render(
      <HourRuler
        name="Cycle"
        remainingMs={31 * HOUR + 20 * MINUTE}
        limitMs={70 * HOUR}
        tickMs={10 * HOUR}
        caption="of 70h cycle"
        size="large"
        note="38h 40m on duty since her Nov 19 restart"
      />,
    );

    expect(screen.getByRole("meter", { name: "Cycle" }).style.getPropertyValue("--tick")).toBe(
      "calc(100% / 7)",
    );
    expect(screen.getByText("31h 20m")).toBeInTheDocument();
    expect(screen.getByText("38h 40m on duty since her Nov 19 restart")).toBeInTheDocument();
  });

  it("leaves the note off the compact size", () => {
    render(
      <HourRuler
        name="Drive"
        remainingMs={2 * HOUR}
        limitMs={11 * HOUR}
        caption="left of 11h"
        note="4h 48m driven today"
      />,
    );

    expect(screen.queryByText("4h 48m driven today")).not.toBeInTheDocument();
  });
});
