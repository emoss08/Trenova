import type { WorkerRecord } from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerTimelineTab from "../worker-timeline-tab";

const { fetchWorkerEmploymentEvents } = vi.hoisted(() => ({
  fetchWorkerEmploymentEvents: vi.fn(),
}));
const permissionState = vi.hoisted(() => ({ create: true, update: true }));

vi.mock("@/lib/graphql/worker-employment", () => ({
  fetchWorkerEmploymentEvents,
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

const handlers = { onRecord: vi.fn(), onAmend: vi.fn() };

function renderTab(highlightId: string | null = null) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <WorkerTimelineTab
        workerId="wrk_1"
        worker={worker}
        highlightId={highlightId}
        now={1_795_305_600}
        {...handlers}
      />
    </QueryClientProvider>,
  );
}

afterEach(() => {
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
    fetchWorkerEmploymentEvents.mockResolvedValue(history);
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
    expect(screen.getByText("3 of 3 events")).toBeInTheDocument();
  });

  it("marks an amended event and says who amended it and why", async () => {
    fetchWorkerEmploymentEvents.mockResolvedValue(history);
    renderTab();

    const rate = within(await screen.findByTestId("timeline-event-wee_rate"));
    expect(rate.getByText("Amended")).toBeInTheDocument();
    expect(
      rate.getByText(/Amended by Priya Raman on .*: effective date corrected/),
    ).toBeInTheDocument();
  });

  it("filters by kind and offers to show every kind when the filter empties the list", async () => {
    fetchWorkerEmploymentEvents.mockResolvedValue(history);
    renderTab();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("combobox", { name: "Show events of kind" }));
    await user.click(await screen.findByRole("option", { name: "Rate changes" }));
    expect(screen.getByText("1 of 3 events")).toBeInTheDocument();
    expect(screen.queryByTestId("timeline-event-wee_transfer")).not.toBeInTheDocument();

    await user.click(screen.getByRole("combobox", { name: "Show events of kind" }));
    await user.click(await screen.findByRole("option", { name: "Leave" }));
    expect(screen.getByText("No events of this kind.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Show all kinds" }));
    expect(screen.getByText("3 of 3 events")).toBeInTheDocument();
  });

  it("amends from the row, and only for a reader who may update events", async () => {
    fetchWorkerEmploymentEvents.mockResolvedValue(history);
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
    fetchWorkerEmploymentEvents.mockResolvedValue(history);
    renderTab();

    const dates = within(await screen.findByRole("region", { name: "Dates" }));
    expect(dates.getByText("Last rate change")).toBeInTheDocument();
    expect(dates.getByText("4 years 2 months")).toBeInTheDocument();
    expect(screen.getByTestId("leave-standing")).toBeInTheDocument();
  });

  it("draws an empty history with the way to record the first event", async () => {
    fetchWorkerEmploymentEvents.mockResolvedValue([]);
    renderTab();
    const user = userEvent.setup();

    expect(await screen.findByText("No employment events yet")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Record event" }));
    expect(handlers.onRecord).toHaveBeenCalledTimes(1);
  });

  it("says the read failed rather than claiming nothing was recorded", async () => {
    fetchWorkerEmploymentEvents.mockRejectedValue(new Error("boom"));
    renderTab();

    expect(await screen.findByText("Couldn't load the employment history")).toBeInTheDocument();
    expect(screen.queryByText("No employment events yet")).not.toBeInTheDocument();
  });
});
