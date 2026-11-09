import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { Slider } from "@trenova/shared/components/ui/slider";

afterEach(cleanup);

function Days() {
  const [days, setDays] = useState(30);
  return (
    <>
      <span id="days-label">Days left</span>
      <Slider
        aria-labelledby="days-label"
        min={0}
        max={90}
        value={days}
        onValueChange={setDays}
        marks={[0, 30, 60, 90].map((d) => ({ value: d, label: `${d}d` }))}
      />
      <output>{days}</output>
    </>
  );
}

describe("Slider", () => {
  // The labels once sat in a justify-between row, which spaces boxes rather than centring them
  // on their values, so 30d and 60d read away from where the thumb stopped for them.
  it("centres each mark on its value along the track", () => {
    render(<Days />);

    expect(screen.getByText("0d").style.left).toBe("0%");
    expect(Number.parseFloat(screen.getByText("30d").style.left)).toBeCloseTo(33.333, 2);
    expect(Number.parseFloat(screen.getByText("60d").style.left)).toBeCloseTo(66.667, 2);
    expect(screen.getByText("90d").style.left).toBe("100%");
    for (const label of ["0d", "30d", "60d", "90d"]) {
      expect(screen.getByText(label).className.split(/\s+/)).toContain("-translate-x-1/2");
    }
  });

  it("is named by its label and moves by a step from the keyboard", () => {
    render(<Days />);

    const slider = screen.getByRole("slider", { name: "Days left" });
    expect(slider.getAttribute("aria-valuenow")).toBe("30");

    fireEvent.keyDown(slider, { key: "ArrowRight" });
    expect(screen.getByRole("status").textContent).toBe("31");
  });
});
