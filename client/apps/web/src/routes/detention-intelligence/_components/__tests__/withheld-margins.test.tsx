import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { CustomerDetentionStat } from "@trenova/shared/types/detention";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { CustomerMargin } from "../customer-margin";
import { DetentionLedger } from "../detention-ledger";
import type { DetentionRollup } from "../use-detention-intelligence";

afterEach(() => {
  cleanup();
  clearPermissions();
});

const withheldRollup: DetentionRollup = {
  facilityCount: 2,
  stopCount: 10,
  breachCount: 4,
  billed: 1200,
  driverPay: null,
  netMargin: null,
  waived: 150,
  disputeCount: 1,
  suppressedCount: 0,
  truncated: false,
};

function customer(overrides: Partial<CustomerDetentionStat> = {}): CustomerDetentionStat {
  return {
    customerId: "cus_01",
    customerName: "Kroger",
    stopCount: 4,
    breachCount: 2,
    billedAmount: 600,
    driverPayAmount: 250,
    netMargin: 350,
    waivedAmount: 0,
    disputeCount: 0,
    suppressedCount: 0,
    ...overrides,
  };
}

describe("detention intelligence without the margins field group", () => {
  it("draws the ledger's driver pay and margin as hidden, keeping what was billed", () => {
    render(<DetentionLedger rollup={withheldRollup} />);

    expect(screen.getByText("$1,200.00")).toBeInTheDocument();
    expect(screen.getAllByText("Hidden")).toHaveLength(3);
    expect(screen.queryByText("$0.00")).toBeNull();
    expect(
      screen.getByText(
        "How much of each billed detention dollar survives driver pay needs access to margins.",
      ),
    ).toBeInTheDocument();
  });

  it("withholds the whole customer margin panel", () => {
    seedPermissions({});
    render(
      <CustomerMargin
        rows={[customer({ driverPayAmount: null, netMargin: null })]}
        isLoading={false}
        isError={false}
        onRetry={() => undefined}
      />,
    );

    expect(
      screen.getByText("Detention margin by customer needs access to margins."),
    ).toBeInTheDocument();
    expect(screen.queryByText("Kroger")).toBeNull();
  });

  it("shows the customer margin panel to a reader with the group", () => {
    seedPermissions({ [Resource.ProtectedMargins]: Operation.Read });
    render(
      <CustomerMargin
        rows={[customer()]}
        isLoading={false}
        isError={false}
        onRetry={() => undefined}
      />,
    );

    expect(screen.getByText("Kroger")).toBeInTheDocument();
    expect(screen.queryByText("Detention margin by customer needs access to margins.")).toBeNull();
  });
});
