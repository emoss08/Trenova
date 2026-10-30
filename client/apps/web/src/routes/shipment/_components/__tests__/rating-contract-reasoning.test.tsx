import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setLocale } from "@trenova/shared/i18n/runtime";
import { api } from "@trenova/shared/lib/api";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RatingContractReasoning } from "../rating-contract-reasoning";

vi.mock("@trenova/shared/lib/api", () => ({
  api: {
    get: vi.fn(),
  },
}));

const toastError = vi.hoisted(() => vi.fn());

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: toastError },
}));

// Contract fixture mirroring ratequote.RateQuote and ratetypes.Trace in services/tms: a rate set
// by hand over the contract (restated in the billing currency), the rule that won on a tie-break
// and one that lost with the reason, and a warning from the rating.
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
      candidateCount: 2,
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
        {
          agreementId: "rag_1",
          agreementName: "Acme 2026",
          ruleId: "rar_2",
          ruleLabel: "Texas to Oklahoma",
          laneKey: "TX>OK",
          won: false,
          rank: 2,
          specificityScore: 20,
          rulePriority: 0,
          agreementPriority: 0,
          effectiveFrom: 1_740_000_000,
          rejectDetail: "A more specific lane matched",
        },
      ],
      tieBreak: "lane specificity",
      warnings: ["Fuel index was a day old"],
      templateId: "ft_dry_van",
      templateName: "Dry van TL",
      tariffInputs: { rate: { $q: "2.2", u: "$/mi" } },
      totals: { linehaul: "1500", fuel: "0", accessorial: "0", total: "1500" },
    },
    createdAt: 1_757_000_100,
  };
}

function renderReasoning() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <RatingContractReasoning shipmentId="shp_1" />
    </QueryClientProvider>,
  );
}

describe("RatingContractReasoning", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset();
    toastError.mockReset();
  });

  afterEach(async () => {
    cleanup();
    await act(() => setLocale("en"));
  });

  it("names the contract and rule that won, and the tie-break that chose it", async () => {
    vi.mocked(api.get).mockResolvedValue(overriddenQuote());
    renderReasoning();

    const reasoning = await screen.findByRole("region", { name: "Why this contract" });
    expect(within(reasoning).getByText("Acme 2026")).toBeInTheDocument();
    expect(within(reasoning).getByText("Dallas to Tulsa")).toBeInTheDocument();
    expect(within(reasoning).getByText("Chosen on lane specificity.")).toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
  });

  it("states the override in the billing currency with a single space before the reason", async () => {
    vi.mocked(api.get).mockResolvedValue(overriddenQuote());
    renderReasoning();

    const message = await screen.findByText(/This rate was set by hand/);
    expect(message.textContent).toBe(
      "This rate was set by hand. The contract would have charged $1,500.00, a difference of $300.00. Reason given: Customer loyalty",
    );
  });

  it("lists every rule that was considered and why it lost", async () => {
    vi.mocked(api.get).mockResolvedValue(overriddenQuote());
    renderReasoning();

    const losers = await screen.findByRole("list", { name: "Considered but not applied" });
    const rows = within(losers).getAllByRole("listitem");
    expect(rows).toHaveLength(1);
    expect(within(rows[0]!).getByText("Texas to Oklahoma")).toBeInTheDocument();
    expect(within(rows[0]!).getByText("A more specific lane matched")).toBeInTheDocument();
  });

  it("shows the rating's warnings", async () => {
    vi.mocked(api.get).mockResolvedValue(overriddenQuote());
    renderReasoning();

    const warnings = await screen.findByRole("list", { name: "Warnings" });
    expect(warnings).toHaveTextContent("Fuel index was a day old");
  });

  it("translates the outcome badge", async () => {
    await act(() => setLocale("es"));
    vi.mocked(api.get).mockResolvedValue(overriddenQuote());
    renderReasoning();

    expect(await screen.findByText("Anulación manual")).toBeInTheDocument();
  });

  it("says the quote is no longer applied when the shipment has none", async () => {
    vi.mocked(api.get).mockResolvedValue(null);
    renderReasoning();

    expect(
      await screen.findByText("The quote this shipment was priced from is no longer applied."),
    ).toBeInTheDocument();
  });

  it("offers a retry when the quote cannot be read, and shows it once it can", async () => {
    vi.mocked(api.get).mockRejectedValueOnce(new Error("offline"));
    vi.mocked(api.get).mockResolvedValue(overriddenQuote());
    renderReasoning();

    expect(
      await screen.findByText("The contract's reasoning could not be loaded."),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(await screen.findByText("Dallas to Tulsa")).toBeInTheDocument();
  });
});
