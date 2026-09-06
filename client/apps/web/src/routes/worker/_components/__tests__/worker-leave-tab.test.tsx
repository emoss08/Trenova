import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerLeaveTab from "../leave/worker-leave-tab";

const { fetchWorkerLeaveFile } = vi.hoisted(() => ({ fetchWorkerLeaveFile: vi.fn() }));

vi.mock("@/lib/graphql/worker-leave", () => ({
  fetchWorkerLeaveFile,
  WORKER_LEAVE_KEY: "worker-leave",
  decideLeaveCase: vi.fn(),
  closeLeaveCase: vi.fn(),
  requestLeaveCertification: vi.fn(),
  recordLeaveCertification: vi.fn(),
  recordLeaveDay: vi.fn(),
  deleteLeaveDay: vi.fn(),
  openLeaveCase: vi.fn(),
  updateLeaveCase: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

function makeCase(overrides: Record<string, unknown> = {}) {
  return {
    id: "lc_1",
    workerId: "wrk_1",
    leaveType: "FMLA",
    status: "Pending",
    frequency: "Continuous",
    reason: "Serious health condition",
    fmlaDesignated: false,
    militaryCaregiver: false,
    requestedAt: 1_800_000_000,
    startsAt: 1_800_086_400,
    endsAt: null,
    decidedAt: null,
    closedAt: null,
    certificationStatus: "NotRequired",
    certificationRequestedAt: null,
    certificationDueAt: null,
    certificationReceivedAt: null,
    recertificationDueAt: null,
    certificationLate: false,
    eligibilityHoursWorked: 1400,
    notes: null,
    version: 0,
    entries: [],
    ...overrides,
  };
}

const baseFile = {
  entitlement: {
    method: "RollingBackward",
    totalHours: "480",
    usedHours: "120",
    remainingHours: "360",
    totalWeeks: "12",
    usedWeeks: "3",
    remainingWeeks: "9",
    exhausted: false,
    eligibleOnTenure: true,
    monthsEmployed: 26,
    openCaseCount: 1,
    militaryCaregiver: false,
    window: { from: 1_768_000_000, through: 1_799_536_000 },
  },
  cases: [makeCase()],
};

function renderTab(file: unknown = baseFile) {
  fetchWorkerLeaveFile.mockResolvedValue(file);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <WorkerLeaveTab workerId="wrk_1" />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkerLeaveTab", () => {
  it("shows the entitlement as hours and weeks against the measurement window", async () => {
    renderTab();

    expect(await screen.findByText("360 h")).toBeInTheDocument();
    expect(screen.getByText("9 weeks")).toBeInTheDocument();
    expect(screen.getByText(/Rolling twelve months looking back/)).toBeInTheDocument();
  });

  // Approving and designating are separate decisions under 825.300(d): a case
  // can be approved as company leave without being charged to the FMLA
  // entitlement, so the two must be offered separately.
  it("offers approve-and-designate and approve-only on a pending case", async () => {
    renderTab();

    expect(await screen.findByRole("button", { name: /Approve & designate/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Approve only/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Deny$/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Record a day/i })).not.toBeInTheDocument();
  });

  it("only offers recording a day once the case is approved", async () => {
    renderTab({
      ...baseFile,
      cases: [makeCase({ status: "Approved", fmlaDesignated: true })],
    });

    expect(await screen.findByRole("button", { name: /Record a day/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Approve & designate/i })).not.toBeInTheDocument();
  });

  // A day recorded against an undesignated case is time off, but it draws
  // nothing down. Reading the list without that mark, the hours look charged.
  it("marks a day that was not charged to the entitlement", async () => {
    renderTab({
      ...baseFile,
      cases: [
        makeCase({
          status: "Approved",
          entries: [
            {
              id: "le_1",
              leaveCaseId: "lc_1",
              usedOn: 1_800_172_800,
              hours: "8",
              countsAgainstEntitlement: false,
              ptoId: null,
              notes: null,
              version: 0,
            },
          ],
        }),
      ],
    });

    const entry = await screen.findByText("8 h");
    expect(within(entry.closest("li") as HTMLElement).getByText("Not counted")).toBeInTheDocument();
  });

  it("flags an exhausted entitlement and a worker short of twelve months", async () => {
    renderTab({
      ...baseFile,
      entitlement: {
        ...baseFile.entitlement,
        usedHours: "480",
        remainingHours: "0",
        remainingWeeks: "0",
        exhausted: true,
        eligibleOnTenure: false,
        monthsEmployed: 7,
      },
    });

    await waitFor(() => expect(screen.getByText("Exhausted")).toBeInTheDocument());
    expect(screen.getByText(/Under 12 months. service/)).toBeInTheDocument();
  });
});
