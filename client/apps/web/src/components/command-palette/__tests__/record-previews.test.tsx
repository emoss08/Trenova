import type { JourneyStop } from "@/routes/desk/_components/artifacts/cards/shipment/shipment-card";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CustomerPreview } from "../_components/preview/customer-preview";
import { StopList } from "../_components/preview/shipment-preview";
import { WorkerPreview } from "../_components/preview/worker-preview";
import { record } from "./palette-test-fixtures";

const { responses } = vi.hoisted(() => ({
  responses: new Map<string, unknown>(),
}));

vi.mock("@/services/api", () => ({ apiService: {} }));

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/graphql")>();
  return {
    ...actual,
    requestGraphQL: vi.fn(async ({ operationName }: { operationName: string }) => {
      if (!responses.has(operationName)) {
        throw new Error(`No response for ${operationName}`);
      }
      return responses.get(operationName);
    }),
  };
});

const DAY = 86_400;
const HOUR_MS = 3_600_000;

function draw(ui: React.ReactElement) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

afterEach(() => {
  responses.clear();
});

/** A worker as `DeskWorkerCard` returns it. */
function deskWorker(fields: Record<string, unknown> = {}, profile: Record<string, unknown> = {}) {
  const now = Math.floor(Date.now() / 1000);
  return {
    id: "wrk_1",
    firstName: "Dana",
    lastName: "Reyes",
    externalId: "D-104",
    type: "Employee",
    driverType: "OTR",
    status: "Active",
    city: "Dallas",
    state: { abbreviation: "TX" },
    assignmentBlocked: "",
    canBeAssigned: true,
    availableForDispatch: true,
    profile: {
      isQualified: true,
      disqualificationReason: "",
      cdlClass: "A",
      licenseState: { abbreviation: "TX" },
      endorsement: "O",
      cdlRestrictions: "",
      licenseExpiry: now + 400 * DAY,
      medicalCardExpiry: now + 400 * DAY,
      hazmatExpiry: null,
      twicExpiry: null,
      physicalDueDate: null,
      mvrDueDate: null,
      ...profile,
    },
    ...fields,
  };
}

/** The driver's clocks as `WorkerHosState` returns them. */
function hosState() {
  return {
    workerId: "wrk_1",
    dutyStatus: "Driving",
    driveRemainingMs: 6 * HOUR_MS,
    driveLimitMs: 11 * HOUR_MS,
    shiftRemainingMs: 8 * HOUR_MS,
    shiftLimitMs: 14 * HOUR_MS,
    cycleRemainingMs: 40 * HOUR_MS,
    cycleLimitMs: 70 * HOUR_MS,
    breakRemainingMs: 2 * HOUR_MS,
    breakLimitMs: 8 * HOUR_MS,
    recordedAt: Math.floor(Date.now() / 1000) - 120,
  };
}

const NO_SHIPMENTS = { shipments: { edges: [], totalCount: 0, totalCountExact: true } };

describe("WorkerPreview", () => {
  function drawWorker() {
    return draw(
      <WorkerPreview
        record={record({ entityType: "worker", id: "wrk_1", title: "Dana Reyes" })}
        onNavigate={vi.fn()}
      />,
    );
  }

  it("says a worker blocked from dispatch is blocked, and why", async () => {
    responses.set("DeskWorkerCard", {
      worker: deskWorker({ assignmentBlocked: "Out on medical leave" }),
    });
    responses.set("WorkerHosState", { workerHosState: null });
    responses.set("DeskRecordShipments", NO_SHIPMENTS);

    drawWorker();

    expect(await screen.findByText("Blocked from dispatch")).toBeInTheDocument();
    expect(screen.getByText("Out on medical leave")).toBeInTheDocument();
  });

  it("draws a ring for each of the driver's four clocks", async () => {
    responses.set("DeskWorkerCard", { worker: deskWorker() });
    responses.set("WorkerHosState", { workerHosState: hosState() });
    responses.set("DeskRecordShipments", NO_SHIPMENTS);

    drawWorker();

    const meters = await screen.findAllByRole("meter");
    expect(meters.map((meter) => meter.getAttribute("aria-label"))).toEqual([
      "Drive time remaining",
      "Shift time remaining",
      "Cycle time remaining",
      "Until a break time remaining",
    ]);
    expect(meters.map((meter) => meter.getAttribute("aria-valuenow"))).toEqual([
      "55",
      "57",
      "57",
      "25",
    ]);
  });

  it("lists a lapsed medical card first among the qualifications", async () => {
    const now = Math.floor(Date.now() / 1000);
    responses.set("DeskWorkerCard", {
      worker: deskWorker({}, { medicalCardExpiry: now - 10 * DAY }),
    });
    responses.set("WorkerHosState", { workerHosState: null });
    responses.set("DeskRecordShipments", NO_SHIPMENTS);

    drawWorker();

    const section = (await screen.findByText("Qualifications")).closest("section")!;
    const labels = within(section)
      .getAllByRole("term")
      .map((term) => term.textContent);
    expect(labels).toEqual(["Medical card expired", "CDL"]);
  });
});

describe("CustomerPreview", () => {
  /** A customer as `CommandPaletteCustomerPreview` returns it. */
  function customer(creditStatus: string) {
    return {
      customer: {
        id: "cus_1",
        status: "Active",
        code: "ACME",
        name: "Acme Foods",
        addressLine1: "1 Main St",
        city: "Dallas",
        postalCode: "75201",
        updatedAt: 1_790_000_000,
        state: { abbreviation: "TX" },
        billingProfile: {
          paymentTerm: "Net30",
          creditStatus,
          creditLimit: "5000",
          creditBalance: "1000",
          invoiceDelivery: "Email",
          billingCycle: "Immediate",
        },
        emailProfile: { toRecipients: "ap@acme.test" },
      },
    };
  }

  /** Receivables as `ArCustomerProfile` returns them: 4,500 open against a 5,000 limit. */
  const RECEIVABLES = {
    arCustomerProfile: {
      creditUtilization: 0.9,
      snapshot: {
        totalOpenMinor: 450_000,
        overdueMinor: 50_000,
        hasCreditLimit: true,
        creditLimitMinor: 500_000,
        buckets: {
          currentMinor: 400_000,
          days1To30Minor: 50_000,
          days31To60Minor: 0,
          days61To90Minor: 0,
          daysOver90Minor: 0,
          totalOpenMinor: 450_000,
        },
      },
    },
  };

  function drawCustomer() {
    return draw(
      <CustomerPreview
        record={record({ entityType: "customer", id: "cus_1", title: "Acme Foods" })}
        onNavigate={vi.fn()}
      />,
    );
  }

  it("says a customer on credit hold is on hold", async () => {
    responses.set("CommandPaletteCustomerPreview", customer("Hold"));
    responses.set("ArCustomerProfile", RECEIVABLES);
    responses.set("DeskRecordShipments", NO_SHIPMENTS);

    drawCustomer();

    expect(await screen.findByText("On credit hold")).toBeInTheDocument();
  });

  it("reads credit used from what is open against the limit", async () => {
    responses.set("CommandPaletteCustomerPreview", customer("Active"));
    responses.set("ArCustomerProfile", RECEIVABLES);
    responses.set("DeskRecordShipments", NO_SHIPMENTS);

    drawCustomer();

    const label = await screen.findByText("Credit used");
    expect(label.parentElement).toHaveTextContent("Credit used90%");
  });

  it("falls back to the billing profile's balance when receivables cannot be read", async () => {
    responses.set("CommandPaletteCustomerPreview", customer("Active"));
    responses.set("DeskRecordShipments", NO_SHIPMENTS);

    drawCustomer();

    const label = await screen.findByText("Credit used");
    expect(label.parentElement).toHaveTextContent("Credit used20%");
  });
});

function stop(index: number, fields: Partial<JourneyStop> = {}): JourneyStop {
  return {
    id: `stp_${index}`,
    moveId: "mv_1",
    moveIndex: 0,
    index,
    type: index === 0 ? "Pickup" : "Delivery",
    place: { name: `Dock ${index + 1}`, city: "Dallas", state: "TX", label: "Dallas, TX" },
    appointment: false,
    windowStart: null,
    windowEnd: null,
    arrivedAt: null,
    departedAt: null,
    phase: "later",
    punctuality: null,
    dwellMinutes: null,
    pieces: null,
    weight: null,
    ...fields,
  };
}

describe("StopList", () => {
  it("lists the first five stops in order and counts the rest", () => {
    render(<StopList stops={Array.from({ length: 7 }, (_, index) => stop(index))} />);

    const items = screen.getAllByRole("listitem");
    expect(items.map((item) => within(item).queryByText(/^Dock /)?.textContent ?? null)).toEqual([
      "Dock 1",
      "Dock 2",
      "Dock 3",
      "Dock 4",
      "Dock 5",
      null,
    ]);
    expect(items.at(-1)).toHaveTextContent("2 more stops");
  });

  it("names where a stop is when its name is not the place", () => {
    render(<StopList stops={[stop(0)]} />);

    expect(screen.getByText("Dallas, TX")).toBeInTheDocument();
    expect(screen.queryByText(/more stop/)).not.toBeInTheDocument();
  });

  it("says how late a stop was", () => {
    render(
      <StopList
        stops={[stop(0, { phase: "done", punctuality: { state: "late", minutes: 45 } })]}
      />,
    );

    expect(screen.getByText(/^Late /)).toBeInTheDocument();
  });
});
