import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import { CLOSED_WORKER_OPEN_STATE, type WorkerOpenState } from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerSafetyTab from "../worker-safety-tab";

const mocks = vi.hoisted(() => ({
  fetchWorkerSafetyScorecard: vi.fn(),
  fetchWorkerSafetyEvents: vi.fn(),
  fetchWorkerDisciplinaryActions: vi.fn(),
  fetchWorkerDisciplinaryLadder: vi.fn(),
  fetchWorkerRecognitions: vi.fn(),
  deleteWorkerSafetyEvent: vi.fn(),
  reviewWorkerSafetyEvent: vi.fn(),
  fetchWorkerInjuries: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

vi.mock("@/lib/graphql/worker-safety", () => ({
  fetchWorkerSafetyScorecard: mocks.fetchWorkerSafetyScorecard,
  fetchWorkerSafetyEvents: mocks.fetchWorkerSafetyEvents,
  fetchWorkerDisciplinaryActions: mocks.fetchWorkerDisciplinaryActions,
  fetchWorkerDisciplinaryLadder: mocks.fetchWorkerDisciplinaryLadder,
  fetchWorkerRecognitions: mocks.fetchWorkerRecognitions,
  deleteWorkerSafetyEvent: mocks.deleteWorkerSafetyEvent,
  reviewWorkerSafetyEvent: mocks.reviewWorkerSafetyEvent,
  reopenWorkerSafetyEvent: vi.fn(),
  deleteWorkerRecognition: vi.fn(),
  fetchDefaultSafetyPoints: vi.fn().mockResolvedValue(3),
  createWorkerSafetyEvent: vi.fn(),
  updateWorkerSafetyEvent: vi.fn(),
  issueDisciplinaryAction: vi.fn(),
  giveWorkerRecognition: vi.fn(),
  WORKER_SAFETY_EVENTS_KEY: "worker-safety-events",
  WORKER_SAFETY_SCORECARD_KEY: "worker-safety-scorecard",
  WORKER_DISCIPLINARY_ACTIONS_KEY: "worker-disciplinary-actions",
  WORKER_DISCIPLINARY_LADDER_KEY: "worker-disciplinary-ladder",
  WORKER_RECOGNITIONS_KEY: "worker-recognitions",
}));

vi.mock("@/lib/graphql/worker-injury", () => ({
  fetchWorkerInjuries: mocks.fetchWorkerInjuries,
  deleteWorkerInjury: vi.fn(),
  recordWorkerInjury: vi.fn(),
  updateWorkerInjury: vi.fn(),
  WORKER_INJURIES_KEY: "worker-injuries",
  OSHA_LOG_KEY: "osha-log",
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({ toast: mocks.toast }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("../close-event-dialog", () => ({
  CloseEventDialog: ({ open, event }: { open: boolean; event?: { id: string } | null }) =>
    open ? <div data-testid="close-dialog">{event?.id}</div> : null,
}));

const scorecard = {
  workerId: "wrk_1",
  asOf: 1_800_000_000,
  score: 74,
  rating: "Watch",
  activePoints: 7,
  pointsWatchThreshold: 6,
  pointsAtRiskThreshold: 10,
  accidents: 0,
  preventableAccidents: 1,
  incidents: 1,
  nearMisses: 0,
  citations: 1,
  inspections: 6,
  inspectionsPassed: 5,
  inspectionsFailed: 1,
  outOfServiceOrders: 0,
  cleanInspectionRate: 0.83,
  openEvents: 1,
  activeDiscipline: 1,
  highestDiscipline: "VerbalWarning",
  daysSinceLastEvent: 41,
  lastEventAt: 1_797_000_000,
  recognitions: 1,
};

const base = {
  businessUnitId: "bu_1",
  organizationId: "org_1",
  workerId: "wrk_1",
  shipmentId: null,
  documentId: null,
  document: null,
  recordedById: "usr_1",
  recordedBy: { id: "usr_1", name: "Dana Whitfield" },
  closedById: null,
  closedBy: null,
  closedAt: null,
  resolution: null,
  fineAmount: null,
  costAmount: null,
  version: 1,
  createdAt: 1,
  updatedAt: 1,
  violations: [],
};

const openInspection = {
  ...base,
  id: "wsev_open",
  kind: "Inspection",
  severity: "Moderate",
  status: "Open",
  occurredAt: 1_799_000_000,
  location: "I-80 eastbound, Morris, IL",
  description: "Walk-around at the Morris scale.",
  preventable: false,
  points: 4,
  pointsExpireAt: 1_860_000_000,
  activePoints: 4,
  referenceNumber: "IL4417002931",
  inspectionLevel: 2,
  inspectionResult: "Fail",
  outOfService: false,
  violations: [
    {
      id: "vio_1",
      safetyEventId: "wsev_open",
      workerId: "wrk_1",
      basic: "UnsafeDriving",
      code: "392.2S",
      description: "Speeding 6 to 10 mph over the limit",
      severityWeight: 4,
      outOfService: false,
      version: 1,
    },
  ],
};

const closedIncident = {
  ...base,
  id: "wsev_closed",
  kind: "Incident",
  severity: "Minor",
  status: "Closed",
  occurredAt: 1_790_000_000,
  location: "Menards DC, Plano, IL",
  description: "Backed into a dock bumper.",
  preventable: true,
  points: 3,
  pointsExpireAt: 1_850_000_000,
  activePoints: 3,
  referenceNumber: null,
  inspectionLevel: null,
  inspectionResult: null,
  outOfService: false,
  costAmount: "640.00",
  resolution: "Coaching and a verbal warning.",
};

const ladder = {
  highestLevel: "VerbalWarning",
  suggestedLevel: "WrittenWarning",
  atFinalStep: false,
  activeActions: [],
};

const action = {
  id: "wdac_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  workerId: "wrk_1",
  level: "VerbalWarning",
  status: "Active",
  reason: "Backed into a dock bumper",
  details: null,
  occurredAt: null,
  issuedAt: 1_790_100_000,
  expiresAt: 1_821_600_000,
  suspensionDays: null,
  safetyEventId: "wsev_closed",
  safetyEvent: null,
  documentId: null,
  issuedById: "usr_1",
  issuedBy: { id: "usr_1", name: "Dana Whitfield" },
  acknowledgedAt: 1_790_200_000,
  workerComment: null,
  rescindedAt: null,
  rescindedById: null,
  rescindReason: null,
  active: true,
  version: 1,
  createdAt: 1,
  updatedAt: 1,
};

function mockAll() {
  mocks.fetchWorkerSafetyScorecard.mockResolvedValue(scorecard);
  mocks.fetchWorkerSafetyEvents.mockResolvedValue([closedIncident, openInspection]);
  mocks.fetchWorkerDisciplinaryActions.mockResolvedValue([action]);
  mocks.fetchWorkerDisciplinaryLadder.mockResolvedValue(ladder);
  mocks.fetchWorkerRecognitions.mockResolvedValue([]);
  mocks.fetchWorkerInjuries.mockResolvedValue([]);
}

function renderTab(open: Partial<WorkerOpenState> = {}) {
  const onOpen = vi.fn();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <WorkPaneProvider>
        <WorkPaneHost>
          <WorkerSafetyTab
            workerId="wrk_1"
            worker={{ id: "wrk_1", firstName: "Maria", lastName: "Alvarez" } as never}
            view={null}
            open={{ ...CLOSED_WORKER_OPEN_STATE, ...open }}
            onOpen={onOpen}
            shown
          />
        </WorkPaneHost>
      </WorkPaneProvider>
    </QueryClientProvider>,
  );
  return onOpen;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkerSafetyTab", () => {
  it("draws the scorecard band: score and rating, the points against the thresholds, the ruled facts", async () => {
    mockAll();
    renderTab();

    const band = within(await screen.findByTestId("safety-scorecard"));
    expect(band.getByText("74")).toBeTruthy();
    expect(band.getAllByText("Watch")).toHaveLength(2);
    expect(band.getByText("Quiet for 41 days")).toBeTruthy();
    expect(
      band.getByRole("img", {
        name: "7 active points on a scale of 0 to 12. Watch from 6, at risk from 10.",
      }),
    ).toBeTruthy();
    expect(band.getByText("5 of 6 clean")).toBeTruthy();
  });

  it("lists open events before closed ones and unfolds an inspection's violations in place", async () => {
    mockAll();
    renderTab();
    const user = userEvent.setup();

    const rows = await screen.findAllByTestId(/^safety-event-/);
    expect(rows.map((row) => row.getAttribute("data-testid"))).toEqual([
      "safety-event-wsev_open",
      "safety-event-wsev_closed",
    ]);
    expect(within(rows[1]).getByText("Resolved: Coaching and a verbal warning.")).toBeTruthy();

    expect(screen.queryByText("Speeding 6 to 10 mph over the limit")).toBeNull();
    await user.click(within(rows[0]).getByRole("button", { name: /Violations cited/ }));
    expect(screen.getByText("Speeding 6 to 10 mph over the limit")).toBeTruthy();
  });

  it("shows only open events under Open", async () => {
    mockAll();
    renderTab();
    const user = userEvent.setup();

    await screen.findByTestId("safety-event-wsev_closed");
    await user.click(screen.getByRole("radio", { name: /Open/ }));
    expect(screen.queryByTestId("safety-event-wsev_closed")).toBeNull();
    expect(screen.getByTestId("safety-event-wsev_open")).toBeTruthy();
  });

  it("opens an event beside the list when its row is chosen", async () => {
    mockAll();
    const onOpen = renderTab();

    await userEvent.setup().click(await screen.findByTestId("safety-event-wsev_open"));
    expect(onOpen).toHaveBeenCalledWith({ tab: "safety", safetyEvent: "wsev_open" });
  });

  it("reads the event in the work pane and closes it through the one-field dialog", async () => {
    mockAll();
    renderTab({ safetyEvent: "wsev_open" });
    const user = userEvent.setup();

    const pane = await screen.findByRole("dialog", { name: "Level 2 inspection — failed" });
    expect(within(pane).getByText("IL4417002931")).toBeTruthy();
    await user.click(within(pane).getByRole("button", { name: "Close event" }));
    expect(screen.getByTestId("close-dialog")).toHaveTextContent("wsev_open");
  });

  it("deletes an event only after the confirmation", async () => {
    mockAll();
    mocks.deleteWorkerSafetyEvent.mockResolvedValue(true);
    renderTab();
    const user = userEvent.setup();

    const row = within(await screen.findByTestId("safety-event-wsev_open"));
    await user.click(row.getByRole("button", { name: /^Actions for/ }));
    await user.click(await screen.findByRole("menuitem", { name: "Delete" }));
    expect(mocks.deleteWorkerSafetyEvent).not.toHaveBeenCalled();

    const dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getByText(/Its 4 points come off the score/)).toBeTruthy();
    await user.click(within(dialog).getByRole("button", { name: "Delete event" }));
    await waitFor(() => expect(mocks.deleteWorkerSafetyEvent).toHaveBeenCalledWith("wsev_open"));
  });

  it("offers no delete for a closed event", async () => {
    mockAll();
    renderTab();
    const user = userEvent.setup();

    const row = within(await screen.findByTestId("safety-event-wsev_closed"));
    await user.click(row.getByRole("button", { name: /^Actions for/ }));
    expect(await screen.findByRole("menuitem", { name: "Reopen event" })).toBeTruthy();
    expect(screen.queryByRole("menuitem", { name: "Delete" })).toBeNull();
  });

  it("draws the stair to the active warning and issues the next step in the work pane", async () => {
    mockAll();
    const onOpen = renderTab();
    const user = userEvent.setup();

    const stair = await screen.findByTestId("discipline-stair");
    expect(
      [...stair.querySelectorAll("[data-level]")].map((step) => step.getAttribute("data-state")),
    ).toEqual(["reached", "active", "next", "rest", "rest", "rest"]);
    expect(screen.getByText(/Next step:/).textContent).toBe("Next step: written warning");

    await user.click(screen.getByRole("button", { name: "Issue action" }));
    expect(onOpen).toHaveBeenCalledWith({ tab: "safety", form: "issue-action" });
  });

  it("says the record could not be loaded when a read fails", async () => {
    mockAll();
    mocks.fetchWorkerSafetyEvents.mockRejectedValue(new Error("boom"));
    renderTab();

    expect(await screen.findByText("The safety record could not be loaded")).toBeTruthy();
  });
});
