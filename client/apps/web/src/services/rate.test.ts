import { beforeEach, describe, expect, it, vi } from "vitest";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { RateAgreement, RateMatrixCell } from "@trenova/shared/types/rate";
import {
  RateAgreementService,
  RateMatrixService,
  RateQuoteService,
  toRateAgreementInput,
  toRateMatrixCellInput,
} from "./rate";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const toastError = vi.hoisted(() => vi.fn());

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: toastError },
}));

type GraphQLCall = { operationName: string; variables: Record<string, unknown> };

function calls(): GraphQLCall[] {
  return vi.mocked(requestGraphQL).mock.calls.map(([request]) => request as GraphQLCall);
}

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
      bill: {
        currency: "USD",
        lines: [
          {
            kind: "Charge",
            label: "Linehaul",
            freight: true,
            amount: "1500.25",
            quantity: "625.1",
            quantityUnit: "mi",
            rate: "2.4",
            rateUnit: "$/mi",
          },
        ],
        freightTotal: "1500.25",
        accessorialTotal: "0",
        total: "1500.25",
      },
      totals: { linehaul: "1500.25", fuel: "0", accessorial: "0", total: "1500.25" },
    },
    createdAt: 1_757_000_100,
  };
}

describe("RateQuoteService.getAppliedForShipment", () => {
  beforeEach(() => {
    vi.mocked(requestGraphQL).mockReset();
    toastError.mockReset();
  });

  it("returns null without a parse error for a shipment that has never been rated", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ appliedShipmentRateQuote: null });

    const quote = await new RateQuoteService().getAppliedForShipment("shp_1");

    expect(calls()).toEqual([
      expect.objectContaining({
        operationName: "AppliedShipmentRateQuote",
        variables: { shipmentId: "shp_1" },
      }),
    ]);
    expect(quote).toBeNull();
    expect(toastError).not.toHaveBeenCalled();
  });

  it("parses the applied quote the server returns for a rated shipment", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ appliedShipmentRateQuote: appliedQuote() });

    const quote = await new RateQuoteService().getAppliedForShipment("shp_1");

    expect(quote).not.toBeNull();
    expect(quote?.outcome).toBe("FormulaFallback");
    expect(quote?.linehaulAmount).toBe(1500.25);
    expect(quote?.foregoneAmount).toBeNull();
    expect(quote?.trace?.bill?.lines[0]?.amount).toEqual({ $q: "1500.25", u: "$" });
    expect(quote?.trace?.bill?.notes).toEqual([]);
    expect(toastError).not.toHaveBeenCalled();
  });
});

describe("RateQuoteService.shop", () => {
  beforeEach(() => {
    vi.mocked(requestGraphQL).mockReset();
  });

  it("shops least cost from the routing guide when no strategy or shortlist is given", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      shopShipmentCarriers: { strategy: "LeastCost", options: [], warnings: [] },
    });

    const result = await new RateQuoteService().shop("shp_1");

    expect(calls()[0]).toEqual(
      expect.objectContaining({
        operationName: "ShopShipmentCarriers",
        variables: {
          shipmentId: "shp_1",
          input: { strategy: "LeastCost", carrierIds: null, asOf: null, limit: null },
        },
      }),
    );
    expect(result.options).toEqual([]);
  });
});

function agreement(overrides: Partial<RateAgreement> = {}): RateAgreement {
  return {
    id: "rag_1",
    version: 4,
    partyType: "Customer",
    customerId: "cus_1",
    carrierId: null,
    code: "ACME-2026",
    name: "Acme 2026",
    description: "",
    agreementType: "Contract",
    status: "Draft",
    contractRef: "",
    documentId: null,
    priority: 0,
    effectiveFrom: 1_767_225_600,
    effectiveTo: null,
    autoRenew: false,
    renewalNoticeDays: 30,
    currency: "USD",
    inputs: {},
    billToCustomerId: null,
    marginFloorPercent: 12.5,
    maxPayPercentOfSell: null,
    reviewComment: "",
    currentVersionNumber: 1,
    rules: [],
    accessorials: [],
    fuelBinding: null,
    versions: [],
    ...overrides,
  } as RateAgreement;
}

describe("toRateAgreementInput", () => {
  it("sends decimals as strings, unset ones as null, and the version it read", () => {
    const input = toRateAgreementInput(agreement());

    expect(input.marginFloorPercent).toBe("12.5");
    expect(input.maxPayPercentOfSell).toBeNull();
    expect(input.version).toBe(4);
    expect(input.fuelBinding).toBeNull();
    expect(input.rules).toEqual([]);
  });

  it("keeps a held rule's id so the save supersedes it, and sends empty id lists as null", () => {
    const input = toRateAgreementInput(
      agreement({
        rules: [
          {
            id: "ragr_1",
            label: "ATL to CHI",
            status: "Active",
            originScopeType: "City",
            originScopeValue: "",
            originCity: "Atlanta",
            destinationScopeType: "City",
            destinationScopeValue: "",
            destinationCity: "Chicago",
            direction: "Directional",
            serviceTypeIds: [],
            commodityIds: ["com_1"],
            minWeight: 1000,
            maxWeight: null,
            daysOfWeek: 0,
            hazmatOnly: false,
            tempControlOnly: false,
            formulaTemplateId: "ft_1",
            inputs: {},
            priority: 0,
            effectiveFrom: 1_767_225_600,
          } as unknown as RateAgreement["rules"][number],
        ],
      }),
    );

    expect(input.rules?.[0]).toEqual(
      expect.objectContaining({
        id: "ragr_1",
        serviceTypeIds: null,
        commodityIds: ["com_1"],
        minWeight: "1000",
        maxWeight: null,
        formulaTemplateId: "ft_1",
      }),
    );
  });
});

describe("RateAgreementService.review", () => {
  beforeEach(() => {
    vi.mocked(requestGraphQL).mockReset();
    vi.mocked(requestGraphQL).mockResolvedValue({});
  });

  it("sends the mutation named for the step with the comment", async () => {
    await new RateAgreementService().review("rag_1", "suspend", "Customer on credit hold");

    expect(calls()).toEqual([
      expect.objectContaining({
        operationName: "SuspendRateAgreement",
        variables: { id: "rag_1", comment: "Customer on credit hold" },
      }),
    ]);
  });
});

describe("toRateMatrixCellInput", () => {
  it("sends only the keys and bands a cell sets, so a large grid stays small", () => {
    const cell = {
      d0Key: "30301",
      d1Key: "60601",
      d2Key: "",
      d3Key: "",
      d0KeyTo: "",
      d1KeyTo: "",
      d2KeyTo: "",
      d3KeyTo: "",
      d0Min: null,
      d0Max: null,
      d1Min: 0,
      d1Max: 1000,
      value: 2.35,
      minCharge: null,
      deficitEligible: true,
    } as RateMatrixCell;

    expect(toRateMatrixCellInput(cell)).toEqual({
      value: "2.35",
      d0Key: "30301",
      d1Key: "60601",
      d1Min: "0",
      d1Max: "1000",
    });
  });

  it("states a cell that is not deficit eligible", () => {
    const cell = { value: 1, deficitEligible: false } as RateMatrixCell;

    expect(toRateMatrixCellInput(cell)).toEqual({ value: "1", deficitEligible: false });
  });
});

describe("RateMatrixService.listCells", () => {
  beforeEach(() => {
    vi.mocked(requestGraphQL).mockReset();
  });

  it("walks a published version's pages until one comes back short", async () => {
    const page = (count: number) =>
      Array.from({ length: count }, (_, index) => ({ id: `rmc_${index}`, value: "1" }));
    vi.mocked(requestGraphQL)
      .mockResolvedValueOnce({ rateMatrixVersion: { id: "rmv_1", cells: page(100) } })
      .mockResolvedValueOnce({ rateMatrixVersion: { id: "rmv_1", cells: page(3) } });

    const cells = await new RateMatrixService().listCells("rm_1", 2);

    expect(cells).toHaveLength(103);
    expect(calls().map((call) => [call.operationName, call.variables])).toEqual([
      ["RateMatrixVersionCells", { id: "rm_1", versionNumber: 2, offset: 0, limit: 100 }],
      ["RateMatrixVersionCells", { id: "rm_1", versionNumber: 2, offset: 100, limit: 100 }],
    ]);
  });

  it("reads the draft when no version is named", async () => {
    vi.mocked(requestGraphQL).mockResolvedValueOnce({ rateMatrix: { id: "rm_1", cells: [] } });

    await expect(new RateMatrixService().listCells("rm_1")).resolves.toEqual([]);
    expect(calls()[0]?.operationName).toBe("RateMatrixDraftCells");
  });
});
