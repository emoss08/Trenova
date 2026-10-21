import type { DeskOrder as DeskOrderRecord } from "@/lib/graphql/order-card";
import type { ActivityEntry } from "@/lib/graphql/record-activity";
import type { RecordShipment } from "@/lib/graphql/record-shipments";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskOrder, DeskOrderArtifact } from "../desk-order";

const fetchers = vi.hoisted(() => ({
  order: vi.fn(),
  activity: vi.fn(),
  shipments: vi.fn(),
}));

vi.mock("@/lib/graphql/order-card", () => ({ fetchDeskOrder: fetchers.order }));
vi.mock("@/lib/graphql/record-activity", () => ({ fetchRecordActivity: fetchers.activity }));
vi.mock("@/lib/graphql/record-shipments", () => ({ fetchRecordShipments: fetchers.shipments }));
// The shipment and customer cards are their own chunks; here each stands in as
// what it was given, with its way back.
function standIn(kind: string) {
  return ({
    recordId,
    onBack,
    backLabel,
  }: {
    recordId: string;
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>
        {kind} card {recordId}
      </p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  );
}
vi.mock("../../shipment/desk-shipment", () => ({
  DeskShipment: standIn("Shipment"),
  DeskShipmentArtifact: () => null,
}));
vi.mock("../../customer/desk-customer", () => ({
  DeskCustomer: standIn("Customer"),
  DeskCustomerArtifact: () => null,
}));

const NOW = Math.floor(Date.now() / 1000);

/** An order as the DeskOrderCard query returns it. */
function order(overrides: Partial<DeskOrderRecord> = {}): DeskOrderRecord {
  return {
    id: "ord_1",
    status: "InProgress",
    orderNumber: "ORD-2610-0118",
    poNumber: "PO-88213",
    bol: null,
    currencyCode: "USD",
    quotedAmount: "4000",
    baseAmount: "3800",
    totalAmount: "4200",
    createdAt: NOW - 3 * 86_400,
    updatedAt: NOW - 3_600,
    customer: { id: "cus_1", code: "ACME", name: "Acme Foods" },
    charges: [
      {
        id: "och_1",
        description: "Lumper",
        amount: "85",
        invoiceId: null,
        invoicedAt: null,
        allocations: [],
      },
    ],
    ...overrides,
  } as DeskOrderRecord;
}

const entry = {
  id: "ae_1",
  operation: "update",
  timestamp: NOW - 3_600,
  changes: { status: { from: "Confirmed", to: "InProgress" } },
  comment: null,
  principalType: "session_user",
  user: { id: "usr_1", name: "Mia Chen" },
} as ActivityEntry;

const shipment = {
  id: "shp_1",
  proNumber: "S26100090038417",
  status: "InTransit",
  totalChargeAmount: "2450",
  moves: [],
} as unknown as RecordShipment;

function draw(props: Partial<Parameters<typeof DeskOrder>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const tree = (more: Partial<typeof props> = {}) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskOrder recordId="ord_1" order={["ord_0", "ord_1"]} {...handlers} {...props} {...more} />
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(tree());
  return {
    ...view,
    ...handlers,
    step: (recordId: string) => view.rerender(tree({ recordId })),
  };
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

beforeEach(() => {
  seedPermissions({
    [Resource.Order]: Operation.Read,
    [Resource.Shipment]: Operation.Read,
    [Resource.AuditLog]: Operation.Read,
    [Resource.Customer]: Operation.Read,
  });
  fetchers.order.mockReset().mockResolvedValue(order());
  fetchers.activity.mockReset().mockResolvedValue({
    rows: [entry],
    total: 8,
    endCursor: "c1",
    hasNextPage: true,
  });
  fetchers.shipments.mockReset().mockResolvedValue({
    rows: [shipment],
    total: 2,
    endCursor: null,
    hasNextPage: false,
  });
});

afterEach(() => {
  clearPermissions();
});

describe("DeskOrder", () => {
  it("draws the next order's outline, not the one it left, while the next one is read", async () => {
    const { container, step } = draw();
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });

    fetchers.order.mockReturnValue(new Promise(() => {}));
    step("ord_0");

    expect(screen.queryByRole("heading", { name: "Acme Foods", level: 2 })).toBeNull();
    expect(container.querySelector(".dk-rec-skel")).not.toBeNull();
  });

  it("holds the card's outline while the order is read, then draws how far it has come", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the order…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-track")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-rec-route")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Acme Foods", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(screen.getByText("Order ORD-2610-0118")).toBeInTheDocument();
    expect(screen.getByText("PO PO-88213")).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-amt b")).toHaveTextContent("$4,200.00");
    expect(screen.getByText("Quoted $4,000.00")).toBeInTheDocument();

    const way = section("Where it stands");
    const track = within(way).getByRole("list", { name: "Order lifecycle" });
    expect(track.querySelector('[aria-current="step"]')).toHaveTextContent("In progress");
    expect(track.querySelectorAll("li.dk-done")).toHaveLength(2);
    expect(track.querySelectorAll("li.dk-done svg")).toHaveLength(2);
    expect(within(way).getByText(/^Step 3 of 6 · last changed /u)).toBeInTheDocument();

    expect(await screen.findByText("2 shipments")).toBeInTheDocument();
    expect(fetchers.shipments).toHaveBeenCalledWith(
      { kind: "order", id: "ord_1" },
      { first: 5, after: null },
      expect.anything(),
    );

    expect(section("Legs").querySelector(".dk-rec-route-n")).toHaveTextContent("1");

    const charges = section("Charges");
    expect(within(charges).getByText("Lumper")).toBeInTheDocument();
    expect(within(charges).getByText("Not yet invoiced")).toBeInTheDocument();
    expect(within(charges).getByText("Still to invoice $85.00")).toBeInTheDocument();
    expect(within(charges).getByText("Invoiced $0.00")).toBeInTheDocument();
    expect(within(charges).getByRole("meter", { name: "Invoiced" })).toHaveAttribute(
      "aria-valuenow",
      "0",
    );
    expect(charges.querySelector(".dk-or-total")).toHaveTextContent("$85.00");

    const references = section("References");
    expect(within(references).getByText("PO-88213")).toBeInTheDocument();

    const activity = section("Activity");
    expect(await within(activity).findByText("Changed status")).toBeInTheDocument();
    const diff = activity.querySelector(".dk-rec-diff") as HTMLElement;
    expect(within(diff).getByText("status")).toBeInTheDocument();
    expect(within(diff).getByText("Confirmed").tagName).toBe("S");
    expect(within(diff).getByText("In progress").tagName).toBe("B");
    expect(within(activity).getByText(/^Mia Chen · /u)).toBeInTheDocument();
    expect(within(activity).getByText("8 changes")).toBeInTheDocument();
    expect(fetchers.activity).toHaveBeenCalledWith(
      "ord_1",
      { first: 5, after: null },
      expect.anything(),
    );
  });

  it("reads earlier activity on request", async () => {
    draw();
    expect(await screen.findByText("Changed status")).toBeInTheDocument();

    fetchers.activity.mockResolvedValue({
      rows: [{ ...entry, id: "ae_2", operation: "create", changes: null }],
      total: null,
      endCursor: null,
      hasNextPage: false,
    });
    fireEvent.click(within(section("Activity")).getByRole("button", { name: "Show more" }));

    expect(await screen.findByText("Created")).toBeInTheDocument();
    expect(fetchers.activity).toHaveBeenLastCalledWith(
      "ord_1",
      { first: 5, after: "c1" },
      expect.anything(),
    );
  });

  it("takes a canceled order off its way, and says so", async () => {
    fetchers.order.mockResolvedValue(order({ status: "Canceled" }));
    draw();

    expect(await screen.findByText("This order was canceled.")).toBeInTheDocument();
    expect(section("Where it stands").querySelector('[aria-current="step"]')).toBeNull();
    expect(section("Where it stands").querySelector(".dk-rec-track")).toHaveClass("dk-off");
    expect(screen.getByText(/^Canceled · last changed /u)).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-st")).toHaveTextContent("Canceled");
  });

  it("opens the customer and a leg in the card's place", async () => {
    draw();

    fireEvent.click(await screen.findByRole("button", { name: /Open customer/ }));
    expect(await screen.findByText("Customer card cus_1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to ORD-2610-0118" }));

    fireEvent.click(await screen.findByRole("button", { name: /S26100090038417/ }));
    expect(await screen.findByText("Shipment card shp_1")).toBeInTheDocument();
  });

  it("stands with nothing around it: not priced, no charges, no legs, no activity", async () => {
    fetchers.order.mockResolvedValue(
      order({
        status: "Draft",
        totalAmount: null,
        quotedAmount: null,
        poNumber: null,
        charges: [],
      }),
    );
    fetchers.shipments.mockResolvedValue({
      rows: [],
      total: 0,
      endCursor: null,
      hasNextPage: false,
    });
    fetchers.activity.mockResolvedValue({
      rows: [],
      total: 0,
      endCursor: null,
      hasNextPage: false,
    });
    draw();

    expect(await screen.findByText("Not priced yet")).toBeInTheDocument();
    expect(screen.getByText("No charge has been added to this order.")).toBeInTheDocument();
    expect(await screen.findByText("No shipment carries this order yet.")).toBeInTheDocument();
    expect(await screen.findByText("Nothing has been done to this order yet.")).toBeInTheDocument();
  });

  it("says a section that could not be read, and draws the rest", async () => {
    fetchers.activity.mockRejectedValue(new Error("Audit log is unavailable"));
    draw();

    expect(await screen.findByText("Audit log is unavailable")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Charges", level: 3 })).toBeInTheDocument();
  });

  it("says the order could not be read, with the way back", async () => {
    fetchers.order.mockRejectedValue(new Error("Order not found"));
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", { name: "This order could not be read", level: 2 }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
    expect(fetchers.activity).not.toHaveBeenCalled();
  });

  it("opens with only what the person may read, asking for nothing else", async () => {
    seedPermissions({ [Resource.Order]: Operation.Read });
    draw();

    expect(
      await screen.findByRole("heading", { name: "Acme Foods", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Legs" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Activity" })).toBeNull();
    // The customer is a linked record: without customer read there is no way to it.
    expect(screen.queryByRole("link", { name: /Open customer/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /Open customer/ })).toBeNull();
    expect(fetchers.shipments).not.toHaveBeenCalled();
    expect(fetchers.activity).not.toHaveBeenCalled();
  });

  it("leaves the outlines of sections the person may not read out of the skeleton", () => {
    seedPermissions({ [Resource.Order]: Operation.Read });
    fetchers.order.mockReturnValue(new Promise(() => {}));
    const { container } = draw();

    expect(container.querySelectorAll(".dk-rec-skel .dk-rec-sec")).toHaveLength(3);
  });

  it("draws the same card as an artifact, with nothing to step through", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskOrderArtifact artifact={{} as never} recordId="ord_1" fallback={null} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(
      await screen.findByRole("heading", { name: "Acme Foods", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Next order" })).toBeNull();
  });
});
