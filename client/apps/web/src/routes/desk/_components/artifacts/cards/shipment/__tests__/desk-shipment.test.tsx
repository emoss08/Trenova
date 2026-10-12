import type { AssistantArtifact } from "@/types/assistant";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { documentContentUrl } from "@/services/document";
import {
  FakeIntersectionObserver,
  installIntersectionObserver,
} from "@/test/intersection-observer";
import type { ShipmentEvent } from "@/types/shipment-event";
import type { Document } from "@trenova/shared/types/document";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import type {
  Assignment,
  Shipment,
  ShipmentBillingReadiness,
  ShipmentCommodity,
  ShipmentMove,
  Stop,
} from "@trenova/shared/types/shipment";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskShipment, DeskShipmentArtifact } from "../desk-shipment";

const service = vi.hoisted(() => ({ get: vi.fn(), getBillingReadiness: vi.fn() }));
const eventService = vi.hoisted(() => ({ list: vi.fn() }));
const documentService = vi.hoisted(() => ({ getByResource: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: {
    shipmentService: service,
    shipmentEventService: eventService,
    documentService,
  },
}));

const NOW = Math.floor(Date.now() / 1000);
const HOUR = 3600;

/** A stop's location as the shipment query selects it, the state by its abbreviation. */
function location(name: string, city: string, abbreviation: string): Stop["location"] {
  return {
    id: `loc_${city.toLowerCase()}`,
    name,
    code: name.slice(0, 4).toUpperCase(),
    city,
    state: { abbreviation, name: abbreviation },
  } as unknown as Stop["location"];
}

/** A stop after the client schema: unset times are null, the address line may be absent. */
function stop(overrides: Partial<Stop> & Pick<Stop, "id">): Stop {
  return {
    organizationId: "org_1",
    businessUnitId: "bu_1",
    shipmentMoveId: "mov_1",
    locationId: "loc_1",
    status: "New",
    type: "Pickup",
    scheduleType: "Appointment",
    sequence: 0,
    pieces: null,
    weight: null,
    scheduledWindowStart: NOW + 4 * HOUR,
    scheduledWindowEnd: NOW + 6 * HOUR,
    actualArrival: null,
    actualDeparture: null,
    location: location("Dallas DC", "Dallas", "TX"),
    ...overrides,
  };
}

function driver(): Assignment {
  return {
    id: "asn_1",
    shipmentMoveId: "mov_1",
    status: "InProgress",
    primaryWorkerId: "wrk_1",
    secondaryWorkerId: null,
    tractorId: "trc_1",
    trailerId: "trl_1",
    tractor: { id: "trc_1", code: "T-101" },
    trailer: { id: "trl_1", code: "TR-9", equipmentTypeId: null },
    primaryWorker: {
      id: "wrk_1",
      firstName: "Ana",
      lastName: "Ruiz",
      wholeName: "Ana Ruiz",
      profilePicUrl: null,
    },
    secondaryWorker: null,
  };
}

function carrier(status: "Pending" | "Confirmed" | "Canceled"): ShipmentMove["carrierAssignment"] {
  return {
    id: `ca_${status}`,
    shipmentMoveId: "mov_2",
    carrierId: "car_1",
    status,
    rateMethod: "Flat",
    baseRate: 1500,
    baseAmount: 1500,
    fuelSurcharge: 150,
    accessorialTotal: 0,
    totalCost: 1650,
    currencyCode: "USD",
    proNumber: "C-77",
    externalDriverName: "Sam Ortiz",
    externalDriverPhone: null,
    externalTractorNumber: null,
    externalTrailerNumber: null,
    carrier: { id: "car_1", code: "BLUE", name: "Blue Line Carriers", scac: null },
  } as ShipmentMove["carrierAssignment"];
}

function move(overrides: Partial<ShipmentMove> & Pick<ShipmentMove, "id">): ShipmentMove {
  return {
    shipmentId: "shp_1",
    status: "InTransit",
    loaded: true,
    sequence: 0,
    distance: 780,
    coverageType: "driver",
    stops: [],
    assignment: null,
    carrierAssignment: null,
    ...overrides,
  };
}

/**
 * One move, driven by a company driver, four stops in sequence: a pickup done,
 * a second pickup canceled, a split delivery next and the final delivery after it.
 * The stops arrive out of order, as nothing in the contract promises otherwise.
 */
function oneMove(): ShipmentMove[] {
  return [
    move({
      id: "mov_1",
      assignment: driver(),
      stops: [
        stop({
          id: "stp_4",
          sequence: 3,
          type: "Delivery",
          scheduledWindowStart: NOW + 20 * HOUR,
          scheduledWindowEnd: NOW + 22 * HOUR,
          location: location("Atlanta Grocers DC", "Atlanta", "GA"),
        }),
        stop({
          id: "stp_1",
          sequence: 0,
          status: "Completed",
          scheduledWindowStart: NOW - 30 * HOUR,
          scheduledWindowEnd: NOW - 28 * HOUR,
          actualArrival: NOW - 29 * HOUR,
          actualDeparture: NOW - 28 * HOUR,
        }),
        stop({
          id: "stp_2",
          sequence: 1,
          status: "Canceled",
          scheduledWindowStart: NOW - 26 * HOUR,
          scheduledWindowEnd: null,
          location: location("Waco Cold Storage", "Waco", "TX"),
        }),
        stop({
          id: "stp_3",
          sequence: 2,
          type: "SplitDelivery",
          scheduleType: "Open",
          location: location("Shreveport Yard", "Shreveport", "LA"),
        }),
      ],
    }),
  ];
}

function commodity(overrides: Partial<ShipmentCommodity> = {}): ShipmentCommodity {
  return {
    id: "sc_1",
    commodityId: "com_1",
    pieces: 20,
    weight: 38_000,
    commodity: {
      id: "com_1",
      name: "Frozen poultry",
      hazardousMaterialId: null,
      minTemperature: -10,
      maxTemperature: 0,
      freightClass: "Class70",
      stackable: true,
      fragile: false,
    },
    ...overrides,
  } as unknown as ShipmentCommodity;
}

/** A shipment as the GraphQL shipment query returns it, after the client schema. */
function shipment(overrides: Partial<Shipment> = {}): Shipment {
  return {
    id: "shp_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    status: "InTransit",
    stage: "Moving",
    eta: { estimatedArrival: NOW + 19 * HOUR, slackMinutes: 80, verdict: "OnTime", reason: null },
    tenderStatus: null,
    entryMethod: "Manual",
    proNumber: "PRO-1001",
    bol: "B-2231",
    externalReference: null,
    cancelReason: "",
    freightChargeAmount: 2300,
    otherChargeAmount: 150,
    totalChargeAmount: 2450,
    baseRate: 2300,
    pieces: null,
    weight: null,
    temperatureMin: null,
    temperatureMax: null,
    actualShipDate: NOW - 28 * HOUR,
    actualDeliveryDate: null,
    canceledAt: null,
    billedAt: null,
    markedReadyToBillAt: null,
    transferredToBillingAt: null,
    rateOverrideAmount: null,
    rateOverrideReason: null,
    rateLocked: false,
    autoRated: true,
    ratingDetail: null,
    formulaTemplate: null,
    profitabilityEstimate: {
      shipmentId: "shp_1",
      loadedMiles: 780,
      deadheadMiles: 40,
      totalMiles: 820,
      costPerMile: "2.10",
      estimatedCost: "1722.00",
      profit: "728.00",
      marginPercent: "29.7",
      breakEvenRpm: "2.21",
      targetMarginPercent: "20",
      missingDistance: false,
    },
    moves: oneMove(),
    additionalCharges: [],
    commodities: [commodity()],
    chargeAllocations: [],
    billingSplitSummary: [],
    freightAllocations: [],
    customer: { id: "cus_1", name: "Acme Foods", code: "ACME" },
    billToCustomer: { id: "cus_2", name: "Acme Holdings", code: "ACMEH" },
    owner: { id: "usr_9", name: "Jordan Lee" },
    ...overrides,
  } as unknown as Shipment;
}

/** The shipment's events as the list query returns them: newest first. */
function events(): ShipmentEvent[] {
  return [
    {
      __typename: "ShipmentMoveEvent",
      id: "sev_2",
      organizationId: "org_1",
      businessUnitId: "bu_1",
      shipmentId: "shp_1",
      type: "MoveDeparted",
      severity: "brand",
      actorType: "system",
      actorLabel: "System",
      summary: "Left Dallas DC",
      metadata: {},
      occurredAt: NOW - 28 * HOUR,
    },
    {
      __typename: "ShipmentLifecycleEvent",
      id: "sev_1",
      organizationId: "org_1",
      businessUnitId: "bu_1",
      shipmentId: "shp_1",
      type: "ShipmentCreated",
      severity: "info",
      actorType: "user",
      actorId: "usr_9",
      actorLabel: "",
      actor: { name: "Jordan Lee" },
      summary: "Shipment created",
      metadata: {},
      occurredAt: NOW - 48 * HOUR,
    },
  ] as ShipmentEvent[];
}

function readiness(overrides: Partial<ShipmentBillingReadiness> = {}): ShipmentBillingReadiness {
  return {
    shipmentId: "shp_1",
    shipmentStatus: "Completed",
    policy: {
      shipmentBillingRequirementEnforcement: "Required",
      rateValidationEnforcement: "Warn",
      billingExceptionDisposition: "Queue",
      notifyOnBillingExceptions: false,
      readyToBillAssignmentMode: "Manual",
      billingQueueTransferMode: "Manual",
    },
    requirements: [
      {
        documentTypeId: "dt_1",
        documentTypeCode: "BOL",
        documentTypeName: "Bill of lading",
        satisfied: true,
        documentCount: 1,
        documentIds: ["doc_1"],
        ineligibleDocuments: [],
      },
      {
        documentTypeId: "dt_2",
        documentTypeCode: "POD",
        documentTypeName: "Proof of delivery",
        satisfied: false,
        documentCount: 0,
        documentIds: [],
        ineligibleDocuments: [],
      },
    ],
    missingRequirements: [
      {
        documentTypeId: "dt_2",
        documentTypeCode: "POD",
        documentTypeName: "Proof of delivery",
        satisfied: false,
        documentCount: 0,
        documentIds: [],
        ineligibleDocuments: [],
      },
    ],
    validationFailures: [],
    warnings: [],
    serviceFailureContext: { hasUnresolved: false, unresolvedCount: 0, serviceFailureIds: [] },
    canMarkReadyToInvoice: false,
    shouldAutoMarkReadyToInvoice: false,
    shouldAutoTransferToBilling: false,
    shouldAutoApproveBilling: false,
    payers: [],
    ...overrides,
  };
}

/** Every stop of the one move run and completed, the truck at the consignee. */
function delivered(overrides: Partial<Shipment> = {}): Shipment {
  const [only] = oneMove();
  return shipment({
    status: "Completed",
    stage: "Delivered",
    eta: null,
    actualDeliveryDate: NOW - 2 * HOUR,
    moves: [
      {
        ...only,
        status: "Completed",
        stops: only.stops.map((entry) =>
          entry.status === "Canceled"
            ? entry
            : {
                ...entry,
                status: "Completed",
                actualArrival: entry.scheduledWindowStart,
                actualDeparture: entry.scheduledWindowStart + HOUR,
              },
        ),
      },
    ],
    ...overrides,
  });
}

function wrap(ui: ReactElement) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
}

function draw(props: Partial<Parameters<typeof DeskShipment>[0]> = {}) {
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const view = wrap(
    <DeskShipment recordId="shp_1" order={["shp_0", "shp_1", "shp_2"]} {...handlers} {...props} />,
  );
  return { ...view, ...handlers };
}

function stopNames(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll(".dk-sh-stop .dk-sh-sh b")).map(
    (node) => node.textContent ?? "",
  );
}

beforeEach(() => {
  seedPermissions({ [Resource.Document]: Operation.Read });
  service.get.mockReset();
  service.getBillingReadiness.mockReset();
  eventService.list.mockReset();
  service.get.mockResolvedValue(shipment());
  service.getBillingReadiness.mockResolvedValue(readiness());
  eventService.list.mockResolvedValue(events());
  documentService.getByResource.mockReset();
  documentService.getByResource.mockResolvedValue([]);
  installIntersectionObserver();
});

afterEach(() => {
  clearPermissions();
  vi.unstubAllGlobals();
});

/** Scrolls the card down to its documents, which are read only once they come near. */
async function scrollToDocuments(container: HTMLElement) {
  await screen.findByRole("heading", { name: "Documents" });
  const body = container.querySelector(".dk-rec-docs-body");
  expect(documentService.getByResource).not.toHaveBeenCalled();
  act(() => FakeIntersectionObserver.show(body as Element));
}

/** A document on the shipment as the documents endpoint returns it, with its type. */
function attached(fields: Partial<Document> & { id: string }): Document {
  return {
    originalName: `${fields.id}.pdf`,
    fileType: "application/pdf",
    fileSize: 120_000,
    createdAt: NOW - HOUR,
    isCurrentVersion: true,
    status: "Active",
    approvedAt: null,
    previewStatus: "Ready",
    contentStatus: "Indexed",
    shipmentDraftStatus: "Unavailable",
    processingProfile: "none",
    documentType: null,
    ...fields,
  } as Document;
}

describe("DeskShipment", () => {
  it("holds the card's outline while the shipment is read, then draws the shipment", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the shipment…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-head")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-sh-now")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
  });

  it("leads with the lane, who it is for, what it bills and where it stands", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    const head = container.querySelector(".dk-rec-head") as HTMLElement;
    expect(within(head).getByText("PRO-1001")).toBeInTheDocument();
    expect(within(head).getByText("BOL B-2231")).toBeInTheDocument();
    expect(within(head).queryByText("EDI")).not.toBeInTheDocument();
    expect(within(head).getByText("Acme Foods · bills to Acme Holdings")).toBeInTheDocument();
    expect(head.querySelector(".dk-rec-amt b")).toHaveTextContent("$2,450.00");
    expect(within(head).getByText("29.7% margin")).toBeInTheDocument();
    expect(within(head).getByText("Moving")).toBeInTheDocument();
    expect(within(head).getByText("780 mi")).toBeInTheDocument();
    expect(within(head).getByText("Jordan Lee")).toBeInTheDocument();
  });

  it("names the customer alone when it pays its own bills, and marks an EDI load", async () => {
    service.get.mockResolvedValue(
      shipment({
        entryMethod: "EDI",
        bol: null,
        billToCustomer: { id: "cus_1", name: "Acme Foods", code: "ACME" },
      }),
    );
    const { container } = draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    const head = container.querySelector(".dk-rec-head") as HTMLElement;
    expect(within(head).getByText("Acme Foods")).toBeInTheDocument();
    expect(within(head).queryByText(/bills to/)).not.toBeInTheDocument();
    expect(within(head).getByText("EDI")).toBeInTheDocument();
    expect(within(head).queryByText(/^BOL/)).not.toBeInTheDocument();
  });

  it("falls back to the PRO for a title when the shipment has no stops", async () => {
    service.get.mockResolvedValue(shipment({ moves: [] }));
    const { container } = draw();

    expect(await screen.findByRole("heading", { name: "PRO-1001" })).toBeInTheDocument();
    expect(container.querySelector(".dk-sh-stops")).toBeNull();
    expect(container.querySelector(".dk-sh-truck")).toBeNull();
  });

  it("runs the stops down the rail in sequence, the canceled one struck", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    expect(stopNames(container)).toEqual([
      "Dallas DC",
      "Waco Cold Storage",
      "Shreveport Yard",
      "Atlanta Grocers DC",
    ]);
    const rows = container.querySelectorAll(".dk-sh-stop");
    expect(rows[0]).toHaveClass("dk-done");
    expect(rows[1]).toHaveClass("dk-canceled");
    expect(rows[2]).toHaveClass("dk-next");
    expect(rows[3]).toHaveClass("dk-later");
    expect(within(rows[2] as HTMLElement).getByText("Split delivery")).toBeInTheDocument();
    expect(screen.getByText("4 stops · 780 mi")).toBeInTheDocument();
    expect(container.querySelector(".dk-sh-mv")).toBeNull();
  });

  it("dots every stop on the strip, the truck between the done and the next", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    const dots = Array.from(container.querySelectorAll<HTMLElement>(".dk-sh-now .dk-sh-dot"));
    const phases = dots.map(
      (dot) => /dk-(done|current|next|later|canceled)/.exec(dot.className)?.[1],
    );
    expect(phases).toEqual(["done", "canceled", "next", "later"]);
    const lefts = dots.map((dot) => Number.parseFloat(dot.style.left));
    expect(lefts).toEqual([...lefts].sort((a, b) => a - b));
    const truck = Number.parseFloat(
      (container.querySelector(".dk-sh-truck") as HTMLElement).style.left,
    );
    expect(truck).toBeGreaterThanOrEqual(lefts[0]);
    expect(truck).toBeLessThanOrEqual(lefts[2]);
    expect(container.querySelector(".dk-sh-now")).toHaveClass("dk-sh-live");
  });

  it("marks a stop on the strip while its row is pointed at, and back", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    const rows = container.querySelectorAll(".dk-sh-stop");
    const dots = container.querySelectorAll(".dk-sh-now .dk-sh-dot");
    fireEvent.mouseEnter(rows[2]);
    expect(dots[2]).toHaveClass("dk-on");
    expect(dots[0]).not.toHaveClass("dk-on");

    fireEvent.mouseLeave(rows[2]);
    fireEvent.mouseEnter(dots[3]);
    expect(rows[3]).toHaveClass("dk-on");
    expect(dots[2]).not.toHaveClass("dk-on");
  });

  it("does not pulse the truck of a shipment that is not moving", async () => {
    service.get.mockResolvedValue(shipment({ status: "Assigned", stage: "Scheduled" }));
    const { container } = draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    expect(container.querySelector(".dk-sh-now")).not.toHaveClass("dk-sh-live");
  });

  it("names the company driver and the equipment that cover a single move", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    const cover = container.querySelector(".dk-sh-cover") as HTMLElement;
    expect(within(cover).getByText("Ana Ruiz")).toBeInTheDocument();
    expect(within(cover).getByText("AR")).toBeInTheDocument();
    expect(within(cover).getByText("Tractor T-101 · Trailer TR-9")).toBeInTheDocument();
    expect(within(cover).queryByText(/Move \d/)).not.toBeInTheDocument();
  });

  it("breaks the rail where each move begins and covers each move its own way", async () => {
    service.get.mockResolvedValue(
      shipment({
        tenderStatus: "Accepted",
        moves: [
          move({
            id: "mov_3",
            sequence: 2,
            status: "New",
            coverageType: "unassigned",
            carrierAssignment: carrier("Canceled"),
            stops: [
              stop({ id: "stp_e", sequence: 0, location: location("Macon Yard", "Macon", "GA") }),
              stop({
                id: "stp_f",
                sequence: 1,
                type: "Delivery",
                location: location("Savannah Port", "Savannah", "GA"),
              }),
            ],
          }),
          move({
            id: "mov_1",
            sequence: 0,
            status: "Completed",
            assignment: driver(),
            stops: [
              stop({ id: "stp_a", sequence: 0, status: "Completed", actualArrival: NOW - HOUR }),
              stop({
                id: "stp_b",
                sequence: 1,
                type: "Delivery",
                status: "Completed",
                location: location("Memphis Crossdock", "Memphis", "TN"),
              }),
            ],
          }),
          move({
            id: "mov_2",
            sequence: 1,
            coverageType: "carrier",
            carrierAssignment: carrier("Confirmed"),
            stops: [
              stop({
                id: "stp_c",
                sequence: 0,
                location: location("Memphis Crossdock", "Memphis", "TN"),
              }),
              stop({
                id: "stp_d",
                sequence: 1,
                type: "Delivery",
                location: location("Atlanta Grocers DC", "Atlanta", "GA"),
              }),
            ],
          }),
        ],
      }),
    );
    const { container } = draw();
    await screen.findByRole("heading", { name: /Dallas, TX →/ });

    const dividers = Array.from(container.querySelectorAll(".dk-sh-mv"));
    expect(dividers.map((divider) => divider.querySelector("b")?.textContent)).toEqual([
      "Move 1",
      "Move 2",
      "Move 3",
    ]);
    expect(dividers[0]).toHaveTextContent("Ana Ruiz");
    expect(dividers[1]).toHaveTextContent("Blue Line Carriers");
    expect(dividers[2]).toHaveTextContent("No driver or carrier yet");
    expect(stopNames(container)).toEqual([
      "Dallas DC",
      "Memphis Crossdock",
      "Memphis Crossdock",
      "Atlanta Grocers DC",
      "Macon Yard",
      "Savannah Port",
    ]);
    expect(dividers[1].nextElementSibling).toHaveTextContent("Memphis Crossdock");

    const cover = container.querySelector(".dk-sh-cover") as HTMLElement;
    const rows = Array.from(cover.querySelectorAll<HTMLElement>(".dk-sh-cv"));
    expect(rows).toHaveLength(3);
    expect(within(rows[0]).getByText("Move 1")).toBeInTheDocument();
    expect(within(rows[0]).getByText("Ana Ruiz")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Blue Line Carriers")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Confirmed")).toBeInTheDocument();
    expect(within(rows[1]).getByText("PRO C-77 · Driver Sam Ortiz")).toBeInTheDocument();
    expect(within(rows[1]).getByText("$1,650.00")).toBeInTheDocument();
    expect(rows[2]).toHaveClass("dk-none");
    expect(within(rows[2]).getByText("No driver or carrier yet")).toBeInTheDocument();
    expect(within(rows[2]).queryByText("Blue Line Carriers")).not.toBeInTheDocument();
    expect(screen.getByText("Tender accepted")).toBeInTheDocument();
  });

  it("lists the freight and its commodity lines", async () => {
    service.get.mockResolvedValue(
      shipment({
        commodities: [
          commodity(),
          commodity({
            id: "sc_2",
            commodityId: "com_2",
            pieces: 4,
            weight: 1_200,
            commodity: {
              id: "com_2",
              name: "Dry ice",
              hazardousMaterialId: "hm_1",
              minTemperature: null,
              maxTemperature: null,
              freightClass: null,
              stackable: false,
              fragile: true,
            } as unknown as ShipmentCommodity["commodity"],
          }),
        ],
      }),
    );
    const { container } = draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    expect(screen.getByRole("heading", { name: "Freight" })).toBeInTheDocument();
    const lines = container.querySelector(".dk-sh-fl") as HTMLElement;
    expect(within(lines).getByText("Frozen poultry")).toBeInTheDocument();
    expect(within(lines).getByText("Class 70")).toBeInTheDocument();
    expect(within(lines).getByText("Dry ice")).toBeInTheDocument();
    expect(within(lines).getByText("Fragile")).toBeInTheDocument();
    expect(within(lines).getByText("Not stackable")).toBeInTheDocument();
    const chips = container.querySelector(".dk-sh-chips") as HTMLElement;
    expect(within(chips).getByText("Hazmat")).toBeInTheDocument();
  });

  it("leaves the freight out when there is nothing to say about it", async () => {
    service.get.mockResolvedValue(shipment({ commodities: [], pieces: null, weight: null }));
    draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    expect(screen.queryByRole("heading", { name: "Freight" })).not.toBeInTheDocument();
  });

  it("states the charges to the total and the margin against its target", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    const ledger = container.querySelector(".dk-sh-led") as HTMLElement;
    expect(within(ledger).getByText("Total")).toBeInTheDocument();
    expect(ledger.querySelector(".dk-sh-lt .dk-sh-n")).toHaveTextContent("$2,450.00");
    const meter = container.querySelector(".dk-sh-margin") as HTMLElement;
    expect(meter).toHaveClass("dk-ok");
    expect(within(meter).getByText("$728.00")).toBeInTheDocument();
    expect(within(meter).getByText("$1,722.00")).toBeInTheDocument();
    expect(meter.querySelector(".dk-sh-tick")).not.toBeNull();
    expect(within(meter).queryByText(/rough estimate/)).not.toBeInTheDocument();
  });

  it("says the margin is rough when the distance is missing", async () => {
    const estimate = shipment().profitabilityEstimate as NonNullable<
      Shipment["profitabilityEstimate"]
    >;
    service.get.mockResolvedValue(
      shipment({ profitabilityEstimate: { ...estimate, missingDistance: true } }),
    );
    draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    expect(screen.getByText("Distance missing; margin is a rough estimate")).toBeInTheDocument();
  });

  it("draws no margin without an estimate, and reads the figure as the total", async () => {
    service.get.mockResolvedValue(shipment({ profitabilityEstimate: null }));
    const { container } = draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    expect(container.querySelector(".dk-sh-margin")).toBeNull();
    const head = container.querySelector(".dk-rec-head") as HTMLElement;
    expect(within(head).getByText("total")).toBeInTheDocument();
  });

  it("does not ask what stands before the invoice while the shipment is moving", async () => {
    draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    expect(service.getBillingReadiness).not.toHaveBeenCalled();
    expect(screen.queryByRole("heading", { name: "Billing" })).not.toBeInTheDocument();
  });

  it("lists what stands between a delivered shipment and its invoice", async () => {
    service.get.mockResolvedValue(delivered());
    draw();

    expect(await screen.findByRole("heading", { name: "Billing" })).toBeInTheDocument();
    expect(service.getBillingReadiness).toHaveBeenCalledWith("shp_1");
    await waitFor(() =>
      expect(screen.getByText("Proof of delivery").closest(".dk-rec-ck")).toHaveClass("dk-bad"),
    );
    expect(screen.getByText("Bill of lading").closest(".dk-rec-ck")).toHaveClass("dk-ok");
    expect(screen.getByText("1 to clear")).toBeInTheDocument();
  });

  it("says a delivered shipment is ready to bill when nothing stands in the way", async () => {
    service.get.mockResolvedValue(delivered());
    service.getBillingReadiness.mockResolvedValue(
      readiness({
        requirements: [readiness().requirements[0]],
        missingRequirements: [],
        canMarkReadyToInvoice: true,
      }),
    );
    draw();

    expect(await screen.findByText("Ready to bill")).toBeInTheDocument();
  });

  it("notes the invoice instead of the checks once the shipment is billed", async () => {
    const billedAt = NOW - 2 * HOUR;
    service.get.mockResolvedValue(delivered({ status: "Invoiced", billedAt }));
    draw();

    expect(await screen.findByRole("heading", { name: "Billing" })).toBeInTheDocument();
    expect(screen.getByText(/^Invoiced /)).toBeInTheDocument();
    expect(service.getBillingReadiness).not.toHaveBeenCalled();
  });

  it("lists the latest activity oldest first, with a way to the rest", async () => {
    draw();

    const created = await screen.findByText("Shipment created");
    const left = screen.getByText("Left Dallas DC");
    expect(created.compareDocumentPosition(left) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(eventService.list).toHaveBeenCalledWith({ shipmentId: "shp_1", limit: 5 });
    expect(screen.getByRole("link", { name: /All activity/ })).toHaveAttribute(
      "href",
      "/shipment-management/shipments?expanded=shp_1&panelType=edit&panelEntityId=shp_1",
    );
  });

  it("leaves the activity out when nothing has happened yet", async () => {
    eventService.list.mockResolvedValue([]);
    draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    expect(screen.queryByRole("heading", { name: "Activity" })).not.toBeInTheDocument();
  });

  it("shows the shipment's documents at the bottom, each opening the document itself", async () => {
    documentService.getByResource.mockResolvedValue([
      attached({
        id: "doc_pod",
        originalName: "pod-1042.jpg",
        fileType: "image/jpeg",
        status: "Active",
        approvedAt: NOW,
        documentType: { name: "Proof of delivery", code: "POD" } as Document["documentType"],
      }),
      attached({ id: "doc_rate", previewStatus: "Pending", createdAt: NOW - 2 * HOUR }),
    ]);
    const { container } = draw();
    await scrollToDocuments(container);

    const pod = await screen.findByRole("link", { name: /Proof of delivery/ });
    expect(documentService.getByResource).toHaveBeenCalledWith("shipment", "shp_1", "", {
      includeDocumentType: "true",
    });
    expect(pod).toHaveAttribute("href", documentContentUrl("doc_pod", "view"));
    expect(pod).toHaveAttribute("target", "_blank");
    expect(within(pod).getByText("POD")).toBeInTheDocument();
    expect(within(pod).getByText("Approved")).toBeInTheDocument();
    expect(pod.querySelector("img")).toHaveAttribute(
      "src",
      documentContentUrl("doc_pod", "preview"),
    );

    const pending = screen.getByRole("link", { name: /doc_rate\.pdf/ });
    expect(pending.querySelector("img")).toBeNull();
    expect(within(pending).getByText("Preparing…")).toBeInTheDocument();
    expect(screen.getByText("2 documents")).toBeInTheDocument();

    const sections = Array.from(container.querySelectorAll(".dk-rec-sec h3")).map(
      (heading) => heading.textContent,
    );
    expect(sections.at(-1)).toBe("Documents");
  });

  it("keeps every document in one rail that scrolls sideways, with the way to manage them", async () => {
    documentService.getByResource.mockResolvedValue(
      Array.from({ length: 11 }, (_, index) =>
        attached({ id: `doc_${index}`, createdAt: NOW - index * HOUR }),
      ),
    );
    const { container } = draw();
    await scrollToDocuments(container);

    expect(await screen.findByText("11 documents")).toBeInTheDocument();
    const rail = container.querySelector(".dk-rec-docs-rail");
    expect(rail?.querySelectorAll("a.dk-rec-doc")).toHaveLength(11);
    expect(screen.getByRole("link", { name: /Manage/ })).toHaveAttribute(
      "href",
      "/shipment-management/shipments?expanded=shp_1&panelType=edit&panelEntityId=shp_1",
    );
  });

  it("says when the shipment has no documents yet", async () => {
    const { container } = draw();
    await scrollToDocuments(container);

    expect(await screen.findByText("No documents yet.")).toBeInTheDocument();
  });

  it("says why its documents cannot be read, and draws the rest of the card", async () => {
    documentService.getByResource.mockRejectedValue(new Error("Document storage is unavailable"));
    const { container } = draw();
    await scrollToDocuments(container);

    expect(await screen.findByText("Document storage is unavailable")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Dallas, TX → Atlanta, GA" })).toBeInTheDocument();
  });

  it("opens the shipment on the board", async () => {
    draw();

    expect(await screen.findByRole("link", { name: /Open shipment/ })).toHaveAttribute(
      "href",
      "/shipment-management/shipments?expanded=shp_1&panelType=edit&panelEntityId=shp_1",
    );
    expect(screen.getByRole("button", { name: /Copy PRO/ })).toBeInTheDocument();
  });

  it("steps through the table's rows and goes back to it", async () => {
    const { onSelect, onBack } = draw();
    await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" });

    expect(screen.getByText("2 of 3")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Next shipment" }));
    fireEvent.click(screen.getByRole("button", { name: "Previous shipment" }));
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));

    expect(onSelect.mock.calls).toEqual([["shp_2"], ["shp_0"]]);
    expect(onBack).toHaveBeenCalledOnce();
  });

  it("draws the card as an artifact, with nothing to step through and no way back", async () => {
    wrap(
      <DeskShipmentArtifact artifact={{} as AssistantArtifact} recordId="shp_1" fallback={null} />,
    );

    expect(
      await screen.findByRole("heading", { name: "Dallas, TX → Atlanta, GA" }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Back to the table" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Next shipment" })).not.toBeInTheDocument();
  });

  it("says when the shipment cannot be read, with the way back", async () => {
    service.get.mockRejectedValue(new Error("not found"));
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", { name: "This shipment could not be read" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalledOnce();
  });
});
