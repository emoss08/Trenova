import { TestRouter } from "@trenova/shared/testing/test-router";
import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { clearPermissions } from "@/test/permissions";
import { BASE_ONLY_UNAVAILABLE, seedPlan } from "@/test/plan";
import ShipmentBillingDetails from "../shipment-billing-details";

const { getUIPolicy } = vi.hoisted(() => ({ getUIPolicy: vi.fn() }));

vi.mock("@/services/shipment", () => ({ shipmentService: { getUIPolicy } }));
vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/graphql")>();
  return {
    ...actual,
    requestGraphQL: (request: Parameters<typeof actual.requestGraphQL>[0]) =>
      request.operationName === "ShipmentCustomerBillingRules"
        ? Promise.resolve({ shipmentCustomerBillingRules: [] })
        : actual.requestGraphQL(request),
  };
});
vi.mock("@/components/autocomplete-fields", () => ({
  CustomerAutocompleteField: () => null,
  FormulaTemplateAutocompleteField: ({ label }: { label?: string }) => (
    <div data-testid="rating-method-picker">{label}</div>
  ),
  OrderAutocompleteField: () => null,
}));
vi.mock("@/hooks/use-shipment-auto-rate", () => ({
  useShipmentAutoRate: () => ({ appliedRate: null, dismissAppliedRate: vi.fn() }),
}));
vi.mock("@/hooks/use-shipment-totals-preview", () => ({
  useShipmentTotalsPreview: () => ({
    isCalculating: false,
    error: null,
    fuelSurchargeChange: null,
    resolveFuelSurchargeChange: vi.fn(),
  }),
}));
vi.mock("../profitability/profitability-summary", () => ({ ProfitabilitySummary: () => null }));
vi.mock("../previous-rates-dialog", () => ({ PreviousRatesButton: () => null }));
vi.mock("../auto-rate-dialog", () => ({ AutoRateDialog: () => null }));
vi.mock("../additional-charges/fuel-surcharge-change-dialog", () => ({
  FuelSurchargeChangeDialog: () => null,
}));
vi.mock("../shipment-tariff-inputs", () => ({
  ShipmentTariffInputs: () => <div data-testid="tariff-inputs" />,
}));

afterEach(() => {
  cleanup();
  clearPermissions();
});

beforeEach(() => {
  vi.clearAllMocks();
  getUIPolicy.mockResolvedValue(undefined);
});

function Harness() {
  const form = useForm<Shipment>({
    defaultValues: { customerId: "", formulaTemplateId: "" } as unknown as Shipment,
  });
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } }),
  );
  return (
    <QueryClientProvider client={client}>
      <TestRouter>
        <FormProvider {...form}>
          <ShipmentBillingDetails />
        </FormProvider>
      </TestRouter>
    </QueryClientProvider>
  );
}

describe("ShipmentBillingDetails rating method", () => {
  // Every plan prices shipments with the standard templates, so the picker and its inputs
  // stand on a plan without the Rating add-on too.
  it("offers the rating method and its inputs on a base plan", () => {
    seedPlan({}, BASE_ONLY_UNAVAILABLE);

    render(<Harness />);

    expect(screen.getByTestId("rating-method-picker")).toHaveTextContent("Rating method");
    expect(screen.getByTestId("tariff-inputs")).toBeInTheDocument();
  });

  it("offers them with Rating on the plan", () => {
    seedPlan({}, []);

    render(<Harness />);

    expect(screen.getByTestId("rating-method-picker")).toBeInTheDocument();
    expect(screen.getByTestId("tariff-inputs")).toBeInTheDocument();
  });
});
