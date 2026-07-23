import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { KpiRing } from "../kpi-ring";

afterEach(() => cleanup());

function getProgressStroke(container: HTMLElement): string | null {
  const circles = container.querySelectorAll("circle");
  return circles[1]?.getAttribute("stroke") ?? null;
}

describe("KpiRing", () => {
  it("renders success ring color when value meets the target", () => {
    const { container } = render(
      <KpiRing label="On-time" value="96.0" unit="%" ringValue={96} target={96} />,
    );
    expect(getProgressStroke(container)).toBe("var(--success)");
  });

  it("renders warning ring color when value is below the target", () => {
    const { container } = render(
      <KpiRing label="On-time" value="94.2" unit="%" ringValue={94.2} target={96} />,
    );
    expect(getProgressStroke(container)).toBe("var(--warning)");
  });

  it("renders success ring color when no target is given", () => {
    const { container } = render(<KpiRing label="X" value="50" ringValue={50} />);
    expect(getProgressStroke(container)).toBe("var(--success)");
  });
});
