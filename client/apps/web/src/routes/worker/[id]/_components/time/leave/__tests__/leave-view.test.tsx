import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import { CLOSED_WORKER_OPEN_STATE, type WorkerOpenState } from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { stubLayout } from "@/test/layout";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import LeaveView from "../leave-view";

const mocks = vi.hoisted(() => ({
  fetchWorkerLeaveFile: vi.fn(),
  deleteLeaveDay: vi.fn(),
  decideLeaveCase: vi.fn(),
}));
const permissions = vi.hoisted(() => ({ read: true }));

vi.mock("@/lib/graphql/worker-leave", () => ({
  fetchWorkerLeaveFile: mocks.fetchWorkerLeaveFile,
  WORKER_LEAVE_KEY: "worker-leave",
  decideLeaveCase: mocks.decideLeaveCase,
  closeLeaveCase: vi.fn(),
  requestLeaveCertification: vi.fn(),
  recordLeaveCertification: vi.fn(),
  recordLeaveDay: vi.fn(),
  deleteLeaveDay: mocks.deleteLeaveDay,
  openLeaveCase: vi.fn(),
  updateLeaveCase: vi.fn(),
}));

vi.mock("@/lib/queries/worker", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/worker")>()),
  worker: { overview: (workerId: string) => ({ queryKey: ["worker", "overview", workerId] }) },
}));

vi.mock("@/hooks/use-permission", async () => {
  const { Operation } = await import("@trenova/shared/types/permission");
  return {
    usePermission: (_resource: string, operation: number) => ({
      allowed: operation === Operation.Read ? permissions.read : true,
      isLoading: false,
    }),
  };
});

vi.mock("@/components/fields/date-field/date-field", () => ({
  AutoCompleteDateField: ({ label }: { label: string }) => <input aria-label={label} />,
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

const entry = {
  id: "le_1",
  leaveCaseId: "lc_1",
  usedOn: 1_800_172_800,
  hours: "8",
  countsAgainstEntitlement: false,
  ptoId: null,
  notes: null,
  version: 0,
};

function renderView(file: unknown = baseFile, open: Partial<WorkerOpenState> = {}) {
  mocks.fetchWorkerLeaveFile.mockResolvedValue(file);
  const onOpen = vi.fn();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <WorkPaneProvider>
        <WorkPaneHost>
          <LeaveView
            workerId="wrk_1"
            open={{ ...CLOSED_WORKER_OPEN_STATE, ...open }}
            onOpen={onOpen}
          />
        </WorkPaneHost>
      </WorkPaneProvider>
    </QueryClientProvider>,
  );
  return { onOpen };
}

// The cases are drawn a screenful at a time, which needs a laid-out region.
let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout(640);
});

afterEach(() => {
  restoreLayout();
  cleanup();
  vi.clearAllMocks();
  permissions.read = true;
});

describe("LeaveView", () => {
  it("shows the entitlement as hours left, weeks and the measurement window", async () => {
    renderView();

    expect(await screen.findByText("of 480 hours left")).toBeInTheDocument();
    expect(screen.getByText("360")).toBeInTheDocument();
    expect(screen.getByText("9 weeks")).toBeInTheDocument();
    expect(screen.getByText(/Rolling twelve months looking back/)).toBeInTheDocument();
  });

  // Approving and designating are separate decisions under 825.300(d): a case can be approved
  // as company leave without being charged to the FMLA entitlement.
  it("offers approve-and-designate, approve-only and deny on a pending case", async () => {
    renderView();

    expect(
      await screen.findByRole("button", { name: "Approve and designate FMLA" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Approve only" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Deny" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Record a day" })).not.toBeInTheDocument();
  });

  it("asks before denying a case", async () => {
    mocks.decideLeaveCase.mockResolvedValue({ id: "lc_1" });
    const user = userEvent.setup();
    renderView();

    await user.click(await screen.findByRole("button", { name: "Deny" }));
    expect(mocks.decideLeaveCase).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Deny case" }));

    await waitFor(() =>
      expect(mocks.decideLeaveCase).toHaveBeenCalledExactlyOnceWith({
        caseId: "lc_1",
        approve: false,
        designate: false,
      }),
    );
  });

  it("only offers recording a day once the case is approved", async () => {
    renderView({ ...baseFile, cases: [makeCase({ status: "Approved", fmlaDesignated: true })] });

    expect(await screen.findByRole("button", { name: "Record a day" })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Approve and designate FMLA" }),
    ).not.toBeInTheDocument();
  });

  it("opens the day dialog for the case, starting from an empty day each time", async () => {
    const user = userEvent.setup();
    renderView({ ...baseFile, cases: [makeCase({ status: "Approved", fmlaDesignated: true })] });

    await user.click(await screen.findByRole("button", { name: "Record a day" }));
    const dialog = await screen.findByRole("dialog", { name: "Record a day of leave" });
    expect(
      within(dialog).getByText(
        "This case is designated as FMLA, so the day draws the entitlement down.",
      ),
    ).toBeInTheDocument();
    const notes = within(dialog).getByLabelText("Notes");
    await user.type(notes, "Half day");
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());

    await user.click(screen.getByRole("button", { name: "Record a day" }));
    const reopened = await screen.findByRole("dialog", { name: "Record a day of leave" });
    expect(within(reopened).getByLabelText("Notes")).toHaveValue("");
  });

  // A day recorded against an undesignated case is time off, but it draws nothing down.
  it("expands the day entries in place and marks a day that was not charged", async () => {
    const user = userEvent.setup();
    renderView({ ...baseFile, cases: [makeCase({ status: "Approved", entries: [entry] })] });

    const toggle = await screen.findByRole("button", { name: /Day entries/ });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    await user.click(toggle);

    const row = screen.getByText("8 h").closest("tr") as HTMLElement;
    expect(within(row).getByText("Not counted")).toBeInTheDocument();
  });

  it("removes a day only after the confirmation", async () => {
    mocks.deleteLeaveDay.mockResolvedValue(true);
    const user = userEvent.setup();
    renderView({ ...baseFile, cases: [makeCase({ status: "Approved", entries: [entry] })] });

    await user.click(await screen.findByRole("button", { name: /Day entries/ }));
    await user.click(screen.getByRole("button", { name: "Remove" }));
    expect(mocks.deleteLeaveDay).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Remove day" }));

    await waitFor(() => expect(mocks.deleteLeaveDay).toHaveBeenCalledExactlyOnceWith("le_1"));
  });

  it("flags an exhausted entitlement and a worker short of twelve months", async () => {
    renderView({
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

    expect(await screen.findByText("Exhausted")).toBeInTheDocument();
    expect(screen.getByText(/Under 12 months. service/)).toBeInTheDocument();
  });

  it("opens the case form in the work pane from the address", async () => {
    renderView(baseFile, { form: "open-leave-case" });
    expect(await screen.findByRole("dialog", { name: "Open a leave case" })).toBeInTheDocument();
  });

  it("opens a case for editing from its record param", async () => {
    renderView(baseFile, { leave: "lc_1" });
    expect(await screen.findByRole("dialog", { name: "Edit leave case" })).toBeInTheDocument();
  });

  it("says plainly that leave is withheld from a reader without the grant", async () => {
    permissions.read = false;
    renderView();

    expect(await screen.findByText("Leave is not shared with you")).toBeInTheDocument();
    expect(mocks.fetchWorkerLeaveFile).not.toHaveBeenCalled();
  });

  it("says no case has been opened when the file has none", async () => {
    renderView({ ...baseFile, cases: [] });
    expect(
      await screen.findByText("No leave case has been opened for this worker."),
    ).toBeInTheDocument();
  });
});
