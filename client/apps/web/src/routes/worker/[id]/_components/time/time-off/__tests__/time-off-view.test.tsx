import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import type { WorkerRecord } from "@/lib/graphql/worker-record";
import { CLOSED_WORKER_OPEN_STATE, type WorkerOpenState } from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { afterEach, describe, expect, it, vi } from "vitest";
import TimeOffView from "../time-off-view";

const mocks = vi.hoisted(() => ({
  balances: vi.fn(),
  assignments: vi.fn(),
  ledger: vi.fn(),
  history: vi.fn(),
}));

vi.mock("@/lib/graphql/pto-policy", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/pto-policy")>()),
  fetchWorkerPtoBalances: mocks.balances,
  fetchWorkerPtoPolicyAssignments: mocks.assignments,
  fetchWorkerPtoLedger: mocks.ledger,
  fetchWorkerPtoAvailability: vi.fn().mockResolvedValue(null),
  runPtoAccrual: vi.fn(),
  adjustWorkerPtoBalance: vi.fn(),
  assignWorkerPtoPolicy: vi.fn(),
}));

vi.mock("@/lib/queries/worker", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/worker")>()),
  worker: {
    ptoHistory: (workerId: string) => ({
      queryKey: ["worker", "pto-history", workerId],
      queryFn: () => mocks.history(workerId),
    }),
    overview: (workerId: string) => ({ queryKey: ["worker", "overview", workerId] }),
    listUpcomingPTO: { _def: ["worker", "list-upcoming-pto"] },
    ptoChartData: { _def: ["worker", "pto-chart-data"] },
  },
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@/hooks/use-select-option", () => ({
  useSelectOption: () => ({ option: null }),
}));

vi.mock("@/components/autocomplete-fields", () => ({
  PtoPolicyAutocompleteField: ({ label }: { label: string }) => <input aria-label={label} />,
  WorkerAutocompleteField: ({ label }: { label: string }) => <input aria-label={label} />,
}));

vi.mock("@/components/fields/date-field/date-field", () => ({
  AutoCompleteDateField: ({ label }: { label: string }) => <input aria-label={label} />,
}));

const DAY = 86_400;
const START = 1_798_070_400;

const assignment = {
  id: "wppa_1",
  workerId: "wrk_1",
  ptoPolicyId: "ptop_1",
  effectiveFrom: 1_767_225_600,
  effectiveTo: null,
  assignedById: "usr_1",
  note: null,
  version: 1,
  createdAt: 1,
  updatedAt: 1,
  ptoPolicy: {
    id: "ptop_1",
    name: "OTR standard",
    code: "PTO-OTR",
    status: "Active",
    countWeekends: true,
    requiresApproval: true,
    enforceBalance: true,
  },
};

const balance = {
  ptoType: "Vacation",
  tracked: true,
  enforced: true,
  balanceDays: "11.50",
  pendingDays: "2.00",
  availableDays: "9.50",
  accruedYtdDays: "12.00",
  usedYtdDays: "5.50",
  carriedDays: "0.00",
  maxBalanceDays: "15.00",
  nextAccrual: null,
};

const decided = {
  id: "wrkpto_2",
  workerId: "wrk_1",
  status: "Approved",
  type: "Vacation",
  startDate: START,
  endDate: START + 4 * DAY,
  reason: "Family trip",
  version: 3,
  approver: { id: "usr_1", name: "Dana Whitfield" },
  worker: { firstName: "Maria", lastName: "Alvarez" },
} as unknown as WorkerPTO;

const worker = { id: "wrk_1", wholeName: "Maria Alvarez" } as WorkerRecord;

function renderView(open: Partial<WorkerOpenState>) {
  const onOpen = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <WorkPaneProvider>
        <WorkPaneHost>
          <TimeOffView
            workerId="wrk_1"
            worker={worker}
            open={{ ...CLOSED_WORKER_OPEN_STATE, ...open }}
            onOpen={onOpen}
          />
        </WorkPaneHost>
      </WorkPaneProvider>
    </QueryClientProvider>,
  );
  return { onOpen };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function seed() {
  mocks.balances.mockResolvedValue([balance]);
  mocks.assignments.mockResolvedValue([assignment]);
  mocks.ledger.mockResolvedValue({ entries: [], hasNextPage: false, endCursor: null });
  mocks.history.mockResolvedValue([decided]);
}

describe("TimeOffView", () => {
  it("shows the policy line, the balances strip and the requests", async () => {
    seed();
    renderView({});

    expect(await screen.findByText("PTO-OTR")).toBeInTheDocument();
    expect(screen.getByTestId("pto-balance-Vacation")).toHaveTextContent("9.5");
    expect(screen.getByText("Dana Whitfield")).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("opens the request form in the work pane from the address, without a worker picker", async () => {
    seed();
    renderView({ form: "request-time-off" });

    const pane = await screen.findByRole("dialog", { name: "Request time off" });
    expect(pane).toBeInTheDocument();
    expect(screen.queryByLabelText("Worker")).not.toBeInTheDocument();
    expect(screen.getByLabelText("First day")).toBeInTheDocument();
    expect(screen.getByTestId("pto-day-count")).toHaveTextContent("0");
  });

  it("opens a decided request read-only from its record param", async () => {
    seed();
    renderView({ timeOff: "wrkpto_2" });

    expect(await screen.findByRole("dialog", { name: "Time off request" })).toBeInTheDocument();
    expect(screen.getByText(/only requested time off can be edited/i)).toBeInTheDocument();
  });

  it("opens the adjustment pane from the address and closes it back to the view", async () => {
    seed();
    const user = userEvent.setup();
    const { onOpen } = renderView({ form: "adjust-balance" });

    expect(await screen.findByRole("dialog", { name: "Adjust a balance" })).toBeInTheDocument();
    expect(screen.getByText("Vacation is 11.50 days today.")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onOpen).toHaveBeenCalledWith({ tab: "time", view: "time-off" });
  });

  it("offers enrollment instead of balances when the worker is on no policy", async () => {
    mocks.balances.mockResolvedValue([]);
    mocks.assignments.mockResolvedValue([]);
    mocks.history.mockResolvedValue([]);
    renderView({});

    expect(await screen.findByText("Not enrolled in a time off policy")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Enroll in a policy" }).length).toBeGreaterThan(0);
    expect(mocks.ledger).not.toHaveBeenCalled();
  });
});
