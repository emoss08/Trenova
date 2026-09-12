import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "@trenova/shared/lib/api";
import { RateQuoteService } from "./rate";

vi.mock("@trenova/shared/lib/api", () => ({
  api: {
    get: vi.fn(),
  },
}));

const toastError = vi.hoisted(() => vi.fn());

vi.mock("sonner", () => ({
  toast: { error: toastError },
}));

// Contract fixture mirroring ratequote.RateQuote and ratetypes.Trace in
// services/tms — nullable pointers and NullDecimals serialize as null, Decimals
// as quoted strings, and omitempty trace fields are absent.
function appliedQuote() {
  return {
    id: "rqt_01J8Z0000000000000000000",
    businessUnitId: "bu_01J8Z0000000000000000000",
    organizationId: "org_01J8Z0000000000000000000",
    shipmentId: "shp_01J8Z0000000000000000000",
    shipmentMoveId: null,
    partyType: "Customer",
    partyId: "cus_01J8Z0000000000000000000",
    purpose: "Rating",
    outcome: "FormulaFallback",
    status: "Applied",
    rateAgreementId: null,
    rateAgreementRuleId: null,
    agreementVersionNumber: null,
    formulaTemplateId: "fmt_01J8Z0000000000000000000",
    specificityScore: 0,
    currency: "USD",
    linehaulAmount: "1500.25",
    fuelAmount: "0",
    accessorialAmount: "0",
    totalAmount: "1500.25",
    costAmount: null,
    marginAmount: null,
    marginPercent: null,
    billingCurrency: "USD",
    billingAmount: "1500.25",
    fxRate: null,
    exchangeRateId: null,
    asOf: 1_757_000_000,
    ratedAt: 1_757_000_100,
    ratedById: null,
    engineVersion: "2026.09.1",
    contextHash: "a1b2c3",
    overrideReason: "",
    foregoneAmount: null,
    trace: {
      engineVersion: "2026.09.1",
      inputs: { ratingDate: 1_757_000_000, weekday: 3, partyType: "Customer", partyId: "cus_1" },
      candidateCount: 0,
      components: [
        {
          sequence: 0,
          kind: "Linehaul",
          label: "Linehaul",
          amount: "1500.25",
          runningTotal: "1500.25",
          source: "FormulaTemplate",
        },
      ],
      totals: { linehaul: "1500.25", fuel: "0", accessorial: "0", total: "1500.25" },
    },
    createdAt: 1_757_000_100,
  };
}

describe("RateQuoteService.getAppliedForShipment", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset();
    toastError.mockReset();
  });

  it("returns null without a parse error for a shipment that has never been rated", async () => {
    vi.mocked(api.get).mockResolvedValue(null);

    const quote = await new RateQuoteService().getAppliedForShipment("shp_1");

    expect(api.get).toHaveBeenCalledWith("/rate-quotes/shipment/shp_1/applied/");
    expect(quote).toBeNull();
    expect(toastError).not.toHaveBeenCalled();
  });

  it("parses the applied quote the server returns for a rated shipment", async () => {
    vi.mocked(api.get).mockResolvedValue(appliedQuote());

    const quote = await new RateQuoteService().getAppliedForShipment("shp_1");

    expect(quote).not.toBeNull();
    expect(quote?.outcome).toBe("FormulaFallback");
    expect(quote?.linehaulAmount).toBe(1500.25);
    expect(quote?.foregoneAmount).toBeNull();
    expect(quote?.trace?.components?.[0]?.amount).toBe(1500.25);
    expect(toastError).not.toHaveBeenCalled();
  });
});

describe("RateQuoteService.listForShipment", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset();
    toastError.mockReset();
  });

  it("parses every quote in the history so amounts arrive as numbers", async () => {
    const superseded = {
      ...appliedQuote(),
      id: "rqt_0",
      status: "Superseded",
      linehaulAmount: "900",
    };
    vi.mocked(api.get).mockResolvedValue([appliedQuote(), superseded]);

    const quotes = await new RateQuoteService().listForShipment("shp_1");

    expect(api.get).toHaveBeenCalledWith("/rate-quotes/shipment/shp_1/");
    expect(quotes.map((quote) => quote.linehaulAmount)).toEqual([1500.25, 900]);
    expect(quotes.map((quote) => quote.status)).toEqual(["Applied", "Superseded"]);
  });

  it("returns an empty history for a shipment that has never been rated", async () => {
    vi.mocked(api.get).mockResolvedValue([]);

    await expect(new RateQuoteService().listForShipment("shp_1")).resolves.toEqual([]);
    expect(toastError).not.toHaveBeenCalled();
  });
});
