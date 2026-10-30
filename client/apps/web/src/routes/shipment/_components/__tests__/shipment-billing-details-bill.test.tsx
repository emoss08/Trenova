import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { TariffInputDeclaration } from "@trenova/shared/types/tariff";
import { MemoryRouter } from "react-router";
import { FormProvider, useForm, useFormContext } from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import ShipmentBillingDetails from "../shipment-billing-details";

const { getBillingProfile, getUIPolicy, getDeclaredInputs, approveRating, getAppliedQuote } =
  vi.hoisted(() => ({
    getBillingProfile: vi.fn(),
    getUIPolicy: vi.fn(),
    getDeclaredInputs: vi.fn(),
    approveRating: vi.fn(),
    getAppliedQuote: vi.fn(),
  }));

vi.mock("@/lib/graphql/shipment", async (importOriginal) => ({
  ...(await importOriginal<Record<string, unknown>>()),
  approveShipmentRatingReviewGraphQL: approveRating,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    shipmentService: { getUIPolicy },
    customerService: { getBillingProfile },
    formulaTemplateService: { getDeclaredInputs },
    rateQuoteService: { getAppliedForShipment: getAppliedQuote },
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
    inputs: { rate: { $q: "2.35", u: "$/mi" } },
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

async function openBreakdown() {
  await userEvent.click(await screen.findByRole("button", { name: "Rating breakdown" }));
}

async function openSection(name: RegExp) {
  await userEvent.click(await screen.findByRole("button", { name }));
}

function VersionProbe() {
  const { watch } = useFormContext<Shipment>();
  return <output aria-label="version">{String(watch("version") ?? "")}</output>;
}

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
          <VersionProbe />
        </FormProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe("ShipmentBillingDetails on a tariff", () => {
  it("names the contract that priced the shipment and keeps its bill out of the card", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    render(<Harness values={RATED} />);

    expect(await screen.findByText("Acme 2026 — Dallas to Chicago")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Rating breakdown" })).toBeInTheDocument();
    expect(screen.queryByText("Linehaul")).not.toBeInTheDocument();
  });

  it("opens the last rating's bill line by line, with its tariff version", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    render(<Harness values={RATED} />);

    await openBreakdown();

    expect(await screen.findByText("Linehaul")).toBeInTheDocument();
    expect(screen.getByText("812 mi × $2.35/mi")).toBeInTheDocument();
    expect(screen.getByText("Extra drops")).toBeInTheDocument();
    expect(screen.getByText("XDROP")).toBeInTheDocument();
    expect(screen.getByText("v4")).toBeInTheDocument();
    const bill = screen.getByRole("region", { name: "Bill" });
    const freight = within(bill).getByRole("region", { name: "Freight" });
    expect(within(freight).getByText("Linehaul")).toBeInTheDocument();
    expect(within(freight).getAllByText("$1,908.20")).toHaveLength(2);
    const accessorials = within(bill).getByRole("region", { name: "Accessorials" });
    expect(within(accessorials).getByText("Extra drops")).toBeInTheDocument();
    expect(within(accessorials).getAllByText("$75.00")).toHaveLength(2);
    expect(within(bill).getByText("$1,983.20")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open tariff" })).toHaveAttribute(
      "href",
      formulaTemplateRoutes.edit("ft_dry_van"),
    );
  });

  it("lists the input values the rating ran with", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    render(<Harness values={RATED} />);

    await openBreakdown();
    await openSection(/^Inputs it ran with/);

    const inputs = await screen.findByLabelText("Inputs it ran with");
    expect(within(inputs).getByText("Rate")).toBeInTheDocument();
    expect(within(inputs).getByText("$2.35/mi")).toBeInTheDocument();
  });

  it("asks for the contract's reasoning only once its row opens", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    getAppliedQuote.mockResolvedValue(null);
    const ratingDetail = {
      ...(RATED.ratingDetail as object),
      rateQuoteId: "rqt_1",
    } as unknown as Shipment["ratingDetail"];
    render(<Harness values={{ ...RATED, ratingDetail }} />);

    await screen.findByText("Acme 2026 — Dallas to Chicago");
    expect(getAppliedQuote).not.toHaveBeenCalled();

    await openBreakdown();
    expect(await screen.findByText("Linehaul")).toBeInTheDocument();
    expect(getAppliedQuote).not.toHaveBeenCalled();

    await openSection(/^Why this contract/);

    await waitFor(() => expect(getAppliedQuote).toHaveBeenCalledWith("shp_1"));
  });

  it("asks for no contract reasoning for a rating no contract priced", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    render(<Harness values={RATED} />);

    await openBreakdown();

    expect(await screen.findByText("Linehaul")).toBeInTheDocument();
    expect(getAppliedQuote).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: /^Why this contract/ })).not.toBeInTheDocument();
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

    await openBreakdown();
    await openSection(/^Choices/);
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
    await openBreakdown();
    expect(await screen.findByText("Linehaul")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Choices/ })).not.toBeInTheDocument();
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

  it("says on the card why the tariff refused to rate, with no bill to open", async () => {
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
    expect(screen.queryByRole("button", { name: "Rating breakdown" })).not.toBeInTheDocument();
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

    await openBreakdown();
    expect(
      await screen.findByText("Not added again, since the tariff billed it: FSC"),
    ).toBeInTheDocument();
  });

  it("offers no breakdown for a shipment that has not been rated", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    render(<Harness values={{ ...RATED, ratingDetail: null }} />);

    await screen.findByText("Paid per loaded mile");
    expect(screen.queryByRole("button", { name: "Rating breakdown" })).not.toBeInTheDocument();
    expect(
      screen.getByText(
        "Automatically calculated based on the rating method, freight charges, and any additional accessorial charges.",
      ),
    ).toBeInTheDocument();
  });

  it("asks for a rating method before it can show any inputs", () => {
    render(<Harness values={{ ...RATED, formulaTemplateId: null, ratingDetail: null }} />);

    expect(
      screen.getByText("Choose a rating method to set the values its tariff reads."),
    ).toBeInTheDocument();
    expect(getDeclaredInputs).not.toHaveBeenCalled();
  });

  it("holds a rating the tariff asked to review and approves it in place", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    const held = {
      ...(RATED.ratingDetail as object),
      bill: {
        ...((RATED.ratingDetail as { bill: object }).bill as object),
        reviews: ["Holiday pickup"],
      },
      awaitingReview: true,
      reviewApproval: null,
    } as unknown as Shipment["ratingDetail"];
    approveRating.mockResolvedValue({
      ...RATED,
      version: 8,
      ratingDetail: {
        ...(held as object),
        awaitingReview: false,
        reviewApproval: {
          reviews: ["Holiday pickup"],
          approvedById: "usr_1",
          approvedAt: 1_800_000_000,
          approvedBy: { id: "usr_1", name: "Avery Lane" },
        },
      },
    });
    render(<Harness values={{ ...RATED, version: 7, ratingDetail: held }} />);

    const hold = await screen.findByTestId("rating-review-hold");
    expect(within(hold).getByRole("list", { name: "Reviews" })).toHaveTextContent("Holiday pickup");

    await userEvent.click(within(hold).getByRole("button", { name: "Approve rating" }));

    expect(await screen.findByTestId("rating-review-approved")).toHaveTextContent(
      /Approved for billing by Avery Lane/,
    );
    expect(approveRating).toHaveBeenCalledWith("shp_1");
    await waitFor(() => expect(screen.getByLabelText("version")).toHaveTextContent("8"));
    expect(screen.queryByTestId("rating-review-hold")).not.toBeInTheDocument();
  });

  it("shows no review hold for a rating the tariff asked nothing about", async () => {
    getDeclaredInputs.mockResolvedValue(DECLARATIONS);
    render(<Harness values={RATED} />);

    expect(await screen.findByText("Acme 2026 — Dallas to Chicago")).toBeInTheDocument();
    expect(screen.queryByTestId("rating-review-hold")).not.toBeInTheDocument();
    expect(screen.queryByTestId("rating-review-approved")).not.toBeInTheDocument();
  });
});
