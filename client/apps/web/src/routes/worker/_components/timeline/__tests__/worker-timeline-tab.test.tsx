import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerTimelineTab from "../../worker-timeline-tab";

const { fetchWorkerEmploymentEvents } = vi.hoisted(() => ({
  fetchWorkerEmploymentEvents: vi.fn(),
}));
const sheetProps = vi.hoisted(() => ({ last: null as Record<string, unknown> | null }));
const permissionState = vi.hoisted(() => ({ create: true, update: true }));

vi.mock("@/lib/graphql/worker-employment", () => ({
  fetchWorkerEmploymentEvents,
  WORKER_EMPLOYMENT_EVENTS_KEY: "worker-employment-events",
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

vi.mock("../employment-event-dialog", () => ({
  EmploymentEventSheet: (props: Record<string, unknown>) => {
    sheetProps.last = props;
    return props.open ? <div role="dialog">event-sheet</div> : null;
  },
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

const hired = event("wee_hire", "Hired", Date.UTC(2024, 2, 4) / 1000, {
  recordedBy: null,
  recordedById: null,
});
const transfer = event("wee_move", "Transferred", Date.UTC(2025, 5, 1) / 1000, {
  reason: "Closer to home",
  fromValues: [
    { key: "fleetCodeId", value: "fc_1" },
    { key: "fleetCode", value: "SOUTH" },
  ],
  toValues: [
    { key: "fleetCodeId", value: "fc_2" },
    { key: "fleetCode", value: "NORTH" },
  ],
});
const terminated = event("wee_term", "Terminated", Date.UTC(2026, 0, 15) / 1000, {
  reason: "Resigned",
  amendedAt: 1_770_000_000,
  amendedBy: { id: "usr_2", name: "Grace Hopper" },
  amendmentNote: "Corrected the last day",
  toValues: [{ key: "status", value: "Inactive" }],
  version: 1,
});

function renderTab() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <WorkerTimelineTab
        workerId="wrk_1"
        worker={{ fleetCodeId: "fc_2", driverType: "OTR", type: "Employee", status: "Inactive" }}
      />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  sheetProps.last = null;
  permissionState.create = true;
  permissionState.update = true;
});

describe("WorkerTimelineTab", () => {
  it("renders events newest first, grouped by year, with what changed", async () => {
    fetchWorkerEmploymentEvents.mockResolvedValue([terminated, transfer, hired]);
    renderTab();

    const items = await screen.findAllByTestId(/^timeline-event-/);
    expect(items.map((item) => item.getAttribute("data-kind"))).toEqual([
      "Terminated",
      "Transferred",
      "Hired",
    ]);
    expect(screen.getByTestId("timeline-year-2026")).toBeInTheDocument();
    expect(screen.getByTestId("timeline-year-2025")).toBeInTheDocument();
    expect(screen.getByTestId("timeline-year-2024")).toBeInTheDocument();

    const move = screen.getByTestId("timeline-event-wee_move");
    expect(within(move).getByText("Transferred")).toBeInTheDocument();
    expect(within(move).getByText("Closer to home")).toBeInTheDocument();
    expect(within(move).getByText("SOUTH")).toBeInTheDocument();
    expect(within(move).getByText("NORTH")).toBeInTheDocument();
    expect(within(move).queryByText("fc_2")).toBeNull();
    expect(within(move).getByText(/Ada Lovelace/)).toBeInTheDocument();

    const term = screen.getByTestId("timeline-event-wee_term");
    expect(within(term).getByText("Amended")).toBeInTheDocument();
    expect(within(term).getByText(/Corrected the last day/)).toBeInTheDocument();

    const hire = screen.getByTestId("timeline-event-wee_hire");
    expect(within(hire).getByText(/System/)).toBeInTheDocument();
  });

  it("filters by kind without refetching", async () => {
    fetchWorkerEmploymentEvents.mockResolvedValue([terminated, transfer, hired]);
    renderTab();
    await screen.findByTestId("timeline-event-wee_hire");

    fireEvent.click(screen.getByRole("button", { name: "Terminated", pressed: false }));
    expect(screen.getByTestId("timeline-event-wee_term")).toBeInTheDocument();
    expect(screen.queryByTestId("timeline-event-wee_hire")).toBeNull();
    expect(fetchWorkerEmploymentEvents).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole("button", { name: "Terminated", pressed: true }));
    expect(screen.getByTestId("timeline-event-wee_hire")).toBeInTheDocument();
  });

  it("opens the sheet to record and to amend", async () => {
    fetchWorkerEmploymentEvents.mockResolvedValue([transfer]);
    renderTab();
    await screen.findByTestId("timeline-event-wee_move");

    fireEvent.click(screen.getByRole("button", { name: "Record event" }));
    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());
    expect(sheetProps.last).toMatchObject({ mode: "record", workerId: "wrk_1" });

    fireEvent.click(screen.getByRole("button", { name: "Amend Transferred" }));
    await waitFor(() =>
      expect(sheetProps.last).toMatchObject({
        mode: "amend",
        event: expect.objectContaining({ id: "wee_move" }),
      }),
    );
  });

  it("hides recording and amending without permission and shows an empty state", async () => {
    permissionState.create = false;
    permissionState.update = false;
    fetchWorkerEmploymentEvents.mockResolvedValue([]);
    renderTab();

    expect(await screen.findByText("No employment events yet")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Record event" })).toBeNull();
  });
});
