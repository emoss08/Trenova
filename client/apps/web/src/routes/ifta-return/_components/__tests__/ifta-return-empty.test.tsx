import type { IftaReturnView } from "@/lib/ifta-return";
import type {
  IftaPeriodFieldsFragment,
  IftaReturnLineFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Operation } from "@trenova/shared/types/permission";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { IftaReturnWorkspace } from "../ifta-return-workspace";

const { fetchIftaReturnForPeriod, fetchIftaPeriod, allowed } = vi.hoisted(() => ({
  fetchIftaReturnForPeriod: vi.fn(),
  fetchIftaPeriod: vi.fn(),
  allowed: { operations: new Set<number>() },
}));

vi.mock("@/lib/graphql/ifta-return", () => ({
  IFTA_RETURN_KEY: "ifta-return",
  IFTA_RETURN_LIST_KEY: "ifta-return-list",
  IFTA_PERIOD_KEY: "ifta-period",
  fetchIftaReturnForPeriod,
  fetchIftaPeriod,
  generateIftaReturn: vi.fn(),
  recomputeIftaReturn: vi.fn(),
  finalizeIftaReturn: vi.fn(),
  reopenIftaReturn: vi.fn(),
  markIftaReturnFiled: vi.fn(),
  amendIftaReturn: vi.fn(),
  deleteIftaReturn: vi.fn(),
  backfillJurisdictionMiles: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (_resource: string, operation: number) => ({
    allowed: allowed.operations.has(operation),
    isLoading: false,
  }),
}));

const APR_1_2026 = 1_775_001_600;
const JUL_1_2026 = 1_782_864_000;
const JUL_31_2026 = 1_785_456_000;

function period(): IftaPeriodFieldsFragment {
  return {
    year: 2026,
    quarter: 2,
    key: "2026Q2",
    label: "Q2 2026",
    start: APR_1_2026,
    end: JUL_1_2026,
    dueDate: JUL_31_2026,
  };
}

function line(over: Partial<IftaReturnLineFieldsFragment> = {}): IftaReturnLineFieldsFragment {
  return {
    id: over.id ?? "irl_1",
    returnId: "ir_1",
    jurisdictionId: "ij_tx",
    fuelType: "Diesel",
    isIftaMember: true,
    totalMiles: "1000.00",
    taxableMiles: "1000.00",
    routeMiles: "900.00",
    manualMiles: "100.00",
    loadedMiles: "800.00",
    emptyMiles: "200.00",
    taxPaidGallons: "150",
    taxPaidGallonsRaw: "150.250",
    purchaseCount: 3,
    taxableGallons: "160",
    netTaxableGallons: "10",
    ratePerGallon: "0.2000",
    surchargeRatePerGallon: null,
    rateMissing: false,
    taxDue: "2.00",
    surchargeDue: "0.00",
    lineTotal: "2.00",
    sortOrder: 1,
    jurisdiction: {
      id: "ij_tx",
      countryCode: "US",
      code: "TX",
      name: "Texas",
      isIftaMember: true,
      hasSurcharge: false,
    },
    ...over,
  };
}

function ret(over: Partial<IftaReturnView> = {}): IftaReturnView {
  return {
    id: "ir_1",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    year: 2026,
    quarter: 2,
    period: period(),
    amendmentNumber: 0,
    amendsReturnId: null,
    amendsReturn: null,
    status: "Draft",
    timezone: "America/New_York",
    periodStart: APR_1_2026,
    periodEnd: JUL_1_2026,
    totalMiles: "1000.00",
    totalTaxableMiles: "1000.00",
    totalGallons: "160.000",
    totalTaxPaidGallons: "150",
    netTaxableGallons: "10",
    taxDue: "2.00",
    surchargeDue: "0.00",
    netDue: "2.00",
    currencyCode: "USD",
    fleetMpgByFuelType: [
      { fuelType: "Diesel", mpg: "6.25", totalMiles: "1000.00", totalGallons: "160.000" },
    ],
    unattributedMiles: "0.00",
    unattributedMoveCount: 0,
    noTractorMiles: "0.00",
    noTractorMoveCount: 0,
    problems: [],
    computedAt: 1_783_000_000,
    finalizedAt: null,
    finalizedById: null,
    finalizedBy: null,
    filedAt: null,
    filedById: null,
    filedBy: null,
    filingReference: null,
    reopenedAt: null,
    reopenedById: null,
    reopenReason: null,
    lines: [line()],
    version: 1,
    createdAt: 1,
    updatedAt: 2,
    ...over,
  };
}

function renderWorkspace() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <IftaReturnWorkspace period={{ year: 2026, quarter: 2 }} onPeriodChange={vi.fn()} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  allowed.operations = new Set<number>([Operation.Read]);
  fetchIftaPeriod.mockResolvedValue(period());
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("IFTA return workspace", () => {
  // The quarter is read before anything can be drawn, so the wait is
  // announced once on the root and the shapes inside are decoration.
  it("announces itself as busy while the return is being read", () => {
    fetchIftaReturnForPeriod.mockReturnValue(new Promise(() => {}));
    renderWorkspace();

    const loading = screen.getByLabelText("Loading the return");
    expect(loading).toHaveAttribute("aria-busy", "true");
  });

  // Nothing generated is not an empty return: there are no figures at all,
  // and the way forward is to generate one.
  it("names the quarter and offers to generate when nothing has been generated", async () => {
    allowed.operations = new Set<number>([Operation.Read, Operation.Create]);
    fetchIftaReturnForPeriod.mockResolvedValue(null);
    renderWorkspace();

    expect(
      await screen.findByRole("heading", { name: "No return for Q2 2026 yet" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Generate the return/i })).toBeInTheDocument();
  });

  it("tells a reader without the permission who can generate it instead", async () => {
    fetchIftaReturnForPeriod.mockResolvedValue(null);
    renderWorkspace();

    expect(
      await screen.findByRole("heading", { name: "No return for Q2 2026 yet" }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Generate the return/i })).toBeNull();
    expect(screen.getByText(/permission to generate returns/i)).toBeInTheDocument();
  });

  it("reads a positive net as tax due", async () => {
    fetchIftaReturnForPeriod.mockResolvedValue(ret({ netDue: "2.00" }));
    renderWorkspace();

    expect(await screen.findByText("Net tax due")).toBeInTheDocument();
    expect(screen.getByText("$2.00")).toBeInTheDocument();
    expect(screen.queryByText("Net credit")).toBeNull();
  });

  // A negative net is money owed back to the fleet, and calling it "due"
  // would read as a bill.
  it("reads a negative net as a credit owed to the fleet", async () => {
    fetchIftaReturnForPeriod.mockResolvedValue(ret({ netDue: "-2.25", taxDue: "-2.25" }));
    renderWorkspace();

    expect(await screen.findByText("Net credit")).toBeInTheDocument();
    expect(screen.getByText("$2.25")).toBeInTheDocument();
    expect(screen.queryByText("Net tax due")).toBeNull();
  });

  // A member line with no published rate is taxed at nothing, which would be
  // wrong on a filed return, so the line says so and finalizing is refused.
  it("flags a line with no published rate and refuses to finalize", async () => {
    allowed.operations = new Set<number>([Operation.Read, Operation.Approve]);
    fetchIftaReturnForPeriod.mockResolvedValue(
      ret({ lines: [line({ rateMissing: true, ratePerGallon: null })] }),
    );
    renderWorkspace();

    expect(await screen.findByText("No rate")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Add rate/i })).toHaveAttribute(
      "href",
      "/fuel/configuration-files/ifta-tax-rates?panelType=create",
    );
    expect(screen.getByRole("button", { name: /Finalize/i })).toBeDisabled();
  });
});
