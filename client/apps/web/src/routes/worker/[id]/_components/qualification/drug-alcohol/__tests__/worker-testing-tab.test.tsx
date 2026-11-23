import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import {
  CLOSED_WORKER_OPEN_STATE,
  type WorkerOpenState,
  type WorkerTarget,
} from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerTestingTab from "../worker-testing-tab";

const { fetchWorkerDrugAlcoholFile, cancelDotTest } = vi.hoisted(() => ({
  fetchWorkerDrugAlcoholFile: vi.fn(),
  cancelDotTest: vi.fn(),
}));
const paneProps = vi.hoisted(() => ({ byName: {} as Record<string, Record<string, unknown>> }));

vi.mock("@/lib/graphql/worker-drug-alcohol", () => ({
  fetchWorkerDrugAlcoholFile,
  cancelDotTest,
  WORKER_DRUG_ALCOHOL_KEY: "worker-drug-alcohol",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

function capture(name: string) {
  return (props: Record<string, unknown>) => {
    paneProps.byName[name] = props;
    return props.open ? <div data-testid={`pane-${name}`} /> : null;
  };
}

vi.mock("../record-collection-pane", () => ({ RecordCollectionPane: capture("collection") }));
vi.mock("../record-result-pane", () => ({ RecordResultPane: capture("result") }));
vi.mock("../clearinghouse-panes", () => ({
  LogQueryPane: capture("log-query"),
  AnswerQueryPane: capture("answer-query"),
}));
vi.mock("../violation-progress-dialog", () => ({ ViolationProgressDialog: capture("violation") }));

const DAY = 86_400;
const now = Math.floor(Date.now() / 1000);

const test = (id: string, overrides: Record<string, unknown> = {}) => ({
  id,
  workerId: "wrk_1",
  testType: "Random",
  substance: "Drug",
  status: "Completed",
  result: "Negative",
  isDot: true,
  reason: null,
  scheduledAt: null,
  collectedAt: now - 90 * DAY,
  resultAt: null,
  collectionSite: null,
  collectorName: null,
  specimenId: null,
  labName: null,
  mroName: null,
  mroVerifiedAt: null,
  alcoholConcentration: null,
  safetyEventId: null,
  drawEntryId: null,
  documentId: null,
  notes: null,
  version: 1,
  createdAt: now - 90 * DAY,
  updatedAt: now - 90 * DAY,
  ...overrides,
});

const query = (id: string, overrides: Record<string, unknown> = {}) => ({
  id,
  workerId: "wrk_1",
  queryType: "AnnualLimited",
  result: "NoViolations",
  consentObtainedAt: null,
  consentExpiresAt: null,
  requestedAt: now - 348 * DAY,
  completedAt: now - 348 * DAY,
  violationCount: 0,
  reference: "CHQ-25-71904",
  documentId: null,
  notes: null,
  version: 1,
  createdAt: 1,
  updatedAt: 1,
  ...overrides,
});

const clearFile = {
  standing: {
    status: "Clear",
    returnToDuty: "NotRequired",
    lastClearinghouseQueryAt: now - 348 * DAY,
    nextClearinghouseQueryDue: now + 17 * DAY,
    hasPreEmploymentTest: true,
    hasPreEmploymentQuery: true,
    openTestCount: 0,
  },
  tests: [
    test("t_done"),
    test("t_open", { status: "Collected", result: "Pending", collectedAt: now - 2 * DAY }),
    test("t_void", { status: "Cancelled", result: "Cancelled", collectedAt: now - 30 * DAY }),
  ],
  violations: [],
  queries: [query("q_done"), query("q_pending", { result: "Pending", completedAt: null })],
  selections: [],
};

function renderTab(file: unknown = clearFile, open: Partial<WorkerOpenState> = {}) {
  fetchWorkerDrugAlcoholFile.mockResolvedValue(file);
  const onOpen = vi.fn<(target: WorkerTarget) => void>();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <WorkPaneProvider>
        <WorkPaneHost>
          <WorkerTestingTab
            workerId="wrk_1"
            open={{ ...CLOSED_WORKER_OPEN_STATE, ...open }}
            onOpen={onOpen}
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
  paneProps.byName = {};
});

describe("WorkerTestingTab", () => {
  it("draws the standing band with when the next query falls due", async () => {
    renderTab();

    const band = await screen.findByRole("region", { name: "Drug and alcohol standing" });
    expect(within(band).getByText("Clear")).toBeInTheDocument();
    expect(within(band).getByText("No open tests, no violations on record.")).toBeInTheDocument();
    expect(within(band).getByTestId("next-query-due")).toHaveTextContent("in 17 days");
  });

  it("lists open tests first and keeps a voided one struck through", async () => {
    renderTab();

    const table = await screen.findByRole("table", { name: "Tests" });
    const rows = within(table)
      .getAllByRole("row")
      .slice(1)
      .map((row) => row.getAttribute("data-testid"));
    expect(rows).toEqual(["dot-test-t_open", "dot-test-t_void", "dot-test-t_done"]);
    expect(screen.getByTestId("dot-test-t_void")).toHaveAttribute("data-voided", "true");
  });

  it("asks why before voiding a test and sends the reason", async () => {
    cancelDotTest.mockResolvedValue(test("t_open", { status: "Cancelled", result: "Cancelled" }));
    renderTab();

    const user = userEvent.setup();
    const row = await screen.findByTestId("dot-test-t_open");
    await user.click(within(row).getByRole("button", { name: /Actions for Random/ }));
    await user.click(await screen.findByRole("menuitem", { name: "Void test" }));
    const dialog = await screen.findByRole("dialog", { name: "Void this test?" });
    expect(cancelDotTest).not.toHaveBeenCalled();

    fireEvent.change(within(dialog).getByRole("textbox"), {
      target: { value: "Recorded against the wrong driver" },
    });
    fireEvent.click(within(dialog).getByRole("button", { name: "Void test" }));
    await waitFor(() =>
      expect(cancelDotTest).toHaveBeenCalledExactlyOnceWith(
        "t_open",
        "Recorded against the wrong driver",
      ),
    );
  });

  it("offers no void or result on a test that is already settled", async () => {
    renderTab();

    const row = await screen.findByTestId("dot-test-t_done");
    expect(within(row).queryByRole("button", { name: /Actions for/ })).toBeNull();
  });

  it("opens the result pane from the address for the open test it names", async () => {
    renderTab(clearFile, { form: "record-test-result", test: "t_open" });

    expect(await screen.findByTestId("pane-result")).toBeInTheDocument();
    expect(paneProps.byName.result).toMatchObject({
      open: true,
      test: expect.objectContaining({ id: "t_open" }),
    });
  });

  it("opens the answer pane from a pending query's menu through the address", async () => {
    const onOpen = renderTab();

    const user = userEvent.setup();
    const row = await screen.findByTestId("clearinghouse-query-q_pending");
    await user.click(within(row).getByRole("button", { name: /Actions for/ }));
    await user.click(await screen.findByRole("menuitem", { name: "Record answer" }));
    expect(onOpen).toHaveBeenCalledWith({
      tab: "qualification",
      view: "drug-and-alcohol",
      form: "answer-query",
      query: "q_pending",
    });
  });

  it("draws return to duty and random selections only when there is something to show", async () => {
    renderTab();
    await screen.findByRole("table", { name: "Tests" });
    expect(screen.queryByRole("region", { name: "Return to duty" })).toBeNull();
    expect(screen.queryByRole("region", { name: "Random selections outstanding" })).toBeNull();
    cleanup();

    renderTab({
      ...clearFile,
      standing: { ...clearFile.standing, status: "Prohibited", returnToDuty: "SAPEvaluation" },
      violations: [
        {
          id: "v_1",
          workerId: "wrk_1",
          violationType: "PositiveTest",
          status: "SAPEvaluation",
          occurredAt: now - 10 * DAY,
          sourceTestId: "t_done",
          reportedToClearinghouseAt: null,
          sapName: null,
          sapReferredAt: null,
          sapEvaluationCompletedAt: null,
          rtdTestId: null,
          rtdCompletedAt: null,
          followUpTestCount: 6,
          followUpTestsCompleted: 0,
          followUpEndsAt: null,
          resolvedAt: null,
          documentId: null,
          notes: null,
          version: 1,
          createdAt: 1,
          updatedAt: 1,
        },
      ],
      selections: [
        {
          id: "de_1",
          drawId: "d_1",
          workerId: "wrk_1",
          substance: "Alcohol",
          rank: 1,
          status: "Notified",
          notifiedAt: null,
          completedAt: null,
          testId: null,
          excuseReason: null,
          version: 1,
          worker: null,
        },
      ],
    });
    expect(await screen.findByRole("region", { name: "Return to duty" })).toBeInTheDocument();
    const selections = screen.getByRole("region", { name: "Random selections outstanding" });
    expect(within(selections).getByText("Alcohol")).toBeInTheDocument();
  });

  it("presets the collection pane from the random selection the address names", async () => {
    renderTab(
      {
        ...clearFile,
        selections: [
          {
            id: "de_1",
            drawId: "d_1",
            workerId: "wrk_1",
            substance: "Alcohol",
            rank: 1,
            status: "Selected",
            notifiedAt: null,
            completedAt: null,
            testId: null,
            excuseReason: null,
            version: 1,
            worker: null,
          },
        ],
      },
      { form: "record-collection", kind: "de_1" },
    );

    expect(await screen.findByTestId("pane-collection")).toBeInTheDocument();
    expect(paneProps.byName.collection).toMatchObject({
      drawEntryId: "de_1",
      defaultSubstance: "Alcohol",
    });
  });
});
