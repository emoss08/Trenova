import { render, screen } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import { describe, expect, it } from "vitest";
import { MoneyLine } from "../expanded-row/money-line";

function figure(label: string): string | null {
  return screen.getByText(label).nextElementSibling?.textContent ?? null;
}

describe("MoneyLine", () => {
  it("prices fuel charges the way the server does, a percentage one against the line haul", () => {
    const shipment = {
      totalChargeAmount: "2390.50",
      freightChargeAmount: "2000",
      additionalCharges: [
        { method: "Percentage", amount: "12.5", unit: 1, fuelSurchargeProgramId: "fsp_1" },
        { method: "PerUnit", amount: "45", unit: 0, fuelSurchargeProgramId: "fsp_2" },
        { method: "Flat", amount: "65.25", unit: 2, fuelSurchargeDetail: { programCode: "FSC" } },
        { method: "Flat", amount: "75", unit: 1 },
      ],
    } as unknown as Shipment;

    render(<MoneyLine shipment={shipment} />);

    // 12.5% of 2,000 is 250; a per-unit charge with no units is nothing; 65.25 twice is 130.50.
    expect(figure("Fuel")).toBe("$380.50");
    expect(figure("Linehaul")).toBe("$2,000.00");
    expect(figure("Accessorials")).toBe("$10.00");
  });
});
