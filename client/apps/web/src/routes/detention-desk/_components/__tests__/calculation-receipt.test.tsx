import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { CalculationTrace } from "@trenova/shared/types/detention";
import { CalculationReceipt } from "../calculation-receipt";

afterEach(cleanup);

function trace(driverPay: number | null): CalculationTrace {
  return {
    engineVersion: "1.0.0",
    computedAt: 1_767_225_600,
    deterministic: true,
    steps: [
      {
        kind: "DriverPay",
        label: "Driver detention pay",
        detail: "1h 15m beyond 2h of pay free time",
        term: "",
        amount: driverPay,
        isReduction: false,
      },
      {
        kind: "Final",
        label: "Billable detention",
        detail: "",
        term: "",
        amount: 225,
        isReduction: false,
      },
    ],
  } as CalculationTrace;
}

describe("CalculationReceipt", () => {
  it("draws the driver's pay as hidden when the server withheld it", () => {
    render(<CalculationReceipt trace={trace(null)} />);

    expect(
      screen
        .getByText("Hidden")
        .closest('[data-slot="locked-reference"]')
        ?.getAttribute("aria-description"),
    ).toBe("You don't have access to margins");
    expect(screen.getByText("$225.00")).toBeInTheDocument();
  });

  it("shows the driver's pay to a reader the server sent it to", () => {
    render(<CalculationReceipt trace={trace(75)} />);

    expect(screen.getByText("$75.00")).toBeInTheDocument();
    expect(screen.queryByText("Hidden")).toBeNull();
  });
});
