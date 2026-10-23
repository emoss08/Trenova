import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { TariffInputDeclaration } from "@trenova/shared/types/tariff";
import { MemoryRouter } from "react-router";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import ShipmentBillingDetails from "../shipment-billing-details";

const { getBillingProfile, getUIPolicy, getDeclaredInputs } = vi.hoisted(() => ({
  getBillingProfile: vi.fn(),
  getUIPolicy: vi.fn(),
  getDeclaredInputs: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    shipmentService: { getUIPolicy },
    customerService: { getBillingProfile },
    formulaTemplateService: { getDeclaredInputs },
  },
}));
vi.mock("@/components/autocomplete-fields", () => ({
  CustomerAutocompleteField: () => null,
  FormulaTemplateAutocompleteField: () => null,
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
vi.mock("../why-this-rate", () => ({ WhyThisRate: () => null }));
vi.mock("../auto-rate-dialog", () => ({ AutoRateDialog: () => null }));
vi.mock("../billing-by-payer-card", () => ({ BillingByPayerCard: () => null }));
vi.mock("../additional-charges/fuel-surcharge-change-dialog", () => ({
  FuelSurchargeChangeDialog: () => null,
}));

afterEach(cleanup);

beforeEach(() => {
  vi.clearAllMocks();
  getUIPolicy.mockResolvedValue(undefined);
  getBillingProfile.mockResolvedValue({ creditStatus: "Good" });
});

// The template's billing version declares these inputs, as `tariff.check` derives them.
const DECLARATIONS: TariffInputDeclaration[] = [
  {
    name: "rate",
    doc: "Paid per loaded mile.",
    type: "Money/mi",
    descriptor: { kind: "quantity", dimension: "Money/Distance", unit: "$/mi" },
    required: true,
    range: null,
  },
  {
    name: "extraDropFee",
    doc: null,
    type: "Money",
    descriptor: { kind: "money" },
    required: false,
    default: { $q: "75", u: "$" },
    range: null,
  },
];

// A shipment as the GraphQL page fragment returns it after a tariff rated it: its input
// values in wire form, its freight lines, and the bill on its rating detail.
const RATED: Partial<Shipment> = {
  id: "shp_1",
  customerId: "cus_1",
  formulaTemplateId: "ft_dry_van",
  tariffInputs: { rate: { $q: "2.35", u: "$/mi" } },
  freightChargeAmount: 1908.2,
  otherChargeAmount: 75,
  totalChargeAmount: 1983.2,
  freightAllocations: [],
  additionalCharges: [],
  // The stored bill as GET /shipments/:id returns it, before the client reads it into wire form.
  ratingDetail: {
    formulaTemplateId: "ft_dry_van",
    formulaTemplateName: "Dry van TL",
    ratedAt: 1_757_000_000,
    versionNumber: 4,
    agreementName: "Acme 2026",
    ruleLabel: "Dallas to Chicago",
    bill: {
      currency: "USD",
      lines: [
        {
          kind: "Charge",
          label: "Linehaul",
          freight: true,
          amount: "1908.2",
          quantity: "812",
          quantityUnit: "mi",
          rate: "2.35",
          rateUnit: "$/mi",
        },
        {
          kind: "Charge",
          label: "Extra drops",
          code: "XDROP",
          freight: false,
          amount: "75",
          quantity: "1",
          rate: "75",
          rateUnit: "$",
        },
      ],
      freightTotal: "1908.2",
      accessorialTotal: "75",
      total: "1983.2",
    },
  } as unknown as Shipment["ratingDetail"],
};

function Harness({ values }: { values: Partial<Shipment> }) {
  const form = useForm<Shipment>({ defaultValues: values as Shipment });
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } }),
  );
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <FormProvider {...form}>
          <ShipmentBillingDetails />
        </FormProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe("ShipmentBillingDetails on a tariff", () => {
  it("shows the last rating's bill line by line with the contract that priced it", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    render(<Harness values={RATED} />);

    expect(await screen.findByText("Linehaul")).toBeInTheDocument();
    expect(screen.getByText("812 mi × $2.35/mi")).toBeInTheDocument();
    expect(screen.getByText("Extra drops")).toBeInTheDocument();
    expect(screen.getByText("XDROP")).toBeInTheDocument();
    expect(screen.getByText("Acme 2026 — Dallas to Chicago")).toBeInTheDocument();
    expect(screen.getByText("v4")).toBeInTheDocument();
  });

  it("shows what each choice kept, with every option's subtotal or refusal", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    // `ShipmentRatingDetail.choices` as GraphQL reads it: subtotals as Decimals, refusals as text.
    const ratingDetail = {
      ...(RATED.ratingDetail as object),
      choices: [
        {
          label: "Linehaul",
          kind: "best",
          currency: "USD",
          chosen: "Flat",
          options: [
            { label: "Per mile", status: "lost", subtotal: "1908.2", refusal: null },
            { label: "Flat", status: "chosen", subtotal: "1500", refusal: null },
            {
              label: "Volume",
              status: "refused",
              subtotal: null,
              refusal: "Volume rates start at 6,000 lb",
            },
          ],
        },
      ],
    } as unknown as Shipment["ratingDetail"];
    render(<Harness values={{ ...RATED, ratingDetail }} />);

    const choices = await screen.findByRole("region", { name: "Choices" });
    expect(within(choices).getByText("Chose Flat")).toBeInTheDocument();
    expect(within(choices).getByText("Lowest wins")).toBeInTheDocument();
    const rows = within(choices).getAllByRole("listitem");
    expect(rows.map((row) => row.getAttribute("data-status"))).toEqual([
      "lost",
      "chosen",
      "refused",
    ]);
    expect(within(rows[0]!).getByText("$1,908.20")).toBeInTheDocument();
    expect(within(rows[1]!).getByText("$1,500.00")).toBeInTheDocument();
    expect(within(rows[2]!).getByText("Volume rates start at 6,000 lb")).toBeInTheDocument();
  });

  it("shows no choices for a rating that made none", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    render(<Harness values={RATED} />);
    expect(await screen.findByText("Linehaul")).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Choices" })).not.toBeInTheDocument();
  });

  it("edits the shipment's own input values for its tariff in place of a base rate", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    render(<Harness values={RATED} />);

    expect(await screen.findByText("Paid per loaded mile")).toBeInTheDocument();
    expect(screen.getByDisplayValue("2.35")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Default: $75.00")).toBeInTheDocument();
    expect(screen.queryByText("Base rate")).not.toBeInTheDocument();
    expect(getDeclaredInputs).toHaveBeenCalledWith("ft_dry_van", null, expect.anything());
  });

  it("says why the tariff refused to rate, in its author's words, in place of a bill", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    render(
      <Harness
        values={{
          ...RATED,
          ratingDetail: {
            ...RATED.ratingDetail,
            bill: null,
            refusal: "This rate needs the shipment's weight",
          } as Shipment["ratingDetail"],
        }}
      />,
    );

    expect(await screen.findByText("The tariff refused to rate this shipment")).toBeInTheDocument();
    expect(screen.getByText("This rate needs the shipment's weight")).toBeInTheDocument();
    expect(screen.queryByText("Linehaul")).not.toBeInTheDocument();
  });

  it("says which charges were left to the tariff", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    render(
      <Harness
        values={{
          ...RATED,
          ratingDetail: {
            ...RATED.ratingDetail,
            skipped: [{ code: "FSC", source: "FuelProgram" }],
          } as Shipment["ratingDetail"],
        }}
      />,
    );

    expect(
      await screen.findByText("Not added again, since the tariff billed it: FSC"),
    ).toBeInTheDocument();
  });

  it("shows no bill for a shipment that has not been rated", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    render(<Harness values={{ ...RATED, ratingDetail: null }} />);

    await screen.findByText("Paid per loaded mile");
    expect(screen.queryByText("Linehaul")).not.toBeInTheDocument();
  });

  it("asks for a rating method before it can show any inputs", () => {
    render(<Harness values={{ ...RATED, formulaTemplateId: null, ratingDetail: null }} />);

    expect(
      screen.getByText("Choose a rating method to set the values its tariff reads."),
    ).toBeInTheDocument();
    expect(getDeclaredInputs).not.toHaveBeenCalled();
  });
});
