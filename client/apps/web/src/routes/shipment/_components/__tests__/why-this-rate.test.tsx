import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { api } from "@trenova/shared/lib/api";
import { setLocale } from "@trenova/shared/i18n/runtime";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { WhyThisRate } from "../why-this-rate";

vi.mock("@trenova/shared/lib/api", () => ({
  api: {
    get: vi.fn(),
  },
}));

const toastError = vi.hoisted(() => vi.fn());

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: toastError },
}));

// Contract fixture mirroring ratequote.RateQuote and ratetypes.Trace in
// services/tms, whose trace now carries the winning tariff's bill in wire form:
// a charge with the author's label (here not in English, and shown as written),
// a minimum adjustment with no label, amounts in the contract currency while an
// override restates the quote in the billing currency.
function overriddenQuote() {
  return {
    id: "rqt_1",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    shipmentId: "shp_1",
    shipmentMoveId: null,
    partyType: "Customer",
    partyId: "cus_1",
    purpose: "Rating",
    outcome: "ManualOverride",
    status: "Applied",
    rateAgreementId: "rag_1",
    rateAgreementRuleId: "rar_1",
    agreementVersionNumber: 3,
    formulaTemplateId: null,
    specificityScore: 40,
    currency: "CAD",
    linehaulAmount: "1200",
    fuelAmount: "0",
    accessorialAmount: "0",
    totalAmount: "1200",
    costAmount: null,
    marginAmount: null,
    marginPercent: null,
    billingCurrency: "USD",
    billingAmount: "1200",
    fxRate: null,
    exchangeRateId: null,
    asOf: 1_757_000_000,
    ratedAt: 1_757_000_100,
    ratedById: null,
    engineVersion: "2026.09.1",
    contextHash: "a1b2c3",
    overrideReason: "Customer loyalty",
    foregoneAmount: "300",
    trace: {
      engineVersion: "2026.09.1",
      inputs: { ratingDate: 1_757_000_000, weekday: 3, partyType: "Customer", partyId: "cus_1" },
      candidateCount: 1,
      candidates: [
        {
          agreementId: "rag_1",
          agreementName: "Acme 2026",
          ruleId: "rar_1",
          ruleLabel: "Dallas to Tulsa",
          won: true,
          rank: 1,
          specificityScore: 40,
          rulePriority: 0,
          agreementPriority: 0,
          effectiveFrom: 1_740_000_000,
        },
      ],
      templateId: "ft_dry_van",
      templateName: "Dry van TL",
      tariffInputs: { rate: { $q: "2.2", u: "$/mi" }, minimumCharge: { $q: "1500", u: "$" } },
      bill: {
        currency: "CAD",
        lines: [
          {
            kind: "Charge",
            label: "Línea principal",
            freight: true,
            amount: "1100",
            quantity: "500",
            quantityUnit: "mi",
            rate: "2.2",
            rateUnit: "$/mi",
          },
          { kind: "Minimum", freight: true, amount: "400" },
        ],
        notes: [{ text: "Minimum applies under 1,500" }],
        freightTotal: "1500",
        accessorialTotal: "0",
        total: "1500",
      },
      totals: { linehaul: "1500", fuel: "0", accessorial: "0", total: "1500" },
    },
    createdAt: 1_757_000_100,
  };
}

async function openPopover() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <WhyThisRate shipmentId="shp_1" />
    </QueryClientProvider>,
  );

  await userEvent.click(
    await screen.findByRole("button", { name: /why this rate|por qué esta tarifa/i }),
  );
}

describe("WhyThisRate", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset();
    toastError.mockReset();
  });

  afterEach(async () => {
    cleanup();
    await act(() => setLocale("en"));
  });

  it("renders nothing for a shipment that has never been rated", async () => {
    vi.mocked(api.get).mockResolvedValue(null);

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { container } = render(
      <QueryClientProvider client={client}>
        <WhyThisRate shipmentId="shp_1" />
      </QueryClientProvider>,
    );

    await vi.waitFor(() => expect(api.get).toHaveBeenCalled());
    await vi.waitFor(() => expect(screen.queryByRole("button")).toBeNull());
    expect(container).toBeEmptyDOMElement();
    expect(toastError).not.toHaveBeenCalled();
  });

  it("shows the tariff's bill in the contract currency, with the charge label as written", async () => {
    vi.mocked(api.get).mockResolvedValue(overriddenQuote());

    await openPopover();

    expect(await screen.findByText("Línea principal")).toBeTruthy();
    expect(screen.getByText("500 mi × CA$2.20/mi")).toBeTruthy();
    expect(screen.getByText("Minimum charge adjustment")).toBeTruthy();
    expect(screen.getByText("Minimum applies under 1,500")).toBeTruthy();
    expect(screen.getAllByText("CA$1,500.00").length).toBeGreaterThan(0);
  });

  it("lists the input values the tariff read, in the contract currency", async () => {
    vi.mocked(api.get).mockResolvedValue(overriddenQuote());

    await openPopover();

    expect(await screen.findByText("Terms it read")).toBeTruthy();
    expect(screen.getByText("Rate")).toBeTruthy();
    expect(screen.getByText("CA$2.20/mi")).toBeTruthy();
    expect(screen.getByText("Minimum charge")).toBeTruthy();
  });

  it("keeps the tariff's own labels untranslated in another locale", async () => {
    await act(() => setLocale("es"));
    vi.mocked(api.get).mockResolvedValue(overriddenQuote());

    await openPopover();

    expect(await screen.findByText("Línea principal")).toBeTruthy();
  });

  it("states the override in the billing currency with a single space before the reason", async () => {
    vi.mocked(api.get).mockResolvedValue(overriddenQuote());

    await openPopover();

    const message = await screen.findByText(/This rate was set by hand/);
    expect(message.textContent).toBe(
      "This rate was set by hand. The contract would have charged $1,500.00, a difference of $300.00. Reason given: Customer loyalty",
    );
  });

  it("translates the outcome badge", async () => {
    await act(() => setLocale("es"));
    vi.mocked(api.get).mockResolvedValue(overriddenQuote());

    await openPopover();

    expect(await screen.findByText("Anulación manual")).toBeTruthy();
  });
});
