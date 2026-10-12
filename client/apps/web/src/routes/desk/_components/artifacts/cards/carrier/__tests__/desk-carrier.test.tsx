import type {
  CarrierEligibility,
  CarrierIntel,
  CarrierSettlementRow,
  DeskCarrier as DeskCarrierRecord,
} from "@/lib/graphql/carrier-card";
import type { RecordShipment } from "@/lib/graphql/record-shipments";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskCarrier, DeskCarrierArtifact } from "../desk-carrier";

const fetchers = vi.hoisted(() => ({
  carrier: vi.fn(),
  eligibility: vi.fn(),
  intel: vi.fn(),
  settlements: vi.fn(),
  shipments: vi.fn(),
}));

vi.mock("@/lib/graphql/carrier-card", () => ({
  fetchDeskCarrier: fetchers.carrier,
  fetchCarrierEligibility: fetchers.eligibility,
  fetchCarrierIntel: fetchers.intel,
  fetchCarrierSettlements: fetchers.settlements,
}));
vi.mock("@/lib/graphql/record-shipments", () => ({ fetchRecordShipments: fetchers.shipments }));
// The shipment card is its own chunk; here it stands in as what it was given,
// with its way back.
vi.mock("../../shipment/desk-shipment", () => ({
  DeskShipment: ({
    recordId,
    onBack,
    backLabel,
  }: {
    recordId: string;
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>Shipment card {recordId}</p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  ),
  DeskShipmentArtifact: () => null,
}));

const DAY = 86_400;
const NOW = Math.floor(Date.now() / 1000);

/** A carrier as the DeskCarrierCard query returns it. */
function carrier(overrides: Partial<DeskCarrierRecord> = {}): DeskCarrierRecord {
  return {
    id: "car_1",
    status: "Active",
    code: "RRF",
    name: "Red River Freight",
    dbaName: "Red River Logistics",
    carrierType: "Common",
    dotNumber: "3141592",
    mcNumber: "MC-271828",
    scac: "RRFT",
    complianceStatus: "Qualified",
    safetyRating: "Satisfactory",
    qualifiedAt: NOW - 200 * DAY,
    disqualifiedReason: null,
    w9OnFile: true,
    paymentMethod: "ACHManual",
    paymentTermDays: 30,
    remitToName: null,
    addressLine1: "400 Industrial Blvd",
    addressLine2: null,
    city: "Shreveport",
    postalCode: "71101",
    phone: "(318) 555-0190",
    email: "dispatch@redriver.example",
    externalId: null,
    notes: null,
    intelRiskLevel: "Low",
    intelReviewRequired: false,
    createdAt: NOW - 900 * DAY,
    state: { abbreviation: "LA" },
    contacts: [
      {
        id: "cc_1",
        name: "Marcus Bell",
        title: "Dispatch",
        email: "marcus@redriver.example",
        phone: "(318) 555-0191",
        isPrimary: true,
        receivesRateConfirmations: true,
      },
    ],
    insurancePolicies: [
      {
        id: "pol_1",
        policyType: "CargoLiability",
        policyNumber: "CG-1182",
        providerName: "Great West Casualty",
        coverageAmount: "100000",
        effectiveDate: NOW - 200 * DAY,
        expirationDate: NOW + 200 * DAY,
        isVerified: true,
      },
    ],
    ...overrides,
  } as DeskCarrierRecord;
}

const clear: CarrierEligibility = {
  blockers: [],
  warnings: [],
  advisories: [],
  findings: [],
} as CarrierEligibility;

const snapshot = {
  id: "snap_1",
  provider: "CarrierOK",
  notFound: false,
  riskLevel: "Low",
  fetchedAt: NOW - 3_600,
  effectiveAsOf: NOW - 3_600,
  profile: {
    authority: {
      common: { status: "Active", pending: false, revocationPending: false, grantedAt: 1 },
      contract: null,
      broker: null,
      totalRevocations: 0,
      lastRevocationAt: null,
    },
    safety: {
      rating: "Satisfactory",
      ratingDate: null,
      outOfServiceOrder: false,
      outOfServiceAt: null,
    },
    basics: [
      { basic: "UnsafeDriving", measure: 2.2, percentile: 41, threshold: 65, alert: false },
      { basic: "HOSCompliance", measure: 6.4, percentile: 81, threshold: 65, alert: true },
    ],
  },
} as CarrierIntel;

const settlement = {
  id: "cs_1",
  settlementNumber: "CS-2610-0042",
  status: "Paid",
  periodStart: NOW - 14 * DAY,
  periodEnd: NOW - DAY,
  payDate: NOW,
  netPayableMinor: 1_245_050,
  shipmentCount: 4,
  currencyCode: "USD",
} as CarrierSettlementRow;

const shipment = {
  id: "shp_1",
  proNumber: "S26100090038417",
  status: "Completed",
  totalChargeAmount: "2450",
  moves: [],
} as unknown as RecordShipment;

function draw(props: Partial<Parameters<typeof DeskCarrier>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskCarrier recordId="car_1" order={["car_0", "car_1"]} {...handlers} {...props} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { ...view, ...handlers };
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

beforeEach(() => {
  seedPermissions({
    [Resource.Carrier]: Operation.Read,
    [Resource.ShipmentMove]: Operation.Read,
    [Resource.CarrierIntelligence]: Operation.Read,
    [Resource.Shipment]: Operation.Read,
    [Resource.CarrierSettlement]: Operation.Read,
  });
  fetchers.carrier.mockReset().mockResolvedValue(carrier());
  fetchers.eligibility.mockReset().mockResolvedValue(clear);
  fetchers.intel.mockReset().mockResolvedValue(snapshot);
  fetchers.settlements.mockReset().mockResolvedValue({
    rows: [settlement],
    total: 6,
    endCursor: "c1",
    hasNextPage: true,
  });
  fetchers.shipments.mockReset().mockResolvedValue({
    rows: [shipment],
    total: 23,
    endCursor: null,
    hasNextPage: false,
  });
});

afterEach(() => {
  clearPermissions();
});

describe("DeskCarrier", () => {
  it("holds the card's outline while the carrier is read, then draws whether it can be tendered", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the carrier…");
    expect(container.querySelector(".dk-rec-skel .dk-ca-basics")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Red River Freight", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(screen.getByText("Doing business as Red River Logistics")).toBeInTheDocument();
    expect(screen.getByText("DOT 3141592")).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Qualified");
    expect(document.querySelector(".dk-rec-amt b")).toHaveTextContent("Low");

    const tender = section("Can we tender to them");
    expect(await within(tender).findByText("Clear to tender")).toBeInTheDocument();
    expect(within(tender).getByText("Nothing stands in the way")).toBeInTheDocument();
    expect(fetchers.eligibility).toHaveBeenCalledWith("car_1", expect.anything());

    expect(within(section("Insurance")).getByText("Cargo liability")).toBeInTheDocument();

    const safety = section("Authority and safety");
    expect(await within(safety).findByText("Common authority")).toBeInTheDocument();
    expect(within(safety).getByRole("meter", { name: "Hours of Service" })).toHaveAttribute(
      "aria-valuenow",
      "81",
    );
    expect(within(safety).getByText("81 of 65").closest(".dk-rec-gauge")).toHaveClass("dk-bad");

    expect(await screen.findByText("23 shipments")).toBeInTheDocument();
    expect(fetchers.shipments).toHaveBeenCalledWith(
      { kind: "carrier", id: "car_1" },
      { first: 5, after: null },
      expect.anything(),
    );
    const paid = section("Settlements");
    expect(await within(paid).findByText("CS-2610-0042")).toBeInTheDocument();
    expect(within(paid).getByText("$12,450.50")).toBeInTheDocument();
    expect(within(paid).getByText("6 settlements")).toBeInTheDocument();

    const contacts = section("Contacts");
    expect(within(contacts).getByText("Marcus Bell")).toBeInTheDocument();
    expect(within(contacts).getByText("Primary · Gets rate confirmations")).toBeInTheDocument();
  });

  it("names what blocks a tender and what needs an override, worst first", async () => {
    fetchers.carrier.mockResolvedValue(
      carrier({
        complianceStatus: "Disqualified",
        disqualifiedReason: "Cargo insurance lapsed",
        intelRiskLevel: "High",
        insurancePolicies: [
          {
            id: "pol_1",
            policyType: "CargoLiability",
            policyNumber: "CG-1182",
            providerName: "Great West Casualty",
            coverageAmount: "100000",
            effectiveDate: NOW - 400 * DAY,
            expirationDate: NOW - 2 * DAY,
            isVerified: true,
          },
        ],
      }),
    );
    fetchers.eligibility.mockResolvedValue({
      blockers: ["Carrier compliance status is Disqualified"],
      warnings: ["Authority granted 4 months ago"],
      advisories: [],
      findings: [
        {
          code: "intel.authority_age",
          source: "Intelligence",
          severity: "Warning",
          message: "Authority granted 4 months ago",
          requiresOverride: true,
        },
        {
          code: "record.compliance_status",
          source: "Record",
          severity: "Blocker",
          message: "Carrier compliance status is Disqualified",
          requiresOverride: false,
        },
      ],
    });
    draw();

    expect(await screen.findByText("Cargo insurance lapsed")).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Disqualified");
    expect(screen.getByText("Cargo insurance lapsed")).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-amt b")).toHaveClass("dk-bad");

    const tender = section("Can we tender to them");
    expect(await within(tender).findByText("Blocked")).toBeInTheDocument();
    expect(Array.from(tender.querySelectorAll(".dk-rec-cb > b")).map((b) => b.textContent)).toEqual(
      ["Carrier compliance status is Disqualified", "Authority granted 4 months ago"],
    );
    expect(within(tender).getByText("From intelligence · Needs an override")).toBeInTheDocument();

    expect(within(section("Insurance")).getByText("Cargo liability expired")).toBeInTheDocument();
  });

  it("opens a load it hauled in the card's place, and reads the next settlements on request", async () => {
    draw();

    fireEvent.click(await screen.findByRole("button", { name: /S26100090038417/ }));
    expect(await screen.findByText("Shipment card shp_1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to Red River Freight" }));

    expect(await screen.findByText("CS-2610-0042")).toBeInTheDocument();
    fetchers.settlements.mockResolvedValue({
      rows: [{ ...settlement, id: "cs_2", settlementNumber: "CS-2609-0031" }],
      total: null,
      endCursor: null,
      hasNextPage: false,
    });
    fireEvent.click(screen.getByRole("button", { name: "Show more" }));
    expect(await screen.findByText("CS-2609-0031")).toBeInTheDocument();
    expect(fetchers.settlements).toHaveBeenLastCalledWith(
      "car_1",
      { first: 5, after: "c1" },
      expect.anything(),
    );
  });

  it("stands with nothing around it: never vetted, no insurance, contacts, loads or settlements", async () => {
    fetchers.carrier.mockResolvedValue(
      carrier({ dbaName: null, intelRiskLevel: null, insurancePolicies: [], contacts: [] }),
    );
    fetchers.intel.mockResolvedValue(null);
    fetchers.shipments.mockResolvedValue({
      rows: [],
      total: 0,
      endCursor: null,
      hasNextPage: false,
    });
    fetchers.settlements.mockResolvedValue({
      rows: [],
      total: 0,
      endCursor: null,
      hasNextPage: false,
    });
    draw();

    expect(
      await screen.findByRole("heading", { name: "Red River Freight", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("Not vetted")).toBeInTheDocument();
    expect(screen.getByText("No insurance policy is on file.")).toBeInTheDocument();
    expect(screen.getByText("No contact is on file.")).toBeInTheDocument();
    expect(
      await screen.findByText(
        "This carrier has not been vetted. Vet it on its page to read its authority and safety.",
      ),
    ).toBeInTheDocument();
    expect(await screen.findByText("This carrier has not hauled a load yet.")).toBeInTheDocument();
    expect(await screen.findByText("This carrier has not been settled yet.")).toBeInTheDocument();
  });

  it("says when the FMCSA has no record of the carrier", async () => {
    fetchers.intel.mockResolvedValue({ ...snapshot, notFound: true });
    draw();

    expect(
      await screen.findByText("The FMCSA has no record of this carrier's DOT number."),
    ).toBeInTheDocument();
  });

  it("says a section that could not be read, and draws the rest", async () => {
    fetchers.eligibility.mockRejectedValue(new Error("Carrier intelligence is unavailable"));
    fetchers.settlements.mockRejectedValue(new Error("boom"));
    draw();

    expect(await screen.findByText("Carrier intelligence is unavailable")).toBeInTheDocument();
    expect(await screen.findByText("boom")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Contacts", level: 3 })).toBeInTheDocument();
  });

  it("says the carrier could not be read, with the way back", async () => {
    fetchers.carrier.mockRejectedValue(new Error("Carrier not found"));
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", { name: "This carrier could not be read", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("Carrier not found")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
    expect(fetchers.eligibility).not.toHaveBeenCalled();
  });

  it("opens with only what the person may read, asking for nothing else", async () => {
    seedPermissions({ [Resource.Carrier]: Operation.Read });
    draw();

    expect(
      await screen.findByRole("heading", { name: "Red River Freight", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Can we tender to them" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Authority and safety" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Settlements" })).toBeNull();
    expect(screen.getByRole("heading", { name: "Insurance", level: 3 })).toBeInTheDocument();
    expect(fetchers.eligibility).not.toHaveBeenCalled();
    expect(fetchers.intel).not.toHaveBeenCalled();
    expect(fetchers.shipments).not.toHaveBeenCalled();
    expect(fetchers.settlements).not.toHaveBeenCalled();
  });

  it("leaves the outlines of sections the person may not read out of the skeleton", () => {
    seedPermissions({ [Resource.Carrier]: Operation.Read });
    fetchers.carrier.mockReturnValue(new Promise(() => {}));
    const { container } = draw();

    expect(container.querySelectorAll(".dk-rec-skel .dk-rec-sec")).toHaveLength(3);
  });

  it("draws the same card as an artifact, with nothing to step through", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskCarrierArtifact artifact={{} as never} recordId="car_1" fallback={null} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(
      await screen.findByRole("heading", { name: "Red River Freight", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Next carrier" })).toBeNull();
  });
});
