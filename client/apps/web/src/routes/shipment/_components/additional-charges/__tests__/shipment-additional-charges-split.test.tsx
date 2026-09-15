import { cleanup, render, screen } from "@testing-library/react";
import type { ChargeAllocation } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it } from "vitest";
import { ChargePayerChip } from "@/components/billing/charge-payer-chip";
import { collectChargeErrorMessages } from "../shipment-additional-charges";

afterEach(cleanup);

describe("collectChargeErrorMessages", () => {
  // A split that does not add up is reported on the allocation array itself,
  // which react-hook-form stores under `allocations.root`. The row tooltip has
  // to surface it, not print "Invalid".
  it("includes the split's array-level message", () => {
    expect(
      collectChargeErrorMessages({
        allocations: { root: { message: "Allocations must total 100%" } },
      }),
    ).toEqual(["Allocations must total 100%"]);
  });

  it("includes a row-level payer message and the plain field messages", () => {
    expect(
      collectChargeErrorMessages({
        ref: { name: "x" },
        amount: { message: "Amount is required" },
        allocations: [undefined, { billToCustomerId: { message: "Payer is required" } }],
      }),
    ).toEqual(["Amount is required", "Payer is required"]);
  });

  it("falls back to a generic message when the tree carries none", () => {
    expect(collectChargeErrorMessages({ amount: { type: "min" } })).toEqual(["Invalid"]);
  });
});

describe("ChargePayerChip", () => {
  it("renders nothing for a charge billed to the shipment's payer", () => {
    const { container } = render(<ChargePayerChip allocations={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("names a single alternate payer", () => {
    render(
      <ChargePayerChip
        allocations={[
          {
            billToCustomerId: "cus_amd",
            method: "Percent",
            percent: 100,
            billToCustomer: { id: "cus_amd", name: "AMD", code: "AMD" },
          } as ChargeAllocation,
        ]}
      />,
    );
    expect(screen.getByTestId("charge-payer-chip")).toHaveTextContent("Bill to AMD – AMD");
  });

  it("counts the ways for a multi-payer split", () => {
    render(
      <ChargePayerChip
        allocations={[
          { billToCustomerId: "cus_amd", method: "Percent", percent: 60 } as ChargeAllocation,
          { billToCustomerId: "cus_intel", method: "Percent", percent: 40 } as ChargeAllocation,
        ]}
      />,
    );
    expect(screen.getByTestId("charge-payer-chip")).toHaveTextContent("Split 2 ways");
  });
});
