import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { Shipment } from "@trenova/shared/types/shipment";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { MarginCell } from "../../board/cells/margin-cell";
import { MoneyLine } from "../../board/expanded-row/money-line";
import { ProfitabilitySummary } from "../profitability-summary";

const mocks = vi.hoisted(() => ({ profitability: vi.fn() }));

vi.mock("@/lib/graphql/shipment", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/shipment")>()),
  getShipmentProfitabilityGraphQL: mocks.profitability,
}));

afterEach(() => {
  cleanup();
  clearPermissions();
  mocks.profitability.mockReset();
});

function renderWithin(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

// What the server answers a reader without the margins field group: the estimate is null.
const withheldShipment = {
  id: "shp_01",
  totalChargeAmount: "2400.00",
  freightChargeAmount: "2000.00",
  additionalCharges: [],
  moves: [],
  profitabilityEstimate: null,
} as unknown as Shipment;

function hiddenReason(): string | null | undefined {
  return screen
    .getByText("Hidden")
    .closest('[data-slot="locked-reference"]')
    ?.getAttribute("aria-description");
}

describe("a shipment's margin without the margins field group", () => {
  it("draws the board's margin cell as hidden, never as a dash", () => {
    seedPermissions({ [Resource.Shipment]: Operation.Read });
    renderWithin(<MarginCell shipment={withheldShipment} />);

    expect(hiddenReason()).toBe("You don't have access to margins");
    expect(screen.queryByText("—")).toBeNull();
  });

  it("draws the expanded row's cost and margin as hidden and keeps the revenue", () => {
    seedPermissions({ [Resource.Shipment]: Operation.Read });
    renderWithin(<MoneyLine shipment={withheldShipment} />);

    expect(screen.getByText("$2,400.00")).toBeInTheDocument();
    expect(screen.getAllByText("Hidden")).toHaveLength(2);
  });

  it("withholds the profitability estimate without asking the server", () => {
    seedPermissions({ [Resource.Shipment]: Operation.Read });
    renderWithin(<ProfitabilitySummary shipmentId="shp_01" />);

    expect(screen.getByText("Profitability estimate")).toBeInTheDocument();
    expect(hiddenReason()).toBe("You don't have access to margins");
    expect(mocks.profitability).not.toHaveBeenCalled();
  });

  it("asks for the estimate for a reader with the group", () => {
    seedPermissions({
      [Resource.Shipment]: Operation.Read,
      [Resource.ProtectedMargins]: Operation.Read,
    });
    mocks.profitability.mockResolvedValue(null);
    renderWithin(<ProfitabilitySummary shipmentId="shp_01" />);

    expect(mocks.profitability).toHaveBeenCalledTimes(1);
    expect(screen.queryByText("Hidden")).toBeNull();
  });
});
