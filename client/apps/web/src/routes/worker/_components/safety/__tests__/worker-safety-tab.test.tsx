import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerSafetyTab from "../../worker-safety-tab";

const {
  fetchWorkerSafetyScorecard,
  fetchWorkerSafetyEvents,
  fetchWorkerDisciplinaryActions,
  fetchWorkerDisciplinaryLadder,
  fetchWorkerRecognitions,
  closeWorkerSafetyEvent,
  toast,
} = vi.hoisted(() => ({
  fetchWorkerSafetyScorecard: vi.fn(),
  fetchWorkerSafetyEvents: vi.fn(),
  fetchWorkerDisciplinaryActions: vi.fn(),
  fetchWorkerDisciplinaryLadder: vi.fn(),
  fetchWorkerRecognitions: vi.fn(),
  closeWorkerSafetyEvent: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

vi.mock("@/lib/graphql/worker-safety", () => ({
  fetchWorkerSafetyScorecard,
  fetchWorkerSafetyEvents,
  fetchWorkerDisciplinaryActions,
  fetchWorkerDisciplinaryLadder,
  fetchWorkerRecognitions,
  closeWorkerSafetyEvent,
  WORKER_SAFETY_EVENTS_KEY: "worker-safety-events",
  WORKER_SAFETY_SCORECARD_KEY: "worker-safety-scorecard",
  WORKER_DISCIPLINARY_ACTIONS_KEY: "worker-disciplinary-actions",
  WORKER_DISCIPLINARY_LADDER_KEY: "worker-disciplinary-ladder",
  WORKER_RECOGNITIONS_KEY: "worker-recognitions",
}));

vi.mock("sonner", () => ({ toast }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("../safety-event-dialog", () => ({
  SafetyEventDialog: ({ open, event }: { open: boolean; event?: { id: string } | null }) =>
    open ? <div data-testid="event-dialog">{event?.id ?? "new"}</div> : null,
}));
vi.mock("../close-event-dialog", () => ({
  CloseEventDialog: ({ open, event }: { open: boolean; event?: { id: string } | null }) =>
    open ? <div data-testid="close-dialog">{event?.id}</div> : null,
}));
vi.mock("../issue-action-dialog", () => ({
  IssueActionDialog: ({ open, suggestedLevel }: { open: boolean; suggestedLevel?: string }) =>
    open ? <div data-testid="action-dialog">{suggestedLevel}</div> : null,
}));
vi.mock("../recognition-dialog", () => ({
  RecognitionDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="recognition-dialog" /> : null,
}));

const scorecard = {
  workerId: "wrk_1",
  asOf: 1_800_000_000,
  score: 55,
  rating: "Watch",
  activePoints: 7,
  pointsWatchThreshold: 6,
  pointsAtRiskThreshold: 10,
  accidents: 1,
  preventableAccidents: 1,
  incidents: 0,
  nearMisses: 1,
  citations: 1,
  inspections: 4,
  inspectionsPassed: 3,
  inspectionsFailed: 1,
  outOfServiceOrders: 1,
  cleanInspectionRate: 0.75,
  openEvents: 1,
  activeDiscipline: 1,
  highestDiscipline: "WrittenWarning",
  daysSinceLastEvent: 30,
  lastEventAt: 1_797_000_000,
  recognitions: 2,
};

const openEvent = {
  id: "wsev_open",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  workerId: "wrk_1",
  kind: "Inspection",
  severity: "Major",
  status: "Open",
  occurredAt: 1_799_000_000,
  location: "I-80 WB",
  description: "Brake adjustment out of service",
  preventable: false,
  points: 5,
  pointsExpireAt: 1_860_000_000,
  activePoints: 5,
  referenceNumber: null,
  shipmentId: null,
  inspectionLevel: 1,
  inspectionResult: "OutOfService",
  outOfService: true,
  fineAmount: null,
  costAmount: null,
  documentId: null,
  document: null,
  recordedById: "usr_1",
  recordedBy: { id: "usr_1", name: "Dana" },
  closedById: null,
  closedBy: null,
  closedAt: null,
  resolution: null,
  version: 1,
  createdAt: 1,
  updatedAt: 1,
};

const closedEvent = {
  ...openEvent,
  id: "wsev_closed",
  kind: "Accident",
  severity: "Moderate",
  status: "Closed",
  preventable: true,
  inspectionLevel: null,
  inspectionResult: null,
  outOfService: false,
  points: 6,
  activePoints: 6,
  description: "Backed into a dock door",
  closedAt: 1_799_500_000,
  closedBy: { id: "usr_1", name: "Dana" },
  resolution: "Coached on backing",
};

const action = {
  id: "wdac_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  workerId: "wrk_1",
  level: "WrittenWarning",
  status: "Active",
  reason: "Out-of-service brake adjustment",
  details: null,
  occurredAt: null,
  issuedAt: 1_799_100_000,
  expiresAt: 1_830_000_000,
  suspensionDays: null,
  safetyEventId: "wsev_open",
  safetyEvent: null,
  documentId: null,
  issuedById: "usr_1",
  issuedBy: { id: "usr_1", name: "Dana" },
  acknowledgedAt: null,
  workerComment: null,
  rescindedAt: null,
  rescindedById: null,
  rescindReason: null,
  active: true,
  version: 1,
  createdAt: 1,
  updatedAt: 1,
};

const ladder = {
  highestLevel: "WrittenWarning",
  suggestedLevel: "FinalWarning",
  atFinalStep: false,
  activeActions: [
    {
      id: "wdac_1",
      level: "WrittenWarning",
      status: "Active",
      reason: "Out-of-service brake adjustment",
      issuedAt: 1_799_100_000,
      expiresAt: 1_830_000_000,
      active: true,
    },
  ],
};

const recognition = {
  id: "wrec_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  workerId: "wrk_1",
  kind: "CustomerPraise",
  title: "Praised by the receiver",
  message: "Smoothest delivery all week",
  occurredAt: 1_799_800_000,
  awardedById: "usr_1",
  awardedBy: { id: "usr_1", name: "Dana" },
  visibleToWorker: true,
  version: 1,
  createdAt: 1,
  updatedAt: 1,
};

function renderTab() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <WorkerSafetyTab workerId="wrk_1" />
    </QueryClientProvider>,
  );
}

function mockAll() {
  fetchWorkerSafetyScorecard.mockResolvedValue(scorecard);
  fetchWorkerSafetyEvents.mockResolvedValue([openEvent, closedEvent]);
  fetchWorkerDisciplinaryActions.mockResolvedValue([action]);
  fetchWorkerDisciplinaryLadder.mockResolvedValue(ladder);
  fetchWorkerRecognitions.mockResolvedValue([recognition]);
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkerSafetyTab", () => {
  it("shows the scorecard with the rating, points against the threshold, and inspection record", async () => {
    mockAll();
    renderTab();

    expect(await screen.findByText("55")).toBeInTheDocument();
    expect(screen.getByText("Watch")).toBeInTheDocument();
    const card = within(screen.getByTestId("safety-scorecard"));
    expect(card.getByText("7")).toBeInTheDocument();
    expect(card.getByText(/of 10 before at-risk/)).toBeInTheDocument();
    expect(card.getByText("3 of 4 clean (75%)")).toBeInTheDocument();
    expect(card.getByText(/30 days/)).toBeInTheDocument();
  });

  it("lists open events before closed ones and describes each in a sentence", async () => {
    mockAll();
    renderTab();

    const rows = await screen.findAllByTestId(/^safety-event-/);
    expect(rows.map((row) => row.getAttribute("data-testid"))).toEqual([
      "safety-event-wsev_open",
      "safety-event-wsev_closed",
    ]);
    expect(within(rows[0]).getByText("Level 1 inspection — out of service")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Moderate accident, preventable")).toBeInTheDocument();
    expect(within(rows[1]).getByText(/Coached on backing/)).toBeInTheDocument();
  });

  it("opens the close dialog for an open event and hides it for a closed one", async () => {
    mockAll();
    renderTab();

    const open = within(await screen.findByTestId("safety-event-wsev_open"));
    fireEvent.click(open.getByRole("button", { name: "Close event" }));
    expect(screen.getByTestId("close-dialog")).toHaveTextContent("wsev_open");

    const closed = within(screen.getByTestId("safety-event-wsev_closed"));
    expect(closed.queryByRole("button", { name: "Close event" })).not.toBeInTheDocument();
  });

  it("shows the ladder with the next rung and opens the issue dialog pre-set to it", async () => {
    mockAll();
    renderTab();

    expect(await screen.findByText(/Next step: Final warning/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Issue action" }));
    expect(screen.getByTestId("action-dialog")).toHaveTextContent("FinalWarning");
  });

  it("lists recognition and lets the office add more", async () => {
    mockAll();
    renderTab();

    expect(await screen.findByText("Praised by the receiver")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Add recognition" }));
    expect(screen.getByTestId("recognition-dialog")).toBeInTheDocument();
  });

  it("warns when the worker has no record at all", async () => {
    fetchWorkerSafetyScorecard.mockResolvedValue({
      ...scorecard,
      score: 100,
      rating: "Excellent",
      activePoints: 0,
      accidents: 0,
      preventableAccidents: 0,
      citations: 0,
      inspections: 0,
      inspectionsPassed: 0,
      inspectionsFailed: 0,
      outOfServiceOrders: 0,
      cleanInspectionRate: null,
      openEvents: 0,
      activeDiscipline: 0,
      highestDiscipline: null,
      daysSinceLastEvent: null,
      lastEventAt: null,
      recognitions: 0,
    });
    fetchWorkerSafetyEvents.mockResolvedValue([]);
    fetchWorkerDisciplinaryActions.mockResolvedValue([]);
    fetchWorkerDisciplinaryLadder.mockResolvedValue({
      highestLevel: null,
      suggestedLevel: "Coaching",
      atFinalStep: false,
      activeActions: [],
    });
    fetchWorkerRecognitions.mockResolvedValue([]);
    renderTab();

    expect(await screen.findByText("Excellent")).toBeInTheDocument();
    expect(screen.getByText("No inspections in the last year")).toBeInTheDocument();
    expect(screen.getByText("Nothing on record")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText(/Next step: Coaching/)).toBeInTheDocument());
  });
});
