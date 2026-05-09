import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { KpiGoalBar } from "../kpi-goal-bar";

afterEach(() => cleanup());

describe("KpiGoalBar", () => {
  it("places the target tick at target/max * 100% and renders warning fill when actual exceeds target", () => {
    const { container } = render(
      <KpiGoalBar
        label="Empty mile %"
        value="11.8"
        unit="%"
        actual={11.8}
        target={10}
        max={20}
      />,
    );
    const fill = container.querySelector<HTMLDivElement>(".absolute.inset-y-0");
    const tick = container.querySelector<HTMLDivElement>("[title='Target 10%']");
    expect(fill).not.toBeNull();
    expect(tick).not.toBeNull();
    expect(fill?.style.background).toBe("var(--warning)");
    expect(parseFloat(fill?.style.width ?? "")).toBeCloseTo(59, 5);
    expect(tick?.style.left).toBe("calc(50% - 1px)");
  });

  it("renders success fill when actual is at or below target", () => {
    const { container } = render(
      <KpiGoalBar label="X" value="8.0" unit="%" actual={8} target={10} max={20} />,
    );
    const fill = container.querySelector<HTMLDivElement>(".absolute.inset-y-0");
    expect(fill?.style.background).toBe("var(--success)");
  });
});
