import type { RecordShipment } from "@/lib/graphql/record-shipments";
import type {
  DeskWorker as DeskWorkerRecord,
  DeskWorkerCredential,
  DeskWorkerHos,
  DeskWorkerStanding,
} from "@/lib/graphql/worker-card";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskWorker, DeskWorkerArtifact } from "../desk-worker";

const fetchers = vi.hoisted(() => ({
  worker: vi.fn(),
  hos: vi.fn(),
  standing: vi.fn(),
  credentials: vi.fn(),
  shipments: vi.fn(),
}));

vi.mock("@/lib/graphql/worker-card", () => ({
  fetchDeskWorker: fetchers.worker,
  fetchWorkerHos: fetchers.hos,
  fetchWorkerStanding: fetchers.standing,
  fetchWorkerCredentials: fetchers.credentials,
}));
vi.mock("@/lib/graphql/record-shipments", () => ({ fetchRecordShipments: fetchers.shipments }));
// The shipment and tractor cards are their own chunks; here each stands in as
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
vi.mock("../../tractor/desk-tractor", () => ({
  DeskTractor: standIn("Tractor"),
  DeskTractorArtifact: () => null,
}));

const DAY = 86_400;
const HOUR_MS = 3_600_000;
const NOW = Math.floor(Date.now() / 1000);

/** A worker as the DeskWorkerCard query returns them. */
function worker(overrides: Partial<DeskWorkerRecord> = {}): DeskWorkerRecord {
  return {
    id: "wrk_1",
    status: "Active",
    type: "Employee",
    driverType: "OTR",
    firstName: "Ana",
    lastName: "Ruiz",
    phoneNumber: "214-555-0100",
    email: "ana@example.com",
    city: "Dallas",
    externalId: "",
    assignmentBlocked: "",
    canBeAssigned: true,
    availableForDispatch: true,
    createdAt: 1_700_000_000,
    state: { abbreviation: "TX" },
    fleetCode: { id: "fc_dal", code: "DAL" },
    manager: { id: "usr_1", name: "Mia Chen" },
    profile: {
      licenseNumber: "D1234567",
      cdlClass: "A",
      cdlRestrictions: "",
      endorsement: "H",
      licenseExpiry: NOW + 400 * DAY,
      hazmatExpiry: NOW + 300 * DAY,
      medicalCardExpiry: NOW + 300 * DAY,
      twicExpiry: null,
      physicalDueDate: null,
      mvrDueDate: null,
      hireDate: NOW - 900 * DAY,
      isQualified: true,
      disqualificationReason: "",
      licenseState: { abbreviation: "TX" },
    },
    ...overrides,
  } as DeskWorkerRecord;
}

function hos(overrides: Partial<NonNullable<DeskWorkerHos["state"]>> = {}): DeskWorkerHos {
  return {
    state: {
      dutyStatus: "driving",
      driveRemainingMs: 6 * HOUR_MS + 15 * 60_000,
      driveLimitMs: 11 * HOUR_MS,
      shiftRemainingMs: 8 * HOUR_MS,
      shiftLimitMs: 14 * HOUR_MS,
      cycleRemainingMs: 40 * HOUR_MS,
      cycleLimitMs: 70 * HOUR_MS,
      breakRemainingMs: 3 * HOUR_MS,
      breakLimitMs: 8 * HOUR_MS,
      currentTractorId: "tr_101",
      rulesetCycle: "USA 70 hour / 8 day",
      recordedAt: NOW - 120,
      ...overrides,
    },
    violations: [],
  } as DeskWorkerHos;
}

const standing: DeskWorkerStanding = {
  standing: "Good",
  concerns: [],
} as DeskWorkerStanding;

const credentials = [
  {
    health: "Valid",
    daysUntilExpiry: 200,
    required: true,
    credentialType: { id: "ct_fork", name: "Forklift", profileField: "" },
    credential: { id: "cr_fork", expiresAt: NOW + 200 * DAY },
  },
] as DeskWorkerCredential[];

const shipment = {
  id: "shp_1",
  proNumber: "S26100090038417",
  status: "Completed",
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

function page(rows: RecordShipment[], total: number) {
  return { rows, total, endCursor: null, hasNextPage: false };
}

function draw(props: Partial<Parameters<typeof DeskWorker>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskWorker recordId="wrk_1" order={["wrk_0", "wrk_1"]} {...handlers} {...props} />
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
    [Resource.Worker]: Operation.Read,
    [Resource.WorkerCredential]: Operation.Read,
    [Resource.Shipment]: Operation.Read,
  });
  fetchers.worker.mockReset().mockResolvedValue(worker());
  fetchers.hos.mockReset().mockResolvedValue(hos());
  fetchers.standing.mockReset().mockResolvedValue(standing);
  fetchers.credentials.mockReset().mockResolvedValue(credentials);
  fetchers.shipments.mockReset().mockResolvedValue(page([shipment], 14));
});

afterEach(() => {
  clearPermissions();
});

describe("DeskWorker", () => {
  it("holds the card's outline while the worker is read, then draws who they are and their day", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the worker…");
    expect(container.querySelectorAll(".dk-rec-skel .dk-wk-ring-sk")).toHaveLength(4);
    expect(container.querySelector(".dk-rec-skel .dk-rec-av")).not.toBeNull();

    expect(await screen.findByRole("heading", { name: "Ana Ruiz", level: 2 })).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(screen.getByText("CDL A · TX · Hazmat (H)")).toBeInTheDocument();
    expect(screen.getByText("Available for dispatch")).toBeInTheDocument();
    expect(screen.getByText("Fleet DAL")).toBeInTheDocument();

    expect(await screen.findByText("drive time left")).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-amt b")).toHaveTextContent("6:15");
    const hours = section("Hours of service");
    expect(within(hours).getByText("Driving")).toBeInTheDocument();
    expect(within(hours).getAllByRole("meter")).toHaveLength(4);
    const drive = within(hours).getByRole("meter", { name: "Drive time remaining" });
    expect(drive).toHaveAttribute("aria-valuenow", "57");
    expect(drive).toHaveTextContent("6:15");
    expect(drive.closest(".dk-wk-ring")).toHaveTextContent("of 11:00");
    expect(drive.querySelector("circle:last-of-type")).toHaveClass("text-brand");
    expect(within(hours).getByRole("link", { name: /Open tractor/ })).toHaveAttribute(
      "href",
      expect.stringContaining("tr_101"),
    );

    expect(await within(section("Standing")).findByText("In good standing")).toBeInTheDocument();
    expect(within(section("Qualifications")).getByText("Nothing due")).toBeInTheDocument();
    expect(await within(section("Other credentials")).findByText("Forklift")).toBeInTheDocument();
    expect(await screen.findByText("14 shipments")).toBeInTheDocument();
    expect(fetchers.shipments).toHaveBeenCalledWith(
      { kind: "worker", id: "wrk_1" },
      { first: 5, after: null },
      expect.anything(),
    );
    expect(fetchers.hos).toHaveBeenCalledWith("wrk_1", 3, expect.anything());

    expect(document.querySelector(".dk-rec-hrow .dk-rec-av")).toHaveTextContent("AR");
    const reach = section("Reach and record");
    expect(within(reach).getByRole("link", { name: /214-555-0100/ })).toHaveAttribute(
      "href",
      "tel:2145550100",
    );
    expect(within(reach).getByRole("link", { name: /ana@example.com/ })).toHaveAttribute(
      "href",
      "mailto:ana@example.com",
    );
  });

  it("draws the worker before their hours, standing and credentials are read, each holding its place", async () => {
    fetchers.hos.mockReturnValue(new Promise(() => {}));
    fetchers.standing.mockReturnValue(new Promise(() => {}));
    fetchers.credentials.mockReturnValue(new Promise(() => {}));
    const { container } = draw();

    expect(await screen.findByRole("heading", { name: "Ana Ruiz", level: 2 })).toBeInTheDocument();
    expect(screen.getByText("Drive time")).toBeInTheDocument();
    expect(section("Hours of service").querySelectorAll(".dk-wk-ring-sk")).toHaveLength(4);
    expect(section("Standing").querySelector(".dk-rec-sk-mark")).not.toBeNull();
    expect(section("Other credentials").querySelector(".dk-wk-wallet .dk-sk")).not.toBeNull();
    expect(container.querySelector('[role="meter"]')).toBeNull();
  });

  it("names what keeps the worker off a load, worst first", async () => {
    fetchers.worker.mockResolvedValue(
      worker({
        assignmentBlocked: "Drug test pending",
        profile: {
          ...worker().profile!,
          medicalCardExpiry: NOW - DAY,
          hazmatExpiry: NOW + 10 * DAY,
        },
      }),
    );
    fetchers.hos.mockResolvedValue({
      ...hos({ driveRemainingMs: 0, shiftRemainingMs: 30 * 60_000 }),
      violations: [
        {
          violationType: "SHIFT_DRIVING",
          description: "Drove past the 11-hour limit",
          durationMs: 25 * 60_000,
          violationStartAt: NOW - 2 * DAY,
        },
      ],
    });
    fetchers.standing.mockResolvedValue({
      standing: "Blocked",
      concerns: [
        {
          severity: "Critical",
          code: "not_assignable",
          headline: "Cannot be assigned",
          detail: "",
        },
        {
          severity: "Warning",
          code: "training_due",
          headline: "Hazmat training due",
          detail: "Due within 30 days.",
        },
      ],
    });
    fetchers.credentials.mockResolvedValue([
      {
        health: "Missing",
        daysUntilExpiry: null,
        required: true,
        credentialType: { id: "ct_port", name: "Port pass", profileField: "" },
        credential: null,
      },
    ]);
    draw();

    expect(await screen.findByText("Blocked from dispatch")).toBeInTheDocument();
    expect(screen.getByText("Drug test pending")).toBeInTheDocument();

    const qualifications = section("Qualifications");
    expect(within(qualifications).getByText("2 need a look")).toBeInTheDocument();
    expect(
      Array.from(qualifications.querySelectorAll(".dk-wk-wallet > li > b")).map(
        (b) => b.textContent,
      ),
    ).toEqual(["Medical card expired", "Hazmat endorsement expires in 10 days", "CDL"]);

    const hours = section("Hours of service");
    const drive = await within(hours).findByRole("meter", { name: "Drive time remaining" });
    expect(drive).toHaveAttribute("aria-valuenow", "0");
    expect(drive.querySelector("circle:last-of-type")).toHaveClass("text-destructive");
    expect(within(hours).getByText("Shift Driving")).toBeInTheDocument();
    expect(within(hours).getByText("Drove past the 11-hour limit")).toBeInTheDocument();
    expect(document.querySelector(".dk-rec-amt b")).toHaveClass("dk-bad");

    const standingSection = section("Standing");
    expect(await within(standingSection).findByText("Blocked")).toBeInTheDocument();
    expect(within(standingSection).getByText("Hazmat training due")).toBeInTheDocument();
    expect(within(standingSection).queryByText("Cannot be assigned")).toBeNull();

    expect(await within(section("Other credentials")).findByText("Port pass")).toBeInTheDocument();
    expect(screen.getByText("Required, not on file")).toBeInTheDocument();
  });

  it("opens the tractor they drive and the loads they ran in the card's place", async () => {
    seedPermissions({
      [Resource.Worker]: Operation.Read,
      [Resource.Tractor]: Operation.Read,
      [Resource.Shipment]: Operation.Read,
    });
    draw();

    fireEvent.click(await screen.findByRole("button", { name: /Open tractor/ }));
    expect(await screen.findByText("Tractor card tr_101")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to Ana Ruiz" }));

    fireEvent.click(await screen.findByRole("button", { name: /S26100090038417/ }));
    expect(await screen.findByText("Shipment card shp_1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to Ana Ruiz" }));
    expect(await screen.findByRole("heading", { name: "Ana Ruiz", level: 2 })).toBeInTheDocument();
  });

  it("stands with nothing around them: no profile, no hours, no loads, no other credentials", async () => {
    fetchers.worker.mockResolvedValue(
      worker({ profile: null, fleetCode: null, manager: null, email: "", city: "", state: null }),
    );
    fetchers.hos.mockResolvedValue({ state: null, violations: [] });
    fetchers.credentials.mockResolvedValue([]);
    fetchers.shipments.mockResolvedValue(page([], 0));
    draw();

    expect(await screen.findByRole("heading", { name: "Ana Ruiz", level: 2 })).toBeInTheDocument();
    expect(screen.getByText("No driver profile on file")).toBeInTheDocument();
    expect(await screen.findByText("No hours reported")).toBeInTheDocument();
    expect(
      screen.getByText("Telematics has not reported this worker's hours."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Qualifications" })).toBeNull();
    expect(screen.queryByRole("link", { name: /Open tractor/ })).toBeNull();
    expect(
      await screen.findByText("No credentials beyond the license and medical card."),
    ).toBeInTheDocument();
    expect(await screen.findByText("This worker has not run a load yet.")).toBeInTheDocument();
    expect(await screen.findByText("Nothing else stands against them")).toBeInTheDocument();
    const reach = section("Reach and record");
    expect(within(reach).getByText("No email on file")).toBeInTheDocument();
    expect(within(reach).queryByRole("link", { name: /Email/ })).toBeNull();
  });

  it("says a section that could not be read, and draws the rest", async () => {
    fetchers.hos.mockRejectedValue(new Error("Telematics is not connected"));
    fetchers.standing.mockRejectedValue(new Error("boom"));
    fetchers.credentials.mockRejectedValue(new Error("boom"));
    draw();

    expect(await screen.findByText("Telematics is not connected")).toBeInTheDocument();
    expect(await screen.findAllByText("boom")).toHaveLength(2);
    expect(screen.getByRole("heading", { name: "Qualifications", level: 3 })).toBeInTheDocument();
  });

  it("says the worker could not be read, with the way back", async () => {
    fetchers.worker.mockRejectedValue(new Error("Worker not found"));
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", { name: "This worker could not be read", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText("Worker not found")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
    expect(fetchers.hos).not.toHaveBeenCalled();
  });

  it("opens with only what the person may read, asking for nothing else", async () => {
    seedPermissions({ [Resource.Worker]: Operation.Read });
    draw();

    expect(await screen.findByRole("heading", { name: "Ana Ruiz", level: 2 })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Recent shipments" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Other credentials" })).toBeNull();
    expect(fetchers.shipments).not.toHaveBeenCalled();
    expect(fetchers.credentials).not.toHaveBeenCalled();
  });

  it("leaves the outlines of sections the person may not read out of the skeleton", () => {
    seedPermissions({ [Resource.Worker]: Operation.Read });
    fetchers.worker.mockReturnValue(new Promise(() => {}));
    const { container } = draw();

    expect(container.querySelectorAll(".dk-rec-skel .dk-rec-sec")).toHaveLength(4);
  });

  it("draws the same card as an artifact, with nothing to step through", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskWorkerArtifact artifact={{} as never} recordId="wrk_1" fallback={null} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(await screen.findByRole("heading", { name: "Ana Ruiz", level: 2 })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Next worker" })).toBeNull();
  });
});
