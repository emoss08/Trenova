import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import { stubLayout } from "@/test/layout";
import { CLOSED_WORKER_OPEN_STATE, type WorkerOpenState } from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import WorkerSafetyTab from "../worker-safety-tab";

const loaded = vi.hoisted(() => ({ issuePane: false, recordPane: false }));

vi.mock("../issue-action-pane", async (importOriginal) => {
  loaded.issuePane = true;
  return importOriginal();
});

vi.mock("../record-safety-event-pane", async (importOriginal) => {
  loaded.recordPane = true;
  return importOriginal();
});

const mocks = vi.hoisted(() => ({
  fetchWorkerSafetyScorecard: vi.fn(),
  fetchWorkerSafetyEvents: vi.fn(),
  fetchWorkerDisciplinaryActions: vi.fn(),
  fetchWorkerDisciplinaryLadder: vi.fn(),
  fetchWorkerRecognitions: vi.fn(),
  deleteWorkerSafetyEvent: vi.fn(),
  reviewWorkerSafetyEvent: vi.fn(),
  fetchWorkerInjuries: vi.fn(),
  fetchWorkerSafetyEvent: vi.fn(),
  fetchWorkerDisciplinaryAction: vi.fn(),
  fetchWorkerInjury: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

vi.mock("@/lib/graphql/worker-safety", () => ({
  fetchWorkerSafetyScorecard: mocks.fetchWorkerSafetyScorecard,
  fetchWorkerSafetyEvents: mocks.fetchWorkerSafetyEvents,
  fetchWorkerDisciplinaryActions: mocks.fetchWorkerDisciplinaryActions,
  fetchWorkerDisciplinaryLadder: mocks.fetchWorkerDisciplinaryLadder,
  fetchWorkerRecognitions: mocks.fetchWorkerRecognitions,
  fetchWorkerSafetyEvent: mocks.fetchWorkerSafetyEvent,
  fetchWorkerDisciplinaryAction: mocks.fetchWorkerDisciplinaryAction,
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
  fetchWorkerInjury: mocks.fetchWorkerInjury,
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

/** One page of a connection, the last one. */
function page<Row>(rows: Row[]) {
  return {
    edges: rows.map((node) => ({ cursor: `c_${String((node as { id: string }).id)}`, node })),
    pageInfo: { hasNextPage: false, endCursor: null },
  };
}

function mockAll() {
  mocks.fetchWorkerSafetyScorecard.mockResolvedValue(scorecard);
  mocks.fetchWorkerSafetyEvents.mockResolvedValue([closedIncident, openInspection]);
  mocks.fetchWorkerDisciplinaryActions.mockResolvedValue(page([action]));
  mocks.fetchWorkerDisciplinaryLadder.mockResolvedValue(ladder);
  mocks.fetchWorkerRecognitions.mockResolvedValue([]);
  mocks.fetchWorkerInjuries.mockResolvedValue(page([]));
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

// The virtual lists measure their region; jsdom lays nothing out, so give every box a size.
let restoreLayout: () => void = () => undefined;
beforeEach(() => {
  restoreLayout = stubLayout(320);
});
afterEach(() => restoreLayout());

describe("WorkerSafetyTab", () => {
  // First in the file: a pane module, once read, stays read for the rest of it.
  it("reads a form pane's code only when it is first opened", async () => {
    mockAll();
    renderTab();
    await screen.findByTestId("safety-scorecard");
    expect(loaded.issuePane).toBe(false);
    expect(loaded.recordPane).toBe(false);

    cleanup();
    renderTab({ form: "issue-action" });
    // The pane's code is read when it first opens; the first read transforms it.
    expect(
      await screen.findByRole("button", { name: "Issue written warning" }, { timeout: 5000 }),
    ).toBeTruthy();
    expect(loaded.issuePane).toBe(true);
    expect(loaded.recordPane).toBe(false);
  }, 10_000);

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
    // The pane's code is read when it first opens; the first read transforms it.
    expect(await within(pane).findByText("IL4417002931", {}, { timeout: 25000 })).toBeTruthy();
    await user.click(within(pane).getByRole("button", { name: "Close event" }));
    expect(screen.getByTestId("close-dialog")).toHaveTextContent("wsev_open");
  }, 30_000);

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

  it("previews the suggested step on the stair while the issue pane is open", async () => {
    mockAll();
    renderTab({ form: "issue-action" });

    await screen.findByRole("button", { name: "Issue written warning" });
    const stair = screen.getByTestId("discipline-stair");
    expect(stair.querySelector('[data-level="WrittenWarning"]')?.getAttribute("data-state")).toBe(
      "preview",
    );
  });

  it("takes the open count from the scorecard, not from the events read", async () => {
    mockAll();
    mocks.fetchWorkerSafetyScorecard.mockResolvedValue({ ...scorecard, openEvents: 3 });
    renderTab();

    expect(await screen.findByRole("radio", { name: /Open 3/ })).toBeTruthy();
    expect(screen.getByRole("radio", { name: /All 2/ })).toBeTruthy();
    expect(screen.queryByText(/Showing the newest/)).toBeNull();
  });

  it("says a full list is the newest hundred and gives no total it cannot know", async () => {
    mockAll();
    mocks.fetchWorkerSafetyEvents.mockResolvedValue(
      Array.from({ length: 100 }, (_, index) => ({
        ...closedIncident,
        id: `wsev_${index}`,
        occurredAt: 1_790_000_000 - index,
      })),
    );
    renderTab();

    expect(await screen.findByText("Showing the newest 100")).toBeTruthy();
    expect(screen.getByRole("radio", { name: "All" })).toBeTruthy();
  });

  it("asks the server for the open events once the whole list came back full", async () => {
    mockAll();
    const olderOpen = { ...openInspection, id: "wsev_older_open", occurredAt: 1_700_000_000 };
    mocks.fetchWorkerSafetyEvents.mockImplementation(
      (_workerId: string, filters: { openOnly?: boolean }) =>
        Promise.resolve(
          filters.openOnly
            ? [olderOpen]
            : Array.from({ length: 100 }, (_, index) => ({
                ...closedIncident,
                id: `wsev_${index}`,
                occurredAt: 1_790_000_000 - index,
              })),
        ),
    );
    renderTab();
    const user = userEvent.setup();

    await screen.findByText("Showing the newest 100");
    expect(mocks.fetchWorkerSafetyEvents).toHaveBeenCalledTimes(1);
    await user.click(screen.getByRole("radio", { name: /Open/ }));
    expect(await screen.findByTestId("safety-event-wsev_older_open")).toBeTruthy();
    expect(mocks.fetchWorkerSafetyEvents).toHaveBeenCalledWith(
      "wrk_1",
      { openOnly: true },
      expect.anything(),
    );
  });

  it("filters the open events itself while the whole list fits under the cap", async () => {
    mockAll();
    renderTab();
    const user = userEvent.setup();

    await screen.findByTestId("safety-event-wsev_closed");
    await user.click(screen.getByRole("radio", { name: /Open/ }));
    expect(screen.queryByTestId("safety-event-wsev_closed")).toBeNull();
    expect(mocks.fetchWorkerSafetyEvents).toHaveBeenCalledTimes(1);
  });

  it("reads an event the address names by id when it is past the newest 100", async () => {
    mockAll();
    const past = { ...openInspection, id: "wsev_past" };
    mocks.fetchWorkerSafetyEvent.mockResolvedValue(past);
    renderTab({ safetyEvent: "wsev_past" });

    expect(
      await screen.findByRole("dialog", { name: "Level 2 inspection — failed" }, { timeout: 5000 }),
    ).toBeTruthy();
    expect(mocks.fetchWorkerSafetyEvent).toHaveBeenCalledWith("wsev_past", expect.anything());
  });

  it("lists the disciplinary actions from the paged read", async () => {
    mockAll();
    renderTab();

    expect(await screen.findByTestId("discipline-wdac_1")).toBeTruthy();
    expect(mocks.fetchWorkerDisciplinaryActions).toHaveBeenCalledWith(
      expect.objectContaining({ workerId: "wrk_1", first: 50, after: null }),
      expect.anything(),
    );
  });

  it("says the record could not be loaded when a read fails", async () => {
    mockAll();
    mocks.fetchWorkerSafetyEvents.mockRejectedValue(new Error("boom"));
    renderTab();

    expect(await screen.findByText("The safety record could not be loaded")).toBeTruthy();
  });
});
