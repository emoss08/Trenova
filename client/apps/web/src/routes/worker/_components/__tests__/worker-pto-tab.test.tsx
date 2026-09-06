import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerPTOTab, { summarizeWorkerPTO } from "../worker-pto-tab";

const { fetchWorkerPTOHistory } = vi.hoisted(() => ({
  fetchWorkerPTOHistory: vi.fn(),
}));

const permissionState = vi.hoisted(() => ({ create: true }));

vi.mock("@/lib/queries", () => ({
  queries: {
    worker: {
      ptoHistory: (workerId: string) => ({
        queryKey: ["worker", "pto-history", workerId],
        queryFn: () => fetchWorkerPTOHistory(workerId),
      }),
      listUpcomingPTO: { _def: ["worker", "list-upcoming-pto"] },
      ptoChartData: { _def: ["worker", "pto-chart-data"] },
    },
  },
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (_resource: string, operation: number) => ({
    allowed: operation === 2 ? permissionState.create : true,
    isLoading: false,
  }),
}));

vi.mock("../pto/pto-form-dialog", () => ({
  PTOFormDialog: ({ open }: { open: boolean }) =>
    open ? <div role="dialog">form-dialog</div> : null,
}));

vi.mock("../pto/pto-reason-dialog", () => ({
  PTOReasonDialog: () => null,
}));

vi.mock("../pto/worker-pto-balances", () => ({
  WorkerPTOBalances: () => <div data-testid="worker-pto-balances" />,
}));

const DAY = 86_400;
const NOW = Math.floor(Date.now() / 1000);

const history = [
  {
    id: "wrkpto_1",
    workerId: "wrk_1",
    status: "Requested",
    type: "Vacation",
    startDate: NOW + DAY * 10,
    endDate: NOW + DAY * 12,
    reason: "Trip",
  },
  {
    id: "wrkpto_2",
    workerId: "wrk_1",
    status: "Approved",
    type: "Sick",
    startDate: NOW + DAY * 20,
    endDate: NOW + DAY * 21,
    reason: "Surgery",
    approver: { id: "usr_1", name: "Dana Dispatcher" },
  },
  {
    id: "wrkpto_3",
    workerId: "wrk_1",
    status: "Rejected",
    type: "Personal",
    startDate: NOW - DAY * 40,
    endDate: NOW - DAY * 39,
    reason: "Errand",
    rejector: { id: "usr_2", name: "Sam Super" },
    rejectionReason: "Short notice",
  },
] as WorkerPTO[];

function renderTab() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <WorkerPTOTab workerId="wrk_1" />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  permissionState.create = true;
});

describe("summarizeWorkerPTO", () => {
  it("counts pending requests and only approved days that are upcoming or in the current year", () => {
    const now = Date.UTC(2026, 5, 15) / 1000;
    const summary = summarizeWorkerPTO(
      [
        { status: "Requested", startDate: now + DAY, endDate: now + DAY * 2 },
        { status: "Approved", startDate: now + DAY * 5, endDate: now + DAY * 6 },
        { status: "Approved", startDate: now - DAY * 30, endDate: now - DAY * 29 },
        { status: "Approved", startDate: now - DAY * 400, endDate: now - DAY * 398 },
        { status: "Rejected", startDate: now + DAY * 8, endDate: now + DAY * 9 },
      ] as WorkerPTO[],
      now,
    );
    expect(summary.pending).toBe(1);
    expect(summary.upcomingApprovedDays).toBe(2);
    expect(summary.approvedDaysThisYear).toBe(4);
  });
});

describe("WorkerPTOTab", () => {
  it("lists the worker's history with decisions and the request button", async () => {
    fetchWorkerPTOHistory.mockResolvedValue(history);
    renderTab();

    expect(await screen.findByText("Dana Dispatcher")).toBeInTheDocument();
    expect(screen.getByText(/Sam Super — Short notice/)).toBeInTheDocument();
    expect(fetchWorkerPTOHistory).toHaveBeenCalledExactlyOnceWith("wrk_1");
    expect(screen.getByRole("button", { name: /request pto/i })).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "PTO actions" })).toHaveLength(2);
  });

  it("shows an empty state and hides the request button without create permission", async () => {
    fetchWorkerPTOHistory.mockResolvedValue([]);
    permissionState.create = false;
    renderTab();

    expect(await screen.findByText(/no time off on record/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /request pto/i })).not.toBeInTheDocument();
  });
});
