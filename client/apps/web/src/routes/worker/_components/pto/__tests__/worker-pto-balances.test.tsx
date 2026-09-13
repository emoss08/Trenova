import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { WorkerPTOBalances } from "../worker-pto-balances";

const { fetchWorkerPtoBalances, fetchWorkerPtoPolicyAssignments, fetchWorkerPtoLedger } =
  vi.hoisted(() => ({
    fetchWorkerPtoBalances: vi.fn(),
    fetchWorkerPtoPolicyAssignments: vi.fn(),
    fetchWorkerPtoLedger: vi.fn(),
  }));

const permissionState = vi.hoisted(() => ({ assign: true, manage: true }));

vi.mock("@/lib/graphql/pto-policy", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/pto-policy")>();
  return {
    ...actual,
    fetchWorkerPtoBalances,
    fetchWorkerPtoPolicyAssignments,
    fetchWorkerPtoLedger,
    runPtoAccrual: vi.fn(),
    adjustWorkerPtoBalance: vi.fn(),
    assignWorkerPtoPolicy: vi.fn(),
  };
});

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (_resource: string, operation: number) => ({
    allowed:
      operation === 1 << 10
        ? permissionState.assign
        : operation === 1 << 25
          ? permissionState.manage
          : true,
    isLoading: false,
  }),
}));

vi.mock("@/components/fields/date-field/date-field", () => ({
  AutoCompleteDateField: () => null,
}));

const balances = [
  {
    ptoType: "Vacation",
    tracked: true,
    enforced: true,
    balanceDays: "6.50",
    pendingDays: "2.00",
    availableDays: "4.50",
    accruedYtdDays: "5.00",
    usedYtdDays: "1.00",
    carriedDays: "2.50",
    maxBalanceDays: "20.00",
    nextAccrual: {
      entryType: "Accrual",
      periodKey: "M:2026-10",
      effectiveAt: 1_790_812_800,
      nominalDays: "0.83",
      deferred: false,
    },
  },
  {
    ptoType: "Sick",
    tracked: true,
    enforced: false,
    balanceDays: "-1.00",
    pendingDays: "0.00",
    availableDays: "-1.00",
    accruedYtdDays: "0.00",
    usedYtdDays: "1.00",
    carriedDays: "0.00",
    maxBalanceDays: null,
    nextAccrual: null,
  },
];

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
    name: "Standard Driver",
    code: "STD-DRIVER",
    status: "Active",
    countWeekends: true,
    requiresApproval: true,
    enforceBalance: true,
  },
};

const ledger = {
  entries: [
    {
      id: "wpl_1",
      workerId: "wrk_1",
      ptoType: "Vacation",
      entryType: "Accrual",
      amountDays: "0.83",
      balanceAfterDays: "6.50",
      sequence: 3,
      effectiveAt: 1_788_220_800,
      periodKey: "M:2026-09",
      sourcePtoId: null,
      assignmentId: "wppa_1",
      ptoPolicyId: "ptop_1",
      actorType: "System",
      createdById: null,
      note: null,
      createdAt: 1,
    },
    {
      id: "wpl_2",
      workerId: "wrk_1",
      ptoType: "Vacation",
      entryType: "Usage",
      amountDays: "-1.00",
      balanceAfterDays: "5.67",
      sequence: 2,
      effectiveAt: 1_787_000_000,
      periodKey: null,
      sourcePtoId: "wrkpto_1",
      assignmentId: null,
      ptoPolicyId: "ptop_1",
      actorType: "User",
      createdById: "usr_1",
      note: null,
      createdAt: 1,
    },
  ],
  hasNextPage: false,
  endCursor: null,
};

function renderBalances() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <WorkerPTOBalances workerId="wrk_1" />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  permissionState.assign = true;
  permissionState.manage = true;
});

describe("WorkerPTOBalances", () => {
  it("shows the assigned policy, one card per tracked type, and the ledger", async () => {
    fetchWorkerPtoBalances.mockResolvedValue(balances);
    fetchWorkerPtoPolicyAssignments.mockResolvedValue([assignment]);
    fetchWorkerPtoLedger.mockResolvedValue(ledger);
    renderBalances();

    expect(await screen.findByText("STD-DRIVER")).toBeInTheDocument();
    const vacation = await screen.findByTestId("pto-balance-Vacation");
    expect(vacation).toHaveTextContent("4.50");
    expect(vacation).toHaveTextContent("Cap 20.00 days");
    expect(vacation).toHaveTextContent("+0.83 on");

    const sick = screen.getByTestId("pto-balance-Sick");
    expect(sick).toHaveTextContent("Not enforced");
    expect(sick).toHaveTextContent("-1.00");

    expect(await screen.findByText("M:2026-09")).toBeInTheDocument();
    expect(screen.getByText("Time off taken")).toBeInTheDocument();
    expect(screen.getByText("Nightly accrual")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /change policy/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Balance actions" })).toBeInTheDocument();
  });

  it("shows the enrolment prompt and hides manage actions when there is no policy", async () => {
    fetchWorkerPtoBalances.mockResolvedValue([]);
    fetchWorkerPtoPolicyAssignments.mockResolvedValue([]);
    permissionState.manage = false;
    renderBalances();

    expect(await screen.findByText(/no pto policy assigned/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /assign policy/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Balance actions" })).not.toBeInTheDocument();
    expect(fetchWorkerPtoLedger).not.toHaveBeenCalled();
  });
});
