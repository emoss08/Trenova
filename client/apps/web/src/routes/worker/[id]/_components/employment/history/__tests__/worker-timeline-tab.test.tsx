import type { WorkerRecord } from "@/routes/worker/use-worker-state";
import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { worker as workerQueries } from "@/lib/queries/worker";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { stubLayout } from "@/test/layout";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import WorkerTimelineTab from "../worker-timeline-tab";

const { fetchWorkerEmploymentEvents, fetchWorkerEmploymentEvent } = vi.hoisted(() => ({
  fetchWorkerEmploymentEvents: vi.fn(),
  fetchWorkerEmploymentEvent: vi.fn(),
}));
const permissionState = vi.hoisted(() => ({ create: true, update: true }));

vi.mock("@/lib/graphql/worker-employment", () => ({
  fetchWorkerEmploymentEvents,
  fetchWorkerEmploymentEvent,
  fetchWorkerEmploymentEventOptions: vi.fn(),
  WORKER_EMPLOYMENT_EVENTS_KEY: "worker-employment-events",
  WORKER_EMPLOYMENT_OPTIONS_KEY: "worker-employment-event-options",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (_resource: string, operation: number) => {
    const map: Record<number, boolean> = {
      [1 << 1]: permissionState.create,
      [1 << 2]: permissionState.update,
    };
    return { allowed: map[operation] ?? true, isLoading: false };
  },
}));

vi.mock("../../../time/leave/leave-standing-strip", () => ({
  LeaveStandingStrip: () => <div data-testid="leave-standing" />,
}));

const event = (
  id: string,
  kind: string,
  effectiveAt: number,
  overrides: Partial<Record<string, unknown>> = {},
): Record<string, unknown> => ({
  id,
  workerId: "wrk_1",
  kind,
  effectiveAt,
  reason: null,
  notes: null,
  fromValues: [],
  toValues: [],
  documentId: null,
  document: null,
  recordedById: "usr_1",
  recordedBy: { id: "usr_1", name: "Ada Lovelace" },
  amendedById: null,
  amendedBy: null,
  amendedAt: null,
  amendmentNote: null,
  version: 0,
  createdAt: 1,
  updatedAt: 1,
  ...overrides,
});

const worker = {
  profile: { hireDate: 1_662_940_800, terminationDate: null },
} as unknown as WorkerRecord;

const handlers = {
  onRecord: vi.fn(),
  onAmend: vi.fn(),
  onCloseAmend: vi.fn(),
  recordPane: { preload: vi.fn(() => Promise.resolve()) },
};

type Row = Record<string, unknown>;

/** The server's read: newest first, narrowed to `kinds`, a page of `first` after `after`. */
function serve(rows: readonly Row[]) {
  fetchWorkerEmploymentEvents.mockImplementation(
    async (input: { kinds?: readonly string[]; first: number; after?: string | null }) => {
      const matching = [...rows]
        .filter((row) => !input.kinds?.length || input.kinds.includes(row.kind as string))
        .sort((a, b) => (b.effectiveAt as number) - (a.effectiveAt as number));
      const start = input.after ? Number(input.after) : 0;
      const page = matching.slice(start, start + input.first);
      const end = start + page.length;
      return {
        edges: page.map((node, index) => ({ cursor: String(start + index + 1), node })),
        pageInfo: { hasNextPage: end < matching.length, endCursor: String(end) },
        // Counted on the first page only, as the operation asks.
        ...(input.after ? {} : { totalCount: matching.length, totalCountExact: true }),
      };
    },
  );
}

function renderTab(
  highlightId: string | null = null,
  eventCount = 3,
  amendingId: string | null = null,
) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(workerQueries.overview("wrk_1").queryKey, {
    employment: { eventCount, hiredAt: null, openChecklistCount: 0 },
  } as never);
  return render(
    <QueryClientProvider client={client}>
      <WorkPaneProvider>
        <WorkPaneHost>
          <WorkerTimelineTab
            workerId="wrk_1"
            worker={worker}
            highlightId={highlightId}
            amendingId={amendingId}
            {...handlers}
          />
        </WorkPaneHost>
      </WorkPaneProvider>
    </QueryClientProvider>,
  );
}

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout(320);
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(1_795_305_600_000);
});

afterEach(() => {
  restoreLayout();
  vi.useRealTimers();
  cleanup();
  vi.clearAllMocks();
  permissionState.create = true;
  permissionState.update = true;
});

const history = [
  event("wee_hire", "Hired", 1_662_940_800, {
    toValues: [
      { key: "status", value: "Active" },
      { key: "fleetCode", value: "CHI-LOCAL" },
      { key: "fleetCodeId", value: "fc_01J9" },
    ],
  }),
  event("wee_transfer", "Transferred", 1_709_510_400, {
    reason: "Requested the move to over the road",
    fromValues: [
      { key: "fleetCode", value: "CHI-LOCAL" },
      { key: "driverType", value: "Local" },
    ],
    toValues: [
      { key: "fleetCode", value: "MW-OTR" },
      { key: "driverType", value: "OTR" },
    ],
  }),
  event("wee_rate", "RateChanged", 1_751_328_000, {
    reason: "Annual increase",
    fromValues: [{ key: "rate", value: "$0.60" }],
    toValues: [{ key: "rate", value: "$0.62" }],
    amendedAt: 1_751_500_000,
    amendedBy: { id: "usr_2", name: "Priya Raman" },
    amendmentNote: "effective date corrected",
  }),
];

describe("Employment history", () => {
  it("groups events by year, newest first, with changes as chips and no raw ids", async () => {
    serve(history);
    renderTab();

    expect(await screen.findByTestId("timeline-year-2025")).toBeInTheDocument();
    const years = screen.getAllByTestId(/^timeline-year-/).map((node) => node.textContent);
    expect(years).toEqual(["2025", "2024", "2022"]);

    const transfer = within(screen.getByTestId("timeline-event-wee_transfer"));
    expect(transfer.getByText("CHI-LOCAL")).toBeInTheDocument();
    expect(transfer.getByText("MW-OTR")).toBeInTheDocument();
    expect(transfer.getByText("Requested the move to over the road")).toBeInTheDocument();

    const hire = within(screen.getByTestId("timeline-event-wee_hire"));
    expect(hire.queryByText("fc_01J9")).not.toBeInTheDocument();
    const panel = within(screen.getByRole("region", { name: "Employment history" }));
    expect(panel.getByText("3")).toBeInTheDocument();
  });

  it("marks an amended event and says who amended it and why", async () => {
    serve(history);
    renderTab();

    const rate = within(await screen.findByTestId("timeline-event-wee_rate"));
    expect(rate.getByText("Amended")).toBeInTheDocument();
    expect(
      rate.getByText(/Amended by Priya Raman on .*: effective date corrected/),
    ).toBeInTheDocument();
  });

  it("filters by kind on the server and offers to show every kind when nothing matches", async () => {
    serve(history);
    renderTab();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("combobox", { name: "Show events of kind" }));
    await user.click(await screen.findByRole("option", { name: "Rate changes" }));
    await waitFor(() =>
      expect(screen.queryByTestId("timeline-event-wee_transfer")).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId("timeline-event-wee_rate")).toBeInTheDocument();
    expect(fetchWorkerEmploymentEvents).toHaveBeenCalledWith(
      expect.objectContaining({ workerId: "wrk_1", kinds: ["RateChanged"], first: 50 }),
      expect.anything(),
    );

    await user.click(screen.getByRole("combobox", { name: "Show events of kind" }));
    await user.click(await screen.findByRole("option", { name: "Leave" }));
    expect(await screen.findByText("No events of this kind.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Show all kinds" }));
    expect(await screen.findByTestId("timeline-event-wee_transfer")).toBeInTheDocument();
  });

  it("counts a kind's events from the server's total on its first page", async () => {
    serve(history);
    renderTab();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("combobox", { name: "Show events of kind" }));
    await user.click(await screen.findByRole("option", { name: "Rate changes" }));
    const panel = within(screen.getByRole("region", { name: "Employment history" }));
    await waitFor(() => expect(panel.getByText("1")).toBeInTheDocument());
  });

  it("reads the event the Amend pane names by id when it is past the pages read", async () => {
    serve(history);
    fetchWorkerEmploymentEvent.mockResolvedValue(
      event("wee_old", "Transferred", 1_600_000_000, { reason: "Long ago" }),
    );
    renderTab(null, 3, "wee_old");

    expect(
      await screen.findByRole("dialog", { name: "Amend event" }, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(fetchWorkerEmploymentEvent).toHaveBeenCalledWith("wee_old", expect.anything());
  });

  it("amends from the row, and only for a reader who may update events", async () => {
    serve(history);
    renderTab();
    const user = userEvent.setup();

    const rate = within(await screen.findByTestId("timeline-event-wee_rate"));
    await user.click(rate.getByRole("button", { name: /^Amend Rate changed/ }));
    expect(handlers.onAmend).toHaveBeenCalledWith(expect.objectContaining({ id: "wee_rate" }));

    cleanup();
    permissionState.update = false;
    renderTab();
    const again = within(await screen.findByTestId("timeline-event-wee_rate"));
    expect(again.queryByRole("button", { name: /^Amend/ })).not.toBeInTheDocument();
  });

  it("shows the dates the employment turns on beside the history", async () => {
    serve(history);
    renderTab();

    const dates = within(await screen.findByRole("region", { name: "Dates" }));
    expect(dates.getByText("Last rate change")).toBeInTheDocument();
    expect(dates.getByText(formatUnixDateMedium(1_751_328_000))).toBeInTheDocument();
    expect(dates.getByText("4 years 2 months")).toBeInTheDocument();
    expect(screen.getByTestId("leave-standing")).toBeInTheDocument();
  });

  it("draws an empty history with the way to record the first event", async () => {
    serve([]);
    renderTab(null, 0);
    const user = userEvent.setup();

    expect(await screen.findByText("No employment events yet")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Record event" }));
    expect(handlers.onRecord).toHaveBeenCalledTimes(1);
  });

  it("says the read failed rather than claiming nothing was recorded", async () => {
    fetchWorkerEmploymentEvents.mockRejectedValue(new Error("boom"));
    renderTab();

    expect(
      await screen.findByText("The employment history could not be loaded"),
    ).toBeInTheDocument();
    expect(screen.queryByText("No employment events yet")).not.toBeInTheDocument();
  });
});
