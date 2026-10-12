import type { ARCustomerProfile } from "@/lib/graphql/accounts-receivable";
import type {
  DeskCustomer as DeskCustomerRecord,
  DeskCustomerRateAgreement,
} from "@/lib/graphql/customer-card";
import type { RecordShipment } from "@/lib/graphql/record-shipments";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskCustomer } from "../desk-customer";

const fetchers = vi.hoisted(() => ({
  customer: vi.fn(),
  shipments: vi.fn(),
  agreements: vi.fn(),
  receivables: vi.fn(),
}));

vi.mock("@/lib/graphql/customer-card", () => ({
  fetchDeskCustomer: fetchers.customer,
  fetchCustomerRateAgreements: fetchers.agreements,
}));
vi.mock("@/lib/graphql/record-shipments", () => ({
  fetchRecordShipments: fetchers.shipments,
}));
// The shipment card is its own chunk; here it stands in as what it was given,
// with its way back.
vi.mock("../../shipment/desk-shipment", () => ({
  DeskShipment: ({
    recordId,
    order,
    onBack,
    backLabel,
  }: {
    recordId: string;
    order: readonly string[];
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>Shipment card {recordId}</p>
      <p>Stepping through {order.join(", ")}</p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  ),
  DeskShipmentArtifact: () => null,
}));
vi.mock("../../rate-agreement/desk-rate-agreement", () => ({
  DeskRateAgreement: ({ recordId, order }: { recordId: string; order: readonly string[] }) => (
    <div>
      <p>Rate agreement card {recordId}</p>
      <p>Stepping through {order.join(", ")}</p>
    </div>
  ),
  DeskRateAgreementArtifact: () => null,
}));
vi.mock("@/lib/graphql/accounts-receivable", async (original) => ({
  ...(await original<typeof import("@/lib/graphql/accounts-receivable")>()),
  fetchArCustomerProfile: fetchers.receivables,
}));

const DAY = 86_400;
const NOW = Math.floor(Date.now() / 1000);

/** A customer as the DeskCustomerCard query returns it. */
function customer(overrides: Partial<DeskCustomerRecord> = {}): DeskCustomerRecord {
  return {
    id: "cus_1",
    status: "Active",
    code: "ACME",
    name: "Acme Foods",
    addressLine1: "1200 Market St",
    addressLine2: null,
    city: "Atlanta",
    postalCode: "30303",
    dotNumber: null,
    mcNumber: "123456",
    createdAt: 1_710_000_000,
    updatedAt: 1_790_000_000,
    state: { abbreviation: "GA" },
    billingProfile: {
      paymentTerm: "Net30",
      billingCycle: "Weekly",
      invoiceDelivery: "PerShipment",
      emailInvoiceEnabled: true,
      ediInvoiceEnabled: false,
      autoBill: true,
      creditLimit: "50000",
      creditBalance: "10000",
      creditStatus: "Active",
      enforceCreditLimit: true,
      creditHoldReason: "",
      fuelSurchargeMode: "Program",
      billingCurrency: "USD",
      requirePONumber: true,
      requireBOLNumber: false,
      requireDeliveryNumber: false,
      taxExempt: false,
      applyLateCharges: false,
      lateChargeRate: null,
      gracePeriodDays: 0,
      billingNotes: "Send invoices to ap@acme.example",
      documentTypes: [{ id: "dt_pod", name: "Proof of delivery" }],
    },
    ...overrides,
  } as DeskCustomerRecord;
}

/** The customer's receivables as arCustomerProfile returns them, in cents. */
function receivables(): ARCustomerProfile {
  return {
    dsoDays: 38,
    creditUtilization: 0.8,
    delinquencyScore: 12,
    snapshot: {
      customerId: "cus_1",
      customerName: "Acme Foods",
      totalOpenMinor: 40_000_00,
      overdueMinor: 6_500_00,
      unappliedCashMinor: 1_200_00,
      creditLimitMinor: 50_000_00,
      hasCreditLimit: true,
      openInvoiceCount: 7,
      oldestOpenInvoiceDate: NOW - 80 * DAY,
      oldestDaysPastDue: 50,
      lastPaymentDate: NOW - 5 * DAY,
      lastPaymentMinor: 3_000_00,
      avgDaysToPay: 52,
      billedTrailing91Minor: 120_000_00,
      buckets: {
        currentMinor: 33_500_00,
        days1To30Minor: 4_000_00,
        days31To60Minor: 2_500_00,
        days61To90Minor: 0,
        daysOver90Minor: 0,
        totalOpenMinor: 40_000_00,
      },
      monthlyCollections: [{ monthStart: NOW - 3 * DAY, amountMinor: 9_000_00 }],
    },
  } as ARCustomerProfile;
}

const shipment = {
  id: "shp_1",
  proNumber: "S26100090038417",
  status: "InTransit",
  totalChargeAmount: "2450",
  moves: [
    {
      sequence: 0,
      stops: [
        {
          sequence: 0,
          type: "Pickup",
          scheduledWindowStart: NOW,
          location: { city: "Atlanta", state: { abbreviation: "GA" } },
        },
        {
          sequence: 1,
          type: "Delivery",
          scheduledWindowStart: NOW + DAY,
          location: { city: "Dallas", state: { abbreviation: "TX" } },
        },
      ],
    },
  ],
} as RecordShipment;

/** A first page of shipments as fetchRecordShipments returns it, with nothing after it. */
function shipmentPage(rows: RecordShipment[], total: number) {
  return { rows, total, endCursor: null, hasNextPage: false };
}

const agreement = {
  id: "ra_1",
  code: "ACME-2026",
  name: "Acme dry van contract",
  status: "Active",
  agreementType: "Contract",
  effectiveFrom: NOW - 100 * DAY,
  effectiveTo: NOW + 12 * DAY,
  autoRenew: false,
  renewalNoticeDays: 30,
} as DeskCustomerRateAgreement;

const EVERYTHING = {
  [Resource.Customer]: Operation.Read,
  [Resource.AccountsReceivable]: Operation.Read,
  [Resource.Shipment]: Operation.Read,
  [Resource.RateAgreement]: Operation.Read,
};

function draw(props: Partial<Parameters<typeof DeskCustomer>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskCustomer
          recordId="cus_1"
          order={["cus_0", "cus_1", "cus_2"]}
          {...handlers}
          {...props}
        />
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
  seedPermissions(EVERYTHING);
  for (const fn of Object.values(fetchers)) fn.mockReset();
  fetchers.customer.mockResolvedValue(customer());
  fetchers.receivables.mockResolvedValue(receivables());
  fetchers.shipments.mockResolvedValue(shipmentPage([shipment], 42));
  fetchers.agreements.mockResolvedValue({ rows: [agreement], total: 1 });
});

afterEach(() => {
  clearPermissions();
});

describe("DeskCustomer", () => {
  it("holds the card's outline while the customer is read, then draws who they are", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the customer…");
    expect(container.querySelector(".dk-rec-skel .dk-cu-age")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Acme Foods", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(screen.getByText("ACME")).toBeInTheDocument();
    expect(screen.getByText("Atlanta, GA")).toBeInTheDocument();
    expect(screen.getByText("MC 123456")).toBeInTheDocument();
    expect(screen.getByText("1200 Market St, Atlanta, GA 30303")).toBeInTheDocument();
    expect(screen.getByText("In good standing")).toBeInTheDocument();
    expect(screen.getByText("2 of 3")).toBeInTheDocument();
  });

  it("leads with what is open and how far past due it has aged", async () => {
    const { container } = draw();

    expect(await screen.findByText("$40,000.00")).toBeInTheDocument();
    expect(screen.getByText("open on 7 invoices")).toBeInTheDocument();
    expect(screen.getByText("$6,500.00 past due")).toBeInTheDocument();
    expect(screen.getByText("Oldest 50 days late")).toBeInTheDocument();

    const owed = section("Receivables");
    expect(within(owed).getByRole("img")).toHaveAccessibleName(
      "Current: $33,500.00, 1–30: $4,000.00, 31–60: $2,500.00",
    );
    expect(container.querySelectorAll(".dk-cu-age > i")).toHaveLength(3);
    expect(within(owed).getByText("52 days").closest("div")).toHaveClass("dk-bad");
    expect(within(owed).getByText("38 days")).toBeInTheDocument();
    expect(
      within(owed).getByText("$1,200.00 received and not yet applied to an invoice."),
    ).toBeInTheDocument();
    expect(within(owed).getByRole("link", { name: /Statement/ })).toHaveAttribute(
      "href",
      "/accounting/ar/customer-statement/cus_1",
    );
    expect(within(owed).getByRole("list", { name: "Collected by month" }).children).toHaveLength(6);
  });

  it("measures credit by what is open now, and reads the billing terms and what invoices need", async () => {
    draw();
    await screen.findByText("$40,000.00");

    const bills = section("Billing");
    expect(within(bills).getByRole("meter", { name: "Credit used" })).toHaveAttribute(
      "aria-valuenow",
      "80",
    );
    expect(bills.querySelector(".dk-cu-credit")).toHaveClass("dk-warn");
    expect(within(bills).getByText("Net 30")).toBeInTheDocument();
    expect(within(bills).getByText("Weekly")).toBeInTheDocument();
    expect(within(bills).getByText("Per shipment · Email")).toBeInTheDocument();
    expect(within(bills).getByText("Fuel surcharge program")).toBeInTheDocument();
    expect(within(bills).getByText("PO number")).toBeInTheDocument();
    expect(within(bills).getByText("Proof of delivery")).toBeInTheDocument();
    expect(within(bills).getByText("Credit limit enforced")).toBeInTheDocument();
    expect(within(bills).getByText("Send invoices to ap@acme.example")).toBeInTheDocument();
  });

  it("lists the latest shipments, each with its lane, and how many there are in all", async () => {
    draw();

    const row = await screen.findByRole("button", { name: /S26100090038417/ });
    expect(within(row).getByText(/Atlanta, GA/)).toHaveTextContent("Atlanta, GA → Dallas, TX");
    expect(within(section("Recent shipments")).getByText("42 shipments")).toBeInTheDocument();
    expect(fetchers.shipments).toHaveBeenCalledWith(
      { kind: "customer", id: "cus_1" },
      { first: 5, after: null },
      expect.anything(),
    );
  });

  // A shipment the card lists opens as a table's row does: as the shipment's
  // own card, in the customer card's place, stepping through the shipments
  // listed, and the way back returns to the customer.
  it("opens a listed shipment in the card's place, and goes back to the customer", async () => {
    fetchers.shipments.mockResolvedValue(
      shipmentPage([shipment, { ...shipment, id: "shp_2", proNumber: "S2" }], 2),
    );
    draw();

    fireEvent.click(await screen.findByRole("button", { name: /S26100090038417/ }));

    expect(await screen.findByText("Shipment card shp_1")).toBeInTheDocument();
    expect(screen.getByText("Stepping through shp_1, shp_2")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Acme Foods", level: 2 })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Back to Acme Foods" }));

    expect(
      await screen.findByRole("heading", { name: "Acme Foods", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Shipment card shp_1")).toBeNull();
  });

  // An agreement the card lists opens as its own card in the customer card's
  // place, the agreements in force first.
  it("opens a listed rate agreement in the card's place, the agreements in force first", async () => {
    fetchers.agreements.mockResolvedValue({
      rows: [{ ...agreement, id: "ra_0", name: "Old draft", status: "Draft" }, agreement],
      total: 2,
    });
    draw();

    const deal = await screen.findByRole("button", { name: /Acme dry van contract/ });
    expect(within(deal).getByText("Ends in 12 days")).toHaveClass("dk-warn");
    const rows = within(section("Rate agreements")).getAllByRole("button");
    expect(rows.map((row) => row.textContent)).toEqual([
      expect.stringContaining("Acme dry van contract"),
      expect.stringContaining("Old draft"),
    ]);

    fireEvent.click(deal);
    expect(await screen.findByText("Rate agreement card ra_1")).toBeInTheDocument();
    expect(screen.getByText("Stepping through ra_1, ra_0")).toBeInTheDocument();
  });

  // A card opens for anyone who may read its record. What it draws from other
  // records is left out for someone who may not read them, and never asked for.
  it("opens with only what the person may read, asking for nothing else", async () => {
    seedPermissions({ [Resource.Customer]: Operation.Read });
    const { container } = draw();

    expect(
      await screen.findByRole("heading", { name: "Acme Foods", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Receivables" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Recent shipments" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Rate agreements" })).toBeNull();
    expect(container.querySelector(".dk-rec-amt")).toBeNull();
    expect(fetchers.receivables).not.toHaveBeenCalled();
    expect(fetchers.shipments).not.toHaveBeenCalled();
    expect(fetchers.agreements).not.toHaveBeenCalled();

    const bills = section("Billing");
    expect(within(bills).getByRole("meter", { name: "Credit used" })).toHaveAttribute(
      "aria-valuenow",
      "20",
    );
    expect(screen.getByRole("link", { name: /Open customer/ })).toBeInTheDocument();
  });

  it("leaves a section's outline out of the skeleton when the person may not read it", () => {
    seedPermissions({ [Resource.Customer]: Operation.Read });
    fetchers.customer.mockReturnValue(new Promise(() => {}));
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the customer…");
    expect(container.querySelector(".dk-rec-skel .dk-cu-age")).toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-cu-terms")).not.toBeNull();
  });

  it("says a section could not be read in its own place, and the rest of the card stands", async () => {
    fetchers.shipments.mockRejectedValue(new Error("Shipments are unavailable"));
    draw();

    const failed = await screen.findByText("Shipments are unavailable");
    expect(section("Recent shipments")).toContainElement(failed);
    expect(await screen.findByText("Ends in 12 days")).toBeInTheDocument();
    expect(screen.getByText("$40,000.00")).toBeInTheDocument();
  });

  it("puts a hold's reason under the head", async () => {
    fetchers.customer.mockResolvedValue(
      customer({
        billingProfile: {
          ...customer().billingProfile!,
          creditStatus: "Hold",
          creditHoldReason: "Two invoices past 90 days",
        },
      }),
    );
    draw();

    expect(await screen.findByText("On credit hold")).toBeInTheDocument();
    expect(screen.getByText("Two invoices past 90 days")).toBeInTheDocument();
  });

  it("says when the customer cannot be read, with the way back", async () => {
    fetchers.customer.mockRejectedValue(new Error("Customer not found"));
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", { name: "This customer could not be read" }),
    ).toBeInTheDocument();
    screen.getByRole("button", { name: "Back to the table" }).click();
    expect(onBack).toHaveBeenCalled();
  });
});
